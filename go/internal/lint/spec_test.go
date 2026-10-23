package lint

import "testing"

// The Spec features a port opts into by field: Severity Advisory, Spec.EmptyError and Spec.BareUsage.
// Expected bytes were recorded from toolbelt/lint-silent-protection.sh.

const k20 = "no Java sources found (K20: wrong path or empty scaffold)"

func TestRunAdvisoryNeverExits(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	main := Command(javaSpec(fakeRule{"lint-a", "MARK", Advisory}, First))
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"alone", []string{"lint-a", root}},
		{"alone --strict", []string{"lint-a", "--strict", root}},
		{"--all --strict", []string{"--all", "--strict", root}},
	} {
		s, o, e := stdio()
		want := "ADVISORY  lint-a  " + root + "/A.java:1  saw: MARK\n"
		if code := main(tc.args, s); code != 0 || o.String() != want || e.Len() != 0 {
			t.Errorf("%s: exit %d out %q err %q, want 0 and %q", tc.name, code, o, e, want)
		}
	}
}

func emptySpec(id string, mode Mode) Spec {
	sp := javaSpec(fakeRule{id, "MARK", Warn}, mode)
	sp.EmptyError = EmptyRow{Msg: k20}
	return sp
}

func TestRunEmptyError(t *testing.T) {
	full := tree(t, map[string]string{"A.java": "MARK\n"})
	empty := tree(t, map[string]string{"sub/a.txt": "MARK\n"}) // a file, but none this rule selects
	later := tree(t, map[string]string{"C.java": "MARK\n"})
	other := Spec{Rule: fakeRule{"lint-o", "MARK", Warn}, Mode: Each, Usage: "u", Select: Selection{Names: []string{"*.txt"}}}
	named := emptySpec("lint-n", One)
	named.EmptyError.Check = "n-shape"
	for _, tc := range []struct {
		name  string
		specs []Spec
		roots []string
		want  string
	}{
		{"single empty root", []Spec{emptySpec("lint-e", One)}, []string{empty},
			"ERROR  lint-e  " + empty + "  " + k20 + "\n"},
		{"check column", []Spec{named}, []string{empty}, "ERROR  n-shape  " + empty + "  " + k20 + "\n"},
		// The bash lint exits on its first empty root: rows before it stay, no later root is scanned.
		{"empty among non-empty roots", []Spec{emptySpec("lint-e", Each)}, []string{full, empty, later},
			"WARN  lint-e  " + full + "/A.java:1  saw: MARK\nERROR  lint-e  " + empty + "  " + k20 + "\n"},
		// The decision is per spec: a rule without EmptyError, or that selects files there, goes on.
		{"other rules continue", []Spec{emptySpec("lint-e", Each), other}, []string{empty, later},
			"ERROR  lint-e  " + empty + "  " + k20 + "\nWARN  lint-o  " + empty + "/sub/a.txt:1  saw: MARK\n"},
	} {
		s, o, e := stdio()
		if code := Run(tc.specs, "n4lint", Args{Strict: true, Roots: tc.roots}, s); code != 3 || o.String() != tc.want || e.Len() != 0 {
			t.Errorf("%s: exit %d out %q err %q, want 3 and %q", tc.name, code, o, e, tc.want)
		}
	}
	s, o, _ := stdio()
	if code := Run([]Spec{emptySpec("lint-e", One)}, "lint-e", Args{Roots: []string{full}}, s); code != 0 || o.Len() == 0 {
		t.Errorf("non-empty root: exit %d out %q, want 0 and its rows", code, o)
	}
}

// TestRunEmptyErrorWalkErrorWins: scan_files_or_exit exits before the K20 check, so a root whose
// walk failed prints the walk error only, even when nothing was listed.
func TestRunEmptyErrorWalkErrorWins(t *testing.T) {
	skipIfRoot(t)
	root, locked := lockedTree(t, map[string]string{"locked/A.java": ""}, "locked")
	for _, cont := range []bool{false, true} {
		sp := emptySpec("lint-e", One)
		sp.ContinueOnWalkError = cont
		s, o, e := stdio()
		code := Run([]Spec{sp}, "lint-e", Args{Roots: []string{root}}, s)
		if want := "lint-e: cannot list every file under " + root + ": find: '" + locked + "': Permission denied\n"; code != 3 || o.Len() != 0 || e.String() != want {
			t.Errorf("continue=%v: exit %d out %q err %q, want 3, no row and %q", cont, code, o, e, want)
		}
	}
}

func TestCommandBareUsage(t *testing.T) {
	bare := javaSpec(fakeRule{"lint-b", "MARK", Warn}, First)
	bare.Usage, bare.BareUsage = "usage: lint-b.sh [--strict] <java-src-dir>", true
	main := Command(bare, javaSpec(fakeRule{"lint-f", "MARK", Warn}, First))
	for _, tc := range []struct {
		args       []string
		code       int
		wantStderr string
	}{
		{[]string{"lint-b", "-x", "d"}, 3, bare.Usage + "\n"},
		{[]string{"lint-b", "-h"}, 3, bare.Usage + "\n"},
		{[]string{"lint-b", "--strict"}, 3, bare.Usage + "\n"},
		{[]string{"lint-f", "-x", "d"}, 3, "lint-f: unknown flag: -x\nusage: lint-f.sh <src-root>\n"}, // unchanged without BareUsage
	} {
		s, o, e := stdio()
		if code := main(tc.args, s); code != tc.code || e.String() != tc.wantStderr || o.Len() != 0 {
			t.Errorf("%q: exit %d out %q err %q, want %d and %q", tc.args, code, o, e, tc.code, tc.wantStderr)
		}
	}
}
