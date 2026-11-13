<!-- review-status: pending -->
# 2026-10-05 · module · panccadia-hmi-antifreeze-and-ford-dashboard

**Session**: One session, three pieces of work:
- DashboardPan 2.10.0 for PANCCADIA: HMI anti-freeze and HMI status. Urgent; installed live the same day.
- Commissioning of the 2026-10-05 install, guided live over oBIX.
- The new DashboardFord module, tasks T0-T5.

**Delta count**: 6

## What happened
The user needed DashboardPan Group A (anti-freeze) and Group B (HMI status) within about 20 minutes. One opus writer built
both. Partway through, the user told us to skip report-module, the lints and the packaging. The jars still passed
verify-module, the native review approved them, and the user installed them on the JACE. I then guided the commissioning
live over oBIX, read-only.

That surfaced four kit-relevant defects:
- A skipped lint let mojibake ship.
- Slot-o-Matic refused a new BFrozenEnum.
- Equal facade and control values had been mistaken for "linked".
- Instance flags stored on the station (`rso`) blocked a link that the module definition allows.

In parallel, DashboardFord T0-T5 surfaced a flaky verify-module types check. It also showed that the DashboardPan reader,
used as the exemplar, has no link detection.

## Evidence
- DashboardPan-rt `module.lexicon` lines 58-61 are raw UTF-8, so Workbench and oBIX show "HMI en lÃ­nea" and "CaÃ­do". lint-lexicon-ascii was skipped on the user's instruction. `[ev: panccadia-leon de36722]`
- Slot-o-Matic rejected the new `BHmiStatus.java` (a BFrozenEnum) with "Found multiple metadata blocks". The class was written by hand in the BFanMode form, and it loads live: the oBIX range `DashboardPan:HmiStatus` resolves. `[ev: panccadia-leon de36722]`
- `verify-module.sh` `check_types_have_classes` runs `printf | grep -qx` under `pipefail`, which raises SIGPIPE. The same jar gave ALL PASS three times and FAIL once, naming a different missing class each run. `[ev: Cliente/ford/dashboard 89c6166]`
- The 31 Condensadoras config slots had equal facade and control values but **no links at all**. The CompressorControl `Link*` slots showed only 5 facade sources; matching defaults hid the gap. `[ev: live oBIX 2026-10-05]`
- On Cuarto3, `evap1-3InDrip` show flags `rso` in the Slot Sheet, and Config Flags greys out Readonly. The module definition has been `SUMMARY` only since 2.9.0; up to 2.8.2 it was `SUMMARY|READONLY`, and it was never OPERATOR. The same slots linked fine on Cuarto2 and Cuarto4. `[ev: panccadia-leon 9e5afc3]`
- `DashboardReader.statusTag(null)` in DashboardPan returns "ok" and the reader never calls `getLinks`, so an unlinked BStatusNumeric shows as 0 {ok}. DashboardFord fixed this with `LinkProbe`, which requires an enabled and active inbound link. `[ev: Cliente/ford/dashboard a08cc80]`
- The user linked `proofFaultOffDelay` facade→control before copying the live value, and the control dropped from 10 min to 5 min. It was corrected live. `[ev: live oBIX 2026-10-05]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Keep the seconds-fast lint-lexicon-ascii in the non-skippable floor even when the user skips the lint suite, because mojibake ships without any visible failure | `BUILD-LOOP.md` § 5 | `[ev: panccadia-leon de36722]` |
| Δ2 | Fix `check_types_have_classes` so no pipe can exit early under pipefail (capture the list first or use a here-string), and add a bats test that runs one jar 20 times | `toolbelt/verify-module.sh` | `[ev: Cliente/ford/dashboard 89c6166]` |
| Δ3 | Make commissioning checks verify that the link exists (inbound links on the target), never that facade and control values match; add a check that lists unlinked facade→control config slots | `types/dashboard.md` § commissioning; `toolbelt/bog-audit.sh` | `[ev: live oBIX 2026-10-05]` |
| Δ4 | Document that instance-level flags such as `rso` survive a module that drops READONLY, and that Workbench Config Flags cannot clear them; add a `started()` normalizer or a commissioning check | `types/dashboard.md` § link-in slots | `[ev: panccadia-leon 9e5afc3]` |
| Δ5 | Require the reader to detect inbound links (`getLinks(slot)` enabled and active, otherwise `unlinked`), forbid `statusTag(null)→"ok"`, and cite DashboardFord LinkProbe as the exemplar | `types/dashboard.md` § reader contract | `[ev: Cliente/ford/dashboard a08cc80]` |
| Δ6 | Document that Slot-o-Matic 4.14 can refuse a new BFrozenEnum, the hand-written fallback, and the live proof that it loads | `types/logic.md` § enums | `[ev: panccadia-leon de36722]` |

## Lessons
- Equal values on both ends prove nothing; only the link object proves the wiring.
- "Copy the live value to the facade before linking" must be repeated at every facade→control step.
- Instance flags stored on the station outlive module flag changes.
- Under an urgent deadline, keep the seconds-fast gates and drop only the minutes-long report.
- Record the real station version: PANCCADIA runs Niagara 4.15.3.28, while builds target 4.14.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-05-panccadia-hmi-antifreeze-and-ford-dashboard.md | module | 2026-10-05 | pending | 6 |`
