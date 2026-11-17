package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Behaviour diff around a deploy (opportunity O6): n4_behavior_snapshot stores the out value and
// status of every control point below a scope; after the deploy and restart, n4_behavior_diff
// compares that snapshot with a second one or with the live station and lists what changed.

// Bounds of the behaviour tools.
const (
	BehaviorDefaultPoints = 5000
	BehaviorDefaultLimit  = 50
)

// BehaviorSnapshot is n4_behavior_snapshot: one BQL read of the control points below scope,
// written to <state-dir>/snapshots/<name>.json (0600, never overwritten).
func (t *Tools) BehaviorSnapshot(args *pyjson.Object) (*pyjson.Object, error) {
	name := strArg(args, "name", "")
	if !snapshotName.MatchString(name) {
		return nil, toolError("name must be letters, digits, '_' or '-' (up to 80), got %s", pyjson.StrRepr(name))
	}
	s, err := t.readBehavior(name, strArg(args, "scope", RootOrd), intArg(args, "max_points", BehaviorDefaultPoints),
		intArg(args, "timeout_s", 60))
	if err != nil {
		return nil, err
	}
	path, err := t.saveSnapshot(s)
	if err != nil {
		return nil, err
	}
	out := obj("name", s.name, "station", s.station, "scope", s.scope, "taken_at", s.takenAt,
		"points", len(s.points), "truncated", s.truncated, "file", path)
	if s.truncated {
		out.Set("truncated_note", behaviorCutNote(len(s.points)))
	}
	return out, nil
}

// BehaviorDiff is n4_behavior_diff: the snapshot before against the snapshot after, or against a
// live read of before's scope when after is omitted (that needs a connected session).
func (t *Tools) BehaviorDiff(args *pyjson.Object) (*pyjson.Object, error) {
	before, err := t.loadSnapshot(strArg(args, "before", ""))
	if err != nil {
		return nil, err
	}
	var after *snapshot
	if name := strArg(args, "after", ""); name != "" {
		after, err = t.loadSnapshot(name)
	} else if !t.Ctx.Connected() {
		return nil, toolError("no after snapshot given and no station connected: call n4_connect to compare with the live station")
	} else {
		after, err = t.readBehavior("live", before.scope, intArg(args, "max_points", BehaviorDefaultPoints),
			intArg(args, "timeout_s", 60))
	}
	if err != nil {
		return nil, err
	}
	var tolerance float64 // the schema allows only a number >= 0
	if n, ok := lookup(args, "tolerance").(pyjson.Number); ok {
		tolerance, _ = n.Float64()
	}
	return behaviorReport(before, after, tolerance, intArg(args, "limit", BehaviorDefaultLimit)), nil
}
