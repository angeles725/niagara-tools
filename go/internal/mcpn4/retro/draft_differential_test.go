package retro

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tiers"
)

const draftScript = `import json, sys
from mcp_n4 import retro
def run(c):
    try:
        return json.dumps(retro.draft(c["dir"], station=c["station"], date="2026-10-24",
                                      since=c["since"] or None, observations=c["obs"]))
    except ValueError as exc:
        return "ValueError: %s" % exc
    except Exception as exc:
        return type(exc).__name__
print(json.dumps([run(c) for c in json.load(sys.stdin)]))`

// sessionState is one state dir and session: the audit and journal bodies ("" for no file), the
// read observations as JSON, the window and the station name.
type sessionState struct {
	audit, journal, obs, since, station string
}

func line(kv ...any) string {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	b, _ := pyjson.Marshal(o, pyjson.Options{})
	return string(b) + "\n"
}

func refused(ts any, tool any, reason string) string {
	return line("ts", ts, "tool", tool, "batch_id", nil, "dry_run", false, "outcome", "refused", "reason", reason,
		"args_redacted", pyjson.NewObject())
}

func errored(ts, tool string, reason string, batch any) string {
	return line("ts", ts, "tool", tool, "batch_id", batch, "outcome", "error", "reason", reason)
}

func batch(id any, ts, verdict any, ops any, inDoubt, result bool) string {
	s := line("batch_id", id, "ts", ts, "tool", "n4_set_slot", "ops", ops, "phase", "intent", "station_name", "LLM")
	if result {
		s += line("batch_id", id, "ts", ts, "phase", "result", "verdict", verdict, "in_doubt", inDoubt)
	}
	return s
}

