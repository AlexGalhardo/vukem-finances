// Command fuzz-targets lists the Go fuzz targets of the repository, one name per
// line, so a loop or a CI matrix can run each of them (`go test -fuzz` runs one
// target of one package at a time). With -packages it prints "<name> <package>"
// instead, because -fuzz refuses a pattern that matches several packages:
//
//	go run ./scripts/qa/fuzz-targets -packages | while read name pkg; do
//	    go test -run='^$' -fuzz="^$name$" -fuzztime=30s "$pkg" || exit 1
//	done
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var target = regexp.MustCompile(`^func (Fuzz[A-Za-z0-9_]*)\(\w+ \*testing\.F\)`)

// skipped are the folders that are not part of the Go module under test.
var skipped = map[string]bool{
	".git": true, "node_modules": true, "desktop": true, "mobile": true, "site": true, "storage": true, "vendor": true, "testdata": true,
}

func main() {
	packages := flag.Bool("packages", false, "print the package of each target too")
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	found, err := scan(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fuzz-targets:", err)
		os.Exit(1)
	}

	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		if *packages {
			fmt.Printf("%s %s\n", name, found[name])
		} else {
			fmt.Println(name)
		}
	}
}

// scan returns every Fuzz function of the *_test.go files under root with the
// package directory that holds it. A name used twice is an error: `-fuzz`
// would match both.
func scan(root string) (map[string]string, error) {
	found := map[string]string{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != root && skipped[d.Name()] {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()

		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}

		pkg := "./" + filepath.ToSlash(rel)
		if rel == "." {
			pkg = "."
		}

		sc := bufio.NewScanner(file)
		for sc.Scan() {
			m := target.FindStringSubmatch(sc.Text())
			if m == nil {
				continue
			}

			if other, dup := found[m[1]]; dup {
				return fmt.Errorf("%s is defined in %s and in %s", m[1], other, pkg)
			}

			found[m[1]] = pkg
		}

		return sc.Err()
	})

	return found, err
}
