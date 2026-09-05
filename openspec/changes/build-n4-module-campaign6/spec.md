# Spec: build-n4-module-campaign6

**Change**: build-n4-module-campaign6
**Version target**: 0.15.1 → 0.16.0
**Spec authority**: proposal.md · explore.md · harvest.md · QA RED qa/c6-marker-index-drift cb0dd7d · QA RED qa/c6-coverage-pct-red d7e52a8
**Delivery**: auto-chain; 6 chained PRs PR1→PR2→PR3→PR4→PR5→PR6; each PR branches after its predecessor merges.

---

## 1. Scope

**In**: B1 marker-sweep (PR1); B3 CI pure-test gate (PR2); A3–A12 + K9 + harvest §M1/§M2 + A10 doctrine (PR3); A1 + A13 + A14/LC1–LC5 + A15 + A16–A18 types (PR4); B2 + B5 + D2 + MM1 + optional B4 toolbelt (PR5); C1–C3 + D4 + release (PR6).

**Out**: A2 content (already folded at METHODOLOGY.md:9+11 — INDEX row and marker flip only, in PR3); D1 scaffold; D3 screenshot; MM3–MM7; any module-repo change.

---

## 2. Capabilities

| Capability | Action | PRs |
|---|---|---|
| `kit-retro-sync-contract` | New | PR1 (sweep marker check), PR3 (citation audit introduced), PR5 (sweep-fold-audit.sh, MM1) |
| `build-n4-module-kit-doctrine` | Modified | PR3 (K1–K9, harvest §M1/§M2, A10), PR4 (LC1–LC5 + types delta) |
| `kit-ci-pure-test-gate` | Modified | PR2 |
| `module-verify-gate-plano` | New (stretch) | PR5 only if B4 lands |

---

## 3. PR1 — marker↔INDEX sweep contract (B1)

**Files**: `toolbelt/sweep-build-state.sh` · `tests/build-retro-sync.bats` · 2 retro files re-stamped.

### Requirements

**R1-1** `sweep-build-state.sh` MUST read each retro file's in-file `<!-- review-status: X -->` comment. The first whitespace-delimited token after the colon is the marker word; a ` · DATE` suffix MUST be stripped before comparison.

**R1-2** A present marker word that equals its INDEX row `review-status` value MUST NOT cause a failure for that file.

**R1-3** A present marker word that DIFFERS from its INDEX row value MUST cause exit 1 with a named diagnostic identifying the file and both values.

**R1-4** A retro file with no `<!-- review-status: -->` line MUST be treated as agreeing (absent = tolerated; exit 0 for that file).

**R1-5** A marker word not in {pending, folded} MUST cause exit 1 (out-of-domain).

**R1-6** `2026-09-02-comppan-fase1-staging.md` and `2026-09-02-dashboardpan-detail-render-doors.md` MUST be re-stamped to `<!-- review-status: folded -->` in the same PR1 commit. Their INDEX rows already say `folded`.

### Scenarios M1–M5 (must match QA RED cb0dd7d exactly)

| Test | Given | When | Then |
|---|---|---|---|
| **M1** (biting) | Retro has `<!-- review-status: pending -->` · INDEX row says `folded` | sweep runs | exit 1 |
| M2 | Retro has `<!-- review-status: folded -->` · INDEX row says `folded` | sweep runs | exit 0 |
| M3 | Retro has no `<!-- review-status: -->` line · INDEX row says `folded` | sweep runs | exit 0 |
| M4 | Retro has `<!-- review-status: fresh · 2026-09-04 -->` · INDEX row says `folded` | sweep runs | exit 1 |
| M5 | Real kit tree with both `fresh` files re-stamped to `folded` | sweep runs on real tree | exit 0 |

**Mutation proof (R1-8)**: Deleting the marker-read block from `sweep-build-state.sh` MUST flip M1 from exit 1 to exit 0, proving the check bites.

