package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
// (cookie, loopback Host, same-origin fetch).
type shell struct {
	tmpDir string
	// openURL opens an address in the system browser.
	openURL func(string) error
	// chooseSavePath asks for a destination through the native save dialog;
	// an empty path means the person cancelled.
	chooseSavePath func(name string) (string, error)
}

func (s *shell) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/desktop/save":
		s.save(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/desktop/open-external":
		s.openExternal(w, r)
	default:
		http.NotFound(w, r)
	}
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
// export holds private financial data).
func copyTo(src *os.File, dest string) error {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, src); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
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
