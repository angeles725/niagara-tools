package writeops_test

// Verifier probes of n4_ack_alarm (independent of the writer's tests).

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/retro"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// scopedRig is newAckRig with the scopes and state dir given.
func scopedRig(t *testing.T, client session.Client, dir string, scopes ...string) *ackRig {
	t.Helper()
	r := &ackRig{dir: dir}
	state, err := plan.NewState(scopes, r.dir, 300, 200)
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

func (r *ackRig) tool(t *testing.T, name string, kv ...any) (string, bool, *pyjson.Object) {
	t.Helper()
	reply := r.srv.Dispatch(obj("id", 1, "method", "tools/call", "params", obj("name", name, "arguments", obj(kv...))))
	res, _ := reply.Get("result")
	content := get(res.(*pyjson.Object), "content").([]any)
	sc, _ := get(res.(*pyjson.Object), "structuredContent").(*pyjson.Object)
	return get(content[0].(*pyjson.Object), "text").(string), get(res.(*pyjson.Object), "isError") == true, sc
}

func TestZZProbeScopeBoundary(t *testing.T) {
	cases := []struct {
		source string
		ok     bool
	}{
		{"slot:/Services", true},
		{"local:|station:|slot:/Services/", true},
		{"slot:/Services/A/B/C", true},
		{"slot:/ServicesX/A", false},
		{"local:|station:|slot:/ServicesX", false},
		{"slot:/Services/../Drivers/X", false},
		{"local:|station:|slot:/Services/A|view:x", false},
		{"station:|slot:/Services/A", false},
		{"", false},
		{"slot:/", false},
	}
	for _, c := range cases {
		db := newAlarmDB()
		db.source[u1] = c.source
		r := newAckRig(t, db)
		text, isErr, _ := r.call(t, "uuids", []any{u1})
		if isErr == c.ok || len(db.calls) != 0 {
			t.Errorf("source %q: isErr=%v text=%s", c.source, isErr, text)
		}
		t.Logf("source %q -> err=%v %s", c.source, isErr, text)
	}
	// several scopes
	db := newAlarmDB()
	db.source[u1], db.source[u2] = "slot:/ImplA/x", "slot:/ImplB"
	r := scopedRig(t, db, filepath.Join(t.TempDir(), "s"), "station:|slot:/ImplA", "station:|slot:/ImplB")
	if text, isErr, _ := r.call(t, "uuids", []any{u1, u2}); isErr {
		t.Errorf("two scopes: %s", text)
	}
	db.source[u2] = "slot:/ImplAB"
	if text, isErr, _ := r.call(t, "uuids", []any{u1, u2}); !isErr || !strings.Contains(text, "outside the write scope: "+u2) {
		t.Errorf("ImplAB: %s", text)
	}
	// no scope at all: refused before any read
	db = newAlarmDB()
	r = scopedRig(t, db, filepath.Join(t.TempDir(), "s"))
	if text, isErr, _ := r.call(t, "uuids", []any{u1}); !isErr || len(db.ords) != 0 {
		t.Errorf("no scope: %v %s ords=%d", isErr, text, len(db.ords))
	} else {
		t.Logf("no scope -> %s", text)
	}
}

func TestZZProbeUUIDForms(t *testing.T) {
	for _, c := range []struct {
		v  any
		ok bool
	}{
		{strings.ToUpper(u1), true},
		{" " + u1, false},
		{"{" + u1 + "}", false},
		{strings.ReplaceAll(u1, "-", ""), false},
		{u1 + "\n", false},
		{nil, false},
	} {
		db := newAlarmDB()
		text, isErr, _ := newAckRig(t, db).call(t, "uuids", []any{c.v})
		if isErr == c.ok {
			t.Errorf("%q: %s", c.v, text)
		}
	}
	// same uuid in two cases is a repeat
	if text, isErr, _ := newAckRig(t, newAlarmDB()).call(t, "uuids", []any{u1, strings.ToUpper(u1)}); !isErr || !strings.Contains(text, "given twice") {
		t.Errorf("case repeat: %s", text)
	}
	// uuids not a list / notes not a string / missing uuids
	for _, kv := range [][]any{{"uuids", u1}, {"uuids", []any{u1}, "notes", 5}, {}} {
		db := newAlarmDB()
		r := newAckRig(t, db)
		reply := r.srv.Dispatch(obj("id", 1, "method", "tools/call", "params", obj("name", "n4_ack_alarm", "arguments", obj(kv...))))
		if len(db.ords) != 0 || len(db.calls) != 0 {
			t.Errorf("%v reached the station", kv)
		}
		t.Logf("%v -> %.300s", kv, pyjson.Repr(reply))
	}
	// 100 distinct uuids, every one existing and unacked, in scope: planned
	db := newAlarmDB()
	var ids []any
	for i := 0; i < 100; i++ {
		id := "bb000000-0000-4000-8000-" + strings.Repeat("0", 9) + pyjson.Repr(100+i)
		db.state[id], db.source[id] = "unacked", "slot:/Services/X"
		ids = append(ids, id)
	}
	if text, isErr, _ := newAckRig(t, db).call(t, "uuids", ids); isErr {
		t.Errorf("100: %s", text)
	}
	ids = append(ids, u1)
	if text, isErr, _ := newAckRig(t, db).call(t, "uuids", ids); !isErr || !strings.Contains(text, "got 101") {
		t.Errorf("101: %s", text)
	}
}

func TestZZProbeTokenAcrossStateChangeAndScope(t *testing.T) {
	// acked by someone else between dry run and execute: refused, nothing sent
	db := newAlarmDB()
	r := newAckRig(t, db)
	_, _, dry := r.call(t, "uuids", []any{u1})
	db.state[u1] = "acked"
	text, isErr, _ := r.call(t, "uuids", []any{u1}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	if !isErr || len(db.calls) != 0 || !strings.Contains(text, "already acknowledged") {
		t.Errorf("state change: %s calls=%q", text, db.calls)
	}
	// notes changed: token mismatch
	db = newAlarmDB()
	r = newAckRig(t, db)
	_, _, dry = r.call(t, "uuids", []any{u1}, "notes", "a")
	text, isErr, _ = r.call(t, "uuids", []any{u1}, "notes", "b", "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	if !isErr || len(db.calls) != 0 {
		t.Errorf("notes changed: %s", text)
	}
	t.Logf("notes changed -> %s", text)
	// token reuse after success
	db = newAlarmDB()
	r = newAckRig(t, db)
	_, _, dry = r.call(t, "uuids", []any{u1})
	r.call(t, "uuids", []any{u1}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	db.state[u1] = "unacked" // even if it were unacked again, the token is spent
	text, isErr, _ = r.call(t, "uuids", []any{u1}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	if !isErr || len(db.calls) != 1 {
		t.Errorf("reuse: %s calls=%q", text, db.calls)
	}
	t.Logf("reuse -> %s", text)
	// token from a wider-scope server, executed on a narrower-scope server with the same state dir
	dir := filepath.Join(t.TempDir(), "s")
	db = newAlarmDB()
	wide := scopedRig(t, db, dir, "station:|slot:/Services", "station:|slot:/Drivers")
	_, isErr, dry = wide.call(t, "uuids", []any{u4})
	if isErr {
		t.Fatal("wide dry run refused")
	}
	narrow := scopedRig(t, db, dir, "station:|slot:/Services")
	text, isErr, _ = narrow.call(t, "uuids", []any{u4}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	if !isErr || len(db.calls) != 0 || !strings.Contains(text, "outside the write scope") {
		t.Errorf("narrow restart: %s calls=%q", text, db.calls)
	}
	t.Logf("narrow restart -> %s", text)
}

// replyDB answers ackAlarms with a fixed reply or error.
type replyDB struct {
	*alarmDB
	reply any
	err   error
	ack   map[string]bool // ids actually acked
}

func (d *replyDB) Call(channel, key string, body any) (any, error) {
	d.calls = append(d.calls, channel+" "+key)
	ids, _ := body.(*pyjson.Object).Get("ids")
	for _, id := range ids.([]any) {
		if d.ack == nil || d.ack[id.(string)] {
			d.state[id.(string)], d.user[id.(string)] = "ackPending", "admin"
		}
	}
	return d.reply, d.err
}

func TestZZProbeRepliesAndFailures(t *testing.T) {
	for _, reply := range []any{nil, "x", obj("a", 1), []any{}} {
		d := &replyDB{alarmDB: newAlarmDB(), reply: reply}
		_, out := newAckRig(t, d).ackBoth(t, "uuids", []any{u1})
		want := "mismatch"
		if l, ok := reply.([]any); ok && len(l) == 0 {
			want = "verified"
		}
		if get(out, "verdict") != want {
			t.Errorf("reply %s: %s", pyjson.Repr(reply), pyjson.Repr(out))
		}
		t.Logf("reply %s -> %s observed=%s", pyjson.Repr(reply), get(out, "verdict"), pyjson.Repr(get(out, "observed")))
	}
	// partial ack
	d := &replyDB{alarmDB: newAlarmDB(), reply: []any{}, ack: map[string]bool{u1: true}}
	_, out := newAckRig(t, d).ackBoth(t, "uuids", []any{u1, u2})
	if get(out, "verdict") != "mismatch" || !strings.Contains(pyjson.Repr(get(out, "observed")), "'still_unacked': ['"+u2+"']") {
		t.Errorf("partial: %s", pyjson.Repr(out))
	}
	// station refusal, nothing applied: refused and journaled as refused
	d = &replyDB{alarmDB: newAlarmDB(), err: &box.Error{Msg: "no permission", Station: true}, ack: map[string]bool{}}
	r := newAckRig(t, d)
	_, _, dry := r.call(t, "uuids", []any{u1})
	text, isErr, _ := r.call(t, "uuids", []any{u1}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	t.Logf("station refusal -> %v %s", isErr, text)
	view, _ := journal.New(r.dir).ListBatches(10)
	t.Logf("list_batches after refusal: %s", pyjson.Repr(view))
	if !isErr || !strings.Contains(pyjson.Repr(view), "refused") {
		t.Errorf("refused not journaled: %s", pyjson.Repr(view))
	}
	// station refusal but the record got acked anyway: in-doubt, not refused
	d = &replyDB{alarmDB: newAlarmDB(), err: &box.Error{Msg: "late", Station: true}}
	r = newAckRig(t, d)
	_, _, dry = r.call(t, "uuids", []any{u1})
	text, _, _ = r.call(t, "uuids", []any{u1}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	if !strings.Contains(strings.ToLower(text), "doubt") {
		t.Errorf("applied-but-error should be in doubt: %s", text)
	}
	t.Logf("applied+error -> %s", text)
	// transport error: in doubt
	d = &replyDB{alarmDB: newAlarmDB(), err: errors.New("connection reset"), ack: map[string]bool{}}
	r = newAckRig(t, d)
	_, _, dry = r.call(t, "uuids", []any{u1})
	text, _, _ = r.call(t, "uuids", []any{u1}, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))
	if !strings.Contains(strings.ToLower(text), "doubt") {
		t.Errorf("transport: %s", text)
	}
	t.Logf("transport -> %s", text)
	view, _ = journal.New(r.dir).ListBatches(10)
	t.Logf("list_batches after in-doubt: %s", pyjson.Repr(view))
	d2, err := retro.Draft(r.dir, "FakeStation", "", nil)
	if err != nil {
		t.Errorf("retro draft: %v", err)
	} else {
		md := pyjson.Str(get(d2, "markdown"))
		i := strings.Index(md, "audit")
		if i < 0 {
			i = 0
		}
		for _, l := range strings.Split(md, "\n") {
			if strings.Contains(l, "None") || strings.Contains(l, "station call failed") || strings.Contains(l, "ack") {
				t.Logf("retro: %s", l)
			}
		}
		_ = i
	}
}

func TestZZProbeRollbackAndListOfAnAckBatch(t *testing.T) {
	db := newAlarmDB()
	r := newAckRig(t, db)
	_, out := r.ackBoth(t, "uuids", []any{u1}, "notes", "n")
	id := get(out, "batch_id")
	text, isErr, sc := r.tool(t, "n4_rollback", "batch_id", id)
	t.Logf("rollback dry -> err=%v %s", isErr, text)
	if !isErr {
		text2, isErr2, _ := r.tool(t, "n4_rollback", "batch_id", id, "dry_run", false, "confirmation_token", get(sc, "confirmation_token"))
		t.Logf("rollback exec -> err=%v %s calls=%q", isErr2, text2, db.calls)
		if len(db.calls) != 1 {
			t.Errorf("rollback sent something: %q", db.calls)
		}
	}
	view, err := journal.New(r.dir).ListBatches(10)
	t.Logf("list_batches -> %v %s", err, pyjson.Repr(view))
	d, err := retro.Draft(r.dir, "FakeStation", "", nil)
	if err != nil {
		t.Errorf("retro: %v", err)
	} else {
		t.Logf("retro ok, %d bytes", len(pyjson.Str(get(d, "markdown"))))
	}
}

// quotedDB answers with RFC-4180 quoting, so a source may hold a newline or a comma.
type quotedDB struct{ *alarmDB }

func (d quotedDB) GetOrd(ord string, _ float64, _ int) (string, bool, error) {
	d.ords = append(d.ords, ord)
	var b strings.Builder
	b.WriteString("Uuid,Ack State,User,Source\n")
	for _, m := range uuidInOrd.FindAllStringSubmatch(ord, -1) {
		st, ok := d.state[m[1]]
		if !ok || strings.Contains(ord, "ackState = 'unacked'") && st != "unacked" {
			continue
		}
		b.WriteString(m[1] + "," + st + "," + d.user[m[1]] + ",\"" + strings.ReplaceAll(d.source[m[1]], "\"", "\"\"") + "\"\n")
	}
	return b.String(), false, nil
}

func TestZZProbeMultiOrdSource(t *testing.T) {
	for _, src := range []string{
		"local:|station:|slot:/Services/A\nlocal:|station:|slot:/Drivers/B",
		"local:|station:|slot:/Services/A;local:|station:|slot:/Drivers/B",
		"local:|station:|slot:/Services/A,local:|station:|slot:/Drivers/B",
		"local:|station:|slot:/Services/A Drivers",
		"slot:/Services/A\nslot:/Drivers/B",
		"slot:/Services/A; slot:/Drivers/B",
		"local:|station:|slot:/Services/A; slot:/Drivers/B",
		"slot:/Services/A; local:|station:|slot:/Drivers/B",
		"slot:/Services/A\r\nslot:/Drivers/B",
	} {
		db := newAlarmDB()
		db.source[u1] = src
		text, isErr, _ := newAckRig(t, quotedDB{db}).call(t, "uuids", []any{u1})
		t.Logf("source %q -> err=%v %.200s", src, isErr, text)
		if !isErr || !strings.Contains(text, "source is not a slot path") || len(db.calls) != 0 {
			t.Errorf("multi-ORD source %q was not refused as not a slot path: %.200s", src, text)
		}
	}
}
