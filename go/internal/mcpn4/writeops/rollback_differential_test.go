package writeops_test

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// pyRollback runs each case through the real n4_rollback: _rollback_plan (journal, scope,
// budget and station checks against a journal file and the loads before), the plan as _process
// hashes it, then on the loads after the write the follow-up runs (_run_components and
// _run_relinks, as _execute calls them, with the station's sync and checkLinks replies queued),
// _rollback_inverse and _rollback_readback.
const pyRollback = `import hashlib, json, sys, types
from mcp_n4 import box, safety, tools_write
STATION = {"name": "FakeStation", "base_url": "http://fake.invalid", "session_id": "0" * 32}
KEYS = ("relinks", "unlinked_inputs", "components", "outgoing_links_broken", "snapshot_truncated")
BID = "b" * 32

def describe(e):
    return {"type": type(e).__name__, "message": str(e)}

class Client:
    def load_tree(self, ord_str, depth=2, **kw):
        nodes = self.loads.get("%s@%d" % (ord_str, depth), self.loads.get(ord_str))
        if nodes is None:
            raise box.BoxError("no such component: " + ord_str)
        if "error" in nodes:
            raise box.BoxError(nodes["error"])
        return nodes

    def answer(self, queue, default):
        r = queue.pop(0) if queue else default
        if isinstance(r, dict) and "error" in r:
            raise box.BoxError(r["error"])
        return r

    def sync(self, op):
        return self.answer(self.syncs, [{"id": "a", "nn": op["n"]}])

    def ssc(self, key, arg):
        return self.answer(self.sscs, [[{"v": True, "r": None, "s": "Link"}]])

    def invalidate_handles(self):
        pass

impl = tools_write._IMPLS["n4_rollback"]
for line in sys.stdin:
    c = json.loads(line)
    client = Client()
    client.loads, client.syncs, client.sscs = c["before"], list(c["syncs"]), list(c["sscs"])
    write = types.SimpleNamespace(journal=safety.Journal(c["dir"] + "/py"), scope=safety.WriteScope(c["scopes"]),
                                  writes_executed=c["executed"], max_writes=c["max"])
    sess = types.SimpleNamespace(station_name="FakeStation", client=client)
    try:
        p = impl.plan(client, c["args"], types.SimpleNamespace(write=write, session=sess))
        inverse = tools_write._inverse_list(p.inverse)
        plan = {"tool": "n4_rollback", "ops": p.ops, "inverse": tools_write._hashed_inverse(inverse),
                "notes": p.notes, "station": STATION}
        for key in KEYS:
            if p.data.get(key):
                plan[key] = p.data[key]
        out = {"plan": plan, "plan_hash": hashlib.sha256(safety.canonical(plan).encode()).hexdigest(),
               "targets": p.data["targets"]}
    except Exception as e:
        print(json.dumps({"error": describe(e)}))
        continue
    client.loads, data, replies = c["after"], p.data, c["replies"]
    try:
        runs = {}
        if data.get("components"):
            data["created"] = tools_write._run_components(sess, write, BID, p, replies)
        if data.get("relinks"):
            data["relink_report"], inv, doubt = tools_write._run_relinks(sess, write, BID, p, replies)
            runs.update(report=data["relink_report"], inverse=inv, doubt=doubt)
        runs.update(frozen=data.get("frozen_not_restored") or [], executed=write.writes_executed)
        out["runs"] = runs
    except Exception as e:
        out["runs_error"] = describe(e)
        print(json.dumps(out))
        continue
    try:
        inverse = tools_write._inverse_list(impl.inverse(p, replies))
        out["readback"] = list(impl.readback(client, c["args"], p, replies, inverse))
    except Exception as e:
        out["readback_error"] = describe(e)
    print(json.dumps(out))
`

// rclient is pyRollback's Client: loads keyed "ORD@depth" or "ORD", queued replies.
type rclient struct {
	loads       *pyjson.Object
	syncs, sscs []any
}

func (c *rclient) LoadTree(o string, depth int, _ box.LoadOptions) (*pyjson.Object, error) {
	n, ok := c.loads.Get(fmt.Sprintf("%s@%d", o, depth))
	if !ok {
		if n, ok = c.loads.Get(o); !ok {
			return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + o}
		}
	}
	if msg, bad := n.(*pyjson.Object).Get("error"); bad {
		return nil, &box.Error{Kind: box.KindBox, Msg: msg.(string)}
	}
	return n.(*pyjson.Object), nil
}

