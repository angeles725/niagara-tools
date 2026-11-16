package ngdeploy

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// race starts n+1 runs that take the lock of dir at the same moment and returns how each ended and what
// each printed on stderr. The one that took it keeps it until the test calls release.
func race(t *testing.T, dir string, n int) (codes []int, errs []string, release func()) {
	t.Helper()
	deploys, outs := make([]*deploy, n+1), make([]bytes.Buffer, n+1)
	codes, errs = make([]int, n+1), make([]string, n+1)
	gate := make(chan struct{})
	var wg sync.WaitGroup
	for i := range deploys {
		deploys[i] = &deploy{s: cli.Stdio{Out: io.Discard, Err: &outs[i]}}
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-gate; codes[i] = deploys[i].lock(dir) }(i)
	}
	close(gate)
	wg.Wait()
	for i := range outs {
		errs[i] = outs[i].String()
	}
	return codes, errs, func() {
		for _, d := range deploys {
			d.unlock()
			if d.signals != nil {
				d.signals.stop()
			}
		}
	}
}

// D5: the lock has its content from the moment it is there. 200 runs that lose the race to the one that
// takes it, which is alive, each read who holds it: none says "it does not say which run made it".
func TestEveryRunThatLosesTheLockRaceReadsWhoHoldsIt(t *testing.T) {
	dir := t.TempDir()
	codes, errs, release := race(t, dir, 200)
	defer release()
	pid := strconv.Itoa(os.Getpid())
	won := 0
	for i, code := range codes {
		if code == proceed && errs[i] == "" {
			won++
			continue
		}
		m := lockedLine.FindStringSubmatch(errs[i])
		if code != ExitEnv || m == nil || m[1] != lockFile(dir) || m[2] != pid || m[8] != "ngdeploy.test" {
			t.Errorf("run %d: exit %d, stderr %q", i, code, errs[i])
		}
	}
	if won != 1 {
		t.Errorf("%d runs took the lock, want 1", won)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 1 {
		t.Errorf("the directory holds %v (%v), want the lock alone", entries, err)
	}
}
