# ODD feature — fold the 2026-09-18 corpus-mining deltas into build-n4-module kit core

**Created**: 2026-09-18 · **Branch**: `odd/fold-kit-deltas-2026-09-18` · **Repo**: niagara-tools/build-n4-module-kit
**Source**: 16 `pending` delta retros (130 deltas) from the niagara-research corpus mining (commit 87f14f6).
**Process**: BUILD-LOOP §7 promotion path (exit c) — per retro: edit kit core → flip `retros/INDEX.md` row `pending`→`folded` → update `BUILD-STATE.md` kit self-envelope → work-unit commit with trailer `Retro: promotion (folds <ids>)` → `sweep-build-state.sh` + `sweep-fold-audit.sh --strict`.
**Delivery**: push / PR / merge remain the user's decision (ODD close). Kill switches: skip a delta that duplicates already-folded content; keep the retro `pending` for owed halves (partial promotion updates BUILD-STATE, not the INDEX flip).

## Tasks (one work-unit commit per retro)

- [x] T1 — corpus-index-refresh (26Δ) → `corpus-index.md` rows (P0/P1/P2 + new §distribution) + README.md §Layout + METHODOLOGY.md P0 ref line — DONE (all 8 edits applied; INDEX folded)
- [x] T2 — sdk-examples (8Δ) → new `types/moduleTest.md` + `verify-module.sh` check_moduletest_present (bats MT1-MT4 green) + `types/dashboard.md` + `types/logic.md` + `types/structure.md` — DONE (FULL)
- [x] T3 — authoring-exemplars (8Δ) → `types/logic.md` (zero-demand/NaN) + `types/logic-authoring.md` (link-lifecycle ptr + security-module skeleton) + `BUILD-LOOP.md §4.b` + `build-verify.md` (OEM SF); Δ5/Δ8 already-folded in T2 — DONE (FULL)
- [x] T4 — module-mechanics (7Δ) → `types/logic-authoring.md` (TypeSubscriber, BEventService, Schedule, DDF, BPollScheduler) + `types/logic.md` (BConverter STRIP) — DONE (FULL)
- [x] T5 — schema-versioning (6Δ) → `METHODOLOGY.md` + `types/logic.md` (docs) + `schema-risk.sh` (L4 + fox-sync WARN) + `verify-module.sh` (check_cross_module_type, bats CROSSMOD1-3) — DONE (FULL; 27/27 + 11/11 green)
- [x] T6 — slots-flags-status-units-java8 (13Δ) → `types/logic.md` (BStatus/propagateFlags/unlinkable/TRANSIENT) + `METHODOLOGY.md` (facets/BIcon) + `verify-module.sh` (check_transient_operator, check_compact3_imports) + `build-verify.md` (Compact3/signing/BVersion) — DONE (FULL; 31/31 green)
- [x] T7 — ux-wb-writesurface-rbac (4Δ) → `wb-widgets.md` (device-model plugin, bajaux dialects, PX taxonomy) + `dashboard.md` (PX complement) — DONE (FULL, doc-only)
- [x] T8 — palette-lexicon-assets-images (5Δ) → `report-module.sh` (dup-keys BUGFIX) + `verify-module.sh` (check_palette root WARN, V18) + `types/structure.md` (palette template + -doc help) + `types/logic-authoring.md`/`METHODOLOGY.md` (icon recipe) — DONE (FULL; 32/32 + 8/8)
- [x] T9 — spa-library-integration (6Δ) → `types/dashboard.md` (JS embed recipe, injection, grunt-decision table, JS-vs-JVM distinction) — DONE (FULL, doc-only)
- [x] T10 — watcher-subscription-livedata (7Δ) → `types/logic-authoring.md` (SubscribeCallbacks/subscribed/TypeSubscriber-scope/BOX-chain) + `dashboard.md` (BOX rate-limit, live-data matrix) + `verify-module.sh` (check_subscription_leak SUBLEAK1-2) + `METHODOLOGY.md` — DONE (FULL; 34/34)
- [ ] T11 — factory-composition-new-types (8Δ) → `types/logic-authoring.md` + `types/logic.md` + `METHODOLOGY.md`
- [ ] T12 — ord-path-nav (9Δ) → `types/logic-authoring.md` + `types/structure.md`
- [ ] T13 — permissions-security-model (8Δ) → `types/dashboard.md` + `types/structure.md` + `build-verify.md`
- [ ] T14 — organization-tags-grouping (6Δ) → `types/logic.md` + `types/logic-authoring.md`
- [ ] T15 — resource-threading-consumption (6Δ) → `types/logic.md` + `verify-module.sh` + `METHODOLOGY.md`
- [ ] T16 — insights-issues-catalog (3Δ) → new `types/issues-and-gotchas.md` + `lint-timers.sh` + `METHODOLOGY.md`

## New retros discovered mid-campaign (operator-driven library/gap topics)
- [x] T19 — third-party-library-integration (3Δ) → new `types/third-party-libraries.md` + `build-verify.md` — DONE (FULL, doc-only)
- [ ] T17 — math-control-library-layer (4Δ) → new `types/math-control-libraries.md` + `logic.md` xref (+ check-compact3.sh recipe sketch)
- [x] T18 — wiresheet-control-library (1Δ) → `types/logic.md` §wire-sheet live-view recipe — DONE (FULL, doc-only)
- [ ] T20 — ml-libraries (pending gen) → ML for N4 (Weka/Smile/ONNX/XGBoost; Supervisor-only; propose-validate architecture; GPL caveat)

## Evidence (commit identities recorded as tasks close)
- (pending)
