// Command sitecheck audits the built static site (site/out) as an artifact:
// no form, no resource from another origin, no network call in the built
// JavaScript outside the Pagefind folder, no server or worker file, the right
// CNAME, and no page that offers a sign-up, login, checkout, contact or demo
// or implies a hosted service (D-019, D-020).
//
//	go run ./scripts/qa/sitecheck site/out
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns 0 when the site is clean, 1 when a problem was found and 2 for a
// usage or read error.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sitecheck", flag.ContinueOnError)
	fs.SetOutput(stderr)

	origin := fs.String("origin", "https://finances.vukem.dev", "the public origin of the site; links to it are not another origin")
	cname := fs.String("cname", "finances.vukem.dev", "the exact content of the CNAME file")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: sitecheck [-origin url] [-cname host] <site/out>")
		return 2
	}

	problems, err := Scan(fs.Arg(0), Options{Origin: *origin, CNAME: *cname})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	for _, p := range problems {
		fmt.Fprintln(stderr, p)
	}

	if len(problems) > 0 {
		fmt.Fprintf(stderr, "sitecheck: %d problem(s)\n", len(problems))

		return 1
	}

	fmt.Fprintln(stdout, "sitecheck ok")

	return 0
}