**Gate**: PR1 carries its own retro (BUILD-LOOP §7 exit a). `sweep-build-state.sh` exits 0 on the final PR1 tree.

---

## 4. PR2 — CI pure-test execution (B3)

**Files**: `.github/workflows/ci.yml` · `tests/run-pure-test.bats`.

**R2-1** CI MUST pre-fetch JUnit 4.13.2 and hamcrest-core 1.3 jars (e.g., a `./gradlew dependencies` step or an explicit gradle cache action) before running `run-pure-test.bats`.

**R2-2** After R2-1, tests P1–P6 in `run-pure-test.bats` MUST execute with 0 SKIP results in the CI log.

**R2-3** Mutation proof: mutating `run-pure-test.sh` to exit non-zero unconditionally MUST fail CI (P2 is the biting case).

**R2-4** PR2 touches only `.github/workflows/ci.yml` and `tests/run-pure-test.bats`. No kit file is changed; no retro is required and none is carried.

---

## 5. PR3 — doctrine meta cluster (A3–A12, K9, harvest §M1/§M2, A10)

**Files**: `METHODOLOGY.md` · `CONTRIBUTING.md` · `retros/INDEX.md` · 7 retro file markers.

**R3-1** Every rule added to `METHODOLOGY.md` or `CONTRIBUTING.md` MUST carry an `[ev: retro <slug>]` citation referencing its source retro.

**R3-2** Every delta MUST pass a grep-before-fold audit of the entire `build-n4-module-kit/` tree. If the rule's key assertion already exists in any core file, flip that retro's INDEX row and marker only — do not re-fold the content. PR description MUST record the audit results per delta.

**R3-3** `METHODOLOGY.md §Kit-maintenance` MUST gain rules K1–K9:

| Rule | Source retro slug | Content (one line) |
|---|---|---|
| K1/A3 | gate-exit-taxonomy-promotion | Gate exit set covers every change class; typed exit with proof-of-work guard |
| K2/A4 | campaign3-close-process-meta-lessons | High-signal/low-FP checks; scope to reported defect; mutation-prove the guard |
| K3/A5 | campaign3-close-process-meta-lessons | WARN over silent removal of a safety guard |
| K4/A6 | campaign3-close-process-meta-lessons | Marker is the promotable unit, not prose |
| K5/A7 | campaign4-close-process-meta-lessons | Coverage checks in a worktree off origin/main, never stale local main |
| K6/A8 | campaign4-close-process-meta-lessons | Grep every kit file before folding; the mined target file may not be where it already lives |
| K7/A9 | campaign5-gate-activation | A feature PR uses close-gate exit (a), not a 4th unclassified shape |
| K8/A11 | ci-server-side-enforcement | Env-coupling check: resolve against repo; prove under HOME=/nonexistent |
| K9 | run-pure-test-set-e-empty-cache | set -e probe isolation (‖ true) so die \<code\> fires, not a bare abort |

**R3-4** `METHODOLOGY.md` MUST gain §Multi-session coordination (harvest §M1): before editing any file in a shared repo, `git status`/`git diff` the target first; a dirty tree is a peer live session's work and is off-limits [ev: retro dashboardpan-2d-to-3d-port + module-authoring-mega-campaign].

**R3-5** `METHODOLOGY.md` MUST gain §Live-verify safety (harvest §M2): read-only prod checks + out-of-band negative check (no-token → 401); test creds from a file outside the repo (`chmod 600`) [ev: retro live-cutover-and-authenticated-control + obix-and-loginless-dashboard-runbooks].

**R3-6** `CONTRIBUTING.md` MUST gain K10/A10: pin any linter/tool a CI gate depends on [ev: retro ci-server-side-enforcement].

**R3-7 (A2 out)** A2 content (BDouble.make rule) MUST NOT be re-folded. It already lives at METHODOLOGY.md:9 and :11. The freeze-stat retro INDEX row MUST NOT be flipped in PR3; it is handled in PR4 after L1 lands.

