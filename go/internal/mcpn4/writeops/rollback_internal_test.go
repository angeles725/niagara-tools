package writeops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const tFolder = "station:|slot:/Folder"

// tnode is a component or slot of the in-memory station the rollback twins run on.
type tnode struct {
	name, t, h string
	v          any
	kids       []*tnode
}

func (n *tnode) child(name string) *tnode {
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
	}
	return nil
}

// tstation models what the rollback needs of a station (the Python FakeStation's rules): adds
// of one component each (a writable gets its frozen proxyExt), removes, checkLinks, and loads
// flattened to a depth. Faults are injected per call.
type tstation struct {
	root     *tnode
	byH      map[string]*tnode
	next     int
	sent     []any
	failLoad func(ord string, depth int) error
	onSync   func(op *pyjson.Object)
}

func newTStation() *tstation {
	s := &tstation{root: &tnode{t: "baja:Station", h: "2"}, byH: map[string]*tnode{}, next: 100}
	folder := &tnode{name: "Folder", t: "baja:Folder", h: "3"}
	s.root.kids = []*tnode{folder}
	s.byH["2"], s.byH["3"] = s.root, folder
	return s
}

func (s *tstation) add(parent *tnode, name, t string) *tnode {
	s.next++
	n := &tnode{name: name, t: t, h: fmt.Sprint(s.next)}
	parent.kids = append(parent.kids, n)
	s.byH[n.h] = n
	if strings.HasSuffix(t, "Writable") {
		s.add(n, "proxyExt", "control:NullProxyExt")
	}
	return n
}

func (s *tstation) resolve(o string) *tnode {
	n := s.root
	for _, part := range strings.Split(strings.TrimPrefix(o, "station:|slot:/"), "/") {
		if part != "" && n != nil {
			n = n.child(part)
		}
	}
	return n
}

func (s *tstation) LoadTree(o string, depth int, _ box.LoadOptions) (*pyjson.Object, error) {
	if s.failLoad != nil {
		if err := s.failLoad(o, depth); err != nil {
			return nil, err
		}
	}
	n := s.resolve(o)
	if n == nil {
		return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + o}
	}
	out := obj()
	var walk func(n *tnode, path string, left int) *pyjson.Object
	walk = func(n *tnode, path string, left int) *pyjson.Object { // a raw node, as box flattens it
		e := obj()
		if path != "" {
			e.Set("n", n.name)
		}
		for _, kv := range [][2]string{{"h", n.h}, {"t", n.t}} {
			if kv[1] != "" {
				e.Set(kv[0], kv[1])
			}
		}
		if n.v != nil {
			e.Set("v", n.v)
		}
		out.Set(path, e)
		var kids []any
		for _, k := range n.kids {
			if left > 0 {
				p := k.name
				if path != "" {
					p = path + "/" + k.name
				}
				kids = append(kids, walk(k, p, left-1))
			}
		}
		if kids != nil {
			e.Set("s", kids)
		}
		return e
	}
	walk(n, "", depth)
	return out, nil
}

func (s *tstation) fill(n *tnode, body *pyjson.Object) {
	n.v = get(body, "v")
	kids, _ := get(body, "s").([]any)
	for _, k := range kids {
		kb := k.(*pyjson.Object)
		t, _ := get(kb, "t").(string)
		c := &tnode{name: get(kb, "n").(string), t: t}
		s.fill(c, kb)
		n.kids = append(n.kids, c)
	}
}

func (s *tstation) Sync(op any) (any, error) {
	s.sent = append(s.sent, op)
	o := op.(*pyjson.Object)
	parent, name := s.byH[get(o, "h").(string)], get(o, "n").(string)
	var reply any = []any{}
	switch get(o, "nm") {
	case "a":
		if parent.child(name) != nil {
			return nil, &box.Error{Kind: box.KindBox, Msg: "Unable to process request."}
		}
		body := get(o, "b").(*pyjson.Object)
		n := s.add(parent, name, get(body, "t").(string))
		s.fill(n, body)
		reply = []any{obj("id", "a", "nn", name)}
	case "v":
		for i, k := range parent.kids {
			if k.name == name {
				parent.kids = append(parent.kids[:i], parent.kids[i+1:]...)
			}
		}
	}
	if s.onSync != nil {
		s.onSync(o)
	}
	return reply, nil
}

