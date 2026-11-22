package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// shellJS is injected into the window after each page load.
//
//go:embed shell.js
var shellJS string

// maxSaveBytes bounds one download sent through /desktop/save.
const maxSaveBytes = 512 << 20

// shell answers the /desktop/ paths: the native actions the web page cannot do
// itself. It is mounted behind the gate, so only the app window reaches it
// (cookie, loopback Host, same-origin fetch). The Wails runtime cannot do this
// job: it ignores messages from a page that is not served by Wails, and the
// window loads the app from the loopback server.
type shell struct {
	tmpDir string
	// openURL opens an address in the system browser.
	openURL func(string) error
	// chooseSavePath asks for a destination through the native save dialog;
	// an empty path means the person cancelled.
	chooseSavePath func(name string) (string, error)
}

// Wrap serves the /desktop/ paths itself and passes everything else to app,
// adding the shell script to its HTML pages.
func (s *shell) Wrap(app http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/desktop/shell.js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = io.WriteString(w, shellJS)
		case r.Method == http.MethodPost && r.URL.Path == "/desktop/save":
			s.save(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/desktop/open-external":
			s.openExternal(w, r)
		case strings.HasPrefix(r.URL.Path, "/desktop/"):
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.Header.Get("X-Inertia") == "" && strings.Contains(r.Header.Get("Accept"), "text/html"):
			// The page is rewritten, so it must arrive uncompressed.
			r = r.Clone(r.Context())
			r.Header.Del("Accept-Encoding")

			iw := &injectWriter{ResponseWriter: w}
			app.ServeHTTP(iw, r)
			iw.finish()
		default:
			app.ServeHTTP(w, r)
		}
	})
}

// scriptTag loads the shell script; the CSP of the app allows 'self' scripts.
const scriptTag = `<script src="/desktop/shell.js" defer></script>`

// injectWriter buffers an HTML page to add the script tag before </head>; any
// other response streams through untouched.
type injectWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
	html   bool
	wrote  bool
}

func (w *injectWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}

	w.wrote = true
	w.status = status
	w.html = strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") && w.Header().Get("Content-Encoding") == ""

	if !w.html {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *injectWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(p))
		}

		w.WriteHeader(http.StatusOK)
	}

	if !w.html {
		return w.ResponseWriter.Write(p)
	}

	return w.buf.Write(p)
}

func (w *injectWriter) Flush() {
	if !w.html {
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// Unwrap lets http.ResponseController reach the real writer.
func (w *injectWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// finish sends the buffered page with the tag inserted.
func (w *injectWriter) finish() {
	if !w.html {
		return
	}

	page := w.buf.Bytes()
	if i := bytes.Index(page, []byte("</head>")); i >= 0 {
		page = append(page[:i:i], append([]byte(scriptTag), page[i:]...)...)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(page)))
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(page)
}

// openExternal opens a web or mail address in the system browser. Any other
// scheme (file:, javascript:, a custom protocol handler) is refused: the page
// must not be able to start programs.
func (s *shell) openExternal(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<10))
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(strings.TrimSpace(string(raw)))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "mailto") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := s.openURL(u.String()); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// save receives the bytes of a download, asks where to put them and writes the
// file. 204 means the person cancelled the dialog.
func (s *shell) save(w http.ResponseWriter, r *http.Request) {
	name := safeFileName(r.URL.Query().Get("name"))

	tmp, err := os.CreateTemp(s.tmpDir, "save-*")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, http.MaxBytesReader(w, r.Body, maxSaveBytes)); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "Bad Request", http.StatusBadRequest)

		return
	}

	dest, err := s.chooseSavePath(name)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if dest == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := copyTo(tmp, dest); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"saved": true, "name": filepath.Base(dest)})
}

// copyTo writes the whole of src to dest (created or replaced, owner-only: an
// export holds private financial data). It writes a file beside dest and
// renames it, so a full disk cannot destroy the backup being replaced.
func copyTo(src io.ReadSeeker, dest string) (err error) {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}

	out, err := os.CreateTemp(filepath.Dir(dest), ".vukem-finances-save-*")
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = out.Close()
			_ = os.Remove(out.Name())
		}
	}()

	if _, err := io.Copy(out, src); err != nil {
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	return os.Rename(out.Name(), dest)
}

// safeFileName is the suggested name of a download: a plain file name, never a
// path.
func safeFileName(name string) string {
	name = strings.NewReplacer("\\", "/", "\x00", "").Replace(name)
	name = path.Base(name)

	if name == "." || name == ".." || name == "/" || strings.TrimSpace(name) == "" {
		return "download"
	}

	return name
}
