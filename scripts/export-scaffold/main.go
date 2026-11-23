// Command export-scaffold writes the tree that the Vukem CLI embeds as its
// finance example: every tracked file of this app that a generated project
// should own, with the module path and the app name turned into template
// placeholders. See docs/scaffold.md.
package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "export-scaffold:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("export-scaffold", flag.ContinueOnError)
	out := fs.String("out", "", "directory to write the tree to (must not exist or be empty)")
	archive := fs.String("archive", "", "also write the tree as a deterministic .tar.gz at this path")
	repo := fs.String("repo", ".", "path of the app repository")
	ref := fs.String("ref", "HEAD", "git revision to export")
	version := fs.String("version", "", "release tag recorded in VERSION (default: the tag at -ref, else \"dev\")")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *out == "" && *archive == "" {
		return fmt.Errorf("-out or -archive is required")
	}

	tracked, err := readTree(*repo, *ref)
	if err != nil {
		return err
	}

	tag := *version
	if tag == "" {
		tag = tagAt(*repo, *ref)
	}

	files, err := Build(tracked, tag)
	if err != nil {
		return err
	}

	if *out != "" {
		if err := WriteDir(*out, files); err != nil {
			return err
		}
	}

	if *archive != "" {
		var buf bytes.Buffer

		zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			return err
		}

		if err := WriteTar(zw, files); err != nil {
			return err
		}

		if err := zw.Close(); err != nil {
			return err
		}

		if err := os.WriteFile(*archive, buf.Bytes(), 0o644); err != nil {
			return err
		}
	}

	var size int
	for _, f := range files {
		size += len(f.Data)
	}

	fmt.Printf("exported %d files, %d bytes, version %s\n", len(files), size, tag)

	return nil
}

func tagAt(repo, ref string) string {
	cmd := exec.Command("git", "-C", repo, "describe", "--tags", "--exact-match", ref)

	b, err := cmd.Output()
	if err != nil {
		return "dev"
	}

	return strings.TrimSpace(string(b))
}
