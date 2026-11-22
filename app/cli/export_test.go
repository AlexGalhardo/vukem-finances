package cli

// WriteForTest writes a value in a format, as a command does.
func (c *CLI) WriteForTest(v any, format string) error {
	return c.write(Command{}, Response{Value: v}, format, "")
}
