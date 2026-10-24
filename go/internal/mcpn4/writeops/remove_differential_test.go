package writeops_test

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// pyRemove runs each case through the real n4_remove_component plan as _process hashes it (the
// plan keys included), then the link_input_values preview, the full journaled inverse, the
// targets and the read-back against the loads after the removal.
const pyRemove = `import hashlib, json, sys
from mcp_n4 import box, safety, tools_write
STATION = {"name": "FakeStation", "base_url": "http://fake.invalid", "session_id": "0" * 32}
KEYS = ("relinks", "unlinked_inputs", "components", "outgoing_links_broken", "snapshot_truncated")

def describe(e):
    return {"type": type(e).__name__, "message": str(e)}

class Client:
    def load_tree(self, ord_str, depth=2, **kw):
        if ord_str not in self.loads:
            raise box.BoxError("no such component: " + ord_str)
        return self.loads[ord_str]

impl = tools_write._IMPLS["n4_remove_component"]
for line in sys.stdin:
    c = json.loads(line)
    client = Client()
    client.loads = c["before"]
    try:
        p = impl.plan(client, c["args"])
        plan = {"tool": "n4_remove_component", "ops": p.ops,
                "inverse": tools_write._hashed_inverse(p.inverse), "notes": p.notes, "station": STATION}
        for key in KEYS:
            if p.data.get(key):
                plan[key] = p.data[key]
        out = {"plan": plan, "plan_hash": hashlib.sha256(safety.canonical(plan).encode()).hexdigest(),
               "preview": tools_write._link_input_values(p.inverse), "inverse": p.inverse,
               "targets": p.data["targets"]}
    except Exception as e:
        print(json.dumps({"error": describe(e)}))
        continue
    client.loads = c["after"]
    try:
        out["readback"] = list(impl.readback(client, c["args"], p, c["replies"], p.inverse))
    except Exception as e:
        out["readback_error"] = describe(e)
    print(json.dumps(out))
`

func goRemove(c *pyjson.Object) *pyjson.Object {
	args := get(c, "args").(*pyjson.Object)
	st := &loads{nodes: get(c, "before").(*pyjson.Object)}
	p, err := writeops.RemoveComponent.Plan(st, args)
	if err != nil {
		return obj("error", describe(err))
	}
	sess := &session.Session{StationName: "FakeStation", BaseURL: "http://fake.invalid", SessionID: strings.Repeat("0", 32)}
	hashed, hash, err := plan.Hashed("n4_remove_component", p, sess)
	if err != nil {
		return obj("error", describe(err))
	}
	preview, err := plan.LinkInputValues(p.Inverse)
	if err != nil {
		return obj("error", describe(err))
	}
	out := obj("plan", hashed, "plan_hash", hash, "preview", preview, "inverse", p.Inverse, "targets", p.Targets)
	st.nodes = get(c, "after").(*pyjson.Object)
	requested, accepted, observed, verdict, err := writeops.RemoveComponent.Readback(st, args, p, get(c, "replies").([]any), p.Inverse)
	if err != nil {
		out.Set("readback_error", describe(err))
	} else {
		out.Set("readback", []any{requested, accepted, observed, verdict})
	}
	return out
}

// flat is box._flatten of a nested node to depth levels below it (what a load of that depth
// returns), as JSON text.
func flat(node *pyjson.Object, depth int) *pyjson.Object {
	out := pyjson.NewObject()
	var walk func(n *pyjson.Object, path string, left int)
	walk = func(n *pyjson.Object, path string, left int) {
		out.Set(path, n)
		if left == 0 {
			return
		}
		kids, _ := get(n, "s").([]any)
		for _, k := range kids {
			name := get(k.(*pyjson.Object), "n").(string)
			if path != "" {
				name = path + "/" + name
			}
			walk(k.(*pyjson.Object), name, left-1)
		}
	}
	walk(node, "", depth)
	return out
}

func mustObj(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatalf("%v: %s", err, text)
	}
	return v.(*pyjson.Object)
}

