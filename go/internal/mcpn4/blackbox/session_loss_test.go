package blackbox

// A station restart drops this server's BOX session; the station then answers every call with
// "Could not find Server Session" (retro 2026-11-16 U4). Through the public tools: a read
// reconnects once and is retried once, a second loss in a row is reported, and a write execution
// is never retried.

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// stationLost is the station's reply text after a restart (BBoxService.validateFrameServerSession).
const stationLost = "com.tridium.box.ServerSessionException: Could not find Server Session"

// loseSession makes the station refuse the first n ssc calls named sck whose ORD argument names the
// Folder (n < 0: every one) with the lost-session reply; it returns the refusal count.
func loseSession(r *Rig, sck string, n int) *int {
	refused := 0
	r.Station.SetFaults(boxstation.Faults{Refuse: func(_ *boxstation.Station, key string, arg map[string]any) error {
		if key != sck || (sck == "loadSlots" && !strings.Contains(str(arg["o"]), "Folder")) {
			return nil
		}
		if n >= 0 && refused >= n {
			return nil
		}
		refused++
		return errors.New(stationLost)
	}})
	return &refused
}

func TestAReadThatLosesItsSessionReconnectsOnceAndSucceeds(t *testing.T) {
	r := stationRig(t)
	r.Add("3", "Pump", "")
	refused := loseSession(r, "loadSlots", 1)
	out := r.Srv.OK("n4_navigate", Args{"ord": Folder})
	if named(out["children"], "Pump") == nil || *refused != 1 {
		t.Errorf("out = %v, refused %d", out, *refused)
	}
}

func TestASecondLossInARowIsReportedNotLooped(t *testing.T) {
	r := stationRig(t)
	refused := loseSession(r, "loadSlots", -1)
	text := r.Srv.Err("n4_navigate", Args{"ord": Folder})
	for _, part := range []string{"Could not find Server Session", "reconnected", "lost again", "n4_connect"} {
		if !strings.Contains(text, part) {
			t.Errorf("missing %q: %s", part, text)
		}
	}
	if *refused != 2 {
		t.Errorf("refused %d calls, want 2 (the call and its one retry)", *refused)
	}
}

func TestAWriteExecutionThatLosesItsSessionIsNotRetried(t *testing.T) {
	r, c, ord := calcRig(t)
	args := Args{"ord": ord, "slot": "out", "value": 5.5, "value_type": "baja:StatusNumeric"}
	plan := r.Srv.Dry("n4_set_slot", args)
	refused := loseSession(r, "syncTo", -1)
	text := r.Srv.Err("n4_set_slot", Execute(args, plan["confirmation_token"]))
	for _, part := range []string{"session lost", "redo the dry run", "not retried"} {
		if !strings.Contains(text, part) {
			t.Errorf("missing %q: %s", part, text)
		}
	}
	if strings.HasPrefix(text, "com.tridium") || *refused != 1 {
		t.Errorf("raw BOX text or a retried write: refused %d, %s", *refused, text)
	}
	var value any
	r.Station.Do(func(st *boxstation.Station) { value = st.ByHandle(c.Handle).Child("out").Child("value").Value })
	if value != "1.0" {
		t.Errorf("out.value = %v, want the unchanged 1.0", value)
	}
}
