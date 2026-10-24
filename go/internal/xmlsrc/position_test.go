package xmlsrc

import "testing"

func TestPosition(t *testing.T) {
	p := newPositions([]byte("ab\ncd\n\nx"))
	for off, want := range map[int64][2]int{0: {1, 1}, 1: {1, 2}, 2: {1, 3}, 3: {2, 1}, 6: {3, 1}, 7: {4, 1}, 8: {4, 2}} {
		if l, c := p.at(off); l != want[0] || c != want[1] {
			t.Errorf("at(%d) = %d:%d, want %d:%d", off, l, c, want[0], want[1])
		}
	}
}
