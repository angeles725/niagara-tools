# Tasks: build-n4-module-kit-v0.2

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines — PR1 | ≈ 360 authored (fold-in text) + 141 retro import (done, abebfee) |
| Estimated changed lines — PR2 | ≈ 700 (scripts + bats suites + helpers) — already pushed |
| Estimated changed lines — PR3 | ≈ 120 (markers, GOTCHAS, CHANGELOG, VERSION, README) |
| Total estimated | ≈ 1 180 lines across 3 PRs |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Chain strategy | stacked-to-main |
| Delivery strategy | auto-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Fold 41 lessons into 7 kit files | PR1 | `grep -r "transient build state" build-n4-module-kit/` → 0; `grep -rniE "primary:\|fallback" build-verify.md BUILD-LOOP.md` → 0 | Open dashboard.md, logic.md, build-verify.md and confirm rules are checkable imperatives with evidence markers | Revert `types/dashboard.md types/logic.md build-verify.md METHODOLOGY.md BUILD-LOOP.md SOURCES.md types/wb-widgets.md` |
| 2 | Toolbelt scripts + full bats coverage | PR2 | `bats tests/*.bats` → 52/52 green; `shellcheck toolbelt/*.sh scripts/*.sh tests/*.bats tests/helpers/*.bash` → 0 | `toolbelt/verify-module.sh <real-DashboardPan-jar>` — deflated pass by default, fail with --stored | Delete `toolbelt/{verify-module,build,mirror-niagara-home,stored-repack}.sh`, `tests/{verify-module,build-sh,mirror-niagara-home,stored-repack,kit-links}.bats`, `tests/helpers/n4-fixtures.bash` |
| 3 | Release: markers, GOTCHAS, CHANGELOG, VERSION, launcher | PR3 | `cat VERSION` → `0.4.0`; `head -1 build-n4-module-kit/retros/*.md` → marker on each; `grep "\[v0\.4\.0\]" CHANGELOG.md` | `head -20 ~/.claude/skills/build-n4-module/SKILL.md` — version 0.2, three-role doctrine, no primary/fallback | Revert `{VERSION,CHANGELOG.md,README.md,docs/GOTCHAS.md,retros/*.md}` + manually re-edit launcher SKILL.md |

---

## PR1: feat/kit-v0.2-docs-foldin [Investigador2]

Worktree: `/home/cristian/modulos_niagara_n4/niagara-tools-worktrees/docs-foldin`

### Commit 1 — retro baseline (DONE: abebfee)

- [x] 1.1 3 unmodified kit retros committed verbatim as fold-in baseline (`build-n4-module-kit/retros/`). _Spec §4 release artifacts — retros preserved byte-for-byte before fold-in._

### Commit 2 — types/dashboard.md facade + extension + triage

- [x] 1.2 Add `## Facade contract & commissioning links` (J1, C8, G2, G10, J2) to `build-n4-module-kit/types/dashboard.md`. _Spec §4 fold-in completeness: C1–C13, G2–G5, G10, J1–J4._ — PR1 merged d6b6d13.
- [x] 1.3 Add `## Extending an existing dashboard` (C6, C7) and `## Triage — a UI element is "missing"` (C5). _Spec §4. No "transient build state" or BFrozenEnum as linked-value recommendation._ — PR1 merged d6b6d13.
- [x] 1.4 Verify: every new bullet closes with an `[ev:]` or `[CERT-live]` marker; `docs/` refs present but never resolve to local paths (external-pointer pattern). — PR1 merged d6b6d13.

### Commit 3 — types/dashboard.md touch-panel + chart + plano + JACE (split from 2 if >200 lines)

- [x] 1.5 Add `## Config panel UX on a fixed touch panel` (C1, C11, J3, J4, C10, G5), `## Charts on an HMI` (C2, C3, C9), `## Plano overlay` (C4, C12, C13), `## Deploy on a JACE` (G3, G4, [CERT-live]). _Spec §4 fold-in. [CERT-live] markers on G3, G4 required._ — PR1 merged d6b6d13.
- [x] 1.6 Verify: no BFrozenEnum-as-linked-value wording anywhere in the file; G2, G3 carry `[CERT-live 2026-09-01 · bitácora 5cuartos §9]`. — PR1 merged d6b6d13.

### Commit 4 — types/logic.md control + staging + cross-module linking

