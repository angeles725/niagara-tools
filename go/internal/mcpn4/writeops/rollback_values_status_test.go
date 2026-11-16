package writeops

import (
	"strings"
	"testing"
)

// The rollback guard (unchangedSince) restores a Status slot only while it still holds what the
// batch wrote. StatusEnum and StatusString (an enum or string schedule's defaultOutput) compare
// their value text and status, as StatusBoolean and StatusNumeric compare theirs.

const guardOrd = "station:|slot:/Folder/Sch"

// guardCase journals one set of slot "defaultOutput" to body (type typ, value v) and loads the
// station holding now (nil: the value child omitted, at its default) with status, the slot of
// type loaded[0] when given (else typ).
func guardCase(t *testing.T, typ, v string, now any, status string, loaded ...string) error {
	t.Helper()
	kid := func(n string, val any) any {
		return obj("nm", "p", "n", n, "v", val)
	}
	body := obj("nm", "p", "t", typ, "s", []any{kid("value", v), kid("status", "0")})
	op := obj("nm", "s", "h", "5", "n", "defaultOutput", "b", body)
	view := obj("batch_id", "b1", "ops", []any{op})
	targets := obj(guardOrd, "5")
	slotType := typ
	if len(loaded) > 0 {
		slotType = loaded[0]
	}
	nodes := obj("", obj("t", "schedule:StringSchedule", "h", "5"), "defaultOutput", obj("t", slotType))
	if now != nil {
		nodes.Set("defaultOutput/value", obj("n", "value", "v", now))
	}
	nodes.Set("defaultOutput/status", obj("n", "status", "v", status))
	return unchangedSince(scanRoot{nodes}, view, []any{op}, targets)
}

func TestTheRollbackGuardPassesAnUnchangedEnumOrStringStatus(t *testing.T) {
	for _, c := range []struct {
		typ, wrote string
		now        any
	}{
		{"baja:StatusString", "eco", "eco"},
		{"baja:StatusEnum", "3", "3"},
		{"baja:StatusEnum", "2", "2@{a=1,b=2}"}, // a DynamicEnum loads as ordinal@range
		{"baja:StatusEnum", "0", nil},           // the ordinal 0 is the omitted default
		{"baja:StatusString", "", nil},          // so is the empty string
	} {
		if err := guardCase(t, c.typ, c.wrote, c.now, "0"); err != nil {
			t.Errorf("%s %q (station %v): %v", c.typ, c.wrote, c.now, err)
		}
	}
}

func TestTheRollbackGuardRefusesAChangedEnumOrStringStatus(t *testing.T) {
	for _, c := range []struct {
		typ, wrote, now, status, want string
	}{
		{"baja:StatusString", "eco", "comfort", "0", `"value":"comfort"`},
		{"baja:StatusEnum", "3", "4", "0", `"value":"4"`},
		{"baja:StatusString", "eco", "eco", "40", `"status":"40"`},
	} {
		err := guardCase(t, c.typ, c.wrote, c.now, c.status)
		if err == nil || !strings.Contains(err.Error(), "changed since batch b1") || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %q -> %q: %v", c.typ, c.wrote, c.now, err)
		}
	}
	// The slot now holds another Status type: it changed, whatever its text.
	err := guardCase(t, "baja:StatusEnum", "3", "3", "0", "baja:StatusString")
	if err == nil || !strings.Contains(err.Error(), "the station now holds null") {
		t.Errorf("type changed: %v", err)
	}
}

// Pins: the guard of StatusBoolean and StatusNumeric is unchanged.
func TestTheRollbackGuardOfBooleanAndNumericStatusIsUnchanged(t *testing.T) {
	for _, c := range []struct {
		typ, wrote, now string
		refused         bool
	}{
		{"baja:StatusBoolean", "true", "true", false},
		{"baja:StatusBoolean", "true", "false", true},
		{"baja:StatusNumeric", "21.5", "21.5", false},
		{"baja:StatusNumeric", "21.5", "19.0", true},
	} {
		err := guardCase(t, c.typ, c.wrote, c.now, "0")
		if refused := err != nil && strings.Contains(err.Error(), "changed since batch b1"); refused != c.refused {
			t.Errorf("%s %s -> %s: %v", c.typ, c.wrote, c.now, err)
		}
	}
}