// sessionStates is the matrix: every refusal class (the tier refusals as tiers words them),
// read-back verdicts, in-doubt batches, BOX errors, dangling observations, windows, odd values
// in the summary, the errors Python raises, and template placeholders in the station name.
func sessionStates() []sessionState {
	// chunkCut is one audit line whose bad byte pair straddles the 8192-byte read chunk, then a
	// readable row whose two-byte character straddles the 16384-byte one (issue #466).
	pad := func(n int) string { return strings.Repeat("a", n) }
	chunkCut := []byte(`{"x": "` + pad(8191-7) + "\xc3(\"}\r\n")
	chunkCut = append(chunkCut, `{"ts": "x", "outcome": "`+pad(16384-len(chunkCut)-25)+"\u00e9\"}\n"...)
	var every string
	for i, r := range append(refusalReasons(), "nothing a class knows") {
		every += refused("2026-10-01T10:00:00+00:00", "n4_set_slot", r) + refused(i, "n4_link", r)
	}
	ok := pyjson.NewObject()
	ok.Set("nm", "a")
	ops := []any{ok, "x", pyjson.NewObject(), ok}
	verdictsBody := batch("b1", "2026-10-01T11:00:00+00:00", "mismatch", []any{}, false, true) +
		batch("b2", "2026-10-01T11:00:00Z", "failed", []any{}, false, true) +
		batch("b3", "2026-10-01T11:00:00", "partial", []any{}, false, true) +
		batch("b4", "2026-10-01 11:00", "unverified", []any{}, false, true) +
		batch("b5", "2026-10-01T11:00:00+00:00", "verified", []any{}, false, true) +
		batch("d1", "2026-10-01T11:00:00+00:00", nil, ops, false, false) +
		batch("d2", "2026-10-01T11:00:00+00:00", "unverified", []any{}, true, true) +
		batch("d3", nil, nil, "ops", false, false) + batch("bad id", 5, nil, []any{}, false, false)
	boxBody := errored("2026-10-01T11:00:01+00:00", "n4_create_component", "station call failed", "d1") +
		errored("2026-10-01T11:00:02+00:00", "n4_remove", "station call failed: x", "d3") +
		errored("2026-10-01T11:00:03+00:00", "n4_link", "station call failed", "unknown") +
		errored("2026-10-01T11:00:04+00:00", "n4_set_slot", "BoxError: HTTP 500", nil) +
		errored("2026-10-01T11:00:05+00:00", "n4_set_slot", "ValueError", "d1") +
		errored("2026-10-01T11:00:06+00:00", "bad|tool", "BoxError", "") + errored("x", "t", "BoxError", 0)
	var many string
	for i := 0; i < 7; i++ {
		many += refused("2026-10-01T10:00:0"+strconv.Itoa(i)+"+00:00", "n4_set_slot", "write budget exhausted")
	}
	odd := ""
	for _, v := range []string{`"refused"`, `"` + strings.Repeat("o", 64) + `"`, `"` + strings.Repeat("p", 65) + `"`, "null", "1", "true", "1.0", `"a|b"`, `"x"`, "2.5", "false", "0", `"None"`,
		"NaN", "-0.0", "1e400", `"True"`} {
		odd += `{"ts": "2026-10-01T10:00:00+00:00", "outcome": ` + v + `, "reason": "write budget exhausted"}` + "\n"
	}
	obs := `[{"ts": "2026-10-01T12:00:00+00:00", "ord": "station:|slot:/F", "count": 2},
		{"ts": "2026-10-01T12:05:00+00:00", "tool": "n4_bql_query", "rows": 4, "elapsed_ms": 9},
		{"ts": "2026-10-01T09:00:00+00:00", "ord": "o", "count": 3}, {"ts": "x", "ord": "o", "count": 0},
		{"ts": null, "ord": "o", "count": 1}]`
	whole := "1970-01-01T00:00:00Z"
	return []sessionState{
		{since: whole, station: "LLM"},
		{audit: every, since: whole, station: "LLM"},
		{audit: every, since: "2026-10-01T10:00:00Z", station: "LLM"},
		{audit: every, since: "2026-10-01T10:00:01", station: "LLM"},
		{journal: verdictsBody, since: whole, station: "LLM"},
		{journal: verdictsBody, since: "2026-10-01T11:00:00+00:01", station: "LLM"},
		{audit: boxBody, journal: verdictsBody, since: whole, station: "LLM"},
		{audit: boxBody, journal: verdictsBody, since: "2026-W40-4T11", station: "LLM"},
		{audit: many + refused(whole, strings.Repeat("t", 64), "not connected") +
			refused(whole, strings.Repeat("u", 65), "not connected"), since: whole, station: "LLM"},
		{audit: odd, journal: line("batch_id", "v", "phase", "intent", "ts", whole) +
			line("batch_id", "v", "phase", "result", "verdict", 3), since: whole, station: "LLM"},
		{obs: obs, since: whole, station: "LLM"},
		{obs: obs, since: "2026-10-01T10:00:00Z", station: "FakeStation"},
		{obs: obs, station: "S"},
		{audit: every + boxBody, journal: verdictsBody, obs: obs, station: "{{DATE}} | {{SUMMARY}}\n{{STATION}}"},
		{audit: many, since: "yesterday", station: "LLM"},
		{audit: many, since: "2026-10-01T10:00:00\xed\xa0\x80", station: "LLM"},
		{audit: line("ts", whole, "outcome", []any{"x"}), since: whole, station: "LLM"},
		{journal: line("batch_id", "v", "phase", "intent", "ts", whole) +
			line("batch_id", "v", "phase", "result", "verdict", []any{}), since: whole, station: "LLM"},
		{audit: errored(whole, "t", "station call failed", "n"), journal: batch("n", whole, nil, 5, false, false),
			since: whole, station: "LLM"},
		{audit: errored(whole, "t", "station call failed", "n"), journal: batch("n", whole, nil, true, false, false),
			since: whole, station: "LLM"},
		{audit: errored(whole, "t", "x", []any{1}), since: whole, station: "LLM"},
		{audit: refused(whole, "t", "write budget exhausted") + "\xff\n", since: whole, station: "LLM"},
		{audit: "\xff\r\n" + refused(whole, "t", "write budget exhausted") + "\xed\xa0\x80\r" + many + "\xfe",
			since: whole, station: "LLM"}, // issue #466: bad lines across CRLF, CR and at EOF
		{audit: string(chunkCut) + many, since: whole, station: "LLM"},
		{journal: line("batch_id", []any{1}, "phase", "intent"), since: "nope", station: "LLM"},
	}
}

