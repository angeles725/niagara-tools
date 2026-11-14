package retro

import "testing"

func TestCheckStation(t *testing.T) {
	for _, ok := range []string{"LLM", "a", "PAN-1_x.y", "A" + string(make([]byte, 0))} {
		if err := CheckStation(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	long := "a"
	for len(long) < 65 {
		long += "b"
	}
	for _, bad := range []string{"", "../evil", "-x", ".x", "a/b", "a b", long, "é"} {
		err := CheckStation(bad)
		if err == nil || err.Error() != "--station must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}" {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
}

func TestCheckDate(t *testing.T) {
	for _, ok := range []string{"2026-10-01", "2024-02-29", "0001-01-01", "9999-12-31"} {
		if err := CheckDate(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"2026-13-45", "2026-02-30", "0000-00-00", "0000-01-01", "2026-1-01",
		"2026-10-01 ", "2026/10/01", "20261001", ""} {
		err := CheckDate(bad)
		if err == nil || err.Error() != "--date must be a real calendar date, YYYY-MM-DD" {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
}

func TestFileNameAndRow(t *testing.T) {
	name := FileName("2026-10-01", "LLM")
	if name != "2026-10-01-LLM-session.md" {
		t.Errorf("name = %q", name)
	}
	if got := IndexRow(name, "LLM", "2026-10-01", 3); got != "| 2026-10-01-LLM-session.md | LLM | 2026-10-01 | pending | 3 |" {
		t.Errorf("row = %q", got)
	}
}

func TestIndexRows(t *testing.T) {
	const n = "2026-10-01-LLM-session.md"
	row := "| " + n + " | LLM | 2026-10-01 | pending | 1 |"
	for _, tc := range []struct{ name, in, want string }{
		{"appends", "# head\n| a |\n", "# head\n| a |\n" + row + "\n"},
		{"appends without a final newline", "# head", "# head\n" + row + "\n"},
		{"empty index", "", row + "\n"},
		{"replaces the row in place", "h\n| " + n + " | old |\nt\n", "h\n" + row + "\nt\n"},
		{"drops later duplicates", "| " + n + " | 1 |\nx\n| " + n + " | 2 |\n", row + "\nx\n"},
		{"a longer name is another row", "| " + n + "x | old |\n", "| " + n + "x | old |\n" + row + "\n"},
		{"CRLF and lone CR split like Python", "a\r\nb\rc\n", "a\nb\nc\n" + row + "\n"},
		{"other Python line breaks", "a\vb\fc\x1cd e\u0085f", "a\nb\nc\nd\ne\nf\n" + row + "\n"},
		{"blank lines are kept", "a\n\nb\n", "a\n\nb\n" + row + "\n"},
	} {
		if got := IndexRows(tc.in, n, row); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