func (s *tstation) SSC(key string, arg any) (any, error) {
	s.sent = append(s.sent, arg)
	a := arg.(*pyjson.Object)
	tgt := s.byH[get(a, "t").(string)]
	name := "Link"
	for i := 1; tgt.child(name) != nil; i++ {
		name = fmt.Sprintf("Link%d", i)
	}
	l := &tnode{name: name, t: "baja:Link"}
	for _, kv := range [][2]any{{"sourceOrd", "h:" + get(a, "s").(string)}, {"sourceSlotName", get(a, "ss")},
		{"targetSlotName", get(a, "ts")}} {
		l.kids = append(l.kids, &tnode{name: kv[0].(string), t: "baja:String", v: kv[1]})
	}
	tgt.kids = append(tgt.kids, l)
	return []any{[]any{obj("v", true, "r", nil, "s", name)}}, nil
}

func (s *tstation) ClearHandles() {}
func (s *tstation) Close()        {}

// failingJournal refuses the entries of one phase, as an OSError would.
type failingJournal struct {
	*journal.Journal
	phase string
}

func (f failingJournal) Append(e *pyjson.Object) error {
	if get(e, "phase") == f.phase {
		return errors.New("disk full")
	}
	return f.Journal.Append(e)
}

// trig is a write server on a tstation, connected and identity-verified.
type trig struct {
	st    *tstation
	ctx   *session.Context
	state *plan.State
	srv   *jsonrpc.Server
	dir   string
}

func newTRig(t *testing.T, maxWrites int) *trig {
	t.Helper()
	r := &trig{st: newTStation(), dir: filepath.Join(t.TempDir(), "state")}
	state, err := plan.NewState([]string{tFolder}, r.dir, 300, maxWrites)
	if err != nil {
		t.Fatal(err)
	}
	r.state = state
	r.ctx = session.New(session.Config{AllowWrites: true})
	r.ctx.Session = session.NewSession(r.st, "FakeStation", "http://fake.invalid", "2", "4.14.0.162", "")
	r.srv = jsonrpc.New(registry.Default())
	r.srv.Ctx = r.ctx
	Register(r.srv, &plan.Engine{Ctx: r.ctx, State: state})
	return r
}

// call is one tools/call through the registered handler: text, isError, structuredContent.
func (r *trig) call(t *testing.T, name string, args *pyjson.Object) (string, bool, *pyjson.Object) {
	t.Helper()
	reply := r.srv.Dispatch(obj("id", 1, "method", "tools/call", "params", obj("name", name, "arguments", args)))
	res, ok := reply.Get("result")
	if !ok {
		t.Fatalf("%s: %s", name, pyjson.Repr(reply))
	}
	text := get(get(res.(*pyjson.Object), "content").([]any)[0].(*pyjson.Object), "text").(string)
	out, _ := get(res.(*pyjson.Object), "structuredContent").(*pyjson.Object)
	return text, get(res.(*pyjson.Object), "isError") == true, out
}

// write is a dry run and its execution; it fails the test when the dry run is refused.
func (r *trig) write(t *testing.T, name string, args *pyjson.Object) (string, bool, *pyjson.Object) {
	t.Helper()
	text, isErr, dry := r.call(t, name, args)
	if isErr {
		t.Fatalf("%s dry run: %s", name, text)
	}
	full := obj("dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	for _, k := range args.Keys() {
		full.Set(k, get(args, k))
	}
	return r.call(t, name, full)
}

func (r *trig) phases(t *testing.T) []string {
	data, _ := os.ReadFile(filepath.Join(r.dir, "journal.jsonl"))
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		v, err := pyjson.Unmarshal([]byte(l))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, get(v.(*pyjson.Object), "phase").(string))
	}
	return out
}

