package readtools

import (
	"fmt"
	"math"
	"sync"
)

// n4_histories_audit progress (issue #906, retro R8 delta N5): a line per stage and one per
// haProgressEvery finished histories of the counts and flat stages go to the operator's progress
// file, the channel n4_inventory uses (--progress-file; nothing is written without it). Each line
// carries the requests made so far in rows and the elapsed ms; the reply does not carry them.

// haProgressEvery is how many finished histories of a stage make a progress line.
const haProgressEvery = 10

// haProgress is the progress of the pooled stage running: the requests made before it, the
// histories it has finished and the reads they made. mu serializes the workers' lines, so the
// counts in the file only grow.
type haProgress struct {
	mu          sync.Mutex
	base        int
	done, reads int
}

// progress writes one progress line: step, with requests made so far.
func (r *haRun) progress(step string, requests int) {
	elapsed := int(math.RoundToEven((r.t.Ctx.Clock() - r.start) * 1000))
	r.t.progress("n4_histories_audit", step, requests, elapsed, nil)
}

// stage writes the line that ends the stage step (its requests are counted in r.requests).
func (r *haRun) stage(step string) { r.progress(step, r.requests) }

// beginPool starts the progress of a pooled stage; r.requests is not final until it ends.
func (r *haRun) beginPool() {
	r.pool.mu.Lock()
	r.pool.base, r.pool.done, r.pool.reads = r.requests, 0, 0
	r.pool.mu.Unlock()
}

// finished notes one history of the stage step (total in it) done after reads station reads; it
// is called from the pool's workers.
func (r *haRun) finished(step string, total, reads int) {
	p := &r.pool
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done, p.reads = p.done+1, p.reads+reads
	if p.done%haProgressEvery == 0 && p.done < total {
		r.progress(fmt.Sprintf("%s %d/%d", step, p.done, total), p.base+p.reads)
	}
}
