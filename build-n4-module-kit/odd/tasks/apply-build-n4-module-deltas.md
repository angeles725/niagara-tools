# ODD feature — Apply all pending build-n4-module kit deltas

**Created**: 2026-09-20 · **Owner**: ODD orchestrator · **Repo**: angeles725/niagara-tools (main)

## Objective
Fold ALL 114 pending propose-never-apply deltas (10 retros) from the 2026-09-19/20 module-corpus
research campaign into the build-n4-module kit CORE, as a chained-PR campaign.

## Problem / why
The deltas are proposals sitting in `build-n4-module-kit/retros/` with `review-status: pending`.
The operator (human reviewer) authorized applying all of them. Until folded, the kit guidance
(`types/*.md`, lints) does not carry these lessons, so a builder cannot benefit from them.

## Authorization & delivery
- Operator authorized: ODD, apply ALL deltas, **in a chain**, with commit + push + PR + PR-view + issues + merge.
- Delivery strategy: **feature-branch-chain**, one PR per retro (FULL promotion → clean INDEX flip),
  merge each to main before branching the next. ~400-line budget is expected to be exceeded per retro
  for the large ones; the chain IS the slicing.

## Fold contract (BUILD-LOOP §7 promotion)
Per retro folded:
1. Insert each Δ's guidance into its target kit file, carrying `[ev: retro <retro-stem> Δn]` citation
   (target `docs/how-to-*` paths that don't exist map to the real `types/*.md`).
2. Flip the retro's `retros/INDEX.md` row `pending → folded` (FULL-promotion structural anchor).
3. Update the kit `BUILD-STATE.md` self-envelope (envelope-pairing rule, same push range).
4. Commit trailer: `Retro: promotion (folds <PD-ids> from <retro>)`.
5. Gate: `sweep-build-state.sh BUILD-STATE.md retros retros/INDEX.md` + `sweep-fold-audit.sh --strict retros/INDEX.md .` + relevant `lint-*.sh`/CI (shellcheck+bats). Pre-push hook enforces.
6. Push → PR → CI green → merge.

## Constraints
- No AI attribution in commits (conventional commits). Promotion trailer required.
- propose-never-apply is being lifted BY the operator's explicit authorization — this campaign is that fold.
- Honor per-delta notes in APPLY-PLAN (e.g. superseded apillm Δ19→Δ20: fold with forward-reference, do not implement).
- Pre-existing gate noise NOT ours: retro `2026-09-19-cxf-to-n4-translator-proposal.md` missing INDEX row.

## Task checklist (one PR per retro, ascending size)
- [x] T1 · wb-field-editors-deltas (3) → types/wb-widgets.md — FOLDED (PD-FE1/2/3), INDEX+marker+BUILD-STATE flipped, sweeps green. Tracking issue #123.
- [x] T2 · wb-manager-framework-deltas (4) → types/wb-widgets.md — FOLDED (PD-WMF1-4: Manager recipe §), sweeps green.
- [x] T3 · our-dashboard-audit-deltas (5) → types/security.md §7 + types/dashboard.md + commissioning-verify.sh MANUAL step — FOLDED (PD-ODA1-5), shellcheck+bats+sweeps green.
- [x] T4 · module-hardening-reqexec-closed-deltas (7) → security.md §7/§8, distribution.md §5/§10, driver-authoring.md §5, logic-authoring.md (dynamic-slot §), issues-and-gotchas.md D4/G1 — FOLDED (PD-MH-UXS1/UXS4/BLD7/RUN5/PER1/PER4/PER7), sweeps green.
- [x] T5 · module-hardening-reference-cards-deltas (8) → types/logic.md + logic-authoring.md — FOLDED (REF1-7 + RUN1), sweeps green.
- [x] T6 · honeywell-wb-rt-wb-deltas (8) → wb-widgets/driver-authoring/security/logic-authoring/distribution/observability — FOLDED (8Δ), sweeps green.
- [x] T7 · wb-vendor-ux-wave3-vendor-drivers-deltas (14) → driver-authoring §9, wb-widgets, security §9, issues H1 — FOLDED (14Δ), sweeps green.
- [ ] T8 · module-hardening-failure-modes-deltas (16) → types/logic-authoring.md, distribution.md, issues-and-gotchas.md, lint candidates
- [ ] T9 · apillm-headless-servlet-rt-4.14-deltas (24) → types/security.md, structure.md, lint-servlet
- [ ] T10 · wb-vendor-ux-rt-wb-pattern-deltas (25) → types/security.md, driver-authoring.md, wb-widgets.md

## Acceptance criteria
- Every retro row in INDEX.md is `folded`; every Δ cited in a kit file (`sweep-fold-audit --strict` clean).
- Both sweeps + CI green on each PR; each PR merged to main.
- BUILD-STATE.md kit envelope updated.

## Progress log
- 2026-09-20: campaign authorized + set up; fold contract confirmed (pre-push hook installed). Starting T1.
- 2026-09-20: T0 (pre-req) — found main CI ALREADY RED (commits 4cd274b, 887e6bb): two non-retro docs in retros/ (cxf-to-n4 proposal, APPLY-PLAN worklist) had no INDEX row → sweep-build-state exit 1 → M5/H3/H8/H10 bats + ledger-sweep step failed. Fixed by adding both INDEX rows (pending, 0 deltas). Full suite now 583/583, both sweeps exit 0. Bundled into T1 PR #124 to green main.
- 2026-09-20: T1 FOLDED (wb-field-editors, 3Δ) — PR #124.
