package prepush

import "github.com/angeles725/niagara-tools/go/internal/changelog"

// gateRef gates one pushed ref and returns its status: 0 lets it through. The order is the contract:
// the citation gate and the changelog gate run before the retro gate, whose early exits waive retro
// bookkeeping and nothing else.
func (h hook) gateRef(localSha, remoteSha string) int {
	if isDeletion(localSha) {
		return 0
	}
	rng, ok := h.pushedRange(localSha, remoteSha)
	if !ok {
		return 1
	}
	changedOut, ok := gitOut(h.s.Err, "diff", "--name-only", rng)
	if !ok {
		return refuse(h.s.Err, msgDiffFailed, rng)
	}
	log, ok := gitOut(h.s.Err, "log", "--format=%B", rng)
	if !ok {
		return refuse(h.s.Err, msgLogFailed, rng)
	}
	changed := lines(changedOut)

	if rc := h.liveCiteGate(changed, localSha); rc != 0 {
		return rc
	}
	if rc := changelog.Main([]string{"--range", rng}, h.s); rc != 0 {
		return 1
	}
	return h.retroGate(rng, changed, log)
}
