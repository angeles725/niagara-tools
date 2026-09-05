# Archive Report — build-n4-module-kit-v0.2

**Date closed**: 2026-09-01 · **Status**: ARCHIVED · **Release**: niagara-tools v0.4.0 · kit v0.2 (GROWING)
**Tag**: `v0.4.0` = `495388b` (pushed) · **History**: linear, 0 merge commits.

## Summary

Folded the three propose-never-apply kit retros plus the operator bitácora into the `build-n4-module` kit, made the verify checklist executable as `toolbelt/verify-module.sh`, rewrote `build.sh`, promoted `mirror-niagara-home.sh`, added `stored-repack.sh`, and covered the toolbelt with generated-fixture bats suites. Delivered by three cooperating Claude sessions (Investigador1 coordinator, Investigador2 docs, QA tests/gates) across six linearly-merged PRs.

## PRs (all rebase/ff-only)

| PR | Title | Merge SHA | Owner |
|----|-------|-----------|-------|
| #2 | docs(kit): fold 41 proven build lessons into kit v0.2 | d6b6d13 | Investigador2 |
| #1 | feat(kit): verify-module gate, build.sh rewrite, bats coverage | 887f0c2 | QA |
| #3 | chore(release): build-n4-module-kit v0.2 / niagara-tools v0.4.0 | ec7e66e | Investigador1 |
| #5 | fix(kit/toolbelt): stop build.sh --help leaking first code line (+B7) | 4b0d047 | QA |
| #4 | docs(kit/build-verify): build.sh usage line + third mirror guard | 78815a5 | Investigador2 |
| #6 | docs(changelog): correct the v0.4.0 bats test count | 495388b | Investigador1 |

## Counts

- 41 lessons folded (dashboard 22 · logic 9 · build-verify 8 · METHODOLOGY/BUILD-LOOP/SOURCES/wb-widgets the rest).
- 4 toolbelt scripts: verify-module.sh (the gate), build.sh (rewrite), mirror-niagara-home.sh (promoted + guards), stored-repack.sh.
- 27 new bats cases (V1–V9, B1–B7, M1–M5, S1–S3, L1–L3) + ng-deploy.bats 26 = 53 total; 0 committed binary fixtures.
- Versions: kit v0.1 → v0.2; niagara-tools 0.3.0 → 0.4.0.

## Gate (final, on v0.4.0 = 495388b)

bats 53/53 · shellcheck clean (scripts + toolbelt + bats + helpers) · verify-module.sh on real DashboardPan jars 12 PASS / 2 SKIP (stored), --stored FAIL by design · `git ls-files -s scripts/ng-deploy.sh` = 100755 · no `scripts/` commits in the change · `git log --merges main` = 0.

## Doctrine settled

verify-module.sh = the gate (independent of who built the jar); build.sh = recommended WSL build that runs the gate (slotomatic for every profile with sources); ng-deploy.sh = station deploy wrapper (backup → build → copy → type-count verify; slotomatic guard rt-only). "Primary/fallback" wording removed. Rule: a jar that has not passed verify-module.sh does not go to a station.

## Evidence correction folded

A1 (restart-persistent timer) demoted from [CERT-live] to [CERT] (Clock/BAbsTime API) + [INFER · pending station smoke-test] (runtime re-arm never smoke-tested). [CERT-live] list = B6, G2, G3.

## Follow-ups (NOT in scope of v0.2)

1. git tag v0.3.0 missing — tag the v0.3.0 commit retrospectively.
2. CONTRIBUTING.md §8 stale "no GitHub remote".
3. ng-deploy.sh runs slotomatic rt-only — a -ux @NiagaraType edit is not regenerated (documented in build-verify.md, not fixed).
4. Commit-attribution conflict: harness trailer vs CONTRIBUTING §6 (PR #1 without trailer, others with) — operator decides.
5. DashboardPan index.html stale `.frame aspect-ratio` masked by `#frame` (latent; fixing the module was out of scope).
6. A1 restart re-arm pending a station smoke test; STORED jar loading on a running station still TODO(verify).

## Artifacts

explore #7937 · proposal #7938 · spec #7939 · design #7940 · tasks #7943 · launcher-diff #7944 · apply-progress #7945 · verify-report #7947 (PASS-WITH-WARNINGS, 0 critical). Retro: niagara-research/retros/2026-09-01-build-n4-module-kit-v0.2-retro.md (propose-never-apply).
