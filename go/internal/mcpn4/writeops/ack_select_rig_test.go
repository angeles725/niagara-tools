package writeops_test

// The rig of the n4_ack_alarm selection tests: the alarm database of ack_rig_test.go, which
// also answers the n4_read_alarms records query and its count, filtered by the clauses the
// selection may send (source subtree, alarm class, timestamp bound, unacked only).

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const u6 = "aa110470-0000-4000-8000-000000000006" // raised between a dry run and its execution

var (
	sourceLikeRe = regexp.MustCompile(`source like 'slot:([^'*]+)'`)
	classEqRe    = regexp.MustCompile(`alarmClass = '([^']*)'`)
	untilRe      = regexp.MustCompile(`timestamp\.millis <= (\d+)`)
)

// selectDB is alarmDB plus each record's class and alarm time (epoch ms).
type selectDB struct {
	*alarmDB
	class  map[string]string
	millis map[string]int64
}

func ms(iso string) int64 {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		panic(err)
	}
	return t.UnixMilli()
}

func newSelectDB() *selectDB {
	return &selectDB{alarmDB: newAlarmDB(),
		class: map[string]string{u1: "Critical", u2: "defaultAlarmClass", u3: "Critical", u4: "Critical", u5: "Critical"},
		millis: map[string]int64{u1: ms("2026-11-15T00:00:00Z"), u2: ms("2026-11-16T00:00:00Z"), u3: ms("2026-11-14T00:00:00Z"),
			u4: ms("2026-11-13T00:00:00Z"), u5: ms("2026-11-12T00:00:00Z")}}
}

// raise adds an unacked record under source with class, newer than every other.
func (d *selectDB) raise(id, source, class string) {
	d.state[id], d.source[id], d.class[id], d.millis[id] = "unacked", source, class, ms("2026-11-17T00:00:00Z")
}

// matches is the records the where clause of ord selects, newest first.
func (d *selectDB) matches(ord string) []string {
	var out []string
	for id, st := range d.state {
		if strings.Contains(ord, "ackState = 'unacked'") && st != "unacked" {
			continue
		}
		if m := sourceLikeRe.FindStringSubmatch(ord); m != nil {
			path, ok := bql.RecordSourcePath(d.source[id])
			if !ok || path != m[1] && !strings.HasPrefix(path, m[1]+"/") {
				continue
			}
		}
		if m := classEqRe.FindStringSubmatch(ord); m != nil && d.class[id] != m[1] {
			continue
		}
		if m := untilRe.FindStringSubmatch(ord); m != nil {
			if n, _ := strconv.ParseInt(m[1], 10, 64); d.millis[id] > n {
				continue
			}
		}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return d.millis[out[i]] > d.millis[out[j]] })
	return out
}

// GetOrd answers the records query and its count; any other ORD is alarmDB's uuid read.
func (d *selectDB) GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error) {
	switch {
	case strings.HasPrefix(ord, "alarm:|bql:"+bql.AlarmCountQuery):
		d.ords = append(d.ords, ord)
		return fmt.Sprintf("count(*)\n%d\n", len(d.matches(ord))), false, nil
	case strings.HasPrefix(ord, "alarm:|bql:"+bql.AlarmRecordsQuery):
		d.ords = append(d.ords, ord)
		var b strings.Builder
		b.WriteString("Timestamp,Uuid,Source,Source State,Ack State,Alarm Class,Priority,Msg Text,T,N,A\n")
		for _, id := range d.matches(ord) {
			iso := time.UnixMilli(d.millis[id]).UTC().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(&b, "x,%s,%s,Offnormal,Unacked,%s,10,m,%s,null,null\n", id, d.source[id], d.class[id], iso)
		}
		return b.String(), false, nil
	}
	return d.alarmDB.GetOrd(ord, timeout, maxBytes)
}

// sel is a select argument.
func sel(kv ...any) *pyjson.Object { return obj(kv...) }
