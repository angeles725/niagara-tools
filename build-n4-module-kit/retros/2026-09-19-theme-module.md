<!-- review-status: folded -->
# 2026-09-19 · kit · theme-module

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the theme-module cluster (vendor mining: themeDistech).
**Delta count**: 3

## What happened
The kit had no guidance on theme/branding modules — a distinct zero-Java ux-profile class found in
the net-new Distech `themeDistech-ux`. Cluster TH-01..TH-03 in the master register.

## Evidence
- zero-Java ux jar + `<defs><def name='themeName'/>` registration: `[ev: code themeDistech-ux module.xml]`
- imageOverrides mirrors module package path: `[ev: code themeDistech-ux imageOverrides/]`
- NSS `#define`/`lineargradient` + hx/theme.css font-URL copy/paste bug: `[ev: code themeDistech-ux nss/theme.nss]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | zero-Java ux theme module + themeName def registration + asset layout | `types/theme.md` §1-3 | `[ev: code themeDistech-ux module.xml]` |
| Δ2 | imageOverrides path-mirror rule (icons per type context) | `types/theme.md` §4 | `[ev: code themeDistech-ux imageOverrides/]` |
| Δ3 | NSS syntax + per-profile CSS + font-URL gotcha | `types/theme.md` §5-6 | `[ev: code themeDistech-ux nss/theme.nss]` |

## Lessons
- A theme module ships NO Java and NO palette; its only registration is the `themeName` def.
- imageOverrides selectively replaces icons by mirroring each module's Java package path.

---
**Status**: FOLDED into `types/theme.md` (2026-09-19).
