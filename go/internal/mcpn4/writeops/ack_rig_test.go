package writeops_test

// The rig of the n4_ack_alarm tests: an in-memory alarm database answering the BQL reads on the
// alarm: ORD and the alarm channel's ackAlarms, behind the write engine.

import (
	"fmt"
	"path/filepath"
	"regexp"
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
	u1 = "aa110470-0000-4000-8000-000000000001"
	u2 = "aa110470-0000-4000-8000-000000000002"
	u3 = "aa110470-0000-4000-8000-000000000003"
)

var uuidInOrd = regexp.MustCompile(`uuid = '([^']+)'`)

// alarmDB is the station: each record's ack state tag and user; ords and calls keep what reached it.
type alarmDB struct {
	state, user map[string]string
	failClasses []any // what ackAlarms answers (the classes the user may not ack)
	noAck       bool  // ackAlarms leaves the records unchanged
	ords, calls []string
}

func newAlarmDB() *alarmDB {
	return &alarmDB{state: map[string]string{u1: "unacked", u2: "unacked", u3: "acked"},
		user: map[string]string{u3: "operator"}, failClasses: []any{}}
}

// GetOrd answers "select uuid, ackState, user" with the records the clause names that exist,
// and only the unacked ones when the clause also asks ackState = 'unacked'.
func (d *alarmDB) GetOrd(ord string, _ float64, _ int) (string, bool, error) {
	d.ords = append(d.ords, ord)
	var b strings.Builder
	b.WriteString("Uuid,Ack State,User\n")
	for _, m := range uuidInOrd.FindAllStringSubmatch(ord, -1) {
		st, ok := d.state[m[1]]
		if !ok || strings.Contains(ord, "ackState = 'unacked'") && st != "unacked" {
			continue
		}
		fmt.Fprintf(&b, "%s,%s,%s\n", m[1], strings.ToUpper(st[:1])+st[1:], d.user[m[1]])
	}
	return b.String(), false, nil
}

func (d *alarmDB) Call(channel, key string, body any) (any, error) {
	d.calls = append(d.calls, channel+" "+key+" "+pyjson.Repr(body))
	ids, _ := body.(*pyjson.Object).Get("ids")
	for _, id := range ids.([]any) {
		if !d.noAck && d.state[id.(string)] == "unacked" {
			d.state[id.(string)], d.user[id.(string)] = "ackPending", "admin"
		}
	}
	return d.failClasses, nil
}

func (d *alarmDB) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) { return nil, nil }
func (d *alarmDB) SSC(string, any) (any, error)                                  { panic("ack sends no ssc op") }
func (d *alarmDB) Sync(any) (any, error)                                         { panic("ack sends no sync op") }
func (d *alarmDB) ClearHandles()                                                 {}

// readsOnly is a station client that can read alarms but cannot Call a channel.
type readsOnly struct{ db *alarmDB }

func (r readsOnly) GetOrd(o string, t float64, m int) (string, bool, error) {
	return r.db.GetOrd(o, t, m)
}
func (r readsOnly) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) { return nil, nil }
func (r readsOnly) SSC(string, any) (any, error)                                  { panic("no ssc") }
func (r readsOnly) Sync(any) (any, error)                                         { panic("no sync") }
func (r readsOnly) ClearHandles()                                                 {}

// ackRig is a write server on client, connected and identity-verified, with its state dir.
type ackRig struct {
	srv *jsonrpc.Server
	dir string
}

func newAckRig(t *testing.T, client session.Client) *ackRig {
	t.Helper()
	r := &ackRig{dir: filepath.Join(t.TempDir(), "state")}
	state, err := plan.NewState([]string{"station:|slot:/Services"}, r.dir, 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	ctx := session.New(session.Config{AllowWrites: true})
	ctx.Session = session.NewSession(client, "FakeStation", "http://fake.invalid", "2", "4.14.0.162", "")
	r.srv = jsonrpc.New(registry.Default())
	r.srv.Ctx = ctx
	writeops.Register(r.srv, &plan.Engine{Ctx: ctx, State: state})
	return r
}

// call is one n4_ack_alarm call: the text, whether it is an error, and structuredContent.
func (r *ackRig) call(t *testing.T, kv ...any) (string, bool, *pyjson.Object) {
	t.Helper()
	reply := r.srv.Dispatch(obj("id", 1, "method", "tools/call", "params", obj("name", "n4_ack_alarm", "arguments", obj(kv...))))
	res, ok := reply.Get("result")
	if !ok {
		t.Fatalf("n4_ack_alarm: %s", pyjson.Repr(reply))
	}
	content := get(res.(*pyjson.Object), "content").([]any)
	sc, _ := get(res.(*pyjson.Object), "structuredContent").(*pyjson.Object)
	return get(content[0].(*pyjson.Object), "text").(string), get(res.(*pyjson.Object), "isError") == true, sc
}

// ackBoth is a dry run of kv, then its execution with the token.
func (r *ackRig) ackBoth(t *testing.T, kv ...any) (*pyjson.Object, *pyjson.Object) {
	t.Helper()
	text, isErr, dry := r.call(t, kv...)
	if isErr {
		t.Fatalf("dry run: %s", text)
	}
	text, isErr, out := r.call(t, append(kv, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))...)
	if isErr {
		t.Fatalf("execute: %s", text)
	}
	return dry, out
}

func (d *alarmDB) Close()  {}
func (r readsOnly) Close() {}