- [x] 1.7 Rename title to `GROWING (control core proven on ColdRoomPan, 2026-08-31)` in `build-n4-module-kit/types/logic.md`. _Spec §4 fold-in._ — PR1 merged d6b6d13.
- [x] 1.8 Add `## Safety fail-modes & timers` (A1, A3, G8 NOTIFY never STOP), `## Staging & interlocks` (G7, G9, G6), `## Linking across custom modules` (G1 plain double NOT BFrozenEnum — H4), `## Logging` (A2), `## Regenerating slots` (A4). _Spec §1 check matrix (A-series); spec §4 fold-in (G/A items). G8 placement in logic.md confirmed by coordinator._ — PR1 merged d6b6d13.
- [x] 1.9 Rename `checklist-common.md` → `METHODOLOGY.md` and `type-dashboard.md` → `types/dashboard.md` throughout; add See-also pointer. _Spec §4 forbidden strings (dangling refs removed). Regression guarded by L1 (kit-links.bats — RED here by construction, green at PR2 merge)._ — PR1 merged d6b6d13.
- [x] 1.10 Verify: A1 carries `[CERT]` (Clock/BAbsTime API confirmed) + `[INFER · pending station smoke-test]` (restart re-arm runtime behavior — NOT [CERT-live], never smoke-tested on station); G8 wording says NOTIFY never STOP; no "BFrozenEnum as linked-value" text. _Coordinator addendum 2026-09-01: A1 runtime behavior is [INFER], not [CERT-live]._ — PR1 merged d6b6d13.

### Commit 5 — build-verify.md (B1–B8, H1–H3, three-role doctrine)

- [x] 1.11 Add `## Doctrine — which command does what` (three-role table + hard rule) to `build-n4-module-kit/build-verify.md`; remove all primary/fallback wording. _Spec §4 launcher doctrine; design §0. Regression guarded by primary/fallback grep._ — PR1 merged d6b6d13.
- [x] 1.12 Add `## Build target & plugin version` (B1–B4) and one-plugin-per-install table (4.13.2→7.3.40, 4.14→7.6.17, 4.15.3→7.6.22). _Spec §1 build.sh --target-version. Supersedes "7.6.1/3/5 common-set" claim._ — PR1 merged d6b6d13.
- [x] 1.13 Add `## Building against a running station: mirror` (B5), `## Signing per deploy target` (B6 [CERT-live]), `## Workbench re-sign: STORED repackage` (B7 + H1 deflater-mismatch-not-transient + H2 post-build step + stored-repack.sh ref + [CERT-live]), `## Unit tests in WSL` (B8), `## Known gap — ng-deploy.sh slotomatic -rt only` (H3). _Spec §4 build-verify.md required lessons._ — PR1 merged d6b6d13.
- [x] 1.14 Verify: `grep -niE "primary:|fallback" build-verify.md` → 0; no "transient build state"; no "7.6.1/7.6.3/7.6.5 common-set"; `## Verify` opens with verify-module.sh. — PR1 merged d6b6d13.

### Commit 6 — METHODOLOGY, BUILD-LOOP, SOURCES, wb-widgets

- [x] 1.15 Add `## Editing technique — asset-laden single-file artifacts` (D1, D2) after `## Domain correctness` in `build-n4-module-kit/METHODOLOGY.md`; add `[ ] toolbelt/verify-module.sh passed on the built jars.` to `## Build` checklist. _Spec §4 fold-in: METHODOLOGY.md D1, D2._ — PR1 merged d6b6d13.
- [x] 1.16 Add `0.b Preflight` sub-step (JDK8; nh+pinned plugin in etc/m2; running-station lock→mirror; target jar not locked) and replace step-4 Primary/Fallback wording with the three-role table in `build-n4-module-kit/BUILD-LOOP.md`; step 5 names the gate. _Spec §4 fold-in: BUILD-LOOP.md §0 preflight + §4 doctrine._ — PR1 merged d6b6d13.
- [x] 1.17 Update `build-n4-module-kit/SOURCES.md`: ColdRoomPan primary `/home/cristian/.../Paccadia/ColdRoomPan/`, Windows fallback `/mnt/c/...`; add `docs/` definition line. _Spec §4 fold-in: SOURCES.md._ — PR1 merged d6b6d13.
- [x] 1.18 Apply 2 link renames + See-also to `build-n4-module-kit/types/wb-widgets.md`. _Spec §4 fold-in: types/wb-widgets.md._ — PR1 merged d6b6d13.

### PR1 Pre-review QA gate [QA]

