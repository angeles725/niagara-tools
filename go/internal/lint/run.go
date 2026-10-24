package lint

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// Run scans every root in order: it lists the files any spec selects (pruning only the directories
// every spec prunes) with one walk, reads each file once, feeds it to every per-file Rule that selects
// it, in spec order, and after the root's walk feeds each TreeRule the ordered slice of the files it
// selects. Each spec's rows stay contiguous, in spec order: the first spec prints as it goes and the
// others are held and printed after it, each with its own stderr lines in place. An incomplete walk
// stops every spec that does not set ContinueOnWalkError (no rows from that root or any later one; the
// run ends once no spec is left) and the others scan the files listed. An incomplete walk or
// an unreadable file is reported on stderr when it happens with the texts of toolbelt/lib/awk-lint.sh,
// a file a per-file rule could not analyze as "cannot scan <path>: <rule>: <error>", a TreeRule error as
// its own text, and the scan goes on. Exit: 3 when anything could not be scanned (it wins over 1), else
// 1 when a FAIL row was printed or a WARN/FAIL row was printed under a.Strict, else 0 (PASS, ADVISORY
// and INFO rows never count). A spec prints its EmptyError row on a root where it selects no file and stops like
// a walk error (exit 3); so does a TreeRule or a RootRule that returns ErrEmptyRoot. A RootRule takes no part in the walk: it gets each root after the others' scan.
func Run(specs []Spec, name string, a Args, s cli.Stdio) int {
	r := &report{name: name, out: bufio.NewWriter(s.Out), err: s.Err, held: make([][]event, len(specs)), strict: a.Strict}
	defer r.out.Flush()
	sel, walks := union(specs)
	stopped := make([]bool, len(specs)) // rules a walk error has stopped (Spec.ContinueOnWalkError unset)
	live := len(specs)
	for _, root := range a.Roots {
		var files []string
		var err error
		if walks {
			files, err = Walk(root, sel)
		}
		if err != nil {
			r.envError(0, fmt.Sprintf("cannot list every file under %s: %s", root, err))
			for i, sp := range specs {
				if sp.Root == nil && !sp.ContinueOnWalkError && !stopped[i] {
					stopped[i], live = true, live-1
				}
			}
		} else { // a walk error wins: scan_files_or_exit exits before the K20 check
			for i, sp := range specs {
				if sp.Root == nil && !stopped[i] && !selectsAny(sp.Select, root, files) {
					r.empty(i, sp, root)
					stopped[i], live = true, live-1
				}
			}
		}
		if live == 0 {
			break // every rule exited: no file is scanned and no later root is walked
		}
		trees := make([][]*javasrc.File, len(specs))
		for _, p := range files {
			data, err := os.ReadFile(p)
			if err != nil {
				if shortScanError(specs, root, p) { // awk_lint_each printed the shell's reason only
					r.envError(0, fmt.Sprintf("cannot scan %s: %s", p, strerror.Text(err)))
				} else { // awk_lint_run printed the first line of awk's own error
					r.envError(0, fmt.Sprintf("cannot scan %s: awk: fatal: cannot open file `%s' for reading: %s", p, p, strerror.Text(err)))
				}
				continue
			}
			lines, _ := javasrc.Lines(bytes.NewReader(data)) // a bytes.Reader never fails
			f := javasrc.NewFile(p, lines)
			for i, sp := range specs {
				if stopped[i] || sp.Root != nil || !sp.Select.selects(baseName(p)) || prunedFor(sp.Select, root, p) {
					continue
				}
				if sp.Tree != nil {
					trees[i] = append(trees[i], f)
					continue
				}
				fds, err := sp.Rule.Check(f)
				shown := p
				if sp.RelPaths {
					shown = strings.TrimPrefix(p, root+"/")
				}
				r.findings(i, sp, shown, fds)
				if err != nil {
					r.envError(i, fmt.Sprintf("cannot scan %s: %s: %s", p, sp.ID(), err))
				}
			}
		}
		for i, sp := range specs {
			if sp.Tree == nil || stopped[i] {
				continue
			}
			fds, err := sp.Tree.CheckTree(trees[i])
			if errors.Is(err, ErrEmptyRoot) { // nothing it checks among the selected files: its ERROR row only
				r.empty(i, sp, root)
				stopped[i], live = true, live-1
				continue
			}
			r.findings(i, sp, "", fds)
			if err != nil {
				r.envError(i, err.Error())
			}
		}
		for i, sp := range specs {
			if sp.Root == nil || stopped[i] {
				continue
			}
			fds, err := sp.Root.CheckRoot(root)
			if errors.Is(err, ErrEmptyRoot) { // its ERROR row only, as for a walking rule's empty root
				r.empty(i, sp, root)
				stopped[i], live = true, live-1
				continue
			}
			r.findings(i, sp, "", fds)
			if err != nil {
				r.envError(i, err.Error())
			}
		}
	}
	r.release()
	switch {
	case r.envErr:
		return 3
	case r.failed, r.rows && a.Strict:
		return 1
	}
	return 0
}

// event is one held output line: a row on stdout or, when err is set, a message on stderr.
type event struct {
	err  bool
	text string
}

// report prints the first spec's output as it comes and holds every other spec's until release.
type report struct {
	name                 string
	out                  *bufio.Writer
	err                  io.Writer
	held                 [][]event
	envErr, rows, failed bool
	strict               bool // --strict: a Spec.StrictFails rule prints WARN rows as FAIL
}

