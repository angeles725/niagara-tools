package readtools

import (
	"fmt"
	"math"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_commissioning_report: what is still wrong or unfinished below a scope, in one reply made
// of reads the server already has (BQL tables, the link scan, slot loads, the licence page).
// Every finding is a fact the station answered; a check whose read failed, or that the station
// cannot answer, says so and is never counted as clean. Only an authentication failure fails
// the call, so the server latches on it like on any other read.

// commBuilders build each section, by name.
var commBuilders = map[string]func(*commRun) (*commSection, error){
	"points": (*commRun).pointsSection, "histories": (*commRun).historiesSection, "alarms": (*commRun).alarmsSection,
	"links": (*commRun).linksSection, "naming": (*commRun).namingSection, "station": (*commRun).stationSection,
	"spyder": (*commRun).spyderSection,
}

// sectionsArg is the sections to build, in reply order (default: commDefaultSections, plus
// spyder when spyder_spec is given).
func sectionsArg(args *pyjson.Object) ([]string, error) {
	v, ok := args.Get("sections")
	if !ok {
		if _, spec := args.Get("spyder_spec"); spec || boolArg(args, "spyder_hardware", false) {
			return commSections, nil
		}
		return commDefaultSections, nil
	}
	wanted := map[string]bool{}
	for _, item := range v.([]any) { // an array of commSections names: the schema's enum
		name, _ := item.(string)
		wanted[name] = true
	}
	var out []string
	for _, name := range commSections {
		if wanted[name] {
			out = append(out, name)
		}
	}
	return out, nil
}

// newCommRun reads the arguments of one report.
func (t *Tools) newCommRun(args *pyjson.Object) (*commRun, error) {
	scope := strArg(args, "scope", RootOrd)
	slot, err := bql.BaseSlot(scope)
	if err != nil {
		return nil, toolError(commBadScope, pyjson.StrRepr(scope))
	}
	r := &commRun{t: t, scope: bql.StationSlot + slot, slot: slot, limit: intArg(args, "limit", commDefaultLimit),
		maxRows: intArg(args, "max_components", bql.DefaultMaxRows), timeout: intArg(args, "timeout_s", commDefaultTimeout),
		depth: intArg(args, "link_depth", commDefaultLinkDepth), tables: map[string]*commTable{}}
	if r.expectHistories, err = expectArg(args, "expect_histories"); err != nil {
		return nil, err
	}
	if r.expectAlarms, err = expectArg(args, "expect_alarms"); err != nil {
		return nil, err
	}
	if r.spyderSpec, err = spyderSpecArg(args); err != nil {
		return nil, err
	}
	r.spyderHardware = boolArg(args, "spyder_hardware", false)
	return r, nil
}

// CommissioningReport is n4_commissioning_report.
func (t *Tools) CommissioningReport(args *pyjson.Object) (*pyjson.Object, error) {
	names, err := sectionsArg(args)
	if err != nil {
		return nil, err
	}
	r, err := t.newCommRun(args)
	if err != nil {
		return nil, err
	}
	start := t.Ctx.Clock()
	summary, sections := []any{}, pyjson.NewObject()
	for _, name := range names {
		s, err := commBuilders[name](r)
		if err != nil {
			return nil, err
		}
		section, line := s.reply()
		summary = append(summary, line)
		sections.Set(name, section)
	}
	out := obj("scope", r.scope, "summary", summary, "requests", r.requests, "requests_note", commRequests,
		"truncated", r.truncated > 0)
	if r.truncated > 0 {
		out.Set("truncated_note", fmt.Sprintf(commTruncated, r.truncated))
	}
	for _, name := range sections.Keys() {
		out.Set(name, lookup(sections, name))
	}
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_commissioning_report", "scope", args, r.rows)
	return out, nil
}