// group is Folder/Grp: Src -> Tgt.in10 inside it, a NumericWritable whose frozen proxyExt is
// configured, a BooleanWritable and Sub/Deep, so the rollback re-creates nested components,
// skips frozen children, relinks and reports a frozen gap.
func (r *trig) group() string {
	g := r.st.add(r.st.byH["3"], "Grp", "baja:Folder")
	src := r.st.add(g, "Src", "kitControl:NumericConst")
	tgt := r.st.add(g, "Tgt", "kitControl:NumericConst")
	tgt.kids = append(tgt.kids, &tnode{name: "Link", t: "baja:Link", kids: []*tnode{
		{name: "sourceOrd", t: "baja:String", v: "h:" + src.h}, {name: "sourceSlotName", t: "baja:String", v: "out"},
		{name: "targetSlotName", t: "baja:String", v: "in10"}}})
	sp := r.st.add(g, "Sp", "control:NumericWritable")
	sp.child("proxyExt").kids = []*tnode{{name: "tuningPolicyName", t: "baja:String", v: "Fast"}}
	r.st.add(g, "En", "control:BooleanWritable")
	r.st.add(r.st.add(g, "Sub", "baja:Folder"), "Deep", "kitControl:NumericConst")
	return "Grp"
}

func (r *trig) removed(t *testing.T) string {
	text, isErr, out := r.write(t, "n4_remove_component", obj("parent_ord", tFolder, "name", r.group()))
	if isErr {
		t.Fatal(text)
	}
	return get(out, "batch_id").(string)
}

// The registered n4_rollback re-creates the subtree one add per component, journaling each
// before it is sent, skips frozen children, relinks, counts every op against the budget and
// reports the frozen gap: the verdict is partial.
func TestRollbackOfARemoveRecreatesTypeSlotsAnnotationAndLinks(t *testing.T) {
	r := newTRig(t, 200)
	bid := r.removed(t)
	var phasesAtSend [][]string
	r.st.onSync = func(*pyjson.Object) { phasesAtSend = append(phasesAtSend, r.phases(t)) }
	before := r.state.WritesExecuted
	text, isErr, out := r.write(t, "n4_rollback", obj("batch_id", bid))
	if isErr || get(out, "verdict") != "partial" {
		t.Fatalf("%s %s", text, pyjson.Repr(out))
	}
	// 1 batch + 6 nested adds (Src, Tgt, Sp, En, Sub, Deep; the two frozen proxyExts are
	// skipped) + 1 relink
	if got := r.state.WritesExecuted - before; got != 1+6+1 {
		t.Errorf("writes executed %d, want 8", got)
	}
	want := []string{"intent", "component-intent", "component-intent", "component-intent", "component-intent",
		"component-intent", "component-intent", "relink-intent", "result"}
	if got := r.phases(t)[2:]; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("phases %v, want %v", got, want)
	}
	for i, p := range phasesAtSend[1:] { // each nested add is journaled before it is sent
		if n := strings.Count(strings.Join(p, ","), "component-intent"); n != i+1 {
			t.Errorf("add %d sent with %d component intents on disk", i+1, n)
		}
	}
	frozen := pyjson.Repr(get(out, "frozen_children_not_restored"))
	if strings.Count(frozen, "'n': 'proxyExt'") != 2 || !strings.Contains(pyjson.Repr(get(out, "frozen_config_not_restored")), "tuningPolicyName") {
		t.Errorf("frozen: %s / %s", frozen, pyjson.Repr(get(out, "frozen_config_not_restored")))
	}
	if got := pyjson.Repr(get(out, "relinks")); got != "{'restored': 1, 'skipped': 0}" {
		t.Errorf("relinks %s", got)
	}
}

// A budget too small for the components and relinks is refused before anything is sent.
func TestTheComponentBudgetIsCheckedBeforeAnythingIsSent(t *testing.T) {
	r := newTRig(t, 10)
	bid := r.removed(t)
	sent := len(r.st.sent)
	text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", bid))
	if !isErr || !strings.Contains(text, "needs 10 write(s) (1 batch + 8 component(s) + 1 relink(s)) but only 9 remain (--max-writes 10)") || len(r.st.sent) != sent {
		t.Fatalf("%v %s", isErr, text)
	}
}

