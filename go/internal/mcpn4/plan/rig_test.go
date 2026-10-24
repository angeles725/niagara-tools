package plan_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const (
	folder = "station:|slot:/Folder"
	calc   = folder + "/Calc"
	sp     = folder + "/Sp"
)

// station is an in-memory station for the Go twins (the black-box suite keeps the Python
// FakeStation): each ORD loads as a flat node dict, a set op writes a slot of Calc (handle 10)
// or Sp (handle 11), and every op that reaches it is kept in sent.
type station struct {
	nodes            map[string]*pyjson.Object
	sent             []any
	sendErr, loadErr error
	failLoadsAfter   bool   // loads fail with loadErr only once something was sent
	onSend           func() // runs after an op changed the model
}

func node(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

func newStation() *station {
	return &station{nodes: map[string]*pyjson.Object{
		calc: node("", node("h", "10", "t", "kitControl:NumericConst"), "out", node("t", "baja:StatusNumeric"),
			"out/value", node("t", "baja:Double", "v", "1.0"), "out/status", node("t", "baja:Status", "v", "0")),
		sp: node("", node("h", "11", "t", "control:NumericWritable"))}}
}

func (s *station) LoadTree(ord string, _ int, _ box.LoadOptions) (*pyjson.Object, error) {
	if s.loadErr != nil && (!s.failLoadsAfter || len(s.sent) > 0) {
		return nil, s.loadErr
	}
	if n, ok := s.nodes[ord]; ok {
		return n, nil
	}
	return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + ord}
}

func (s *station) send(op any) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, op)
	return nil
}

// applied runs onSend once an op has changed the model.
func (s *station) applied() {
	if s.onSend != nil {
		s.onSend()
	}
}

func (s *station) SSC(key string, arg any) (any, error) {
	if err := s.send(arg); err != nil {
		return nil, err
	}
	a := arg.(*pyjson.Object)
	if action, _ := a.Get("a"); action == "set" {
		b, _ := a.Get("b")
		v, _ := b.(*pyjson.Object).Get("v")
		s.nodes[sp].Set("fallback", node("t", "baja:StatusNumeric"))
		s.nodes[sp].Set("fallback/value", node("t", "baja:Double", "v", v))
	}
	s.applied()
	return nil, nil
}

func (s *station) Sync(op any) (any, error) {
	if err := s.send(op); err != nil {
		return nil, err
	}
	o := op.(*pyjson.Object)
	n, _ := o.Get("n")
	b, _ := o.Get("b")
	nodes := s.nodes[calc]
	nodes.Set(n.(string), b)
	kids, _ := b.(*pyjson.Object).Get("s")
	for _, k := range kids.([]any) {
		name, _ := k.(*pyjson.Object).Get("n")
		nodes.Set(n.(string)+"/"+name.(string), k)
	}
	s.applied()
	return []any{}, nil
}

func (s *station) ClearHandles() {}
func (s *station) Close()        {}

// rig is a server with the write tools on st, connected and identity-verified.
type rig struct {
	st    *station
	ctx   *session.Context
	state *plan.State
	srv   *jsonrpc.Server
	dir   string
}

func newRig(t *testing.T, scopes ...string) *rig {
	t.Helper()
	if scopes == nil {
		scopes = []string{folder}
	}
	r := &rig{st: newStation(), dir: filepath.Join(t.TempDir(), "state")}
	state, err := plan.NewState(scopes, r.dir, 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	r.state = state
	r.ctx = session.New(session.Config{AllowWrites: true})
	r.ctx.Session = session.NewSession(r.st, "FakeStation", "http://fake.invalid", "2", "4.14.0.162", "")
	r.srv = jsonrpc.New(registry.Default())
	r.srv.Ctx = r.ctx
	writeops.Register(r.srv, &plan.Engine{Ctx: r.ctx, State: state})
	return r
}

// call is one tools/call: the text content, whether it is an error, and structuredContent.
func (r *rig) call(t *testing.T, name string, args *pyjson.Object) (string, bool, *pyjson.Object) {
	t.Helper()
	reply := r.srv.Dispatch(node("id", 1, "method", "tools/call", "params", node("name", name, "arguments", args)))
	res, ok := reply.Get("result")
	if !ok {
		t.Fatalf("%s: %s", name, pyjson.Repr(reply))
	}
	content, _ := res.(*pyjson.Object).Get("content")
	text, _ := content.([]any)[0].(*pyjson.Object).Get("text")
	isErr, _ := res.(*pyjson.Object).Get("isError")
	sc, _ := res.(*pyjson.Object).Get("structuredContent")
	out, _ := sc.(*pyjson.Object)
	return text.(string), isErr == true, out
}

// dry is a dry run that must succeed; it returns the token.
func (r *rig) dry(t *testing.T, name string, args *pyjson.Object) string {
	t.Helper()
	text, isErr, out := r.call(t, name, args)
	if isErr {
		t.Fatalf("%s dry run: %s", name, text)
	}
	tok, _ := out.Get("confirmation_token")
	return tok.(string)
}

// execute repeats args with dry_run=false and tok.
func (r *rig) execute(t *testing.T, name string, args *pyjson.Object, tok string) (string, bool, *pyjson.Object) {
	t.Helper()
	full := node("dry_run", false, "confirmation_token", tok)
	for _, k := range args.Keys() {
		v, _ := args.Get(k)
		full.Set(k, v)
	}
	return r.call(t, name, full)
}

// lines is every JSON line of a state file.
func (r *rig) lines(t *testing.T, name string) []*pyjson.Object {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		return nil
	}
	var out []*pyjson.Object
	for _, l := range strings.SplitAfter(strings.TrimSuffix(string(data), "\n"), "\n") {
		v, err := pyjson.Unmarshal([]byte(l))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, v.(*pyjson.Object))
	}
	return out
}

func setArgs() *pyjson.Object {
	return node("ord", calc, "slot", "out", "value", pyjson.FloatNumber(5.5), "value_type", "baja:StatusNumeric")
}

func get(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}
