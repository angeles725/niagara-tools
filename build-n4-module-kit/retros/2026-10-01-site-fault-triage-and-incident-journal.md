<!-- review-status: pending -->
# 2026-10-01 · kit · site-fault-triage-and-incident-journal

**Session**: PANCCADIA gap review — faults that are not in our module (device offline, duplicate ID) and incident history kept ad hoc
**Delta count**: 3

## What happened
Console triage in the kit attributes only our own module's exceptions. On PANCCADIA the recurring console noise came
from the site (BACnet controller timeouts, two bus addresses answering with the same board ID that flooded the log after
every restart), and each time the attribution was re-derived by hand. Incident history lives in an ad hoc `bitacora/`
folder in one client repo and in project memory; the kit has no template or rule. Today's HMI freeze after restart #11
sits on the module/site boundary: the in-page watchdog existed and still did not recover the panel, so "module bug or
site condition" had to be decided from evidence each time.

## Evidence
- `triage-console.sh` attribution channels are own-frame, own-logger and bog-drift only; header lists C1-C3 and no site/device class `[ev: toolbelt/triage-console.sh:1-8]`
- BUILD-LOOP presents it as the post-reload own-module check `[ev: BUILD-LOOP.md:147]`
- Site-caused recurring errors found by manual log review: 483 setpoint writes plus TC500 BACnet timeouts, `time<=0` and a missing HoaMode class on 2026-09-06 `[ev: memory panccadia-station-audit-log-2026-09-06.md]`
- Two bus addresses report the same board ID bytes (`6caab91d`) after a clean restart, the flood resumes right after restart `[ev: memory panccadia-address6-same-board-as-address1.md]`
- Ad hoc incident journal in one client: `Cliente/Leon-Guanjuato/bitacora/` holds 13 files (e.g. `2026-09-02-commissioning-3-modulos.md`, `PROPOSAL-research-sdd-journal-mode.md`); no common template `[ev: ls Cliente/Leon-Guanjuato/bitacora 2026-10-01]`
- Kit has no incident-journal guidance: grep `bitacora|incident` over top-level docs, `types/`, `skill/` and `toolbelt/*.sh` returns nothing (only retros mention it) `[ev: kit grep 2026-10-01]`
- Freeze after restart #11 despite the watchdog recovering the previous 10: `[ev: retros/2026-10-01-dashboard-frontend-reliability-rules.md:8-12]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `triage-console.sh --site`: classify repeating non-own errors (device offline, duplicate device ID, comm timeout, history flood) into a separate "site issues" list with counts and first/last seen, so they stop being re-diagnosed each session. | `toolbelt/triage-console.sh` header | `[ev: memory panccadia-address6-same-board-as-address1.md]` |
| Δ2 | Rule "attribute before fixing": for any recurring console or field fault, record a verdict (own module / site device / platform / unknown) with its evidence before changing code; unknown stays unknown. | `BUILD-LOOP.md` § `6.a Post-deploy verification (after hot module reload or station restart)` | `[ev: retros/2026-10-01-dashboard-frontend-reliability-rules.md:8-12]` |
| Δ3 | Incident journal template: symptom, timeline, hypotheses, evidence, ruled out, next step, owner; one file per incident under the client repo's journal folder. | `ORCHESTRATION.md` (new §) "Incident journal" after § `8. Per-run retro/ticket loop` | `[ev: ls Cliente/Leon-Guanjuato/bitacora]` |

## Lessons
- Our module is not the only thing that writes to the console; classify noise once, not per session.
- A verdict without evidence is a guess; "ruled out" is as valuable as "found".
- An existing safeguard (the watchdog) does not prove the next failure is a different class.
