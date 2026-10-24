package writeops_test

import "strings"

const (
	folderOrd = "station:|slot:/Folder"
	srcOrd    = folderOrd + "/Src"
	tgtOrd    = folderOrd + "/Tgt"
)

// casesOf is one case line per (args, before, replies, after); the tool's loads are keyed by
// the ORDs in before and after.
func casesOf(tool string, args, befores, replies, afters []string) []string {
	var out []string
	for _, a := range args {
		for _, b := range befores {
			for _, r := range replies {
				for _, af := range afters {
					out = append(out, strings.ReplaceAll(`{"tool": "`+tool+`", "args": {`+a+`}, "before": `+b+
						`, "replies": [`+r+`], "after": `+af+`}`, "\n\t", " "))
				}
			}
		}
	}
	return out
}

func createCases() []string {
	base := `"parent_ord": "` + folderOrd + `", "type": "kitControl:NumericConst", `
	accepted := []string{base + `"name": "Pump"`, base + `"name": "Pump", "wire_sheet": {"x": 1, "y": 2}`,
		base + `"name": "Pump", "wire_sheet": {"x": -1, "y": 2, "w": 8}`,
		base + `"name": "Pump", "wire_sheet": {"x": 10000000000000000000000, "y": -0, "w": 3}`, base + `"name": "Pump\n"`}
	refused := []string{base + `"name": "../x"`, base + `"name": ""`, base + `"name": "1x"`, base + `"name": "a\n\n"`,
		base + `"name": "é"`, `"parent_ord": "` + folderOrd + `", "name": "P", "type": "NoModule"`,
		`"parent_ord": "` + folderOrd + `", "name": "P", "type": "a:b:c"`, `"parent_ord": "` + folderOrd + `", "name": "P", "type": "k:T\n"`,
		`"parent_ord": "` + folderOrd + `", "name": "P", "type": "1:T"`, `"parent_ord": "` + folderOrd + `", "name": "P", "type": "k_:T_9"`,
		`"parent_ord": "` + folderOrd + `", "name": "P x", "type": "x"`,
		base + `"name": "P", "wire_sheet": {}`, base + `"name": "P", "wire_sheet": {"x": 1}`,
		base + `"name": "P", "wire_sheet": {"x": 1.0, "y": 2}`, base + `"name": "P", "wire_sheet": {"x": true, "y": 2}`,
		base + `"name": "P", "wire_sheet": {"x": 1, "y": 2, "w": 0}`, base + `"name": "P", "wire_sheet": {"x": 1, "y": 2, "w": -100000000000000000000000}`,
		base + `"name": "P", "wire_sheet": {"x": 1, "y": 2, "w": null}`, base + `"name": "P", "wire_sheet": {"x": 1, "y": "2"}`,
		base + `"name": "P", "wire_sheet": {"x": 1, "y": 2, "w": 1e3}`, base + `"name": "P", "wire_sheet": null`,
		`"parent_ord": "station:|slot:/Missing", "name": "P", "type": "k:T"`}
	before := func(root string) string { return `{"` + folderOrd + `": {"": ` + root + `}}` }
	befores := []string{before(`{"h": "3", "t": "baja:Folder"}`), before(`{"h": 3}`)}
	odd := []string{before(`{"h": ""}`), before(`{"t": "baja:Folder"}`), before(`{"h": null}`), before(`{"h": [1]}`), `{}`}
	replies := []string{`[{"id": "a", "nn": "Pump"}]`, `[{"nn": "Pump1"}]`, `[]`, `[{"nn": null}]`, `["x"]`,
		`[[1]]`, `"junk"`, `""`, `"é!"`, `"\ud800x"`, `{"a": 1}`, `{}`, `7`, `0`, `1.5`, `true`, `false`, `null`,
		`[{"nn": ["l"]}]`, `[{"nn": {"a": 1}}]`, `[{"nn": 5}]`, `[{"nn": 1.0}]`, `[{"nn": ""}]`, `[{"nn": "Pump"}, {"nn": "X"}]`}
	afters := []string{`{"` + folderOrd + `": {"": {"h": "3", "t": "baja:Folder"}, "Pump": {"t": "kitControl:NumericConst"},
	"Pump/wsAnnotation": {"t": "baja:WsAnnotation", "v": "1,2,8"}, "Pump1": {"t": "k:Other"}, "Pump1/wsAnnotation": {}}}`,
		`{"` + folderOrd + `": {"": {"h": "3"}, "Pump": {"t": "kitControl:NumericConst"}, "Pump1": {"t": "kitControl:NumericConst"},
	"Pump\n": {"t": "kitControl:NumericConst"}, "Pump/wsAnnotation": {"v": "-1,2,8"}}}`, `{}`}
	out := casesOf("n4_create_component", accepted, befores, replies, afters)
	out = append(out, casesOf("n4_create_component", refused, befores, replies[:1], afters[:1])...)
	return append(out, casesOf("n4_create_component", accepted[:1], odd, replies[:1], afters[:1])...)
}