// refusalReasons are the refusal texts of every class as the raise sites word them.
func refusalReasons() []string {
	return []string{
		"confirmation_token is missing or malformed: run a dry run first",
		"confirmation_token does not match this tool, these arguments", "confirmation_token expired: run again",
		"confirmation_token already used: run the dry run again", "station identity not verified: reconnect",
		"no --write-scope configured: every write is refused", "ord '/etc' is not a plain station ORD",
		"ord 'station:|slot:/X' is outside the write scope", "write budget exhausted: 2 writes",
		"write budget too small: 3 needed", "not connected: call n4_connect first",
		"refusing to write 'out' on its own: Status slots are written whole",
		"action 'foo' is not allowed in this version (allowed: set)", "value_type must be one of: bool",
		"type must look like module:Type, got 'x'",
		tiers.WriteBlock("C", "", "LLM", nil), tiers.WriteBlock("B", "4.15.3.28", "LLM", nil),
	}
}

func goDraft(t *testing.T, dir string, st sessionState) string {
	t.Helper()
	var obs []*pyjson.Object
	if st.obs != "" {
		v, err := pyjson.Loads([]byte(st.obs))
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range v.([]any) {
			obs = append(obs, o.(*pyjson.Object))
		}
	}
	out, err := draft(dir, st.station, "2026-10-24", st.since, obs, sinceFilter)
	var ve *pyjson.ValueError
	var typed interface{ PyType() string }
	switch {
	case errors.As(err, &ve):
		return "ValueError: " + ve.Msg
	case errors.As(err, &typed):
		return typed.PyType()
	case err != nil:
		return "Go error " + err.Error()
	}
	b, _ := pyjson.Marshal(out, pyjson.Options{})
	return string(b)
}

func TestDraftDifferentialPython(t *testing.T) {
	fixture, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "journal", "python-written"))
	states := sessionStates()
	dirs := []string{fixture, filepath.Join(t.TempDir(), "missing")}
	for _, st := range states {
		dir := t.TempDir()
		for name, body := range map[string]string{"audit.jsonl": st.audit, "journal.jsonl": st.journal} {
			if body != "" {
				os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600)
			}
		}
		dirs = append(dirs, dir)
	}
	states = append([]sessionState{{since: "1970-01-01T00:00:00Z", station: "LLM"}, {station: "LLM"}}, states...)
	var cases []any
	for i, st := range states {
		c := pyjson.NewObject()
		c.Set("dir", dirs[i])
		c.Set("station", st.station)
		c.Set("since", st.since)
		obs, _ := pyjson.Loads([]byte(st.obs))
		if obs == nil {
			obs = []any{}
		}
		c.Set("obs", obs)
		cases = append(cases, c)
	}
	want := runPython(t, draftScript, cases)
	kinds := map[string]int{}
	for i, st := range states {
		got := goDraft(t, dirs[i], st)
		if w := want[i].(string); got != w {
			t.Errorf("state %d (%s):\nPython %s\nGo     %s", i, dirs[i], w, got)
		}
		for _, k := range []string{"refusal:tier", "verdict:partial", "in_doubt", "box_error:", "dangling", "+2 more",
			"no new deltas", "ValueError: not an ISO", "unreadable audit line", "TypeError"} {
			if strings.Contains(got, k) {
				kinds[k]++
			}
		}
	}
	if len(kinds) != 10 {
		t.Errorf("the matrix reaches only %v", kinds)
	}
}
