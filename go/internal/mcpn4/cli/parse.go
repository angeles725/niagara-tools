package cli

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// UsageError is an argparse error: printed after the usage text as
// "mcp_n4.server: error: <Msg>", exit 2.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

// token is one classified argument (argparse _parse_optional).
type token struct {
	arg      string
	opt      *option // nil: a positional, or an unknown option when unknown is set
	explicit *string // the value after "=" (or after "-h")
	unknown  bool    // not consumable as a value: an unknown option, or the first "--"
	attached bool    // "-h<chars>": the chars are more short flags, not a value
}

var negativeNumber = regexp.MustCompile(`^-\d+$|^-\d*\.\d+$`)

func lookupOption(name string) *option {
	for i := range options {
		for _, n := range options[i].names {
			if n == name {
				return &options[i]
			}
		}
	}
	return nil
}

// classify is argparse's _parse_optional for this parser: exact option strings, NAME=VALUE,
// unique long prefixes (allow_abbrev), "-h" with an attached value; negative numbers and
// arguments with a space are positionals; anything else starting with "-" is an unknown option.
func classify(arg string) (token, *UsageError) {
	t := token{arg: arg}
	if arg == "" || arg[0] != '-' {
		return t, nil
	}
	if o := lookupOption(arg); o != nil {
		t.opt = o
		return t, nil
	}
	if len(arg) == 1 {
		return t, nil
	}
	name, value, hasEq := strings.Cut(arg, "=")
	if o := lookupOption(name); hasEq && o != nil {
		t.opt, t.explicit = o, &value
		return t, nil
	}
	var matches []*option
	var shown []string
	if strings.HasPrefix(arg, "--") {
		for i := range options {
			long := options[i].names[len(options[i].names)-1]
			if strings.HasPrefix(long, name) {
				matches, shown = append(matches, &options[i]), append(shown, long)
			}
		}
		if hasEq {
			t.explicit = &value
		}
	} else if strings.HasPrefix(arg, "-h") {
		rest := arg[2:]
		matches, shown, t.explicit, t.attached = append(matches, &options[0]), append(shown, "-h"), &rest, true
	}
	if len(matches) > 1 {
		return t, &UsageError{"ambiguous option: " + arg + " could match " + strings.Join(shown, ", ")}
	}
	if len(matches) == 1 {
		t.opt = matches[0]
		return t, nil
	}
	t.explicit = nil
	if negativeNumber.MatchString(arg) || strings.Contains(arg, " ") {
		return t, nil
	}
	t.unknown = true
	return t, nil
}

// Parse is parse_args(args). help is true when -h/--help was reached (print the help, exit 0).
// Errors come in argparse's order: an ambiguous option first, then the first bad option in
// argument order, then the unrecognized arguments.
func Parse(args []string) (cfg Config, help bool, err *UsageError) {
	cfg = Defaults()
	tokens := make([]token, 0, len(args))
	dd := false
	for _, a := range args {
		if !dd && a == "--" { // kept: with no positionals it is reported as unrecognized
			dd = true
			tokens = append(tokens, token{arg: a, unknown: true})
			continue
		}
		if dd {
			tokens = append(tokens, token{arg: a}) // after the first "--": a positional
			continue
		}
		t, uerr := classify(a)
		if uerr != nil {
			return cfg, false, uerr
		}
		tokens = append(tokens, t)
	}
	var extras []string
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.opt == nil {
			extras = append(extras, t.arg)
			continue
		}
		o := t.opt
		if o.flag {
			// "-hx" runs -h and leaves "-x" unrecognized; help wins (argparse
			// consume_optional), unless the next char is "-" again.
			if t.attached && (*t.explicit)[0] != '-' {
				return cfg, true, nil
			}
			if t.explicit != nil {
				return cfg, false, &UsageError{"argument " + o.display() + ": ignored explicit argument " + pyjson.StrRepr(*t.explicit)}
			}
			if o.set == nil { // -h/--help
				return cfg, true, nil
			}
			_ = o.set(&cfg, "")
			continue
		}
		var value string
		switch {
		case t.explicit != nil:
			value = *t.explicit
		case i+1 < len(tokens) && tokens[i+1].opt == nil && !tokens[i+1].unknown:
			i++
			value = tokens[i].arg
		default:
			return cfg, false, &UsageError{"argument " + o.display() + ": expected one argument"}
		}
		if serr := o.set(&cfg, value); serr != nil {
			return cfg, false, &UsageError{"argument " + o.display() + ": " + serr.Error()}
		}
	}
	if len(extras) > 0 {
		return cfg, false, &UsageError{"unrecognized arguments: " + strings.Join(extras, " ")}
	}
	return cfg, false, nil
}
