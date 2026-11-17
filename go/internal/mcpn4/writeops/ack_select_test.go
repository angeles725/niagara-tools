package writeops_test

// n4_ack_alarm by selection (station-audits S6): select {source?, alarm_class?, before?, max?}
// resolved to exact uuids in the dry run, the token bound to that list.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// selected is the uuids of a dry run's plan.selection.records, in order.
func selected(t *testing.T, dry *pyjson.Object) []string {
	t.Helper()
	s := get(get(dry, "plan").(*pyjson.Object), "selection").(*pyjson.Object)
	var ids []string
	for _, r := range get(s, "records").([]any) {
		ids = append(ids, get(r.(*pyjson.Object), "uuid").(string))
	}
	return ids
}

func TestAckSelectBySourceSubtreeMatchesBothStoredSpellings(t *testing.T) {
	for _, source := range []string{"station:|slot:/Services", "slot:/Services", "local:|station:|slot:/Services"} {
		db := newSelectDB()
		text, isErr, dry := newAckRig(t, db).call(t, "select", sel("source", source))
		if isErr {
			t.Fatalf("%s: %s", source, text)
		}
		if got := strings.Join(selected(t, dry), ","); got != u2+","+u1 { // u2 slot:/ form, u1 local: form; u3 acked
			t.Errorf("%s: selected %s", source, got)
		}
		where, _ := bql.AlarmWhere(bql.AlarmFilter{Source: "slot:/Services", AckState: "unacked"})
		if want := bql.ComposeAlarmOrd(bql.AlarmRecordsQuery, where, bql.AlarmOrder); db.ords[0] != want {
			t.Errorf("records ORD %q, want the n4_read_alarms query %q", db.ords[0], want)
		}
		s := get(get(dry, "plan").(*pyjson.Object), "selection").(*pyjson.Object)
		rec := get(s, "records").([]any)[1].(*pyjson.Object)
		if pyjson.Repr(rec) != "{'uuid': '"+u1+"', 'source': 'local:|station:|slot:/Services/A', 'alarm_class': 'Critical', "+
			"'timestamp': '2026-11-15T00:00:00.000Z'}" || get(s, "matched") != 2 || get(s, "truncated") != false {
			t.Errorf("selection %s", pyjson.Repr(s))
		}
		if p := get(dry, "plan").(*pyjson.Object); !strings.Contains(pyjson.Repr(get(p, "ops")), "'ids': ['"+u2+"', '"+u1+"']") {
			t.Errorf("ops %s", pyjson.Repr(get(p, "ops")))
		}
	}
}

func TestAckSelectByClassAndBefore(t *testing.T) {
	text, isErr, dry := newAckRig(t, newSelectDB()).call(t, "select", sel("alarm_class", "defaultAlarmClass"))
	if isErr || strings.Join(selected(t, dry), ",") != u2 {
		t.Errorf("class: %s", text)
	}
	text, isErr, dry = newAckRig(t, newSelectDB()).call(t, "select", sel("source", "slot:/Services", "before", "2026-11-15T12:00:00Z"))
	if isErr || strings.Join(selected(t, dry), ",") != u1 {
		t.Errorf("before: %s", text)
	}
}

func TestAckSelectTruncatesAtMax(t *testing.T) {
	_, isErr, dry := newAckRig(t, newSelectDB()).call(t, "select", sel("source", "slot:/Services", "max", pyjson.IntLiteral("1")))
	s := get(get(dry, "plan").(*pyjson.Object), "selection").(*pyjson.Object)
	if isErr || strings.Join(selected(t, dry), ",") != u2 || get(s, "matched") != 2 || get(s, "truncated") != true || get(s, "max") != 1 {
		t.Errorf("selection %s", pyjson.Repr(s))
	}
}

func TestAckSelectWithNoMatchIsADryRunWithoutAToken(t *testing.T) {
	db := newSelectDB()
	text, isErr, dry := newAckRig(t, db).call(t, "select", sel("alarm_class", "Nope"))
	if isErr {
		t.Fatal(text)
	}
	if _, has := dry.Get("confirmation_token"); has || get(dry, "dry_run") != true || len(db.calls) != 0 {
		t.Errorf("reply %s", pyjson.Repr(dry))
	}
	s := get(dry, "selection").(*pyjson.Object)
	if pyjson.Repr(get(s, "records")) != "[]" || get(s, "matched") != 0 || !strings.Contains(text, "nothing to acknowledge") {
		t.Errorf("selection %s; text %s", pyjson.Repr(s), text)
	}
}

