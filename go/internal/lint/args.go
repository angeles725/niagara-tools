package lint

import (
	"errors"
	"os"
	"strings"
)

// Mode is how a lint takes its arguments; each one is an awk_lint_parse arity of toolbelt/lib/awk-lint.sh,
// so a ported lint keeps the argument contract of the bash lint it replaces.
type Mode int

const (
	One      Mode = iota // [--strict] [--] <root>: exactly one root
	First                // [--strict] [--] <root>...: only the first root is checked and scanned
	Each                 // [--strict] [--] <root>...: every root is checked first, then scanned in order
	Raw                  // <root>...: no flag is parsed (-h, -- or --strict is a root); only the first counts
	Exact                // <root>: exactly one argument, no flag is parsed (-h, -- or --strict is the root)
	Trailing             // <root> [--strict]...: the root as given first; only --strict may follow
)

// Args is a parsed command line.
type Args struct {
	Strict bool
	Roots  []string
}

// UsageError is a Parse error that prints the usage line (a wrong argument count or an unknown flag);
// its exit code is the spec's UsageExit. Every other Parse error is an environment error (exit 3).
type UsageError struct{ Text string }

func (e *UsageError) Error() string { return e.Text }

// Parse reads args under mode. The error text is what the bash lint printed on stderr before exiting
// 3: the usage line, "<name>: unknown flag: <flag>" plus the usage line, "<name>: unknown option: <arg>"
// (Trailing), or "<name>: not a directory: <root>".
func Parse(mode Mode, name, usage string, args []string) (Args, error) {
	var a Args
	if mode == Trailing && len(args) > 0 {
		for _, arg := range args[1:] {
			if arg != "--strict" {
				return Args{}, &UsageError{name + ": unknown option: " + arg + "\n"}
			}
			a.Strict = true
		}
		args = args[:1]
	}
flags:
	for mode != Raw && mode != Exact && mode != Trailing && len(args) > 0 {
		switch arg := args[0]; {
		case arg == "--strict":
			a.Strict = true
			args = args[1:]
		case arg == "--":
			args = args[1:]
			break flags
		case strings.HasPrefix(arg, "-"):
			return Args{}, &UsageError{name + ": unknown flag: " + arg + "\n" + usage + "\n"}
		default:
			break flags
		}
	}
	if len(args) < 1 || ((mode == One || mode == Exact || mode == Trailing) && len(args) != 1) {
		return Args{}, &UsageError{usage + "\n"}
	}
	a.Roots = args[:1]
	if mode == Each {
		a.Roots = args
	}
	for _, root := range a.Roots {
		if st, err := os.Stat(root); err != nil || !st.IsDir() {
			return Args{}, errors.New(name + ": not a directory: " + root + "\n")
		}
	}
	return a, nil
}
