<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-c8-close-retire

**Session**: niagara-tools quality round 10, retire a close-gate pin whose fixture no longer exists; branch
`r10/c8-close-retire` from 591b88db (origin/main)
**Delta count**: 1

## What happened
`tests/c8-close.bats` SC16-smoke pinned exact `bog-audit.sh` row counts (CHECK11 FAIL x17, one CHECK14 WARN, two
CHECK18 FAIL) to the campaign-8 snapshot of a client station bog. That snapshot was only ever kept as an extracted copy
in a temporary agent-session directory, which has since been cleaned. Run by hand against a current station bog, the
test fails at its first count, because the station changed after the snapshot.

1. A read-only search of the user homes on 2026-10-25 for `*.bog` files dated in the snapshot's week found none: the
   fixture is gone and the pinned counts cannot be re-derived.
2. SC16-smoke now skips unconditionally right after `_close`, with a message that says why; its old body stays below
   the skip, unchanged, for history. Env var names and every other test are unchanged.
3. The `c8-close.bats` header records the retirement; `tests/bog-audit.bats` gets a comment above its hard-coded
   session-scratch default path (value and logic unchanged; its real-bog smokes already skip when the file is missing).

## Evidence
- `bats tests/c8-close.bats` without `C8_CLOSE`: every test skips. `C8_CLOSE=1 bats tests/c8-close.bats -f SC16`:
  one skip with the retirement message.
- No archived home for close-gate fixtures exists in the kit: `build-n4-module-kit/fixtures/` holds module skeletons
  (MinimalPan, MinimalDash, MinimalDriver) and `tests/fixtures/bog/` holds synthetic one-check bogs
  (`station-logic-CHECK13..19[-clean].bog`), neither a recorded close snapshot.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Close-gate pins must reference a durable archived fixture, never a session scratchpad. A close fixture is a client station bog, so it lives in a PRIVATE location outside the public repository (a private repository, or a configured private directory); the public gate records only the fixture's sha256 and an env var naming the private file, and SKIPs loudly when that file is absent or its sha256 does not match | `build-verify.md` § Toolbelt authoring | `[ev: code tests/c8-close.bats]` |

## Lessons
- A pin is only as durable as its fixture: exact counts against a file nobody archived turn into a test that can only
  fail or skip, and nobody can tell a regression from drift.
- The repository is public, so a client station bog never goes into it, not even anonymised as a close fixture: the
  public side keeps only a sha256 and the env var that names the private copy.

## Fold
- Δ1 is cited in `build-verify.md` § Toolbelt authoring (the close-gate fixture bullet).
- Follow-up, not done here: give the next real-bog close pin a private archived fixture, its recorded sha256 and the
  skip-on-mismatch guard. No in-repo location is proposed: the repository is public.

---
**Status**: FOLDED 2026-10-25 (branch `r10/c8-close-retire`; log in `ledger.d/2026-10-25-r10-c8-close-retire.md`) — Δ1 in `build-verify.md` § Toolbelt authoring.
