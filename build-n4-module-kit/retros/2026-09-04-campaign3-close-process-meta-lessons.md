<!-- review-status: folded -->
<!-- kit-retro -->
# Campaign 3 close: the script-implementation campaign's own meta-lessons

Date: 2026-09-04 · Module: kit · SDD: build-n4-module-continuity Campaign 3 (PRs #23-#27, v0.10.0→0.13.1)

Campaign 3 implemented the 7 owed script/gate behaviors the Campaign 2 promotion had folded as rules but
left un-implemented (B4 rc/-backup, B6 build-target auto-detect, B7 gradle-root walk-up, soft-start
clean-lock message, B8 type-count, B10 lightweight-backup, and the empty-palette gate), each RED-first and
mutation-proven, then closed with a doc tidy pass. Doing it surfaced three process lessons that are
genuinely NEW (the promotion-process lessons from Campaign 2 already live in the kit). Captured here so a
future gate/script campaign inherits them.

## Meta-lessons (proven across C3-PR1..PR4 + close)

1. **A new gate/verify check must be HIGH-signal / low-false-positive — scope it to the reported defect, not
   to a literal completeness rule.** The palette check bites only on empty-palette-WITH-declared-types (the
   actual field defect: operator saw nothing to drag), NOT one-`<p>`-per-type — a literal completeness check
   would false-positive on types legitimately kept out of the palette (internal / slot-only) and become
   noise the operator learns to ignore. Same discipline sized B4 (only editor/backup files under `rc/`, not
   "any unexpected file"). Rule: a check earns its place only if it WARNs when it bites and stays silent
   otherwise; prove the guard with a mutation that removes it and shows a legitimate case would then fire.

2. **When a lesson removes a safety guard, prefer a frictionless WARN over a silent removal.** B10 dropped
   the `--no-backup` `--i-know-what-im-doing` gate (the retro's operator wanted trivial use), but the deploy
   still prints a one-line rollback reminder instead of skipping backup silently. This honors the
   trivial-use intent AND keeps the risk visible — a removed guard should leave a trace, not a blank. The
   safer default was chosen for the backup itself too (lightweight-on by default, not off), so the guard
   removal never became a net safety reduction.

3. **When a retro's prose and its PROPOSED-delta marker disagree, the marker is what promotes.** B10's retro
   recorded an OPERATOR SESSION DECISION in prose ("no backup by default — the jars are in GitHub") but its
   `PROPOSED delta` marker named a DIFFERENT rule (lightweight backup + keep-N autopurge). Under
   propose-never-apply, the marker is the promotable unit — read the kit rule from the marker, not from the
   operator's session narrative. Reading the source verbatim (not from memory or a mining note) is what
   surfaced the split and prevented promoting "no-backup-default" as the kit rule.

## Cost / evidence
- Signal/noise (1): C3-PR4 V17 — a typeless module with an empty palette must NOT warn; the type-count guard
  is exactly the false-positive control, mutation-proven (drop the guard → V17 warns → red).
- Safety-WARN (2): C3-PR3 Test 3 asserts `--no-backup` prints "backup skipped … rollback" and does NOT exit
  on a gate; the backup default is lightweight-ON (test35/36), not off.
- Marker reading (3): the B10 retro `2026-09-03-ng-deploy-backup-liviano-y-autopurga.md` lines 15-28 — the
  operator-decision prose vs the two-part PROPOSED toolbelt delta; the promoted BUILD-LOOP §6 rule is the
  marker's lightweight-default, and the stale "default to NO backup" doc line was corrected in the same PR.

## Nota de alcance
Process/tooling retro only — no module code, no corpus. The three rules should guide a future gate/script
campaign; consider folding them into METHODOLOGY's "Kit maintenance — retro promotion discipline" section
the next time it is edited (do NOT add speculatively now). A coordination note for multi-session work: when
a peer link drops, route through the hub session that still holds both links rather than blocking.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| Campaign 3 implemented all 7 owed script-impls, each mutation-proven | [CERT] | CHANGELOG v0.10.0-v0.13.0; QA mutation verdicts on PR #23-#26 |
| The palette check is scoped to empty-palette-with-types (not per-type) | [CERT] | verify-module.sh check_palette; tests/verify-module.bats V13/V17 |
| --no-backup WARNs instead of silently skipping; backup default is lightweight-ON | [CERT] | scripts/ng-deploy.sh guard_no_backup + backup(); ng-deploy.bats Test3/test35 |
| The B10 retro's prose and PROPOSED marker disagreed; the marker promoted | [CERT] | retro ng-deploy-backup-liviano-y-autopurga lines 15-28; BUILD-LOOP.md §6 |

Connections: [[2026-09-04-campaign2-promotion-process-meta-lessons]]; [[2026-09-04-gate-exit-taxonomy-promotion]].
