<!-- review-status: pending -->
# Retro — Campaign 6 PR4: types fold — LC1–LC6, B762/B763 seams, mode-B slotomatic gap, DashboardPan ledger · 2026-09-05

PR4 of 8 in `build-n4-module-campaign6`. Doc-only fold: zero new bats tests. Branch: `feat/c6-types`.
PROPOSED kit deltas (propose-never-apply). Files changed: types/logic.md, types/dashboard.md,
types/wb-widgets.md, build-verify.md, SOURCES.md, corpus-index.md, retros/INDEX.md (freeze-stat),
BUILD-STATE.md (kit self-envelope + DashboardPan ledger corrections).

## What this fold proved / lessons from the fold itself

1. **Grep-before-fold consistently returns 0 for genuinely new deltas.** Every new LC1–LC6, DUX1–DWS2,
   DWB1, V1–V4, S1–S2 delta had 0 hits across the non-retro kit files. The existing `write.surface`
   hit in dashboard.md (U5 bullet) is a DIFFERENT rule (wider-than-read-surface generic warning, not
   the 5-gate DWS1 checklist) — grep-before-fold correctly identified it as non-duplicate.

2. **The freeze-stat retro (Δ1, Δ4) was already folded from a prior campaign.** build-verify.md had
   "Free the lock FIRST" (Δ1) and the junit cache glob note (Δ4); only Δ2 (annotation-only, L1) was
   genuinely un-folded. Δ3 (BDouble import) was already at METHODOLOGY.md:9,11. Folding one retro with
   4 numbered items: the grep-before-fold correctly showed 3 of 4 already present.

3. **DashboardPan ledger corrections required evidence from an external run.** The `verify_gate: unknown`
   → `pass` and the `profiles: rt,ux,wb` → `rt,ux` (DashboardPan-wb is a scaffold: 0 .java files,
   never built) corrections were driven by `verify-module.sh --src . --target-version 4.14` on module
   repo HEAD 4f5f1c7 (rt 7/7, ux 7/7). The wb scaffold finding closes the open_issues count at 3 (was 2).

4. **The U5 open_issue reword required B763 code evidence.** The prior wording ("write endpoint lacks
   OPERATOR-flag check") was wrong: `handleSetpointWrite` IS gated fail-closed by
   `DashboardRbacHelper.checkCanWrite` (BDashboardServlet.java:198). Corrected wording now states the
   genuine residue: missing pure-RBAC test seam (DWS2), no per-Ord lock/423, optional allowlist.

5. **The kit self-envelope update is part of the close gate even for a doc-only PR.** The branch push
   passes (pre-push hook checks within-branch range), but the main push is BLOCKED until BUILD-STATE.md
   carries the new retro_pending/last_commit/last_session anchor — same lesson as PR3.