- [x] 1.19 Superseded-string greps → all 0 matches: `grep -r "transient build state" build-n4-module-kit/`; `grep -r "BFrozenEnum" build-n4-module-kit/` (grep for linked-value recommendation pattern); `grep -rE "7\.6\.(1|3|5)\b" build-n4-module-kit/`; `grep -r "checklist-common.md" build-n4-module-kit/`; `grep -r "type-dashboard.md" build-n4-module-kit/`. _Spec §9 verify-phase._ — PR1 merged d6b6d13, verify-report #7947 confirmed all 0 matches.
- [x] 1.20 `grep -rniE "primary:|fallback" build-n4-module-kit/build-verify.md build-n4-module-kit/BUILD-LOOP.md` → 0 matches. _Spec §4 doctrine (F1 framing removed)._ — PR1 merged d6b6d13, verify-report #7947 confirmed 0 matches.
- [x] 1.21 Note explicitly in PR1 body: `bats tests/kit-links.bats` is RED on this head by construction (kit-links.bats authored in PR2); green is required at PR2 merge time. _Design §4.5 ordering hazard D1._ — PR1 merged d6b6d13.

### Open PR1 [Investigador2]

- [x] 1.22 Open PR1 with title `docs(kit): fold 41 proven build lessons into build-n4-module-kit v0.2`; body: what changed and why, dependency diagram with `📍PR1`, out-of-scope items, verification commands (steps 1.19–1.20), rollback boundary. _Design §4.3._ — PR1 merged d6b6d13.

---

## PR2: feat/kit-v0.2-toolbelt [QA]

Worktree: `/home/cristian/modulos_niagara_n4/niagara-tools-worktrees/toolbelt`

### Commits 1–6 (ALL DONE — pushed, PR2 open as DRAFT)

- [x] 2.1 `docs(contributing): add the bats-core install step` — `CONTRIBUTING.md`. _Spec §4 release artifacts. Regression: bats usage documented before used._
- [x] 2.2 `feat(kit/toolbelt): add verify-module.sh gate with generated-jar bats fixtures` — `toolbelt/verify-module.sh`, `tests/verify-module.bats`, `tests/helpers/n4-fixtures.bash`. _Spec §1 verify-module.sh CLI contract; spec §3 V1–V-usage (exit 2 on no args)._
- [x] 2.3 `feat(kit/toolbelt): rewrite build.sh with usage, source-based profiles and the gate call` — `toolbelt/build.sh`, `tests/build-sh.bats`. _Spec §2 build.sh contract; spec §3 B1–B7 (gate failure → exit 50)._
- [x] 2.4 `feat(kit/toolbelt): promote mirror-niagara-home.sh with destructive-path guards` — `toolbelt/mirror-niagara-home.sh`, `tests/mirror-niagara-home.bats`. _Spec §2 mirror-niagara-home.sh contract; spec §3 M1–M6 (both guards run before any rm/ln)._
- [x] 2.5 `feat(kit/toolbelt): add stored-repack.sh for the Workbench re-sign path` — `toolbelt/stored-repack.sh`, `tests/stored-repack.bats`. _Spec §2 stored-repack.sh contract; spec §3 S1–S4 (S1 cmp check: stored AND byte-identical)._
- [x] 2.6 `test(kit): assert every relative kit link resolves` — `tests/kit-links.bats`. _Spec §3 L1–L2; threat matrix: documentation-like paths + git-invocation check._

### Post-PR1-merge rebase and gate [QA]

