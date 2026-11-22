package cli

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Shells are the shells `completion` writes a script for.
var Shells = []string{"bash", "zsh", "fish", "powershell"}

// DefaultProgram is the name the scripts complete when none is given.
const DefaultProgram = "finances"

var programName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

var outputFormats = []string{"table", "json", "csv"}

// completionSpec is what a script offers: the resources, the verbs of each and
// the flags of each verb. Everything comes from the tree, so a new operation
// completes with no edit here.
type completionSpec struct {
	resources []string
	verbs     map[string][]string
	flags     map[string][]string
}

func (c *CLI) completionSpec() completionSpec {
	s := completionSpec{verbs: map[string][]string{}, flags: map[string][]string{}}

	s.resources = append(s.resources, c.Tree.Resources()...)

	for _, r := range s.resources {
		for _, cmd := range c.Tree.Of(r) {
			s.verbs[r] = append(s.verbs[r], cmd.Verb)
			s.flags[r+" "+cmd.Verb] = flagNames(cmd)
		}
	}

	s.resources = append(s.resources, "auth", "completion")
	s.verbs["auth"] = []string{"login", "logout", "status"}
	s.flags["auth login"] = []string{"--server"}
	s.verbs["completion"] = Shells

	sort.Strings(s.resources)

	return s
}

func flagNames(cmd Command) []string {
	var out []string

	for _, f := range cmd.Fields {
		if f.Positional >= 0 {
			continue
		}

		out = append(out, "--"+f.Flag)

		if f.Cents {
			out = append(out, "--"+strings.TrimSuffix(f.Flag, "-cents"))
		}
	}

	out = append(out, "--input", "--output", "--server", "--local")

	if cmd.Op.Output == downloadType {
		out = append(out, "--out")
	}

	if cmd.Op.Kind != "read" {
		out = append(out, "--if-match", "--idempotency-key")
	}

	if cmd.Destructive() {
		out = append(out, "--yes")
	}

	return out
}

// completion prints the script: `completion <shell> [program]`.
func (c *CLI) completion(args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return usagef("usage: completion %s [program]", strings.Join(Shells, "|"))
	}

	program := DefaultProgram
	if len(args) == 2 {
		program = args[1]
	}

	script, err := c.Completion(args[0], program)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(c.Env.Stdout, script)

	return err
}

// Completion returns the completion script of a shell for a program name.
func (c *CLI) Completion(shell, program string) (string, error) {
	if !programName.MatchString(program) {
		return "", usagef("the program name %q has characters a completion script cannot take", program)
	}

	spec := c.completionSpec()

	switch shell {
	case "bash":
		return spec.bash(program), nil
	case "zsh":
		return spec.zsh(program), nil
	case "fish":
		return spec.fish(program), nil
	case "powershell":
		return spec.powershell(program), nil
	}

	return "", usagef("completion needs a shell: %s", strings.Join(Shells, ", "))
}

func (s completionSpec) keys() []string {
	var out []string

	for k := range s.flags {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

func ident(program string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}

		return '_'
	}, program)
}

func (s completionSpec) bash(program string) string {
	var b strings.Builder

	fn := "_" + ident(program) + "_complete"

	fmt.Fprintf(&b, "# bash completion for %s\n%s() {\n", program, fn)
	b.WriteString("\tlocal cur prev\n\tcur=\"${COMP_WORDS[COMP_CWORD]}\"\n\tprev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n")
	fmt.Fprintf(&b, "\tif [ \"$prev\" = \"--output\" ] || [ \"$prev\" = \"-o\" ]; then\n\t\tCOMPREPLY=($(compgen -W %q -- \"$cur\"))\n\t\treturn\n\tfi\n", strings.Join(outputFormats, " "))
	fmt.Fprintf(&b, "\tif [ \"$COMP_CWORD\" -eq 1 ]; then\n\t\tCOMPREPLY=($(compgen -W %q -- \"$cur\"))\n\t\treturn\n\tfi\n", strings.Join(s.resources, " "))
	b.WriteString("\tif [ \"$COMP_CWORD\" -eq 2 ]; then\n\t\tcase \"${COMP_WORDS[1]}\" in\n")

	for _, r := range s.resources {
		fmt.Fprintf(&b, "\t\t\t%s) COMPREPLY=($(compgen -W %q -- \"$cur\")) ;;\n", r, strings.Join(s.verbs[r], " "))
	}

	b.WriteString("\t\tesac\n\t\treturn\n\tfi\n")
	b.WriteString("\tcase \"${COMP_WORDS[1]} ${COMP_WORDS[2]}\" in\n")

	for _, k := range s.keys() {
		fmt.Fprintf(&b, "\t\t%q) COMPREPLY=($(compgen -W %q -- \"$cur\")) ;;\n", k, strings.Join(s.flags[k], " "))
	}

	b.WriteString("\tesac\n}\n")
	fmt.Fprintf(&b, "complete -F %s %s\n", fn, program)

	return b.String()
}

