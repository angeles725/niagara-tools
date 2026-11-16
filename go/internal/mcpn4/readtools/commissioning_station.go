package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

// The station section, only for the whole station: the services in fault, a licence expiring
// soon (the licence section of n4_station_health), and the two states the station does not
// expose to a read, each marked not checked with the reason. Nothing here is guessed.

// stationSection is the station section.
func (r *commRun) stationSection() (*commSection, error) {
	s := r.section("station")
	names := []string{"services_in_fault", "licence_expiry", "unsaved_changes", "last_backup"}
	if r.slot != "/" {
		for _, name := range names {
			s.check(name).skipped = skipNotStation
		}
		return s, nil
	}
	if err := r.servicesInFault(s.check(names[0])); err != nil {
		return nil, err
	}
	if err := r.licenceExpiry(s.check(names[1])); err != nil {
		return nil, err
	}
	s.check(names[2]).why, s.check(names[3]).why = whyUnsaved, whyBackup
	return s, nil
}

// servicesInFault lists in c the components one or two levels below Services (a service, or a
// service inside a container) whose status slot holds the fault flag, with their faultCause.
// A component with no status slot in the load is at the default status, ok.
func (r *commRun) servicesInFault(c *commCheck) error {
	r.requests++
	nodes, _, err := r.t.timedLoad(servicesOrd, servicesDepth)
	if err != nil {
		c.why, err = r.failure("the slot load of "+servicesOrd+" failed", err)
		return err
	}
	checked := 0
	for _, p := range nodes.Keys() {
		if h, _ := box.Node(nodes, p).Get("h"); p == "" || level(p) > 2 || h == nil || h == "" {
			continue
		}
		checked++
		flags, err := box.StatusFlags(box.ChildValue(box.Children(nodes, p), "status", "0"))
		if err != nil || !contains(flags, faultFlag) {
			continue
		}
		fact := fmt.Sprintf(factService, strings.Join(flags, ", "))
		if cause, _ := box.ChildValue(box.Children(nodes, p), "faultCause", "").(string); strings.TrimSpace(cause) != "" {
			fact += ": " + bql.StripControl(cause)
		}
		c.add(absSlot(servicesOrd, p), bql.UnescapeSlot(lastSegment(p)), fact)
	}
	c.extra = []any{"checked", checked}
	return nil
}

// absSlot is the slot ORD of path below ord.
func absSlot(ord, path string) string { return slotPrefix + absPath(ord, path) }

// licenceExpiry puts in c the warning of the health tool's licence section (an expiry within
// its window or past, a licence that is not valid). An unreadable page, or a station date the
// expiry cannot be compared with, is not checked.
func (r *commRun) licenceExpiry(c *commCheck) error {
	r.requests += 2 // the SystemService slots (the station's own date) and the licence page
	system, err := r.t.healthSystem()
	if err != nil {
		return err
	}
	section, err := r.t.healthLicense(r.timeout, system)
	if err != nil {
		return err
	}
	warning, warned := lookup(section, "warning").(string)
	clock := stationClock(system)
	switch failed, isErr := lookup(section, "error").(string); {
	case isErr:
		c.why = failed
	case warned:
		c.add(licenseSource, "licence", warning)
	case !clock.known:
		c.why = fmt.Sprintf(whyNoClock, clock.why)
	}
	if nearest := lookup(section, "nearest_expiry"); nearest != nil {
		c.extra = []any{"nearest_expiry", nearest}
	}
	return nil
}
