package readtools

import (
	"fmt"
	"strings"
)

// undecodableNote is the note of a declared default the reader cannot decode: left out, named.
const undecodableNote = "the declared default of %s could not be decoded (%s): it is not listed"

// addNote adds note once, in the order the listing met it.
func (d *declared) addNote(note string) {
	for _, n := range d.notes {
		if n == note {
			return
		}
	}
	d.notes = append(d.notes, note)
}

// skip notes that the declared default at path could not be decoded.
func (d *declared) skip(path string, err error) {
	d.addNote(fmt.Sprintf(undecodableNote, path, err.Error()))
}

// noteText is the listing's defaults_note: every note, "" when none.
func (d *declared) noteText() string { return strings.Join(d.notes, "; ") }