func (s completionSpec) zsh(program string) string {
	var b strings.Builder

	fn := "_" + ident(program)

	fmt.Fprintf(&b, "#compdef %s\n%s() {\n", program, fn)
	fmt.Fprintf(&b, "\tif [[ \"$words[CURRENT-1]\" == (--output|-o) ]]; then\n\t\tcompadd -- %s\n\t\treturn\n\tfi\n", strings.Join(outputFormats, " "))
	fmt.Fprintf(&b, "\tif (( CURRENT == 2 )); then\n\t\tcompadd -- %s\n\t\treturn\n\tfi\n", strings.Join(s.resources, " "))
	b.WriteString("\tif (( CURRENT == 3 )); then\n\t\tcase \"$words[2]\" in\n")

	for _, r := range s.resources {
		fmt.Fprintf(&b, "\t\t\t%s) compadd -- %s ;;\n", r, strings.Join(s.verbs[r], " "))
	}

	b.WriteString("\t\tesac\n\t\treturn\n\tfi\n")
	b.WriteString("\tcase \"$words[2] $words[3]\" in\n")

	for _, k := range s.keys() {
		fmt.Fprintf(&b, "\t\t%q) compadd -- %s ;;\n", k, strings.Join(s.flags[k], " "))
	}

	b.WriteString("\tesac\n}\n")
	fmt.Fprintf(&b, "compdef %s %s\n", fn, program)

	return b.String()
}

func (s completionSpec) fish(program string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# fish completion for %s\ncomplete -c %s -f\n", program, program)
	fmt.Fprintf(&b, "complete -c %s -n '__fish_use_subcommand' -a %q\n", program, strings.Join(s.resources, " "))
	fmt.Fprintf(&b, "complete -c %s -l output -s o -x -a %q\n", program, strings.Join(outputFormats, " "))

	for _, r := range s.resources {
		fmt.Fprintf(&b, "complete -c %s -n '__fish_seen_subcommand_from %s; and not __fish_seen_subcommand_from %s' -a %q\n",
			program, r, strings.Join(s.verbs[r], " "), strings.Join(s.verbs[r], " "))
	}

	for _, k := range s.keys() {
		res, verb, _ := strings.Cut(k, " ")

		for _, f := range s.flags[k] {
			if f == "--output" {
				continue
			}

			fmt.Fprintf(&b, "complete -c %s -n '__fish_seen_subcommand_from %s; and __fish_seen_subcommand_from %s' -l %s\n",
				program, res, verb, strings.TrimPrefix(f, "--"))
		}
	}

	return b.String()
}

func (s completionSpec) powershell(program string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# PowerShell completion for %s\n", program)
	fmt.Fprintf(&b, "Register-ArgumentCompleter -Native -CommandName '%s' -ScriptBlock {\n", program)
	b.WriteString("\tparam($wordToComplete, $commandAst, $cursorPosition)\n")
	b.WriteString("\t$words = @($commandAst.CommandElements | ForEach-Object { $_.ToString() })\n")
	b.WriteString("\tif ($wordToComplete -ne '') { $words = $words[0..($words.Count - 2)] }\n")
	b.WriteString("\t$candidates = @()\n")
	fmt.Fprintf(&b, "\tif ($words.Count -ge 2 -and ($words[-1] -eq '--output' -or $words[-1] -eq '-o')) { $candidates = @(%s) }\n", psList(outputFormats))
	fmt.Fprintf(&b, "\telseif ($words.Count -eq 1) { $candidates = @(%s) }\n", psList(s.resources))
	b.WriteString("\telseif ($words.Count -eq 2) {\n\t\tswitch ($words[1]) {\n")

	for _, r := range s.resources {
		fmt.Fprintf(&b, "\t\t\t'%s' { $candidates = @(%s) }\n", r, psList(s.verbs[r]))
	}

	b.WriteString("\t\t}\n\t}\n\telse {\n\t\tswitch (\"$($words[1]) $($words[2])\") {\n")

	for _, k := range s.keys() {
		fmt.Fprintf(&b, "\t\t\t'%s' { $candidates = @(%s) }\n", k, psList(s.flags[k]))
	}

	b.WriteString("\t\t}\n\t}\n")
	b.WriteString("\t$candidates | Where-Object { $_ -like \"$wordToComplete*\" } | ForEach-Object {\n\t\t[System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)\n\t}\n}\n")

	return b.String()
}

func psList(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = "'" + w + "'"
	}

	return strings.Join(q, ", ")
}
