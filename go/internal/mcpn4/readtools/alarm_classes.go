package readtools

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_list_alarm_classes: every alarm class with its priority and ack requirement per transition
// decoded, its alarm counts, and the recipients its alarm topic is linked to. Read-only: two
// BQL selects at the station root and tree loads (alarm_recipients.go).

// The fixed queries, their columns read by position.
const (
	alarmClassesQuery    = "select slotPath, name, priority, ackRequired, totalAlarmCount, openAlarmCount, inAlarmCount, unackedAlarmCount from alarm:AlarmClass"
	alarmRecipientsQuery = "select slotPath, type from alarm:AlarmRecipient"
)

var (
	alarmClassFields     = []string{"path", "name", "priority", "ack_required", "total_alarm_count", "open_alarm_count", "in_alarm_count", "unacked_alarm_count"}
	alarmRecipientFields = []string{"path", "type"}
	// alarmTransitions are the BAlarmTransitionBits names, in the order BAlarmPriorities encodes them.
	alarmTransitions = []string{"toOffnormal", "toFault", "toNormal", "toAlert"}
	// ackTokens map the words of BAlarmTransitionBits.toString to the transition ("alert" is toAlert).
	ackTokens = map[string]string{"toOffnormal": "toOffnormal", "toFault": "toFault", "toNormal": "toNormal",
		"alert": "toAlert", "toAlert": "toAlert"}
)

// ListAlarmClasses is n4_list_alarm_classes.
func (t *Tools) ListAlarmClasses(args *pyjson.Object) (*pyjson.Object, error) {
	timeout := intArg(args, "timeout_s", alarmDefaultTimeout)
	classes, err := t.selectKeyed(alarmClassesQuery, alarmClassFields, timeout)
	if err != nil {
		return nil, err
	}
	recipients, err := t.selectKeyed(alarmRecipientsQuery, alarmRecipientFields, timeout)
	if err != nil {
		return nil, err
	}
	for _, c := range classes {
		c.Set("path", slotPathOf(get(c, "path")))
		c.Set("priority", decodePriorities(pyjson.Str(get(c, "priority"))))
		c.Set("ack_required", decodeAckRequired(pyjson.Str(get(c, "ack_required"))))
		for _, k := range alarmClassFields[4:] {
			if n, err := strconv.Atoi(strings.TrimSpace(pyjson.Str(get(c, k)))); err == nil {
				c.Set(k, n)
			}
		}
		c.Set("recipients", []any{})
	}
	routes, err := t.alarmRoutes(classes, recipients, timeout)
	if err != nil {
		return nil, err
	}
	t.recordRead("n4_list_alarm_classes", "", args, len(classes))
	return obj("classes", list(classes), "recipients", routes.recipients, "unresolved_links", routes.unresolved,
		"recipients_truncated", routes.truncated), nil
}

// selectKeyed is one projected select at the station root, its rows keyed by position.
func (t *Tools) selectKeyed(query string, fields []string, timeout int) ([]*pyjson.Object, error) {
	columns, rows, _, _, err := t.query(RootOrd, query, bql.DefaultMaxRows, timeout)
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, fields)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	return keyed, nil
}

// decodePriorities is BAlarmPriorities text (toOffnormal=255;toFault=255;...;) as
// {transition: priority}; any other text is returned unchanged.
func decodePriorities(text string) any {
	values := map[string]int{}
	for _, part := range strings.Split(strings.TrimSuffix(text, ";"), ";") {
		k, v, ok := strings.Cut(part, "=")
		n, err := strconv.Atoi(v)
		if _, dup := values[k]; !ok || err != nil || dup || ackTokens[k] != k {
			return text
		}
		values[k] = n
	}
	if len(values) != len(alarmTransitions) {
		return text
	}
	out := pyjson.NewObject()
	for _, k := range alarmTransitions {
		out.Set(k, values[k])
	}
	return out
}

// decodeAckRequired is BAlarmTransitionBits text ({toOffnormal toFault toNormal}, {} for none)
// as {transition: required}; any other text is returned unchanged.
func decodeAckRequired(text string) any {
	inner, opened := strings.CutPrefix(text, "{")
	inner, closed := strings.CutSuffix(inner, "}")
	if !opened || !closed {
		return text
	}
	set := map[string]bool{}
	for _, word := range strings.Fields(inner) {
		name, known := ackTokens[word]
		if !known {
			return text
		}
		set[name] = true
	}
	out := pyjson.NewObject()
	for _, k := range alarmTransitions {
		out.Set(k, set[k])
	}
	return out
}

// slotPathOf is a BQL slotPath cell without its "slot:" scheme.
func slotPathOf(v any) string {
	return strings.TrimPrefix(pyjson.Str(v), slotPrefix)
}

func get(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(k)
	return v
}
