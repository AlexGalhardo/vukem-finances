package cli

import (
	"fmt"
	"strings"
)

// Reference is docs/reference/cli.md: the command reference, written from the
// tree so it cannot drift from the registry. The same tree always gives the
// same bytes.
func Reference(t *Tree) []byte {
	var b strings.Builder

	b.WriteString(strings.ReplaceAll(`# CLI reference

<!-- Generated from the operation registry: do not edit. Run ~go run ./cmd/app cli:reference~. -->

Every operation of the registry is a command ~<resource> <verb> [arguments] [flags]~.
The same registry feeds the REST API (~docs/api.md~), so a command and its endpoint
take the same input and give the same answer.

## Modes

- **Local** (~--local~): runs in this process as the owner of a personal installation
  (~APP_EDITION=personal~). Any other edition exits 3 and changes nothing.
- **Remote** (~--server URL~ or the stored login): calls the REST API with a token.
  ~auth login --server URL~ reads the token from the standard input; ~FINANCES_TOKEN~
  is read for CI. A token is never a flag.

## Shared flags

| Flag | Meaning |
| --- | --- |
| ~--output table, json or csv~ | Output format. The default is a table on a terminal and JSON when piped. |
| ~--input FILE~ or ~--input -~ | The whole input as a JSON object, from a file or the standard input. |
| ~--server URL~ | Run against this server. |
| ~--local~ | Run in this process (personal edition). |
| ~--if-match VERSION~ | Optimistic concurrency, for commands that change data. |
| ~--idempotency-key KEY~ | Safe retries, for commands that honor a key. |
| ~--yes~ | Skip the confirmation of a destructive command; required without a terminal. |
| ~--out FILE~ | Where a download is written (required on a terminal). |

An amount flag such as ~--amount~ takes ~12.50~ or ~1.234,56~; its ~--amount-cents~
twin takes whole cents. JSON output keeps amounts as exact decimal text next to integer cents.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Any other failure |
| 2 | Usage error, or a destructive command without a terminal and without ~--yes~ |
| 3 | Authentication or permission |
| 4 | Not found |
| 5 | Conflict |
| 6 | Rate limit or plan limit |

## Other commands

- ~auth login --server URL~, ~auth logout~, ~auth status~: manage the stored login.
- ~completion bash|zsh|fish|powershell [program]~: print a shell completion script.
`, "~", "`"))

	for _, r := range t.Resources() {
		fmt.Fprintf(&b, "\n## %s\n", r)

		for _, c := range t.Of(r) {
			writeCommand(&b, c)
		}
	}

	return []byte(b.String())
}

func writeCommand(b *strings.Builder, c Command) {
	fmt.Fprintf(b, "\n### `%s`\n\n%s.\n\n", c.Name(), strings.TrimRight(c.Op.Summary, "."))

	usage := c.Name()
	for _, p := range c.Params {
		usage += " <" + p + ">"
	}

	fmt.Fprintf(b, "```sh\n%s [flags]\n```\n\n", usage)

	if c.Method != "" {
		fmt.Fprintf(b, "- REST: `%s %s`\n", c.Method, c.Path)
	}

	fmt.Fprintf(b, "- Operation: `%s`, scope `%s`, %s\n", c.Op.Name, c.Op.Scope, c.Op.Kind)

	if c.Destructive() {
		b.WriteString("- Asks for confirmation on a terminal; needs `--yes` otherwise.\n")
	}

	var rows []string

	for _, f := range c.Fields {
		if f.Positional >= 0 {
			continue
		}

		need := ""
		if f.Required {
			need = "yes"
		}

		switch {
		case f.Cents:
			short := strings.TrimSuffix(f.Flag, "-cents")
			rows = append(rows,
				fmt.Sprintf("| `--%s` | whole cents | %s |", f.Flag, need),
				fmt.Sprintf("| `--%s` | amount, `12.50` or `1.234,56` | |", short))
		case f.Kind == KindBool:
			rows = append(rows, fmt.Sprintf("| `--%s` | true or false | %s |", f.Flag, need))
		case f.Kind == KindInt:
			rows = append(rows, fmt.Sprintf("| `--%s` | number | %s |", f.Flag, need))
		case f.Kind == KindStrings:
			rows = append(rows, fmt.Sprintf("| `--%s` | text, repeat for more than one | %s |", f.Flag, need))
		default:
			rows = append(rows, fmt.Sprintf("| `--%s` | text | %s |", f.Flag, need))
		}
	}

	if len(rows) > 0 {
		b.WriteString("\n| Flag | Value | Required |\n| --- | --- | --- |\n" + strings.Join(rows, "\n") + "\n")
	}
}
