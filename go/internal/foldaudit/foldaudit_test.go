package foldaudit

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	badDate := "fold-audit: --deltas-since needs a YYYY-MM-DD date\n" + usage + "\n"
	cases := []struct {
		name string
		args []string
		want options
		err  string // stderr on a usage error; "" means the parse succeeds
	}{
		{"positional only", []string{"I", "K"}, options{index: "I", root: "K"}, ""},
		{"strict and since", []string{"--strict", "--deltas-since", "2026-09-24", "I", "K"},
			options{strict: true, since: "2026-09-24", index: "I", root: "K"}, ""},
		{"last --deltas-since wins", []string{"--deltas-since", "2026-01-01", "--deltas-since", "2026-02-02", "I", "K"},
			options{since: "2026-02-02", index: "I", root: "K"}, ""},
		{"-- ends the flags", []string{"--", "--strict", "K"}, options{index: "--strict", root: "K"}, ""},
		{"empty positional ends the flags", []string{"", "K"}, options{index: "", root: "K"}, ""},
		{"unknown flag", []string{"--bogus", "I", "K"}, options{}, "fold-audit: unknown flag: --bogus\n" + usage + "\n"},
		{"lone dash is a flag", []string{"-", "K"}, options{}, "fold-audit: unknown flag: -\n" + usage + "\n"},
		{"malformed date", []string{"--deltas-since", "2026-9-1", "I", "K"}, options{}, badDate},
		{"missing date", []string{"--deltas-since"}, options{}, badDate},
		{"date with a trailing newline", []string{"--deltas-since", "2026-09-24\n", "I", "K"}, options{}, badDate},
		{"one positional", []string{"I"}, options{}, usage + "\n"},
		{"flags after the positionals", []string{"I", "--strict", "K"}, options{}, usage + "\n"},
	}
	for _, c := range cases {
		got, errText := parseArgs(c.args)
		if errText != c.err || (c.err == "" && got != c.want) {
			t.Errorf("%s: got (%+v, %q), want (%+v, %q)", c.name, got, errText, c.want, c.err)
		}
	}
}

