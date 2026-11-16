package retro

// A batch the station refused with nothing applied (journal state refused, bench retro
// 2026-11-15 delta 4) is its own candidate and summary count, never an in-doubt one.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestARefusedBatchIsItsOwnCandidateNotAnInDoubtOne(t *testing.T) {
	s := newStateDir(t)
	j := journal.New(s.dir)
	for _, r := range []*pyjson.Object{
		obj("batch_id", b1, "ts", "2026-10-01T10:00:00+00:00", "tool", "n4_create_link", "ops", []any{obj("ssc", "checkLinks")},
			"phase", "intent", "station_name", "LLM", "inverse_plan", []any{}, "targets", obj()),
		obj("batch_id", b1, "ts", "2026-10-01T10:00:01+00:00", "phase", "result", "accepted", nil, "inverse", []any{},
			"verdict", "refused", "refused", true, "station_error", "ValueError: Duplicate link already exists"),
	} {
		if err := j.Append(r); err != nil {
			t.Fatal(err)
		}
	}
	out := s.draft("")
	c := candidates(t, out)
	if got := evidence(c.get("verdict:refused")); !strings.Contains(got, b1) {
		t.Errorf("evidence = %s", got)
	}
	if c.m["in_doubt"] != nil {
		t.Errorf("in_doubt candidate = %v", c.m["in_doubt"])
	}
	if md := get(out, "markdown").(string); !strings.Contains(md, "Journaled batches: 1 (refused 1)") {
		t.Errorf("markdown = %s", md)
	}
}
