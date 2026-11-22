// Command drills runs the recovery drills of the quality campaign (epic 24,
// task 10) against a throwaway installation: backup and restore, an upgrade
// from a database the previous release made, a kill during a sync push and a
// kill during an import. Each drill proves that no row is lost and none is
// doubled. Run all of them with scripts/qa/drills.sh.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type drill struct {
	name string
	run  func(ctx context.Context) error
}

func main() {
	drills := []drill{
		{"backup-restore", backupRestore},
		{"upgrade", upgrade},
		{"kill-push", killPush},
		{"kill-import", killImport},
	}

	want := os.Args[1:]
	failed := 0

	for _, d := range drills {
		if len(want) > 0 && !contains(want, d.name) && !contains(want, "all") {
			continue
		}

		start := time.Now()
		err := d.run(context.Background())

		if err != nil {
			failed++

			fmt.Printf("FAIL  %-15s %s: %v\n", d.name, time.Since(start).Round(time.Millisecond), strings.TrimSpace(err.Error()))

			continue
		}

		fmt.Printf("ok    %-15s %s\n", d.name, time.Since(start).Round(time.Millisecond))
	}

	if failed > 0 {
		fmt.Printf("drills: %d failed\n", failed)
		os.Exit(1)
	}

	fmt.Println("drills: all passed")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}
