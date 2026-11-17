package readtools_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// history is a summary-query answer (newest first, as the station orders it) from minute:value
// pairs on 2026-11-16 at 10:00-06:00.
func history(pairs ...string) string {
	var b strings.Builder
	b.WriteString("\xef\xbb\xbfTimestamp,Value,Status\n")
	for i := len(pairs) - 1; i >= 0; i-- {
		m, v, _ := strings.Cut(pairs[i], ":")
		if len(m) == 1 {
			m = "0" + m
		}
		b.WriteString("2026-11-16T10:" + m + ":00.000-06:00," + v + ",{ok}\n")
	}
	return b.String()
}

func refrigServer(t *testing.T, routes ...readtoolstest.OrdRoute) (*readtoolstest.Server, *readtoolstest.Station) {
	t.Helper()
	st := readtoolstest.NewStation()
	st.OrdRoutes = routes
	s := readtoolstest.NewServer(st, session.Config{})
	s.Connect(t)
	return s, st
}

func num(s string) pyjson.Number {
	if strings.Contains(s, ".") {
		f, _ := pyjson.ParseFloat(s)
		return pyjson.FloatNumber(f)
	}
	return pyjson.IntLiteral(s)
}

func spec(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

func TestRefrigerationFaultsRunsEachRuleOnItsHistories(t *testing.T) {
	s, st := refrigServer(t,
		readtoolstest.OrdRoute{Needle: "/FakeStation/Run?", Body: history("0:true", "1:false", "10:true", "25:false")},
		readtoolstest.OrdRoute{Needle: "/FakeStation/Amps?", Body: history("0:10.0", "30:21.5", "45:25.0")},
		readtoolstest.OrdRoute{Needle: "/FakeStation/Defrost?", Body: history("0:true", "50:false")},
		readtoolstest.OrdRoute{Needle: "/FakeStation/Coil?", Body: history("0:-20.0", "40:4.0")},
	)
	rules := spec(
		"short_cycling", spec("run_history", "Run", "min_run_s", num("120"), "max_starts_per_hour", num("6")),
		"over_amps", spec("amps_history", "Amps", "limit", num("20")),
		"defrost_termination", spec("defrost_history", "Defrost", "coil_history", "Coil", "terminate_temp", num("10"),
			"max_minutes", num("50"), "slack_minutes", num("1")),
	)
	text, isErr, out := s.Call(t, "n4_refrigeration_faults", "rules", rules, "start", "2026-11-16T10:00:00.000-06:00",
		"end", "2026-11-16T11:00:00.000-06:00")
	if isErr {
		t.Fatal(text)
	}
	if got := marshal(t, lookupT(out, "summary")); got != `{"short_cycling": "1 finding(s)", "over_amps": "1 finding(s)", "defrost_termination": "1 finding(s)"}` {
		t.Errorf("summary %s", got)
	}
	want := map[string]string{
		"short_cycling": `{"count": 1, "items": [{"fact": "short_run", "run": {"start": "2026-11-16T10:00:00-06:00", ` +
			`"end": "2026-11-16T10:01:00-06:00", "minutes": 1.0, "open": false}}], "more": 0}`,
		"over_amps": `{"count": 1, "items": [{"fact": "over_amps", "hour": "2026-11-16T10:00:00-06:00", "samples_over": 2, ` +
			`"samples": 3, "peak": 25.0, "peak_time": "2026-11-16T10:45:00-06:00"}], "more": 0}`,
		"defrost_termination": `{"count": 1, "items": [{"fact": "ended_by_time", "cycle": {"start": "2026-11-16T10:00:00-06:00", ` +
			`"end": "2026-11-16T10:50:00-06:00", "minutes": 50.0, "open": false}, "max_coil_temp": 4.0}], "more": 0, ` +
			`"cycles": {"temperature": 0, "time": 1, "early": 0, "open": 0}}`,
	}
	for k, w := range want {
		if got := marshal(t, lookupT(out, k)); got != w {
			t.Errorf("%s =\n %s\nwant\n %s", k, got, w)
		}
	}
	ord := st.OrdRequests[0].Ord
	if !strings.Contains(ord, "?start=2026-11-16T10:00:00.000-06:00;end=2026-11-16T11:00:00.000-06:00|bql:select timestamp.encodeToString") {
		t.Errorf("ord %s", ord)
	}
	if got := marshal(t, lookupT(out, "requests")); got != "4" {
		t.Errorf("requests %s", got)
	}
}

func TestRefrigerationFaultsReportsAnUnreadableHistoryAsNotChecked(t *testing.T) {
	s, _ := refrigServer(t, readtoolstest.OrdRoute{Needle: "/FakeStation/Temp?", Code: 404})
	rules := spec("setpoint_not_reached", spec("temp_history", "Temp", "setpoint", num("2"), "band", num("2"), "min_minutes", num("30")))
	text, isErr, out := s.Call(t, "n4_refrigeration_faults", "rules", rules, "period", "today")
	if isErr {
		t.Fatal(text)
	}
	line := pyjson.Str(lookupT(lookupT(out, "summary").(*pyjson.Object), "setpoint_not_reached"))
	if !strings.HasPrefix(line, "not checked: temp_history /FakeStation/Temp could not be read") {
		t.Errorf("summary %s", line)
	}
}

func TestRefrigerationFaultsRefusesAnEmptyRuleSetAndAMissingThreshold(t *testing.T) {
	s, _ := refrigServer(t)
	if text, isErr, _ := s.Call(t, "n4_refrigeration_faults", "rules", spec()); !isErr || !strings.Contains(text, "rules must name at least one of") {
		t.Errorf("empty: %v %s", isErr, text)
	}
	// A missing threshold never defaults: the schema refuses it before any station read.
	reply, _ := s.Server.HandleLine([]byte(`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"n4_refrigeration_faults",` +
		`"arguments":{"rules":{"over_amps":{"amps_history":"A"}}}}}`))
	if !strings.Contains(string(reply), "missing required argument: rules.over_amps.limit") {
		t.Errorf("missing limit: %s", reply)
	}
}

func lookupT(o *pyjson.Object, k string) any { v, _ := o.Get(k); return v }
