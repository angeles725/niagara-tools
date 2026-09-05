<!-- review-status: folded -->
<!-- kit-retro -->
# Campaign 7 close: meta-lessons from the close process

Date: 2026-09-05 · Module: kit · SDD: build-n4-module-campaign7 (PRs #52–#59, v0.17.0→v0.18.0)

Campaign 7 ran as eight PRs under a stacked-to-main auto-chain: retro fold (PR1 #52), tool
integration (PR2 #53), dashboard exemplar (PR3 #54), scaffold-module.sh + fixtures (PR4 #55,
size:exception ~700-880 lines), schema-risk.sh (PR5 #56), verify-module --plano (PR6 #57),
logic.md split (PR7 #58), and report-module.sh (PR8 #59, stretch). Tests 152 → 179. Main at
0442a28. Launcher v0.4 → v0.7 (install-skill.sh re-run by lead after PR2, PR7, PR8).

The lessons below are what the campaign's process taught at the close stage.

## Meta-lessons

1. **QA's real-module smoke (B798) caught a schema-risk that synthetic fixtures cannot prove.**
   `report-module.sh` on ColdRoomPan-rt produced 9 PASS · 1 FAIL · 1 WARN · 1 SKIP with `--src`
   vs the B798 jar-mode baseline of 7 PASS · 1 FAIL · 1 WARN · 0 SKIP — the discrepancy showed
   that the design's expected count was from an older jar-mode run, not from the shipped `--src`
   mode. Synthetic fixtures in bats prove gate logic; a real module proves the integrated tool
   shape. Always pin the REAL output, not the design's prose estimate. [ev: retro campaign7-close]

2. **The L5 routing guard found two unrouted scripts (`scaffold-module.sh`, `schema-risk.sh`) that
   landed in PR4/PR5 without BUILD-LOOP or SKILL.md entries.** The guard turned RED in PR7 exactly
   as designed; PR7 closed the gap. A routing guard that can only fail after the gap exists is
   correct — it is the right time to fix it, not a pre-merge gate. [ev: retro campaign7-close]

3. **PL6 was dropped; L6 was kept — test bite is judged by mutation, not by intuition.**
   PL6 (`.jar` operand yields same verdict as HTML) was removed because the test added no new
   failure mode beyond PL1's PASS path (no named mutation). L6 (both `types/logic.md` and
   `types/logic-authoring.md` exist and cite each other) was kept because removing one file
   makes L6 RED. Mutation-provability is the criterion, not perceived importance. [ev: retro campaign7-close]

4. **Ledger discipline: settle `evidence-revision` = sha256 of HEAD; budgets sized to content.**
   The ledger token for this campaign is `sha256:8a2fba58b49bc74aab1fd00bc0e1df3bd63f113f0be2fb35838170cf3ce92044`.
   Budget was 800 lines (sized to the mix of doc-only and code PRs in the chain). The PR4
   `size:exception` (~700-880 lines) was declared upfront in tasks.md so the reviewer had
   context before opening the diff. [ev: retro campaign7-close]

5. **User stopped the chain after Campaign 7 by decision; Campaign 8 seed = only inter-module
   comms lacks a block.** The research backlog after Campaign 7: only inter-module communication
   patterns have no dedicated block. Candidate seeds: see
   `niagara-research/campaign8-research-candidates.md`. [ev: retro campaign7-close]

## Proposed kit deltas (applied in this close commit)

- **METHODOLOGY §Kit-maintenance (K19):** a toolbelt script is not done until named in BOTH
  `BUILD-LOOP.md` and `skill/SKILL.md`; `kit-links.bats` L4/L5 are the guards. [lesson 2]
- **METHODOLOGY §Kit-maintenance:** anchor a file split at a heading marking a clear audience
  change; verify with the sort-comm invariant; retarget `corpus-index.md` atomically. [logic-split]
- **build-verify.md §Pre-release:** smoke every new parser on ≥1 real module before merge. [lesson 1]
- **build-verify.md §Toolbelt authoring:** prefer row-level parsing over exit-code-only aggregation
  when composing toolbelt scripts. [report-module]
- **CONTRIBUTING §2:** scope the binary TEST fixture rule; add gawk `delete arr`, heredoc oracle
  diff, and shellcheck disable placement rules. [schema-risk, logic-split]

## Self-verify

| Claim | Evidence |
|---|---|
| 8 PRs #52–#59 merged ff-only | `git log --oneline origin/main` |
| tests 152 → 179 | `bats tests/*.bats` on 0442a28 |
| 0 pending retros at close | `grep -c '| pending |' retros/INDEX.md` == 0 |
| 9 campaign-7 retros folded | All INDEX rows campaign7-* show `folded` |
| fold-audit strict | `sweep-fold-audit.sh --strict` exit 0 |
| shellcheck 0 | Verified on all *.sh + *.bats |
| VERSION 0.18.0 | `cat VERSION` |