func (r *report) emit(i int, e event) {
	if i > 0 {
		r.held[i] = append(r.held[i], e)
		return
	}
	if !e.err {
		fmt.Fprintln(r.out, e.text)
		return
	}
	r.out.Flush() // keep stdout and stderr in the order the bash lint printed them
	fmt.Fprintln(r.err, r.name+": "+e.text)
}

func (r *report) envError(i int, msg string) {
	r.envErr = true
	r.emit(i, event{err: true, text: msg})
}

// findings fills each finding's Rule, an Unset Severity and an empty Path, then emits its row.
func (r *report) findings(i int, sp Spec, path string, fds []Finding) {
	for _, fd := range fds {
		fd.Rule = sp.ID()
		if fd.Path == "" {
			fd.Path = path
		}
		if fd.Severity == Unset {
			fd.Severity = sp.severity()
		}
		if fd.Severity == Warn && r.strict && sp.StrictFails {
			fd.Severity = Fail
		}
		r.failed = r.failed || fd.Severity == Fail
		r.rows = r.rows || fd.Severity == Warn || fd.Severity == Fail
		r.emit(i, event{text: fd.row(sp.BarePathRow)})
	}
}

// empty prints the EmptyError row of a root where spec i selects no file.
func (r *report) empty(i int, sp Spec, root string) {
	check := sp.EmptyError.Check
	if check == "" {
		check = sp.ID()
	}
	r.envErr = true
	r.emit(i, event{text: "ERROR  " + check + "  " + root + "  " + sp.emptyMsg()})
}

// release prints the held output of every spec after the first, in spec order.
func (r *report) release() {
	for i := 1; i < len(r.held); i++ {
		for _, e := range r.held[i] {
			r.emit(0, e)
		}
	}
}

// union selects what any walking spec (every spec but a RootRule) selects and prunes what every walking
// spec prunes; walks reports whether there is a walking spec at all.
func union(specs []Spec) (u Selection, walks bool) {
	count := map[string]int{}
	n := 0
	for _, sp := range specs {
		if sp.Root != nil {
			continue
		}
		n++
		u.Names = append(u.Names, sp.Select.Names...)
		for _, d := range sp.Select.Prune {
			count[d]++
		}
	}
	for d, c := range count {
		if c == n {
			u.Prune = append(u.Prune, d)
		}
	}
	sort.Strings(u.Prune)
	return u, n > 0
}

// selectsAny reports whether sel selects any of the listed files under root (read or not, as the
// bash lints count their file list).
func selectsAny(sel Selection, root string, files []string) bool {
	for _, p := range files {
		if sel.selects(baseName(p)) && !prunedFor(sel, root, p) {
			return true
		}
	}
	return false
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
// argument contract (Mode, Usage, UsageExit, message prefix) of its toolbelt wrapper; "n4lint <rule>,<rule>..."
// and "n4lint --all" run several rules over one walk with the n4lint contract (Each mode, n4lint
// prefix), in the requested (or registry) order with each rule's rows contiguous. --all takes source
// roots, so it leaves out every RootRule (whose root is a module root, e.g. lint-structure's): such a
// rule runs only by name. Usage errors exit the spec's UsageExit (3 by default, always 3 for a rule list
// or --all), environment errors 3. A duplicate rule ID or a spec that is not exactly one of Rule,
// TreeRule and RootRule panics (a registration bug).
func Command(specs ...Spec) cli.Main {
	byID := map[string]Spec{}
	for _, sp := range specs {
		if err := sp.validate(); err != nil {
			panic(err.Error())
		}
		if _, dup := byID[sp.ID()]; dup {
			panic("lint: duplicate rule ID " + sp.ID())
		}
		byID[sp.ID()] = sp
	}
	return func(args []string, s cli.Stdio) int {
		if len(args) == 0 {
			fmt.Fprintln(s.Err, usage)
			return 3
		}
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Fprintf(s.Out, "%s\nrules:\n", usage)
			for _, sp := range specs {
				fmt.Fprintln(s.Out, sp.ID())
			}
			return 0
		}
		var selected []Spec
		if args[0] == "--all" {
			for _, sp := range specs {
				if sp.Root == nil && !sp.ModuleRoot { // --all runs on source roots
					selected = append(selected, sp)
				}
			}
		} else {
			for _, id := range strings.Split(args[0], ",") {
				sp, ok := byID[id]
				if !ok {
					fmt.Fprintf(s.Err, "n4lint: unknown rule: %s\n%s\n", id, usage)
					return 3
				}
				selected = append(selected, sp)
			}
		}
		mode, name, use, usageExit, bare := Each, "n4lint", usage, 3, false
		if len(selected) == 1 && args[0] != "--all" {
			sp := selected[0]
			mode, name, use, usageExit, bare = sp.Mode, sp.ID(), sp.Usage, sp.usageExit(), sp.BareUsage
		}
		a, err := Parse(mode, name, use, args[1:])
		if err != nil {
			var ue *UsageError
			usageErr := errors.As(err, &ue)
			if usageErr && bare {
				fmt.Fprintln(s.Err, use) // the usage line only, without Parse's "unknown flag" line
			} else {
				fmt.Fprint(s.Err, err)
			}
			if usageErr {
				return usageExit
			}
			return 3
		}
		return Run(selected, name, a, s)
	}
}

// shortScanError reports that every walking spec selecting file p words its read error as awk_lint_each did.
func shortScanError(specs []Spec, root, p string) bool {
	any := false
	for _, sp := range specs {
		if sp.Root != nil || !sp.Select.selects(baseName(p)) || prunedFor(sp.Select, root, p) {
			continue
		}
		if !sp.ShortScanError {
			return false
		}
		any = true
	}
	return any
}
