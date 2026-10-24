package citeresolve

import "testing"

// Rerendered.txt is a guide page after a re-render: the "Absolute Logoff option" sentence sits on line 8.
func TestAnchorDrift(t *testing.T) {
	t.Parallel()
	const page = "guides-clean/Platform/Rerendered.txt"
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"anchor on the cited line", "[ev: niagara-help 4.15.6.20 " + page + `:8 "Absolute Logoff option"]`, []want{{Resolved, "As of Niagara 4.15"}}},
		{"anchor two lines below the cited line", "[ev: guide Platform/Rerendered.txt:6 \"Absolute Logoff option\"]", []want{{Resolved, "If your station"}}},
		{"anchor inside a cited range", "[ev: guide Platform/Rerendered.txt:1-12 \"Parent topic\"]", []want{{Resolved, "TITLE"}}},
		{"anchor whitespace is not significant", "[ev: guide Platform/Rerendered.txt:10 \"Absolute Logoff Enabled property\"]", []want{{Resolved, "Per user"}}},
		{"anchor moved: drift names the new line", "[ev: guide Platform/Rerendered.txt:3 \"Absolute Logoff option\"]",
			[]want{{Drift, `anchor "Absolute Logoff option" not within 2 lines of :3; at line 8`}}},
		{"anchor three lines away is drift", "[ev: guide Platform/Rerendered.txt:5 \"Absolute Logoff option\"]", []want{{Drift, "at line 8"}}},
		{"anchor on several other lines lists them", "[ev: guide Platform/Rerendered.txt:1 \"Absolute Logoff\"]", []want{{Drift, "at lines 8, 10"}}},
		{"anchor gone", "[ev: guide Platform/Rerendered.txt:8 \"Relative Logoff\"]", []want{{Drift, `anchor "Relative Logoff" not in Rerendered.txt`}}},
		{"no anchor keeps the line check only", "[ev: guide Platform/Rerendered.txt:3]", []want{{Resolved, ""}}},
		{"a line past EOF stays out of range", "[ev: guide Platform/Rerendered.txt:40 \"Absolute Logoff option\"]", []want{{LineOutOfRange, "file has 12"}}},
		{"an anchor outside guides-clean is not checked", "x `types/platform.md:1` \"not in the card\" y",
			[]want{{Resolved, "unchecked: anchor outside guides-clean"}}},
		{"inline backticked pointer with an anchor", "| `niagara-help/versions/4.15.6.20/" + page + ":3` \"Absolute Logoff option\" |",
			[]want{{Drift, "at line 8"}}},
	})
}

func TestAnchorDriftOnAHistoryCard(t *testing.T) {
	t.Parallel()
	if got := scoped(Result{Status: Drift, Detail: "x"}, History, "retros"); got.Status != HistoricalDrift {
		t.Errorf("history DRIFT = %s, want %s", got.Status, HistoricalDrift)
	}
	if got := scoped(Result{Status: Drift}, Live, ""); got.Status != Drift || passing[Drift] {
		t.Errorf("live DRIFT = %s (passing %v), want a strict finding", got.Status, passing[Drift])
	}
}
