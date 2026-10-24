package lint

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParseOptions: Mode Options is lint-write-path's argument loop, read left to right: --strict and the
// value flags anywhere, the first bad argument reported, an empty argument leaving the root unset.
func TestParseOptions(t *testing.T) {
	const name, usage = "lint-w", "usage: lint-w.sh <module-root> [--bog <f>] [--strict]"
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args   []string
		strict bool
		values map[string]string
		errMsg string
	}{
		{[]string{dir}, false, map[string]string{}, ""},
		{[]string{"--strict", "--bog", "b", dir, "--strict", "--bog", ""}, true, map[string]string{"--bog": ""}, ""},
		{[]string{"", dir}, false, map[string]string{}, ""},
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
		got, err := Parse(Options, name, usage, tc.args, "--bog")
		if tc.errMsg != "" {
			var ue *UsageError
			if err == nil || err.Error() != tc.errMsg || errors.As(err, &ue) == (tc.args != nil && tc.args[0] == file) {
				t.Errorf("Parse(Options, %q) error = %v, want %q", tc.args, err, tc.errMsg)
			}
			continue
		}
		if err != nil || got.Strict != tc.strict || !reflect.DeepEqual(got.Roots, []string{dir}) || !reflect.DeepEqual(got.Values, tc.values) {
			t.Errorf("Parse(Options, %q) = %+v, %v; want strict=%v values=%v", tc.args, got, err, tc.strict, tc.values)
		}
	}
}

func TestRowStatus(t *testing.T) {
	f := Finding{Rule: "lint-w", Severity: Warn, Status: "STALE", Path: "docs/m.md", Line: 4, Msg: "slot x: gone"}
	if got, want := f.Row(), "STALE  lint-w  docs/m.md:4  slot x: gone"; got != want {
		t.Errorf("Row() = %q, want %q", got, want)
	}
}

// optionFake is an OptionRule: it reports the --bog value it got, a STALE row (a Warn), then fails with a
// RowError when the value is "broken".
type optionFake struct{}

func (optionFake) ID() string                          { return "lint-w" }
func (optionFake) Severity() Severity                  { return Fail }
func (optionFake) CheckRoot(string) ([]Finding, error) { return nil, errors.New("CheckRoot called") }
func (optionFake) CheckRootOptions(root string, v map[string]string) ([]Finding, error) {
	out := []Finding{{Path: root, Msg: "bog=" + v["--bog"], Severity: Warn, Status: "STALE"}}
	if v["--bog"] == "broken" {
		return out, &RowError{Row: "lint-w  ERROR  " + root + "  broken bog"}
	}
	return out, nil
}

// TestCommandOptionRule: a rule run alone parses its Spec.ValueFlags and gets their values; a Status row
// counts as its severity (exit 1 only under --strict); a RowError prints on stdout after the rows, exit 3.
// In a rule list the flag is not parsed and the rule sees no value.
func TestCommandOptionRule(t *testing.T) {
	dir := t.TempDir()
	sp := Spec{Root: optionFake{}, Mode: Options, Usage: "usage: lint-w.sh <root>", ValueFlags: []string{"--bog"}}
	main := Command(sp, rootSpec(rootFake{seen: new([]string)}))
	for _, tc := range []struct {
		args     []string
		code     int
		out, err string
	}{
		{[]string{"lint-w", dir, "--bog", "b"}, 0, "STALE  lint-w  " + dir + ": bog=b\n", ""},
		{[]string{"lint-w", "--strict", dir}, 1, "STALE  lint-w  " + dir + ": bog=\n", ""},
		{[]string{"lint-w", dir, "--bog", "broken"}, 3, "STALE  lint-w  " + dir + ": bog=broken\nlint-w  ERROR  " + dir + "  broken bog\n", ""},
		{[]string{"lint-w,lint-r", dir}, 1, "STALE  lint-w  " + dir + ": bog=\nFAIL  lint-r  " + dir + ": root seen\n", ""},
		{[]string{"lint-w,lint-r", dir, "--bog", "b"}, 3, "", "n4lint: not a directory: --bog\n"},
	} {
		s, o, e := stdio()
		if code := main(tc.args, s); code != tc.code || o.String() != tc.out || e.String() != tc.err {
			t.Errorf("%q: exit %d out %q err %q, want %d, %q and %q", tc.args, code, o, e, tc.code, tc.out, tc.err)
		}
	}
}
