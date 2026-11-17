package blackbox

// n4_tag_coverage (station-audits S9): one read-only call lists the control points missing
// hs:point or an hs:equipRef relation, the equips no point references and the direct tags no
// installed dictionary defines, from a fake station's tree and query answers.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// tcDictionaries installs the Niagara dictionary (namespace n from its type, defining point)
// and, with lab, a Lab dictionary (namespace tst, defining cap).
func tcDictionaries(r *Rig, lab bool) {
	svc := r.Add("2", "Services", "baja:Folder")
	tds := r.Add(svc.Handle, "TagDictionaryService", "tagdictionary:TagDictionaryService")
	if lab {
		d := r.Add(tds.Handle, "Lab", "tagdictionary:TagDictionary")
		r.Set(d.Handle, "namespace", map[string]any{"nm": "p", "t": "baja:String", "v": "tst"})
		defs := r.Add(d.Handle, "tagDefinitions", "tagdictionary:TagInfoList")
		r.Add(defs.Handle, "cap", "tagdictionary:SimpleTagInfo")
	}
	nia := r.Add(tds.Handle, "Niagara", "tagdictionary:NiagaraTagDictionary")
	r.Add(r.Add(nia.Handle, "tagDefinitions", "tagdictionary:TagInfoList").Handle, "point", "tagdictionary:SimpleTagInfo")
	r.Station.SetRegistry(map[string]any{"tagdictionary:NiagaraTagDictionary": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "namespace", "v": map[string]any{"t": "baja:String", "v": "n"}}}}})
}

// tcTag adds the direct tag id (a Marker, or a String with value) to the component h.
func tcTag(r *Rig, h, id string) {
	r.T.Helper()
	body := map[string]any{"nm": "p", "t": "baja:Marker", "v": "M"}
	if _, err := r.Station.AddSlot(h, strings.Replace(id, ":", "$3a", 1), body, "m"); err != nil {
		r.T.Fatal(err)
	}
}

// tcRig is a connected rig whose Folder holds the equips AHU (referenced) and Orphan, the point
// Covered (hs:point and hs:equipRef to AHU, plus tst:bogus) and the point Bare (tst:bogus only).
func tcRig(t *testing.T, lab bool) *Rig {
	t.Helper()
	r, sv := servletRig(t, nil)
	tcDictionaries(r, lab)
	ahu, orphan := r.Add("3", "AHU", "baja:Folder"), r.Add("3", "Orphan", "baja:Folder")
	tcTag(r, ahu.Handle, "hs:equip")
	tcTag(r, orphan.Handle, "hs:equip")
	covered, bare := r.Add("3", "Covered", "control:NumericWritable"), r.Add("3", "Bare", "control:NumericWritable")
	tcTag(r, covered.Handle, "hs:point")
	tcTag(r, covered.Handle, "tst:bogus")
	tcTag(r, bare.Handle, "tst:bogus")
	rel := map[string]any{"nm": "p", "t": "baja:Relation", "s": []any{
		map[string]any{"nm": "p", "n": "relationId", "t": "baja:String", "v": "hs:equipRef"},
		map[string]any{"nm": "p", "n": "sourceOrd", "t": "baja:Ord", "v": "h:" + ahu.Handle}}}
	if _, err := r.Station.AddSlot(covered.Handle, "r?", rel, "r"); err != nil {
		t.Fatal(err)
	}
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte("Slot Path\nslot:/Folder/Covered\nslot:/Folder/Bare\n")},
		boxstation.OrdRoute{Needle: "neql:hs:equip", Body: []byte("Ord\nlocal:|station:|slot:/Folder/AHU\nlocal:|station:|slot:/Folder/Orphan\n")})
	return r
}

func TestTagCoverageListsUncoveredPointsOrphanEquipsAndUnknownTags(t *testing.T) {
	r := tcRig(t, true)
	out := r.Srv.OK("n4_tag_coverage", Args{})
	gaps := objects(out["points_without_tags"])
	if len(gaps) != 1 || gaps[0]["point"] != Folder+"/Bare" ||
		!reflect.DeepEqual(gaps[0]["missing"], []any{"hs:point", "hs:equipRef"}) {
		t.Errorf("points_without_tags %v", gaps)
	}
	if got := field(out["equips_without_points"], "equip"); !reflect.DeepEqual(got, []any{Folder + "/Orphan"}) {
		t.Errorf("equips_without_points %v", got)
	}
	unknown := objects(out["unknown_tags"])
	if len(unknown) != 2 || unknown[0]["tag"] != "hs:*" || unknown[0]["count"] != float64(3) ||
		unknown[1]["tag"] != "tst:bogus" || unknown[1]["count"] != float64(2) || unknown[1]["example"] != Folder+"/Covered" ||
		unknown[1]["why"] != "not defined by the installed dictionary Lab" {
		t.Errorf("unknown_tags %v", unknown)
	}
	counts := out["counts"].(map[string]any)
	if counts["points"] != float64(2) || counts["equips"] != float64(2) || counts["missing_equip_ref"] != float64(1) ||
		counts["equips_without_points"] != float64(1) || out["truncated"] != false {
		t.Errorf("counts %v truncated %v", counts, out["truncated"])
	}
}

func TestTagCoverageWithoutAHaystackDictionaryNotesItOnce(t *testing.T) {
	r := tcRig(t, false) // the lab: only the Niagara dictionary, every hs: tag ad hoc
	out := r.Srv.OK("n4_tag_coverage", Args{})
	unknown := objects(out["unknown_tags"])
	hs := []map[string]any{}
	for _, u := range unknown {
		if strings.HasPrefix(u["tag"].(string), "hs") {
			hs = append(hs, u)
		}
	}
	if len(hs) != 1 || hs[0]["tag"] != "hs:*" || hs[0]["count"] != float64(3) ||
		hs[0]["why"] != "no dictionary installed for namespace 'hs' (ad hoc tags)" {
		t.Errorf("unknown_tags %v", unknown)
	}
	if got := out["dictionaries"]; !reflect.DeepEqual(got, []any{"Niagara (n)"}) {
		t.Errorf("dictionaries %v", out["dictionaries"])
	}
}

func TestTagCoverageDictionaryArgumentChecksThatNamespaceOnly(t *testing.T) {
	r := tcRig(t, true)
	out := r.Srv.OK("n4_tag_coverage", Args{"dictionary": "Lab"})
	if got := field(out["unknown_tags"], "tag"); !reflect.DeepEqual(got, []any{"tst:bogus"}) {
		t.Errorf("unknown_tags %v", got)
	}
	if msg := r.Srv.Err("n4_tag_coverage", Args{"dictionary": "Haystack"}); !strings.Contains(msg, "no tag dictionary named") {
		t.Errorf("unknown dictionary: %s", msg)
	}
}
