<!-- review-status: folded -->
# 2026-09-07 · CompPan · two-differential-fase2

**Session**: PANCCADIA — express Fase-2 staging as two independent differentials (single slice, build+verify).
**Delta count**: 2

## What happened
The client kept Fase 1 but wanted rack pressure staging expressed as two INDEPENDENT differentials ("prende a sp+Δarriba / apaga a sp−Δabajo"), mirroring the per-room `differentialUp/Down` idiom. CompPan Fase 2 used ONE symmetric `suctionBand` (`hi=sp+band/2`, `lo=sp−band/2`). Replaced `suctionBand` with `suctionDiffUp` + `suctionDiffDown` in the pure logic, the `@NiagaraProperty` slots, the lexicon and the test.

## Evidence
- `CompressorControl.step`: enable `suctionValid && diffUp>0 && diffDown>0`; `hi = sp + suctionDiffUp`, `lo = sp − suctionDiffDown` [ev: CompPan-rt CompressorControl.java]
- `run-pure-test` 38/38 incl. new `asymmetricDiff_stagesUpAboveDiffUp_notBelow` [ev: run-pure-test CompPan-rt]
- build major 52 + signed; `javap` shows `suctionDiffUp/Down` present, `suctionBand` gone; verify ALL PASS; `2.0.3 → 2.1.0` [ev: build.sh CompPan]
- `schema-risk` = LOSSY (`remove_slot_simple suctionBand`) BUT bog-nav on live `Programacion/CompressorControl` = 0 persisted `suctionBand` → actual loss = 0 [ev: schema-risk.sh + bog-nav]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | A `schema-risk` LOSSY on `remove_slot_simple` needs a companion step: grep the DEPLOYED bog for the removed slot; a LOSSY whose slot is not persisted in the live bog is a non-issue, not a deploy blocker. Document/automate this cross-check. | build-verify.md schema-risk §; toolbelt/schema-risk.sh header | [ev: schema-risk.sh] |
| 2 | Record symmetric-band→two-edge (rename+split) as a recognised LOSSY-but-safe shape. | METHODOLOGY §schema | [ev: CompPan v2.1.0] |

## Lessons
- Replacing one symmetric parameter with two asymmetric edges = 1 REMOVE + 2 ADDs; `schema-risk` marks the remove LOSSY by design (pessimistic, no persistence knowledge).
- LOSSY is data-loss risk, never a crash (never OUTAGE); cross-check the live bog before treating it as blocking.
- Keeping `decideCall`/`step` pure let the asymmetric behaviour be proven in plain JUnit, no station.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-two-differential-fase2.md | CompPan | 2026-09-07 | pending | 2 |`
