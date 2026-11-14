package livesmoke

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// The command line is read as argparse read it: each row is one refusal or one accepted spelling.
func TestParseArgsRefusalsAndSpellings(t *testing.T) {
	refused := map[string][]string{
		"unrecognized arguments: extra":                                                                {"--apply", "extra"},
		"unrecognized arguments: --nope":                                                               {"--nope"},
		"unrecognized arguments: -x":                                                                   {"-x"},
		"argument --station: expected one argument":                                                    {"--station"},
		"argument --station: expected one argument (an option follows)":                                {"--station", "--apply"},
		"argument --apply: ignored explicit argument 'yes'":                                            {"--apply=yes"},
		"argument -h/--help: ignored explicit argument 'x'":                                            {"--help=x"},
		"argument --settle: invalid float value: 'soon'":                                               {"--settle", "soon"},
		"ambiguous option: --allow could match --allow-tier-b, --allow-tier-c, --allow-http-for-tests": {"--allow", "Lab"},
		"ambiguous option: --s=1 could match --station, --station-home, --state-dir, --settle":         {"--s=1"},
	}
	for want, argv := range refused {
		_, err := ParseArgs(argv)
		var usage *UsageError
		if !errors.As(err, &usage) || err == errHelp {
			t.Errorf("%v: err = %v, want a usage error", argv, err)
			continue
		}
		if msg := usage.Msg; msg != want && msg+" (an option follows)" != want {
			t.Errorf("%v:\n got  %s\n want %s", argv, msg, want)
		}
	}
	for _, argv := range [][]string{{"-h"}, {"--help"}, {"--he"}, {"-hx"}, {"--apply", "-h", "--nope"}} {
		if _, err := ParseArgs(argv); err != errHelp {
			t.Errorf("%v: err = %v, want the help", argv, err)
		}
	}
	a, err := ParseArgs([]string{"--app", "--station=Lab=https://h", "--sett", "-1.5", "--call-timeout=7", "--rep", "r.json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Apply || len(a.Stations) != 1 || a.Stations[0] != "Lab=https://h" || a.Settle != -1.5 || a.CallTimeout != 7 || a.Report != "r.json" {
		t.Errorf("parsed = %+v", a)
	}
	if _, err := ParseArgs([]string{"--", "--apply"}); err == nil || err.Error() != "unrecognized arguments: -- --apply" {
		t.Errorf("after --: err = %v", err)
	}
}

// A server named without a directory is the file in the working directory, as the check that
// found it says, never a command of the same name on PATH: the path handed to exec is absolute.
func TestResolveServerMakesARelativeGoBinAbsolute(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mcp-n4"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	a := &Args{GoBin: "mcp-n4"}
	if problem := ResolveServer(a, func(string) string { return "" }); problem != "" {
		t.Fatal(problem)
	}
	want, _ := filepath.Abs("mcp-n4")
	if a.GoBin != want {
		t.Errorf("GoBin = %s, want %s", a.GoBin, want)
	}
	missing := &Args{GoBin: "absent"}
	if problem := ResolveServer(missing, func(string) string { return "" }); problem != "no executable Go server at absent: build it (cd go && go build -o bin/ ./cmd/mcp-n4) or pass --go-bin" {
		t.Errorf("problem = %s", problem)
	}
}
