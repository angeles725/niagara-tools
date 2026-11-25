package writeops_test

// n4_remove_history paths the first tests left unproven (#1073).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A count that answers a table with no whole number is planned with a note, not refused.
func TestRemoveHistoryNotesACountWithNoWholeNumber(t *testing.T) {
	db := newHistoryDB()
	db.countBody = "count(toString)\nmany\n"
	text, isErr, dry := remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if isErr {
		t.Fatal(text)
	}
	info := get(dry, "history_removal").(*pyjson.Object)
	if get(info, "record_count") != nil ||
		pyjson.Repr(get(info, "count_note")) != "'the record count answered no whole number'" {
		t.Errorf("history_removal %s", pyjson.Repr(info))
	}
}

// An auth failure on the count read is an error of the call, never a note of the plan.
func TestRemoveHistoryCountAuthFailureIsAnError(t *testing.T) {
	db := newHistoryDB()
	db.countErr = &box.Error{Kind: box.KindAuth, Msg: "HTTP 401 from station"}
	text, isErr, dry := remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if !isErr || (dry != nil && get(dry, "confirmation_token") != nil) || !strings.Contains(text, "401") || len(db.calls) != 0 {
		t.Errorf("%v %q", isErr, text)
	}
}

// An extension row whose slot is not a plain slot path cannot be scope-checked: refused.
func TestRemoveHistoryRefusesAnExtensionSlotThatIsNoSlotPath(t *testing.T) {
	db := newHistoryDB()
	db.exts = append(db.exts, "h:12,false,"+hTemp)
	text, isErr, _ := remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if !isErr || !strings.Contains(text, "extension slot 'h:12' is not a plain slot path") || len(db.calls) != 0 {
		t.Errorf("%v %q", isErr, text)
	}
}

// Several extensions recording into one history: each is listed, and one outside the scope refuses.
func TestRemoveHistoryChecksEveryExtensionOfTheHistory(t *testing.T) {
	db := newHistoryDB()
	db.exts = append(db.exts, "slot:/Services/D/hist,false,"+hTemp)
	text, isErr, dry := remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if isErr {
		t.Fatal(text)
	}
	exts := pyjson.Repr(get(get(dry, "history_removal").(*pyjson.Object), "extensions"))
	for _, want := range []string{"slot:/Services/A/hist", "slot:/Services/D/hist"} {
		if !strings.Contains(exts, want) {
			t.Errorf("extensions %s lack %s", exts, want)
		}
	}
	db = newHistoryDB()
	db.exts = append(db.exts, "slot:/Drivers/Y/hist,false,"+hTemp)
	text, isErr, _ = remove(t, histRig(t, db), "history_id", hTemp, ack, true)
	if !isErr || !strings.Contains(text, "extension station:|slot:/Drivers/Y/hist is outside the write scope") {
		t.Errorf("an outside second extension: %v %q", isErr, text)
	}
}

// journalOp is one channel op as the journal intent records it.
type journalOp struct {
	Channel string   `json:"channel"`
	Name    string   `json:"key"`
	Body    []string `json:"body"`
}

// The journal intent holds the one channel op exactly: channel, key and body.
func TestRemoveHistoryJournalsTheExactChannelOp(t *testing.T) {
	db := newHistoryDB()
	r := histRig(t, db)
	removeBoth(t, r, "history_id", hTemp, ack, true)
	data, err := os.ReadFile(filepath.Join(r.dir, "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var ops []journalOp
	found := false
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var entry struct {
			Phase string      `json:"phase"`
			Ops   []journalOp `json:"ops"`
		}
		if json.Unmarshal([]byte(line), &entry) == nil && entry.Phase == "intent" {
			ops, found = entry.Ops, true
			break
		}
	}
	if !found || len(ops) != 1 || ops[0].Channel != "history" || ops[0].Name != "deleteHistories" ||
		strings.Join(ops[0].Body, ",") != "history:"+hTemp {
		t.Errorf("journal intent ops %+v\n%s", ops, data)
	}
}
