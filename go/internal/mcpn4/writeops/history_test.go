package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const (
	ptA = "station:|slot:/Drivers/BacnetNetwork/$32004/points/RA_T"
	ptB = "station:|slot:/Drivers/BacnetNetwork/$32005/points/RA_T"
)

// pointsClient serves the given ORD -> nodes map with the recorded loadTypes reply.
func pointsClient(t *testing.T, kv ...any) *typed {
	t.Helper()
	c := newTyped(t, extNodes())
	c.nodes = obj(kv...)
	return c
}

func point(h string, children ...any) *pyjson.Object {
	n := obj("", obj("h", h, "t", "bacnet:BacnetNumericPoint"))
	for i := 0; i < len(children); i += 2 {
		n.Set(children[i].(string), children[i+1])
	}
	return n
}

func histArgs(kv ...any) *pyjson.Object {
	a := obj("points", []any{ptA, ptB}, "history_name", "S%parent.parent.parent.name%_%parent.name%",
		"interval_ms", pyjson.IntLiteral("300000"))
	for i := 0; i < len(kv); i += 2 {
		a.Set(kv[i].(string), kv[i+1])
	}
	return a
}

func TestAddHistoryPlansOneConfiguredAddPerPointInOneBatch(t *testing.T) {
	c := pointsClient(t, ptA, point("10"), ptB, point("11"))
	p, err := writeops.AddHistory.Plan(c, histArgs())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "a", "h": "10", "n": "NumericInterval", "b": {"nm": "p", "t": "history:NumericIntervalHistoryExt", "s": [` +
		`{"nm": "p", "n": "historyName", "t": "baja:Format", "v": "S%parent.parent.parent.name%_%parent.name%"}, ` +
		`{"nm": "p", "n": "interval", "t": "baja:RelTime", "v": "300000"}, ` +
		`{"nm": "p", "n": "enabled", "t": "baja:Boolean", "v": "true"}]}}`
	if len(p.Ops) != 2 || js(t, p.Ops[0]) != want {
		t.Fatalf("ops = %d, first\n got  %s\n want %s", len(p.Ops), js(t, p.Ops[0]), want)
	}
	if got := js(t, p.Inverse); got != `[{"nm": "v", "h": "10", "n": "NumericInterval"}, {"nm": "v", "h": "11", "n": "NumericInterval"}]` {
		t.Errorf("inverse = %s", got)
	}
	off, err := writeops.AddHistory.Plan(c, histArgs("enabled", false))
	if err != nil || !strings.Contains(js(t, off.Ops[1]), `"n": "enabled", "t": "baja:Boolean", "v": "false"`) {
		t.Errorf("enabled=false: err = %v, op %s", err, js(t, off.Ops[1]))
	}
	if got := writeops.AddHistory.ScopeOrds(histArgs()); len(got) != 2 {
		t.Errorf("scope ords = %v, want both points", got)
	}
}

func TestAddHistoryRefusals(t *testing.T) {
	for _, tc := range []struct {
		name string
		args *pyjson.Object
		c    *typed
		want string
	}{
		{"existing child", histArgs(), pointsClient(t, ptA, point("10", "NumericInterval", obj("t", "x")), ptB, point("11")),
			ptA + " already has a child named 'NumericInterval'"},
		{"repeated name", histArgs("history_name", "%parent.name%"), pointsClient(t, ptA, point("10"), ptB, point("11")),
			"history name 'RA_T' would repeat on " + ptA + " and " + ptB},
		{"no points", histArgs("points", []any{}), pointsClient(t), "points must list 1 to 50 point ORDs"},
		{"duplicate point", histArgs("points", []any{ptA, ptA}), pointsClient(t), "points must be distinct"},
		{"not interval type", histArgs("ext_type", "history:NumericCovHistoryExt"), pointsClient(t), "ext_type must be one of"},
		{"empty pattern", histArgs("history_name", ""), pointsClient(t), "history_name must be a non-empty"},
		{"zero interval", histArgs("interval_ms", pyjson.IntLiteral("0")), pointsClient(t), "interval_ms must be at least 1"},
		{"text interval", histArgs("interval_ms", "5m"), pointsClient(t), "whole number of milliseconds"},
	} {
		_, err := writeops.AddHistory.Plan(tc.c, tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestAddHistoryReadbackComparesEveryExtensionAndUsesTheAssignedName(t *testing.T) {
	c := pointsClient(t, ptA, point("10"), ptB, point("11"))
	args := histArgs()
	p, err := writeops.AddHistory.Plan(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", "NumericInterval")}, []any{obj("nn", "NumericInterval1")}}
	inverse, err := writeops.AddHistory.Inverse(p, replies)
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, inverse[1]); got != `{"nm": "v", "h": "11", "n": "NumericInterval1"}` {
		t.Fatalf("inverse[1] = %s", got)
	}
	done := func(name string, enabled string) *pyjson.Object {
		return point("10", name, obj("h", "20", "t", "history:NumericIntervalHistoryExt"),
			name+"/historyName", obj("t", "baja:Format", "v", "S%parent.parent.parent.name%_%parent.name%"),
			name+"/interval", obj("t", "baja:RelTime", "v", "300000"),
			name+"/enabled", obj("t", "baja:Boolean", "v", enabled))
	}
	c.nodes = obj(ptA, done("NumericInterval", "true"), ptB, done("NumericInterval1", "true"))
	_, _, observed, verdict, err := writeops.AddHistory.Readback(c, args, p, replies, inverse)
	if err != nil || verdict != "verified" {
		t.Fatalf("verdict = %s, err = %v, observed %s", verdict, err, js(t, observed))
	}
	c.nodes = obj(ptA, done("NumericInterval", "true"), ptB, done("NumericInterval1", "false"))
	if _, _, _, verdict, _ = writeops.AddHistory.Readback(c, args, p, replies, inverse); verdict != "mismatch" {
		t.Errorf("a disabled extension: verdict = %s, want mismatch", verdict)
	}
}

func TestResolveHistoryName(t *testing.T) {
	ext := ptA + "/NumericInterval"
	for _, tc := range []struct {
		pattern, want string
		ok            bool
	}{
		{"S%parent.parent.parent.name%_%parent.name%", "S2004_RA_T", true},
		{"%parent.name%", "RA_T", true},
		{"%name%", "NumericInterval", true},
		{"%parent.displayName%", "", false},
		{"%parent.name", "", false},
		{"plain", "plain", true},
	} {
		got, ok := writeops.ResolveHistoryName(tc.pattern, ext)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%q: got %q %v, want %q %v", tc.pattern, got, ok, tc.want, tc.ok)
		}
	}
}
