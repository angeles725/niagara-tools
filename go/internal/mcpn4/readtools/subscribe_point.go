package readtools

import (
	"fmt"
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The bounds of n4_subscribe_point (#875): the whole call answers inside the 60 s an MCP client
// commonly waits for one request, and the point is re-read every few seconds meanwhile.
const (
	subDefaultSeconds = 10
	subMaxSeconds     = 40
	subCheckEvery     = 2.0
	subLoadDepth      = 2 // the point, its out and proxyExt, and their value and status children
)

// subscriber is the part of a station client that subscribes components over the BOX
// component space (box.Client.Subscribe and Unsubscribe).
type subscriber interface {
	Subscribe(ords []string) error
	Unsubscribe(ords []string) error
	DrainPendingEvents() []any
}

// SubscribePoint is n4_subscribe_point: subscribe one control point over BOX, so its driver
// polls it as a Workbench view would, re-read it every few seconds until its stale flag clears
// or seconds pass, then always unsubscribe. BOX subscriptions carry no lease, so the unsub runs
// on every exit, including a failed read.
func (t *Tools) SubscribePoint(args *pyjson.Object) (out *pyjson.Object, err error) {
	ord, err := ordArg(args)
	if err != nil {
		return nil, err
	}
	secs := intArg(args, "seconds", subDefaultSeconds)
	if secs < 1 || secs > subMaxSeconds {
		return nil, toolError("seconds must be 1 to %d, got %d", subMaxSeconds, secs)
	}
	sub, ok := t.Ctx.Session.Client.(subscriber)
	if !ok {
		return nil, toolError("this session's client cannot subscribe a component")
	}
	before, err := t.readSubState(ord)
	if err != nil {
		return nil, err
	}
	if err := sub.Subscribe([]string{ord}); err != nil {
		return nil, err
	}
	defer func() {
		uerr := sub.Unsubscribe([]string{ord})
		sub.DrainPendingEvents() // the subscription's events: nothing here reads them
		if out == nil {
			return
		}
		if uerr != nil {
			out.Set("unsubscribed", false)
			out.Set("unsubscribe_error", uerr.Error())
			return
		}
		out.Set("unsubscribed", true)
	}()
	after, ended, waited, readErr := t.watchPoint(ord, before, float64(secs))
	out = obj("ord", ord, "seconds", secs, "waited_s", round3(waited), "ended", ended,
		"before", before.entry(), "after", after.entry())
	if readErr != nil {
		out.Set("error", readErr.Error())
	}
	out.Set("notes", subscribeNotes(ended, after, secs))
	t.recordRead("n4_subscribe_point", "ord", args, 1)
	return out, nil
}

// watchPoint re-reads ord every subCheckEvery seconds, for at most total seconds: it ends
// stale_cleared when a read shows no stale flag on a point that had it (not_stale on one that
// never had it), not_polled when one shows the point down,
// disabled or in fault (the driver does not poll it), read_failed on a failed read, and
// timeout when the time is up.
func (t *Tools) watchPoint(ord string, before subState, total float64) (subState, string, float64, error) {
	start, last := t.Ctx.Clock(), before
	for {
		waited := t.Ctx.Clock() - start
		if waited >= total {
			return last, "timeout", waited, nil
		}
		t.sleep(math.Min(subCheckEvery, total-waited))
		st, err := t.readSubState(ord)
		waited = t.Ctx.Clock() - start
		switch {
		case err != nil:
			return last, "read_failed", waited, err
		case st.notPolled():
			return st, "not_polled", waited, nil
		case !st.has("stale") && before.has("stale"):
			return st, "stale_cleared", waited, nil
		case !st.has("stale"):
			return st, "not_stale", waited, nil
		}
		last = st
	}
}

// subscribeNotes says why a point did not clear and what to check next.
func subscribeNotes(ended string, after subState, secs int) []any {
	switch ended {
	case "not_polled":
		return []any{fmt.Sprintf("the point is {%s}: its driver does not poll it while the point or its "+
			"device is down, disabled or in fault, so a subscription cannot clear stale; check the "+
			"device with n4_invoke_action ping on its device ext", strings.Join(after.flags, ", "))}
	case "timeout":
		return []any{fmt.Sprintf("still stale after %d s: the device may not answer this point, or the "+
			"point's poll rate (its tuning policy and poll frequency) is slower than the wait; ping "+
			"the device ext, then subscribe again with more seconds", secs)}
	}
	return []any{}
}
