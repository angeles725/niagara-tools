package prepush

import (
	"fmt"
	"io"
)

// Every line the hook prints of its own, word for word what the bash hook printed.
const (
	refusal = "pre-push: %s; refusing to push unchecked\n"

	msgRefused         = "pre-push: refused %s -> %s (exit %d)\n"
	msgStdinUnreadable = "cannot read the pushed refs from stdin (%v)"

	msgGuardMissing = "pre-push: client-guard is not on PATH but a client denylist exists; refusing to push unchecked.\n" +
		"          install it from a niagara-go checkout: GOBIN=$HOME/.local/bin go install ./cmd/client-guard\n"
	msgGuardSkipped = "pre-push: client-guard not installed and no client denylist; client-identifier check skipped\n"

	msgNoBase     = "cannot find where %s left origin/main (git fetch origin first)"
	msgDiffFailed = "git diff %s failed"
	msgLogFailed  = "git log %s failed"

	msgLsTreeFailed   = "git ls-tree %s failed"
	msgTreeDiffers    = "the working tree differs from the pushed %s on the live doctrine set or the resolver (check out the pushed commit)"
	msgCiteFindings   = "live doctrine has unresolved pointers (kit-cite-resolve --strict; rows above, first 20)"
	msgCiteSkipped    = "pre-push: live cite check SKIPPED — roots absent (%s %s)\n"
	msgCiteToolFailed = "kit-cite-resolve failed (exit %d; above), a tool failure is not a zero result"

	msgPatchFailed = "git diff %s -- retros failed"
	msgMktemp      = "mktemp failed"
	msgIncoherent  = "pre-push: ledger data in range is incoherent (BUILD-STATE.src.md, ledger.d/ or retros/; sweep-build-state.sh above); refusing to push\n"
	msgNoRetro     = "pre-push: build-relevant kit files changed without a ledger.d/ fragment (or a BUILD-STATE.src.md edit) + a pending retro,\n" +
		"          and no 'Retro: none (trivial: <reason>)' trailer.\n" +
		"          See BUILD-LOOP.md §7.\n"

	msgCannotRun = "pre-push: cannot run %s: %v\n"
)

// printf writes one of the messages above.
func printf(w io.Writer, format string, args ...any) { fmt.Fprintf(w, format, args...) }

// refuse prints "pre-push: <reason>; refusing to push unchecked" and returns the refusal status 1.
func refuse(w io.Writer, reason string, args ...any) int {
	fmt.Fprintf(w, refusal, fmt.Sprintf(reason, args...))
	return 1
}
