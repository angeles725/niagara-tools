<!-- review-status: folded -->
# 2026-09-06 · kit · campaign9-silent-protection

**Session**: Campaign 9 PR3 — toolbelt/lint-silent-protection.sh (B824 §824.2 two-part scan: trip detection + surface resolution)
**Delta count**: 3

## What happened

Campaign 9 S18 added a new toolbelt lint for silent protection trips. A protection trip is a boolean condition that gates an output (forces skip/shed) but never reaches an externally visible alarm/reason/status surface. The trigger was real-tree evidence in ColdRoomPan-rt and CompPan-rt where getSetpoint/getMode/getRunCmd conditionally forced an output with no associated *Alarm/*Reason SUMMARY slot or BAlarmSourceExt. The lint was implemented TDD (RED cherry-pick e38e503, then GREEN) against 9 bats tests (SP1-SP8 + SP-smoke). Two key implementation discoveries: (1) system awk does not support \b word boundaries — every boundary had to be spelled out as (^|[^A-Za-z0-9_]); (2) a naïve backward if-scan produced false positives when a plain return followed a one-liner if on the previous line — Pattern 6 was constrained to fire only when the return line itself contains a { (brace-delimited block) or also has an if ( on the same line.

## Evidence

- SP1-SP8 + SP-smoke all GREEN after implementation `[ev: bats tests/lint-silent-protection.bats]`
- shellcheck 0 issues `[ev: shellcheck toolbelt/lint-silent-protection.sh]`
- Real-tree smoke: ColdRoomPan-rt 44 WARNs (BColdRoom.java, BDefrostController.java, BEvaporatorUnit.java, ColdRoomControl.java); CompPan-rt 3 WARNs (BCompressorControl.java:1786,:1822,:2033); DashboardPan-rt 4 WARNs; DashboardPan-ux 62 WARNs; all exit=0 `[ev: smoke output 2026-09-06]`
- SP8 flip (named mutation): removing *Skip*/*Reason* from surface allowlist causes the private-field trip in SP8 fixture to FLAG — WARN appears; restoring → CLEAN `[ev: bats tests/lint-silent-protection.bats SP8]`
- CP-2 (dischargeHighAlarm) CLEAN: absent from CompPan-rt output — cross-file SURF_WRITE follow working `[ev: smoke output 2026-09-06]`
- defrostSkipped CLEAN: absent from ColdRoomPan-rt output — *Skip* surface allowlist working `[ev: smoke output 2026-09-06]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add lint-silent-protection.sh to §5 pre-gate sequence (beside lint-wb-threading.sh) | `BUILD-LOOP.md` § `5. Verify gate` | `[ev: retro campaign9-silent-protection]` |
| Δ2 | Add lint-silent-protection.sh to toolbelt list in skill/SKILL.md step 5 and References | `skill/SKILL.md` § `Execution Steps` | `[ev: retro campaign9-silent-protection]` |
| Δ3 | Add member row §5.5 in report-module.sh to surface lint-silent-protection WARNs in the aggregated punch-list | `toolbelt/report-module.sh` § `5.5` | `[ev: retro campaign9-silent-protection]` |

## Lessons

- System awk (mawk/gawk without --posix flag) may not support \b word boundaries; always test boundaries with (^|[^A-Za-z0-9_]) patterns when authoring portable awk.
- A backward if-scan that looks N lines above a trip statement will produce false positives when a plain `return` follows a one-liner `if` — constrain the backward scan to the same line or require an opening `{` on the return line.
- The effect-slot exemption (a trip writing its OWN forced output is not a surface) must be scoped per-method; a field written in one method does not exempt that field as a surface in a different method.
- Cross-file SURF_WRITE follow (pass1.awk collecting adapter field→slot writes) eliminates false positives for the alarm-wired pattern (SP2/CP-2) — worth the extra awk pass.
- Real-tree smoke rows at line numbers that differ from spec pins are normal when the source version changed; record the observed rows, not the predicted pins.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign9-silent-protection.md | kit | 2026-09-06 | pending | 3 |`
