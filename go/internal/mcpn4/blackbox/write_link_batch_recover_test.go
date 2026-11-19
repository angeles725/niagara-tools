package blackbox

// n4_create_link batch form when the station's replies cannot be trusted: each entry is looked
// for by its slots, and one entry whose lookup fails must not cost the others their inverse
// (verifier note on PR #812).

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestOneFailedLookupKeepsTheOtherEntriesInverse(t *testing.T) {
	r, _, tgts := linkBatchRig(t)
	checked := 0
	r.Station.SetFaults(boxstation.Faults{
		CheckLink: func(_ *boxstation.Station, arg map[string]any, orig func(map[string]any) any) any {
			checked++
			return []any{withArgs(orig(arg).([]any)[0].(map[string]any), Args{"s": nil})}
		},
		Refuse: func(_ *boxstation.Station, sck string, arg map[string]any) error {
			raw, _ := json.Marshal(arg)
			if checked == 3 && sck != "checkLinks" && strings.Contains(string(raw), "T2") {
				return errors.New("lookup refused")
			}
			return nil
		},
	})
	out := r.Srv.RunWrite("n4_create_link", Args{"links": threeLinks()})
	view := journalView(t, r, out)
	want := []any{map[string]any{"nm": "v", "h": tgts[0].Handle, "n": "Link"},
		map[string]any{"nm": "v", "h": tgts[2].Handle, "n": "Link"}}
	if out["verdict"] != "unverified" || out["in_doubt"] != true || !reflect.DeepEqual(out["inverse"], want) ||
		!reflect.DeepEqual(view["inverse"], want) {
		t.Fatalf("out = %v, view = %v", out, view)
	}
	entries, _ := out["observed"].([]any)
	if len(entries) != 3 || Dig(entries[1], "lookup_error") == nil || Dig(entries[0], "observed", "link") != "Link" {
		t.Errorf("observed = %v", out["observed"])
	}
}
