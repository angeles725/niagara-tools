<!-- review-status: pending -->
# 2026-09-06 · kit · campaign9-ext-writable-shape

**Session**: Campaign 9 PR10 — lint-ext-writable-shape.sh (S19, B823/B826 slot-type doctrine)
**Delta count**: 3

## What happened

The live probe (B823, 2026-09-06) confirmed that an oBIX PUT against a `BStatusNumeric` OPERATOR
slot either rejects ("Cannot translate") or silently writes 0.0 when the wrapped-`<obj>` body omits
the `value` child. The kit needed a static lint to surface every `BStatusNumeric`, `BStatusBoolean`,
and `BStatusEnum` OPERATOR property whose declaring class exposes NO `@NiagaraAction`, so the
write-server author knows to use the oBIX child-leaf bare `<real>` path (B826-G2) instead of
attempting a direct parent-slot PUT. The lint was implemented as `toolbelt/lint-ext-writable-shape.sh`
following the EW1-EW11 RED contract pinned at `269be48` with the four-root EXACT counts (1/0/0/0).
`[ev: corpus B823 §823.2]`

## Evidence
- Real regression: `BRoomPanel.setpoint` (`BStatusNumeric`, `SUMMARY|OPERATOR`, no action) at
  `a109249` (DashboardPan-rt) — WARN at line 124 confirmed by smoke. `[ev: RED 269be48 EW10]`
- CompPan-rt 0 WARNs: `BCompressorControl.faultReset` (`BStatusBoolean`, `SUMMARY|OPERATOR`)
  exempt ONLY because the CLASS declares actions (`tick`/`powerOnExpired`, hidden, unrelated) — the
  class-level exemption is a COARSE proxy with parity to `tools/module-find.py ext-writable`; `faultReset`
  itself has NO writing action, so under B823 it is a genuine hazard the C9 rule does not flag (known
  false negative; per-slot writing-action rule = C10 seed S22, kit #89 cluster). `[ev: corpus B823]`
- EW3 positive pin: a `@NiagaraAction(name="setSetpoint", ...)` makes the slot clean. `[ev: RED 269be48 EW3]`
- EW11: empty dir / README-only dir → exit 3 + ERROR row (K20 no-silent-0 rule). `[ev: RED 269be48 EW11]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Add `lint-ext-writable-shape.sh` to §5 pre-gate | `BUILD-LOOP.md` §5 | `[ev: retro campaign9-ext-writable-shape]` |
| 2 | Add `lint-ext-writable-shape.sh` to toolbelt inventory | `skill/SKILL.md` step 5 + toolbelt list | `[ev: retro campaign9-ext-writable-shape]` |
| 3 | Add §5.6 member to `report-module.sh` | `toolbelt/report-module.sh` | `[ev: retro campaign9-ext-writable-shape]` |

## Lessons
- A `BStatusNumeric` OPERATOR slot with no `@NiagaraAction` is a silent-zero hazard for oBIX writers; the child-leaf bare `<real>` PUT is the safe no-code write path (B826-G2). `[ev: corpus B823 §823.2]`
- The class-level action check ("does the file have ANY @NiagaraAction?") matches `tools/module-find.py ext-writable`; it gives the correct four-root counts 1/0/0/0. `[ev: RED 269be48 EW10]`
- Multi-line `@NiagaraProperty` annotations (with nested `@Facet(...)` parens) require paren-balanced joining — single-line scanning misses the flags field. `[ev: corpus B823]`
- EW3 (slot has matching action → clean) uses class-level presence of any `@NiagaraAction` as the exemption signal; the mutation "drop @NiagaraAction exemption" flips EW3 from clean to WARN. `[ev: corpus B823]`

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign9-ext-writable-shape.md | kit | 2026-09-06 | pending | 3 |`
