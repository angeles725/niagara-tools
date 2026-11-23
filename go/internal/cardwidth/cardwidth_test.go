package cardwidth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func words(n int) string { return strings.TrimSpace(strings.Repeat("word ", n)) }

// TestOverlongReadsWhatCouldBeWrapped pins which lines the bound refuses.
func TestOverlongReadsWhatCouldBeWrapped(t *testing.T) {
	long := words(40) // 199 characters, every space breakable
	span := "`" + strings.Repeat("x y ", 40) + "`"
	for _, c := range []struct {
		name, text string
		want       []int
	}{
		{"short lines", "a\nb c\n", nil},
		{"a long paragraph line", "ok\n" + long + "\n", []int{2}},
		{"exactly the limit", strings.Repeat("a", Limit-1) + " b", nil},
		{"one past the limit with text after it", strings.Repeat("a", Limit) + " b", []int{1}},
		{"a bullet", "- " + long, []int{1}},
		{"a line that one code span fills", "  " + span, nil},
		{"a code span, then words past the limit", span + " " + words(5), []int{1}},
		{"one long tag then nothing", "  [ev: retro " + strings.Repeat("a-", 70) + " Δ1]", nil},
		{"a trailing hard break is no text", strings.Repeat("a", Limit) + "  ", nil},
		{"a table row", "| " + long + " |", nil},
		{"a heading", "## " + long, nil},
		{"a comment", "<!-- " + long + " -->", nil},
		{"fenced code", "```\n" + long + "\n```\n" + long, []int{4}},
		{"wide characters count once", strings.Repeat("Ω", Limit-1) + " b", nil},
	} {
		got := Overlong(c.text)
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParseBaseline(t *testing.T) {
	got, err := ParseBaseline("# why\n\nBUILD.md 3\n  x.md 12 \n")
	if err != nil || got["BUILD.md"] != 3 || got["x.md"] != 12 || len(got) != 2 {
		t.Errorf("baseline %v, error %v", got, err)
	}
	for _, bad := range []string{"a.md", "a.md x", "a.md 0", "a.md 1 2", "a.md -1"} {
		if _, err := ParseBaseline(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

// TestCheckOnlyLetsTheCountGoDown: more overlong lines than the baseline, a card with none in it and a
// baseline above the count are each a message; a card at its baseline is clean.
func TestCheckOnlyLetsTheCountGoDown(t *testing.T) {
	counts := map[string]int{"a.md": 0, "b.md": 5, "c.md": 2, "d.md": 1}
	msgs := Check(counts, map[string]int{"b.md": 5, "c.md": 3, "d.md": 0, "gone.md": 1})
	want := []string{
		"c.md: 2 line(s) over 120 characters, baseline says 3: lower it in go/internal/cardwidth/baseline.txt",
		"d.md: 1 line(s) over 120 characters, 0 allowed: wrap them",
		"gone.md: in the baseline but not a live card",
	}
	if len(msgs) != len(want) {
		t.Fatalf("messages %q", msgs)
	}
	for i := range want {
		if !strings.HasPrefix(msgs[i], want[i]) {
			t.Errorf("message %d: %q, want prefix %q", i, msgs[i], want[i])
		}
	}
}

func TestCountsReadsTheLiveCardsOnly(t *testing.T) {
	root := t.TempDir()
	long := words(40) + "\n"
	for rel, text := range map[string]string{
		"build-n4-module-kit/BUILD-LOOP.md":      long,
		"build-n4-module-kit/retros/2026-x.md":   long,
		"build-n4-module-kit/ledger.d/2026-x.md": long,
		"build-n4-module-kit/types/driver.md":    "short\n",
		"README.md":                              long + long,
		"docs/other.md":                          long,
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Counts(root)
	want := map[string]int{"build-n4-module-kit/BUILD-LOOP.md": 1, "build-n4-module-kit/types/driver.md": 0, "README.md": 2}
	if err != nil || len(got) != len(want) {
		t.Fatalf("counts %v, error %v", got, err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %d, want %d", k, got[k], v)
		}
	}
}

// TestTheLiveCardsStayWithinTheirBaseline is the lint on the committed cards: BUILD-LOOP.md and
// skill/SKILL.md are wrapped (not in the baseline), the other cards do not get worse.
func TestTheLiveCardsStayWithinTheirBaseline(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile("baseline.txt")
	if err != nil {
		t.Fatal(err)
	}
	base, err := ParseBaseline(string(text))
	if err != nil {
		t.Fatal(err)
	}
	counts, err := Counts(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range Check(counts, base) {
		t.Error(msg)
	}
	for _, card := range []string{"build-n4-module-kit/BUILD-LOOP.md", "build-n4-module-kit/skill/SKILL.md"} {
		if n, ok := counts[card]; !ok || n != 0 {
			t.Errorf("%s must stay wrapped: %d overlong line(s), live = %v", card, n, ok)
		}
	}
}