**R3-8** `retros/INDEX.md` MUST flip exactly 7 rows from `pending` to `folded` in PR3: kit-continuity-and-retro-gate-campaign, run-pure-test-set-e-empty-cache, gate-exit-taxonomy-promotion, campaign3-close-process-meta-lessons, campaign4-close-process-meta-lessons, campaign5-gate-activation, ci-server-side-enforcement. Each retro file MUST receive `<!-- review-status: folded -->`.

**R3-9** No new bats tests for PR3 (doctrine-only, per RX-2). Guard: `sweep-build-state.sh` exits 0 on the post-PR3 tree with 7 rows flipped and 1 still pending (freeze-stat).

**R3-10** PR3 carries its own retro (BUILD-LOOP §7 exit a).

---

## 6. PR4 — types + build-verify + SOURCES (A1, A13, LC1–LC5, A15, A16–A18)

**Files**: `types/logic.md` · `types/dashboard.md` · `build-verify.md` · `SOURCES.md` · `corpus-index.md` · freeze-stat retro marker + INDEX row.

**R4-1** Same grep-before-fold obligation as R3-2. PR description records audit results per delta.

**R4-2** `types/logic.md §Regenerating slots` MUST gain A1/L1: annotation-only is sufficient in the build.sh/slotomatic flow; AUTO stub is needed only when compiling WITHOUT slotomatic [ev: retro freeze-stat-leds Δ2].

**R4-3** `types/logic.md` MUST gain A13/L2: control BComponent station MOUNT/ORD is integrator-placed config, not source-derivable [ev: retro dashboardpan-2d-to-3d-port Δ4].

**R4-4** `types/logic.md` MUST gain LC1–LC5 (from closed kitControl corpus, no new research blocks):
- LC1: writable priority array — 16-level first-valid-wins, null = relinquished; add NEW slot to make config oBIX-writable [ev: corpus B536+B544]
- LC2: BLoopPoint/PID — executeTime clamp [100ms,60s], disableAction bumpless pre-load, PI recommended/PID seldom [ev: corpus B539]
- LC3: multi-input null contract — nulls skipped not zeroed; latch = rising-edge vs both-edges [ev: corpus B537]
- LC4: control-logic fail-safe checklist — 6 unsafe defaults: disableAction, propagateFlags, rampTime, alarm-ext, 999 sentinel, clHVAC [ev: corpus B543]
- LC5: point-extension chain — BAlarmSourceExt vs BIntervalHistoryExt vs BCovHistoryExt; alarm-ext is notification-only, never writes value/priority-array [ev: corpus B552]

**R4-5** After LC1–LC5 land, `types/logic.md` header MUST NOT label the type "GROWING".

**R4-6** `types/dashboard.md` MUST gain A15/D1: off-station consumer must recompute `*ElapsedMs`/`*RemainingMs` derived keys from anchor slots; read DashboardReader to distinguish real oBIX slot from reader-derived value [ev: retro dashboardpan-2d-to-3d-port Δ2].

**R4-7** `build-verify.md §Known gap` MUST be retitled to "mode B ignores `--with-slotomatic`": modes A/C regenerate `-ux` slots when annotated (ng-deploy.sh:493–500); mode B never runs slotomatic (ng-deploy.sh:552–553). The exact string "Known gap — ng-deploy.sh runs slotomatic for -rt only" MUST NOT survive in any kit file after PR4.

**R4-8** `build-verify.md` MUST gain A17/V1–V4: consumer-absence delta proof; live-vs-doc precedence; verify freshness before labeling; headless-QA/CORS boundary [ev: respective retros].

**R4-9** `SOURCES.md` / `corpus-index.md` MUST gain A18/S1–S2: tool-zero ≠ absence; no `[CERT]` off a mangled decompile [ev: respective retros].

**R4-10** `retros/INDEX.md` MUST flip the freeze-stat retro (`2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md`) from `pending` to `folded`. The retro file MUST receive `<!-- review-status: folded -->`. After PR4 merges, pending count = 0.

