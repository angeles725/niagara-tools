package readtools_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// behaviorCSV is a ControlPoint answer as the 4.15 station writes it: a BOM, localized headers,
// a locale decimal comma; status.bits 0 is ok, 2 is fault... (decoded by box.StatusFlags).
func behaviorCSV(rows ...string) string {
	return "\xef\xbb\xbfRuta de slot,Valor,Bits\n" + strings.Join(rows, "\n") + "\n"
}

func behaviorServer(t *testing.T, body string) (*readtoolstest.Server, *readtoolstest.Station, string) {
	t.Helper()
	st := readtoolstest.NewStation()
	st.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "control:ControlPoint", Body: body}}
	dir := filepath.Join(t.TempDir(), "state")
	s := readtoolstest.NewServer(st, session.Config{StateDir: dir})
	s.Connect(t)
	return s, st, dir
}

func TestBehaviorSnapshotWritesThePointsAndRefusesATakenName(t *testing.T) {
	s, st, dir := behaviorServer(t, behaviorCSV("slot:/A/T1,\"21,5\",0", "slot:/A/F1,true,0"))
	text, isErr, out := s.Call(t, "n4_behavior_snapshot", "name", "before", "scope", "station:|slot:/A")
	if isErr {
		t.Fatal(text)
	}
	if got := st.OrdRequests[0].Ord; !strings.HasPrefix(got, "station:|slot:/A|bql:select slotPath, out.value, status.bits from control:ControlPoint") {
		t.Errorf("ord %s", got)
	}
	file := filepath.Join(dir, "snapshots", "before.json")
	for k, want := range map[string]string{"points": "2", "truncated": "false", "scope": `"station:|slot:/A"`, "file": `"` + file + `"`} {
		v, _ := out.Get(k)
		if got := marshal(t, v); got != want {
			t.Errorf("%s = %s, want %s", k, got, want)
		}
	}
	data, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(data), `{"path": "slot:/A/T1", "value": "21,5", "status": "0"}`) {
		t.Fatalf("file %s: %v", data, err)
	}
	if info, _ := os.Stat(file); info.Mode().Perm() != 0o600 {
		t.Errorf("mode %o", info.Mode().Perm())
	}
	if text, isErr, _ := s.Call(t, "n4_behavior_snapshot", "name", "before"); !isErr || !strings.Contains(text, "already exists") {
		t.Errorf("second snapshot: %v %s", isErr, text)
	}
	if text, isErr, _ := s.Call(t, "n4_behavior_snapshot", "name", "../x"); !isErr || !strings.Contains(text, "name must be") {
		t.Errorf("bad name: %v %s", isErr, text)
	}
}

func TestBehaviorDiffAgainstTheLiveStationListsEveryKindOfChange(t *testing.T) {
	s, st, _ := behaviorServer(t, behaviorCSV("slot:/A/T1,\"21,5\",0", "slot:/A/F1,true,0", "slot:/A/Gone,1,0", "slot:/A/Same,7,0"))
	if text, isErr, _ := s.Call(t, "n4_behavior_snapshot", "name", "before", "scope", "station:|slot:/A"); isErr {
		t.Fatal(text)
	}
	st.OrdRoutes[0].Body = behaviorCSV("slot:/A/T1,\"21,6\",0", "slot:/A/F1,false,2", "slot:/A/Same,7,0", "slot:/A/New,0,0")
	text, isErr, out := s.Call(t, "n4_behavior_diff", "before", "before", "tolerance", pyjson.FloatNumber(0.05))
	if isErr {
		t.Fatal(text)
	}
	if got := st.OrdRequests[1].Ord; !strings.HasPrefix(got, "station:|slot:/A|bql:") {
		t.Errorf("live read %s", got)
	}
	for k, want := range map[string]string{
		"summary":        `{"status_changed": 1, "value_changed": 2, "unchanged": 1, "missing": 1, "new": 1}`,
		"status_changed": `{"count": 1, "items": [{"path": "slot:/A/F1", "before": [], "after": ["fault"]}], "more": 0}`,
		"value_changed": `{"count": 2, "items": [{"path": "slot:/A/F1", "before": "true", "after": "false"}, ` +
			`{"path": "slot:/A/T1", "before": "21,5", "after": "21,6"}], "more": 0}`,
		"missing": `{"count": 1, "items": ["slot:/A/Gone"], "more": 0}`,
		"new":     `{"count": 1, "items": ["slot:/A/New"], "more": 0}`,
	} {
		v, _ := out.Get(k)
		if got := marshal(t, v); got != want {
			t.Errorf("%s = %s\n want %s", k, got, want)
		}
	}
}

