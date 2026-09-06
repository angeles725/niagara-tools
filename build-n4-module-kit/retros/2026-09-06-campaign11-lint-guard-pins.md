<!-- review-status: pending -->
# 2026-09-06 · kit · campaign11-lint-guard-pins

**Session**: C11 PR4 — lint-guard-pins.sh meta-check + # Mutation: header retrofit of 10 lints (T4)
**Delta count**: 3

## What happened
The kit had zero `# Mutation:` header lines in any `toolbelt/lint-*.sh` before C11. A meta-check
that only verifies declared mutations reports a vacuous clean pass over zero declarations —
the exact class of silent unpinned guard that C11 kept catching by hand (S23-and missed by design,
EW-s22-nondo missed by investigador1 first read, WP-stale decoys only caught after the concept-drift
RED). C11 T4 adds `lint-guard-pins.sh` (scope = `lint-*.sh` only, D4b) and retrofits 10 lint headers
with 15 `# Mutation:` lines so the real-kit smoke asserts MATCH count ≥ 10, not "0 found". `[ev: design build-n4-module-campaign11 §D4a; retro campaign11-lint-guard-pins ebc15e8]`

## Evidence
- `grep -r '# Mutation:' build-n4-module-kit/toolbelt/` on `dab0807` → 0 occurrences: confirms the vacuous baseline `[ev: design §D4a CERT-read]`
- `bats tests/guard-pins.bats` RED on `dab0807` (command-not-found for lint-guard-pins.sh); GREEN after T4: 7/7 including T4-smoke (15 MATCH, 10 distinct scripts, 0 WARN) `[ev: guard-pins.bats ebc15e8]`
- OBSERVED mutation 1: rename WBT1c @test id → 1 WARN `mutation WBT1c -> no fixture`; restore → MATCH `[ev: apply PR4]`
- OBSERVED mutation 2: replace LD1 id with NONEXISTENT-ID-999 → 1 WARN `mutation NONEXISTENT-ID-999 -> no fixture` `[ev: apply PR4]`
- OBSERVED mutation 3: widen find scope from `lint-*.sh` to `*.sh` → report-module.sh flagged with WARN `[ev: apply PR4]`
- `grep -H -n` fix: single-file grep omits filename prefix without `-H`; adding `-H` forces `file:line:match` format regardless of file count `[ev: apply PR4 debug T4-match]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add `lint-guard-pins.sh` to pre-gate step 5 description | BUILD-LOOP.md §5 | `[ev: retro campaign11-lint-guard-pins]` |
| Δ2 | Add `# Mutation:` grammar as K24(7) doctrine | METHODOLOGY.md §K24 | `[ev: retro campaign11-lint-guard-pins]` |
| Δ3 | Add `lint-guard-pins.sh` routing reference | skill/SKILL.md Toolbelt | `[ev: retro campaign11-lint-guard-pins]` |

## Lessons
- A meta-check that only verifies declared mutations is vacuously clean over zero declarations; the MATCH count, not "0 found", is the anti-vacuity assertion.
- `grep -noE` without `-H` omits the filename prefix when only one file is passed; always use `-H` when the result is parsed as `file:line:text`.
- The fixture-id resolution scope must be the exact `@test "<id>:"` pattern — ids with spaces are unreferenceable by construction.
- Scope rules stated in the script header are checkable (lint-*.sh only); an allowlist of exempt scripts is a lie generator.
- The kit had zero `# Mutation:` lines before C11 T4; adding the meta-check and the retrofit in the same PR is the only shape that makes the smoke non-vacuous.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign11-lint-guard-pins.md | kit | 2026-09-06 | pending | 3 |`
