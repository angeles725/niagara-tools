package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The recipients of n4_list_alarm_classes: a class routes to a recipient through a link stored
// on the recipient, from the class's "alarm" topic to the recipient's "routeAlarm" action. Each
// recipient is loaded two levels deep (its links and their slots); a link source given by
// handle is resolved against the handles of the folders holding the classes.

// maxAlarmRecipients bounds the recipients loaded (one tree load each).
const maxAlarmRecipients = 50

// The slots of a class-to-recipient route.
const (
	alarmTopic  = "alarm"
	routeAction = "routeAlarm"
)

// alarmRouting is what the recipient scan leaves for the reply.
type alarmRouting struct {
	recipients, unresolved []any
	truncated              bool
	total                  int // the recipients the station listed
}

// alarmRoutes reads the links into each recipient's routeAlarm action (at most
// maxAlarmRecipients recipients) and adds {path, type, link_name} to the "recipients" of the class
// each one comes from; audit adds each recipient's settings and problems (alarm_recipient_audit.go).
func (t *Tools) alarmRoutes(classes, recipients []*pyjson.Object, timeout int, audit bool) (*alarmRouting, error) {
	byPath := map[string]*pyjson.Object{}
	for _, c := range classes {
		byPath[pyjson.Str(get(c, "path"))] = c
	}
	byHandle, err := t.classHandles(classes)
	if err != nil {
		return nil, err
	}
	out := &alarmRouting{recipients: []any{}, unresolved: []any{}, truncated: len(recipients) > maxAlarmRecipients,
		total: len(recipients)}
	for i, r := range recipients {
		if i == maxAlarmRecipients {
			break
		}
		path, kind := slotPathOf(get(r, "path")), get(r, "type")
		nodes, err := t.Ctx.Session.Client.(Client).LoadTree(bql.StationSlot+path, 2, box.LoadOptions{})
		if err != nil {
			return nil, err
		}
		linked := []any{}
		for _, name := range nodes.Keys() {
			if strings.Contains(name, "/") || !isLinkType(get(box.Node(nodes, name), "t")) {
				continue
			}
			kids := box.Children(nodes, name)
			if box.ChildValue(kids, "sourceSlotName", nil) != alarmTopic || box.ChildValue(kids, "targetSlotName", nil) != routeAction {
				continue
			}
			sourceOrd := box.ChildValue(kids, "sourceOrd", nil)
			source, _, err := sourcePath(byHandle, sourceOrd)
			if err != nil {
				return nil, err
			}
			class, ok := byPath[pyjson.Str(source)]
			if !ok || source == nil {
				out.unresolved = append(out.unresolved, obj("recipient_path", path, "link_name", name, "source_ord", sourceOrd))
				continue
			}
			class.Set("recipients", append(get(class, "recipients").([]any), obj("path", path, "type", kind, "link_name", name)))
			linked = append(linked, get(class, "name"))
		}
		entry := obj("path", path, "type", kind, "classes", linked)
		if audit {
			found := recipientAudit(nodes, pyjson.Str(kind))
			for _, k := range found.Keys() {
				entry.Set(k, get(found, k))
			}
		}
		out.recipients = append(out.recipients, entry)
	}
	return out, nil
}

// classHandles maps the handle of each component in the folders holding the classes to its slot
// path: one tree load per folder, one level deep.
func (t *Tools) classHandles(classes []*pyjson.Object) (map[string]string, error) {
	byHandle, loaded := map[string]string{}, map[string]bool{}
	for _, c := range classes {
		path := pyjson.Str(get(c, "path"))
		cut := strings.LastIndexByte(path, '/')
		if cut < 0 {
			continue
		}
		folder := bql.StationSlot + path[:cut]
		if path[:cut] == "" {
			folder = RootOrd
		}
		if loaded[folder] {
			continue
		}
		loaded[folder] = true
		nodes, err := t.Ctx.Session.Client.(Client).LoadTree(folder, 1, box.LoadOptions{})
		if err != nil {
			return nil, err
		}
		for _, name := range nodes.Keys() {
			if h, ok := get(box.Node(nodes, name), "h").(string); ok && h != "" && name != "" {
				byHandle[h] = absPath(folder, name)
			}
		}
	}
	return byHandle, nil
}