func TestBehaviorDiffOfTwoSnapshotsNeedsNoStationAndLimitsItems(t *testing.T) {
	s, st, _ := behaviorServer(t, behaviorCSV("slot:/A/P1,1,0", "slot:/A/P2,2,0"))
	s.Call(t, "n4_behavior_snapshot", "name", "b")
	st.OrdRoutes[0].Body = behaviorCSV("slot:/A/P1,5,0", "slot:/A/P2,6,0")
	s.Call(t, "n4_behavior_snapshot", "name", "a")
	reads := len(st.OrdRequests)
	text, isErr, out := s.Call(t, "n4_behavior_diff", "before", "b", "after", "a", "limit", pyjson.IntLiteral("1"))
	if isErr {
		t.Fatal(text)
	}
	if len(st.OrdRequests) != reads {
		t.Errorf("the diff of two snapshots read the station")
	}
	v, _ := out.Get("value_changed")
	if got := marshal(t, v); got != `{"count": 2, "items": [{"path": "slot:/A/P1", "before": "1", "after": "5"}], "more": 1}` {
		t.Errorf("value_changed %s", got)
	}
}

func TestBehaviorDiffDoesNotJudgeMissingPointsFromATruncatedSide(t *testing.T) {
	s, _, _ := behaviorServer(t, behaviorCSV("slot:/A/P1,1,0", "slot:/A/P2,2,0"))
	s.Call(t, "n4_behavior_snapshot", "name", "cut", "max_points", pyjson.IntLiteral("1"))
	text, isErr, out := s.Call(t, "n4_behavior_diff", "before", "cut")
	if isErr {
		t.Fatal(text)
	}
	v, _ := out.Get("missing")
	if got := marshal(t, v); !strings.HasPrefix(got, `{"not_checked": "not_checked: a side was truncated`) {
		t.Errorf("missing %s", got)
	}
}

func TestBehaviorDiffRefusesAnUnknownSnapshotAndALiveReadWithoutSession(t *testing.T) {
	st := readtoolstest.NewStation()
	dir := filepath.Join(t.TempDir(), "state")
	s := readtoolstest.NewServer(st, session.Config{StateDir: dir})
	if text, isErr, _ := s.Call(t, "n4_behavior_diff", "before", "nope"); !isErr || !strings.Contains(text, "no snapshot 'nope'") {
		t.Errorf("unknown: %v %s", isErr, text)
	}
	os.MkdirAll(filepath.Join(dir, "snapshots"), 0o700)
	os.WriteFile(filepath.Join(dir, "snapshots", "x.json"), []byte(`{"format": "mcp-n4.behavior-snapshot/v1", "points": []}`), 0o600)
	if text, isErr, _ := s.Call(t, "n4_behavior_diff", "before", "x"); !isErr || !strings.Contains(text, "call n4_connect") {
		t.Errorf("no session: %v %s", isErr, text)
	}
	os.WriteFile(filepath.Join(dir, "snapshots", "y.json"), []byte(`{"format": "other"}`), 0o600)
	if text, isErr, _ := s.Call(t, "n4_behavior_diff", "before", "y", "after", "x"); !isErr || !strings.Contains(text, "is not a mcp-n4.behavior-snapshot/v1 file") {
		t.Errorf("foreign: %v %s", isErr, text)
	}
}
