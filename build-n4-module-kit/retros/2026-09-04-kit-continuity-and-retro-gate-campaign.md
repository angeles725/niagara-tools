<!-- review-status: folded -->
<!-- kit-retro -->
# Kit self-improvement campaign: continuity ledger + retro-enforcement gate + lesson promotion

Date: 2026-09-04 · Module: kit (self) · SDD change: build-n4-module-continuity (PRs #12/#13/#14/PR3)

The kit had two structural gaps a mining sweep confirmed: no in-repo "where did we leave off" record
(continuity was outsourced to Engram + client `bitacora/`), and no enforcement that a retro gets written
(≈24 of 29 retros had untracked fold-status, forcing a manual re-mining sweep). This campaign closed both
and started promoting the trapped lessons.

## What was built (already applied — this retro records it, it is not a proposal)

1. **Continuity ledger (PR #12, v0.6.0):** `BUILD-STATE.md` — one `build-state.v1` envelope per module,
   seeded from evidence, with a GATED (`retro_required`/`retro_pending`, kit-local) vs DECLARED
   (jar/module/src state, separate repos) split. `BUILD-LOOP.md` §0.a orient + §7 hard close gate.
2. **Retro-enforcement gate (PR #13, v0.7.0):** `toolbelt/sweep-build-state.sh` (content-only, VCS-free
   per kit-links L2) + `.githooks/pre-push` (opt-in, owns the diff half) + `retros/INDEX.md` registry.
   A build-relevant kit change cannot land without a BUILD-STATE update + a pending retro + its INDEX
   row, or a `Retro: none (trivial: <reason>)` trailer.
3. **run-pure-test runner (PR #14, v0.8.0):** `toolbelt/run-pure-test.sh` (see its own retro).
4. **Lesson promotion (PR3, v0.8.1):** the TOP-5 + items 6-8 of the 42-lesson mining folded into
   METHODOLOGY/BUILD-LOOP/types/build-verify, the 2 contradictions resolved to ONE rule each, and this
   `kit` self-section added so kit-infra work has a row to update.

## Proposed kit deltas (a second promotion pass is owed)

- Fold the remaining mined lessons still `pending` in `retros/INDEX.md`: L3-L14/L16-L22 (control),
  U1-U4/U6-U9 (dashboard/WebView), B4/B6-B10 (build/deploy), D1-D3 (deploy), S2-S4 (schema/versioning).
  Each has a target file already named in the mining report.
- Consider a curated `corpus-index.md` (T7) wiring the B729-B760 authoring blocks the kit does not yet
  reference, and folding the WB ladder (B751) into `types/wb-widgets.md`.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| Continuity + enforcement were absent before this campaign | [CERT] | mining sweep §3(a)/(b); no STATE/INDEX file existed in `retros/` |
| The gate is enforced, not discipline-only | [CERT] | `tests/build-retro-sync.bats` 16 cases, mutation-checked to bite |
| The remaining lessons are documented but not folded | [CERT] | `retros/INDEX.md` review-status `pending` rows |

Connections: [[build-n4-module-continuity]] tasks.md; the 42-lesson mining report.
