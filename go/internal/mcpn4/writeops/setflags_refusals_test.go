package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// flagsRigRefusal is the refusal of a plan; a plan that is not refused fails the test.
func flagsRigRefusal(t *testing.T, c plan.Client, slot, flags string) string {
	t.Helper()
	p, err := writeops.SetFlags.Plan(c, flagsRigArgs(slot, flags))
	if err == nil {
		t.Fatalf("slot %q flags %q: planned %s, want a refusal", slot, flags, js(t, p.Ops))
	}
	return err.Error()
}

// flagsRigNoted is the typed rig holding the dynamic slot note with flags "ho".
func flagsRigNoted(t *testing.T) *typed {
	return newTyped(t, extNodes("note", obj("nm", "p", "t", "baja:String", "v", "x", "f", "ho")))
}

// A bad flags string is refused before anything is read from the station.
func TestSetFlagsRefusesAnUnknownOrRepeatedLetterBeforeReadingTheStation(t *testing.T) {
	unread := &typed{loads: loads{nodes: obj()}} // any load would fail with "no such component"
	got := flagsRigRefusal(t, unread, "note", "hq")
	if !strings.HasPrefix(got, "unknown flag letter 'q' in flags 'hq'; valid letters: r readonly, t transient, h hidden, ") ||
		!strings.HasSuffix(got, "4 user-defined-4") {
		t.Errorf("unknown letter: %s", got)
	}
	if got := flagsRigRefusal(t, unread, "note", "hh"); got != "flag letter 'h' is repeated in flags 'hh'; give each letter once" {
		t.Errorf("repeated letter: %s", got)
	}
	if len(unread.calls) != 0 {
		t.Errorf("the type registry was read: %v", unread.calls)
	}
}

func TestSetFlagsRefusesASlotNameThatIsNotOneSlotOfTheComponent(t *testing.T) {
	c := flagsRigNoted(t)
	for _, slot := range []string{"", "historyConfig/historyName", "a b"} {
		if got := flagsRigRefusal(t, c, slot, "h"); !strings.HasPrefix(got, "slot must be a plain slot name") {
			t.Errorf("slot %q: %s", slot, got)
		}
	}
}

func TestSetFlagsRefusesASlotTheComponentDoesNotHave(t *testing.T) {
	want := "slot 'nope' does not exist on " + ext + ": the station does not list it and its type " +
		"history:NumericIntervalHistoryExt does not declare it; check the name with n4_read_slots"
	if got := flagsRigRefusal(t, flagsRigNoted(t), "nope", "h"); got != want {
		t.Errorf("refusal\n got  %s\n want %s", got, want)
	}
}

// Nothing to do is a refusal, in any letter order, for a listed and for a declared set.
func TestSetFlagsRefusesTheSetTheSlotAlreadyHas(t *testing.T) {
	c := flagsRigNoted(t)
	want := "slot 'note' on " + ext + ` already has flags "ho" (h hidden, o operator): nothing to do`
	for _, flags := range []string{"ho", "oh"} {
		if got := flagsRigRefusal(t, c, "note", flags); got != want {
			t.Errorf("flags %q\n got  %s\n want %s", flags, got, want)
		}
	}
	want = "slot 'enabled' on " + ext + ` already has flags "o" (o operator): nothing to do`
	if got := flagsRigRefusal(t, c, "enabled", "o"); got != want { // omitted: the declared flags
		t.Errorf("declared\n got  %s\n want %s", got, want)
	}
	want = "slot 'historyName' on " + ext + ` already has flags "" (none): nothing to do`
	if got := flagsRigRefusal(t, c, "historyName", ""); got != want {
		t.Errorf("no flags\n got  %s\n want %s", got, want)
	}
}

// Without "f" on the loaded slot and without its type contract the slot may be a frozen slot at
// declared flags nobody can read: no faithful inverse exists, so nothing is planned.
func TestSetFlagsRefusesWhenTheCurrentFlagsCannotBeDetermined(t *testing.T) {
	const tail = "; it may be a frozen slot at the flags its type declares, and without them no faithful inverse " +
		"exists, so nothing is planned"
	head := func(slot string) string {
		return "cannot determine the current flags of slot '" + slot + "' on " + ext + ": the station sends no flags for it and "
	}
	plain := obj("nm", "p", "t", "baja:Boolean", "v", "true")
	for _, tc := range []struct {
		name string
		c    plan.Client
		slot string
		want string
	}{
		{"a client without the registry channel", &loads{nodes: obj(ext, extNodes())}, "active",
			head("active") + "this client cannot read the station's type definitions (registry channel)" + tail},
		{"a slot the station omits, same client", &loads{nodes: obj(ext, extNodes())}, "enabled",
			head("enabled") + "this client cannot read the station's type definitions (registry channel)" + tail},
		{"a type the station does not describe", &typed{loads: loads{nodes: obj(ext, obj("", obj("h", "42", "t", "acme:Widget"),
			"x", plain))}, types: obj()}, "x", head("x") + "the station did not describe type 'acme:Widget'" + tail},
		{"a component without a type", &typed{loads: loads{nodes: obj(ext, obj("", obj("h", "42"), "x", plain))},
			types: obj()}, "x", head("x") + "the component has no type" + tail},
	} {
		if got := flagsRigRefusal(t, tc.c, tc.slot, "h"); got != tc.want {
			t.Errorf("%s\n got  %s\n want %s", tc.name, got, tc.want)
		}
	}
}

// Flags the station sends on the slot are the current ones whatever the contract says: the
// registry is not needed and not read.
func TestSetFlagsNeedsNoContractWhenTheSlotCarriesItsFlags(t *testing.T) {
	c := &loads{nodes: obj(ext, extNodes("note", obj("nm", "p", "t", "baja:String", "v", "x", "f", "h")))}
	p := flagsRigPlan(t, c, "note", "")
	if got := js(t, p.Inverse[0]); got != `{"nm": "f", "h": "42", "n": "note", "f": "h"}` {
		t.Errorf("inverse = %s", got)
	}
	bad := &loads{nodes: obj(ext, extNodes("note", obj("nm", "p", "t", "baja:String", "v", "x", "f", pyjson.IntLiteral("4"))))}
	if got := flagsRigRefusal(t, bad, "note", "h"); !strings.Contains(got, "the station sent flags that are not a string") {
		t.Errorf("flags not a string: %s", got)
	}
}
