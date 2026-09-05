<!-- review-status: folded -->
<!-- kit-retro -->
# §7 close-gate exit taxonomy: a PROMOTION is neither a new retro nor a trivial change

Date: 2026-09-04 · Module: kit (.githooks/pre-push + BUILD-LOOP §7) · SDD: build-n4-module-continuity Campaign 2

## The lesson (surfaced by dogfooding Campaign 2)

Campaign 1's §7 gate offered exactly TWO exits for a build-relevant kit change: (a) write a NEW
retro + INDEX row + BUILD-STATE update, or (b) declare it TRIVIAL (`Retro: none (trivial: …)`).
Campaign 2's promotion PRs — folding ALREADY-FILED lessons from `retros/` into the core — fit
NEITHER: they discover no new lesson (so (a) is redundant) and they are substantive documentation
work (so (b) would be a FALSE `trivial` label — the "wrong value in the record" the campaign fights).
Refusing to mislabel a promotion as trivial is what exposed the gap.

## The fix (applied this PR)

A THIRD exit, HOOK-ONLY (the trailer is a commit/diff concept; `sweep-build-state.sh` stays
VCS-free per kit-links L2):

- Commit trailer `Retro: promotion (folds <ids> from existing retros)`, AND
- an in-range `retros/INDEX.md` diff (the registry move that flips folded/pending marks).

Both are required: a promotion trailer with NO INDEX diff FAILS (fail-closed), so the exit cannot
become a blanket escape. Documented as §7 exit (c); enforced in `.githooks/pre-push` parallel to the
trivial waiver; asserted by `tests/build-retro-sync.bats` H8 (pass) / H9 (fail).

## Proposed kit deltas

- Rule to keep: a gate's exit set must cover EVERY legitimate change class it will actually see —
  when a real workflow (promotion) matches no exit, add a typed exit with its own proof-of-work
  guard, never stretch an existing label to fit.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| The gate had only two exits before this PR | [CERT] | BUILD-LOOP §7 (a)/(b) at 949aa30 |
| The third exit is fail-closed without an INDEX diff | [CERT] | build-retro-sync.bats H9 asserts FAIL |
| The trailer requires an in-range INDEX.md move | [CERT] | `.githooks/pre-push` promotion branch: `grep 'Retro: promotion (folds' && has_index>=1` |

Connections: [[2026-09-04-kit-continuity-and-retro-gate-campaign]]; BUILD-LOOP.md §7 exit (c).
