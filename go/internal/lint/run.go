package lint

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// Run scans every root in order: it lists the files any spec selects (pruning only the directories
// every spec prunes), reads each file once and runs the specs that select it, in spec order, printing
// their rows. An incomplete walk or an unreadable file is reported on stderr with the texts of
// toolbelt/lib/awk-lint.sh, a file a rule could not analyze as "cannot scan <path>: <rule>: <error>",
// and the scan goes on. Exit: 3 when anything could not be scanned, else 1
// when a row was printed under a.Strict, else 0.
func Run(specs []Spec, name string, a Args, s cli.Stdio) int {
	out := bufio.NewWriter(s.Out)
	defer out.Flush()
	fail := func(format string, v ...any) {
		out.Flush() // keep stdout and stderr in the order the bash lint printed them
		fmt.Fprintf(s.Err, name+": "+format+"\n", v...)
	}
	sel := union(specs)
	envErr, rows := false, false
	for _, root := range a.Roots {
		files, err := Walk(root, sel)
		if err != nil {
			fail("cannot list every file under %s: %s", root, err)
			envErr = true
		}
		for _, p := range files {
			data, err := os.ReadFile(p)
			if err != nil {
				// awk_lint_run printed the first line of awk's own error.
				fail("cannot scan %s: awk: fatal: cannot open file `%s' for reading: %s", p, p, strerror(err))
				envErr = true
				continue
			}
			lines, _ := javasrc.Lines(bytes.NewReader(data)) // a bytes.Reader never fails
			f := javasrc.NewFile(p, lines)
			for _, sp := range specs {
				if !sp.Select.selects(baseName(p)) || prunedFor(sp.Select, root, p) {
					continue
				}
				fds, err := sp.Rule.Check(f)
				for _, fd := range fds {
					fd.Rule, fd.Path = sp.Rule.ID(), p
					if fd.Severity == Unset {
						fd.Severity = sp.Rule.Severity()
					}
					fmt.Fprintln(out, fd.Row())
					rows = true
				}
				if err != nil {
					fail("cannot scan %s: %s: %s", p, sp.Rule.ID(), err)
					envErr = true
				}
			}
		}
	}
	switch {
	case envErr:
		return 3
	case rows && a.Strict:
		return 1
	}
	return 0
}

// union selects what any spec selects and prunes what every spec prunes.
func union(specs []Spec) Selection {
	var u Selection
	count := map[string]int{}
	for _, sp := range specs {
		u.Names = append(u.Names, sp.Select.Names...)
		for _, d := range sp.Select.Prune {
			count[d]++
		}
	}
	for d, n := range count {
		if n == len(specs) {
			u.Prune = append(u.Prune, d)
		}
	}
	sort.Strings(u.Prune)
	return u
}

func baseName(p string) string { return p[strings.LastIndexByte(p, '/')+1:] }

// prunedFor reports whether a directory between root and file p is one sel prunes.
func prunedFor(sel Selection, root, p string) bool {
	if len(sel.Prune) == 0 {
		return false
	}
	dirs := strings.Split(strings.TrimPrefix(p, strings.TrimSuffix(root, "/")+"/"), "/")
	for _, d := range dirs[:len(dirs)-1] {
		if d != "" && sel.prunes(d) {
			return true
		}
	}
	return false
}

const usage = "usage: n4lint <rule>[,<rule>...] <args>... | n4lint --all [--strict] [--] <src-root>..."

// Command is the n4lint entry point over specs. "n4lint <rule> <args>..." runs one rule with the
// argument contract (Mode, Usage, message prefix) of its toolbelt wrapper; "n4lint <rule>,<rule>..."
// and "n4lint --all" run several rules over one walk with the n4lint contract (Each mode, n4lint
// prefix). Usage and environment errors exit 3. A duplicate rule ID panics (a registration bug).
func Command(specs ...Spec) cli.Main {
	byID := map[string]Spec{}
	for _, sp := range specs {
		if _, dup := byID[sp.Rule.ID()]; dup {
			panic("lint: duplicate rule ID " + sp.Rule.ID())
		}
		byID[sp.Rule.ID()] = sp
	}
	return func(args []string, s cli.Stdio) int {
		if len(args) == 0 {
			fmt.Fprintln(s.Err, usage)
			return 3
		}
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Fprintf(s.Out, "%s\nrules:\n", usage)
			for _, sp := range specs {
				fmt.Fprintln(s.Out, sp.Rule.ID())
			}
			return 0
		}
		selected := specs
		if args[0] != "--all" {
			selected = nil
			for _, id := range strings.Split(args[0], ",") {
				sp, ok := byID[id]
				if !ok {
					fmt.Fprintf(s.Err, "n4lint: unknown rule: %s\n%s\n", id, usage)
					return 3
				}
				selected = append(selected, sp)
			}
		}
		mode, name, use := Each, "n4lint", usage
		if len(selected) == 1 && args[0] != "--all" {
			mode, name, use = selected[0].Mode, selected[0].Rule.ID(), selected[0].Usage
		}
		a, err := Parse(mode, name, use, args[1:])
		if err != nil {
			fmt.Fprint(s.Err, err)
			return 3
		}
		return Run(selected, name, a, s)
	}
}
