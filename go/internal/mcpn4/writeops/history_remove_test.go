package writeops_test

// n4_remove_history through the write engine: the guards, the one deleteHistories call, the
// read-back and the journal.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const ack = "acknowledge_data_loss"

func TestRemoveHistoryPlansOneDeleteCallAndNoInverse(t *testing.T) {
	db := newHistoryDB()
	text, isErr, dry := remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if isErr {
		t.Fatal(text)
	}
	p := get(dry, "plan").(*pyjson.Object)
	if got := pyjson.Repr(get(p, "ops")); got != "[{'channel': 'history', 'key': 'deleteHistories', 'body': ['history:"+hTemp+"']}]" {
		t.Errorf("ops %s", got)
	}
	notes := pyjson.Repr(get(p, "notes"))
	if got := pyjson.Repr(get(p, "inverse")); got != "[]" || !strings.Contains(notes, "no inverse") ||
		!strings.Contains(notes, "records are gone") || !strings.Contains(notes, "slot:/Services/A/hist") {
		t.Errorf("inverse %s, notes %s", got, notes)
	}
	info := get(dry, "history_removal").(*pyjson.Object)
	if pyjson.Repr(get(info, "history_id")) != "'"+hTemp+"'" || get(info, "record_count") != 42 ||
		pyjson.Repr(get(info, "extensions")) != "[{'ext_ord': 'station:|slot:/Services/A/hist', 'enabled': False}]" {
		t.Errorf("history_removal %s", pyjson.Repr(info))
	}
	if len(db.calls) != 0 || len(db.records) != 4 {
		t.Errorf("a dry run reached the station: calls %q", db.calls)
	}
}

func TestRemoveHistorySendsTheDeleteAndReadsItGone(t *testing.T) {
	db := newHistoryDB()
	r := histRig(t, db)
	_, out := removeBoth(t, r, "history_id", hTemp, ack, true)
	if got := strings.Join(db.calls, "\n"); got != "history deleteHistories ['history:"+hTemp+"']" {
		t.Errorf("calls %q", got)
	}
	if get(out, "verdict") != "verified" || pyjson.Repr(get(out, "inverse")) != "[]" ||
		pyjson.Repr(get(out, "observed")) != "{'history_id': '"+hTemp+"', 'exists': False, 'record_count': None}" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	data, _ := os.ReadFile(filepath.Join(r.dir, "journal.jsonl"))
	if !strings.Contains(string(data), `"key": "deleteHistories"`) && !strings.Contains(string(data), `"key":"deleteHistories"`) {
		t.Errorf("journal %s", data)
	}
}

func TestRemoveHistoryIsAMismatchWhenTheHistoryStays(t *testing.T) {
	db := newHistoryDB()
	db.noDelete = true
	_, out := removeBoth(t, histRig(t, db), "history_id", hTemp, ack, true)
	if get(out, "verdict") != "mismatch" ||
		pyjson.Repr(get(out, "observed")) != "{'history_id': '"+hTemp+"', 'exists': True, 'record_count': 42}" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
}

func TestRemoveHistoryRefusesBeforeAnyCall(t *testing.T) {
	for name, c := range map[string]struct {
		kv   []any
		want string
	}{
		"no acknowledgement":    {[]any{"history_id", hTemp}, "acknowledge_data_loss: true"},
		"false acknowledgement": {[]any{"history_id", hTemp, ack, false}, "acknowledge_data_loss: true"},
		"bare name":             {[]any{"history_id", "Temp", ack, true}, "history_id must be /<station>/<history name>"},
		"enabled extension": {[]any{"history_id", hLive, ack, true},
			"extension station:|slot:/Services/B/hist is enabled and still records into it; pass allow_enabled_extension: true"},
		"outside the scope": {[]any{"history_id", hOut, ack, true},
			"extension station:|slot:/Drivers/X/hist is outside the write scope"},
		"no extension": {[]any{"history_id", hOrphan, ack, true},
			"no history extension records into " + hOrphan + ": its slot path is unknown, so it is outside every write scope"},
	} {
		db := newHistoryDB()
		text, isErr, _ := remove(t, histRig(t, db), c.kv...)
		if !isErr || !strings.Contains(text, c.want) || len(db.calls) != 0 || len(db.records) != 4 {
			t.Errorf("%s: %v %q, calls %q", name, isErr, text, db.calls)
		}
	}
}

