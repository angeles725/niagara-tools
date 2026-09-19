<!-- review-status: folded -->
# 2026-09-19 · kit · distribution

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the OEM distribution cluster (audience now includes OEM builders).
**Delta count**: 9

## What happened
The kit's corpus-index pointed at the n4-distribution blocks but no types/ doc captured OEM
packaging. Cluster KD-1..KD-8 in the master register.

## Evidence
- module JAR vs .dist + dist.xml contract + Supervisor-as-installer: `[ev: corpus B1023]`
- overlay step-11 last-write-wins: `[ev: corpus B1026]`; OEM trust certs (DSA XML, not X.509): `[ev: corpus B1027]`
- BOG schema-safety matrix: `[ev: corpus B754]`; version floor: `[ev: corpus B755]`; .ntpl: `[ev: corpus B1021]`; AX→N4: `[ev: corpus B1024]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | module JAR vs distribution .dist | `types/distribution.md` §1 | `[ev: corpus B1023]` |
| Δ2 | dist.xml metadata contract | `types/distribution.md` §2 | `[ev: corpus B1023]` |
| Δ3 | OEM overlay anatomy + step-11 | `types/distribution.md` §3-4 | `[ev: corpus B1026]` |
| Δ4 | OEM vendor trust certs for production | `types/distribution.md` §5 | `[ev: corpus B1027]` |
| Δ5 | BOG schema-safety matrix | `types/distribution.md` §6 | `[ev: corpus B754]` |
| Δ6 | version floor (3-part) vs build-stamp | `types/distribution.md` §7 | `[ev: corpus B755]` |
| Δ7 | .ntpl station template | `types/distribution.md` §8 | `[ev: corpus B1021]` |
| Δ8 | Supervisor-as-installer | `types/distribution.md` §3 | `[ev: corpus B1023]` |
| Δ9 | AX→N4 migration readiness | `types/distribution.md` §9 | `[ev: corpus B1024]` |

## Lessons
- .dist carries platform/firmware, NOT module code; wrong @noStation/@reboot corrupts field devices.
- ADD-never-retype on a live schema; a RETYPE or removed enum tag = station won't boot (OUTAGE).

---
**Status**: FOLDED into `types/distribution.md` (2026-09-19).