func (c *rclient) answer(queue *[]any, def any) (any, error) {
	r := def
	if len(*queue) > 0 {
		r, *queue = (*queue)[0], (*queue)[1:]
	}
	if o, ok := r.(*pyjson.Object); ok {
		if msg, bad := o.Get("error"); bad {
			return nil, &box.Error{Kind: box.KindBox, Msg: msg.(string)}
		}
	}
	return r, nil
}

func (c *rclient) Sync(op any) (any, error) {
	return c.answer(&c.syncs, []any{obj("id", "a", "nn", get(op.(*pyjson.Object), "n"))})
}

func (c *rclient) SSC(string, any) (any, error) {
	return c.answer(&c.sscs, []any{[]any{obj("v", true, "r", nil, "s", "Link")}})
}

func (c *rclient) ClearHandles() {}

func goRollback(c *pyjson.Object) *pyjson.Object {
	client := &rclient{loads: get(c, "before").(*pyjson.Object), syncs: get(c, "syncs").([]any), sscs: get(c, "sscs").([]any)}
	var scopes []string
	for _, s := range get(c, "scopes").([]any) {
		scopes = append(scopes, s.(string))
	}
	scope, _ := ord.NewScope(scopes)
	num := func(k string) int { n, _ := get(c, k).(pyjson.Number).Float64(); return int(n) }
	w := &plan.State{Scope: scope, Journal: journal.New(get(c, "dir").(string) + "/go"), MaxWrites: num("max"),
		WritesExecuted: num("executed")}
	sess := &session.Session{StationName: "FakeStation", BaseURL: "http://fake.invalid", SessionID: strings.Repeat("0", 32)}
	args := get(c, "args").(*pyjson.Object)
	p, err := writeops.Rollback.PlanWith(client, args, w, sess)
	if err != nil {
		return obj("error", describe(err))
	}
	hashed, hash, err := plan.Hashed("n4_rollback", p, sess)
	if err != nil {
		return obj("error", describe(err))
	}
	out := obj("plan", hashed, "plan_hash", hash, "targets", p.Targets)
	client.loads = get(c, "after").(*pyjson.Object)
	replies := get(c, "replies").([]any)
	r, err := writeops.Rollback.Run(&plan.Batch{Client: client, State: w, ID: strings.Repeat("b", 32)}, p, replies)
	if err != nil {
		out.Set("runs_error", describe(err))
		return out
	}
	runs := obj()
	if r.Report != nil {
		runs.Set("report", r.Report)
		runs.Set("inverse", r.Inverse)
		runs.Set("doubt", r.Doubt)
	}
	frozen := r.Frozen
	if frozen == nil {
		frozen = []any{}
	}
	runs.Set("frozen", frozen)
	runs.Set("executed", w.WritesExecuted)
	out.Set("runs", runs)
	inverse, err := writeops.Rollback.Inverse(p, replies)
	if err == nil {
		var rq, ac, ob any
		var v string
		if rq, ac, ob, v, err = writeops.Rollback.Readback(client, args, p, replies, inverse); err == nil {
			out.Set("readback", []any{rq, ac, ob, v})
		}
	}
	if err != nil {
		out.Set("readback_error", describe(err))
	}
	return out
}

func TestRollbackPlanRunsAndReadbackDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	cases := rollbackCases(t)
	cmd := exec.Command(py, "-c", pyRollback)
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
		got, _ := pyjson.Marshal(goRollback(c.(*pyjson.Object)), pyjson.Options{})
		if string(got) != sc.Text() {
			t.Errorf("case %d %s:\n go     %s\n python %s", i, cases[i], got, sc.Text())
		}
		for _, k := range rollbackOutcomes(sc.Text()) {
			kinds[k]++
		}
		n++
	}
	if n != len(cases) {
		t.Fatalf("python answered %d of %d cases", n, len(cases))
	}
	t.Logf("%d cases: %v", n, kinds)
	for _, k := range []string{"refused", "internal", "runs_in_doubt", "runs_internal", "readback_error", "verified",
		"mismatch", "partial", "unverified", "frozen", "relinks", "ambiguous", "components"} {
		if kinds[k] == 0 {
			t.Errorf("no case reaches %s", k)
		}
	}
}

