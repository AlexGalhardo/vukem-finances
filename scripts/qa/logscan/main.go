package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run scans every file named (or standard input for "-") and returns 0 when
// nothing was found, 1 when something was and 2 for a usage or read error.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("logscan", flag.ContinueOnError)
	fs.SetOutput(stderr)

	knownFile := fs.String("known", "", "file with one literal secret per line that must never appear")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: logscan [-known file] <file|dir|->...")
		return 2
	}

	var known []string

	if *knownFile != "" {
		f, err := os.Open(*knownFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			known = append(known, sc.Text())
		}

		_ = f.Close()
	}

	scanner := NewScanner(known)
	total, files := 0, 0

	for _, arg := range fs.Args() {
		paths, err := expand(arg)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}

		for _, p := range paths {
			r, closeFn, err := open(p, stdin)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}

			found, err := scanner.Scan(p, r)
			closeFn()

			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}

			files++

			for _, f := range found {
				fmt.Fprintln(stdout, f)
			}

			total += len(found)
		}
	}

	if total > 0 {
		fmt.Fprintf(stdout, "logscan: %d finding(s) in %d input(s)\n", total, files)
		return 1
	}

	fmt.Fprintf(stdout, "logscan: clean, %d input(s) scanned\n", files)

	return 0
}

func expand(arg string) ([]string, error) {
	if arg == "-" {
		return []string{arg}, nil
	}

	info, err := os.Stat(arg)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{arg}, nil
	}

	var out []string

	err = filepath.WalkDir(arg, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}

		return err
	})

	return out, err
}

func open(p string, stdin io.Reader) (io.Reader, func(), error) {
	if p == "-" {
		return stdin, func() {}, nil
	}

	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { _ = f.Close() }, nil
}
