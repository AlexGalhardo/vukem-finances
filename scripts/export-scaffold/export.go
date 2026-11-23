package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	modulePath = "github.com/AlexGalhardo/vukem-finances"
	appName    = "vukem-finances"

	manifestName = "MANIFEST.sha256"
	versionName  = "VERSION"
	licenseName  = "LICENSE.vukem-finances"
)

// File is one file of the tree, as git stores it.
type File struct {
	Path string
	Mode fs.FileMode
	Data []byte
}

// excludedDirs hold what only the app's own repository needs: the other
// clients, the site, the team's tooling and rules, generated output. scripts/
// is all QA and release tooling of this repository, none of it runtime.
var excludedDirs = []string{
	".agents/",
	".claude/",
	".config/",
	".githooks/",
	".github/",
	"development/",
	"desktop/",
	"docs/qa/",
	"mobile/",
	"resources/public/build/",
	"scripts/",
	"site/",
	"storage/",
}

// excludedFiles are single files: the module files (the Vukem CLI writes
// them for the chosen framework version), the changelog and the team's
// agent instructions. LICENSE is not lost: it is exported as
// LICENSE.vukem-finances.
var excludedFiles = []string{
	"AGENTS.md",
	"CHANGELOG.md",
	"CLAUDE.md",
	"LICENSE",
	"go.mod",
	"go.sum",
}

// keptFiles are inside an excluded directory but read by the app's own tests,
// which a generated project runs: the QA inventory counts the authorization
// cases of the matrix test, and the site data holds the plan limits.
var keptFiles = []string{
	"docs/qa/inventory.md",
	"site/generated/plans.json",
}

var imageExts = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".ico", ".icns", ".svg"}

// Include reports whether a tracked path belongs to the exported tree.
func Include(p string) bool {
	if slices.Contains(keptFiles, p) {
		return true
	}

	if slices.Contains(excludedFiles, p) {
		return false
	}

	for _, d := range excludedDirs {
		if strings.HasPrefix(p, d) {
			return false
		}
	}

	return !slices.Contains(imageExts, strings.ToLower(path.Ext(p)))
}

// templated reports whether placeholders may be written into the file. Docs
// and licenses keep their text: they describe the app, they are not code.
func templated(p string, data []byte) bool {
	if strings.HasSuffix(p, ".md") || strings.HasPrefix(path.Base(p), "LICENSE") {
		return false
	}

	return !bytes.Contains(data, []byte{0})
}

// Build turns the tracked files into the exported tree, sorted by path and
// including VERSION, the license notice and MANIFEST.sha256.
func Build(tracked []File, version string) ([]File, error) {
	var (
		out     []File
		license []byte
	)

	for _, f := range tracked {
		if f.Path == "LICENSE" {
			license = f.Data
		}

		if !Include(f.Path) {
			continue
		}

		if !templated(f.Path, f.Data) {
			out = append(out, f)

			continue
		}

		data, changed := Placeholders(f.Data)
		if changed {
			f.Path += ".tmpl"
			f.Data = data
		}

		out = append(out, f)
	}

	if license == nil {
		return nil, fmt.Errorf("the app has no LICENSE file to carry as %s", licenseName)
	}

	out = append(out,
		File{Path: versionName, Mode: 0o644, Data: []byte(version + "\n")},
		File{Path: licenseName, Mode: 0o644, Data: license},
	)

	slices.SortFunc(out, func(a, b File) int { return strings.Compare(a.Path, b.Path) })

	for i := 1; i < len(out); i++ {
		if out[i].Path == out[i-1].Path {
			return nil, fmt.Errorf("duplicate path %s in the exported tree", out[i].Path)
		}
	}

	var m bytes.Buffer

	for _, f := range out {
		fmt.Fprintf(&m, "%s  %s\n", hashHex(f.Data), f.Path)
	}

	out = append(out, File{Path: manifestName, Mode: 0o644, Data: m.Bytes()})
	slices.SortFunc(out, func(a, b File) int { return strings.Compare(a.Path, b.Path) })

	return out, nil
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)

	return hex.EncodeToString(sum[:])
}

// Placeholders replaces the module path with {{.Module}} and the app name
// with {{.Name}}. The Vukem CLI renders files with a .tmpl suffix through
// text/template, so in a file that gets a placeholder every literal "{{" is
// written as {{"{{"}} and renders back to itself. A file without a
// placeholder is returned untouched and not rendered at all.
func Placeholders(src []byte) ([]byte, bool) {
	s := string(src)
	hits := findHits(s)

	if len(hits) == 0 {
		return src, false
	}

	var b strings.Builder

	last := 0

	for _, h := range hits {
		b.WriteString(literal(s[last:h.start], true))
		b.WriteString(h.repl)

		last = h.end
	}

	b.WriteString(literal(s[last:], false))

	return []byte(b.String()), true
}

type hit struct {
	start, end int
	repl       string
}

// findHits locates the module path and the app name. The ones that point at
// the app's own published artifacts stay (see keepsName), and so does a
// module path preceded by "://", a link to the repository.
func findHits(s string) []hit {
	var hits []hit

	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], modulePath) && !strings.HasSuffix(s[:i], "://"):
			hits = append(hits, hit{i, i + len(modulePath), "{{.Module}}"})
			i += len(modulePath)
		case strings.HasPrefix(s[i:], appName) && keepsName(s[:i]):
			i += len(appName)
		case strings.HasPrefix(s[i:], appName):
			hits = append(hits, hit{i, i + len(appName), "{{.Name}}"})
			i += len(appName)
		default:
			i++
		}
	}

	return hits
}

// literal escapes the "{{" of text that sits next to a placeholder. A lone
// "{" right before a placeholder would join it into "{{{", so it is escaped
// too.
func literal(text string, beforeAction bool) string {
	text = strings.ReplaceAll(text, "{{", `{{"{{"}}`)

	if beforeAction && strings.HasSuffix(text, "{") {
		text = strings.TrimSuffix(text, "{") + `{{"{"}}`
	}

	return text
}

// WriteDir writes the tree to dir, which must not exist or be empty.
func WriteDir(dir string, files []File) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s is not empty", dir)
	}

	for _, f := range files {
		dst := filepath.Join(dir, filepath.FromSlash(f.Path))

		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}

		if err := os.WriteFile(dst, f.Data, f.Mode); err != nil {
			return err
		}
	}

	return nil
}

// WriteTar writes the tree as a tar stream with fixed owners and times, so
// the same tree is the same bytes.
func WriteTar(w io.Writer, files []File) error {
	tw := tar.NewWriter(w)

	for _, f := range files {
		h := &tar.Header{
			Name:     f.Path,
			Mode:     int64(f.Mode.Perm()),
			Size:     int64(len(f.Data)),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatPAX,
		}

		if err := tw.WriteHeader(h); err != nil {
			return err
		}

		if _, err := tw.Write(f.Data); err != nil {
			return err
		}
	}

	return tw.Close()
}

// keepsName reports whether an app name that follows before is a reference
// to the app itself and not the name of the generated project: the owner
// part of a repository or image path (github.com/AlexGalhardo/vukem-finances,
// ghcr.io/alexgalhardo/vukem-finances), or a Go type name that the OpenAPI
// builder mangled from the module path.
func keepsName(before string) bool {
	return strings.HasSuffix(strings.ToLower(before), "alexgalhardo/") || strings.HasSuffix(before, "_")
}
