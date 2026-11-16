package ntplmanifest

import (
	"archive/zip"
	"fmt"
)

// readUses reads the entries the ntpl-uses check needs, in order, each within maxUsesRead inflated bytes and
// all of them within what the run has left of its budget (review of #702: per-entry bounds alone let an
// archive of many entries inflate without limit; T26: a bound per archive let many archives do the same). An
// entry over its own bound is a member with that message; the first entry the budget leaves no room for ends
// the read with one member naming it, and no later entry is opened.
func readUses(files []*zip.File, b *budget) []member {
	var out []member
	for _, f := range files {
		data, res, err := b.inflate(f, maxUsesRead)
		var why string
		switch res {
		case overBudget:
			// Mutation: NTPL-uses-total -- reading on past the run's budget lets a template of many entries inflate without limit
			return append(out, member{name: f.Name, why: spentMsg(f.Name)})
		case overLimit:
			why = fmt.Sprintf(entryTooLarge, f.Name, maxUsesRead)
		case unread:
			why = fmt.Sprintf(entryUnread, f.Name, err)
		default:
			var mods []string
			mods, why = modulesUsed(f.Name, data)
			out = append(out, member{name: f.Name, mods: mods, why: why})
			continue
		}
		out = append(out, member{name: f.Name, why: why})
	}
	return out
}
