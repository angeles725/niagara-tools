package session_test

import (
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// TestAWriteDryRunIsRefusedWhilePaused: a write tool is a station call (F4), refused while the
// auth latch holds before anything reaches the station, a tool of a later slice too.
func TestAWriteDryRunIsRefusedWhilePaused(t *testing.T) {
	s := server()
	state, err := plan.NewState([]string{"station:|slot:/Folder"}, filepath.Join(t.TempDir(), "state"), 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	writeops.Register(s.Server, &plan.Engine{Ctx: s.Ctx, State: state})
	s.Connect(t)
	calls, rejected := 0, true
	failingReads(s, &calls, &rejected)
	mustErr(t, s, "n4_navigate", "ord", "station:|slot:/")
	before := s.Station.Requests
	for _, call := range [][]any{
		{"n4_create_component", "parent_ord", "station:|slot:/Folder", "name", "A", "type", "kitControl:NumericConst"},
		{"n4_set_slot", "ord", "station:|slot:/Folder/A", "slot", "out", "value", 1, "value_type", "baja:Double"},
		{"n4_invoke_action", "ord", "station:|slot:/Folder/A", "action", "auto"},
	} {
		contains(t, mustErr(t, s, call[0].(string), call[1:]...), "authentication failure")
	}
	if s.Station.Requests != before || calls != 1 {
		t.Errorf("%d station requests, %d reads after the latch", s.Station.Requests-before, calls)
	}
}
