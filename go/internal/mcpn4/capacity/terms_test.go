package capacity

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
)

func capInfo(attrs ...[2]string) *license.Info {
	return &license.Info{HostID: "Lab-0000", Features: []license.Feature{{Name: "tridium:about", Expires: "never"},
		{Name: FeatureName, Expires: "never", Attrs: attrs}}}
}

func TestTermReadsTheLimitAndTheExcludedModules(t *testing.T) {
	info := capInfo([2]string{"device.limit", "26"}, [2]string{"excludedDevices", "ndio;nrio,niagaraDriver"})
	got := TermOf(info, Devices)
	if got.Limit == nil || *got.Limit != 26 || got.Unlimited || !got.ExcludedKnown ||
		!reflect.DeepEqual(got.Excluded, []string{"ndio", "nrio", "niagaraDriver"}) {
		t.Fatalf("TermOf = %+v", got)
	}
}

func TestTermIsUnlimitedForNoneAnAbsentKeyOrNoFeature(t *testing.T) {
	for _, info := range []*license.Info{capInfo([2]string{"device.limit", "None"}), capInfo(),
		{HostID: "Lab-0000"}} {
		got := TermOf(info, Devices)
		if got.Limit != nil || !got.Unlimited || !got.ExcludedKnown || len(got.Excluded) != 0 {
			t.Errorf("TermOf = %+v, want unlimited with nothing excluded", got)
		}
	}
}

func TestTermOfAnUnreadableLimitIsUnknownWithANote(t *testing.T) {
	got := TermOf(capInfo([2]string{"point.limit", "lots"}), Points)
	if got.Limit != nil || got.Unlimited || got.Note == "" {
		t.Fatalf("TermOf = %+v, want an unknown limit with a note", got)
	}
}

func TestTermOfACutExclusionListIsUnknown(t *testing.T) {
	got := TermOf(capInfo([2]string{"excludedPoints", "a;b" + license.CutMarker}), Points)
	if got.ExcludedKnown || got.Note == "" {
		t.Fatalf("TermOf = %+v, want exclusions unknown with a note", got)
	}
}

func TestCountRowsSkipsExcludedModules(t *testing.T) {
	csv := "Type\nbacnet:BacnetDevice\nnrio:NrioDevice\nniagaraDriver:NiagaraStation\nmodbusAsync:ModbusAsyncDevice\n"
	got, note := countTypes(csv, false, []string{"nrio", "niagaraDriver"})
	if got == nil || *got != 2 || note != "" {
		t.Fatalf("countTypes = %v %q, want 2", got, note)
	}
	if got, note := countTypes(csv, true, nil); got != nil || note == "" {
		t.Fatalf("a cut answer = %v %q, want null with a note", got, note)
	}
}
