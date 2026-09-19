<!-- review-status: folded -->
# 2026-09-19 · kit · gotchas-tier-a

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU adding field-proven Tier-A gotchas.
**Delta count**: 6

## What happened
Six recurring, field-proven traps (from client deploys + memories) had no entry in the gotchas
catalog. Added in the doc's Symptom/Root-cause/Fix format.

## Evidence
- Clock.schedule <=0 floor at 1ms: `[ev: mem coldroompan-defrost-time-le-0-bug]`
- rt lifecycle seam untestable; Missing class on deploy: `[ev: mem panccadia-station-audit-log]`
- moduleTest 7.6.17 + ux Jasmine: `[ev: corpus B1028]`; EC-Net set(value): `[ev: mem harbor-greenmax-b851-pilot]`; GET-destructive: `[ev: mem nmodsreflow]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | Clock.schedule zero-delay floor at 1ms (Math.max(0L,x) is not enough) | `types/issues-and-gotchas.md` B4 | `[ev: mem coldroompan-defrost-time-le-0-bug]` |
| Δ2 | rt lifecycle seam cancelRunTickets vs cancelTicket untestable | `types/issues-and-gotchas.md` C3 | `[ev: mem coldroompan-defrost-time-le-0-bug]` |
| Δ3 | Missing class at station start (version/name mismatch) | `types/issues-and-gotchas.md` A5 | `[ev: mem panccadia-station-audit-log]` |
| Δ4 | -ux modules missing Jasmine devDeps | `types/issues-and-gotchas.md` C4 | `[ev: corpus B1028]` |
| Δ5 | EC-Net 4.3/Hx browser set(value) fails → HOA BooleanWritable | `types/issues-and-gotchas.md` E1 | `[ev: mem harbor-greenmax-b851-pilot]` |
| Δ6 | destructive HTTP GET endpoints → require POST + CSRF | `types/issues-and-gotchas.md` F1 | `[ev: mem nmodsreflow]` |

## Lessons
- A self-firing timer with a computed delay must floor at 1 ms; 0 is rejected like a negative.
- Lifecycle ticket-cancel intent is invisible to lint/pure-JUnit — it needs review + a live check.

---
**Status**: FOLDED into `types/issues-and-gotchas.md` (2026-09-19).
