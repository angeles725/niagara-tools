<!-- review-status: folded -->
# 2026-09-06 · kit · campaign9-demand-scope

**Session**: Campaign 9 PR2 — `toolbelt/lint-demand-scope.sh` (demand-in-scope lint, B820)
**Delta count**: 4

## What happened

B820 §820.2 identified a statically-decidable write-path gap: a control/staging method that reads
a process variable (suction/pressure/discharge/temp/cv/coil/head) in a comparison but has NO
demand-shaped input ({demand*, *call*, enable, loopEnable, *count, BStatusBoolean in*}) in its
parameter list or enclosing-class fields can run capacity to defend the PV with nobody calling —
the "pressure without demand" failure mode. A two-pass awk lint was authored, TDD RED→GREEN, and
routed into the pre-gate chain and report-module as a WARN-only member (--strict promotes to exit
1; never a hard FAIL per B820 §820.3, which establishes that a pure-modulator block driven by an
upstream demand gate would false-positive on a hard FAIL). [ev: corpus B820 §820.2-820.4]

Named mutation DS2 (remove demandCount param + gate from a CompressorControl.step fixture):
method WARNs with `demand-in-scope` and method name `step` in the row. DS-smoke confirmed the
REAL CompressorControl.step(long now, int demandCount, ...) produces zero WARN.*step rows (demand
gates on `demandCount` at FASE-2 tail and FASE-1 fallback). [ev: qa/c9-demand-in-scope d0f5942]

Real-tree smoke on four client module roots at client tip a109249 measured exact WARN counts.
[ev: corpus B820 §820.4] [ev: kit toolbelt/lint-delays.sh]

## Evidence
- DS1-DS7 + DS-smoke: 8/8 bats GREEN after implementation, confirmed via `bats tests/demand-in-scope.bats` [ev: qa/c9-demand-in-scope d0f5942]
- DS2 mutation (demand removed): `WARN  demand-in-scope  ...  step reads suction with no demand-shaped input in scope` — flip confirmed GREEN→RED without guard, RED→GREEN with guard [ev: corpus B820 §820.4]
- shellcheck -S warning exit 0 on `toolbelt/lint-demand-scope.sh` [ev: kit toolbelt/lint-demand-scope.sh]
- K19 routing: `BUILD-LOOP.md` §5 and `skill/SKILL.md` step 5, beside lint-wb-threading precedent [ev: corpus B820 §820.5]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| 1 | Add `lint-demand-scope.sh [--strict] <src>` to pre-gate chain (WARN-only; --strict -> exit 1) | `BUILD-LOOP.md` §5, `skill/SKILL.md` step 5 | [ev: corpus B820 §820.2] |
| 2 | Add `lint-demand-scope.sh` member row to `report-module.sh` (WARN rows aggregate as WARN) | `toolbelt/report-module.sh` §5b | [ev: corpus B820 §820.3] |
| 3 | Two-pass awk algorithm: Pass 1 collects class-level demand fields; Pass 2 applies rules 1-4 with brace-balanced method extraction | `toolbelt/lint-demand-scope.sh` | [ev: corpus B820 §820.2] |
| 4 | Scope = params UNION class fields (DS3): method-local demand fields do not clear the check but enclosing-class fields do | `toolbelt/lint-demand-scope.sh` rule 3 | [ev: corpus B820 §820.2] |

## Lessons
- A statically-decidable ABSENCE (no demand input in scope) warrants WARN; semantic judgment (is a present input really demand vs modulator?) stays advisory — the B820 §820.3 boundary is the right cut for any write-path lint. [ev: corpus B820 §820.3]
- Scope for the demand-in-scope check is params UNION enclosing-class fields: DS3 proved that a class-level `demandCount` field clears the check for methods with no demand params.
- The apostrophe (`'`) inside a single-quoted awk program passed to a shell variable closes the shell string prematurely; awk comments that use contractions must be rewritten.
- Dot-dir pruning (D9b) via `find -type d -name '.*' -prune` is the canonical pattern; DS7 confirmed Stale.java under `.deploy-baseline/` is never traversed.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign9-demand-scope.md | kit | 2026-09-06 | pending | 4 |`