func TestHarvestText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text    string
		tokens, delta []string
	}{
		{"stem and Δ citation", "a [ev: retro full-name Δ2] b", []string{"full-name"}, []string{"full-name Δ2"}},
		{"deferred counts only as a Δ", "[deferred: retro parked-one Δ1]", nil, []string{"parked-one Δ1"}},
		{"token stops at a disallowed byte", "[ev: retro abc_def/x] [ev: retro a.b-c]", []string{"a.b-c", "abc_def"}, nil},
		{"Δ form needs the closing bracket", "[ev: retro open-ended Δ3 and more", []string{"open-ended"}, nil},
		{"Δ form needs digits", "[ev: retro no-digit Δ]", []string{"no-digit"}, nil},
		{"several per line", "[ev: retro one-a Δ1][ev: retro two-b Δ12]", []string{"one-a", "two-b"}, []string{"one-a Δ1", "two-b Δ12"}},
		{"first char must be alphanumeric", "[ev: retro -dash] [ev: retro .dot]", nil, nil},
		{"short tokens are harvested (the floor applies at match time)", "[ev: retro X]", []string{"X"}, nil},
		{"a file with a NUL byte is binary to grep and yields nothing", "[ev: retro binary-one Δ1]\x00", nil, nil},
		{"invalid UTF-8 next to a token does not extend it", "[ev: retro bad-utf\xff8]", []string{"bad-utf"}, nil},
	}
	for _, c := range cases {
		h := newHarvest(true)
		h.add([]byte(c.text))
		gotT := h.sortedTokens()
		gotD := keys(h.deltas)
		if !reflect.DeepEqual(gotT, c.tokens) || !reflect.DeepEqual(gotD, c.delta) {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", c.name, gotT, gotD, c.tokens, c.delta)
		}
	}
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestRowParsing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, line                   string
		dated                        bool
		file, status, deltas, stemOf string
	}{
		{"generated row", "| 2026-10-01-gap.md | kit | 2026-10-01 | folded | 3 |", true, "2026-10-01-gap.md", "folded", "3", "gap"},
		{"pending row", "| 2026-10-02-later.md | kit | 2026-10-02 | pending | 4 |", true, "2026-10-02-later.md", "pending", "", "later"},
		{"date inside the name", "| APPLY-2026-09-03-x.md | kit | x | folded | 1 |", true, "APPLY-2026-09-03-x.md", "folded", "1", "APPLY-2026-09-03-x"},
		{"INDEX.md is skipped as the file", "| see INDEX.md | 2026-01-01-a.md | folded | 1 |", true, "2026-01-01-a.md", "folded", "1", "a"},
		{"first non-INDEX .md token is the file", "| INDEX.md 2026-01-01-b.md | other.md | folded |", true, "2026-01-01-b.md", "folded", "", "b"},
		{"longest token (dots allowed)", "| 2026-01-01-v1.2.md.md | folded | 1 |", true, "2026-01-01-v1.2.md.md", "folded", "1", "v1.2.md"},
		{"date and .md split by a pipe", "| 2026-01-01 | x.md | folded | 1 |", false, "", "", "", ""},
		{"first status cell decides", "| 2026-01-01-c.md | pending | folded | 1 |", true, "2026-01-01-c.md", "pending", "", "c"},
		{"deltas cell keeps tabs, drops spaces", "| 2026-01-01-d.md | folded | 1 2\t|", true, "2026-01-01-d.md", "folded", "12\t", "d"},
		{"no cell after folded", "| 2026-01-01-e.md | folded", true, "2026-01-01-e.md", "folded", "", "e"},
		{"status trims spaces, not tabs", "| 2026-01-01-f.md |\tfolded | 1 |", true, "2026-01-01-f.md", "", "", "f"},
		{"invalid UTF-8 breaks [^|]*", "| 2026-01-01-\xffg.md | folded | 1 |", false, "", "", "", ""},
	}
	for _, c := range cases {
		if got := datedRow(c.line); got != c.dated {
			t.Errorf("%s: datedRow = %v, want %v", c.name, got, c.dated)
			continue
		}
		if !c.dated {
			continue
		}
		if got := rowFile(c.line); got != c.file {
			t.Errorf("%s: rowFile = %q, want %q", c.name, got, c.file)
		}
		if got := rowStatus(c.line); got != c.status {
			t.Errorf("%s: rowStatus = %q, want %q", c.name, got, c.status)
		}
		if c.status == "folded" {
			if got := deltasCell(c.line); got != c.deltas {
				t.Errorf("%s: deltasCell = %q, want %q", c.name, got, c.deltas)
			}
		}
		if got := stem(c.file); got != c.stemOf {
			t.Errorf("%s: stem = %q, want %q", c.name, got, c.stemOf)
		}
	}
}

func TestBashInt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"0", 0, true}, {"3", 3, true}, {"10", 10, true},
		{"010", 8, true}, {"00", 0, true}, // a leading zero is octal in bash arithmetic
		{"09", 0, false}, {"0189", 0, false},
		{"18446744073709551617", 1, true}, // decimal wraps modulo 2^64
		{"0777777777777777777777777", -1, true},
	}
	for _, c := range cases {
		got, ok := bashInt(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("bashInt(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestCorpusWalk(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("core.md", "[ev: retro in-core]")
	write("deep/nested/x.md", "[ev: retro in-deep]")
	write("ledger.d/2026-10-22-x.md", "[ev: retro in-ledger]")
	write("retros/a.md", "[ev: retro in-retros]")
	write("deep/retros/b.md", "[ev: retro in-nested-retros]")
	write("INDEX.md", "[ev: retro in-index]")
	write("notes.txt", "[ev: retro in-txt]")
	write("dir.md/inner.md", "[ev: retro in-dir-md]")
	if err := os.Symlink(filepath.Join(root, "notes.txt"), filepath.Join(root, "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "deep"), filepath.Join(root, "linkdir")); err != nil {
		t.Fatal(err)
	}
	want := []string{"in-core", "in-deep", "in-dir-md", "in-ledger", "in-txt"}
	for _, r := range []string{root, root + "/"} {
		if got := harvestCorpus(r, false).sortedTokens(); !reflect.DeepEqual(got, want) {
			t.Errorf("harvestCorpus(%q) = %q, want %q", r, got, want)
		}
	}
	// find -P does not follow a symlinked starting point (no trailing slash).
	link := filepath.Join(t.TempDir(), "kit")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if got := harvestCorpus(link, false).sortedTokens(); got != nil {
		t.Errorf("symlinked root: got %q, want nothing", got)
	}
	if got := harvestCorpus(link+"/", false).sortedTokens(); !reflect.DeepEqual(got, want) {
		t.Errorf("symlinked root with a slash: got %q, want %q", got, want)
	}
}
