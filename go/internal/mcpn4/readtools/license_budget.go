package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// capped is a list of a licence reply filled under a listCap: once an entry does not fit,
// nothing more is kept and cut is set.
type capped struct {
	limit listCap
	items []any
	bytes int
	cut   bool
}

func newCapped(limit listCap) *capped { return &capped{limit: limit, items: []any{}} }

// add keeps entry when it fits and reports whether more may follow.
func (c *capped) add(entry any) bool {
	if c.cut {
		return false
	}
	raw, err := pyjson.Marshal(entry, pyjson.Options{})
	if err != nil || len(c.items) >= c.limit.items || c.bytes+len(raw) > c.limit.bytes {
		c.cut = true
		return false
	}
	c.items, c.bytes = append(c.items, entry), c.bytes+len(raw)
	return true
}

// someNames is up to licenseNoteNames of names, then how many more there are.
func someNames(names []string) string {
	if len(names) <= licenseNoteNames {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:licenseNoteNames], ", ") + fmt.Sprintf(andMore, len(names)-licenseNoteNames)
}

// joinNotes is notes as one note of at most licenseNoteRunes characters, "" for none.
func joinNotes(notes []string) string {
	return license.Clip(strings.Join(notes, "; "), licenseNoteRunes)
}