// rollbackOutcomes classifies a Python result line, so the corpus provably reaches every branch.
func rollbackOutcomes(line string) []string {
	var out []string
	switch {
	case strings.Contains(line, `{"error": {"type": "ToolError"`):
		return []string{"refused"}
	case strings.HasPrefix(line, `{"error"`):
		return []string{"internal", strings.SplitN(strings.SplitN(line, `"type": "`, 2)[1], `"`, 2)[0]}
	case strings.Contains(line, `"runs_error": {"type": "ToolError", "message": "station call failed`):
		out = append(out, "runs_in_doubt")
	case strings.Contains(line, `"runs_error"`):
		out = append(out, "runs_internal")
	case strings.Contains(line, `"readback_error"`):
		out = append(out, "readback_error")
	}
	for _, v := range []string{"verified", "mismatch", "partial", "unverified"} {
		if strings.HasSuffix(line, `, "`+v+`"]}`) {
			out = append(out, v)
		}
	}
	for k, marker := range map[string]string{"frozen": `"frozen": [{`, "relinks": `"report": {`,
		"ambiguous": `"ambiguous"`, "components": `"components": [`} {
		if strings.Contains(line, marker) {
			out = append(out, k)
		}
	}
	return out
}

// rcase is one rollback case: the journal lines, the batch rolled back, the loads before and
// after, the replies of the top ops and the queued replies of the runs, scope and budget.
type rcase struct {
	journal              []string
	bid                  any
	before, after        string
	replies, syncs, sscs string
	scopes               []any
	max, executed        int
}

