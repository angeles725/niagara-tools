package writeops

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// AckAlarm is n4_ack_alarm: one ackAlarms call on the station's alarm channel (box-rt
// BAlarmChannel.ackAlarms: body {ids, notes?}, the station sets user, ackTime and ackPending,
// updates the alarm database and routes the ack). The engine only requires some write scope;
// the plan then checks each record's source against it (ackPlan, sourceOrd). No inverse: an
// acknowledgement cannot be undone.
var AckAlarm = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil }, PlanWith: ackPlan,
	Readback: ackReadback, Unapplied: ackUnapplied}

// The alarm channel call.
const (
	ackChannel = "alarm"
	ackKey     = "ackAlarms"
	maxAckIDs  = 100
)

// uuidRe is BUuid's text form (lower case once normalized).
var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Messages of n4_ack_alarm, as data.
const (
	msgAckCount    = "uuids must hold 1 to %d alarm record uuids, got %d"
	msgAckNotUUID  = "uuids[%d] is not a uuid (8-4-4-4-12 hex digits, as n4_read_alarms returns it): %s"
	msgAckTwice    = "uuid %s is given twice"
	msgAckNotes    = "notes must be a string"
	msgAckMissing  = "no alarm record has uuid %s"
	msgAckDone     = "already acknowledged or ack pending: %s"
	msgAckRefused  = "n4_ack_alarm refused, nothing was planned: %s"
	msgAckNoReader = "the station client cannot read alarm records"
	msgAckSourceOf = "%s (source %s)"
	msgAckOutside  = "outside the write scope: %s"
	msgAckUnparsed = "source is not a slot path: %s"
	noteAckUndo    = "an acknowledgement cannot be undone: there is no inverse and n4_rollback cannot reverse it"
	noteAckUser    = "the station records the ack under the connected user and needs operatorWrite on each record's alarm class; " +
		"a class it refuses is named in the reply and makes the verdict mismatch"
)

// ordReader is a client that can read an ORD as text (box.Client.GetOrd).
type ordReader interface {
	GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error)
}

// ackPlan checks the uuids (or resolves the selection), that the client can Call the alarm
// channel, and that every record exists, is in scope and is unacked; the plan is the one channel
// op, with no inverse. A selection also joins the hashed plan as its selection report.
func ackPlan(c plan.Client, args *pyjson.Object, w *plan.State, sess *session.Session) (*plan.Planned, error) {
	rawSel, bySelect := args.Get("select")
	_, byIDs := args.Get("uuids")
	switch {
	case bySelect && byIDs:
		return nil, plan.ToolError(msgSelBoth)
	case !bySelect && !byIDs:
		return nil, plan.ToolError(msgSelNone)
	}
	var ids []any
	var selection *ackSelection
	var err error
	if bySelect {
		selection, err = parseSelect(rawSel)
	} else {
		ids, err = ackIDs(args)
	}
	if err != nil {
		return nil, err
	}
	notes, hasNotes := args.Get("notes")
	if _, isStr := notes.(string); hasNotes && !isStr {
		return nil, plan.ToolError(msgAckNotes)
	}
	if !plan.CanCall(c) {
		return nil, plan.NoChannelError(ackChannel, ackKey)
	}
	r, ok := c.(ordReader)
	if !ok {
		return nil, plan.ToolError(msgAckNoReader)
	}
	var keys *pyjson.Object
	if bySelect {
		var nothing *pyjson.Object
		ids, keys, nothing, err = ackSelected(r, selection, args, w, sess)
		if err != nil {
			return nil, err
		}
		if nothing != nil {
			return &plan.Planned{Nothing: nothing}, nil
		}
	}
	if err := checkAckRecords(r, ids, w); err != nil {
		return nil, err
	}
	if err := holdDrySelection(args, w, sess, ids, keys); err != nil {
		return nil, err
	}
	body := obj("ids", ids)
	if hasNotes {
		body.Set("notes", notes)
	}
	return &plan.Planned{Ops: []any{plan.ChannelOp(ackChannel, ackKey, body)}, Notes: []any{noteAckUndo, noteAckUser},
		Keys: keys, Data: ids}, nil
}

// checkAckRecords refuses, naming them, the uuids with no record, a source outside the write
// scope or not a slot path, or a record already acknowledged or ack pending.
func checkAckRecords(r ordReader, ids []any, w *plan.State) error {
	exists, err := readAcks(r, ids, false)
	if err != nil {
		return err
	}
	unacked, err := readAcks(r, ids, true)
	if err != nil {
		return err
	}
	var missing, outside, unparsed, done []string
	for _, id := range ids {
		row, found := exists[id.(string)]
		source := pyjson.Str(get(row, "source"))
		ord, parsed := sourceOrd(source)
		switch {
		case !found:
			missing = append(missing, id.(string))
		case !parsed:
			unparsed = append(unparsed, fmt.Sprintf(msgAckSourceOf, id, source))
		case w.Scope.Check(ord) != nil:
			outside = append(outside, fmt.Sprintf(msgAckSourceOf, id, source))
		case unacked[id.(string)] == nil:
			done = append(done, id.(string))
		}
	}
	var why []string
	for _, g := range []struct {
		msg  string
		list []string
	}{{msgAckMissing, missing}, {msgAckOutside, outside}, {msgAckUnparsed, unparsed}, {msgAckDone, done}} {
		if g.list != nil {
			why = append(why, fmt.Sprintf(g.msg, strings.Join(g.list, ", ")))
		}
	}
	if why != nil {
		return plan.ToolError(fmt.Sprintf(msgAckRefused, strings.Join(why, "; ")))
	}
	return nil
}

// sourceOrd is the station slot ORD of a record source in either stored form
// (local:|station:|slot:/a or slot:/a); false for any other form (fail closed): a source
// holding several ORDs (BOrdList joins them with "; " or a newline), any whitespace, a chained
// '|' (a view), a ".." segment or a ':' in the path (never in an escaped slot name; the CSV
// reader joins the lines of a quoted cell, so "slot:/a\nslot:/b" arrives as "slot:/aslot:/b")
// is never read as its first slot path.
func sourceOrd(source string) (string, bool) {
	slot := strings.TrimPrefix(source, "local:|station:|")
	if !strings.HasPrefix(slot, "slot:/") || strings.ContainsAny(slot[len("slot:"):], ";:") ||
		strings.IndexFunc(slot, unicode.IsSpace) >= 0 || !ord.PlainSlotOrd("station:|"+slot) {
		return "", false
	}
	return "station:|" + slot, true
}

// ackIDs is the checked uuids argument, lower case, in the caller's order.
func ackIDs(args *pyjson.Object) ([]any, error) {
	v, _ := args.Get("uuids")
	list, _ := v.([]any)
	if len(list) < 1 || len(list) > maxAckIDs {
		return nil, plan.ToolError(fmt.Sprintf(msgAckCount, maxAckIDs, len(list)))
	}
	seen, ids := map[string]bool{}, make([]any, 0, len(list))
	for i, x := range list {
		s, ok := x.(string)
		s = strings.ToLower(s)
		if !ok || !uuidRe.MatchString(s) {
			return nil, plan.ToolError(fmt.Sprintf(msgAckNotUUID, i, pyjson.Repr(x)))
		}
		if seen[s] {
			return nil, plan.ToolError(fmt.Sprintf(msgAckTwice, s))
		}
		seen[s] = true
		ids = append(ids, s)
	}
	return ids, nil
}
