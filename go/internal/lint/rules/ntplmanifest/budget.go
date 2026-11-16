package ntplmanifest

import (
	"archive/zip"
	"fmt"
	"io"
)

// budget is what one lint run may still inflate (retro coordination-stretch T26): every template of a jar,
// and the manifest, template.bog and every Px entry of every archive, draw on it. A bound per archive let a
// small jar of many templates, each inflating its own bound, run for minutes; one budget for the run caps
// the work whatever the input holds. Once an entry does not fit, spent is set and the run reads nothing more.
type budget struct {
	left  int64
	spent bool
}

func newBudget() *budget { return &budget{left: maxInflate} }

// result is how an inflate ended.
type result int

const (
	whole      result = iota // data is the whole entry
	overLimit                // the entry holds more than its own bound: not read
	overBudget               // the entry holds more than the run has left: not read, and the run stops
	unread                   // the entry could not be inflated (err)
)

// inflate reads f within limit, its own bound, and what the run has left. The declared size is checked
// first, at no cost; the read is bounded too, so an entry that lies about its size costs at most one byte
// past the smaller bound. The bytes inflated are drawn from the budget whatever the result.
func (b *budget) inflate(f *zip.File, limit int64) (data []byte, res result, err error) {
	switch {
	case f.UncompressedSize64 > uint64(limit):
		return nil, overLimit, nil
	case f.UncompressedSize64 > uint64(b.left):
		// Mutation: NTPL-run-budget -- a budget per archive lets a jar of many templates inflate without bound
		b.spent = true
		return nil, overBudget, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, unread, err
	}
	defer rc.Close()
	data, err = io.ReadAll(io.LimitReader(rc, min(limit, b.left)+1))
	left := b.left
	b.left -= min(int64(len(data)), left)
	switch {
	case err != nil:
		return nil, unread, err
	case int64(len(data)) > limit:
		return nil, overLimit, nil
	case int64(len(data)) > left:
		b.spent = true
		return nil, overBudget, nil
	}
	return data, whole, nil
}

// spentMsg is the row of the entry the budget stopped at (name, or "the template" for a template of a jar).
func spentMsg(name string) string { return fmt.Sprintf(runBudget, maxInflate, name) }
