package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The reads of n4_ack_alarm: the records by uuid over the alarm: ORD, all of them or only the
// unacked ones (the ack state is compared by its tag in BQL, never by the station's localized
// display text), and the read-back of an executed ack.

// ackQuery reads a record's uuid, ack state and acking user, by position.
const ackQuery = "select uuid, ackState, user"

// ackTimeout is the HTTP timeout of each read, in seconds.
const ackTimeout = 60

var ackFields = []string{"uuid", "ack_state", "user"}

// readAcks is the records among ids that exist (unackedOnly: that are still unacked), by uuid.
func readAcks(r ordReader, ids []any, unackedOnly bool) (map[string]*pyjson.Object, error) {
	terms := make([]string, len(ids))
	for i, id := range ids { // checked against uuidRe: no quote reaches the clause
		terms[i] = "uuid = '" + id.(string) + "'"
	}
	where := "(" + strings.Join(terms, " or ") + ")"
	if unackedOnly {
		where = "ackState = 'unacked' and " + where
	}
	text, cut, err := r.GetOrd(bql.ComposeAlarmOrd(ackQuery, where, ""), ackTimeout, bql.MaxResponseBytes)
	if err != nil {
		return nil, err
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, rows, _, err := bql.ParseCSV(text, maxAckIDs, []int{})
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, ackFields)
	if err != nil {
		return nil, err
	}
	out := map[string]*pyjson.Object{}
	for _, row := range keyed {
		out[strings.ToLower(pyjson.Str(get(row, "uuid")))] = row
	}
	return out, nil
}

// ackReadback re-reads the records: verified when every one exists, none is still unacked, each
// has a user and the station named no class it refused; else mismatch.
func ackReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	ids := p.Data.([]any)
	requested := obj("uuids", ids)
	if v, ok := args.Get("notes"); ok {
		requested.Set("notes", v)
	}
	r := c.(ordReader) // the plan proved it
	all, err := readAcks(r, ids, false)
	if err != nil {
		return nil, nil, nil, "", err
	}
	unacked, err := readAcks(r, ids, true)
	if err != nil {
		return nil, nil, nil, "", err
	}
	failed, isList := replies[0].([]any)
	ok := isList && len(failed) == 0
	records, still := []any{}, []any{}
	for _, id := range ids {
		row := all[id.(string)]
		if row == nil {
			records = append(records, obj("uuid", id, "ack_state", nil, "user", nil))
			ok = false
			continue
		}
		records = append(records, obj("uuid", id, "ack_state", get(row, "ack_state"), "user", get(row, "user")))
		if unacked[id.(string)] != nil {
			still = append(still, id)
		}
		ok = ok && pyjson.Str(get(row, "user")) != ""
	}
	verdict := "verified"
	if !ok || len(still) > 0 {
		verdict = "mismatch"
	}
	var shown any = failed
	if !isList {
		shown = replies[0]
	}
	observed := obj("records", records, "still_unacked", still, "permission_failed_classes", shown)
	return requested, replies[0], observed, verdict, nil
}

// ackUnapplied is true when the station refused the call and every record is still unacked.
func ackUnapplied(c plan.Client, _ *pyjson.Object, p *plan.Planned) (bool, error) {
	r, ok := c.(ordReader)
	if !ok {
		return false, nil
	}
	ids := p.Data.([]any)
	unacked, err := readAcks(r, ids, true)
	return err == nil && len(unacked) == len(ids), err
}