const (
	fold = "station:|slot:/Folder"
	rbid = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// batch is a journaled batch: its intent (ops, station, targets, extra keys) and its result.
func batch(bid, tool, ops, inverse, targets, station, verdict, extra string) []string {
	return []string{
		`{"batch_id": "` + bid + `", "phase": "intent", "tool": "` + tool + `", "ops": ` + ops + `, "inverse_plan": ` + inverse +
			`, "station_name": ` + station + `, "targets": ` + targets + extra + `}`,
		`{"batch_id": "` + bid + `", "phase": "result", "accepted": null, "inverse": ` + inverse + `, "verdict": "` + verdict + `"}`,
	}
}

func statusBody(v, st string) string {
	return `{"nm": "p", "t": "baja:StatusNumeric", "s": [{"nm": "p", "t": "baja:Double", "n": "value", "v": "` + v +
		`"}, {"nm": "p", "t": "baja:Status", "n": "status", "v": "` + st + `"}]}`
}

// grpBody is a remove snapshot: wsAnnotation, Src (with a configured fallback), Tgt, a
// writable with a configured frozen proxyExt, and Sub/Deep.
func grpBody(extra string) string {
	return `{"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "t": "baja:WsAnnotation", "n": "wsAnnotation", "v": "1,2,3"}, ` +
		`{"nm": "p", "t": "kitControl:NumericConst", "n": "Src", "s": [{"nm": "p", "t": "baja:WsAnnotation", "n": "wsAnnotation", "v": "5,6,7"}, ` +
		`{"nm": "p", "t": "baja:StatusNumeric", "n": "fallback", "s": [{"nm": "p", "n": "value", "v": "7.0"}, {"nm": "p", "n": "status", "v": "40"}]}]}, ` +
		`{"nm": "p", "t": "kitControl:NumericConst", "n": "Tgt"}, ` +
		`{"nm": "p", "t": "control:NumericWritable", "n": "Sp", "s": [{"nm": "p", "t": "control:NullProxyExt", "n": "proxyExt", "s": [` +
		`{"nm": "p", "t": "baja:String", "n": "tuningPolicyName", "v": "Fast"}, ` +
		`{"nm": "p", "t": "baja:StatusNumeric", "n": "readValue", "s": [{"nm": "p", "n": "value", "v": "0.0"}, {"nm": "p", "n": "status", "v": "0"}]}, ` +
		`{"nm": "p", "n": "status", "t": "baja:Status", "v": "0"}, {"nm": "p", "n": "enabled", "t": "baja:Boolean", "v": "false"}]}]}, ` +
		`{"nm": "p", "t": "baja:Folder", "n": "Sub", "s": [{"nm": "p", "t": "kitControl:NumericConst", "n": "Deep"}]}` + extra + `]}`
}

const (
	relink   = `{"relink": {"source_path": "Src", "source_slot": "out", "target_path": "Tgt", "target_slot": "in10", "prior": {"t": "baja:StatusNumeric", "v": 4.5}}}`
	relink2  = `{"relink": {"source_path": "Sub/Deep", "source_slot": "out", "target_path": "Tgt", "target_slot": "in12"}}`
	unlinked = `{"unlinked_input": {"source_slot": "out", "target_path": "Tgt", "target_slot": "in11", "prior": {"v": 1}}}`
)

// grpAfter is the station after the rollback's top add as name nn: every nested parent loads
// with its new handle (Sp with its frozen proxyExt), the relink load holds both ends, and the
// read-back finds every component with its type and annotation; edit overrides entries.
func grpAfter(nn string, edit map[string]string) string {
	g := fold + "/" + nn
	m := map[string]string{
		fold + "@2": `{"": {"h": "3"}, "` + nn + `": {"t": "baja:Folder"}}`,
		g + "@1":    `{"": {"h": "g"}}`,
		g + "@3":    `{"": {"h": "g"}, "Src": {"h": "s"}, "Tgt": {"h": "t"}, "Sub/Deep": {"h": "d"}}`,
		g + "@2": `{"": {"h": "g"}, "Src": {"t": "kitControl:NumericConst"}, "Src/wsAnnotation": {"v": "5,6,7"}, ` +
			`"Tgt": {"t": "kitControl:NumericConst"}, "Sp": {"t": "control:NumericWritable"}, "Sub": {"t": "baja:Folder"}}`,
		g + "/Sp@1":  `{"": {"h": "sp"}, "proxyExt": {"t": "control:NullProxyExt"}}`,
		g + "/Sub@1": `{"": {"h": "sb"}}`,
		g + "/Sub@2": `{"": {"h": "sb"}, "Deep": {"t": "kitControl:NumericConst"}}`,
		g + "/Sp@3": `{"": {"h": "sp"}, "proxyExt": {"t": "control:NullProxyExt"}, "proxyExt/tuningPolicyName": {"t": "baja:String", "v": "Fast"}, ` +
			`"proxyExt/status": {"v": "0;x"}}`,
	}
	for k, v := range edit {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = `"` + k + `": ` + m[k]
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func rollbackCases(t *testing.T) []string {
	dir := t.TempDir()
	var out []string
	emit := func(c rcase) {
		d := filepath.Join(dir, fmt.Sprint(len(out)))
		for _, side := range []string{"py", "go"} {
			if err := os.MkdirAll(filepath.Join(d, side), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d, side, "journal.jsonl"), []byte(strings.Join(c.journal, "\n")+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if c.scopes == nil {
			c.scopes = []any{fold}
		}
		if c.max == 0 {
			c.max = 200
		}
		for _, s := range []*string{&c.before, &c.after} {
			if *s == "" {
				*s = `{"` + fold + `": {"": {"h": "3"}}}`
			}
		}
		for _, s := range []*string{&c.syncs, &c.sscs} {
			if *s == "" {
				*s = `[]`
			}
		}
		if c.replies == "" {
			c.replies = `[[{"id": "a", "nn": "Grp"}]]`
		}
		var bid any = rbid
		if c.bid != nil {
			bid = c.bid
		}
		line, err := pyjson.Marshal(obj("dir", d, "args", obj("batch_id", bid), "scopes", c.scopes, "max", c.max,
			"executed", c.executed, "before", mustObj(t, c.before), "after", mustObj(t, c.after),
			"replies", get(mustObj(t, `{"r": `+c.replies+`}`), "r"), "syncs", get(mustObj(t, `{"r": `+c.syncs+`}`), "r"),
			"sscs", get(mustObj(t, `{"r": `+c.sscs+`}`), "r")), pyjson.Options{})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(line))
	}
	removeOf := func(body, extra string, targets string) []string {
		return batch(rbid, "n4_remove_component", `[{"nm": "v", "h": "3", "n": "Grp"}]`,
			`[{"nm": "a", "h": "3", "n": "Grp", "b": `+body+`}`+extra+`]`, targets, `"FakeStation"`, "verified", "")
	}
	ft := `{"` + fold + `": "3"}`
	full := removeOf(grpBody(""), ", "+relink+", "+relink2+", "+unlinked, ft)
	// the station after the top add, as the runs and the read-back meet it
	for _, nn := range []string{"Grp", "Grp1"} {
		reply := `[[{"id": "a", "nn": "` + nn + `"}]]`
		g := fold + "/" + nn
		for _, edit := range []map[string]string{
			{},
			{g + "@3": `{"": {"h": "g"}, "Src": {"h": "s"}, "Tgt": {}}`},
			{g + "@3": `{"": {"h": "g"}, "Tgt": {"h": "t"}}`},
			{g + "@3": `{"error": "boom"}`},
			{g + "@1": `{"error": "HTTP 500"}`},
			{g + "@1": `{"": {}}`},
			{g + "/Sp@1": `{"": {"h": "sp"}}`},
			{g + "/Sp@1": `{"": {"h": "sp"}, "proxyExt": {}, "proxyExt/x": {}, "Other": {}}`},
			{g + "/Sp@3": `{"error": "HTTP 503 from station"}`},
			{g + "/Sp@3": `{"": {"h": "sp"}}`},
			{g + "/Sp@3": `{"proxyExt": {"t": "bacnet:BacnetNumericProxyExt"}, "proxyExt/tuningPolicyName": {"v": "Fast"}}`},
			{g + "/Sp@3": `{"proxyExt": {"t": "control:NullProxyExt", "v": "x"}, "proxyExt/tuningPolicyName": {"t": "baja:String", "v": "Slow"}, "proxyExt/readValue": {"t": "baja:StatusNumeric"}, "proxyExt/readValue/value": {"v": "2.0"}, "proxyExt/enabled": {"v": "true"}}`},
			{g + "@2": `{"Src": {"t": "kitControl:BooleanConst"}, "Src/wsAnnotation": {"v": "5,6,7"}, "Tgt": {"t": "kitControl:NumericConst"}}`},
			{g + "@2": `{"Src": {"t": "kitControl:NumericConst"}, "Tgt": {"t": "kitControl:NumericConst"}, "Sp": {}, "Sub": {"t": "baja:Folder"}}`},
			{g + "@2": `{"error": "boom"}`},
			{fold + "@2": `{"` + nn + `": {"t": "baja:Component"}}`},
			{fold + "@2": `{"error": "gone"}`},
			{g + "/Sub@2": ""},
		} {
			emit(rcase{journal: full, replies: reply, after: grpAfter(nn, edit)})
		}
		for _, q := range []struct{ syncs, sscs string }{
			{`[[{"nn": "Src1"}], [], "x", [{"nn": 7}]]`, ``},
			{`[{"error": "Unable to process request."}]`, ``},
			{`[[{"nn": "Src"}], {"error": "boom"}]`, ``},
			{`[[{"nn": "a/b"}]]`, ``},
			{`[[{"nn": ["l"]}]]`, ``},
			{``, `[[[{"v": false}]], [[{"r": "huh"}]]]`},
			{``, `[[[{"v": true}]], {"error": "HTTP 500"}]`},
			{``, `["junk", [[{"v": true, "s": "L9"}]]]`},
			{``, `[[[{"v": 1, "s": 2}]], [[{"v": 0}]]]`},
			{``, `[[], [[]]]`},
		} {
			emit(rcase{journal: full, replies: reply, after: grpAfter(nn, nil), syncs: q.syncs, sscs: q.sscs})
		}
	}
	for _, r := range []string{`[]`, `"junk"`, `[{"nn": null}]`, `[{"nn": 5}]`, `[["x"]]`, `[{"nn": ""}]`, `{"nn": "G"}`} {
		emit(rcase{journal: full, replies: "[" + r + "]", after: grpAfter("Grp", nil)})
		emit(rcase{journal: removeOf(`{"nm": "p", "t": "baja:Folder"}`, "", ft), replies: "[" + r + "]",
			after: `{"` + fold + `@2": {"Grp": {"t": "baja:Folder"}}}`})
	}
	// budget, scope and the journal and station checks
	for _, b := range []struct{ max, executed int }{{9, 0}, {10, 0}, {200, 191}, {200, 190}} {
		emit(rcase{journal: full, max: b.max, executed: b.executed, after: grpAfter("Grp", nil)})
	}
	for _, s := range [][]any{{"station:|slot:/Elsewhere"}, {fold, "station:|slot:/Other"}, {fold + "/Grp/Src"}, {}} {
		emit(rcase{journal: full, scopes: s})
	}
	emit(rcase{journal: full, before: `{"` + fold + `": {"": {"h": "4"}}}`})
	emit(rcase{journal: full, before: `{"` + fold + `": {"": {"h": 3}}}`})
	emit(rcase{journal: full, before: `{"` + fold + `": {"": {}}}`})
	emit(rcase{journal: full, before: `{}`})
	for _, bid := range []any{"x", 7, nil, rbid + "\n", strings.Repeat("f", 32)} {
		if bid == nil {
			emit(rcase{journal: full, bid: obj()})
			continue
		}
		emit(rcase{journal: full, bid: bid})
	}
	odd := func(inverse, targets string) []string {
		return batch(rbid, "n4_remove_component", `[]`, inverse, targets, `"FakeStation"`, "verified", "")
	}
	ok := `{"nm": "a", "h": "3", "n": "X", "b": {"nm": "p", "t": "baja:Folder"}}`
	for _, c := range [][2]string{
		{`[{"nm": "v", "h": "3", "n": "x"}]`, `{}`}, {`[{"nm": "v", "h": "9", "n": "x"}]`, ft},
		{`[{"nm": "v", "h": "3", "n": "x"}]`, `["` + fold + `"]`}, {`[{"nm": "v", "h": "3", "n": "x"}]`, `"abc"`},
		{`[{"nm": "v", "h": 3.0, "n": "x"}]`, ft}, {`[{"nm": "v", "n": "x"}]`, `{"` + fold + `": null}`},
		{`[]`, ft}, {`"ab"`, ft}, {`7`, ft}, {`{"nm": "v"}`, ft}, {`null`, ft}, {`[1, "x", null]`, ft},
		{`[` + ok + `, {"relink": {"source_path": 1, "source_slot": "out"}}]`, ft},
		{`[` + ok + `, {"relink": "x"}]`, ft},
		{`[` + ok + `, {"relink": {"source_path": "a b", "source_slot": "out", "target_path": "", "target_slot": "x"}}]`, ft},
		{`[` + ok + `, {"relink": {"source_path": "A//B", "source_slot": "o", "target_path": "T", "target_slot": "x"}}]`, ft},
		{`[` + ok + `, {"relink": {"source_path": "", "source_slot": "o", "target_path": "", "target_slot": "x", "prior": 5}}]`, ft},
		{`[` + ok + `, {"unlinked_input": 5}]`, ft},
		{`[` + ok + `, {"unlinked_input": {"source_slot": "o", "target_path": "Y", "target_slot": "in", "source_path": 9, "prior": {"v": 1}}}]`, ft},
		{`[` + ok + `, {"unlinked_input": {"source_slot": "o", "target_path": "Y", "target_slot": 1}}]`, ft},
		{`[{"unlinked_input": {"x": 1}}, {"relink": {"y": 1}}, ` + ok + `]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "t": "kitControl:NumericConst", "n": "bad name"}]}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "t": "kitControl:NumericConst"}]}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": [5]}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": "ab"}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": 7}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "t": "baja:Folder", "n": "K", "s": {"a": 1}}]}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "n": "Top", "v": 1, "s": [{"nm": "p", "t": "bacnet:BacnetAddress", "n": "addr"}, {"nm": "p", "t": "baja:UnrestrictedFolder", "n": "U"}, {"nm": "p", "t": 5, "n": "num"}]}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Z", "b": [1]}]`, ft}, {`[{"nm": "a", "h": "3", "n": "W"}]`, ft},
		{`[{"nm": "a", "h": "3", "n": 7, "b": {"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "t": "kitControl:X", "n": "K"}]}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "a|b", "b": {"nm": "p", "t": "baja:Folder"}}, ` + relink + `]`, ft},
		{`[{"nm": "a", "h": "3", "b": {"nm": "p", "t": "baja:Folder"}}]`, ft},
		{`[{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p"}}]`, ft},
		{`[` + ok + `, {"nm": "v", "h": "3", "n": "Q"}]`, ft},
		{`[{"nm": "x", "h": "3", "n": "Q", "b": {"nm": "p", "t": "baja:Double", "v": "1"}}]`, ft},
		{`[{"nm": "v", "h": "3", "n": ["Q"]}]`, ft}, {`[{"nm": "v", "h": "3", "n": 5}]`, ft},
		{`[` + ok + `, ` + ok + `]`, `{"` + fold + `": "3", "station:|slot:/Folder/B": "3"}`},
	} {
		emit(rcase{journal: odd(c[0], c[1]), replies: `[[{"nn": "X"}], [{"nn": "X1"}]]`,
			after: `{"` + fold + `@2": {"X": {"t": "baja:Folder"}, "Y": {"t": "baja:Folder"}, "Q": {"v": "1"}}}`})
	}
	// the journal's own refusals: another station, a rollback, in-doubt, rolled back before
	for _, j := range [][]string{
		batch(rbid, "n4_set_slot", `[]`, `[{"nm": "v", "h": "3", "n": "x"}]`, ft, `"OtherStation"`, "verified", ""),
		batch(rbid, "n4_set_slot", `[]`, `[{"nm": "v", "h": "3", "n": "x"}]`, ft, `null`, "verified", ""),
		batch(rbid, "n4_rollback", `[]`, `[{"nm": "v", "h": "3", "n": "x"}]`, ft, `"FakeStation"`, "verified", `, "rollback_of": "`+strings.Repeat("c", 32)+`"`),
		batch(rbid, "n4_set_slot", `[]`, `[{"nm": "v", "h": "3", "n": "x"}]`, ft, `"FakeStation"`, "verified", "")[:1],
		append(append([]string{}, full...), batch(strings.Repeat("e", 32), "n4_rollback", `[]`, `[]`, ft, `"FakeStation"`, "failed", `, "rollback_of": "`+rbid+`"`)...),
		append(append([]string{}, full...), batch(strings.Repeat("e", 32), "n4_rollback", `[]`, `[]`, ft, `"FakeStation"`, "verified", `, "rollback_of": "`+rbid+`"`)...),
		append(append([]string{}, full...), `{"batch_id": "`+strings.Repeat("e", 32)+`", "phase": "intent", "rollback_of": "`+rbid+`"}`,
			`{"batch_id": "`+strings.Repeat("e", 32)+`", "phase": "result", "verdict": "failed", "readback_failed": true, "accepted": [1]}`),
		append(append([]string{}, full...), `{"batch_id": [1], "phase": "intent"}`, `{"batch_id": "x", "phase": "intent", "rollback_of": {"a": 1}}`),
	} {
		emit(rcase{journal: j, after: grpAfter("Grp", nil)})
	}
	// slot restores: unchanged-since and the read-back of set and invoke-set batches
	calc := fold + "/Calc"
	setOps := `[{"nm": "s", "h": "10", "n": "out", "b": ` + statusBody("9.0", "0") + `}]`
	setInv := `[{"nm": "s", "h": "10", "n": "out", "b": ` + statusBody("1.0", "0") + `}]`
	for _, now := range []string{`"9.0"`, `"4.0"`, `"x y"`, `null`, `9`} {
		for _, back := range []string{`"1.0"`, `"2.0"`, `"z"`} {
			emit(rcase{journal: batch(rbid, "n4_set_slot", setOps, setInv, `{"`+calc+`": "10"}`, `"FakeStation"`, "verified", ""),
				before: `{"` + calc + `@1": {"": {"h": "10"}}, "` + calc + `@2": {"out": {"t": "baja:StatusNumeric"}, "out/value": {"v": ` + now + `}, "out/status": {"v": "0"}}}`,
				after:  `{"` + calc + `@2": {"out": {"t": "baja:StatusNumeric"}, "out/value": {"v": ` + back + `}}}`, replies: `[[]]`})
		}
	}
	for _, c := range []struct{ ops, inv, before string }{
		{setOps, setInv, `{"out": {"t": "baja:StatusBoolean"}}`},
		{setOps, setInv, `{}`},
		{setOps, setInv, `{"out": {"t": "baja:Double", "v": "9.0"}}`},
		{`[{"nm": "s", "h": "10", "n": "grp/sp", "b": ` + statusBody("9.0", "0") + `}]`, `[{"nm": "s", "h": "10", "n": "grp/sp", "b": ` + statusBody("1.0", "0") + `}]`,
			`{"grp/sp": {"t": "baja:StatusNumeric"}, "grp/sp/value": {"v": "9.0"}}`},
		{`[{"nm": "s", "h": "10", "n": "d", "b": {"nm": "p", "t": "baja:Double", "v": "2"}}, {"nm": "s", "h": "10", "n": "b", "b": {"nm": "p", "t": "baja:Boolean", "v": "true"}}, {"nm": "s", "h": "10", "n": "s", "b": {"nm": "p", "t": "baja:String", "v": ""}}]`,
			`[{"nm": "s", "h": "10", "n": "d", "b": {"nm": "p", "t": "baja:Double"}}, {"nm": "s", "h": "10", "n": "b", "b": {"nm": "p", "t": "baja:Boolean", "v": "false"}}, {"nm": "s", "h": "10", "n": "s", "b": {"nm": "p", "t": "baja:String", "v": "x"}}]`,
			`{"d": {"v": "2.0"}, "b": {"v": "true"}}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": {"t": "baja:StatusNumeric", "s": [{"n": "value", "v": null}]}}]`, setInv, `{"out": {"t": "baja:StatusNumeric"}}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": {"t": "baja:StatusNumeric", "s": [{"n": "value", "v": [1]}]}}]`, setInv, `{}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": {"t": 5}}]`, setInv, `{}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": {"v": 1}}]`, setInv, `{}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": {"t": "baja:StatusNumeric", "s": [{"n": ["v"]}]}}]`, setInv, `{}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": {"t": "baja:StatusNumeric", "s": "ab"}}]`, setInv, `{}`},
		{`[{"nm": "s", "h": ["10"], "n": "out", "b": {"t": "baja:Double"}}]`, setInv, `{}`},
		{`[{"nm": "s", "h": "10", "n": "out", "b": "x"}, 5, {"nm": "s"}]`, setInv, `{}`},
		{`"junk"`, setInv, `{}`}, {`7`, setInv, `{}`},
		{setOps, `[{"nm": "s", "h": "10", "n": "out", "b": [1]}]`, `{"out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "9.0"}}`},
		{setOps, `[{"nm": "s", "h": "10", "n": "out", "b": {"nm": "p"}}]`, `{"out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "9.0"}}`},
		{setOps, `[{"nm": "s", "h": "10", "n": "out", "b": {"t": 5}}]`, `{"out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "9.0"}}`},
		{`[{"nm": "s", "h": "10", "n": 5, "b": {"t": "baja:Double"}}]`, `[{"nm": "s", "h": "10", "n": 5, "b": {"t": "baja:Double"}}]`, `{}`},
		{`[{"ssc": "invokeAction", "arg": {"h": "10", "a": "set", "b": {"nm": "p", "t": "baja:Double", "v": "7.0"}}}]`,
			`[{"nm": "s", "h": "10", "n": "fallback", "b": ` + statusBody("0.0", "0") + `}]`, `{"fallback": {"t": "baja:StatusNumeric"}, "fallback/value": {"v": "7.0"}}`},
		{`[{"ssc": "invokeAction", "arg": {"h": "10", "a": "set", "b": {"nm": "p", "t": "baja:Double", "v": "7.0"}}}]`,
			`[{"nm": "s", "h": "10", "n": "fallback", "b": ` + statusBody("0.0", "0") + `}]`, `{"fallback": {"t": "baja:StatusNumeric"}, "fallback/value": {"v": "3.0"}}`},
		{`[{"ssc": "invokeAction", "arg": {"h": "10", "a": "set", "b": {"t": "baja:Boolean", "v": "true"}}}]`,
			`[{"nm": "s", "h": "10", "n": "fallback", "b": {"t": "baja:StatusBoolean", "s": [{"n": "value", "v": "false"}]}}]`, `{"fallback": {"t": "baja:StatusBoolean"}, "fallback/value": {"v": "true"}}`},
		{`[{"ssc": "invokeAction", "arg": {"h": "10", "a": "set", "b": 5}}, {"ssc": "x", "arg": {"a": "set"}}]`, setInv, `{}`},
	} {
		emit(rcase{journal: batch(rbid, "n4_set_slot", c.ops, c.inv, `{"`+calc+`": "10"}`, `"FakeStation"`, "verified", ""),
			before:  `{"` + calc + `@1": {"": {"h": "10"}}, "` + calc + `": ` + c.before + `}`,
			after:   `{"` + calc + `": {"out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "1.0"}, "d": {}, "b": {"v": "false"}, "s": {"v": "x"}, "fallback": {"t": "baja:StatusNumeric"}, "fallback/value": {"v": "0.0"}}}`,
			replies: `[[], [], []]`})
	}
	// removals: a create's and a link's inverse
	for _, after := range []string{`{"Pump": {}}`, `{}`, `{"error": "boom"}`} {
		emit(rcase{journal: batch(rbid, "n4_create_component", `[]`, `[{"nm": "v", "h": "3", "n": "Pump"}]`, ft, `"FakeStation"`, "verified", ""),
			after: `{"` + fold + `": ` + after + `}`, replies: `[[]]`})
	}
	return out
}
