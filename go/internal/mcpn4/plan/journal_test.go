package plan_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The Go twins of the state file, journal and audit tests of test_tools_write (go_twin("plan",
// ...)), driven through n4_set_slot.

func TestAStateFileLoosenedWhileRunningFailsBeforeAnySend(t *testing.T) {
	r := newRig(t)
	r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs())) // creates both files 0600
	for _, name := range []string{"journal.jsonl", "audit.jsonl"} {
		tok := r.dry(t, "n4_set_slot", setArgs())
		path := r.dir + "/" + name
		os.Chmod(path, 0o644)
		sent, executed := len(r.st.sent), r.state.WritesExecuted
		if text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), tok); !isErr || !strings.Contains(text, "chmod 600") {
			t.Fatalf("%s: %v %q", name, isErr, text)
		}
		if len(r.st.sent) != sent || r.state.WritesExecuted != executed {
			t.Fatalf("%s: a loose file let an op through", name)
		}
		os.Chmod(path, 0o600)
		if _, _, out := r.execute(t, "n4_set_slot", setArgs(), tok); get(out, "verdict") != "verified" {
			t.Fatalf("%s: the token was spent: %s", name, pyjson.Repr(out))
		}
	}
}

// failing wraps a state file: Append fails for entries whose phase is phase ("" fails all).
type failing struct {
	plan.StateFile
	phase string
}

func (f failing) Append(entry *pyjson.Object) error {
	if p, _ := entry.Get("phase"); f.phase == "" || p == f.phase {
		return errors.New("disk full")
	}
	return f.StateFile.Append(entry)
}

func TestAnIntentThatCannotBeWrittenSendsNothingAndKeepsTheTokenSpent(t *testing.T) {
	r := newRig(t)
	tok := r.dry(t, "n4_set_slot", setArgs())
	real := r.state.Journal
	r.state.Journal = failing{real, ""}
	if text, _, _ := r.execute(t, "n4_set_slot", setArgs(), tok); !strings.Contains(text, "nothing was sent") {
		t.Fatal(text)
	}
	if len(r.st.sent) != 0 || r.state.WritesExecuted != 0 {
		t.Fatal("an unjournaled intent was sent")
	}
	r.state.Journal = real
	if text, _, _ := r.execute(t, "n4_set_slot", setArgs(), tok); !strings.Contains(text, "already used") || len(r.st.sent) != 0 {
		t.Fatal(text)
	}
}

func TestASendExceptionAfterTheTokenWasSpentLeavesAnInDoubtBatch(t *testing.T) {
	r := newRig(t)
	tok := r.dry(t, "n4_set_slot", setArgs())
	r.st.sendErr = &box.Error{Kind: box.KindBox, Msg: "HTTP 500 from station"}
	text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), tok)
	lines := r.lines(t, "journal.jsonl")
	id := get(lines[0], "batch_id").(string)
	if !isErr || !strings.Contains(text, "in-doubt") || !strings.Contains(text, id) || len(lines) != 1 {
		t.Fatalf("%v %q %d journal lines", isErr, text, len(lines))
	}
	view, err := journal.New(r.dir).Read(id)
	if err != nil || get(view, "state") != "in-doubt" {
		t.Fatal(pyjson.Repr(view), err)
	}
	r.st.sendErr = nil
	if text, _, _ := r.execute(t, "n4_set_slot", setArgs(), tok); !strings.Contains(text, "already used") {
		t.Fatal(text)
	}
	audit := r.lines(t, "audit.jsonl")
	if get(audit[len(audit)-2], "batch_id") != id || get(audit[len(audit)-2], "outcome") != "error" {
		t.Fatal(pyjson.Repr(audit[len(audit)-2]))
	}
}

func TestAResultThatCannotBeWrittenIsAWarningWithTheInverse(t *testing.T) {
	r := newRig(t)
	r.state.Journal = failing{r.state.Journal, "result"}
	_, _, out := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
	warnings := pyjson.Repr(get(out, "warnings"))
	if get(out, "verdict") != "verified" || !strings.Contains(warnings, "in-doubt") ||
		!strings.Contains(warnings, `"v":"1.0"`) {
		t.Fatal(pyjson.Repr(out))
	}
}

func TestAReadBackBoxErrorIsFailedWithTheBatchId(t *testing.T) {
	r := newRig(t)
	tok := r.dry(t, "n4_set_slot", setArgs())
	r.st.loadErr, r.st.failLoadsAfter = &box.Error{Kind: box.KindBox, Msg: "no load event for x after 6 polls"}, true
	_, _, out := r.execute(t, "n4_set_slot", setArgs(), tok)
	lines := r.lines(t, "journal.jsonl")
	if get(out, "verdict") != "failed" || get(out, "readback_error") != "no load event for x after 6 polls" ||
		!batchIDPattern.MatchString(get(out, "batch_id").(string)) || get(lines[len(lines)-1], "verdict") != "failed" {
		t.Fatal(pyjson.Repr(out))
	}
}

func TestAnUnwritableAuditLogIsAWarningNotAnError(t *testing.T) {
	r := newRig(t)
	r.state.Audit = failing{r.state.Audit, ""}
	_, _, out := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
	if get(out, "verdict") != "verified" || pyjson.Repr(get(out, "warnings")) != "['the audit log could not be written']" {
		t.Fatal(pyjson.Repr(out))
	}
}