func linkCases() []string {
	base := `"source_ord": "` + srcOrd + `", "target_ord": "` + tgtOrd + `", "source_slot": "out", `
	accepted := []string{base + `"target_slot": "in10"`, base + `"target_slot": "in10\n"`}
	refused := []string{base + `"target_slot": "a b"`, base + `"target_slot": ""`, `"source_ord": "` + srcOrd +
		`", "target_ord": "` + tgtOrd + `", "source_slot": "1x", "target_slot": "a b"`,
		`"source_ord": "` + srcOrd + `", "target_ord": "` + tgtOrd + `", "source_slot": "o\n\n", "target_slot": "in"`,
		`"source_ord": "station:|slot:/Missing", "target_ord": "` + tgtOrd + `", "source_slot": "out", "target_slot": "in"`,
		`"source_ord": "` + srcOrd + `", "target_ord": "station:|slot:/Missing", "source_slot": "out", "target_slot": "in"`}
	loads := func(src, tgt string) string {
		return `{"` + srcOrd + `": {"": ` + src + `}, "` + tgtOrd + `": ` + tgt + `}`
	}
	tgt := `{"": {"h": "11"}, "Link": {"t": "baja:Link"}, "Other": {"t": "k:T"}}`
	befores := []string{loads(`{"h": "10"}`, tgt), loads(`{"h": 16}`, tgt)}
	odd := []string{loads(`{"h": ""}`, tgt), loads(`{"h": "10"}`, `{"": {"h": ""}}`), loads(`{"h": "10"}`, `{"": {"h": "11"}, "Link": {"t": 5}}`)}
	replies := []string{`[[{"v": true, "r": null, "s": "Link1"}]]`, `[[{"v": false, "r": "type mismatch", "s": null}]]`,
		`[[{"v": 0}]]`, `[[{"v": true, "s": ""}]]`, `[[{"v": true}]]`, `[[{"r": 1}]]`, `"junk"`, `[]`, `[null]`, `[[]]`,
		`[[[{"v": 1}]]]`, `[[{"v": 1, "s": 5}]]`, `[[{"v": "yes", "s": ["l"]}]]`, `[[{"v": true, "s": {"a": 1}}]]`,
		`[[{"v": true, "s": "Ghost"}]]`, `[[{"v": true, "s": "Link"}]]`, `null`, `[[{"v": true, "s": "Link1"}, 2], 3]`}
	link := func(name, src, ss, ts string) string {
		return `, "` + name + `": {"t": "baja:Link"}, "` + name + `/sourceOrd": {"v": ` + src + `}, "` + name +
			`/sourceSlotName": {"v": ` + ss + `}, "` + name + `/targetSlotName": {"v": ` + ts + `}`
	}
	after := func(links string) string {
		return `{"` + tgtOrd + `": {"": {"h": "11"}, "Link": {"t": "baja:Link"}` + links + `}}`
	}
	afters := []string{after(link("Link1", `"h:10"`, `"out"`, `"in10"`)), after(link("Link1", `"station:|h:10"`, `"out"`, `"in10"`)),
		after(link("Link1", `"h:10"`, `"out"`, `"in10"`) + link("Link2", `"h:10"`, `"out"`, `"in10"`)),
		after(link("Link1", `"h:10"`, `"out"`, `"x"`)), after(link("Link1", `5`, `"out"`, `"in10"`)),
		after(link("Link1", `null`, `"out"`, `"in10"`)), after(link("Link1", `"h:16"`, `"out"`, `"in10\n"`)),
		after(link("Link1", `"h:10"`, `5`, `"in10"`)), `{}`, `{"` + tgtOrd + `": {"": {"t": "k:T"}}}`}
	out := casesOf("n4_create_link", accepted, befores, replies, afters)
	out = append(out, casesOf("n4_create_link", refused, befores, replies[:1], afters[:1])...)
	return append(out, casesOf("n4_create_link", accepted[:1], odd, replies[:1], afters[:1])...)
}
