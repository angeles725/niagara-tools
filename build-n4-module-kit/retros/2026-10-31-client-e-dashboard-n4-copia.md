<!-- review-status: pending -->
<!-- module: ClientEDashboard -->
<!-- deltas: 18 -->
# 2026-10-31 · ClientEDashboard · client-e-dashboard-n4-copia

**Session**: ClientEDashboard feat/client-e-dashboard-n4-copia (v1.4.0 -> 1.4.1): N4 viewer from the reference, inventory merge, EX removal, wire-sheet relayout, equipment view restored
**Delta count**: 18

## What happened
A live test of ClientEDashboard 1.3.0 on a local 4.15.6.20 station surfaced five defects that no gate caught: (a) the 30-oct
build was signed by the mirror's throwaway key while the deployed 1.2.2 carried the user's dev key; (b) Nivel 4
wire-sheet glyphs overlapped because a grid change never reaches nodes already placed; (c) the inventory generator
invented `AHU_SinTag_*` names for untagged reference equipment; (d) a data decision (keep the CAD inventory) was
silently extended to "keep the old N4 viewer"; (e) the reference port of the level viewers dropped the whole equipment
view (classic-ui.js 646 -> 220 lines) because its data was simulated, and the user found it live ("No puede volver a
pasar esto"). The fixes shipped as 1.4.0/1.4.1; RDD could not review the inventory/EX slice even after splitting it.

## Evidence
- signer drift: deployed 1.2.2 SHA256 77:80:23..., build/libs 1.3.0 A5:60:4B... (kit-main build.sh mirror user.home isolation, build.sh:243-262) `[ev: retro r8-mirror-userhome]`
- wire sheet: InventoryPlan GRID rowSpacing 28 (d298b35) -> 60 (9cf813d); sync adds wsAnnotation only when absent `[ev: corpus B1172]`
- generator: LevelsInventory rows "AHU (sin tag)" -> AHU_SinTag_C in 4 levels; fixed by levelgen NameOverrides (6735dc2) `[ev: 6735dc2]`
- dropped view: reference classic-ui.js lines 187-380 vs port 220 lines; design table said "same" `[ev: 008de14]`
- schema-risk on the EX removal reported `ClientEDashboardService.frequency/staticPressure/...` for slots of the deleted BExhaustPanel `[ev: c3ee8f6]`
- RDD `lens_context_budget_exceeded` on a 176-line / 31-file commit with regenerated fixtures `[ev: 13cac94]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | verify-module / ng-deploy WARN when the new jar's signer fingerprint differs from the deployed jar's | `build-verify.md` § `Signing per deploy target` | `[ev: retro r8-mirror-userhome]` |
| Δ2 | A mirror build is never deployable as-is: add `toolbelt/resign-for-deploy.sh` (strip NIAGARA4.*, re-sign with the user's signing profile, run verify-module) or ng-deploy `--from-build-libs` | `build-verify.md` § `Station deploy: ng-deploy.sh (the established wrapper)` | `[ev: retro r8-mirror-userhome]` |
| Δ3 | Wire-sheet placement is write-once: document per-row heights (tallest glyph + gap) and a guarded relayout of nodes still on an automatic cell as the sanctioned exception to "never overwrite an annotation" | `types/logic-authoring.md` § `Wire-sheet placement of code-created components` | `[ev: corpus B1172]` |
| Δ4 | Inventory generators FAIL on untagged source equipment unless a name-override table covers it; pattern-inferred names are asked, never guessed | `types/dashboard.md` § `Facade contract & commissioning links` | `[ev: 6735dc2]` |
| Δ5 | Behavior-decisions gate: every UI consequence of a data decision is its own question | `BUILD-LOOP.md` § `1. Design` | `[ev: retro behavior-decisions-ask-dont-assume]` |
| Δ6 | A reference port runs port-parity (files, DOM ids, functions) before hand-off; a dropped feature needs a waiver with a user decision reference | `types/frontend-standard.md` § `2. Structure and organization` | `[ev: 008de14]` |
| Δ7 | schema-risk reports a deleted type as `remove_type <Type>` (B754: Type not found -> subtree skipped) instead of attributing its slots to another class | `build-verify.md` § `Schema-risk deploy workflow` | `[ev: c3ee8f6]` |
| Δ8 | A module without `.env.local` gets a doctor/preflight WARN before a deploy step is planned | `BUILD-LOOP.md` § `6. Deploy (station) — operator` | `[ev: 5f9853d]` |
| Δ9 | Structure tests pin service actions by name, not by count | `types/logic-authoring.md` § `Wire-sheet placement of code-created components` | `[ev: dd64527]` |
| Δ10 | Parallel writer units: every shared file (service, lexicon, module map) has one owner so each unit commits alone under ~400 lines | `ORCHESTRATION.md` § `3. Delegation triggers` | `[ev: a8b9a8f]` |
| Δ11 | Regenerated fixtures go in their own commit; the RDD lens budget is driven by whole touched files, not changed lines | `BUILD-LOOP.md` § `7. Retro + close (HARD close gate — not optional)` | `[ev: 13cac94]` |
| Δ12 | report-module L10 (absolute host path in tracked gradle.properties) changed from WARN to FAIL between kit checkouts; record the intent in the lint row | `BUILD-LOOP.md` § `5. Verify gate (before "done")` | `[ev: 5f9853d]` |
| Δ13 | A preview "Sin respuesta de la estación" under headless software rendering is re-run before it counts as a defect (API timeouts) | `BUILD-LOOP.md` § `3. Preview (UI types) — BEFORE compiling` | `[ev: 008de14]` |
| Δ14 | An RDD driver stops on `correction_plan_required` and hands back to the parent (it needs `--request-hash` + `--correction-lines`); replaying provider tokens there loops on `invalid_request` | `ORCHESTRATION.md` § `3. Delegation triggers` | `[ev: 008de14]` |
| Δ15 | Explicitly review an under_budget slice that adds operator write paths in the browser: it found a CRITICAL cross-unit write race (a sequence must bind its target path before the first write) | `types/dashboard.md` § `Critical-write step-up auth` | `[ev: 008de14]` |
| Δ16 | Two sessions of the same resumed conversation worked one module branch (a closed terminal window left the first alive in tmux); add a per-module session lock (`.build-n4.lock` with session name + pid, checked at orient-guard) and name a single owner | `BUILD-LOOP.md` § `0.a Orient from BUILD-STATE (before touching anything)` | `[ev: 0ea3508]` |
| Δ17 | port-parity's first real run gave 20 false FAILs for names defined only in already-waived files; a `file` waiver now cascades to ids/functions only that file defines (fixed in the gate, c9f7bad2) | `types/frontend-standard.md` § `2. Structure and organization` | `[ev: bc48630]` |
| Δ18 | A design that says "port of X" must ship a `.port-parity` declaration; today a module without the file is silently unchecked by lint-port-parity | `BUILD-LOOP.md` § `1. Design` | `[ev: bc48630]` |

## Lessons
- A reference port is complete only when every reference feature is present or waived by a recorded user decision.
- Placement written by code is write-once on a station; changing the algorithm needs a relayout path.
- The signer of a deployable jar is part of its identity: compare it with the deployed jar before copying.
- Untagged source data is a question for the user, never a generated name.
- A data decision and its UI consequence are two decisions.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-31-client-e-dashboard-n4-copia.md | ClientEDashboard | 2026-10-31 | pending | 18 |`
