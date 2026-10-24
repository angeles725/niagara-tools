package retro

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unreadableAudit is the audit body of the Python black-box test (issue #466): readable rows
// end in CRLF, CR and LF; bad lines end in CRLF and CR, and one is cut at the 8192-byte read
// chunk boundary (a lead byte at 8191, an ASCII byte at 8192). Lines 2, 4 and 5 are unreadable.
func unreadableAudit(t *testing.T) []byte {
	t.Helper()
	row := func(ts string) string {
		return strings.TrimSuffix(refused("2026-10-01T10:"+ts+":00+00:00", "n4_set_slot", "write budget exhausted: "+ts), "\n")
	}
	head := []byte(row("00") + "\r\n\xff bad\r\n" + row("01") + "\r\xfe\r")
	cut := append(append([]byte(`{"x": "`), bytes.Repeat([]byte("a"), 8191-len(head)-7)...), "\xc3(\"}\n"...)
	if at := len(head) + bytes.IndexByte(cut, 0xc3); at != 8191 {
		t.Fatalf("the cut lead byte is at %d, not 8191", at)
	}
	return append(append(head, cut...), row("02")+"\n"...)
}

func TestAnUnreadableAuditLineCostsThatLineOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "audit.jsonl"), unreadableAudit(t), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := draft(dir, "LLM", "2026-10-24", "1970-01-01T00:00:00Z", nil, sinceFilter)
	if err != nil {
		t.Fatalf("draft failed: %v", err)
	}
	md := get(out, "markdown").(string)
	for _, want := range []string{"- Audit entries: 3 (refused 3)",
		"- Incomplete: 3 audit line(s) are unreadable and left out of this draft",
		"\n- unreadable audit line 2: UnicodeDecodeError", "\n- unreadable audit line 4: UnicodeDecodeError",
		"\n- unreadable audit line 5: UnicodeDecodeError"} {
		if !strings.Contains(md, want) {
			t.Errorf("draft lacks %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, "unreadable audit line 1:") || strings.Contains(md, "unreadable audit line 3:") {
		t.Errorf("a readable line is named unreadable:\n%s", md)
	}
	if n := len(get(out, "candidates").([]any)); n != 1 {
		t.Errorf("%d candidates, want the one budget refusal class", n)
	}
}