**R4-11** `kit-links.bats` MUST exit 0. No new bats tests for doc-only deltas (per RX-2).

**R4-12** PR4 carries its own retro (BUILD-LOOP §7 exit a).

---

## 7. PR5 — tools and derivations (B2, B5, D2, MM1, optional B4)

**Files**: `toolbelt/sweep-fold-audit.sh` (new) · `toolbelt/preflight.sh` (new) · `toolbelt/slot-coverage.sh` (new) · `toolbelt/verify-module.sh` (modified) · `tests/*.bats` × ≥4.

**R5-1 (B2)** `toolbelt/sweep-fold-audit.sh` MUST, for each `folded` slug in `retros/INDEX.md`, grep core kit files for the literal pattern `[ev: retro <slug>`. When a folded slug has no citation in any core kit file, MUST WARN to stderr (non-fatal; exit 0). Scope MUST avoid false positives from INDEX rows or cross-links.

**R5-2 (B2 mutation)** Fixture: a slug marked `folded` in INDEX with no `[ev: retro slug]` citation in any core file → audit MUST WARN. Removing the WARN logic MUST flip the test.

**R5-3 (B5)** `toolbelt/preflight.sh <niagara_home>` MUST automate BUILD-LOOP §0.b: check (a) JDK 8 present; (b) pinned gradle plugin version in `<niagara_home>/etc/m2`; (c) station lock on target jar. On any failure, MUST exit a typed non-zero code with a named message identifying which check failed.

**R5-4 (B5 mutations)** Named mutations: (i) JDK 8 path missing → typed exit fires; (ii) locked jar → typed exit fires. Tests MUST use fakebin probes; no network calls; MUST pass under `HOME=/nonexistent` (K8 env-coupling rule).

**R5-5 (D2)** `toolbelt/slot-coverage.sh` MUST compare `module.lexicon` keys against declared `@NiagaraType` slots. An empty lexicon with declared types MUST WARN to stderr (exit 0). The CompPan T8 defect (empty lexicon, types declared) is the biting fixture.

**R5-6 (D2 mutation)** Removing the empty-lexicon WARN logic MUST flip the T8-fixture test from WARN-present to no-WARN.

**R5-7 (MM1)** `verify-module.sh coverage <p> <f> <w> <s>` MUST:
- Echo a percentage (float, 0–100) when `(p+f+w+s) > 0`, computed as `p / (p+f+w+s) * 100`.
- Echo the literal string `N-A` when `p+f+w+s == 0`. Zero total is NOT 100% — "no applicable checks" and "all checks passed" are distinct states.
- Exit 0 in all cases (pure computation).
- Conform to the 6-pin interface from QA RED qa/c6-coverage-pct-red (d7e52a8). **This sub-command surface supersedes the proposal's standalone `coverage(covered, total)` signature.** The spec adopts the pinned surface because QA RED d7e52a8 already commits to it with 6 test fixtures.

**R5-8 (MM1 mutation)** Returning 100 when `p+f+w+s == 0` MUST flip the N-A assertion. All 6 QA RED fixture pins MUST pass.

**R5-9 (B4, stretch)** If B4 lands: `verify-module.sh --plano <index.html>` MUST assert exactly one frame `aspect-ratio` CSS declaration AND that all four plano overlay dimensions agree. Fixture with two `aspect-ratio` declarations → exit 1.

**R5-10** All three new scripts MUST pass `shellcheck` exit 0.

**R5-11** Full bats suite MUST be ≥104 pass, ≤15 s wall time, no regressions.

**R5-12** If PR5 draft exceeds 400 authored lines, MUST split into PR5a (B2 + MM1) and PR5b (B5 + D2 + B4). Do not shrink code to fit.

**R5-13** PR5 carries its own retro (BUILD-LOOP §7 exit a).

---

## 8. PR6 — SKILL.md + openspec archive + release (C1–C3, D4)

