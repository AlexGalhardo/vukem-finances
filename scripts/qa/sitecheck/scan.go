package main

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Options are the facts of the site that are not in its files.
type Options struct {
	Origin string
	CNAME  string
}

// searchDir is the one folder whose JavaScript may read static files: Pagefind
// fetches its own index from the same origin (D-020).
const searchDir = "pagefind/"

var (
	networkAPI = regexp.MustCompile(`\bfetch\s*\(|XMLHttpRequest|WebSocket|EventSource|sendBeacon`)
	serverName = regexp.MustCompile(`(?i)worker|server|functions|_routes\.json|entry\.m?js`)
	formTag    = regexp.MustCompile(`(?i)<form[\s>/]`)
	tagRe      = regexp.MustCompile(`(?is)<(script|link|img|source|iframe|a|audio|video|embed|object)(?:\s[^>]*)?>`)
	attrRe     = regexp.MustCompile(`(?is)\s([a-z-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	cssURL     = regexp.MustCompile(`(?i)url\(\s*["']?([^"')\s]+)`)
	cssImport  = regexp.MustCompile(`(?i)@import\s+(?:url\()?\s*["']?([^"')\s;]+)`)
	scriptBody = regexp.MustCompile(`(?is)<script(?:\s[^>]*)?>(.*?)</script>`)
	scriptsAnd = regexp.MustCompile(`(?is)<(?:script|style)(?:\s[^>]*)?>.*?</(?:script|style)>`)
	anyTag     = regexp.MustCompile(`(?s)<[^>]+>`)
	spaces     = regexp.MustCompile(`\s+`)
	htmlLang   = regexp.MustCompile(`(?i)<html[^>]*\slang\s*=\s*["']?([A-Za-z-]+)`)

	// Paths that look like an account, purchase or API flow of an app.
	appPath = regexp.MustCompile(`(?i)^/(?:[a-z]{2}(?:-[a-z]{2})?/)?(?:api|graphql|login|log-in|signin|sign-in|register|signup|sign-up|checkout|billing|subscribe|contact|demo)(?:/|$|\?|#)`)
)

// forbidden mirrors the phrases of site/lib/claims.ts: they would imply a
// hosted service, a sign-up, a purchase or a demo. The landing and legal pages
// are checked, not the docs, which describe the CLI and its login command.
var forbidden = map[string][]string{
	"en":    {"sign up", "signup", "start free", "log in", "login", "subscribe now", "buy", "get started free", "hosted for you", "try the demo", "contact us"},
	"pt-br": {"cadastre-se", "comece grátis", "faça login", "entrar na sua conta", "assine agora", "comprar", "compre agora", "hospedado para você", "experimente a demo", "fale conosco"},
	"es":    {"regístrate", "empieza gratis", "inicia sesión", "suscríbete ahora", "comprar", "compra ahora", "alojado para ti", "prueba la demo", "contáctanos"},
}

// Scan crawls dir and returns every problem found, sorted.
func Scan(dir string, opt Options) ([]string, error) {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("sitecheck: %s is not a directory (build the site first)", dir)
	}

	base, err := url.Parse(opt.Origin)
	if err != nil || base.Host == "" {
		return nil, fmt.Errorf("sitecheck: bad origin %q", opt.Origin)
	}

	var problems []string

	add := func(rel, msg string) { problems = append(problems, rel+": "+msg) }

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if serverName.MatchString(e.Name()) {
			add(e.Name(), "a server or worker file in the static output")
		}
	}

	cname, err := os.ReadFile(filepath.Join(dir, "CNAME"))

	switch {
	case err != nil:
		add("CNAME", "missing")
	case strings.TrimSpace(string(cname)) != opt.CNAME:
		add("CNAME", fmt.Sprintf("is %q, want %q", strings.TrimSpace(string(cname)), opt.CNAME))
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		switch strings.ToLower(filepath.Ext(rel)) {
		case ".html":
			scanHTML(rel, string(body), base, add)
		case ".js", ".mjs", ".cjs":
			if !strings.HasPrefix(rel, searchDir) && networkAPI.Match(body) {
				add(rel, "calls a network API (fetch, XMLHttpRequest, WebSocket, EventSource or sendBeacon)")
			}

			if !strings.HasPrefix(rel, searchDir) && serverName.MatchString(filepath.Base(rel)) {
				add(rel, "a server or worker file in the static output")
			}
		case ".css":
			scanCSS(rel, string(body), base, add)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(problems)

	return problems, nil
}

func attrs(tag string) map[string]string {
	out := map[string]string{}

	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		out[strings.ToLower(m[1])] = m[2] + m[3] + m[4]
	}

	return out
}

func otherOrigin(raw string, base *url.URL) bool {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		raw = base.Scheme + ":" + raw
	}

	if !strings.Contains(raw, "://") {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil {
		return true
	}

	return u.Host != base.Host
}

func scanCSS(rel, css string, base *url.URL, add func(string, string)) {
	for _, m := range cssURL.FindAllStringSubmatch(css, -1) {
		if otherOrigin(m[1], base) {
			add(rel, "loads "+m[1]+" from another origin")
		}
	}

	for _, m := range cssImport.FindAllStringSubmatch(css, -1) {
		if otherOrigin(m[1], base) {
			add(rel, "imports "+m[1]+" from another origin")
		}
	}
}

func scanHTML(rel, html string, base *url.URL, add func(string, string)) {
	if formTag.MatchString(html) {
		add(rel, "contains a <form")
	}

	for _, m := range tagRe.FindAllStringSubmatch(html, -1) {
		name := strings.ToLower(m[1])
		a := attrs(m[0])

		if name == "a" {
			href := a["href"]
			if u, err := url.Parse(href); err == nil {
				path := href
				if u.Host != "" {
					if u.Host != base.Host {
						continue
					}

					path = u.Path
				}

				if appPath.MatchString(path) {
					add(rel, "links to "+href+", a page of an app, not of this static site")
				}
			}

			continue
		}

		if r := strings.ToLower(a["rel"]); strings.Contains(r, "canonical") || strings.Contains(r, "alternate") {
			continue
		}

		for _, key := range []string{"src", "href", "srcset", "data"} {
			f := strings.Fields(strings.ReplaceAll(a[key], ",", " "))
			if len(f) > 0 && otherOrigin(f[0], base) {
				add(rel, "<"+name+"> loads "+a[key]+" from another origin")
			}
		}
	}

	for _, m := range scriptBody.FindAllStringSubmatch(html, -1) {
		if networkAPI.MatchString(m[1]) {
			add(rel, "an inline script calls a network API")
		}
	}

	for _, m := range cssURL.FindAllStringSubmatch(html, -1) {
		if otherOrigin(m[1], base) {
			add(rel, "loads "+m[1]+" from another origin")
		}
	}

	if strings.HasPrefix(rel, "docs/") {
		return
	}

	lang := "en"
	if m := htmlLang.FindStringSubmatch(html); m != nil {
		lang = strings.ToLower(m[1])
	}

	phrases, ok := forbidden[lang]
	if !ok {
		return
	}

	text := spaces.ReplaceAllString(anyTag.ReplaceAllString(scriptsAnd.ReplaceAllString(html, " "), " "), " ")

	for _, p := range phrases {
		re := regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])` + strings.ReplaceAll(regexp.QuoteMeta(p), `\ `, `\s+`) + `(?:$|[^\p{L}\p{N}])`)
		if re.MatchString(text) {
			add(rel, fmt.Sprintf("the text contains the forbidden phrase %q", p))
		}
	}
}
