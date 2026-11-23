package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/capacity"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The capacity check of n4_copy_component and n4_replicate_points (retro R12 ΔM3): the copies of a
// device (a driver:Device subtype) or of a point's proxy extension (a driver:ProxyExt subtype) count
// against the licence's globalCapacity device.limit and point.limit like a create does, and a copy
// over a limit is added but sits in fault at start, so the dry run refuses it before a token.
const (
	proxyExtType = "driver:ProxyExt"
	// msgCapacityOver is the refusal: what the plan is, how many it adds, the count against the limit.
	msgCapacityOver = "%s would add %d %s: the station already counts %d/%d %s (licence %s %s), so the new ones would " +
		"sit in fault at start (modules not counted: %s); free some or raise the limit first " +
		"(n4_license_info shows the usage)"
	// noteCapacityUnread is a fixed note (no counts, so the plan hash does not move between the dry run
	// and the execute) when the licence sets a limit whose count could not be read.
	noteCapacityUnread = "The licence %s limit was not checked (the station's count could not be read); " +
		"this plan is not refused on it."
)

// capacityAdd is what a plan adds, by rule name and then by the module of the added type.
type capacityAdd map[string]map[string]int

// kinds says, once per type, which rules a type counts against; a type the station does not
// describe counts for none (Contracts.Is says false: never refuse on it).
type kinds map[string]map[string]bool

func (k kinds) of(ct *plan.Contracts, t string) map[string]bool {
	if m, ok := k[t]; ok {
		return m
	}
	m := map[string]bool{}
	for rule, super := range map[string]string{capacity.Devices.Name: deviceType, capacity.Points.Name: proxyExtType} {
		m[rule], _ = ct.Is(t, super)
	}
	k[t] = m
	return m
}

// addTree counts the devices and proxy extensions of node and its descendants, times copies.
func (a capacityAdd) addTree(k kinds, ct *plan.Contracts, node *pyjson.Object, copies int) error {
	kids, err := childObjects(node, "")
	if err != nil {
		return err
	}
	if t, ok := get(node, "t").(string); ok {
		module, _, _ := strings.Cut(t, ":")
		for rule, counts := range k.of(ct, t) {
			if counts {
				if a[rule] == nil {
					a[rule] = map[string]int{}
				}
				a[rule][module] += copies
			}
		}
	}
	for _, kid := range kids {
		if err := a.addTree(k, ct, kid, copies); err != nil {
			return err
		}
	}
	return nil
}

// withinLicence refuses a plan (what names it: "this copy") that takes the counted devices or proxy
// points over the licence, and returns a fixed note for a limit whose count could not be read. A
// licence that cannot be read, or that sets no limit, is not checked and does not touch the plan.
// walk adds what the plan adds to the capacityAdd it is given; it runs only when a limit is set.
func withinLicence(c plan.Client, what string, walk func(capacityAdd) error) ([]any, error) {
	s, _ := c.(capacity.SpyGetter)
	g, _ := c.(capacity.OrdGetter)
	info, _ := capacity.ReadLicence(s, licenceTimeout)
	if info == nil || capacity.TermOf(info, capacity.Devices).Unlimited && capacity.TermOf(info, capacity.Points).Unlimited {
		return nil, nil
	}
	a := capacityAdd{}
	if err := walk(a); err != nil {
		return nil, err
	}
	var notes []any
	for _, r := range capacity.Rules {
		if len(a[r.Name]) == 0 {
			continue
		}
		u, n, over, why := capacity.Add(g, info, r, a[r.Name], licenceTimeout)
		switch {
		case over:
			excluded := "none"
			if len(u.Excluded) > 0 {
				excluded = strings.Join(u.Excluded, ", ")
			}
			return nil, plan.ToolError(fmt.Sprintf(msgCapacityOver, what, n, r.Name, *u.Used, *u.Limit, r.Name,
				capacity.FeatureName, r.LimitKey, excluded))
		case why != "" && !u.Unlimited && (u.Used == nil || u.Limit == nil):
			notes = append(notes, fmt.Sprintf(noteCapacityUnread, r.Name))
		}
	}
	return notes, nil
}
