package blackbox

// Verifier probes for retro 2026-11-16 row M15 (PR #738): what the remove token must still bind.

import (
	"strings"
	"testing"
)

// scheduleContracts declare a readonly struct holding configuration, as BDateSchedule declares
// yearSchedule (Flags.READONLY: the struct cannot be replaced, its year is edited in place).
var scheduleContracts = map[string]any{
	"schedule:DailySchedule": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "days", "v": map[string]any{"nm": "p", "t": "schedule:DateSchedule"}}},
		"p": "baja:Component"},
	"schedule:DateSchedule": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "yearSchedule", "f": "r", "v": map[string]any{"nm": "p", "t": "schedule:YearSchedule"}}},
		"p": "baja:Component"},
}

func integer(v string) map[string]any { return map[string]any{"nm": "p", "t": "baja:Integer", "v": v} }

// A configured value below a readonly (not transient) struct still binds the token: live
// Prueba_4_15 2026-11-16, Daily days/yearSchedule/year 2027 -> 2028 after the dry run executed.
func TestARemoveTokenIsRefusedWhenConfigBelowAReadonlyStructChanged(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(scheduleContracts)
	daily := d.Add("3", "Daily", "schedule:DailySchedule")
	days := d.Add(daily.Handle, "days", "schedule:DateSchedule")
	year := d.Add(days.Handle, "yearSchedule", "schedule:YearSchedule")
	d.Set(year.Handle, "year", integer("2027"))
	dry := d.Srv.Dry("n4_remove_component", removeArgs(daily.Name))
	d.Set(year.Handle, "year", integer("2028"))
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(daily.Name), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q (a configured year changed after the dry run)", err)
	}
}

// Flagging a slot transient after the dry run (and changing it) cannot carry a stale token.
func TestARemoveTokenIsRefusedWhenASlotTurnsTransientAfterTheDryRun(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	d.Set(ext, "note", relTime("1"))
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	if _, err := d.sync(map[string]any{"nm": "f", "h": ext, "n": "note", "f": "t"}); err != nil {
		t.Fatal(err)
	}
	d.Set(ext, "note", relTime("2"))
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
}

// Clearing a transient flag after the dry run (and changing the value) is refused too.
func TestARemoveTokenIsRefusedWhenATransientFlagIsCleared(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	d.Set(ext, "note", relTime("1"))
	if _, err := d.sync(map[string]any{"nm": "f", "h": ext, "n": "note", "f": "t"}); err != nil {
		t.Fatal(err)
	}
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	if _, err := d.sync(map[string]any{"nm": "f", "h": ext, "n": "note", "f": ""}); err != nil {
		t.Fatal(err)
	}
	d.Set(ext, "note", relTime("2"))
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
}

// A remove whose subtree holds no runtime slot shows and hashes the plan exactly as before.
func TestARemovePlanWithoutRuntimeSlotsHasNoRuntimeNote(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(alarmContracts)
	amps := d.Add("3", "Plain", "baja:Folder")
	d.Set(amps.Handle, "timeDelay", relTime("5000"))
	dry := d.Srv.Dry("n4_remove_component", removeArgs(amps.Name))
	if got := noteWith(planOf(dry)["notes"], "runtime"); len(got) != 0 {
		t.Errorf("runtime note on a plan without runtime slots: %v", got)
	}
}
