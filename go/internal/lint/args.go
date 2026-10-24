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
	Leading              // [--strict] <root>...: only a first --strict is a flag (-h, -- or a second --strict is a root); only the first root counts
	Anywhere             // [--strict] <root> [--strict]: --strict anywhere; another "-" argument or a second root is an env error naming it (lint-wb-threading)
	Valued               // [<option> <value>]... <root>: Spec.Option in any position, the last value wins; any other "-" argument, a missing value or a second root is the usage line alone
)

// Args is a parsed command line.
type Args struct {
	Strict bool
	Roots  []string
	Option string // the value of Spec.Option (Mode Valued), "" when absent
}

// UsageError is a Parse error that prints the usage line (a wrong argument count or an unknown flag);
// its exit code is the spec's UsageExit. Every other Parse error is an environment error (exit 3).
type UsageError struct{ Text string }

func (e *UsageError) Error() string { return e.Text }

// Parse reads args under mode. The error text is what the bash lint printed on stderr before exiting
// 3: the usage line, "<name>: unknown flag: <flag>" plus the usage line, "<name>: unknown option: <arg>"
// (Trailing), or "<name>: not a directory: <root>".
func Parse(mode Mode, name, usage string, args []string) (Args, error) {
	return ParseOption(mode, name, usage, "", args)
}

// ParseOption is Parse for a mode that takes a value option (Valued: option is its name, e.g. "--wiring-map").
func ParseOption(mode Mode, name, usage, option string, args []string) (Args, error) {
	var a Args
	if mode == Valued {
		// the bash `while [ $# -gt 0 ]; case "$1" in <option>) [ $# -ge 2 ] || usage; v=$2; shift 2 ;;
		// -*) usage ;; *) [ -z "$root" ] || usage; root=$1 ;; esac`
		var roots []string
		for i := 0; i < len(args); i++ {
			switch arg := args[i]; {
			case arg == option:
				if i+1 >= len(args) {
					return Args{}, &UsageError{usage + "\n"}
				}
				a.Option, i = args[i+1], i+1
			case strings.HasPrefix(arg, "-") || len(roots) == 1:
				return Args{}, &UsageError{usage + "\n"}
			default:
				roots = append(roots, arg)
			}
		}
		args, mode = roots, Raw // the root check below
	}
	if mode == Trailing && len(args) > 0 {
		for _, arg := range args[1:] {
			if arg != "--strict" {
				return Args{}, &UsageError{name + ": unknown option: " + arg + "\n"}
			}
			a.Strict = true
		}
		args = args[:1]
	}
	if mode == Anywhere {
		// the bash `while [ $# -gt 0 ]; case "$1" in --strict) ;; -*) unknown option ;; *) [ -z "$SRC" ] ||
		// unexpected argument ;; esac` after a `[ $# -lt 1 ]` usage check
		if len(args) == 0 {
			return Args{}, &UsageError{usage + "\n"}
		}
		var roots []string
		for _, arg := range args {
			switch {
			case arg == "--strict":
				a.Strict = true
			case strings.HasPrefix(arg, "-"):
				return Args{}, errors.New(name + ": unknown option: " + arg + "\n")
			case len(roots) == 1:
				return Args{}, errors.New(name + ": unexpected argument: " + arg + "\n")
			default:
				roots = append(roots, arg)
			}
		}
		args, mode = roots, Raw // the usage and root checks below
	}
	if mode == Leading && len(args) > 0 && args[0] == "--strict" {
		// the bash `[ "${1:-}" = "--strict" ] && { STRICT=1; shift; }`: one optional flag, then raw roots
		a.Strict, args = true, args[1:]
	}
flags:
	for mode != Raw && mode != Exact && mode != Trailing && mode != Leading && len(args) > 0 {
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
