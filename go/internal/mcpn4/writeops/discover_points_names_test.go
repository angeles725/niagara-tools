package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// #820: the BACnet objectName lives only on the device, so the point discovery is where the
// station's point name meets it. A proxied row says whether the point is named differently from
// its object (name_differs), and observed counts those rows (names_differ); an unproxied row
// has no point to compare.
func TestDiscoverPointsFlagsAPointNamedUnlikeItsObject(t *testing.T) {
	c := ptsStation("success")
	ext, _ := c.nodes.Get(ptsExt)
	ext.(*pyjson.Object).Set("DamperCmd", obj("h", "q2", "t", "control:NumericWritable"))
	ext.(*pyjson.Object).Set("DamperCmd/proxyExt", obj("t", "bacnet:BacnetNumericProxyExt"))
	ext.(*pyjson.Object).Set("DamperCmd/proxyExt/objectId",
		obj("t", "bacnet:BacnetObjectIdentifier", "v", "analogOutput:2"))
	args := obj("device_ord", ptsDevice)
	p, err := writeops.DiscoverPoints.PlanWith(c, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	reply := obj("t", "baja:Ord", "v", "slot:/Services/JobService/BacnetDiscoverPointsJob2")
	_, _, observed, _, err := writeops.DiscoverPoints.Readback(c, args, p, []any{reply}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := js(t, observed)
	for _, want := range []string{
		`"in_database": "Folder/RoomTemp", "name_differs": true}`,
		`"in_database": "DamperCmd", "name_differs": false}`,
		`"names_differ": 1`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	if strings.Count(got, "name_differs") != 2 {
		t.Errorf("an unproxied row carries name_differs: %s", got)
	}
}
