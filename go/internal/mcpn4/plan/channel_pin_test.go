package plan_test

// The ssc and sync paths stay byte-identical with channel ops in the engine: an existing write's
// reply and journal records, volatile fields (ts, batch_id) masked, are pinned as recorded before
// channel ops existed.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// masked is the canonical JSON of o without its volatile keys.
func masked(t *testing.T, o *pyjson.Object) string {
	t.Helper()
	c := pyjson.NewObject()
	for _, k := range o.Keys() {
		if k != "ts" && k != "batch_id" {
			v, _ := o.Get(k)
			c.Set(k, v)
		}
	}
	b, err := pyjson.Marshal(c, pyjson.Canonical)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSyncAndSscWritesKeepItsReplyAndJournalWithChannelOps(t *testing.T) {
	r := newRig(t)
	var got []string
	for _, call := range []struct {
		name string
		args *pyjson.Object
	}{{"n4_set_slot", setArgs()}, {"n4_save_station", node()}} {
		text, isErr, out := r.execute(t, call.name, call.args, r.dry(t, call.name, call.args))
		if isErr {
			t.Fatal(text)
		}
		got = append(got, masked(t, out))
	}
	for _, l := range r.lines(t, "journal.jsonl") {
		got = append(got, masked(t, l))
	}
	if strings.Join(got, "\n") != sscPin {
		t.Errorf("set_slot reply and journal changed:\n%s", strings.Join(got, "\n"))
	}
}

const sscPin = `{"accepted":[],"dry_run":false,"inverse":[{"b":{"nm":"p","s":[{"n":"value","nm":"p","t":"baja:Double","v":"1.0"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],"t":"baja:StatusNumeric"},"h":"10","n":"out","nm":"s"}],"observed":{"status":"0","value":5.5},"requested":{"status":"0","value":5.5},"verdict":"verified","write_budget":{"executed":1,"max":200,"remaining":199,"rollback_counts":false,"save_counts":false}}
{"accepted":null,"dry_run":false,"evidence":{"hint":"the station's save jobs could not be read before the save (no such component: station:|slot:/Services/JobService); configure --station-home FakeStation=<directory containing config.bog> to check config.bog directly","ord":"station:|slot:/Services/JobService","route":"station save job"},"inverse":[],"observed":{"evidence":{"hint":"the station's save jobs could not be read before the save (no such component: station:|slot:/Services/JobService); configure --station-home FakeStation=<directory containing config.bog> to check config.bog directly","ord":"station:|slot:/Services/JobService","route":"station save job"},"persisted":"unknown"},"persisted":"unknown","requested":{"action":"save"},"verdict":"unverified","write_budget":{"executed":1,"max":200,"remaining":199,"rollback_counts":false,"save_counts":false}}
{"inverse_plan":[{"b":{"nm":"p","s":[{"n":"value","nm":"p","t":"baja:Double","v":"1.0"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],"t":"baja:StatusNumeric"},"h":"10","n":"out","nm":"s"}],"ops":[{"b":{"nm":"p","s":[{"n":"value","nm":"p","t":"baja:Double","v":"5.5"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],"t":"baja:StatusNumeric"},"h":"10","n":"out","nm":"s"}],"phase":"intent","station_name":"FakeStation","targets":{"station:|slot:/Folder/Calc":"10"},"tool":"n4_set_slot"}
{"accepted":[],"inverse":[{"b":{"nm":"p","s":[{"n":"value","nm":"p","t":"baja:Double","v":"1.0"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],"t":"baja:StatusNumeric"},"h":"10","n":"out","nm":"s"}],"phase":"result","verdict":"verified"}
{"inverse_plan":[],"ops":[{"arg":{"a":"save","h":"2"},"ssc":"invokeAction"}],"phase":"intent","station_name":"FakeStation","targets":{},"tool":"n4_save_station"}
{"accepted":null,"inverse":[],"phase":"result","verdict":"unverified"}`
