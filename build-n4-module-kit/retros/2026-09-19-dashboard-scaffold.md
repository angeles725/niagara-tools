<!-- review-status: folded -->
# 2026-09-19 · kit · dashboard-scaffold

**Session**: 2026-09-19 kit-improvement campaign (branch odd/issue-115-scaffold) — multi-profile dashboard scaffold (issue #115).
**Delta count**: 3

## What happened
scaffold-module.sh emitted only a single-profile -rt skeleton (fixtures/MinimalPan); there was no
-ux/dashboard fixture. Added a MinimalDash fixture (-rt facade + -ux BWebServlet/SPA/spec) and a
`--type dashboard` mode that round-trips clean.

## Evidence
- Both round-trips clean (MinimalDash + MinimalPan no-regression); scaffold-module.bats green; full suite 578 ok / 0 not-ok; shellcheck 0.10.0 exit 0. `[ev: tests/scaffold-module.bats]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | new fixtures/MinimalDash (25 files): -rt facade (BMinimalDash + pure logic + test) + -ux (BWebServlet + pure router + rc/index.html + Jasmine spec + dispatch test) | `fixtures/MinimalDash/` | `[ev: retro new-lints]` |
| Δ2 | scaffold-module.sh `--type <logic|dashboard>` (default logic unchanged); dashboard copies MinimalDash, -ux-before-rt substitution | `toolbelt/scaffold-module.sh` | `[ev: tests/scaffold-module.bats]` |
| Δ3 | CI scaffold-diff (dashboard) step + TC-DASH1/TC-DASH2 bats | `.github/workflows/ci.yml`, `tests/scaffold-module.bats` | `[ev: tests/scaffold-module.bats]` |

## Lessons
- A fixture-driven scaffold round-trips iff the emitted tree byte-equals the fixture post-substitution — the CI diff -r is the invariant; the -ux fixture's srcTest/rc/spec keeps lint-jasmine-ux clean.
- Substitute the more-specific path segment first (-ux/-rt before the bare module name).

---
**Status**: FOLDED — --type dashboard + MinimalDash fixture (2026-09-19), issue #115.
