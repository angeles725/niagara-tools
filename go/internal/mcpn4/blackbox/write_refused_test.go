package blackbox

// An explicit station refusal with nothing applied is journaled as refused, not in-doubt (bench
// retro 2026-11-15 delta 4); anything else after the intent stays in-doubt.

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// refuseSSC makes the station answer every ssc call named sck with an error reply of text,
// after apply (nil: nothing) ran on it.
func refuseSSC(sck, text string, apply func(st *boxstation.Station, arg map[string]any)) boxstation.Faults {
	return boxstation.Faults{Refuse: func(st *boxstation.Station, key string, arg map[string]any) error {
		if key != sck {
			return nil
		}
		if apply != nil {
			apply(st, arg)
		}
		return errors.New(text)
	}}
}

// executeErr is a dry run then its execution, which must answer a tool error; it returns the
// error text and the batch id of the last journal line.
func executeErr(t *testing.T, r *Rig, tool string, args Args) (text string, batchID any) {
	t.Helper()
	plan := r.Srv.Dry(tool, args)
	text = r.Srv.Err(tool, Execute(args, plan["confirmation_token"]))
	lines := r.Lines("journal.jsonl")
	if len(lines) == 0 {
		t.Fatalf("nothing journaled: %s", text)
	}
	return text, lines[len(lines)-1]["batch_id"]
}

// lastAudit is the newest audit line.
func lastAudit(t *testing.T, r *Rig) map[string]any {
	t.Helper()
	lines := r.Lines("audit.jsonl")
	if len(lines) == 0 {
		t.Fatal("no audit line")
	}
	return lines[len(lines)-1]
}

func assertRefused(t *testing.T, r *Rig, text string, id any, station string) {
	t.Helper()
	if !strings.Contains(text, "refused") || !strings.Contains(text, "nothing was applied") ||
		!strings.Contains(text, station) || strings.Contains(text, "in-doubt") ||
		strings.ContainsAny(strings.ReplaceAll(text, "& ", ""), "&") {
		t.Errorf("reply = %q", text)
	}
	view := batchView(t, r, id)
	if view["state"] != "refused" || view["verdict"] != "refused" || view["refused"] != true ||
		view["station_error"] != station || !reflect.DeepEqual(view["inverse"], []any{}) {
		t.Errorf("journal view = %v", view)
	}
	if a := lastAudit(t, r); a["outcome"] != "refused" || a["batch_id"] != id || a["reason"] != text {
		t.Errorf("audit = %v", a)
	}
	rows := objects(Dig(r.Srv.OK("n4_list_batches", Args{}), "batches"))
	if len(rows) == 0 || rows[0]["batch_id"] != id || rows[0]["state"] != "refused" || rows[0]["verdict"] != "refused" {
		t.Errorf("n4_list_batches = %v", rows)
	}
	back := r.Srv.Err("n4_rollback", Args{"batch_id": id})
	if !strings.Contains(back, "nothing to roll back") || !strings.Contains(back, "refused") {
		t.Errorf("rollback = %q", back)
	}
}

func TestAStationRefusedLinkIsJournaledRefusedWithItsTextDecoded(t *testing.T) {
	r, _, tgt, args := linkRig(t)
	r.Station.SetFaults(refuseSSC("checkLinks", "ValueError: Could not find Source Slot &quot;out&quot; &amp; more", nil))
	text, id := executeErr(t, r, "n4_create_link", args)
	assertRefused(t, r, text, id, `ValueError: Could not find Source Slot "out" & more`)
	if hasChild(r, tgt.Handle, "Link") {
		t.Error("Tgt has a Link")
	}
}

func TestAStationRefusedCreateIsJournaledRefused(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetFaults(refuseSSC("syncTo", "ValueError: Illegal parent &lt;baja:Folder&gt;", nil))
	text, id := executeErr(t, r, "n4_create_component", createPump)
	assertRefused(t, r, text, id, "ValueError: Illegal parent <baja:Folder>")
	if len(r.Children()) != 0 {
		t.Errorf("children %v", r.Children())
	}
}

// The station answered an error but the read-back shows the component: in-doubt, as before.
func TestARefusalWithTheWriteAppliedStaysInDoubt(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetFaults(refuseSSC("syncTo", "ValueError: Slot &apos;Pump&apos; already exists",
		func(st *boxstation.Station, _ map[string]any) {
			st.Component(st.ByHandle("3"), "Pump", "kitControl:NumericConst", "")
		}))
	text, id := executeErr(t, r, "n4_create_component", createPump)
	if !strings.Contains(text, "is in-doubt") || !strings.Contains(text, "Slot 'Pump' already exists") {
		t.Errorf("reply = %q", text)
	}
	if view := batchView(t, r, id); view["state"] != "in-doubt" {
		t.Errorf("journal view = %v", view)
	}
	if a := lastAudit(t, r); a["outcome"] != "error" {
		t.Errorf("audit = %v", a)
	}
}

// A transport failure is no refusal: the station may have applied the batch.
func TestATransportFailureAfterTheIntentStaysInDoubt(t *testing.T) {
	r, _, _, args := linkRig(t)
	r.Station.SetHook(func(frame map[string]any) (int, []byte) {
		msgs, _ := frame["m"].([]any)
		for _, m := range msgs {
			if Dig(m, "b", "sck") == "checkLinks" {
				return 500, []byte("boom")
			}
		}
		return 0, nil
	})
	text, id := executeErr(t, r, "n4_create_link", args)
	if !strings.Contains(text, "is in-doubt") || strings.Contains(text, "refused") {
		t.Errorf("reply = %q", text)
	}
	if view := batchView(t, r, id); view["state"] != "in-doubt" {
		t.Errorf("journal view = %v", view)
	}
}