func TestAckSelectTokenExecutesExactlyTheDryRunUuids(t *testing.T) {
	db := newSelectDB()
	r := newAckRig(t, db)
	args := []any{"select", sel("source", "slot:/Services")}
	text, isErr, dry := r.call(t, args...)
	if isErr {
		t.Fatal(text)
	}
	db.raise(u6, "slot:/Services/A", "Critical") // matches the selection, after the dry run
	text, isErr, out := r.call(t, append(args, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))...)
	if isErr {
		t.Fatalf("execute: %s", text)
	}
	if got := strings.Join(db.calls, "\n"); got != "alarm ackAlarms {'ids': ['"+u2+"', '"+u1+"']}" {
		t.Errorf("calls %q", got)
	}
	if get(out, "verdict") != "verified" || db.state[u6] != "unacked" {
		t.Errorf("reply %s; u6 %s", pyjson.Repr(out), db.state[u6])
	}
	if !strings.Contains(pyjson.Repr(get(out, "requested")), "'select': {'source': 'slot:/Services'}") {
		t.Errorf("requested %s", pyjson.Repr(get(out, "requested")))
	}
}

func TestAckSelectExecuteWithoutItsDryRunIsRefused(t *testing.T) {
	db := newSelectDB()
	text, isErr, _ := newAckRig(t, db).call(t, "select", sel("source", "slot:/Services"), "dry_run", false,
		"confirmation_token", "1.a.b")
	if !isErr || !strings.Contains(text, "no dry run of this selection") || len(db.calls) != 0 || len(db.ords) != 0 {
		t.Errorf("%v %s (ords %q)", isErr, text, db.ords)
	}
}

func TestAckSelectRefusals(t *testing.T) {
	for _, c := range []struct {
		args []any
		want string
	}{
		{[]any{"uuids", []any{u1}, "select", sel("source", "slot:/Services")}, "give uuids or select, not both"},
		{[]any{"notes", "x"}, "give uuids or select"},
		{[]any{"select", sel()}, "select needs at least one of source, alarm_class or before"},
		{[]any{"select", sel("alarm_class", "x' or alarmClass != 'y")}, "alarm_class must be an alarm class name"},
		{[]any{"select", sel("source", "slot:/Services/A$27")}, "source must not hold a $ escape"},
		{[]any{"select", sel("source", "slot:/Services' or source like '*")}, "select.source must be a slot ORD"},
		{[]any{"select", sel("source", "h:12")}, "select.source must be a slot ORD"},
		{[]any{"select", sel("before", "yesterday")}, "select.before must be a BAbsTime"},
		{[]any{"select", sel("source", "slot:/Drivers")}, "outside the write scope: " + u4},
	} {
		db := newSelectDB()
		text, isErr, _ := newAckRig(t, db).call(t, c.args...)
		if !isErr || !strings.Contains(text, c.want) || len(db.calls) != 0 {
			t.Errorf("%v: %v %s", pyjson.Repr(c.args), isErr, text)
		}
		if c.want != "outside the write scope: "+u4 && len(db.ords) != 0 {
			t.Errorf("%v reached the station: %q", pyjson.Repr(c.args), db.ords)
		}
	}
}

// The schema refuses a select key it does not declare and a max outside 1..100, before the tool.
func TestAckSelectSchemaRefusals(t *testing.T) {
	for _, c := range []struct {
		select_ *pyjson.Object
		want    string
	}{
		{sel("source", "slot:/Services", "max", pyjson.IntLiteral("0")), "argument select.max must be >= 1"},
		{sel("source", "slot:/Services", "max", pyjson.IntLiteral("101")), "argument select.max must be <= 100"},
		{sel("source", "slot:/Services", "other", "x"), "select.other"},
		{sel("source", 5), "argument select.source must be string"},
	} {
		db := newSelectDB()
		reply := newAckRig(t, db).srv.Dispatch(obj("id", 1, "method", "tools/call",
			"params", obj("name", "n4_ack_alarm", "arguments", obj("select", c.select_))))
		e, _ := reply.Get("error")
		if e == nil || !strings.Contains(pyjson.Repr(e), c.want) || len(db.ords) != 0 {
			t.Errorf("%s: %s", pyjson.Repr(c.select_), pyjson.Repr(reply))
		}
	}
}

// The uuids path is unchanged: the same reads, plan keys and ops as before select existed.
func TestAckUuidsPathIsUnchangedBySelect(t *testing.T) {
	db := newAlarmDB()
	_, isErr, dry := newAckRig(t, db).call(t, "uuids", []any{u1, u2})
	p := get(dry, "plan").(*pyjson.Object)
	if isErr || strings.Join(p.Keys(), ",") != "tool,ops,inverse,notes,station" ||
		strings.Join(db.ords, "\n") != existsOrd+"\n"+unackedOrd {
		t.Errorf("plan %s; ords %q", pyjson.Repr(p), db.ords)
	}
	_, out := newAckRig(t, newAlarmDB()).ackBoth(t, "uuids", []any{u1})
	if pyjson.Repr(get(out, "requested")) != "{'uuids': ['"+u1+"']}" {
		t.Errorf("requested %s", pyjson.Repr(get(out, "requested")))
	}
}
