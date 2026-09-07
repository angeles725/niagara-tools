<!-- review-status: pending -->
# 2026-09-07 · DashboardPan · condensadoras-facade-crashsafe

**Session**: PANCCADIA — back the Condensadoras tab with a facade, crash-safety first.
**Delta count**: 2

## What happened
The SPA Condensadoras tab (HOA + reads) was a preview stub with no backing: `BDashboardService` had only `Cuarto1-5`, `DashboardReader` emitted no `Condensadoras/*`, and the servlet couldn't resolve the ord. Added a new facade type `BCompressorPanel` as a frozen child `Condensadoras`, wired to CompPan by Workbench LINKS (never a Java type import) so a missing/version-skewed CompPan only DANGLES a link instead of throwing the `Missing class` SEVERE seen live (`ColdRoomPan:HoaMode`).

## Evidence
- verify-module: 3 rt types RESOLVE + typecount == module-include.xml → anti-Missing-class gate passed [ev: verify-module.sh]
- `schema-risk` = SAFE (additive frozen child `Condensadoras`) [ev: schema-risk.sh]
- servlet unchanged (generic ord write); reader gained a rack emit block; naming reconciled compN(SPA) ↔ condenserN(CompPan) at the link [ev: DashboardReader.java / BDashboardService.java]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Doctrine: a dashboard facade decouples from control modules via a PLAIN `BComponent` child + Workbench links, NEVER a cross-module type import — so a version-skewed dependency dangles a link, not a `Missing class` crash. | types/dashboard.md §facade-decouple | [ev: BCompressorPanel] |
| 2 | Lint idea: flag a `-rt` facade/`BDashboardService`-adjacent type that `import`s a sibling control module's `B*` type (cross-module type coupling → Missing-class blast radius). | new lint-facade-decouple; report-module.sh | [ev: BCompressorPanel] |

## Lessons
- Adding a frozen child to a saved station is additive-safe (framework injects the default on load).
- The single hard crash mode is type non-resolution: register the type in module-include.xml + generated module.xml AND bump vendorVersion so the registry adopts the jar.
- Reconcile a naming mismatch at the LINK (name the facade slots after the SPA), not by importing the control type.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-condensadoras-facade-crashsafe.md | DashboardPan | 2026-09-07 | pending | 2 |`
