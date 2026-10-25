package newretro

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConcurrentRunsKeepEveryFlip runs one new-retro per module at once against one kit root and checks that
// every module's retro_pending flip, every stub and every INDEX.md row survives. The afterRead hook is a
// barrier: each run waits there, just after reading BUILD-STATE.src.md, until all runs have read it (or, when
// runs are serialised so the others cannot arrive, for a short timeout). Without a lock every run reaches the
// barrier, so all read the same envelope before any writes it: the read-read-write-write interleaving is
// forced, not left to timing, and the later renames drop the earlier flips.
// Mutation: NR-lock -- dropping the retros/ lock loses all but one flip in this test.
func TestConcurrentRunsKeepEveryFlip(t *testing.T) {
	const runs = 4
	root := fixture(t)
	var state strings.Builder
	for i := range runs {
		fmt.Fprintf(&state, "<!-- build-state.v1 -->\n## m%d\nretro_pending: false\n<!-- /build-state.v1 -->\n", i)
	}
	write(t, filepath.Join(root, "BUILD-STATE.src.md"), state.String())

	var mu sync.Mutex
	arrived, all := 0, make(chan struct{})
	saved := afterRead
	t.Cleanup(func() { afterRead = saved })
	afterRead = func() {
		mu.Lock()
		if arrived++; arrived == runs {
			close(all)
		}
		mu.Unlock()
		select {
		case <-all:
		case <-time.After(100 * time.Millisecond):
		}
	}

	var wg sync.WaitGroup
	codes := make([]int, runs)
	for i := range runs {
		wg.Go(func() { codes[i], _, _ = runIn(root, fmt.Sprintf("m%d", i), fmt.Sprintf("slug-m%d", i)) })
	}
	wg.Wait()

	got := tree(t, root)
	for i := range runs {
		stub := fmt.Sprintf("2026-01-02-slug-m%d.md", i)
		if codes[i] != ExitOK {
			t.Errorf("run m%d: exit %d, want %d", i, codes[i], ExitOK)
		}
		if _, ok := got["retros/"+stub]; !ok {
			t.Errorf("stub %s is missing", stub)
		}
		if !strings.Contains(got["retros/INDEX.md"], "| "+stub+" |") {
			t.Errorf("INDEX.md has no row for %s:\n%s", stub, got["retros/INDEX.md"])
		}
		if want := fmt.Sprintf("## m%d\nretro_pending: true\n", i); !strings.Contains(got["BUILD-STATE.src.md"], want) {
			t.Errorf("the flip of m%d was lost:\n%s", i, got["BUILD-STATE.src.md"])
		}
	}
}