**Files**: `~/.claude/skills/build-n4-module/SKILL.md` (outside git) · `openspec/changes/archive/` · `VERSION` · `CHANGELOG.md`.

**R6-1** `SKILL.md` decision table MUST NOT include the `state` column (growing/mature/seed not actionable). The SKILL MUST add an explicit warning that `types/wb-widgets.md` is SEED status, directing wb builders to corpus B751–B753. §Execution step 1 wording MUST align with BUILD-LOOP's orient/corpus-nav order.

**R6-2** `openspec/changes/niagara-tools-slotomatic-integration/` MUST be moved to `openspec/changes/archive/niagara-tools-slotomatic-integration/` with a supersession note (100% superseded as of v0.15.1; all 40 tasks live in ng-deploy.sh + tests; zero tasks applied).

**R6-3** `openspec/` MUST be tracked in git (committed). This satisfies the hybrid-store durability requirement.

**R6-4** `VERSION` = `0.16.0`. `CHANGELOG.md` MUST contain a release entry per CONTRIBUTING §4–5.

**R6-5** SKILL.md changes are outside git and MUST be documented in full in the PR6 description (before/after diff) and recorded in engram. Rollback is a manual re-edit.

**R6-6** Full bats suite green (≥104 pass). `sweep-build-state.sh` exits 0; pending count = 0.

**R6-7** PR6 carries its own retro (BUILD-LOOP §7 exit a).

---

## 9. Cross-cutting Requirements

**RX-1 (gate-exit)** PRs 1, 3, 4, 5, 6 are kit-changing pushes → BUILD-LOOP §7 exit (a) (carry a new retro). PR2 carries no retro (only CI/test files touched).

**RX-2 (doc-test prohibition)** PRs 3, 4, 6 are doc-only → MUST add zero new bats tests. Test padding is a defect. Sweep green is the guard.

**RX-3 (chain integrity)** Each child PR targets the branch of its immediate predecessor after merge. A child PR diff MUST NOT contain its parent's commits; rebase until clean.

**RX-4 (grep-before-fold)** Every doctrine/doc delta MUST be preceded by a grep audit of the entire `build-n4-module-kit/` tree (not only the proposed destination file). A rule found already present → flip INDEX/marker only. PR description MUST record the audit evidence.

**RX-5 (shellcheck)** All `toolbelt/*.sh` and `scripts/*.sh` MUST pass `shellcheck` exit 0 after each PR.

---

## 10. Success Criteria (from proposal §10 as pass/fail gates)

| # | Criterion | Verification |
|---|---|---|
| SC-1 | `retros/INDEX.md` pending count = 0 after PR4 | `grep pending build-n4-module-kit/retros/INDEX.md` → 0 rows |
| SC-2 | Every flipped row has `[ev: retro <slug>]` in a core kit file | `sweep-fold-audit.sh` exits 0 with 0 WARNs |
| SC-3 | sweep exits 1 when present marker ≠ INDEX row | M1 passes |
| SC-4 | CI runs P1–P6 with 0 SKIPs | CI log post-PR2 |
| SC-5 | CI fails when P2 exit code is mutated | P2 mutation proof |
| SC-6 | Every new script ships a named-mutation biting test | PR5 test inventory |
| SC-7 | A2 not re-folded; documented as already folded at METHODOLOGY.md:9+11 | PR3 description + grep |
| SC-8 | build-verify.md loses the blanket "-rt only" slotomatic claim | `rg "rt only" build-n4-module-kit/build-verify.md` → 0 matches |
| SC-9 | Full suite ≥104 pass, shellcheck exit 0, wall ≤15 s | CI green |
| SC-10 | `openspec/` tracked; slotomatic change under archive/ | `git ls-files openspec/` |
| SC-11 | `VERSION` = 0.16.0; CHANGELOG entry present | grep both files |
| SC-12 | Each PR has clean child diff and `Closes #N` | PR review |
