package lint

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParseValuedNamed: Mode Valued under ValueOptions.Named is lint-write-path's argument loop, read left
// to right: --strict and the value options anywhere (every value kept, the rule reads the last), the
// first bad argument reported with its named text, an empty argument leaving the root unset.
func TestParseValuedNamed(t *testing.T) {
	const name, usage = "lint-w", "usage: lint-w.sh <module-root> [--bog <f>] [--strict]"
	vo := ValueOptions{Repeat: []string{"--bog"}, Strict: true, Named: true}
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args   []string
		strict bool
		values map[string][]string
		errMsg string
	}{
		{[]string{dir}, false, nil, ""},
		{[]string{"--strict", "--bog", "b", dir, "--strict", "--bog", ""}, true, map[string][]string{"--bog": {"b", ""}}, ""},
		{[]string{"", dir}, false, nil, ""},
		{[]string{"", "", dir}, false, nil, ""}, // each empty argument leaves the root unset in turn
		{nil, false, nil, usage + "\n"},
		{[]string{"--strict"}, false, nil, usage + "\n"},
		{[]string{""}, false, nil, usage + "\n"},
		{[]string{dir, "--bog"}, false, nil, name + ": --bog requires a file argument\n"},
		{[]string{"--x", "--bog"}, false, nil, name + ": unknown option: --x\n"},
		{[]string{"--"}, false, nil, name + ": unknown option: --\n"},
		{[]string{"-h", dir}, false, nil, name + ": unknown option: -h\n"},
		{[]string{dir, dir}, false, nil, name + ": unexpected argument: " + dir + "\n"},
		{[]string{dir, dir, "--x"}, false, nil, name + ": unexpected argument: " + dir + "\n"},
		{[]string{file}, false, nil, name + ": not a directory: " + file + "\n"},
	} {
		got, err := ParseValued(Valued, name, usage, vo, tc.args)
		if tc.errMsg != "" {
			var ue *UsageError
			if err == nil || err.Error() != tc.errMsg || errors.As(err, &ue) == (tc.args != nil && tc.args[0] == file) {
				t.Errorf("ParseValued(%q) error = %v, want %q", tc.args, err, tc.errMsg)
			}
			continue
		}
		if err != nil || got.Strict != tc.strict || !reflect.DeepEqual(got.Roots, []string{dir}) || !reflect.DeepEqual(got.Values, tc.values) {
			t.Errorf("ParseValued(%q) = %+v, %v; want strict=%v values=%v", tc.args, got, err, tc.strict, tc.values)
		}
	}
	// without Named the same loop keeps the usage line alone and an empty argument is a root
	if _, err := ParseValued(Valued, name, usage, ValueOptions{Repeat: []string{"--bog"}}, []string{"", dir}); err == nil || err.Error() != usage+"\n" {
		t.Errorf("unnamed: error = %v, want the usage line", err)
	}
}

