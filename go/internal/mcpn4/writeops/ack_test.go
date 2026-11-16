package writeops_test

// n4_ack_alarm through the write engine: the pre-check, the one ackAlarms call, the read-back
// and the journal.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const (
	existsOrd  = "alarm:|bql:select uuid, ackState, user where (uuid = '" + u1 + "' or uuid = '" + u2 + "')|view:file:ITableToCsv"
	unackedOrd = "alarm:|bql:select uuid, ackState, user where ackState = 'unacked' and (uuid = '" + u1 + "' or uuid = '" + u2 + "')|view:file:ITableToCsv"
)

func TestAckAlarmPlansOneAckAlarmsCallAndNoInverse(t *testing.T) {
	db := newAlarmDB()
	text, isErr, dry := newAckRig(t, db).call(t, "uuids", []any{u1, strings.ToUpper(u2)}, "notes", "checked on site")
	if isErr {
		t.Fatal(text)
	}
	p := get(dry, "plan").(*pyjson.Object)
	if got := pyjson.Repr(get(p, "ops")); got != "[{'channel': 'alarm', 'key': 'ackAlarms', 'body': {'ids': ['"+u1+"', '"+u2+"'], 'notes': 'checked on site'}}]" {
		t.Errorf("ops %s", got)
	}
	if got := pyjson.Repr(get(p, "inverse")); got != "[]" || !strings.Contains(pyjson.Repr(get(p, "notes")), "cannot be undone") {
		t.Errorf("inverse %s, notes %s", got, pyjson.Repr(get(p, "notes")))
	}
	if strings.Join(db.ords, "\n") != existsOrd+"\n"+unackedOrd || len(db.calls) != 0 {
		t.Errorf("ords %q, calls %q", db.ords, db.calls)
	}
}

func TestAckAlarmSendsTheAckAndReadsItBack(t *testing.T) {
	db := newAlarmDB()
	r := newAckRig(t, db)
	_, out := r.ackBoth(t, "uuids", []any{u1, u2})
	if got := strings.Join(db.calls, "\n"); got != "alarm ackAlarms {'ids': ['"+u1+"', '"+u2+"']}" {
		t.Errorf("calls %q", got)
	}
	if get(out, "verdict") != "verified" || pyjson.Repr(get(out, "inverse")) != "[]" || pyjson.Repr(get(out, "accepted")) != "[]" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	if got := pyjson.Repr(get(out, "observed")); got != "{'records': [{'uuid': '"+u1+"', 'ack_state': 'AckPending', 'user': 'admin'}, "+
		"{'uuid': '"+u2+"', 'ack_state': 'AckPending', 'user': 'admin'}], 'still_unacked': [], 'permission_failed_classes': []}" {
		t.Errorf("observed %s", got)
	}
	data, _ := os.ReadFile(filepath.Join(r.dir, "journal.jsonl"))
	if !strings.Contains(string(data), `"ops": [{"channel": "alarm", "key": "ackAlarms", "body": {"ids": ["`+u1) &&
		!strings.Contains(string(data), `"ops":[{"channel":"alarm","key":"ackAlarms","body":{"ids":["`+u1) {
		t.Errorf("journal intent %s", data)
	}
}

func TestAckAlarmIsAMismatchWhenTheStationNamesClassesOrRecordsStayUnacked(t *testing.T) {
	db := newAlarmDB()
	db.failClasses, db.noAck = []any{"Critical"}, true
	_, out := newAckRig(t, db).ackBoth(t, "uuids", []any{u1, u2})
	obs := get(out, "observed").(*pyjson.Object)
	if get(out, "verdict") != "mismatch" || pyjson.Repr(get(obs, "permission_failed_classes")) != "['Critical']" ||
		pyjson.Repr(get(obs, "still_unacked")) != "['"+u1+"', '"+u2+"']" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	db = newAlarmDB()
	db.failClasses = []any{"Critical"}
	if _, out = newAckRig(t, db).ackBoth(t, "uuids", []any{u1}); get(out, "verdict") != "mismatch" {
		t.Errorf("a non-empty reply with every record acked: %s", pyjson.Repr(out))
	}
	db = newAlarmDB()
	db.noAck, db.user[u1] = true, "admin" // a user left from an earlier ack: only the state tells
	if _, out = newAckRig(t, db).ackBoth(t, "uuids", []any{u1}); get(out, "verdict") != "mismatch" {
		t.Errorf("a record still unacked: %s", pyjson.Repr(out))
	}
}

func TestAckAlarmRefusesBeforeAnyCall(t *testing.T) {
	missing := "aa110470-0000-4000-8000-0000000000ff"
	for _, c := range []struct {
		uuids []any
		want  string
	}{
		{[]any{u1, missing, u3}, "no alarm record has uuid " + missing + "; already acknowledged or ack pending: " + u3},
		{[]any{}, "uuids must hold 1 to 100"},
		{[]any{"aa11' or uuid != '"}, "is not a uuid"},
		{[]any{u1, u1}, "is given twice"},
		{[]any{5}, "is not a uuid"},
	} {
		db := newAlarmDB()
		text, isErr, _ := newAckRig(t, db).call(t, "uuids", c.uuids)
		if !isErr || !strings.Contains(text, c.want) || len(db.calls) != 0 {
			t.Errorf("%v: %v %s (calls %q)", c.uuids, isErr, text, db.calls)
		}
	}
	many := make([]any, 101)
	for i := range many {
		many[i] = u1
	}
	if text, isErr, _ := newAckRig(t, newAlarmDB()).call(t, "uuids", many); !isErr || !strings.Contains(text, "uuids must hold 1 to 100") {
		t.Errorf("101 uuids: %s", text)
	}
	db := newAlarmDB()
	text, isErr, _ := newAckRig(t, readsOnly{db}).call(t, "uuids", []any{u1})
	if !isErr || !strings.Contains(text, `cannot call the BOX channel "alarm"`) || len(db.ords) != 0 {
		t.Errorf("no Call: %s (ords %q)", text, db.ords)
	}
}
