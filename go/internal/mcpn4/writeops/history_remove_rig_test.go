package writeops_test

// The rig of the n4_remove_history tests: an in-memory history database answering the history
// extension read, the count of one history and the history channel's deleteHistories.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const (
	hTemp   = "/FakeStation/Temp"   // recorded by a disabled extension in scope
	hLive   = "/FakeStation/Live"   // recorded by an enabled extension in scope
	hOut    = "/FakeStation/Remote" // recorded by an extension outside the scope
	hOrphan = "/FakeStation/Orphan" // no extension records into it
	hGone   = "/FakeStation/Gone"   // its extension exists, the history does not
)

// historyDB is the station: the extension rows, the record count of each history that exists.
type historyDB struct {
	exts        []string
	records     map[string]int
	noDelete    bool // deleteHistories leaves the history
	cutExts     bool // the extension read hit its byte cap
	errAfter    bool // after the delete call, a count read answers HTTP 500
	ords, calls []string
}

func newHistoryDB() *historyDB {
	return &historyDB{records: map[string]int{hTemp: 42, hLive: 7, hOut: 3, hOrphan: 9},
		exts: []string{"slot:/Services/A/hist,false," + hTemp, "slot:/Services/B/hist,true," + hLive,
			"slot:/Drivers/X/hist,false," + hOut, "slot:/Services/C/hist,false," + hGone}}
}

func (d *historyDB) GetOrd(ord string, _ float64, _ int) (string, bool, error) {
	d.ords = append(d.ords, ord)
	if strings.Contains(ord, "history:HistoryExt") {
		return "Slot Path,Enabled,Id\n" + strings.Join(d.exts, "\n") + "\n", d.cutExts, nil
	}
	id := strings.TrimSuffix(strings.SplitN(strings.TrimPrefix(ord, "history:"), "|", 2)[0], "?")
	if d.errAfter && len(d.calls) > 0 {
		return "", false, &box.Error{Kind: box.KindBox, Msg: "HTTP 500 from station"}
	}
	n, ok := d.records[id]
	if !ok {
		// the shape a real ORD read gives (box.Client.mapError)
		return "", false, &box.Error{Kind: box.KindBox, Msg: "HTTP 404 from station"}
	}
	return fmt.Sprintf("count(toString)\n%d\n", n), false, nil
}

func (d *historyDB) Call(channel, key string, body any) (any, error) {
	d.calls = append(d.calls, channel+" "+key+" "+pyjson.Repr(body))
	for _, o := range body.([]any) {
		if !d.noDelete {
			delete(d.records, strings.TrimPrefix(o.(string), "history:"))
		}
	}
	return true, nil
}

func (d *historyDB) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) { return nil, nil }
func (d *historyDB) SSC(string, any) (any, error)                                  { panic("no ssc op") }
func (d *historyDB) Sync(any) (any, error)                                         { panic("no sync op") }
func (d *historyDB) ClearHandles()                                                 {}
func (d *historyDB) Close()                                                        {}

// histRig is a write server on db with the scope station:|slot:/Services.
func histRig(t *testing.T, db *historyDB) *ackRig {
	t.Helper()
	return scopedRig(t, db, t.TempDir()+"/state", "station:|slot:/Services")
}

// remove is one n4_remove_history call.
func remove(t *testing.T, r *ackRig, kv ...any) (string, bool, *pyjson.Object) {
	t.Helper()
	return r.tool(t, "n4_remove_history", kv...)
}

// removeBoth is a dry run of kv, then its execution with the token.
func removeBoth(t *testing.T, r *ackRig, kv ...any) (*pyjson.Object, *pyjson.Object) {
	t.Helper()
	text, isErr, dry := remove(t, r, kv...)
	if isErr {
		t.Fatalf("dry run: %s", text)
	}
	text, isErr, out := remove(t, r, append(kv, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))...)
	if isErr {
		t.Fatalf("execute: %s", text)
	}
	return dry, out
}

// historyReads is a station client that can read histories but cannot Call a channel.
type historyReads struct{ db *historyDB }

func (r historyReads) GetOrd(o string, t float64, m int) (string, bool, error) {
	return r.db.GetOrd(o, t, m)
}
func (r historyReads) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) {
	return nil, nil
}
func (r historyReads) SSC(string, any) (any, error) { panic("no ssc") }
func (r historyReads) Sync(any) (any, error)        { panic("no sync") }
func (r historyReads) ClearHandles()                {}
func (r historyReads) Close()                       {}
