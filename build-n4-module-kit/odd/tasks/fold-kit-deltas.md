# ODD feature — fold the 2026-09-18 corpus-mining deltas into build-n4-module kit core

**Created**: 2026-09-18 · **Branch**: `odd/fold-kit-deltas-2026-09-18` · **Repo**: niagara-tools/build-n4-module-kit
**Source**: 16 `pending` delta retros (130 deltas) from the niagara-research corpus mining (commit 87f14f6).
**Process**: BUILD-LOOP §7 promotion path (exit c) — per retro: edit kit core → flip `retros/INDEX.md` row `pending`→`folded` → update `BUILD-STATE.md` kit self-envelope → work-unit commit with trailer `Retro: promotion (folds <ids>)` → `sweep-build-state.sh` + `sweep-fold-audit.sh --strict`.
**Delivery**: push / PR / merge remain the user's decision (ODD close). Kill switches: skip a delta that duplicates already-folded content; keep the retro `pending` for owed halves (partial promotion updates BUILD-STATE, not the INDEX flip).

## Tasks (one work-unit commit per retro)

- [x] T1 — corpus-index-refresh (26Δ) → `corpus-index.md` rows (P0/P1/P2 + new §distribution) + README.md §Layout + METHODOLOGY.md P0 ref line — DONE (all 8 edits applied; INDEX folded)
- [x] T2 — sdk-examples (8Δ) → new `types/moduleTest.md` + `verify-module.sh` check_moduletest_present (bats MT1-MT4 green) + `types/dashboard.md` + `types/logic.md` + `types/structure.md` — DONE (FULL)
- [x] T3 — authoring-exemplars (8Δ) → `types/logic.md` (zero-demand/NaN) + `types/logic-authoring.md` (link-lifecycle ptr + security-module skeleton) + `BUILD-LOOP.md §4.b` + `build-verify.md` (OEM SF); Δ5/Δ8 already-folded in T2 — DONE (FULL)
- [ ] T4 — module-mechanics (7Δ) → `types/logic-authoring.md` + `types/logic.md`
- [ ] T5 — schema-versioning (6Δ) → `METHODOLOGY.md` + `types/logic.md` + `toolbelt/schema-risk.sh` + `verify-module.sh`
- [ ] T6 — slots-flags-status-units-java8 (13Δ) → `types/logic.md` + `METHODOLOGY.md` + `verify-module.sh` + `build-verify.md`
- [ ] T7 — ux-wb-writesurface-rbac (4Δ) → `wb-widgets.md` + `dashboard.md`
- [ ] T8 — palette-lexicon-assets-images (5Δ) → `verify-module.sh` + `types/structure.md` + `types/logic-authoring.md` + `report-module.sh` (dup-keys bug fix)
- [ ] T9 — spa-library-integration (6Δ) → `types/dashboard.md`
- [ ] T10 — watcher-subscription-livedata (7Δ) → `types/logic-authoring.md` + `dashboard.md` + `verify-module.sh` + `METHODOLOGY.md`
- [ ] T11 — factory-composition-new-types (8Δ) → `types/logic-authoring.md` + `types/logic.md` + `METHODOLOGY.md`
- [ ] T12 — ord-path-nav (9Δ) → `types/logic-authoring.md` + `types/structure.md`
- [ ] T13 — permissions-security-model (8Δ) → `types/dashboard.md` + `types/structure.md` + `build-verify.md`
- [ ] T14 — organization-tags-grouping (6Δ) → `types/logic.md` + `types/logic-authoring.md`
- [ ] T15 — resource-threading-consumption (6Δ) → `types/logic.md` + `verify-module.sh` + `METHODOLOGY.md`
- [ ] T16 — insights-issues-catalog (3Δ) → new `types/issues-and-gotchas.md` + `lint-timers.sh` + `METHODOLOGY.md`

## Evidence (commit identities recorded as tasks close)
- (pending)
