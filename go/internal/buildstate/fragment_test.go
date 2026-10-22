package buildstate

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

const okFragment = "<!-- ledger: kit -->\n<!-- date: 2026-10-23 -->\n<!-- last-change-note: T1 thing -->\n\n" +
	"log: 2026-10-23 · T1 → did a thing\nopen_issue: OWED T1 half\nopen_issue: OWED T1 other half\n"

func TestParseFragment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, file, text string
		want             Fragment
		reason           string
	}{
		{"valid", "2026-10-23-t1.md", okFragment, Fragment{Name: "2026-10-23-t1.md", Date: "2026-10-23", Note: "T1 thing",
			Logs: []string{"2026-10-23 · T1 → did a thing"}, Issues: []string{"OWED T1 half", "OWED T1 other half"}}, ""},
		{"note and issues are optional", "2026-10-23-t1.md", "<!-- ledger: kit -->\n<!-- date: 2026-10-23 -->\nlog: 2026-10-23 · x\n",
			Fragment{Name: "2026-10-23-t1.md", Date: "2026-10-23", Logs: []string{"2026-10-23 · x"}}, ""},
		{"archive keeps many log lines of any date, in file order", "0000-archive-pre-2026-10-23.md",
			"<!-- ledger: kit -->\n<!-- date: 2026-10-22 -->\nlog: 2026-10-23 · b\nlog: 2026-09-01 · a\nopen_issue: X\n",
			Fragment{Name: "0000-archive-pre-2026-10-23.md", Date: "2026-10-22", Archive: true,
				Logs: []string{"2026-10-23 · b", "2026-09-01 · a"}, Issues: []string{"X"}}, ""},
		{"no ledger header", "2026-10-23-t1.md", "<!-- date: 2026-10-23 -->\nlog: 2026-10-23 · x\n", Fragment{}, errLedger},
		{"ledger for another module", "2026-10-23-t1.md", strings.Replace(okFragment, "ledger: kit", "ledger: CompPan", 1), Fragment{}, errLedger},
		{"no date header", "2026-10-23-t1.md", "<!-- ledger: kit -->\nlog: 2026-10-23 · x\n", Fragment{}, errDate},
		{"malformed date header", "2026-10-23-t1.md", strings.Replace(okFragment, "date: 2026-10-23", "date: 23/10/2026", 1), Fragment{}, errDate},
		{"filename date differs from the header", "2026-10-24-t1.md", okFragment, Fragment{}, errName},
		{"filename without a date", "t1.md", okFragment, Fragment{}, errName},
		{"filename with spaces", "2026-10-23 t1.md", okFragment, Fragment{}, errName},
		{"no log line", "2026-10-23-t1.md", "<!-- ledger: kit -->\n<!-- date: 2026-10-23 -->\nopen_issue: X\n", Fragment{}, errLogCount},
		{"two log lines outside the archive", "2026-10-23-t1.md", okFragment + "log: 2026-10-23 · y\n", Fragment{}, errLogCount},
		{"log line date differs from the header", "2026-10-23-t1.md", strings.Replace(okFragment, "log: 2026-10-23", "log: 2026-10-22", 1), Fragment{}, "line 5: " + errLogDate},
		{"log line without the date prefix", "2026-10-23-t1.md", strings.Replace(okFragment, "log: 2026-10-23 · ", "log: ", 1), Fragment{}, "line 5: " + errLogShape},
		{"empty open_issue", "2026-10-23-t1.md", okFragment + "open_issue: \n", Fragment{}, "line 8: " + errIssue},
		{"unknown line", "2026-10-23-t1.md", okFragment + "  - a hand-written item\n", Fragment{}, "line 8: " + errLine},
		{"duplicated header", "2026-10-23-t1.md", okFragment + "<!-- last-change-note: again -->\n", Fragment{}, "line 8: " + errDuplicate},
		{"CRLF line", "2026-10-23-t1.md", strings.Replace(okFragment, "open_issue: OWED T1 half\n", "open_issue: OWED T1 half\r\n", 1), Fragment{}, "line 6: " + errLine},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, reason, err := ParseFragment(c.file, strings.NewReader(c.text))
			if err != nil {
				t.Fatalf("unexpected read error %v", err)
			}
			if reason != c.reason {
				t.Fatalf("reason = %q, want %q", reason, c.reason)
			}
			if c.reason == "" && !reflect.DeepEqual(got, c.want) {
				t.Errorf("fragment = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestParseFragmentReadErrorIsNotAViolation(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	_, reason, err := ParseFragment("2026-10-23-t1.md", iotest.ErrReader(boom))
	if !errors.Is(err, boom) || reason != "" {
		t.Errorf("got (%q, %v), want the read error", reason, err)
	}
}

func TestScanSortsByNameAndRejectsStrayFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("2026-10-23-t1.md", okFragment)
	write("0000-archive-pre-2026-10-23.md", "<!-- ledger: kit -->\n<!-- date: 2026-10-22 -->\nlog: 2026-09-01 · a\n")
	write(".hidden.md", "ignored")
	write("notes.txt", "stray")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	frags, problems, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(frags) != 2 || frags[0].Name != "0000-archive-pre-2026-10-23.md" || frags[1].Name != "2026-10-23-t1.md" {
		t.Errorf("fragments = %+v", frags)
	}
	want := []Problem{{Name: "notes.txt", Reason: errStray}, {Name: "sub", Reason: errStray}}
	if !reflect.DeepEqual(problems, want) {
		t.Errorf("problems = %+v, want %+v", problems, want)
	}
}

func TestScanUnreadableFragmentIsAnIOProblem(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "2026-10-23-t1.md")
	if err := os.WriteFile(p, []byte(okFragment), 0o000); err != nil {
		t.Fatal(err)
	}
	_, problems, err := Scan(dir)
	if err != nil || len(problems) != 1 || !problems[0].IO {
		t.Errorf("got (%+v, %v), want one IO problem", problems, err)
	}
}
