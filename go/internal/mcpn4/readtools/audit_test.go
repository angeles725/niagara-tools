package readtools_test

// n4_read_audit, station-audits S7: who changed what, from the station's AuditHistory, filtered
// by component, slot, user, operation and time range, in one call.

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const (
	auditSelect = "history:/FakeStation/AuditHistory|bql:select timestamp.encodeToString, operation, target, " +
		"slotName, oldValue, value, userName"
	auditTail = " order by timestamp desc|view:file:ITableToCsv"
	// auditRecords are three audit rows newest first, as a localized 4.15 station heads them.
	auditRecords = "Encode To String,Operación,Destino,Nombre de ranura,Valor anterior,Valor,Usuario\n" +
		"2026-11-16T10:05:00.000-06:00,Changed,/Plant/Comp1/Control,overAmpsDelay,5 s,3 s,admin\n" +
		"2026-11-16T09:00:00.000-06:00,Invoked,/Plant/Comp1/Control,resetFault,,,operador\n" +
		"2026-11-15T18:30:00.000-06:00,Changed,/Plant/Comp2,setpoint,\"4,5\",\"5,0\",admin\n"
)

// auditServer is a connected server whose station answers the audit records (body) and their
// count (count), at 2026-11-16T11:00-06:00.
func auditServer(t *testing.T, body, count string) *readtoolstest.Server {
	t.Helper()
	now := time.Date(2026, 11, 16, 17, 0, 0, 0, time.UTC)
	saved := bql.Now
	bql.Now = func() time.Time { return now }
	t.Cleanup(func() { bql.Now = saved })
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "select count(*)", Body: "count(toString)\n" + count + "\n"},
		{Needle: "userName", Body: body},
	}
	return s
}

func auditRows(t *testing.T, out *pyjson.Object) []any {
	t.Helper()
	rows, _ := field(out, "rows").([]any)
	return rows
}

func TestReadAuditAnswersWhoChangedOverAmpsDelay(t *testing.T) {
	s := auditServer(t, strings.SplitAfterN(auditRecords, "\n", 3)[0]+strings.SplitAfterN(auditRecords, "\n", 3)[1], "1")
	out := callOK(t, s, "n4_read_audit", "target", "station:|slot:/Plant/Comp1", "slot", "overAmpsDelay")
	where := "(target like '/Plant/Comp1' or target like '/Plant/Comp1/*') and slotName = 'overAmpsDelay'"
	want := []string{auditSelect + " where " + where + auditTail,
		"history:/FakeStation/AuditHistory|bql:select count(*) where " + where + "|view:file:ITableToCsv"}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ORDs sent\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	rows := auditRows(t, out)
	wantRow := "{'timestamp': '2026-11-16T10:05:00.000-06:00', 'operation': 'Changed', 'target': '/Plant/Comp1/Control', " +
		"'slot': 'overAmpsDelay', 'old_value': '5 s', 'value': '3 s', 'user': 'admin'}"
	if len(rows) != 1 || pyjson.Repr(rows[0]) != wantRow {
		t.Fatalf("rows %s\nwant [%s]", pyjson.Repr(rows), wantRow)
	}
	if field(out, "history_id") != "/FakeStation/AuditHistory" || field(out, "where") != where ||
		field(out, "order") != "newest_first" || field(out, "truncated") != false || field(out, "total_records") != 1 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
}

func TestReadAuditFiltersByUserAndTimeRange(t *testing.T) {
	s := auditServer(t, auditRecords, "3")
	out := callOK(t, s, "n4_read_audit", "user", "admin", "operation", "Changed",
		"start", "24h", "end", "2026-11-16T10:30:00.000-06:00")
	got := strings.Join(sent(s), "\n")
	for _, part := range []string{
		"history:/FakeStation/AuditHistory?start=2026-11-15T17:00:00.000Z;end=2026-11-16T10:30:00.000-06:00|bql:",
		" where userName = 'admin' and operation = 'Changed' order by timestamp desc"} {
		if !strings.Contains(got, part) {
			t.Errorf("ORDs sent %s\nlack %s", got, part)
		}
	}
	if field(out, "range") != "start=2026-11-15T17:00:00.000Z;end=2026-11-16T10:30:00.000-06:00" ||
		len(auditRows(t, out)) != 3 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	s = auditServer(t, auditRecords, "3")
	callOK(t, s, "n4_read_audit", "period", "today")
	if got := strings.Join(sent(s), "\n"); !strings.Contains(got, "AuditHistory?period=today|bql:select timestamp") ||
		strings.Contains(got, " where ") {
		t.Errorf("ORDs sent %s", got)
	}
}

func TestReadAuditRefusesAnInjectionShapedValueBeforeAnyRequest(t *testing.T) {
	for _, kv := range [][]any{
		{"slot", "x$27$20or$20slotName$20like$20$27$2a"},
		{"user", "admin' or userName like '*"},
		{"target", "station:|slot:/Plant/x$27"},
		{"target_glob", "/Plant/*' or target like '*"},
		{"target", "station:|slot:/Plant", "target_glob", "/Plant/*"},
		{"period", "today", "start", "24h"},
		{"start", "yesterday"},
	} {
		s := auditServer(t, auditRecords, "3")
		text := callFail(t, s, "n4_read_audit", kv...)
		if len(sent(s)) != 0 {
			t.Errorf("%v sent %v before refusing: %s", kv, sent(s), text)
		}
	}
	s := auditServer(t, auditRecords, "3")
	if text := rpcError(t, s, "n4_read_audit", "operation", "Changed' or operation like '*"); len(sent(s)) != 0 ||
		!strings.Contains(text, "operation") {
		t.Errorf("operation outside the enum: %s, sent %v", text, sent(s))
	}
}

func TestReadAuditSaysWhenMoreRecordsMatchThanTheLimit(t *testing.T) {
	s := auditServer(t, auditRecords, "57")
	out := callOK(t, s, "n4_read_audit", "limit", pyjson.IntLiteral("2"))
	if len(auditRows(t, out)) != 2 || field(out, "truncated") != true || field(out, "total_records") != 57 ||
		field(out, "row_count") != 2 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	s = auditServer(t, auditRecords, "many")
	out = callOK(t, s, "n4_read_audit")
	if field(out, "total_records") != nil || !strings.Contains(pyjson.Str(field(out, "total_records_note")), "count") ||
		field(out, "truncated") != false {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
}

func TestReadAuditWarnsWhenAFilteredReadIsEmptyAndItsCountFailed(t *testing.T) {
	s := auditServer(t, strings.SplitAfterN(auditRecords, "\n", 2)[0], "null")
	out := callOK(t, s, "n4_read_audit", "user", "nobody")
	const want = "no rows came back and the record count (select count(*)) did not answer: " +
		"the filter may name no field, so this empty list does not show that no record matches"
	if got := field(out, "warning"); got != want || field(out, "row_count") != 0 {
		t.Errorf("warning %v, reply %s", got, pyjson.Repr(out))
	}
	s = auditServer(t, strings.SplitAfterN(auditRecords, "\n", 2)[0], "0")
	if _, has := callOK(t, s, "n4_read_audit", "user", "nobody").Get("warning"); has {
		t.Error("warning when the count answered 0")
	}
}