func TestRemoveHistoryRefusesWhenTheExtensionReadIsCut(t *testing.T) {
	db := newHistoryDB()
	db.cutExts = true
	text, isErr, _ := remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if !isErr || !strings.Contains(text, "extension read was cut") || len(db.calls) != 0 {
		t.Errorf("%v %q", isErr, text)
	}
}

func TestRemoveHistoryPlansAnEnabledExtensionOnlyWhenAllowed(t *testing.T) {
	db := newHistoryDB()
	text, isErr, dry := remove(t, histRig(t, db), "history_id", hLive, ack, true, "allow_enabled_extension", true)
	if isErr {
		t.Fatal(text)
	}
	if notes := pyjson.Repr(get(get(dry, "plan").(*pyjson.Object), "notes")); !strings.Contains(notes, "ENABLED") {
		t.Errorf("notes %s", notes)
	}
}

func TestRemoveHistoryPlansAHistoryWhoseCountDoesNotAnswer(t *testing.T) {
	db := newHistoryDB()
	text, isErr, dry := remove(t, histRig(t, db), "history_id", hGone, ack, true)
	if isErr {
		t.Fatal(text)
	}
	info := get(dry, "history_removal").(*pyjson.Object)
	if get(info, "record_count") != nil || !strings.Contains(pyjson.Repr(get(info, "count_note")), "did not answer") {
		t.Errorf("history_removal %s", pyjson.Repr(info))
	}
}

func TestRemoveHistoryTakesOnlyABooleanAcknowledgement(t *testing.T) {
	db := newHistoryDB()
	r := histRig(t, db)
	reply := r.srv.Dispatch(obj("id", 1, "method", "tools/call", "params",
		obj("name", "n4_remove_history", "arguments", obj("history_id", hTemp, ack, "yes"))))
	if got := pyjson.Repr(get(reply, "error")); !strings.Contains(got, "argument acknowledge_data_loss must be boolean") ||
		len(db.ords) != 0 {
		t.Errorf("%s, reads %q", got, db.ords)
	}
}

func TestRemoveHistoryRefusesAClientThatCannotCallAChannel(t *testing.T) {
	db := newHistoryDB()
	r := scopedRig(t, historyReads{db}, t.TempDir()+"/state", "station:|slot:/Services")
	text, isErr, _ := remove(t, r, "history_id", hTemp, ack, true)
	if !isErr || !strings.Contains(text, "cannot read histories or call the history channel") || len(db.ords) != 0 {
		t.Errorf("%v %q, reads %q", isErr, text, db.ords)
	}
}

// Only the station's 404 proves the history is gone: another error after the delete is unverified.
func TestRemoveHistoryIsUnverifiedWhenTheReadBackFailsOtherThanNotFound(t *testing.T) {
	db := newHistoryDB()
	db.errAfter = true
	_, out := removeBoth(t, histRig(t, db), "history_id", hTemp, ack, true)
	if get(out, "verdict") != "unverified" || !strings.Contains(pyjson.Repr(get(out, "observed")), "'exists': None") {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
}

// A delete the station refuses is unapplied only when the history still reads; a re-read that
// answers neither way leaves the batch in doubt, never "nothing applied".
func TestRemoveHistoryRefusedCallIsUnappliedOnlyWhenTheHistoryStillReads(t *testing.T) {
	for name, c := range map[string]struct {
		errAfter bool
		want     string
	}{"history still reads": {false, "refused"}, "re-read fails": {true, "in-doubt"}} {
		db := newHistoryDB()
		db.callErr = &box.Error{Kind: box.KindBox, Msg: "station error", Station: true}
		db.errAfter = c.errAfter
		r := histRig(t, db)
		_, _, dry := remove(t, r, "history_id", hTemp, ack, true)
		tok := get(dry, "confirmation_token")
		text, isErr, _ := remove(t, r, "history_id", hTemp, ack, true, "dry_run", false, "confirmation_token", tok)
		if !isErr || !strings.Contains(text, c.want) {
			t.Errorf("%s: reply lacks %q: %s", name, c.want, text)
		}
	}
}
