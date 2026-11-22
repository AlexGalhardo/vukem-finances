// Command vukem-finances-desktop is the desktop edition of vukem-finances: the
// personal edition with local data, served on a loopback port behind a gate and
// shown in a native window.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// stopTimeout bounds the wait for the app to stop; the framework's own
// SHUTDOWN_TIMEOUT (15s) drains requests inside it.
const stopTimeout = 30 * time.Second

type flags struct {
	headless bool
	printURL bool
	dataDir  string
}

func parseFlags(args []string, errOut io.Writer) (flags, error) {
	var f flags

	fs := flag.NewFlagSet("vukem-finances-desktop", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.BoolVar(&f.headless, "headless", false, "start only the server and the gate, without a window")
	fs.BoolVar(&f.printURL, "print-url", false, "print the entry URL (with the one-time launch token) on stdout")
	fs.StringVar(&f.dataDir, "data-dir", "", "data directory (default: the vukem-finances folder of the user config directory)")

	return f, fs.Parse(args)
}

func main() {
	f, err := parseFlags(os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, f, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "vukem-finances:", err)

		if errors.Is(err, ErrAlreadyRunning) {
			os.Exit(3)
		}

		os.Exit(1)
	}
}

// run is the one code path of the app. The headless run differs only in not
// opening the window, so CI exercises everything but the webview.
func run(ctx context.Context, f flags, stdout io.Writer) error {
	var (
		paths Paths
		err   error
	)

	if f.dataDir != "" {
		paths, err = ResolvePaths(f.dataDir)
	} else {
		paths, err = DefaultPaths()
	}

	if err != nil {
		return err
	}

	lock, err := AcquireLock(paths.Lock)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Release() }()

	srv, err := StartServer(ctx, paths, startOptions{})
	if err != nil {
		return err
	}

	if f.printURL {
		fmt.Fprintln(stdout, srv.EntryURL())
	}

	if !f.headless {
		return errors.New("the window is not built yet; run with --headless")
	}

	select {
	case <-ctx.Done():
	case err := <-srv.Done():
		return err
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()

	return srv.Stop(stopCtx)
}
