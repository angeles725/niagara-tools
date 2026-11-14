package livesmoke

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Args is the parsed command line.
type Args struct {
	Apply, AllowHTTPForTests               bool
	Stations, StationHomes, InsecureTLS    []string // NAME=URL, NAME=PATH, NAME; in order
	AllowTierB, AllowTierC                 []string // station names the operator opted in, in order
	CredentialEnv, StateDir, Report, GoBin string
	Settle, CallTimeout                    float64
}

// UsageError is an argparse error, printed after the usage text (exit 2).
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

// errHelp asks for the help text on stdout (exit 0).
var errHelp = &UsageError{"help"}

// option is one argparse action, in add_argument order (the order ambiguity errors list).
type option struct {
	names []string
	value bool // takes one argument
	set   func(a *Args, v string) error
}

func (o option) display() string { return strings.Join(o.names, "/") }

func floatSetter(field func(*Args) *float64) func(*Args, string) error {
	return func(a *Args, v string) error {
		f, ok := pyjson.ParseFloat(v)
		if !ok {
			return fmt.Errorf("invalid float value: %s", pyjson.StrRepr(v))
		}
		*field(a) = f
		return nil
	}
}

var options = []option{
	{names: []string{"-h", "--help"}},
	{names: []string{"--apply"}, set: func(a *Args, _ string) error { a.Apply = true; return nil }},
	{names: []string{"--station"}, value: true, set: func(a *Args, v string) error { a.Stations = append(a.Stations, v); return nil }},
	{names: []string{"--station-home"}, value: true, set: func(a *Args, v string) error { a.StationHomes = append(a.StationHomes, v); return nil }},
	{names: []string{"--credential-env"}, value: true, set: func(a *Args, v string) error { a.CredentialEnv = v; return nil }},
	{names: []string{"--insecure-tls"}, value: true, set: func(a *Args, v string) error { a.InsecureTLS = append(a.InsecureTLS, v); return nil }},
	{names: []string{"--allow-tier-b"}, value: true, set: func(a *Args, v string) error { a.AllowTierB = append(a.AllowTierB, v); return nil }},
	{names: []string{"--allow-tier-c"}, value: true, set: func(a *Args, v string) error { a.AllowTierC = append(a.AllowTierC, v); return nil }},
	{names: []string{"--state-dir"}, value: true, set: func(a *Args, v string) error { a.StateDir = v; return nil }},
	{names: []string{"--report"}, value: true, set: func(a *Args, v string) error { a.Report = v; return nil }},
	{names: []string{"--settle"}, value: true, set: floatSetter(func(a *Args) *float64 { return &a.Settle })},
	{names: []string{"--call-timeout"}, value: true, set: floatSetter(func(a *Args) *float64 { return &a.CallTimeout })},
	{names: []string{"--allow-http-for-tests"}, set: func(a *Args, _ string) error { a.AllowHTTPForTests = true; return nil }},
	{names: []string{"--go-bin"}, value: true, set: func(a *Args, v string) error { a.GoBin = v; return nil }},
}

// negativeNumber is argparse's _negative_number_matcher: such an argument is a value.
var negativeNumber = regexp.MustCompile(`^-\.?\d`)

// classified is one argument: a value ('A'), the "--" marker ('-'), or an option ('O'),
// known (opt set, explicit holding a "=value" or a joined short-flag rest) or not.
type classified struct {
	kind     byte
	opt      *option
	explicit *string
	shortRun bool // "-h<rest>": a joined short flag, not "-h=<rest>"
}

func lookup(name string) *option {
	for i := range options {
		for _, n := range options[i].names {
			if n == name {
				return &options[i]
			}
		}
	}
	return nil
}

// classify is argparse's _parse_optional; an ambiguous prefix is an error.
func classify(arg string) (classified, error) {
	switch {
	case !strings.HasPrefix(arg, "-"):
		return classified{kind: 'A'}, nil
	case lookup(arg) != nil:
		return classified{kind: 'O', opt: lookup(arg)}, nil
	case len(arg) == 1:
		return classified{kind: 'A'}, nil
	}
	if name, rest, found := strings.Cut(arg, "="); found && lookup(name) != nil {
		return classified{kind: 'O', opt: lookup(name), explicit: &rest}, nil
	}
	matches, err := prefixMatches(arg)
	if err != nil || len(matches) == 1 {
		if len(matches) == 1 {
			return matches[0], nil
		}
		return classified{}, err
	}
	if negativeNumber.MatchString(arg) || strings.Contains(arg, " ") {
		return classified{kind: 'A'}, nil
	}
	return classified{kind: 'O'}, nil
}

// prefixMatches is argparse's _get_option_tuples: long options by unique prefix, and "-h"
// joined to a rest.
func prefixMatches(arg string) ([]classified, error) {
	var out []classified
	var names []string
	if strings.HasPrefix(arg, "--") {
		prefix, rest, found := strings.Cut(arg, "=")
		for i := range options {
			for _, n := range options[i].names {
				if strings.HasPrefix(n, "--") && strings.HasPrefix(n, prefix) {
					c := classified{kind: 'O', opt: &options[i]}
					if found {
						c.explicit = &rest
					}
					out, names = append(out, c), append(names, n)
				}
			}
		}
	} else if rest := arg[2:]; arg[:2] == "-h" {
		out = append(out, classified{kind: 'O', opt: lookup("-h"), explicit: &rest, shortRun: true})
	}
	if len(out) > 1 {
		return nil, &UsageError{fmt.Sprintf("ambiguous option: %s could match %s", arg, strings.Join(names, ", "))}
	}
	return out, nil
}

// ParseArgs is the command line as argparse parses it: errHelp for -h/--help, a *UsageError
// for anything it refuses.
func ParseArgs(argv []string) (*Args, error) {
	a := &Args{CredentialEnv: "MCP_N4", Settle: 5.0, CallTimeout: 120.0}
	kinds := make([]classified, len(argv))
	dashed := false
	for i, arg := range argv {
		switch {
		case dashed:
			kinds[i] = classified{kind: 'A'}
		case arg == "--":
			kinds[i], dashed = classified{kind: '-'}, true
		default:
			c, err := classify(arg)
			if err != nil {
				return nil, err
			}
			kinds[i] = c
		}
	}
	var extras []string
	for i := 0; i < len(argv); i++ {
		c := kinds[i]
		if c.opt == nil {
			extras = append(extras, argv[i])
			continue
		}
		if c.opt.set == nil { // help
			if c.explicit != nil && !c.shortRun {
				return nil, &UsageError{fmt.Sprintf("argument %s: ignored explicit argument %s", c.opt.display(), pyjson.StrRepr(*c.explicit))}
			}
			return nil, errHelp
		}
		value := ""
		switch {
		case !c.opt.value && c.explicit != nil:
			return nil, &UsageError{fmt.Sprintf("argument %s: ignored explicit argument %s", c.opt.display(), pyjson.StrRepr(*c.explicit))}
		case c.opt.value && c.explicit != nil:
			value = *c.explicit
		case c.opt.value:
			if i+1 >= len(argv) || kinds[i+1].kind != 'A' {
				return nil, &UsageError{fmt.Sprintf("argument %s: expected one argument", c.opt.display())}
			}
			i++
			value = argv[i]
		}
		if err := c.opt.set(a, value); err != nil {
			return nil, &UsageError{fmt.Sprintf("argument %s: %s", c.opt.display(), err)}
		}
	}
	if len(extras) > 0 {
		return nil, &UsageError{"unrecognized arguments: " + strings.Join(extras, " ")}
	}
	return a, nil
}
