// Package retrocells parses the cells of a retros/INDEX.md row that more than one kit sweep reads.
// It holds no sweep-specific types: sweep-fold-audit (go/internal/foldaudit) uses it today, and the
// Go port of sweep-build-state is meant to import it too. [ev: retro go-sweep-fold-audit]
package retrocells

import (
	"fmt"
	"strconv"
)

// MaxDeltas is the largest deltas count a cell may declare. A larger number is a typo or a corrupt
// row, never a real retro (the kit's largest has a few dozen), so it is unreadable instead of
// enumerating millions of Δ entries.
const MaxDeltas = 999

// Delta is one proposed kit delta of a retro, Δk in its "Proposed kit deltas" table and in
// citations such as `[ev: retro <stem> Δk]`.
type Delta struct {
	K int // the 1-based delta number
}

// String is the citation form, "Δ<k>".
func (d Delta) String() string { return "Δ" + strconv.Itoa(d.K) }

// CellError reports a deltas cell that is not a readable count.
type CellError struct {
	Cell   string // the cell text as given
	Reason string
}

func (e *CellError) Error() string { return fmt.Sprintf("deltas cell %q: %s", e.Cell, e.Reason) }

// ParseDeltasCell reads a deltas cell (callers remove the padding spaces first) as a decimal count n
// and returns Δ1..Δn in order (an empty, non-nil slice for "0"). The cell must be ASCII digits only;
// a leading zero is still decimal ("010" is 10, "09" is 9), unlike bash arithmetic, which read it as
// octal. An empty, non-numeric or oversize (> MaxDeltas) cell returns a *CellError: the caller must
// report it as unreadable (fail closed), never treat it as a count.
func ParseDeltasCell(cell string) ([]Delta, error) {
	if cell == "" {
		return nil, &CellError{Cell: cell, Reason: "empty"}
	}
	n := 0
	for i := 0; i < len(cell); i++ {
		c := cell[i]
		if c < '0' || c > '9' {
			return nil, &CellError{Cell: cell, Reason: "not a decimal count"}
		}
		if n = n*10 + int(c-'0'); n > MaxDeltas {
			return nil, &CellError{Cell: cell, Reason: fmt.Sprintf("more than %d deltas", MaxDeltas)}
		}
	}
	out := make([]Delta, n)
	for k := range out {
		out[k] = Delta{K: k + 1}
	}
	return out, nil
}
