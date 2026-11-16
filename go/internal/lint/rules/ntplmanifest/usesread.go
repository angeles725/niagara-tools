package ntplmanifest

import (
	"archive/zip"
	"fmt"
	"io"
)

// readUses reads the entries the ntpl-uses check needs, in order, each within maxUsesRead inflated bytes and
// all of them within maxUsesTotal (review of #702: per-entry bounds alone let an archive of many entries
// inflate without limit). An entry over its own bound is a member with that message and costs nothing; the
// first entry that would pass the archive's budget ends the read with one member naming it, and no later
// entry is opened.
func readUses(files []*zip.File) []member {
	var out []member
	left := int64(maxUsesTotal)
	for _, f := range files {
		if f.UncompressedSize64 > maxUsesRead {
			out = append(out, member{name: f.Name, why: fmt.Sprintf(entryTooLarge, f.Name, maxUsesRead)})
			continue
		}
		budget := member{name: f.Name, why: fmt.Sprintf(usesBudget, maxUsesTotal, f.Name)}
		if f.UncompressedSize64 > uint64(left) {
			// Mutation: NTPL-uses-total -- dropping the archive's budget lets a template of many entries inflate without limit
			return append(out, budget)
		}
		limit := min(int64(maxUsesRead), left)
		data, why, over := inflateEntry(f, limit)
		if over && limit < maxUsesRead { // the entry holds more than its header says, past what is left
			return append(out, budget)
		}
		left -= int64(len(data))
		if why == "" {
			var mods []string
			mods, why = modulesUsed(f.Name, data)
			out = append(out, member{name: f.Name, mods: mods, why: why})
			continue
		}
		out = append(out, member{name: f.Name, why: why})
	}
	return out
}

// inflateEntry reads f up to limit bytes; why is its ntpl-uses message when it could not be read whole, and
// over reports that it holds more than limit bytes.
func inflateEntry(f *zip.File, limit int64) (data []byte, why string, over bool) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Sprintf(entryUnread, f.Name, err), false
	}
	defer rc.Close()
	data, err = io.ReadAll(io.LimitReader(rc, limit+1))
	switch {
	case err != nil:
		return nil, fmt.Sprintf(entryUnread, f.Name, err), false
	case int64(len(data)) > limit:
		return nil, fmt.Sprintf(entryTooLarge, f.Name, maxUsesRead), true
	}
	return data, "", false
}
