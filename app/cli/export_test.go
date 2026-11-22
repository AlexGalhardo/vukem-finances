package cli

import "io"

// WriteForTest writes a value in a format, as a command does.
func (c *CLI) WriteForTest(v any, format string) error {
	return c.write(Command{}, Response{Value: v}, format, "")
}

// WriteFileForTest streams a download to a file as --out does.
func (c *CLI) WriteFileForTest(target string, stream func(io.Writer) error) error {
	return c.writeFile(Response{Stream: stream}, target)
}
