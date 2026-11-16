package readtools_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A numeric history with a nan or inf record (a faulted source) keeps its min and max: the
// non-finite records are counted, never compared.
func TestReadHistorySummaryKeepsMinMaxPastANonFiniteRecord(t *testing.T) {
	s := summaryServer(t, []string{"15.0", "nan", "23.1", "+inf", "20.0"}, []string{"{ok}", "{fault}", "{ok}", "{fault}", "{ok}"})
	out := readOK(t, s, "history_id", "/FakeStation/NW", "summary", true)
	sum := field(out, "summary").(*pyjson.Object)
	min, _ := sum.Get("min")
	max, _ := sum.Get("max")
	if field(sum, "numeric") != true || min == nil || max == nil ||
		pyjson.Repr(min) != "{'value': 15.0, 'timestamp': '2026-11-16T02:20:00.000-06:00'}" ||
		pyjson.Repr(max) != "{'value': 23.1, 'timestamp': '2026-11-16T02:22:00.000-06:00'}" {
		t.Errorf("summary %s", pyjson.Repr(sum))
	}
}
