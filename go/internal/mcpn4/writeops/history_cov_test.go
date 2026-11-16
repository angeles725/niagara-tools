package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// covArgs are cov-mode arguments: no interval.
func covArgs(kv ...any) *pyjson.Object {
	return without(histArgs(kv...), "interval_ms")
}

func TestAddHistoryCovModeCreatesAChangeOfValueExtensionWithoutInterval(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptBool, typedPoint("control:BooleanWritable", "12"))
	p, err := planHistory(c, covArgs("points", []any{ptA, ptBool}, "mode", "cov"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "a", "h": "12", "n": "BooleanCov", "b": {"nm": "p", "t": "history:BooleanCovHistoryExt", "s": [` +
		`{"nm": "p", "n": "historyName", "t": "baja:Format", "v": "S%parent.parent.parent.name%_%parent.name%"}, ` +
		`{"nm": "p", "n": "enabled", "t": "baja:Boolean", "v": "true"}]}}`
	if got := js(t, p.Ops[1]); got != want {
		t.Errorf("op\n got  %s\n want %s", got, want)
	}
	if !strings.Contains(js(t, p.Ops[0]), `"n": "NumericCov", "b": {"nm": "p", "t": "history:NumericCovHistoryExt"`) {
		t.Errorf("numeric op %s", js(t, p.Ops[0]))
	}
	note := "Capacity stays at the extension default (500 records, Roll): a change-of-value history keeps the " +
		"last 500 changes; how long that lasts depends on how often the value changes."
	if !strings.Contains(js(t, p.Notes), note) {
		t.Errorf("notes %s", js(t, p.Notes))
	}
	typed, err := planHistory(c, covArgs("points", []any{ptBool}, "ext_type", "history:BooleanCovHistoryExt"))
	if err != nil || !strings.Contains(js(t, typed.Ops[0]), `"n": "BooleanCov"`) {
		t.Errorf("a cov ext_type without mode: err = %v", err)
	}
}

func TestAddHistoryModeAndCapacityRefusals(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	for _, tc := range []struct {
		name string
		args *pyjson.Object
		want string
	}{
		{"cov with interval", histArgs("mode", "cov"), "interval_ms does not apply to mode cov"},
		{"interval without interval", covArgs(), "interval_ms is required for mode interval"},
		{"mode against type", histArgs("mode", "interval", "ext_type", "history:NumericCovHistoryExt"),
			"ext_type history:NumericCovHistoryExt is a cov extension, but mode is interval"},
		{"zero", histArgs("capacity", pyjson.IntLiteral("0")), "capacity must be"},
		{"negative", histArgs("capacity", pyjson.IntLiteral("-5")), "capacity must be"},
		{"too big", histArgs("capacity", pyjson.IntLiteral("2147483648")), "capacity must be"},
		{"fraction", histArgs("capacity", pyjson.FloatNumber(1.5)), "capacity must be"},
		{"text", histArgs("capacity", "lots"), "capacity must be a whole number of records from 1 to 2147483647, " +
			"or \"unlimited\", got 'lots'"},
		{"bool", histArgs("capacity", true), "capacity must be"},
	} {
		_, err := planHistory(c, tc.args)
		wantRefusal(t, tc.name, err, tc.want)
	}
}

func TestAddHistoryCapacityIsSetInTheSameBatch(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	p, err := planHistory(c, histArgs("capacity", pyjson.IntLiteral("20160")))
	if err != nil {
		t.Fatal(err)
	}
	set := `{"nm": "s", "h": "11", "n": "NumericInterval/historyConfig/capacity", "b": {"nm": "p", "t": "history:Capacity", "v": "1:20160"}}`
	if len(p.Ops) != 4 || js(t, p.Ops[3]) != set || len(p.Inverse) != 2 {
		t.Fatalf("ops %s, inverse %s", js(t, p.Ops), js(t, p.Inverse))
	}
	if note := js(t, p.Notes); !strings.Contains(note, "Capacity 20160 records (Roll): about 70 d at 5 min.") {
		t.Errorf("notes %s", note)
	}
	unlimited, err := planHistory(c, covArgs("mode", "cov", "capacity", "unlimited"))
	if err != nil || !strings.Contains(js(t, unlimited.Ops[1]), `"t": "history:Capacity", "v": "0:0"`) ||
		!strings.Contains(js(t, unlimited.Notes), "Capacity unlimited: the history keeps every record") {
		t.Errorf("unlimited: err = %v, plan %v", err, unlimited)
	}
	replies := []any{[]any{obj("nn", "NumericInterval")}, []any{}, []any{obj("nn", "NumericInterval1")}, []any{}}
	inverse, err := writeops.AddHistory.Inverse(p, replies)
	if got := js(t, inverse); err != nil || got != `[{"nm": "v", "h": "10", "n": "NumericInterval"}, {"nm": "v", "h": "11", "n": "NumericInterval1"}]` {
		t.Errorf("inverse = %s, err = %v", got, err)
	}
	head := *p
	head.Ops = p.Ops[:3]
	if inverse, err = writeops.AddHistory.Inverse(&head, replies[:3]); err != nil || len(inverse) != 2 {
		t.Errorf("a batch cut after the second add: inverse = %s, err = %v", js(t, inverse), err)
	}
}

func TestAddHistoryReadbackChecksCapacityAndSkipsCovInterval(t *testing.T) {
	c := histPoints(t, ptA, point("10"))
	args := histArgs("points", []any{ptA}, "capacity", pyjson.IntLiteral("20160"))
	p, err := planHistory(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", "NumericInterval")}, []any{}}
	inverse, _ := writeops.AddHistory.Inverse(p, replies)
	for capacity, want := range map[string]string{"1:20160": "verified", "1:500": "mismatch"} {
		c.nodes = obj(ptA, doneExt("NumericInterval", "true", "S2004_RA_T", "historyConfig/capacity",
			obj("t", "history:Capacity", "v", capacity)))
		_, _, observed, verdict, err := writeops.AddHistory.Readback(c, args, p, replies, inverse)
		if err != nil || verdict != want || !strings.Contains(js(t, observed), `"capacity": "`+capacity+`"`) {
			t.Errorf("capacity %s: verdict = %s, err = %v, observed %s", capacity, verdict, err, js(t, observed))
		}
	}
	cov := covArgs("points", []any{ptA}, "mode", "cov")
	if p, err = planHistory(c, cov); err != nil {
		t.Fatal(err)
	}
	ext := without(doneExt("NumericCov", "true", "S2004_RA_T"), "NumericCov/interval")
	ext.Set("NumericCov", obj("h", "20", "t", "history:NumericCovHistoryExt"))
	c.nodes = obj(ptA, ext)
	replies = []any{[]any{obj("nn", "NumericCov")}}
	inverse, _ = writeops.AddHistory.Inverse(p, replies)
	if _, _, observed, verdict, _ := writeops.AddHistory.Readback(c, cov, p, replies, inverse); verdict != "verified" ||
		strings.Contains(js(t, observed), `"interval"`) {
		t.Errorf("cov: verdict = %s, observed %s", verdict, js(t, observed))
	}
}
