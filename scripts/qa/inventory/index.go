package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// hole stands for a part of a string that is not a constant (a variable, a call):
// "/transactions/" + id becomes "/transactions/\x00".
const hole = "\x00"

// unit is the smallest thing that covers an entry: a test function, or a file
// when the evidence sits in a table outside any function.
type unit struct {
	file string
	test string // "" for evidence outside a test function
}

// name is how the inventory names the unit.
func (u unit) name() string {
	if u.test != "" {
		return u.test
	}

	return u.file
}

// request is one HTTP request a test makes, as a method and a path template.
type request struct {
	method string // "" when the test does not say
	path   string
	unit   unit
}

// index is what the tests of the repository say they touch.
type index struct {
	// strings maps a constant string to the units that hold it.
	strings map[string]map[unit]bool
	// idents maps an identifier or the name selected in `pkg.Name` to its units.
	idents map[string]map[unit]bool
	// requests are the HTTP calls found.
	requests []request
	// tests are the names of the test functions that exist.
	tests map[string]bool
	// consts are the path constants of the app (`PixWebhookPath`), by name, so a
	// test that posts to one covers its route.
	consts map[string]string
	// scheduleFiles are the files that mention Schedule.
	scheduleFiles map[string]bool
}

func newIndex() *index {
	return &index{
		strings: map[string]map[unit]bool{}, idents: map[string]map[unit]bool{},
		tests: map[string]bool{}, scheduleFiles: map[string]bool{},
	}
}

// skipDirs are never read: dependencies, generated output and other modules.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "desktop": true, "mobile": true, "site": true, "storage": true,
	"vendor": true, "testdata": true, "test-results": true, ".vukem": true, "dist": true, "build": true,
}

// buildIndex reads every Go test file and every Playwright spec under root.
// hidden names tests (or files) to leave out, for the scratch run of the check.
func buildIndex(root string, hidden map[string]bool) (*index, error) {
	ix := newIndex()

	consts, err := pathConstants(root)
	if err != nil {
		return nil, err
	}

	ix.consts = consts

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipDirs[d.Name()] && path != root {
				return filepath.SkipDir
			}

			return nil
		}

		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)

		// The tests of this tool hold made-up samples, not coverage.
		if strings.HasPrefix(rel, "scripts/qa/inventory/") {
			return nil
		}

		switch {
		case strings.HasSuffix(rel, "_test.go"):
			if hidden[rel] || hidden[filepath.Base(rel)] {
				return nil
			}

			return ix.addGo(path, rel, hidden)
		case strings.HasPrefix(rel, "e2e/") && strings.HasSuffix(rel, ".spec.ts"):
			if hidden[rel] || hidden[filepath.Base(rel)] {
				return nil
			}

			return ix.addSpec(path, rel)
		}

		return nil
	})

	return ix, err
}

func (ix *index) noteString(s string, u unit) {
	if ix.strings[s] == nil {
		ix.strings[s] = map[unit]bool{}
	}

	ix.strings[s][u] = true
}

func (ix *index) noteIdent(s string, u unit) {
	if ix.idents[s] == nil {
		ix.idents[s] = map[unit]bool{}
	}

	ix.idents[s][u] = true
}

var verbCall = regexp.MustCompile(`^([Gg]et|[Pp]ost|[Pp]ut|[Pp]atch|[Dd]elete|[Hh]ead)([A-Z][A-Za-z]*)?$`)

var httpMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true}

