package reportmodule

import (
	"bytes"
	"reflect"
	"testing"
)

func TestSplitRuns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a b", []string{"a b"}},
		{"a  b", []string{"a", "b"}},
		{"a \tb\t\tc", []string{"a", "b", "c"}},
		{"abc  ", []string{"abc", ""}},
		{"  abc", []string{"", "abc"}},
		{"   ", []string{"", ""}},
		{"a\vb  c", []string{"a\vb", "c"}},
	}
	for _, tt := range tests {
		if got := splitRuns(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitRuns(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseRows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		out  string
		f    filter
		want []row
	}{
		{"site", "FAIL  delay  /a/B.java:12  no floor\nnoise\nWARN  delay  C.java  no line",
			filter{sel: "FAIL|WARN"}, []row{{"FAIL", "delay", "B.java:12  no floor"}, {"WARN", "delay", "C.java:  no line"}}},
		{"site keeps a second colon in the line", "WARN  x  /d/E.java:9:3  r", filter{sel: "WARN"}, []row{{"WARN", "x", "E.java:9:3  r"}}},
		{"reason fields joined by two spaces", "WARN  x  /f.java:1  a  b", filter{sel: "WARN"}, []row{{"WARN", "x", "f.java:1  a  b"}}},
		{"a pipe in the check shifts the record split", "FAIL  de|lay  /j/K.java:7  r", filter{sel: "FAIL"},
			[]row{{"FAIL", "de", "lay|K.java:7  r"}}},
		{"base", "WARN  u  /r/x.jar:3  colon kept", filter{sel: "WARN", mode: base}, []row{{"WARN", "u", "x.jar:3  colon kept"}}},
		{"nosite", "WARN  j  /m/A-ux  no specs  add", filter{sel: "WARN", mode: nosite}, []row{{"WARN", "j", "no specs  add"}}},
		{"ws", "FAIL\ttypecount\t/p/x.jar\t3   declared   2 found\nSKIP  facets", filter{sel: "PASS|FAIL|SKIP|WARN", mode: ws},
			[]row{{"FAIL", "typecount", "3 declared 2 found"}, {"SKIP", "facets", ""}}},
		{"swap keeps the exact status", "t  WARN  /w/P.java:7  d\nt  FAIL  /w/B.java:1  x\nx  WARN|y  /w/C.java:2  z\n\n   ",
			filter{mode: swap, keep: "WARN"}, []row{{"WARN", "t", "P.java:7  d"}, {"WARN", "y", "x|C.java:2  z"}}},
		{"as-WARN relay", "STALE  wp  /m.md:4  gone\nFAIL  wp  /B.java:1  x\nWARN  wp  /x:1  no", filter{sel: "FAIL", asw: "STALE|DRIFT"},
			[]row{{"WARN", "wp", "m.md:4  gone"}, {"FAIL", "wp", "B.java:1  x"}}},
		{"empty sel skips only empty lines", "a  b\n\n", filter{mode: base}, []row{{"a", "b", "  "}}},
	}
	for _, tt := range tests {
		if got := parseRows(tt.out, tt.f); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: parseRows = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPassAfterRelay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rl   relay
		want string
	}{
		{relay{}, "A  PASS  c  clean\n"},
		{relay{warn: 1}, "A  PASS  c  no failing row (1 WARN row above)\n"},
		{relay{warn: 2, adv: 1}, "A  PASS  c  no failing row (2 WARN, 1 ADVISORY rows above)\n"},
		{relay{adv: 1}, "A  PASS  c  no failing row (0 WARN, 1 ADVISORY row above)\n"},
		{relay{fail: true, warn: 3}, ""},
	}
	for _, tt := range tests {
		var b bytes.Buffer
		r := &report{out: &b, n: map[string]int{}}
		if r.passAfterRelay("A", "c", tt.rl); b.String() != tt.want {
			t.Errorf("passAfterRelay(%+v) = %q, want %q", tt.rl, b.String(), tt.want)
		}
	}
}

func TestShellCuts(t *testing.T) {
	t.Parallel()
	if got := cutBefore("FAIL  x  y", "  "); got != "FAIL" {
		t.Errorf("cutBefore = %q", got)
	}
	if got := cutAfter("FAIL  x  y", "  "); got != "x  y" {
		t.Errorf("cutAfter = %q", got)
	}
	if cutBefore("FAIL", "  ") != "FAIL" || cutAfter("FAIL", "  ") != "FAIL" {
		t.Error("a missing separator must leave the text whole, as ${s%%  *} and ${s#*  } do")
	}
}

func TestSplitStatusCheck(t *testing.T) {
	t.Parallel()
	for in, want := range map[string][3]string{
		"FAIL  chk  /p/F.java:3  msg  more": {"FAIL", "chk", "/p/F.java:3  msg  more"},
		"FAIL  x":                           {"FAIL", "x", "x"},
		"FAIL":                              {"FAIL", "FAIL", "FAIL"},
	} {
		if st, c, r := splitStatusCheck(in); [3]string{st, c, r} != want {
			t.Errorf("splitStatusCheck(%q) = %q %q %q, want %q", in, st, c, r, want)
		}
	}
}

func TestTrimOutput(t *testing.T) {
	t.Parallel()
	if got := trimOutput([]byte("a\x00b\n\x00\n\n")); got != "ab" {
		t.Errorf("trimOutput = %q, want \"ab\"", got)
	}
}
