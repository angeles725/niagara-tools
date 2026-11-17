package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// laterWorkState is "found" for a copy no later journaled batch wrote under, else "later_work": a
// later batch may have configured or extended it, and removing it would destroy that work (the
// check a regular rollback of a create makes, rollback_later.go; review R1-001, R3-001). A copy
// deleted and re-created by hand outside the journal is not detectable here.
func laterWorkState(w *plan.State, view *pyjson.Object, bid, where string, name any) (string, error) {
	comp, err := childOrd(where, name)
	if err != nil {
		return "", err
	}
	batches, err := laterBatches(w, view, bid, comp)
	if err != nil || len(batches) > 0 {
		return copyLaterWork, err
	}
	return copyFound, nil
}
