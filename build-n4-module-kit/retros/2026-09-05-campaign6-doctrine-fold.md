<!-- review-status: pending -->
<!-- kit-retro -->
# Campaign 6 PR3: doctrine fold — K1–K10, multi-session + live-verify rules, what-to-test-where

Date: 2026-09-05 · Module: kit · SDD change: build-n4-module-campaign6 PR3 (feat/c6-doctrine)

Folded 7 pending meta-retros into METHODOLOGY.md §Kit-maintenance (K1–K9), §Multi-session coordination (M1),
§Live-verify safety (M2), §Build LC7 what-to-test-where table, and root CONTRIBUTING.md K10/A10.
Retros flipped: kit-continuity-and-retro-gate-campaign, run-pure-test-set-e-empty-cache,
gate-exit-taxonomy-promotion, campaign3-close-process-meta-lessons, campaign4-close-process-meta-lessons,
campaign5-gate-activation, ci-server-side-enforcement. All pending→folded, markers stamped.

## What the fold itself taught

1. **Grep-before-fold: zero pre-existing hits for K1–K9/M1/M2/K10 in non-retro kit files.** The T3.1–T3.4
   grep pass found each pattern only in retros/ (excluded) or BUILD-STATE.md (state envelope, not a rule
   file). None of the 13 new rules duplicated existing kit content. The "per.*type" grep did hit METHODOLOGY.md,
   but the match was the header phrase "Applies to all module types" — not an LC7 what-to-test-where table.
   The rule K6 ("grep every kit file before folding") paid off immediately by confirming the fold was clean.

2. **CONTRIBUTING.md stale "No CI" claim required simultaneous fix.** The K10 linter-pin rule added CI
   tooling as a live reality, so the old "No CI" bullet in §8 had to be replaced in the same change — a
   doc-vs-code defect identical to the BV1 class the kit's own promotion discipline targets.

3. **The 7 retro wordings mapped cleanly to K-numbered rules; no disambiguation required.** Every source
   retro's "Proposed kit deltas" section stated its rule precisely enough to fold verbatim (terse, one
   sentence per K). No retro had competing interpretations between its prose and its PROPOSED-delta marker.

## Cost / evidence
- METHODOLOGY.md: 51 lines → 74 lines (+23, within the +90 budget).
- CONTRIBUTING.md: 201 lines → 202 lines net (+2 replacing "No CI" paragraph).
- 7 retro markers: pending→folded (line-1 sed); 7 INDEX rows: pending→folded (python regex).
- sweep-build-state.sh exit 0; bats tests/*.bats 110/110; shellcheck exit 0; kit-links.bats 3/3.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| K1–K9 added to §Kit-maintenance with `[ev: retro <slug>]` | [CERT] | METHODOLOGY.md lines 53–61 |
| M1/M2 added as new sections with `[ev:]` citations | [CERT] | METHODOLOGY.md §Multi-session coordination, §Live-verify safety |
| LC7 table added to §Build per module type | [CERT] | METHODOLOGY.md §Build, after 4-layer stack bullet |
| K10/A10 added to CONTRIBUTING.md §8, "No CI" fixed | [CERT] | CONTRIBUTING.md line ~200 |
| 7 retro markers stamped folded; 7 INDEX rows flipped | [CERT] | `grep "review-status: folded" retros/*.md` (7 new); INDEX grep |
| No new bats tests (doc-only PR, per RX-2) | [CERT] | `bats tests/*.bats` 110 = same count as PR2 |
| sweep exit 0; 2 pending rows (freeze-stat + PR1 retro) | [CERT] | sweep run above |
| grep-before-fold: zero pre-existing hits in non-retro kit files | [CERT] | T3.1–T3.4 grep results (all clear) |

Connections: [[2026-09-04-kit-continuity-and-retro-gate-campaign]], [[2026-09-04-campaign5-gate-activation]];
build-n4-module-campaign6 PR3 tasks T3.1–T3.6.
