<!-- review-status: pending -->
<!-- kit-retro -->
# CI: server-side enforcement — and the two coupling lessons its first run earned

Date: 2026-09-04 · Module: kit · SDD: build-n4-module-continuity Campaign 5 AG-PR2 (v0.15.1)

AG-PR2 added GitHub Actions CI so the same checks the pre-commit checklist and the pre-push hook cover run
server-side on every PR — un-bypassable, unlike the opt-in client hook. On its FIRST runs the CI immediately
caught two real latent problems that were invisible on the developer's machine, which is precisely the value
of a clean-environment server net. This retro is the exit-(a) artifact of the CI feature PR (a substantive
feature carries its own retro, per the AG-PR1 ruling).

## What this PROVED / the deltas

1. **CI = server-side, un-bypassable enforcement (complements the opt-in client hook).**
   `.github/workflows/ci.yml` runs on every push to `main` and every pull request: `shellcheck`
   (scripts + toolbelt + tests, matching CONTRIBUTING §6), `bats tests/*.bats`, and
   `sweep-build-state.sh` (ledger ↔ INDEX coherence). Least privilege (`permissions: contents: read`).
   The pre-push hook (AG-PR1) is immediate but opt-in per clone; CI is universal — together they are both
   immediate (local) and un-bypassable (remote).
   → **DELTA (landed):** the workflow above.

2. **PIN the CI linter version — an unpinned linter drifts newer on the runner and fails what passed
   locally.** The first CI run failed `shellcheck` on findings the developer's older local shellcheck never
   emitted (SC2317 on the indirectly-invoked `check_*`, SC2015 on an `A && B || C` guard, SC2154 on a bats
   `$stderr`). None were real bugs — they were version drift. Fix: pin shellcheck to a known release
   (v0.10.0, downloaded in the workflow) so CI is reproducible, and make the code clean under that pinned
   version (documented `# shellcheck disable` for the false positives + a behavior-identical `if` refactor of
   the arg-guard).
   → **DELTA (landed):** pinned shellcheck in ci.yml; SC-disable comments in verify-module.sh / ng-deploy.bats;
     build.sh arg-guard refactor. **RULE (proposed):** pin any linter/tool a CI gate depends on; an unpinned
     tool turns a green build red with no code change.

3. **Check for ENVIRONMENT COUPLING — a check that resolves a path via `$HOME` passes on a dev machine and
   FAILS on a clean/CI checkout (sibling to the stale-checkout trap).** `tests/kit-links.bats` L1 resolved
   `SKILL.md` by probing `$HOME/.claude/skills/build-n4-module/SKILL.md` — present on the developer's box,
   absent on a fresh runner — so L1 passed locally and failed in CI. `SKILL.md` is the launcher, external to
   the repo by design. Fix: resolve references IN-REPO only; treat a known-external pointer (the launcher) as
   external and skip it; verify the check still bites with `HOME=/nonexistent`.
   → **DELTA (landed):** L1 skips `SKILL.md` as an external pointer, no `$HOME` probe. **RULE (proposed):** a
     gate/check must resolve against the repo (or an explicitly-declared external), never a developer's
     machine state; prove it under a clean environment (`HOME=/nonexistent`, a fresh clone) before trusting
     a local green. This is the same "stale/hidden local state misleads work" family as the
     stale-checkout trap ([[2026-09-04-campaign4-close-process-meta-lessons]] lesson 1).

## Cost / evidence
- Green run: GitHub Actions run 33913424180 (success) on branch build-n4-ag-ci — checkout@v5, bats, pinned
  shellcheck 0.10.0, shellcheck, bats tests, sweep all pass.
- The two failures it first surfaced: run 33912310228 (shellcheck drift) and 33912901072 (L1 `$HOME`
  coupling) — both fixed, both invisible to the local `bats tests/*.bats` (104/104) beforehand.
- Gate dogfood: the LIVE pre-push gate correctly allowed every AG-PR2 push (CI/tests/root files are not
  build-relevant; the one commit touching verify-module.sh + build.sh used a trivial trailer, exit b).

## Nota de alcance
Repo-infra (CI) + behavior-preserving lint fixes; no kit control-logic or corpus change. The two RULES are
PROPOSED (propose-never-apply): consider a one-line pointer from CONTRIBUTING (pin CI tools) and the
METHODOLOGY "Kit maintenance" section (env-coupling check) next time they are edited; do NOT add speculatively.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| CI runs shellcheck+bats+sweep on every push/PR, server-side | [CERT] | .github/workflows/ci.yml; green run 33913424180 |
| An unpinned shellcheck drifted newer on the runner and failed what passed locally | [CERT] | run 33912310228 SC2317/SC2015/SC2154; local shellcheck was clean |
| L1 was coupled to $HOME and failed on a clean checkout | [CERT] | run 33912901072 "dangling reference: SKILL.md"; kit-links.bats L1 old $HOME probe |
| The L1 fix stays biting under a clean env | [CERT] | L1 passes with HOME=/nonexistent; a renamed retro still resolves nowhere → fails |

Connections: [[2026-09-04-campaign5-gate-activation]]; [[2026-09-04-campaign4-close-process-meta-lessons]].
