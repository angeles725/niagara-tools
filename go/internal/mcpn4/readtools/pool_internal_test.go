package readtools

import (
	"strings"
	"sync/atomic"
	"testing"
)

// A panic in one pooled read must not end the server: runPool returns it as that item's error and the other items
// still run (issue #968 item 1).
func TestRunPoolReturnsAPanicAsThatItemsErrorAndRunsTheRest(t *testing.T) {
	var ran atomic.Int32
	errs := runPool(9, func(i int) {
		if i == 3 {
			panic("boom")
		}
		ran.Add(1)
	})
	for i, err := range errs {
		if bad := i == 3 && (err == nil || !strings.Contains(err.Error(), "boom")) || i != 3 && err != nil; bad {
			t.Errorf("errs[%d] = %v", i, err)
		}
	}
	if len(errs) != 9 || ran.Load() != 8 {
		t.Errorf("%d errs, %d items ran, want 9 and the 8 that do not panic", len(errs), ran.Load())
	}
}
