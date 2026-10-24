package xmlsrc

import (
	"fmt"
	"sort"
)

// Error is a malformed document: "<path>:<line>:<col>: <msg>". For a check of this package the
// position is the offending tag or text; for a check of encoding/xml (entities, character ranges,
// UTF-8, tag syntax) it is where the decoder stopped reading, at or just past the offending bytes.
type Error struct {
	Path      string
	Line, Col int
	Msg       string
}

func (e *Error) Error() string { return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Col, e.Msg) }

// positions maps a byte offset to its line and column.
type positions struct{ starts []int64 }

func newPositions(data []byte) positions {
	starts := []int64{0}
	for i, b := range data {
		if b == '\n' {
			starts = append(starts, int64(i+1))
		}
	}
	return positions{starts}
}

func (p positions) at(off int64) (line, col int) {
	i := sort.Search(len(p.starts), func(i int) bool { return p.starts[i] > off }) - 1
	return i + 1, int(off-p.starts[i]) + 1
}
