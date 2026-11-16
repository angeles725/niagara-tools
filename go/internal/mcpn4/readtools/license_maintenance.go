package readtools

import (
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// maintenanceEntry is what the page states of the software maintenance agreement: per feature
// its sma.* attributes, the features whose sma.exempt is true (each once), both under their
// caps with the whole counts beside them, and a feature named after maintenance. expires is
// null: the page carries no end date of it (maintenanceNote).
func maintenanceEntry(info *license.Info, clock licenseClock) *pyjson.Object {
	groups, feature, exempt := info.Maintenance()
	names, list := newCapped(capExempt), newCapped(capSMA)
	for _, name := range exempt {
		if !names.add(name) {
			break
		}
	}
	for _, g := range groups {
		entry := obj("feature", g.Feature, "attributes", attributes(g.Attrs))
		if g.AttrsCut {
			entry.Set("attributes_truncated", true)
		}
		if !list.add(entry) {
			break
		}
	}
	out := obj("expires", nil, "sma_exempt_features", names.items, "sma_exempt_count", len(exempt),
		"sma_attributes", list.items, "sma_feature_count", len(groups),
		"sma_truncated", len(names.items) < len(exempt) || len(list.items) < len(groups), "feature", nil, "note", maintenanceNote)
	if feature != nil {
		entry := featureEntry(*feature)
		entry.Set("days_left", clock.daysLeft(feature.Expires))
		out.Set("feature", entry)
	}
	return out
}

// maintenance adds the end date of the maintenance agreement to its entry: the earliest
// maintenanceExpiration the files state, with the licence that states it; without one, the
// note says whether files were read at all.
func (l licenseFiles) maintenance(entry *pyjson.Object, clock licenseClock) {
	names := make([]string, 0, len(l.byName))
	for name := range l.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	best := ""
	for _, name := range names {
		date := l.byName[name].MaintenanceExpiration
		if _, ok := license.Day(date); ok && (best == "" || date < l.byName[best].MaintenanceExpiration) {
			best = name
		}
	}
	switch {
	case best != "":
		date := l.byName[best].MaintenanceExpiration
		entry.Set("expires", date)
		entry.Set("days_left", clock.daysLeft(date))
		entry.Set("license", best)
		entry.Set("note", maintenanceFromFile)
	case len(l.byName) > 0:
		entry.Set("note", maintenanceNoDate)
	}
}
