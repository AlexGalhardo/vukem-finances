package ledger

import (
	"errors"
	"net/http"
	"testing"
)

type fakeDownload struct {
	written bool
	header  http.Header
}

func (f fakeDownload) Written() bool       { return f.written }
func (f fakeDownload) Header() http.Header { return f.header }

func TestFinishDownload(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	attachment := func() http.Header { return http.Header{"Content-Disposition": {"attachment"}} }

	if err := finishDownload(fakeDownload{written: true, header: attachment()}, nil); err != nil {
		t.Fatalf("a finished download = %v", err)
	}

	before := fakeDownload{header: attachment()}
	if err := finishDownload(before, boom); !errors.Is(err, boom) {
		t.Fatalf("a failure before the first byte = %v", err)
	}

	if before.header.Get("Content-Disposition") != "" {
		t.Error("an error page would be saved as the file")
	}

	defer func() {
		if p := recover(); p != http.ErrAbortHandler { //nolint:errorlint // recover() value, compared by identity
			t.Fatalf("a failure mid-stream recovered %v, want the connection aborted", p)
		}
	}()

	_ = finishDownload(fakeDownload{written: true, header: attachment()}, boom)
	t.Fatal("a cut download ended as if it were complete")
}