// link is a baja:Link child with the given slot values (JSON texts; "" leaves a slot out).
func link(name, sourceOrd, sourceSlot, targetSlot string) string {
	var s []string
	for _, kv := range [][2]string{{"sourceOrd", sourceOrd}, {"sourceSlotName", sourceSlot}, {"targetSlotName", targetSlot}} {
		if kv[1] != "" {
			s = append(s, `{"n": "`+kv[0]+`", "v": `+kv[1]+`}`)
		}
	}
	return `{"n": "` + name + `", "t": "baja:Link", "s": [` + strings.Join(s, ", ") + `]}`
}

// status is a Status input as the live station loads it (frozen children carry no type).
func status(name, t, value, st string) string {
	return `{"n": "` + name + `", "t": "` + t + `", "s": [{"n": "value", "v": ` + value + `}, {"n": "status", "v": ` + st + `}]}`
}

// removeSubtrees are the removed component Grp (handle "g"): odd handles, types, values and
// links, link-driven inputs, runtime outputs and subtrees deeper than the snapshot.
func removeSubtrees() []string {
	deep := func(n int) string {
		var leaves []string
		for i := range n {
			leaves = append(leaves, fmt.Sprintf(`{"n": "K%02d", "t": "baja:Folder", "s": [{"n": "x", "v": "1"}]}`, (i*7)%n))
		}
		return `{"h": "g", "t": "baja:Folder", "s": [{"n": "L1", "h": "l1", "s": [{"n": "L2", "s": [{"n": "C", "h": "c", "s": [` +
			strings.Join(leaves, ", ") + `]}]}]}]}`
	}
	src := `{"n": "Src", "h": "s1", "t": "kitControl:NumericConst", "s": [` + status("out", "baja:StatusNumeric", `"4.5"`, `"0;activeLevel=e_in16"`) +
		`, ` + status("fallback", "baja:StatusNumeric", `"7.0"`, `"40;activeLevel=e_def"`) + `]}`
	tgt := func(links ...string) string {
		return `{"n": "Tgt", "h": "t1", "t": "kitControl:NumericConst", "s": [` + status("in10", "baja:StatusNumeric", `"2.5"`, `"0;x"`) +
			`, ` + status("inB", "baja:StatusNumeric", `"25.0"`, `"0;activeLevel=e:17"`) + `, {"n": "out", "t": "baja:StatusNumeric"}` +
			strings.Join(append([]string{""}, links...), ", ") + `]}`
	}
	grp := func(kids ...string) string {
		return `{"h": "g", "t": "baja:Folder", "s": [{"n": "wsAnnotation", "t": "baja:WsAnnotation", "v": "1,2,3"}, ` +
			strings.Join(kids, ", ") + `]}`
	}
	return []string{
		`{"h": "g", "t": "baja:Folder"}`,
		`{"h": "g", "n": "Grp", "t": "baja:Folder", "v": "root"}`,
		grp(src, tgt(link("Link", `"h:s1"`, `"out"`, `"in10"`)), link("Link", `"h:s1"`, `"out"`, `"in10"`),
			status("in10", "baja:StatusNumeric", `"8.5"`, `"0;g"`)),
		`{"h": "g"}`,
		`{"h": "g", "t": null, "v": null, "s": []}`,
		grp(src, tgt(link("Link", `"h:s1"`, `"out"`, `"in10"`))),
		grp(src, tgt(link("Link", `"station:|h:s1"`, `"out"`, `"in10"`), link("Link1", `"h:zz"`, `"out"`, `"inB"`))),
		grp(src, tgt(link("Link", `"h:g"`, `"wsAnnotation"`, `"in10"`), link("Link1", `"h:"`, `"out"`, `"inC"`))),
		grp(src, tgt(link("Link", `null`, `null`, `null`), link("Link1", `""`, `1`, `2`))),
		grp(src, tgt(link("Link", `"h:s1"`, `"out"`, `"out"`))),
		grp(src, tgt(link("Link", `7`, `"out"`, `"in10"`))),
		grp(src, tgt(link("Link", `"hxs1"`, `"out"`, `"in10"`), link("Link1", `"a|hxt1"`, `"out"`, `"inB"`))),
		grp(src, tgt(link("Link", `["h:s1"]`, `"out"`, `"in10"`))),
		grp(src, tgt(link("Link", `"h:s1"`, `"out"`, `["in10"]`))),
		grp(src, tgt(link("Link", `"h:s1"`, `"out"`, `{"a": 1}`))),
		grp(src, tgt(link("Link", `"h:s1"`, `"out"`, `1.5`))),
		grp(src, tgt(`{"n": "Link", "t": "baja:ConversionLink", "s": [{"n": "sourceOrd", "v": "h:s1"}, {"n": "targetSlotName", "v": "inB"}, {"n": "targetSlotName", "v": "in10"}]}`)),
		grp(src, `{"n": "Tgt", "h": "t1", "s": [{"n": "Link", "t": "baja:Link"}]}`),
		grp(`{"n": "A", "h": 5, "s": [{"n": "B", "h": 5.0}, {"n": "C", "h": true}]}`, tgt(link("Link", `"h:5"`, `"out"`, `"in10"`))),
		grp(`{"n": "A", "h": [1]}`),
		grp(`{"n": "A", "h": {"a": 1}}`),
		grp(`{"n": "A", "h": []}`, `{"n": "B", "h": {}}`, `{"n": "C", "h": 0}`, `{"n": "D", "h": ""}`),
		grp(`{"n": "S", "t": "baja:Status", "v": "40;activeLevel=e_def"}`, `{"n": "N", "t": "baja:Status", "v": 7}`,
			`{"n": "F", "t": "baja:Status", "v": 1e400}`, `{"n": "L", "t": "baja:Status", "v": [1, ";"]}`,
			`{"n": "D", "t": "baja:Status", "v": {"a": "b;c"}}`, `{"n": "B", "t": "baja:Status", "v": true}`,
			`{"n": "Z", "t": "baja:Status", "v": null}`, `{"n": "status", "v": "1;2"}`),
		grp(`{"n": "P", "t": "baja:StatusBoolean", "s": [{"n": "status", "v": "0;a"}, {"n": "value", "v": "x;y"}]}`,
			`{"n": "Q", "t": null, "s": [{"n": "status", "v": "0;a"}]}`, `{"n": "R", "t": 5, "s": [{"n": "status", "v": "0;a"}]}`),
		grp(`{"n": "out", "t": "baja:StatusNumeric", "s": [{"n": "value", "v": "1"}]}`, `{"n": "O", "t": "kitControl:X", "s": [{"n": "out", "t": "baja:StatusNumeric"}, {"n": "out2", "t": "baja:StatusNumeric"}]}`,
			`{"n": "Bo", "s": [{"n": "out", "t": "baja:Double", "v": "3"}]}`, `{"n": "Nt", "s": [{"n": "out", "t": null}]}`),
		grp(`{"n": "W", "t": 7, "v": [1], "s": [{"n": "x", "t": true, "v": {"k": 1}}]}`),
		deep(1), deep(20), deep(21), deep(25),
		`{"h": "g", "s": [{"n": "A", "t": "baja:Folder", "s": [{"n": "B", "s": [{"n": "C", "s": [{"n": "Link", "t": "baja:Link", "s": [{"n": "sourceOrd", "v": "h:g"}]}]}]}]}]}`,
		`{"h": "g", "s": [{"n": "A", "h": "a", "s": [{"n": "B", "h": "b", "s": [{"n": "Link", "t": "baja:Link", "s": [{"n": "sourceOrd", "v": "h:a"}, {"n": "targetSlotName", "v": "in"}]}, ` +
			status("in", "baja:StatusNumeric", `"1"`, `"0;q"`) + `]}]}]}`,
	}
}

