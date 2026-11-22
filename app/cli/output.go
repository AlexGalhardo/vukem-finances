package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func (c *CLI) write(_ Command, res Response, _ string, outFile string) error {
	if res.Stream != nil {
		return c.writeFile(res, outFile)
	}

	return writeJSON(c.Env.Stdout, res.Value)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	return enc.Encode(v)
}

func (c *CLI) writeFile(res Response, outFile string) error {
	if outFile == "" {
		return res.Stream(c.Env.Stdout)
	}

	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("writing %s: %w", outFile, err)
	}

	if err := res.Stream(f); err != nil {
		_ = f.Close()

		return err
	}

	return f.Close()
}
