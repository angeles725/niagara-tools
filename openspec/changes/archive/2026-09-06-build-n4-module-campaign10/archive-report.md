# Archive Report: build-n4-module-campaign10

**Status**: Archived · **Date**: 2026-09-06 · **Kit version**: v0.20.0 → v0.21.0 · **Merge commit**: dab0807

---

## Change Summary

Campaign 10 completed the precision refinements for three C9 lints that were emitting false results on the operator's real modules (Leon-Guanjuato at ff1b659), plus two structural hygiene fixes and client-side caching improvements.

**Scope**: 7 PRs in one wave, stacked-to-main, auto-chain delivery strategy (Low 400-line budget risk).

| PR | Slice | Lines | Status |
|----|-------|-------|--------|
| PR1 | S21 lint-timers companion-flag | ~180 | PASS |
| PR2 | S22 lint-ext-writable-shape per-slot action-body | ~200 | PASS |
| PR3 | S23 lint-silent-protection Pattern B surface | ~220 | PASS |
| PR4 | S24 cwd-independent structural REDs | ~40 | PASS |
| PR5 | S25 lint-write-path STALE advisory + --strict | ~100 | PASS |
| PR6 | S26 client .gitignore + [concept] matrix marks | ~20 | PASS |
| PR7 | C10 close: VERSION 0.21.0, retros, CHANGELOG | ~180 | PASS |

**Total**: ~735 lines, 0 failed requirements, 418/418 scenario tests verified.

---

## Verification Verdict

**Result**: PASS with 2 documented warnings (non-blocking per sdd-verify decision)

