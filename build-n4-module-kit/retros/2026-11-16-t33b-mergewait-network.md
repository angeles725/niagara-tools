<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-11-16 · kit · t33b-mergewait-network

**Session**: row T33b of the 2026-11-15 coordination-stretch retro, a pre-existing finding of the #718 verifier.
**Delta count**: 1

## What happened
When every send of the merge call failed on a network error and the PR stayed OPEN, merge-when-green exited 8
with "gh pr merge failed … nothing merged", the refusal outcome, although GitHub never answered any send. That path
is now exit 10 with its own message ("no send of gh pr merge … got an answer from GitHub … GitHub refused nothing
and nothing merged: run this again when the network is back"), documented in the help and CONTRIBUTING.md § 8.
Exit 10 was unused (0-9 taken). A network error followed by GitHub's refusal stays 8, a PR CLOSED after lost sends
stays 8, and MERGED at the checked head after lost sends stays the success.

## Evidence
- RED before GREEN: TestEveryMergeSendLostOnTheNetworkIsNotARefusal (exit written as 10) failed on origin/main with
  exit 8 and the "gh pr merge failed" line. `[ev: go/internal/mergewait/settle_t33b_test.go]`
- Mutation, not committed: replacing the new case with `case false:` turns it red.
  `[ev: go/internal/mergewait/settle.go]`
- Differential with the stub gh of tests/merge-when-green.bats, origin/main binary against this one, 13 scenarios
  (green, draft, behind, failed check, refusal, pending, network then refusal, all sends lost, lost then CLOSED, lost
  then MERGED, reset then OPEN, usage, help): only "all sends lost" (8 → 10, stderr) and the help text differ.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. merge-when-green exits 10, not 8, when every merge send failed on a network error and the PR stays OPEN | `go/internal/mergewait/settle.go` | `[ev: go/internal/mergewait/settle_t33b_test.go]` |
