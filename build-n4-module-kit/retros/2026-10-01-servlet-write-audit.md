<!-- review-status: folded -->
# 2026-10-01 · kit · servlet-write-audit

**Session**: PANCCADIA gap review — operators change setpoints through the dashboard, but who/what/when is not visible to the customer
**Delta count**: 3

## What happened
The station audit review found 483 setpoint writes in three days from admin and API sources, and the access model shows
that writes arriving through the write-server share one machine-level oBIX user. The kit's security guide already
explains how to attribute a servlet write with a real `Context` (so the platform audit fires) and documents the oBIX PUT
attribution gap as a known limitation. What is missing is the application-level contract: every dashboard write records
user, ord, old value, new value and source in a store the customer can read, and the write-path lint checks matrix
coverage, not audit. The R14 second-login work persisted audit but did not show it.

## Evidence
- 483 setpoint writes Sep 3-6 (admin + API) found only by manual audit-history review `[ev: memory panccadia-station-audit-log-2026-09-06.md]`
- Browser never touches the station; one oBIX write user sits behind the write-server, so there is no who-changed-what at the station level `[ev: memory panccadia-access-model-viewer-writeserver.md]`
- R14 second login: "audit persisted not shown" `[ev: memory dashboardpan-r14-second-login.md]`
- Kit documents platform audit and the gap: `types/security.md` § `1 · Audit — who-changed-what` (§1.1 null-Context, §1.2 oBIX PUT attribution gap, lines 8-44) `[ev: types/security.md:8-44]`
- `NO_AUDIT` is described only as a flag for internal callback actions `types/actions.md:47-58` `[ev: types/actions.md:47-58]`
- `lint-write-path.sh` checks that each OPERATOR property has a write-path matrix row (header lines 1-6), not that a write handler calls an audit routine `[ev: toolbelt/lint-write-path.sh:1-6]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | New section "Servlet write audit": every write handler records user, ord, old value, new value, source (HMI / viewer / API) and timestamp, persisted in a bounded store and viewable; the write response echoes what was stored. | `types/security.md` (new §) after § `1 · Audit — who-changed-what` | `[ev: memory panccadia-station-audit-log-2026-09-06.md]` |
| Δ2 | Extend `lint-write-path.sh` (or a sibling) with a check that each servlet write handler calls the audit routine before returning success; WARN for legacy modules. | `toolbelt/lint-write-path.sh` header | `[ev: toolbelt/lint-write-path.sh:1-6]` |
| Δ3 | Dashboard convention: a "recent changes" view (last N writes, filter by room/user) visible to authorized users; the view reads the audit store, never the browser's local state. | `types/dashboard.md` § `Critical-write step-up auth` | `[ev: memory dashboardpan-r14-second-login.md]` |

## Lessons
- Platform audit and customer-visible audit are different deliverables.
- A shared service account erases attribution; carry the human identity in the payload.
- Persisting a record nobody can read does not answer "who changed this".
