package blackbox

// n4_create_component (port of test_tools_write TestCreateComponent).

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestHappyPathIsVerifiedWithAnnotation(t *testing.T) {
	r := verifiedRig(t)
	out := r.Srv.RunWrite("n4_create_component", withArgs(createPump,
		Args{"wire_sheet": Args{"x": 4, "y": 6, "w": 10}}))
	wantObserved := map[string]any{"name": "Pump", "type": "kitControl:NumericConst", "wsAnnotation": "4,6,10"}
	wantInverse := []any{map[string]any{"nm": "v", "h": "3", "n": "Pump"}}
	if out["verdict"] != "verified" || Dig(out, "accepted", "nn") != "Pump" ||
		!reflect.DeepEqual(out["observed"], wantObserved) || !reflect.DeepEqual(out["inverse"], wantInverse) {
		t.Errorf("out = %v", out)
	}
	if got := nodeValue(r, "3", "Pump", "wsAnnotation"); got != "4,6,10" {
		t.Errorf("wsAnnotation on the station = %v", got)
	}
}

func TestCollisionRenameIsFollowedInReadBackAndInverse(t *testing.T) {
	r := verifiedRig(t)
	r.Add("3", "Pump", "")
	out := r.Srv.RunWrite("n4_create_component", createPump)
	wantInverse := []any{map[string]any{"nm": "v", "h": "3", "n": "Pump1"}}
	if Dig(out, "accepted", "nn") != "Pump1" || out["verdict"] != "verified" ||
		!reflect.DeepEqual(out["inverse"], wantInverse) {
		t.Errorf("out = %v", out)
	}
}

func TestBadNameOrTypeIsRefused(t *testing.T) {
	r := verifiedRig(t)
	for _, bad := range []Args{{"name": "../x"}, {"name": "a/b"}, {"name": "a$20b"}, {"name": ""}, {"type": "NoModule"},
		{"type": "a:b:c"}} {
		r.Srv.Err("n4_create_component", withArgs(createPump, bad))
	}
	if len(r.Children()) != 0 {
		t.Errorf("children %v", r.Children())
	}
}

func TestBadWireSheetIsRefused(t *testing.T) {
	r := verifiedRig(t)
	for _, ws := range []Args{{"x": 1}, {"x": "a", "y": 1}, {"x": 1, "y": 2, "w": true}} {
		r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "P", "wire_sheet": ws}))
	}
}

func TestCreateComponentReadBackMismatchIsReportedNotRaised(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		if pump := st.ByHandle("3").Child("Pump"); pump != nil {
			kept := []*boxstation.Node{}
			for _, c := range pump.Children {
				if c.Name != "wsAnnotation" {
					kept = append(kept, c)
				}
			}
			pump.Children = kept
		}
	}})
	out := r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"wire_sheet": Args{"x": 1, "y": 2}}))
	id, _ := out["batch_id"].(string)
	if out["verdict"] != "mismatch" || !batchIDPattern.MatchString(id) {
		t.Errorf("out = %v", out)
	}
}