var routeLiteral = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|HEAD) (/\S*)$`)

func (ix *index) addGo(path, rel string, hidden map[string]bool) error {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	for _, decl := range file.Decls {
		u := unit{file: rel}

		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && isTestName(fn.Name.Name) {
			if hidden[fn.Name.Name] {
				continue
			}

			u.test = fn.Name.Name
			ix.tests[fn.Name.Name] = true
		}

		// A path kept in a variable (`base := "/cards/" + id`) is followed
		// through the function, in source order.
		env := map[string]string{}
		inner := map[ast.Node]bool{}

		ast.Inspect(decl, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.BinaryExpr:
				// "GET /admin/users/" + id + "/edit" is a request pattern when
				// read whole; its parts are not.
				if v.Op == token.ADD && !inner[v] {
					if m := routeLiteral.FindStringSubmatch(constString(v, env, ix.consts)); m != nil {
						ix.addRoute(m[1], m[2], u)
					}

					markInner(v, inner)
				}
			case *ast.AssignStmt:
				if len(v.Lhs) == len(v.Rhs) {
					for i, l := range v.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							if s := constString(v.Rhs[i], env, ix.consts); strings.Contains(s, "/") {
								env[id.Name] = s
							}
						}
					}
				}
			case *ast.BasicLit:
				if v.Kind == token.STRING {
					if s, err := strconv.Unquote(v.Value); err == nil {
						ix.noteString(s, u)

						if m := routeLiteral.FindStringSubmatch(s); m != nil {
							ix.addRoute(m[1], m[2], u)
						}
					}
				}
			case *ast.Ident:
				ix.noteIdent(v.Name, u)

				if v.Name == "Schedule" {
					ix.scheduleFiles[rel] = true
				}
			case *ast.SelectorExpr:
				ix.noteIdent(v.Sel.Name, u)

				if v.Sel.Name == "Schedule" {
					ix.scheduleFiles[rel] = true
				}
			case *ast.CallExpr:
				ix.addCall(v, u, env)
			}

			return true
		})
	}

	return nil
}

// addRoute records a "METHOD /path" pattern, without its query.
func (ix *index) addRoute(method, path string, u unit) {
	if p, ok := pathOf(path); ok {
		ix.requests = append(ix.requests, request{method: method, path: p, unit: u})
	}
}

func markInner(e ast.Expr, inner map[ast.Node]bool) {
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op == token.ADD {
		inner[b.X], inner[b.Y] = true, true

		markInner(b.X, inner)
		markInner(b.Y, inner)
	}
}

func isTestName(name string) bool {
	return strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz")
}

// addCall records an HTTP request made by a call: a verb method
// (`c.Get("/x")`, `c.PostJSON("/x", ...)`, a helper like `putForm(c, "/x", v)`) or a call that names the method
// (`httptest.NewRequest("POST", "/x", ...)`).
func (ix *index) addCall(call *ast.CallExpr, u unit, env map[string]string) {
	name := ""

	switch f := call.Fun.(type) {
	case *ast.SelectorExpr:
		name = f.Sel.Name
	case *ast.Ident:
		name = f.Name
	}

	{
		if m := verbCall.FindStringSubmatch(name); m != nil {
			for _, a := range call.Args[:min(len(call.Args), 3)] {
				if p, ok := pathOf(constString(a, env, ix.consts)); ok {
					ix.requests = append(ix.requests, request{method: strings.ToUpper(m[1]), path: p, unit: u})

					break
				}
			}
		}
	}

	for i, a := range call.Args {
		s, ok := methodOf(a)
		if !ok || i+1 >= len(call.Args) {
			continue
		}

		if p, ok := pathOf(constString(call.Args[i+1], env, ix.consts)); ok {
			ix.requests = append(ix.requests, request{method: s, path: p, unit: u})
		}
	}
}

// pathOf keeps what looks like a request path: it starts with "/" once a
// leading base URL (a hole) is dropped.
func pathOf(s string) (string, bool) {
	s = strings.TrimPrefix(s, hole)
	if !strings.HasPrefix(s, "/") {
		return "", false
	}

	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}

	return s, true
}

// methodOf reads an HTTP method given as "POST" or as http.MethodPost.
func methodOf(e ast.Expr) (string, bool) {
	if s, ok := stringConst(e); ok {
		return s, httpMethods[s]
	}

	if sel, ok := e.(*ast.SelectorExpr); ok && strings.HasPrefix(sel.Sel.Name, "Method") {
		m := strings.ToUpper(strings.TrimPrefix(sel.Sel.Name, "Method"))

		return m, httpMethods[m]
	}

	return "", false
}

func stringConst(e ast.Expr) (string, bool) {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, err := strconv.Unquote(lit.Value)

		return s, err == nil
	}

	return "", false
}

var verb = regexp.MustCompile(`%[-+# 0-9.*]*[a-zA-Z]`)

// constString rebuilds the text of an expression, with a hole for every part
// that is not constant: the template of a path a test builds.
func constString(e ast.Expr, env, consts map[string]string) string {
	switch v := e.(type) {
	case *ast.Ident:
		if s, ok := env[v.Name]; ok {
			return s
		}

		if s, ok := consts[v.Name]; ok {
			return s
		}
	case *ast.SelectorExpr:
		if s, ok := consts[v.Sel.Name]; ok {
			return s
		}
	case *ast.BasicLit:
		if s, ok := stringConst(v); ok {
			return s
		}
	case *ast.ParenExpr:
		return constString(v.X, env, consts)
	case *ast.BinaryExpr:
		if v.Op == token.ADD {
			return constString(v.X, env, consts) + constString(v.Y, env, consts)
		}
	case *ast.CallExpr:
		if sel, ok := v.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Sprintf" && len(v.Args) > 0 {
			if f, ok := stringConst(v.Args[0]); ok {
				return strings.ReplaceAll(verb.ReplaceAllString(strings.ReplaceAll(f, "%%", "\x01"), hole), "\x01", "%")
			}
		}
	}

	return hole
}

