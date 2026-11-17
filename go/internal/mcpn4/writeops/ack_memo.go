package writeops

import (
	"fmt"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/token"
)

// The resolved selections of n4_ack_alarm dry runs. Executing a selection never re-runs it: the
// execution re-plans from the uuids and report its dry run resolved, so the plan it hashes is
// the one the token was issued for, and an alarm raised in between is never acknowledged. A
// selection is held per write state, station session and exact bound arguments; the newest dry
// run of the same arguments replaces it (an older token then no longer matches the plan). It is
// only read at execution: a token is single use and a new one comes only from a new dry run,
// which resolves the selection again, so a held selection never authorizes a second ack.

// maxHeldSelections bounds the held selections; the oldest goes first.
const maxHeldSelections = 64

const msgSelNoDryRun = "no dry run of this selection is held on this session: run the dry run again and " +
	"execute with its confirmation_token"

// heldSelection is one dry run's resolution: its uuids and its plan.selection report.
type heldSelection struct {
	ids    []any
	report *pyjson.Object
}

var held = struct {
	sync.Mutex
	byKey map[string]heldSelection
	order []string
}{byKey: map[string]heldSelection{}}

// selectionKey names a selection by write state, station session and bound arguments.
func selectionKey(w *plan.State, sess *session.Session, args *pyjson.Object) (string, error) {
	bound, err := pyjson.Marshal(token.BoundArgs(args), pyjson.Canonical)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%p|%s|%s|%s|%s", w, sess.StationName, sess.BaseURL, sess.SessionID, bound), nil
}

// holdSelection keeps a dry run's resolution for its execution.
func holdSelection(key string, s heldSelection) {
	held.Lock()
	defer held.Unlock()
	if _, ok := held.byKey[key]; !ok {
		held.order = append(held.order, key)
	}
	held.byKey[key] = s
	for len(held.order) > maxHeldSelections {
		delete(held.byKey, held.order[0])
		held.order = held.order[1:]
	}
}

// heldFor is the resolution held for key.
func heldFor(key string) (heldSelection, error) {
	held.Lock()
	defer held.Unlock()
	s, ok := held.byKey[key]
	if !ok {
		return heldSelection{}, plan.ToolError(msgSelNoDryRun)
	}
	return s, nil
}

// ackSelected is the selection's uuids and plan keys {selection: report}: at a dry run resolved
// from the station, at an execution the ones its dry run held (never resolved again). nothing is
// the reply of a dry run whose selection matched no record.
func ackSelected(r ordReader, s *ackSelection, args *pyjson.Object, w *plan.State, sess *session.Session) (
	ids []any, keys, nothing *pyjson.Object, err error) {
	key, err := selectionKey(w, sess, args)
	if err != nil {
		return nil, nil, nil, err
	}
	var h heldSelection
	if isDryRun(args) {
		h.ids, h.report, err = s.resolve(r)
	} else {
		h, err = heldFor(key)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if len(h.ids) == 0 {
		return nil, nil, nothingSelected(h.report), nil
	}
	return h.ids, obj("selection", h.report), nil, nil
}

// holdDrySelection keeps a dry run's checked selection (keys not nil) for its execution.
func holdDrySelection(args *pyjson.Object, w *plan.State, sess *session.Session, ids []any, keys *pyjson.Object) error {
	if keys == nil || !isDryRun(args) {
		return nil
	}
	key, err := selectionKey(w, sess, args)
	if err != nil {
		return err
	}
	holdSelection(key, heldSelection{ids: ids, report: get(keys, "selection").(*pyjson.Object)})
	return nil
}

// isDryRun is the engine's reading of dry_run: true unless given and falsy.
func isDryRun(args *pyjson.Object) bool {
	v, ok := args.Get("dry_run")
	return !ok || pyjson.Truthy(v)
}