// TestParseValuedFlags: a Mode Valued boolean flag (ValueOptions.Flags) may appear anywhere, with the value
// options, --strict and the root, and repeats harmlessly; in a value position it is that option's value; a
// flag given a value (--no-x=1), an unknown flag and "--" stay the usage error they are today.
func TestParseValuedFlags(t *testing.T) {
	const name, usage = "lint-f", "usage: lint-f.sh [--strict] [--o <v>] [--r <v>]... [--no-x] <root>"
	vo := ValueOptions{Option: "--o", Repeat: []string{"--r"}, Strict: true, Flags: []string{"--no-x"}}
	dir := t.TempDir()
	on := map[string]bool{"--no-x": true}
	for _, tc := range []struct {
		args   []string
		flags  map[string]bool
		values map[string][]string
		option string
		strict bool
		errMsg string
	}{
		{args: []string{dir}},
		{args: []string{"--no-x", dir}, flags: on},
		{args: []string{dir, "--no-x"}, flags: on},
		{args: []string{"--no-x", "--no-x", dir}, flags: on},
		{args: []string{"--r", "a", "--no-x", "--o", "v", dir, "--strict"}, flags: on, values: map[string][]string{"--r": {"a"}}, option: "v", strict: true},
		{args: []string{"--r", "--no-x", dir}, values: map[string][]string{"--r": {"--no-x"}}},
		{args: []string{"--o", "--no-x", dir}, option: "--no-x"},
		{args: []string{"--no-x=1", dir}, errMsg: usage + "\n"},
		{args: []string{"--no-y", dir}, errMsg: usage + "\n"},
		{args: []string{"--", "--no-x", dir}, errMsg: usage + "\n"},
		{args: []string{"--no-x"}, errMsg: usage + "\n"},
	} {
		got, err := ParseValued(Valued, name, usage, vo, tc.args)
		if tc.errMsg != "" {
			var ue *UsageError
			if err == nil || err.Error() != tc.errMsg || !errors.As(err, &ue) {
				t.Errorf("ParseValued(%q) error = %v, want usage error %q", tc.args, err, tc.errMsg)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got.Flags, tc.flags) || !reflect.DeepEqual(got.Values, tc.values) ||
			got.Option != tc.option || got.Strict != tc.strict || !reflect.DeepEqual(got.Roots, []string{dir}) {
			t.Errorf("ParseValued(%q) = %+v, %v; want flags=%v values=%v option=%q strict=%v", tc.args, got, err, tc.flags, tc.values, tc.option, tc.strict)
		}
	}
	// under Named a flag given a value is named like any unknown option
	named := vo
	named.Named = true
	if _, err := ParseValued(Valued, name, usage, named, []string{"--no-x=1", dir}); err == nil || err.Error() != name+": unknown option: --no-x=1\n" {
		t.Errorf("named: error = %v, want the unknown-option line", err)
	}
}

func TestRowStatus(t *testing.T) {
	f := Finding{Rule: "lint-w", Severity: Warn, Status: "STALE", Path: "docs/m.md", Line: 4, Msg: "slot x: gone"}
	if got, want := f.Row(), "STALE  lint-w  docs/m.md:4  slot x: gone"; got != want {
		t.Errorf("Row() = %q, want %q", got, want)
	}
}

// configuredRoot is a RootRule and a Configurer: it reports the last --bog value it was configured with as a
// STALE row (a Warn), then fails with a RowError when that value is "broken".
type configuredRoot struct{ bog string }

func (*configuredRoot) ID() string         { return "lint-w" }
func (*configuredRoot) Severity() Severity { return Fail }
func (r *configuredRoot) Configure(a Args) (Configured, error) {
	c := &configuredRoot{}
	if vs := a.Values["--bog"]; len(vs) > 0 {
		c.bog = vs[len(vs)-1]
	}
	return c, nil
}
func (r *configuredRoot) CheckRoot(root string) ([]Finding, error) {
	out := []Finding{{Path: root, Msg: "bog=" + r.bog, Severity: Warn, Status: "STALE"}}
	if r.bog == "broken" {
		return out, &RowError{Row: "lint-w  ERROR  " + root + "  broken bog"}
	}
	return out, nil
}

// TestCommandValuedRootRule: a RootRule run alone in Mode Valued gets its option values through Configure; a
// Status row counts as its severity (exit 1 only under --strict); a RowError prints on stdout after the rows,
// exit 3. In a rule list no option is parsed.
func TestCommandValuedRootRule(t *testing.T) {
	dir := t.TempDir()
	sp := Spec{Root: &configuredRoot{}, Mode: Valued, Usage: "usage: lint-w.sh <root>", Options: []string{"--bog"},
		ValuedStrict: true, ValuedNamed: true}
	main := Command(sp, rootSpec(rootFake{seen: new([]string)}))
	for _, tc := range []struct {
		args     []string
		code     int
		out, err string
	}{
		{[]string{"lint-w", dir, "--bog", "a", "--bog", "b"}, 0, "STALE  lint-w  " + dir + ": bog=b\n", ""},
		{[]string{"lint-w", "--strict", dir}, 1, "STALE  lint-w  " + dir + ": bog=\n", ""},
		{[]string{"lint-w", dir, "--bog", "broken"}, 3, "STALE  lint-w  " + dir + ": bog=broken\nlint-w  ERROR  " + dir + "  broken bog\n", ""},
		{[]string{"lint-w", dir, "--nope"}, 3, "", "lint-w: unknown option: --nope\n"},
		{[]string{"lint-w,lint-r", dir}, 1, "STALE  lint-w  " + dir + ": bog=\nFAIL  lint-r  " + dir + ": root seen\n", ""},
		{[]string{"lint-w,lint-r", dir, "--bog", "b"}, 3, "", "n4lint: not a directory: --bog\n"},
	} {
		s, o, e := stdio()
		if code := main(tc.args, s); code != tc.code || o.String() != tc.out || e.String() != tc.err {
			t.Errorf("%q: exit %d out %q err %q, want %d, %q and %q", tc.args, code, o, e, tc.code, tc.out, tc.err)
		}
	}
}

// TestConfigureNeverMutatesTheRegisteredRule: Configure returns a configured copy, so in one process a
// configured run leaves no option value behind: the next run of the same Command, alone without the option
// or in a rule list (where Configure is not called), sees none.
func TestConfigureNeverMutatesTheRegisteredRule(t *testing.T) {
	dir := t.TempDir()
	sp := Spec{Root: &configuredRoot{}, Mode: Valued, Usage: "usage: lint-w.sh <root>", Options: []string{"--bog"},
		ValuedStrict: true, ValuedNamed: true}
	main := Command(sp, rootSpec(rootFake{seen: new([]string)}))
	for _, tc := range []struct {
		args []string
		out  string
	}{
		{[]string{"lint-w", dir, "--bog", "x"}, "STALE  lint-w  " + dir + ": bog=x\n"},
		{[]string{"lint-w,lint-r", dir}, "STALE  lint-w  " + dir + ": bog=\nFAIL  lint-r  " + dir + ": root seen\n"},
		{[]string{"lint-w", dir}, "STALE  lint-w  " + dir + ": bog=\n"},
	} {
		s, o, _ := stdio()
		if main(tc.args, s); o.String() != tc.out {
			t.Errorf("%q: out %q, want %q", tc.args, o, tc.out)
		}
	}
}
