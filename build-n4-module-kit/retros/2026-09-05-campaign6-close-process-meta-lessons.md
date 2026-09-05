<!-- review-status: folded -->
<!-- kit-retro -->
# Campaign 6 close: retro harvest, research lanes, and the kit's own meta-lessons

Date: 2026-09-05 · Module: kit · SDD: build-n4-module-campaign6 (PRs #35–#43, v0.15.1→0.17.0)

Campaign 6 ran as three lanes under one SDD chain: the kit chain (eight PRs, every behavior RED-first with a named
mutation, every doc fold grep-before-fold and fidelity-read), a research lane that documented Tridium's own modules as
authoring exemplars (`/research-sdd`, 23 blocks B762–B793 across four focuses), and a verification lane (QA). It closed
the eight pending retros the previous campaigns left, folded 27 exemplar-backed deltas, and shipped seven new checks
(marker-aware sweep, CI-executed pure tests, fold-citation audit, gate coverage %, exposed-set coverage + dup keys,
environment preflight, timer-ticket lint, retro-debt aging). Tests 104 → 152. The lessons below are the ones the
campaign itself taught; the research lane's own lessons are in `openspec/changes/build-n4-module-campaign6/close-research-lessons.md`.

## Meta-lessons (proven across PR1..PR8)

1. **Every close-gate exit owes the kit self-envelope in the SAME push range; a branch push passing is not proof.**
   PR3 folded seven retros, flipped their rows and stamped their markers, and its branch push passed — but the ff-only
   push of `main` was BLOCKED by `.githooks/pre-push:54`, which sees the whole PR on `main` and requires
   `BUILD-STATE.md` + a pending retro + its INDEX row together. Fixed forward with the envelope commit; PR4..PR8 carried
   the envelope in their feature commit. [ev: PR #37 c978490]

2. **Check every commit body for attribution before anything is published.** The PR5b writer's feature commit carried
   the harness's default `Co-Authored-By` + session trailers despite the repo rule; caught by `git log --format=%B`
   before push and amended locally (never after publishing). Rule: the orchestrator greps commit bodies AND the PR body
   for `co-authored|generated with|claude` before opening a PR. [ev: PR #40 86aa533]

3. **Workers write only in their worktree; live documents are MERGED, never overwritten — diff the numstat before
   opening the PR.** PR5b's writer committed openspec artifacts on the main checkout (undone with a soft reset, files
   kept); PR8's writer replaced `apply-progress.md` (−520 lines) with a PR8-only copy — restored from `main` and
   appended. A `git diff --numstat origin/main..branch` sorted by deletions surfaces both in one line. [ev: PR #40, PR #43 91690fb]

4. **The native attempt ledger is serial and literal.** One active attempt per change regardless of the work-unit
   label (PR2's acquire was blocked while PR1 was open); `--evidence-goal` must be one short line; the untracked
   inventory sha changes whenever files appear (read it from the error, do not cache it); and a store-only commit
   (27 openspec files, +6317 lines) blew the 800 changed-line budget sized for code, forcing a recorded maintainer
   reset. Rule: keep doc-store commits early and separate, or size `--max-changed-lines` to the PR's real content. [ev: PR #41 settle → reset 5a7e12bc]

5. **Two independent reads of a doc fold catch what one cannot.** The block author's fidelity read (26/27 clean) and
   QA's mechanical gate plus three verbatim spot-checks together surfaced a REAL pre-existing inconsistency: METHODOLOGY
   line 15 (B636: neutralize the wizard's `module-permissions.xml`) against the promoted B777 row ("permissions inline,
   not a separate file"). Resolved with evidence on real jars: one source file, one `<permissions>` element inlined by
   the gradle plugin, two child kinds. A decompiled-corpus claim describes the artifact; the source shape needs the
   build. [ev: PR #42 87710f3/11bc298; CompPan-rt jar module.xml:11-15; saml-rt jar:72-78; wbutil-wb/rc/module-permissions.xml]

6. **Cite a QA RED by branch and re-read the tip at apply time.** The design validator found two stale commit hashes
   after QA rebased its RED branches during the same session; hard-pinned hashes in a design go stale within hours. [ev: design validator report; qa/c6-marker-index-drift fe6b88d→cb0dd7d]

7. **A fold-citation audit only credits standalone `[ev: retro <token>]` tokens; abbreviated, hyphen-aligned matching
   is required by the corpus's own convention; code-only folds need a "folded as code" prose line.** The first real-tree
   run found 2 legacy folded retros with no citation; PR7 fixed them and promoted the CI step to `--strict`. [ev: PR #39 real-tree output; PR #42 T7.5]

8. **Name what a metric measures.** `slot-coverage.sh` parse mode counts TYPE-set coverage (a type with ≥1 lexicon key
   counts as declared) while B788 measured ~25 % per-SLOT coverage on the same module; both are honest only if the output
   says which. Labelled `(type-set)` in PR6. [ev: PR #40 QA metric-honesty note; PR #41 T6.11]

9. **A conformance lint earns a hard FAIL only when it is statically decidable; the rest is a human-review checklist.**
   OMV1's negative (an "action without OPERATOR flag" lint would trip every legitimate admin action) is as valuable as
   the three real findings; the kit now separates lintable rules from review rules. [ev: B787–B789; PR #42 METHODOLOGY]

10. **Concurrent research lanes in one multi-focus corpus need reserved block ranges, a worktree per lane, and
    hand-recomputed counters.** Two lanes collided once on the shared index; ranges (B762–B771 / B772–B791), sibling
    worktrees pushing `HEAD:main` after rebase, and "last pusher recomputes the envelope" removed the collisions; the
    kit's `verify-state --sync-state` must not run on a shared-global focus. A public-repo push was correctly
    permission-gated by one lane and escalated to the owner rather than laundered through a peer. [ev: niagara-research 928c3a34e…0667896d6]

11. **Test budget discipline held: 48 new tests, each with a named mutation, zero tests on doc-only PRs, and QA's
    low-bite audit of the inherited 104 found none to re-harden.** Speed and bite are not in tension when every test is
    tied to a defect or a pinned contract. [ev: PR #35–#43; QA audit 2026-09-05]

## Proposed kit deltas

- **BUILD-LOOP §7:** state that every exit — (a) new retro, (b) trailer, (c) promotion — pairs its anchor with the kit
  `BUILD-STATE.md` self-envelope in the same push range, and that the hook evaluates the whole PR on `main`. [lesson 1]
- **METHODOLOGY §Kit-maintenance:** K11 — grep commit bodies and the PR body for attribution before publishing; K12 —
  workers write only in their worktree, live docs are merged never overwritten, `git diff --numstat` sorted by deletions
  before opening a PR; K13 — cite QA RED branches by name, re-read the tip at apply time; K14 — a metric's name states
  what it measures. [lessons 2, 3, 6, 8]
- **CONTRIBUTING §SDD ledger (new short section):** one active attempt per change; short single-line evidence goal;
  inventory sha from the error; doc-store commits early or budget sized to content. [lesson 4]
- **types/*.md fold discipline (already applied in PR7, promote as a rule):** two independent reads for every research
  fold — author fidelity + a mechanical gate with verbatim spot-checks of every negative/correction. [lesson 5]

## Handover (what the next campaign inherits)

- Pending retros: the nine Campaign-6 retros (PR1…PR8 + this one); the original eight are folded. `sweep-build-state.sh
  --age` escalates them after 30 days.
- Follow-up issues on niagara-tools: scaffold-module.sh (fixture = the built MinimalPan, B793), schema-risk.sh (MM3),
  `verify-module --plano`, campaign-retro fold, client punch-list, station-required research gaps.
- Research: focuses module-authoring-exemplars, own-modules-vs-exemplars, module-ux-testing-and-write-surface,
  module-web-tier-exemplars are STOPPED; requires-execution gaps MAE1-G1, MAE7-G1, B793-G1 remain.

## Self-verify

| Claim | Evidence |
|---|---|
| 8 PRs merged ff-only, linear | `git log --oneline 48f3736..91690fb` on origin/main; PRs #35–#43 MERGED |
| tests 104 → 152 | `bats tests/*.bats` on 48f3736 vs 91690fb |
| 8 pending → 0 (originals) | `retros/INDEX.md` pending rows = Campaign-6 retros only |
| 23 research blocks | niagara-research CATALOG B762–B763, B772–B785, B787–B793 |
| fold-audit strict 38/38 | `sweep-fold-audit.sh --strict` exit 0 on 91690fb |
| lessons 1–4 are real incidents | commits/PRs cited inline |
