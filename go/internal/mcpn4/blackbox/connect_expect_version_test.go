package blackbox

// n4_connect expected_version (field retro delta U14): a prefix of the detected version, by
// whole dotted parts; a mismatch or an undetected version refuses the connect and leaves no
// session.

import (
	"strings"
	"testing"
)

func TestConnectExpectedVersionMatchesByDottedPrefix(t *testing.T) {
	for _, want := range []string{"4.14.0.162", "4.14", "4"} {
		r := NewRig(t, nil)
		out := r.Connect(Args{"expected_version": want})
		if out["version"] != "4.14.0.162" {
			t.Errorf("expected_version %s: connect = %v", want, out)
		}
	}
}

func TestConnectExpectedVersionMismatchRefusesAndLeavesNoSession(t *testing.T) {
	for _, c := range []struct{ station, want string }{
		{"4.14.0.162", "4.15"}, {"4.14.0.162", "4.1"}, {"4.14.0.162", "4.14.0.16"}, {"", "4.14"},
	} {
		r := NewRig(t, nil)
		r.Station.SetVersion(c.station)
		text := r.Srv.Err("n4_connect", Args{"station": StationName, "expected_version": c.want})
		if !strings.Contains(text, "station version mismatch") || !strings.Contains(text, c.want) {
			t.Errorf("station %q expected %q: %s", c.station, c.want, text)
		}
		if desc := r.Srv.OK("n4_describe_session", nil); desc["station_name"] != nil {
			t.Errorf("a session survived the refusal: %v", desc)
		}
	}
}

func TestConnectWithoutExpectedVersionIsUnchanged(t *testing.T) {
	r := NewRig(t, nil)
	r.Station.SetVersion("4.15.3.28")
	if out := r.Connect(nil); out["version"] != "4.15.3.28" || out["tier"] != "B" {
		t.Errorf("connect = %v", out)
	}
}