// A component or relink intent that cannot be journaled stops the batch in-doubt, names the
// components already created and sends nothing more; the audit line is an error of the batch.
func TestAComponentOrRelinkIntentThatCannotBeWrittenStopsInDoubt(t *testing.T) {
	for _, c := range []struct{ phase, says string }{
		{"component-intent", "the component intent could not be written, nothing more was sent"},
		{"relink-intent", "the relink intent could not be written (components were re-created, links were not)"},
	} {
		r := newTRig(t, 200)
		bid := r.removed(t)
		_, _, dry := r.call(t, "n4_rollback", obj("batch_id", bid))
		r.state.Journal = failingJournal{journal.New(r.dir), c.phase}
		sent := len(r.st.sent)
		text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", bid, "dry_run", false,
			"confirmation_token", get(dry, "confirmation_token")))
		if !isErr || !strings.Contains(text, c.says) || !strings.Contains(text, "in-doubt") ||
			!strings.Contains(text, "Components already created by this batch (remove the top-level one to clean up): "+tFolder+"/Grp") {
			t.Fatalf("%s: %s", c.phase, text)
		}
		links := 0
		for _, op := range r.st.sent[sent:] {
			if get(op.(*pyjson.Object), "ss") != nil {
				links++
			}
		}
		if adds := len(r.st.sent) - sent - links; links != 0 || c.phase == "component-intent" && adds != 1 {
			t.Errorf("%s: %d adds and %d links sent", c.phase, adds, links)
		}
		audit, _ := os.ReadFile(filepath.Join(r.dir, "audit.jsonl"))
		lines := strings.Split(strings.TrimSpace(string(audit)), "\n")
		if last := lines[len(lines)-1]; !strings.Contains(last, `"outcome": "error"`) || strings.Contains(last, `"batch_id": null`) {
			t.Errorf("%s: audit %s", c.phase, last)
		}
	}
}

// A load of a restored component for its relinks that fails leaves the batch in-doubt, listing
// the components created.
func TestAFailingRelinkLoadIsInDoubtNotACrash(t *testing.T) {
	r := newTRig(t, 200)
	bid := r.removed(t)
	r.st.failLoad = func(o string, depth int) error {
		if depth == SnapshotDepth && strings.HasSuffix(o, "/Grp") && len(r.st.sent) > 2 {
			return &box.Error{Kind: box.KindBox, Msg: "boom"}
		}
		return nil
	}
	text, isErr, _ := r.write(t, "n4_rollback", obj("batch_id", bid))
	if !isErr || !strings.Contains(text, "in-doubt (the station may or may not have applied it; inspect it before retrying): boom. Components already created") ||
		!strings.Contains(text, tFolder+"/Grp/Sub/Deep") {
		t.Fatal(text)
	}
}

// A frozen child the read-back cannot load is unobserved, not lost: the gap carries the
// scrubbed read error and no `missing`, and the verdict is unverified.
func TestAFrozenCheckReadErrorMakesTheRollbackUnverifiedAndIsScrubbed(t *testing.T) {
	r := newTRig(t, 200)
	r.ctx.SetSecret("s3cret")
	bid := r.removed(t)
	r.st.failLoad = func(o string, depth int) error {
		if depth == FrozenCheckDepth && strings.HasSuffix(o, "/En") {
			return &box.Error{Kind: box.KindBox, Msg: "HTTP 401: credentials s3cret rejected"}
		}
		return nil
	}
	_, _, out := r.write(t, "n4_rollback", obj("batch_id", bid))
	gaps := pyjson.Repr(get(out, "frozen_config_not_restored"))
	if get(out, "verdict") != "unverified" || !strings.Contains(gaps, "'readback_error': 'HTTP 401: credentials *** rejected'") ||
		strings.Contains(gaps, "s3cret") || strings.Count(gaps, "'missing'") != 1 {
		t.Fatalf("%s %s", get(out, "verdict"), gaps)
	}
}

// A rollback the station accepted but whose read-back failed cannot be retried: it may
// already be applied.
func TestARollbackAcceptedButUnverifiedCannotBeRetried(t *testing.T) {
	r := newTRig(t, 200)
	_, _, out := r.write(t, "n4_create_component", obj("parent_ord", tFolder, "name", "Pump", "type", "kitControl:NumericConst"))
	bid := get(out, "batch_id").(string)
	r.st.failLoad = func(string, int) error {
		if len(r.st.sent) > 1 {
			return &box.Error{Kind: box.KindBox, Msg: "boom"}
		}
		return nil
	}
	_, _, back := r.write(t, "n4_rollback", obj("batch_id", bid))
	r.st.failLoad = nil
	text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", bid))
	if get(back, "verdict") != "failed" || !isErr || !strings.Contains(text, "the station accepted its ops but the read-back failed") {
		t.Fatalf("%s / %v %s", pyjson.Repr(back), isErr, text)
	}
}

