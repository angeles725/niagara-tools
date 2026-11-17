package blackbox

// A remove's confirmation token survives a runtime slot changing between the dry run and the
// execute (retro 2026-11-16 row M15): an alarm extension's timeInCurrentState (declared readonly
// and transient, BAlarmSourceExt flags 3) ticks every second. A change of configuration or of
// structure in the removed subtree still invalidates the token, and the rollback still restores
// the configuration.

import (
	"strings"
	"testing"
)

// alarmContracts declare two slots of an alarm extension as BAlarmSourceExt does:
// timeInCurrentState "rt" (flags 3), timeDelay without flags (configuration).
var alarmContracts = map[string]any{
	"alarm:OutOfRangeAlarmExt": map[string]any{"c": []any{}, "p": "alarm:AlarmSourceExt"},
	"alarm:AlarmSourceExt": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "timeInCurrentState", "f": "rt", "v": map[string]any{"nm": "p", "t": "baja:RelTime"}},
		map[string]any{"st": "p", "n": "timeDelay", "v": map[string]any{"nm": "p", "t": "baja:RelTime"}}},
		"p": "baja:Component"},
}

func relTime(ms string) map[string]any {
	return map[string]any{"nm": "p", "t": "baja:RelTime", "v": ms}
}

// alarmed adds a Folder "Amps" holding an alarm extension "Alarm" (each by its own add, as the
// station takes them) and returns the folder's name and the extension's handle.
func (d *dRig) alarmed() (nn, ext string) {
	d.T.Helper()
	d.withContractsOf(alarmContracts)
	amps := d.Add("3", "Amps", "baja:Folder")
	alarm := d.Add(amps.Handle, "Alarm", "alarm:OutOfRangeAlarmExt")
	d.Set(alarm.Handle, "timeInCurrentState", relTime("1000"))
	d.Set(alarm.Handle, "timeDelay", relTime("5000"))
	return amps.Name, alarm.Handle
}

func TestARemoveTokenSurvivesARuntimeSlotTicking(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Set(ext, "timeInCurrentState", relTime("6000"))
	done := d.Srv.Call("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"]))
	if done.IsError {
		t.Fatalf("execute refused: %s", done.Text)
	}
	if d.node(nn) != nil {
		t.Fatalf("%s was not removed", nn)
	}
	if !strings.Contains(strings.Join(noteWith(planOf(dry)["notes"], "runtime"), ""), "confirmation") {
		t.Errorf("notes = %v", planOf(dry)["notes"])
	}
	back := d.rollback(done.Structured["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	if got := d.value(d.node(nn, "Alarm").Handle, "timeDelay"); got != "5000" {
		t.Errorf("restored timeDelay = %v, want 5000", got)
	}
}

func TestARemoveTokenIsRefusedWhenConfigurationChanged(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Set(ext, "timeDelay", relTime("9000"))
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
}

func TestARemoveTokenIsRefusedWhenAChildWasAdded(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.alarmed()
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Add(d.node(nn).Handle, "Extra", "")
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
	if d.node(nn) == nil {
		t.Errorf("%s was removed on a refused token", nn)
	}
}

// A dynamic slot flagged transient on the instance is never persisted: a runtime slot too.
func TestARemoveTokenSurvivesATransientDynamicSlotChanging(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	d.Set(ext, "lastSeen", relTime("1"))
	if _, err := d.sync(map[string]any{"nm": "f", "h": ext, "n": "lastSeen", "f": "t"}); err != nil {
		t.Fatal(err)
	}
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Set(ext, "lastSeen", relTime("2"))
	if done := d.Srv.Call("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); done.IsError {
		t.Fatalf("execute refused: %s", done.Text)
	}
}

// The dry-run plan marks each runtime slot it shows without a value; configuration keeps its value.
func TestARemovePlanMarksEachRuntimeSlot(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.alarmed()
	add := objects(planOf(d.Srv.Dry("n4_remove_component", removeArgs(nn)))["inverse"])[0]
	ext := byName(Dig(add, "b", "s"))["Alarm"]
	slots := byName(ext["s"])
	if tic := slots["timeInCurrentState"]; tic["runtime"] != true || tic["v"] != nil {
		t.Errorf("timeInCurrentState = %v", tic)
	}
	if delay := slots["timeDelay"]; delay["runtime"] != nil || delay["v"] != "5000" {
		t.Errorf("timeDelay = %v", delay)
	}
}

// transientCache gives the alarm extension a dynamic Folder "Cache" flagged transient on the
// instance, holding a child value "count": the whole subtree is runtime (row M15b).
func (d *dRig) transientCache(ext string) string {
	d.T.Helper()
	cache := d.Add(ext, "Cache", "baja:Folder")
	if _, err := d.sync(map[string]any{"nm": "f", "h": ext, "n": "Cache", "f": "t"}); err != nil {
		d.T.Fatal(err)
	}
	d.Set(cache.Handle, "count", relTime("1"))
	return cache.Handle
}

// A child value below a transient slot is runtime too: its change keeps the token.
func TestARemoveTokenSurvivesAChildValueBelowATransientSlot(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	cache := d.transientCache(ext)
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Set(cache, "count", relTime("2"))
	if done := d.Srv.Call("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); done.IsError {
		t.Fatalf("execute refused: %s", done.Text)
	}
}

// Structure below a transient slot still counts: a child added there refuses the token.
func TestARemoveTokenIsRefusedWhenAChildIsAddedBelowATransientSlot(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	cache := d.transientCache(ext)
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Set(cache, "extra", relTime("1"))
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
}

// A child retyped below a transient slot refuses the token: types stay hashed.
func TestARemoveTokenIsRefusedWhenAChildIsRetypedBelowATransientSlot(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, ext := d.alarmed()
	cache := d.transientCache(ext)
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	d.Set(cache, "count", map[string]any{"nm": "p", "t": "baja:Integer", "v": "1"})
	if err := d.Srv.Err("n4_remove_component", Execute(removeArgs(nn), dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
}
