package niagaratest

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The walk is bounded: a nuh\ over the entry cap or deeper than the depth cap is an error, never a silent pass.
func TestWalkNUHBounds(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	deep := root
	for range nuhMaxDepth + 1 {
		deep = filepath.Join(deep, "d")
	}
	write(t, filepath.Join(deep, "f"), "x")
	if _, err := walkNUH(root); err == nil || !strings.Contains(err.Error(), "deeper than") {
		t.Errorf("walk of a nuh %d dirs deep: err %v, want a depth error", nuhMaxDepth+1, err)
	}
	wide := t.TempDir()
	for i := range nuhMaxEntries + 1 {
		write(t, filepath.Join(wide, "d"+strconv.Itoa(i%100), "f"+strconv.Itoa(i)), "")
	}
	if _, err := walkNUH(wide); err == nil || !strings.Contains(err.Error(), "over ") {
		t.Errorf("walk of a nuh with %d entries: err %v, want an entry-cap error", nuhMaxEntries+1, err)
	}
	if s, err := walkNUH(filepath.Join(root, "absent")); err != nil || len(s) != 0 {
		t.Errorf("walk of an absent nuh: %v %v, want empty", s, err)
	}
}