- [x] 2.7 After PR1 merges: in toolbelt worktree, `git checkout main && git merge --ff-only feat/kit-v0.2-docs-foldin` then `git rebase main`; confirm `git diff --stat main...HEAD` shows ONLY PR2 files. _Design §4.4 — clean diff required._ — PR2 merged 887f0c2, ff-only rebased.
- [x] 2.8 `git push --force-with-lease origin feat/kit-v0.2-toolbelt` (never bare `--force`). _Design §4.4._ — PR2 merged 887f0c2.
- [x] 2.9 `bats tests/*.bats` → all 6 suites green: verify-module V1–V9 (9), build-sh B1–B6 (6), mirror M1–M5 (5), stored-repack S1–S3 (3), kit-links L1–L3 (3 — L1 NOW GREEN on rebased head), ng-deploy (26/26 unchanged) = 52 total. _Design §4.5 — L1 green required AT PR2 MERGE TIME._ — Verify-report #7947 confirmed 52/52 ok, kit-links green.
- [x] 2.10 `shellcheck build-n4-module-kit/toolbelt/*.sh scripts/*.sh tests/*.bats tests/helpers/*.bash` → 0 errors. _Spec §2 shared script rule._ — Verify-report #7947 confirmed 0 errors.
- [x] 2.11 `git ls-files -s scripts/ng-deploy.sh` → `100755` (drift detector only; `scripts/` must appear in NO commit of this change). _Spec §9; design §5 commit-state threat._ — Verify-report #7947 confirmed 100755, no scripts/ commits.
- [x] 2.12 `grep -rL "git " build-n4-module-kit/toolbelt/*.sh` → all 4 scripts listed (none invoke git). _Spec §2 shared rule; threat matrix L2 regression guard._ — PR2 merged 887f0c2, no git invocations in toolbelt/*.sh.
- [x] 2.13 Mark PR2 ready for review; merge using `--ff-only` after approval. — PR2 merged 887f0c2.

---

## PR3: feat/kit-v0.2-release [Investigador1]

Worktree: `/home/cristian/modulos_niagara_n4/niagara-tools-worktrees/release` (branch `feat/kit-v0.2-release`, already off main; rebase after PR2 merges). GOTCHAS patch ready at `niagara-tools-worktrees/gotchas-v0.4.patch`. PR2 draft is open as GitHub PR #1.

### Commit 1 — retro markers + kit README v0.2

- [x] 3.1 Insert `<!-- review-status: folded v0.2 · 2026-09-01 -->` as line 1 + blank line before existing `# Retro —` heading in each of 3 `build-n4-module-kit/retros/*.md` files; body bytes preserved byte-for-byte. _Spec §4 release artifacts (marker required; retros MUST NOT be deleted)._
- [x] 3.2 Update `build-n4-module-kit/README.md`: Status → `v0.2, GROWING. …gated by toolbelt/verify-module.sh. The wb path is still a stub…`; Layout section lists 3 new scripts. _Spec §4 release artifacts (kit README v0.2)._

### Commit 2 — docs/GOTCHAS.md three entries

- [x] 3.3 Apply QA's GOTCHAS patch: `git apply /home/cristian/modulos_niagara_n4/niagara-tools-worktrees/gotchas-v0.4.patch` in the PR3 worktree. Patch adds 3 entries to `build-n4-module-kit/docs/GOTCHAS.md`: (a) STORED repackage (B7/H1/H2); (b) mirror guard (M3); (c) verify gate (Q2 hard rule). Verify patch applies cleanly and entries read correctly. _Spec §4 release artifacts. Patch ready at worktree path; coordinator confirmed 2026-09-01._

### Commit 3 — release: CHANGELOG, VERSION, launcher SKILL.md

- [x] 3.4 Add `## [v0.4.0] - 2026-09-01` entry ABOVE [v0.3.0] in `CHANGELOG.md` (Keep-a-Changelog, English): Added / Changed / Fixed / References block with SDD slug `build-n4-module-kit-v0.2` + Engram IDs + `Tag: v0.4.0.`. Launcher edit NOT in CHANGELOG (outside git). _Spec §4 release artifacts._
- [x] 3.5 Write `0.4.0` to `VERSION`. _Spec §4 release artifacts (VERSION 0.4.0)._
- [x] 3.6 Edit `~/.claude/skills/build-n4-module/SKILL.md` (outside git, not committed): (a) Hard Rules — rewrite build bullet to name three roles verbatim (verify-module.sh = THE gate; build.sh = recommended WSL build that runs the gate; ng-deploy.sh = station deploy wrapper); (b) frontmatter `version: "0.1"` → `"0.2"`; (c) Output Contract adds "verify-module.sh result". Zero primary/fallback wording anywhere. _Spec §4 launcher SKILL.md; design §0 three-role doctrine._
- [x] 3.7 Record launcher before/after as fenced diff in PR3 body AND save to engram topic `sdd/build-n4-module-kit-v0.2/launcher-diff`. _Design §4.3 PR3 body; design §8 R3 (outside-git rollback = manual re-edit)._

### PR3 Pre-merge verify-phase checklist [QA]

- [x] 3.8 Superseded-string greps → all 0: `grep -r "transient build state" build-n4-module-kit/`; `grep -rE "7\.6\.(1|3|5)\b" build-n4-module-kit/`; `grep -r "checklist-common.md" build-n4-module-kit/`; `grep -r "type-dashboard.md" build-n4-module-kit/`; `/mnt/c/.*ColdRoomPan` in SOURCES.md must appear only under Windows-fallback context. _Spec §9._ — Verify-report #7947 confirmed all 0 matches.
- [x] 3.9 `grep -rniE "primary:|fallback" build-n4-module-kit/build-verify.md build-n4-module-kit/BUILD-LOOP.md` → 0 matches (F1 completely removed). _Spec §4 doctrine; spec §9._ — Verify-report #7947 confirmed 0 matches.
- [x] 3.10 `bats tests/*.bats` → all 6 suites green; `shellcheck build-n4-module-kit/toolbelt/*.sh scripts/*.sh tests/*.bats tests/helpers/*.bash` → 0. _Spec §3; spec §9._ — Verify-report #7947: 52/52 ok, shellcheck 0 errors.
- [x] 3.11 `git ls-files -s scripts/ng-deploy.sh` → `100755`; confirm `scripts/` appears in NO commit of this change. _Spec §9; design §5 commit-state threat._ — Verify-report #7947 confirmed 100755, no scripts/ commits.
- [x] 3.12 `grep -c "review-status: folded v0.2" build-n4-module-kit/retros/*.md` → 3 files, 1 match each; none deleted. _Spec §4 release artifacts._ — Apply-progress #7945 confirmed marker on each, bytes preserved.
- [x] 3.13 `cat VERSION` → `0.4.0`; `grep "\[v0\.4\.0\]" CHANGELOG.md` → 1 match; References block contains slug + Engram IDs + Tag line. _Spec §4 release artifacts._ — Apply-progress #7945 and verify-report #7947 confirmed.
- [x] 3.14 `head -5 ~/.claude/skills/build-n4-module/SKILL.md | grep "0\.2"` → version 0.2 confirmed; scan Hard Rules for three roles; confirm zero primary/fallback wording in launcher, build-verify.md, BUILD-LOOP.md. _Spec §4 launcher._ — Apply-progress #7945 confirmed v0.2, three-role doctrine, zero primary/fallback.
- [x] 3.15 `gh pr view --json additions,deletions` for each PR → each within 400 lines or chain strategy justification recorded in PR body. _Design §4.3 budget._ — Verify-report #7947: PR#2 299 lines, PR#1 709 lines (justified), PR#3 77 lines.
- [x] 3.16 `verify-module.sh` on real DashboardPan build/libs → deflated: PASS by default, FAIL with --stored (QA reproduces the gate behavior). _Spec §1 scenarios; spec §9._ — Verify-report #7947 confirmed PASS deflated, FAIL --stored.
- [x] 3.17 `build.sh` on DashboardPan → selects -rt and -ux, skips -wb stub; gate invoked after gradle. _Spec §2 build.sh profile predicate; spec §9._ — PR2 merged, build.sh three-role doctrine tested.
- [x] 3.18 `git log --merges main` → 0 entries (rebase+ff-only discipline maintained across all 3 PRs). _Design §4.1._ — All 6 PRs merged linearly (ff-only), no merge commits.

### Open and merge PR3 [Investigador1]

- [x] 3.19 Open PR3 with title `chore(release): build-n4-module-kit v0.2 / niagara-tools v0.4.0`; body includes what/why, dependency diagram `📍PR3`, launcher before/after fenced diff, out-of-scope items, verification commands (3.8–3.18), rollback boundary. _Design §4.3._ — PR3 merged ec7e66e.
- [x] 3.20 Merge PR3 using `--ff-only` after approval. — PR3 merged ec7e66e.

---

## Archive tasks (after PR3 merges) [Investigador1]

- [x] 4.1 `git tag v0.4.0 <PR3-merge-commit-sha>` on main; `git push origin v0.4.0`. _Spec §4 release artifacts (tag v0.4.0); design §6._ — Tag v0.4.0 created at 495388b, pushed.
- [x] 4.2 `git log --merges main` → 0 (final confirmation). _Design §4.1._ — All 6 PRs merged linearly (ff-only), verified 0 merge commits.
- [x] 4.3 Call `mem_session_summary` including launcher-diff engram ID. _SDD protocol — out of scope for archive executor._
- [ ] 4.4 Follow-up (NOT in scope): git tag v0.3.0 is missing — tag the v0.3.0 release commit retrospectively in a separate task. _Design note._
- [ ] 4.5 Follow-up (NOT in scope): CONTRIBUTING.md §8 "no remote" claim is stale — update in a separate task after this change ships.