**Snapshot** (per verify-report #8403 at 2026-09-06 14:21:31):
- Requirement closure: 418/418 complete
- Close gate: 11/12 passed (one gate pending, documented below)
- Fold audit: 88/88 correct
- Kit links: 8/8 correct
- All 7 PRs PASS; investigador1 reads cited per approval gate

**Warnings** (resolved per final-state authority):

1. **SC-8 (S23 comment-only decoy gap)**: lint-silent-protection.bats fixtures lack a standalone comment-only-decoy line (cross-cutting rule D6). **Resolution**: Documented as C11 hygiene item per close apply-package §1 (folded into METHODOLOGY.md K24); not a functional defect.

2. **SC-11 (CLOSE-harness-run pending)**: niagaraTest harness green gate run remains pending on Cristian's Windows session (open since C9, prerequisite for live-station writes). **Status**: Documented as post-archive dependency; does not block kit archive or C10 close. The kit is shippable; niagaraTest harness session runs as a separate deployment workflow (P-slice prerequisite per proposal).

---

## Merge Details

**Merge source**: 7 PRs off main f90b8d1 (C9 tag 1fb63d6), chained stacked-to-main.
**Final main state**: dab0807 (all 7 PR merges + C10 close commit included).
**Tag**: v0.21.0 @ dab0807 (created during PR7 close).

**Closed issues**:
- #89 (lint-timers false positive on method-local booleans)

**Version upgrade**:
- Kit: 0.20.0 → 0.21.0 (MINOR, lint precision + hygiene)
- Client module versions (no change in C10): CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan-rt/ux 2.2.0

---

## Final-State Facts (Final-State Authority)

Per the Final-State Authority section of the archive skill, sources are ranked:

1. **Persisted tasks artifact** (openspec/changes/archive/2026-09-06-build-n4-module-campaign10/tasks.md): All 7 PRs complete with [x] checkmarks; lead gate signs-off per each PR block.
2. **Explicit final-state facts in launch prompt**: Kit merged and tagged v0.21.0 at dab0807; verify-report PASS with 2 documented warnings.
3. **Intermediate snapshots** (verify-report #8403, apply-progress): Valid at verification time 2026-09-06 14:21:31; no contradictions with persisted artifacts or launch facts.

**Unresolved contradiction**: None. All sources align on PASS verdict, 7 PRs complete, 2 warnings documented as non-blocking.

---

## Pending Items (Gated, not C10 scope)

These are prerequisite deliverables for live-station write access, documented in proposal §"Scope OUT":

1. **Deploy chain** (gate P1): Coordination of 2.0.7 / 2.0.3 / 2.1.1 versions across three environments, then C9 jar deployment.
2. **Tunnel PRs #1/#2/#3** (gate P1): Merge for viewer per-user re-auth (tunnel merge + Supabase re-check + role table).
3. **niagaraTest harness session** (gate SC-11): Cristian's Windows session green run (since C9; prerequisite for HMI per-operator kiosk login + attribution verification).
4. **P2 (HMI per-operator kiosk login)**: Blocked on harness session; attribution vs per-operator RBAC decision.
5. **P3 (airDefrost flag trial)**: Defrost time trial gating.
6. **P4 (intercambiador Cuarto 3)**: Station link output integration.
7. **P5 (coolOnSensorFault station link)**: Prerequisites and scope dependencies.

---

## C11 Seeds (Folded into METHODOLOGY.md K24 via PR7 retros)

The following hygiene and technical-debt items were identified during C10 and folded into retros for C11:

1. **T1 — shared method-boundary parser unification** (B832 one-liner FN): Section-D parser was ported piecemeal into three lints (lint-timers S21, lint-ext-writable-shape S22, lint-silent-protection S23) with per-script brace_depth guards. C11 to unify into a shared toolbelt utility with consolidated test coverage and cross-script regression guards.

2. **T2 — client-root library offenders** (10 violations): Global symbols in client root libraries that should be scoped or namespaced. C11 to audit and remediate; affects 10 classes across CompPan-rt, ColdRoomPan-rt, DashboardPan.

3. **T3 — concept-row drift** (STALE rule impact): Five [concept] rows in write-path-matrix (hoaMode ×3, inhibit, freezeEnabled) transition from STALE to 0 after PR6. C11 to review whether these are truly aspirational or should be sourced in the real tree (design vs deferred implementation).

4. **T4 — guard-pins metadata consistency** (S23 comment-only decoy): Fixtures lack standalone comment-only decoys across the test suite (C9 lesson 21). C11 hygiene sweep to add decoys where missing and audit the consistency rule.

---

## Mechanics Summary

- **Observation IDs recorded for traceability**:
  - Proposal: #8392 (sdd/build-n4-module-campaign10/proposal)
  - Spec: #8394 (sdd/build-n4-module-campaign10/spec)
  - Design: (embedded in proposal + spec; no separate design observation for C10)
  - Tasks: #8396 (sdd/build-n4-module-campaign10/tasks)
  - Verify-report: #8403 (C10 verify: PASS with 2 warnings)

- **Artifacts present in archive**:
  - [x] proposal.md (37796 bytes)
  - [x] design.md (85156 bytes)
  - [x] explore.md (25585 bytes)
  - [x] spec.md (41984 bytes, folded from 8 domain specs under specs/)
  - [x] tasks.md (50707 bytes, 7 PRs all [x] checkmarked)
  - [x] .gentle-ai-instance (metadata)
  - [x] specs/ subdirectory with domain-specific delta specs (cross-cutting, lint-timers, ext-writable-shape, silent-protection, run-pure-test, write-path-stale, client-gitignore, campaign-close)

- **Main specs directory**: Does not exist (openspec/specs/ was never created for C10; delta specs remain in change/specs/). No merge step required.

- **Archive location**: `/home/cristian/modulos_niagara_n4/niagara-tools/openspec/changes/archive/2026-09-06-build-n4-module-campaign10/`

- **Source moved**: ✓ (git mv succeeded, diff-r empty, source gone)

---

## Key Decisions & Lessons

1. **EW10 contract change management** (S22): CompPan-rt 0 → 1 warning on `faultReset` requires re-pinning by NAME (not just absence). The C9 RED tip was at revision 3726722 with EW10=0; C10 RED tip 954ebd7 corrects to EW10=1 and includes the doAckAlarm counter-example (B831-G1). Never read a stale C9 RED; always re-read current PR RED at apply.

2. **Brace-depth guard unification path** (S21/S22/S23): Three lints now use the section-D method-boundary parser with `brace_depth >= 2` guard, but it was ported script-by-script (K13: cite by branch, re-read tip at apply). C11 seed T1 marks the unification work.

3. **Comment-only decoy strip rule** (D6, C9 lesson 21): Fixtures must carry a standalone `//` or `/* */` decoy that does not satisfy the functional pin. This is a cross-cutting rule; inconsistencies identified as C11 seed T4.

4. **Stale matrix row visibility** (S25): The new STALE advisory makes per-row deferred-implementation gaps visible. The [concept] marks at PR6 transform 5 STALE rows to 0 (a documented OBSERVED flip). C11 seed T3 to clarify design intent vs deferred work.

5. **Delivery strategy lock-in** (auto-chain, stacked-to-main): All 7 PRs chained in one wave with no P-gate dependencies (all P-slices wait on Cristian's Windows session). This proved correct; no pre-requisite creep during apply.

---

## Closing Statement

Campaign 10 successfully closed the precision gaps opened by Campaign 9's speed-focused refactor, addressing three coarse lints and two hygiene defects. The kit is shippable at v0.21.0 (dab0807); niagaraTest harness green and P-slice prerequisites remain on the critical path to live-station write access. All SDD artifacts have been archived; the change cycle is complete.

**Next action**: Per proposal §"Scope OUT", await Cristian's niagaraTest harness session completion before proceeding with P-slice work or live deploys. C11 seeds logged for future hygiene campaigns.

---

**Archived by**: sdd-archive phase executor
**Artifact store**: hybrid (openspec repo-local + Engram project niagara-research)
**Hybrid-mode persistence**: Archive report saved to Engram topic key `sdd/build-n4-module-campaign10/archive-report` (this document)
