package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"strconv"
	"strings"
)

// readTree returns the files of the revision as git stores them. Reading the
// blobs, not the working tree, keeps the bytes the same on every checkout:
// a Windows clone with CRLF files exports exactly what a Linux one does.
func readTree(repo, ref string) ([]File, error) {
	listing, err := exec.Command("git", "-C", repo, "ls-tree", "-r", "-z", "--full-tree", ref).Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: %w", err)
	}

	type entry struct {
		path, hash string
		mode       fs.FileMode
	}

	var entries []entry

	for rec := range bytes.SplitSeq(listing, []byte{0}) {
		if len(rec) == 0 {
			continue
		}

		meta, path, ok := strings.Cut(string(rec), "\t")
		if !ok {
			return nil, fmt.Errorf("unexpected ls-tree record %q", rec)
		}

		fields := strings.Fields(meta)
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected ls-tree record %q", rec)
		}

		if fields[1] != "blob" || fields[0] == "120000" {
			continue
		}

		mode := fs.FileMode(0o644)
		if fields[0] == "100755" {
			mode = 0o755
		}

		entries = append(entries, entry{path: path, hash: fields[2], mode: mode})
	}

	cmd := exec.Command("git", "-C", repo, "cat-file", "--batch")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		defer stdin.Close()

		for _, e := range entries {
			_, _ = io.WriteString(stdin, e.hash+"\n")
		}
	}()

	r := bufio.NewReader(stdout)
	files := make([]File, 0, len(entries))

	for _, e := range entries {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file: %w", err)
		}

		parts := strings.Fields(header)
		if len(parts) != 3 {
			return nil, fmt.Errorf("git cat-file: unexpected header %q", header)
		}

		size, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
		}

		data := make([]byte, size+1)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("git cat-file: %w", err)
		}

		files = append(files, File{Path: e.path, Mode: e.mode, Data: data[:size]})
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git cat-file: %w", err)
	}

	return files, nil
}