// Today's scope covers every recorded target, every re-created component and both ends of
// every relink.
func TestCurrentScopeIsEnforcedOnEveryRecordedTarget(t *testing.T) {
	r := newTRig(t, 200)
	bid := r.removed(t)
	r.state.Scope, _ = ord.NewScope([]string{"station:|slot:/Elsewhere"})
	if text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", bid)); !isErr || !strings.Contains(text, "outside the write scope") {
		t.Fatal(text)
	}
	targets := obj(tFolder, "3")
	ops := []any{obj("nm", "a", "h", "3", "n", "Grp")}
	own := []any{obj("nm", "v", "h", "3", "n", "Grp")}
	relink := obj("source_path", "Src", "source_slot", "out", "target_path", "Tgt", "target_slot", "in10")
	comp := obj("top", 0, "parent_path", "", "n", "Src", "b", obj())
	for _, c := range []struct {
		scope             string
		components, links []any
	}{{tFolder + "/Grp/Src", nil, []any{relink}}, {tFolder + "/Grp/Tgt", []any{comp}, nil}} {
		r.state.Scope, _ = ord.NewScope([]string{c.scope})
		if err := checkRestoredEnds(r.state, targets, ops, own, c.components, c.links); err == nil ||
			!strings.Contains(err.Error(), "outside the write scope") {
			t.Errorf("scope %s: %v", c.scope, err)
		}
	}
}

// _frozen_check compares only the captured configuration: an omitted slot is at its default
// (also a whole Status slot), a differing one is listed, and a read error is not `missing`.
func TestTheFrozenCheckTreatsOmittedDefaultsAsEqualAndAReadErrorAsUnread(t *testing.T) {
	status := func(v string) *pyjson.Object {
		return obj("nm", "p", "n", "readValue", "t", "baja:StatusNumeric", "s", []any{
			obj("nm", "p", "n", "value", "v", v), obj("nm", "p", "n", "status", "v", "0")})
	}
	captured := obj("nm", "p", "t", "control:NullProxyExt", "s", []any{
		obj("nm", "p", "n", "status", "t", "baja:Status", "v", "0"), status("0.0"),
		obj("nm", "p", "n", "tuningPolicyName", "t", "baja:String", "v", "Fast")})
	st := newTStation()
	sp := st.add(st.byH["3"], "Sp", "control:NumericWritable")
	ext := sp.child("proxyExt")
	ext.kids = []*tnode{{name: "readValue", t: "baja:StatusNumeric"}, {name: "tuningPolicyName", t: "baja:String", v: "Fast"}}
	check := obj("path", tFolder+"/Sp/proxyExt", "b", captured)
	if gap, err := frozenCheck(st, check); gap != nil || err != nil {
		t.Fatalf("defaults: %s %v", pyjson.Repr(gap), err)
	}
	ext.kids[1].v = "Slow"
	ext.kids = ext.kids[1:] // readValue omitted whole: still at its default
	captured.Set("s", []any{status("21.5"), obj("nm", "p", "n", "tuningPolicyName", "t", "baja:String", "v", "Fast")})
	if gap, _ := frozenCheck(st, check); pyjson.Repr(get(gap, "slots")) != "['readValue', 'tuningPolicyName']" || get(gap, "missing") != false {
		t.Errorf("differs: %s", pyjson.Repr(gap))
	}
	st.failLoad = func(string, int) error { return &box.Error{Kind: box.KindBox, Msg: "HTTP 503 from station"} }
	gap, _ := frozenCheck(st, check)
	if _, missing := gap.Get("missing"); missing || get(gap, "readback_error") != "HTTP 503 from station" {
		t.Errorf("read error: %s", pyjson.Repr(gap))
	}
}
