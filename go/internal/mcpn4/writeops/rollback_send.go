package writeops

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// The one link writer of a rollback: every link it re-creates goes through sendLinks.

// The outcome of one sent link.
const (
	linkRefused   = iota // the station said no
	linkRestored         // accepted with a name: its inverse removes it
	linkAmbiguous        // no verdict, or accepted without a name: maybe applied
)

// linkOp is the checkLinks op that creates the link src.ss -> tgt.ts (n4_create_link's shape).
func linkOp(src, ss, tgt, ts any) *pyjson.Object {
	return obj("ssc", "checkLinks", "arg", obj("s", src, "ss", ss, "t", tgt, "ts", ts, "c", true))
}

// sendLinks journals ops as the batch's relink intent before the first is sent, counts them
// against the budget and sends them in order; each restored link's remove joins runs.Inverse
// (targets[i] holds op i's link). A reply that cannot be read leaves the batch in-doubt.
func sendLinks(b *plan.Batch, d *rollbackData, base map[int]string, ops, targets []any, runs *plan.Runs) ([]int, error) {
	states := make([]int, len(ops))
	if len(ops) == 0 {
		return states, nil
	}
	if b.State.Journal.Append(obj("batch_id", b.ID, "ts", session.Now(), "phase", "relink-intent", "ops", ops)) != nil {
		return nil, withCreated(plan.InDoubtBecause(b.ID, "nothing more was sent: the relink intent could "+
			"not be written (components were re-created, links were not)"), base, d.created)
	}
	b.State.WritesExecuted += len(ops)
	for i, op := range ops {
		result, err := relinkResult(b.Client, get(op.(*pyjson.Object), "arg"))
		if err != nil {
			return nil, withCreated(plan.InDoubtOf(b.ID, err), base, d.created)
		}
		v, hasV := result.Get("v")
		switch {
		case pyjson.Truthy(v) && pyjson.Truthy(get(result, "s")):
			states[i] = linkRestored
			runs.Inverse = append(runs.Inverse, obj("nm", "v", "h", targets[i], "n", get(result, "s")))
		case !hasV || pyjson.Truthy(v):
			states[i] = linkAmbiguous
		}
	}
	return states, nil
}

// tally counts the restored and the ambiguous links of states.
func tally(states []int) (restored, ambiguous int) {
	for _, s := range states {
		switch s {
		case linkRestored:
			restored++
		case linkAmbiguous:
			ambiguous++
		}
	}
	return restored, ambiguous
}

// relinkResult is one relink's verdict dict; a station error or a malformed reply is {}.
func relinkResult(c plan.Client, arg any) (*pyjson.Object, error) {
	reply, err := c.SSC("checkLinks", arg)
	var be *box.Error
	if errors.As(err, &be) {
		return obj(), nil
	} else if err != nil {
		return nil, err
	}
	result, err := linkResult([]any{reply})
	if err != nil {
		return obj(), nil
	}
	return result, nil
}