var specCall = regexp.MustCompile("(?:request|page|api|client|context)\\w*\\.(goto|get|post|put|patch|delete|head)\\(\\s*[`'\"]([^`'\"]+)[`'\"]")

var specInterp = regexp.MustCompile(`\$\{[^}]*\}`)

// addSpec reads a Playwright spec for the requests it makes.
func (ix *index) addSpec(path, rel string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	u := unit{file: rel}

	for _, m := range specCall.FindAllStringSubmatch(string(raw), -1) {
		p, ok := pathOf(specInterp.ReplaceAllString(m[2], hole))
		if !ok {
			continue
		}

		method := strings.ToUpper(m[1])
		if method == "GOTO" {
			method = "GET"
		}

		ix.requests = append(ix.requests, request{method: method, path: p, unit: u})
	}

	return nil
}

// segments splits a path into its segments: "/a/b/" is [a b], "/" is [].
func segments(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}

	return strings.Split(p, "/")
}

// matches reports whether a request path template reaches a route pattern.
// A static segment of the route needs the same text in the test (a hole there
// would match any route, so it does not count); a `{param}` of the route takes
// anything; `{name...}` takes the rest.
func matches(route, tmpl string) bool {
	r, t := segments(strings.TrimSuffix(route, "{$}")), segments(tmpl)

	for i, seg := range r {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "...}") {
			return true
		}

		if i >= len(t) {
			return false
		}

		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			continue
		}

		if strings.Contains(t[i], hole) {
			re := "^" + strings.ReplaceAll(regexp.QuoteMeta(strings.ReplaceAll(t[i], hole, "\x02")), "\x02", ".+") + "$"
			if t[i] == hole || !regexp.MustCompile(re).MatchString(seg) {
				return false
			}

			continue
		}

		if seg != t[i] {
			return false
		}
	}

	return len(t) == len(r)
}

// coverRequest returns the units that make a request reaching method and route.
func (ix *index) coverRequest(method, route string) []unit {
	seen := map[unit]bool{}

	for _, rq := range ix.requests {
		if (method == "ANY" || rq.method == "" || rq.method == method) && matches(route, rq.path) {
			seen[rq.unit] = true
		}
	}

	return sortedUnits(seen)
}

// coverString returns the units that hold the exact string.
func (ix *index) coverString(s string) []unit { return sortedUnits(ix.strings[s]) }

// coverIdent returns the units that name the identifier.
func (ix *index) coverIdent(s string) []unit { return sortedUnits(ix.idents[s]) }

// coverCommand returns the units with a string that is the command, or the
// command followed by an argument.
func (ix *index) coverCommand(cmd string) []unit {
	seen := map[unit]bool{}

	for s, us := range ix.strings {
		if s == cmd || strings.HasPrefix(s, cmd+" ") || strings.HasPrefix(s, cmd+"@") {
			for u := range us {
				seen[u] = true
			}
		}
	}

	return sortedUnits(seen)
}

func sortedUnits(m map[unit]bool) []unit {
	out := make([]unit, 0, len(m))
	for u := range m {
		out = append(out, u)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].file != out[j].file {
			return out[i].file < out[j].file
		}

		return out[i].test < out[j].test
	})

	return out
}
