<!-- review-status: pending -->
# 2026-09-28 · DashboardPan · panccadia-persistent-config-hoa

**Session**: PANCCADIA campaign "persistent config + HOA" — Cliente/panccadia-leon feat/panccadia-persistent-config-hoa (ColdRoomPan 2.4.0, CompPan 2.8.0, DashboardPan 2.9.0), 13 native reviews approved
**Delta count**: 6

## What happened
Field incidents on 2026-09-28: after a station restart compressor 3 (operator OFF) came back AUTO and tried to
start; and an operator OFF on compressor 2 stopped compressor 1 (AUTO) with 4 rooms calling. Root causes: every
HOA/mode slot on the dashboard facade and on BEvaporatorUnit was `Flags.TRANSIENT` (reverts to default 0 = AUTO
on restart; the facade->control link then re-pushes AUTO onto the persisted CompPan mode), no code ever requested
a station save (N4 auto-save is 24 h on controllers), and the installed CompPan predated the staging-count fix.
The campaign also found: link-in facade status slots declared READONLY, which Workbench refuses as link targets
(compressor hours on the dashboard showed 0 vs 81.7 h live); the HMI (Honeywell 10-inch, Chrome 83 WebView, no
scroll) could not open the second login from the Control tab because gated buttons were `disabled`; several CSS
constructs the panel does not support; and a HAND low-suction stop without hysteresis that would short-cycle.

## Evidence
- TRANSIENT HOA + link re-push: `BCompressorPanel.comp1..3Mode` TRANSIENT default 0 linked to CompPan `condenserNMode`; live AuditHistory 2026-09-28 09:16:41 `comp3Mode 0.00 -> 2.00` right after the 09:03 restart `[ev: 7009d91]`
- TRANSIENT never encoded: `ValueDocEncoder.java:326-333`; auto-save 24 h controllers / 1 h Windows `[ev: corpus B402]`
- READONLY link target refused: docSource `javax/baja/sys/LinkCheck.java:147-148` (`linkcheck.propReadonly`); fixed `[ev: 9e5afc3]` `[ev: 7085c25]`
- Disabled gated buttons never fire click on Chrome 83 (Control HOA): fixed and verified on real Chromium 83.0.4103.0 via CDP touch events `[ev: 6959756]` `[ev: 207aabb]`
- Panel-proven CSS rules (station login theme v1 failed / v2 worked): no `inset`, no CSS `min()/max()`, no `backdrop-filter`; flex `gap` and `aspect-ratio` also unsupported `[ev: Dashboard/docs/chrome83-verification-2026-09-28.md]`
- Linking facade->control pushes the facade DEFAULT onto the live control value (live minOff 3 min vs facade 0; proofFaultOffDelay 10 min vs 5 min) `[ev: Dashboard/docs/commissioning-live-values-2026-09-28.md]`
- HAND low-suction stop/restart without hysteresis short-cycles (coupled suction model: >3x starts) `[ev: edef064]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | HOA/mode (and every operator choice) slots must be persisted: never `Flags.TRANSIENT`, on BOTH the facade and the control end; add a lint row (`*Mode`/`*Hoa` OPERATOR slot with TRANSIENT → FAIL). A TRANSIENT facade source linked into a persisted control slot overwrites it on every restart. | `types/logic.md` § `Safety fail-modes & timers` + `types/dashboard.md` | `[ev: 7009d91]` |
| Δ2 | Operator writes must reach config.bog: request a debounced `Sys.getStation().save()` on any OPERATOR, non-TRANSIENT property change of a running component at steady state (pure `ConfigSavePolicy` + one-ticket `ConfigSaveScheduler`); never rely on platform auto-save (24 h on controllers). | `types/logic.md` § `Schema-safe evolution` | `[ev: corpus B402]` |
| Δ3 | Facade link-in status mirrors must NOT be READONLY (LinkCheck rejects a READONLY target); protect them from operator writes by omitting `Flags.OPERATOR` and a server-side `WritePolicy` (reject READONLY/TRANSIENT/non-OPERATOR). Add a lint/structural test: no facade property is READONLY. | `types/dashboard.md` + `METHODOLOGY.md` § `Conformance rules — lintable vs advisory` | `[ev: 7085c25]` |
| Δ4 | HMI (Chrome 83 WebView) rules for `rc-scan.sh`: FAIL on `inset:`, CSS `min(/max(/clamp(`, `backdrop-filter`, flex `gap`, `aspect-ratio`; FAIL on `.disabled = ` applied to login-gated controls (mark with a class instead; a disabled button dispatches no click, so a capture-phase login gate never runs); require a no-scroll check at the panel resolution in the verify step. | `toolbelt/rc-scan.sh` + `types/dashboard.md` | `[ev: 207aabb]` |
| Δ5 | Commissioning: before creating any facade->control link, set the facade value to the live control value (the link pushes the facade default immediately); ship a live-values sheet with every new facade mirror. | `types/dashboard.md` § facade linking | `[ev: Dashboard/docs/commissioning-live-values-2026-09-28.md]` |
| Δ6 | A protection that stops a unit and auto-restarts it (e.g. low suction for HAND) must have trip confirmation, a restart differential and a minimum restart delay; a continuously-invalid sensor may release the latch only after the full delay, never on a single sample; test with a coupled plant model that bounds starts per window. | `types/logic.md` § `Protection anatomy` | `[ev: edef064]` |

## Lessons
- TRANSIENT on an operator choice is a restart bug, not a safety default.
- A write that never reaches config.bog is lost on the first crash; request the save.
- READONLY on a link target silently disables the mirror; use flags + server policy instead.
- Verify HMI UI on the panel's real browser version; a modern headless browser hides the panel's gaps.
- Every auto-restart protection needs hysteresis and a delay, proven against a coupled plant model.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-28-panccadia-persistent-config-hoa.md | DashboardPan | 2026-09-28 | pending | 6 |`
