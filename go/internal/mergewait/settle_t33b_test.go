package mergewait

import (
	"errors"
	"strings"
	"testing"
)

// T33b (the #718 verifier): when every send of the merge call fails on a network error and the PR stays OPEN,
// GitHub refused nothing; the outcome is its own exit, 10, with a message saying so, never exit 8.
func TestEveryMergeSendLostOnTheNetworkIsNotARefusal(t *testing.T) {
	code, out, errs, reads := settleReads(t, []reply{netErr(), netErr(), netErr(), netErr()})
	if code != ExitUnsent || out != "" || reads != 4 {
		t.Errorf("exit %d after %d read(s), out %q\n%s", code, reads, out, errs)
	}
	if !strings.Contains(errs, "no send of gh pr merge for #5 at "+head[:12]+" got an answer from GitHub") ||
		strings.Contains(errs, "gh pr merge failed") {
		t.Errorf("the message must say GitHub never answered, not that the merge failed:\n%s", errs)
	}
	// the safety rule holds on this path too: MERGED at the checked head is still the success
	code, out, errs, _ = settleReads(t, []reply{netErr(), netErr(), netErr(), netErr()}, view(false, head), merged(head))
	if code != ExitMerged || !strings.Contains(out, "merged #5 at "+head) {
		t.Errorf("merged after lost sends: exit %d, out %q\n%s", code, out, errs)
	}
}

// T33b: a network error followed by GitHub's own refusal is still exit 8 (a send reached GitHub).
func TestANetworkErrorThenARefusalStaysExit8(t *testing.T) {
	code, _, errs, _ := settleReads(t, []reply{netErr(), {err: errors.New(notMergeable)}})
	if code != ExitRefused || !strings.Contains(errs, "gh pr merge failed") {
		t.Errorf("exit %d\n%s", code, errs)
	}
}