// scanTrees are the loads of link_scan_ord (Folder, handle "3"): outgoing links from the
// subtree's handles, links whose holder is inside it, odd holders, slots and names.
func scanTrees() []string {
	comp := func(name, h string, links ...string) string {
		return `{"n": "` + name + `", "h": ` + h + `, "t": "kitControl:NumericConst", "s": [` + strings.Join(links, ", ") + `]}`
	}
	return []string{
		`{"h": "3"}`,
		`{"h": "3", "s": [` + comp("Out", `"o1"`, link("Link", `"h:s1"`, `"out"`, `"in10"`)) + `, ` +
			comp("Out2", `"o2"`, link("Link", `"station:|h:t1"`, `"out"`, `"in1"`), link("Link1", `"h:s1"`, `"out"`, `"in2"`)) + `]}`,
		`{"h": "3", "s": [` + comp("Grp", `"g"`, comp("Src", `"s1"`), link("Link", `"h:s1"`, `"out"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [` + comp("B", `"b"`, link("Link", `"h:s1"`, `null`, `null`)) + `, ` +
			comp("A", `"a"`, link("L2", `"h:t1"`, `"x"`, `"in"`), link("L1", `"h:s1"`, `"x"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [` + comp("A", `"a"`, link("L1", `"h:s1"`, `"x"`, `null`), link("L2", `"h:s1"`, `"x"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [` + comp("A", `"a"`, link("L1", `"h:s1"`, `"x"`, `1`), link("L2", `"h:t1"`, `"x"`, `true`), link("L3", `"h:s1"`, `"x"`, `1.0`)) + `]}`,
		`{"h": "3", "s": [` + comp("A", `[1]`, link("L1", `"h:s1"`, `"x"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [` + comp("A", `"g"`, link("L1", `"h:s1"`, `"x"`, `"in"`)) + `, ` + comp("N", `null`, link("L1", `"h:t1"`, `"x"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [` + comp("A|B", `"x"`, link("L1", `"h:s1"`, `"x"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [` + comp("A", `"a"`, link("L1", `"hxs1"`, `"x"`, `"in"`), link("L2", `"h:t1"`, `"x"`, `"in"`),
			link("L3", `"h:"`, `"x"`, `"in3"`)) + `]}`,
		`{"h": "3", "s": [` + comp("..", `"x"`, link("L1", `"h:s1"`, `"x"`, `"in"`)) + `]}`,
		`{"h": "3", "s": [{"n": "Deep", "s": [{"n": "D2", "s": [` + comp("Z", `"z"`, link("L1", `"h:s1"`, `"x"`, `"in"`), link("L0", `7`, `"x"`, `"in"`)) + `]}]}]}`,
		`{"h": 3, "s": [` + comp("A", `"a"`, link("L1", `"h:s1"`, `"x"`, `["l"]`), link("L2", `"h:s1"`, `"x"`, `"in"`)) + `]}`,
	}
}

func removeCases(t *testing.T) []string {
	const folder, child = "station:|slot:/Folder", "station:|slot:/Folder/Grp"
	var out []string
	add := func(args string, sub *pyjson.Object, scan *pyjson.Object, reply string, gone int) { // 0 still, 1 gone, 2 no load
		before := obj(folder, mustObj(t, `{"": {"h": "3", "t": "baja:Folder"}, "Grp": {"n": "Grp"}, "Other": {}}`), child, flat(sub, writeops.SnapshotDepth+1))
		if scan != nil {
			before.Set(folder+"/Scan", flat(scan, writeops.LinkScanDepth))
		}
		after := obj(folder, mustObj(t, `{"": {"h": "3"}, "Other": {}}`))
		switch gone {
		case 0:
			after.Set(folder, mustObj(t, `{"": {"h": "3"}, "Grp": {}}`))
		case 2:
			after = obj()
		}
		c := obj("args", mustObj(t, "{"+args+"}"), "before", before,
			"replies", []any{get(mustObj(t, `{"r": `+reply+`}`), "r")}, "after", after)
		line, err := pyjson.Marshal(c, pyjson.Options{})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(line))
	}
	base := `"parent_ord": "` + folder + `", "name": "Grp"`
	scanArg := base + `, "link_scan_ord": "` + folder + `/Scan"`
	subs, scans := removeSubtrees(), scanTrees()
	for i, s := range subs {
		sub := mustObj(t, s)
		add(base, sub, nil, `[]`, i%3)
		for j, sc := range scans {
			add(scanArg, sub, mustObj(t, sc), []string{`[]`, `null`, `"x"`, `[{"a": 1}]`}[j%4], (i+j)%3)
		}
	}
	plain := mustObj(t, subs[3])
	for _, a := range []string{`"parent_ord": "` + folder + `", "name": "../x"`, `"parent_ord": "` + folder + `", "name": ""`,
		`"parent_ord": "` + folder + `", "name": "Grp\n"`, `"parent_ord": "` + folder + `", "name": "Nope"`,
		`"parent_ord": "` + folder + `", "name": 7`, `"parent_ord": "station:|slot:/Missing", "name": "Grp"`,
		base + `, "link_scan_ord": "station:|slot:/Missing"`, base + `, "link_scan_ord": null`, base + `, "link_scan_ord": "` + folder + `"`,
		`"parent_ord": "` + folder + `/", "name": "Grp"`, `"parent_ord": "` + folder + `//", "name": "Grp"`} {
		add(a, plain, mustObj(t, scans[1]), `[]`, 1)
	}
	for _, parent := range []string{folder + "//", folder + "/"} { // loadable: ChildOrd decides
		c := obj("args", obj("parent_ord", parent, "name", "Grp"), "before",
			obj(parent, mustObj(t, `{"": {"h": "3"}, "Grp": {}}`), folder+"/Grp", flat(plain, 4)), "replies", []any{[]any{}},
			"after", obj(parent, mustObj(t, `{"": {"h": "3"}}`)))
		line, _ := pyjson.Marshal(c, pyjson.Options{})
		out = append(out, string(line))
	}
	return out
}

func TestRemovePlanAndReadbackDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	cases := removeCases(t)
	cmd := exec.Command(py, "-c", pyRemove)
	cmd.Dir = kit
	cmd.Env = append(cmd.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	cmd.Stdin = strings.NewReader(strings.Join(cases, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<24)
	kinds, n := map[string]int{}, 0
	for i := 0; sc.Scan(); i++ {
		c, err := pyjson.Unmarshal([]byte(cases[i]))
		if err != nil {
			t.Fatal(err)
		}
		got, _ := pyjson.Marshal(goRemove(c.(*pyjson.Object)), pyjson.Options{})
		if string(got) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", i, cases[i], got, sc.Text())
		}
		for _, k := range removeOutcomes(sc.Text()) {
			kinds[k]++
		}
		n++
	}
	if n != len(cases) {
		t.Fatalf("python answered %d of %d cases", n, len(cases))
	}
	t.Logf("%d cases: %v", n, kinds)
	for _, k := range []string{"refused", "internal", "verified", "mismatch", "readback_error", "relink", "unlinked_input",
		"prior", "snapshot_truncated", "outgoing_links_broken"} {
		if kinds[k] == 0 {
			t.Errorf("no case reaches %s", k)
		}
	}
}

// removeOutcomes classifies a Python result line (its outcome, then each plan feature it
// shows), so the corpus provably reaches every branch.
func removeOutcomes(line string) []string {
	out := []string{outcome(line)}
	if out[0] == "internal" { // and which Python class
		out = append(out, strings.SplitN(strings.SplitN(line, `"type": "`, 2)[1], `"`, 2)[0])
	}
	for _, k := range []string{"outgoing_links_broken", "snapshot_truncated", `"prior"`, `"unlinked_input"`, `"relink"`} {
		if strings.Contains(line, k) {
			out = append(out, strings.Trim(k, `"`))
		}
	}
	return out
}
