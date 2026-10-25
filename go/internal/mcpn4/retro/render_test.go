package retro

// Go twins of the test_retro.TestRender tests and of the template test of TestDerive
// (go_server.go_twin("retro", "unit")).

import (
	"strings"
	"testing"
)

func TestMarkdownFollowsTheTemplateWithDeltaTable(t *testing.T) {
	s := newStateDir(t)
	s.batch(b1, "2026-10-01T11:00:00+00:00", "mismatch", false, true)
	md := get(s.draft(""), "markdown").(string)
	if !strings.HasPrefix(md, "<!-- review-status: pending -->") || strings.Contains(md, "{{") {
		t.Errorf("%s", md)
	}
	for _, want := range []string{"## Proposed kit deltas", deltaHeader, "## Already covered", "## Honest verdict", b1, "LLM", "2026-10-01"} {
		if !strings.Contains(md, want) {
			t.Errorf("no %q", want)
		}
	}
}

func TestHonestyLineWhenThereIsNoFriction(t *testing.T) {
	s := newStateDir(t)
	s.audit("2026-10-01T10:00:00+00:00", "n4_set_slot", "executed", "verdict=verified", b1)
	s.batch(b1, "2026-10-01T10:00:00+00:00", "verified", false, true)
	out := s.draft("")
	md := get(out, "markdown").(string)
	if len(get(out, "candidates").([]any)) != 0 || !strings.Contains(md, "no new deltas;") || CountDeltas(md) != 0 {
		t.Errorf("%s", md)
	}
}

func TestCountDeltasCountsTableRowsUnderTheHeadingOnly(t *testing.T) {
	s := newStateDir(t)
	s.batch(b1, "2026-10-01T11:00:00+00:00", "mismatch", false, true)
	s.batch(b2, "2026-10-01T11:01:00+00:00", "failed", false, true)
	out := s.draft("")
	if n := CountDeltas(get(out, "markdown").(string)); n != 2 || len(get(out, "candidates").([]any)) != 2 {
		t.Errorf("%d rows", n)
	}
}

func TestPipesInCellsAreEscaped(t *testing.T) {
	if got := cell("a|b\nc\rd"); got != `a\|b c d` {
		t.Errorf("%q", got)
	}
}
