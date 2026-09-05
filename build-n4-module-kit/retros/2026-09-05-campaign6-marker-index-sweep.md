<!-- review-status: folded -->
<!-- kit-retro -->
# Campaign 6 PR1: marker-aware sweep-build-state — the marker↔INDEX contract (2-lite)

Date: 2026-09-05 · Module: kit · SDD: build-n4-module-campaign6 PR1 (v0.15.1 → v0.16.0)
[ev: QA RED cb0dd7d]

`sweep-build-state.sh` validated only the INDEX.md ledger; it never opened a retro file to check
its in-file `<!-- review-status: X -->` comment against the INDEX row. Two retros that had already been
promoted (their INDEX rows say `folded`) still had `fresh · 2026-09-02` markers — an undetected drift.
Campaign 6 PR1 closes that gap with the option 2-lite marker contract: read the marker from the
first 5 lines, compare the first word against the INDEX row, fail loud on any mismatch or unknown word.

## What this PROVED / the deltas

1. **The sweep was marker-blind; the two fresh-marker retros were invisible to it.**
   QA staged the RED first (`qa/c6-marker-index-drift` tip `cb0dd7d`, 25 tests):
   M1 (disagreement → exit 1) and M4 (out-of-domain `fresh · DATE` → exit 1) FAILED because the sweep
   returned exit 0 in both cases. The real tree also had live drift: `2026-09-02-comppan-fase1-staging.md`
   and `2026-09-02-dashboardpan-detail-render-doors.md` both carried `fresh · 2026-09-02` while their
   INDEX rows said `folded`. M5 (real tree → exit 0) only passed because the sweep never read the markers.

2. **Three real marker shapes exist on the tree; first-word parsing is the only rule that covers all three.**
   The three shapes found in the retros directory at the start of Campaign 6 were:
   - `<!-- review-status: folded -->` (bare — no suffix)
   - `<!-- review-status: fresh · 2026-09-02 -->` (suffix: ` · DATE`)
   - `<!-- review-status: pending -->` (bare — no suffix)
   An exact-match approach would have silently skipped the suffixed `fresh` markers and left the drift
   undetected. First-word parsing (`awk '{print $1}'` after stripping the prefix) handles all three.
   → **DELTA (option 2-lite contract):** `marker_of()` function in `sweep-build-state.sh`:
     `sed -n '1,5p'` (first-5-line window) + `grep -m1 -E '^<!--[[:space:]]*review-status:'`
     (column-0 anchor) + prefix-strip `sed` + `awk '{print $1}'` (first word).
     Domain check `{pending|folded}` → exit 1 out-of-domain.
     Marker ≠ INDEX row → exit 1 naming file + both values.
     Absent marker → tolerated (exit 0 for that file — protects legacy folded files).

3. **The column-0 anchor and 5-line window prevent false positives from prose mentions.**
   The kit-continuity retro (`2026-09-04-kit-continuity-and-retro-gate-campaign.md`) mentions
   `review-status:` inside a table cell near line 40. Without the column-0 `^<!--` anchor and the
   first-5-line window, a naive `grep -m1 review-status:` would grab that prose word, compare it
   against the INDEX row, and wrongly fail. Test M6 (prose mention past line 5, INDEX folded → exit 0)
   was already in the QA set and was green under the implementation.
   → **DELTA (re-stamp):** the two drifted retros are re-stamped from `fresh · 2026-09-02` to
     `folded` in this same PR commit, making the real tree clean after the marker contract is active.

4. **Mutation proof: deleting the marker-read block flips M1 from exit 1 to exit 0.**
   Verified: with the `m=$(marker_of …)` + `if [ -n "$m" ]` block removed from the sweep, M1
   (`marker=folded, row=pending → exit 1`) falls through and returns exit 0 — the test flips to
   `not ok`. This proves the marker-read clause, not any pre-existing check, carries the bite.
   Revert confirmed 25/25 green.

## Cost / evidence

- QA RED: `qa/c6-marker-index-drift` tip `cb0dd7d` — 25 tests; M1 and M4 were `not ok` (sweep returned
  exit 0 where exit 1 was expected); 23 other tests green pre-impl.
- Implementation: `marker_of()` function + marker check block in `sweep-build-state.sh` (~12 lines added).
  Re-stamp: 2 retro files, 1 line each.
- Tests post-impl: `bats tests/build-retro-sync.bats` 25/25 green including M1/M4/M5/M6.
  Full suite: `bats tests/*.bats` 110/110 green. `shellcheck 0.10.0` exit 0 on all scripts.
  Real-tree sweep: `sweep-build-state.sh BUILD-STATE.md retros retros/INDEX.md` → exit 0.
  `HOME=/nonexistent` does not affect results (script takes explicit path args).
- Mutation: deleted marker block → M1 `not ok` (exit 0 observed where 1 expected); reverted.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| Sweep was marker-blind before PR1 | [CERT] | QA RED cb0dd7d: M1/M4 `not ok` with unmodified sweep |
| marker_of() reads first 5 lines, column-0 anchored, first-word result | [CERT] | sweep-build-state.sh marker_of() function; M6 green (prose-mention past line 5 tolerated) |
| Out-of-domain marker word → exit 1 | [CERT] | M4 green (`fresh` → exit 1); domain check in case statement |
| Marker ≠ INDEX row → exit 1 naming file + both values | [CERT] | M1 green; fail message in sweep |
| Absent marker → tolerated (exit 0) | [CERT] | M3 green; `if [ -n "$m" ]` guard |
| Mutation deleting block flips M1 to not ok | [CERT] | mutation proof documented above |
| Real tree exits 0 after re-stamp | [CERT] | M5 green; direct sweep run exit 0 |
| shellcheck 0.10.0 exit 0 | [CERT] | run on scripts/*.sh + toolbelt/*.sh + tests/*.bats + helpers/*.bash |

Connections: [[2026-09-04-campaign5-gate-activation]]; [[2026-09-04-kit-continuity-and-retro-gate-campaign]].
