<!-- review-status: folded -->
# 2026-09-06 · kit · campaign9-doctrine-fold

**Session**: Campaign 9 PR12 — doctrine fold: protection anatomy patterns A/B, unified write audit, K22 cross-ref, C8/C9 process lessons
**Delta count**: 7

## What happened

Campaign 9 (wave 3) produced four concrete doctrine gaps that the wave's implementation PRs (PR8/PR9 alarm patterns, PR5/PR6 write audit, PR2/PR3/PR10 real-tree smoke discipline) closed in code but had not folded into kit docs. PR12 folds them: (1) Pattern A and Pattern B alarm authoring in a new `types/logic.md` §Protection anatomy H2; (2) the unified-write-audit one-canonical-sink doctrine in `types/dashboard.md`; (3) a K22 cross-reference in `BUILD-LOOP.md §5` (module-root/profile convention, exit-3 no-silent-0); (4) the `lint-ext-writable-shape.sh` pointer at the end of `types/logic-authoring.md` §Slot types; (5) OBSERVED-flip mutation table rule and fragment-merge protocol folded into `METHODOLOGY.md` §Kit maintenance (deferred S9/C8 lessons); (6) lead merge/settle order in `BUILD-LOOP.md §7`. `[ev: retro campaign8-close-process-meta-lessons]`

## Evidence
- Pattern A legality (ext PARENT must be BControlPoint; algorithm grandparent must be BBooleanPoint) confirmed from corpus B827 §827.3 decompile and RED qa/c9-alarm-cr3 `70a357b`. `[ev: corpus B827 §827.3]`
- Pattern B edge-machine contract (FIRE|CLEAR|NONE, re-seed in started()) confirmed from RED qa/c9-alarm-cp1 `8b43488` CPB2/CPB4 pins. `[ev: corpus B827 §827.4 §827.6]`
- Unified write audit (change_log, surface column, spool) confirmed from corpus B829/B830 probe notes. `[ev: corpus B829]` `[ev: corpus B830 §830.4]`
- K22 cross-ref target: METHODOLOGY.md:86 K22 exists; BUILD-LOOP §5 had no cross-reference line. `[ev: retro campaign8-close-process-meta-lessons §lesson 11]`
- OBSERVED flips rule: campaign8-close Lesson 7. Fragment-merge: Lesson 2 (four files: BUILD-LOOP §5, SKILL.md, INDEX.md, BUILD-STATE.md). `[ev: retro campaign8-close-process-meta-lessons]`
- C10 seeds: kit issue #89 cluster (lint-timers companion-flag false positive = S21; per-slot writing-action rule replacing class-level exemption = S22). `[ev: retro campaign9-ext-writable-shape]`

## C10 seeds filed in this retro
| Seed | Description | Source |
|---|---|---|
| S21 | lint-timers companion-flag false positive (kit issue #89): a companion boolean cleared in `started()` AND in the expiry handler — the lint fires because it sees the `started()` clear but the rule requires clear in `stopped()` OR `started()`, so `started()` alone should suffice. Fix: accept `started()` clear as satisfying the companion-flag contract. | `[ev: retro campaign9-ext-writable-shape]` — kit #89 cluster |
| S22 | Per-slot writing-action rule for lint-ext-writable-shape.sh: replace the class-level `@NiagaraAction` exemption with a per-slot check (a slot is clean only when THAT slot has a matching action, not merely when the class has any action). Eliminates the `faultReset` false negative. | `[ev: retro campaign9-ext-writable-shape]` |

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | New `## Protection anatomy` H2: Pattern A (child BControlPoint + BAlarmSourceExt + BBooleanChangeOfStateAlgorithm) and Pattern B (BIAlarmSource + AlarmSupport edge machine) | `types/logic.md` | `[ev: retro campaign9-doctrine-fold]` |
| 2 | `lint-ext-writable-shape.sh` pointer at end of slot-types section | `types/logic-authoring.md` | `[ev: retro campaign9-doctrine-fold]` |
| 3 | K22 cross-reference bullet in §5 pre-gate | `BUILD-LOOP.md` §5 | `[ev: retro campaign9-doctrine-fold]` |
| 4 | Unified write-audit one-canonical-sink doctrine | `types/dashboard.md` | `[ev: retro campaign9-doctrine-fold]` |
| 5 | OBSERVED-flip mutation table rule + fragment-merge protocol | `METHODOLOGY.md` §Kit maintenance | `[ev: retro campaign9-doctrine-fold]` |
| 6 | Lead merge/settle order (ff-only → verify log -1 → settle) | `BUILD-LOOP.md` §7 | `[ev: retro campaign9-doctrine-fold]` |
| 7 | C10 seeds S21/S22 filed | this retro | `[ev: retro campaign9-doctrine-fold]` |

## Lessons
- A protection anatomy section anchors the two alarm patterns (Pattern A declarative ext, Pattern B BIAlarmSource) in a single searchable location; without it, Pattern A/B legality is scattered across corpus blocks and RED branches only.
- The unified write-audit doctrine (one canonical sink, surface column, spool) must be in types/dashboard.md before S12 write-server PRs land, so the author has a kit reference; landing it after makes the doctrine a retro footnote.
- The class-level `@NiagaraAction` exemption in lint-ext-writable-shape.sh is a known coarse proxy (faultReset false negative) — C10 per-slot rule (S22) tightens it without breaking the four-root smoke counts.
- Companion-flag false positive (S21, kit #89): a `started()` clear should satisfy the companion-flag contract; the current lint may fire on a `started()`-only clear that is semantically correct — investigate before C10 lands.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign9-doctrine-fold.md | kit | 2026-09-06 | pending | 7 |`
