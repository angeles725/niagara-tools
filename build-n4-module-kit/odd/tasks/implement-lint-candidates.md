# ODD feature — Implement the documented lint candidates

**Created**: 2026-09-20 · **Repo**: angeles725/niagara-tools (main) · follows the apply-deltas campaign (#123)

## Objective
Turn the lint CANDIDATES documented (not scripted) during the apply-deltas folds into real
`toolbelt/lint-*.sh` gates, each with a `# Mutation:` guard-pin and a `tests/*.bats`.

## Close-gate approach
Each lint candidate was proposed inside an already-FOLDED delta retro, so implementing it is a
**partial promotion** of that retro: commit trailer `Retro: promotion (folds the <name> lint
candidate from <source-retro>)` + a BUILD-STATE kit-envelope log line (the structural anchor;
no INDEX flip — the source retro stays folded). One lint per PR, chained, CI green, merged.

## Feasibility triage
**IMPLEMENT (HIGH confidence, low false-positive):**
- [x] L1 · `lint-subscribe-without-unsubscribe` (rt src; WARN) — from failure-modes Δ3 (RUN4). PR pending.
- [x] L2 · `lint-no-md5-credential-digest` (src; WARN) — DONE, auto-wired report-module §5.21 + RM21.
- [x] L3 · `lint-bundled-jar-class-version` (src; FAIL) — DONE, auto-wired report-module §12 + RM23.
- [~] L4 · `split-package-check` — DEFERRED to project/CI bucket (cross-module: same package in 2 modules; not a per-module report-module lint). Pair with lint-guard-pins CI.
- [x] L5 · `lint-wb-file-chooser` (-wb; WARN) — DONE, auto-wired report-module §5.22 + RM22.

**DEFER (semantic / needs .bog or cross-facet join → false-positive-prone; a noisy lint regresses the kit's precision doctrine, Campaigns 10/11):**
- `dangling-link-watchdog` (needs cross-station link + heartbeat analysis; .bog)
- `dynamic-slot-orphan-prune` (needs started() flow analysis)
- `lint-wb-refresh`, `lint-wb-usability` (semantic view-behavior heuristics)
- `precision-facet-learn-mismatch` (cross-facet import-learn analysis)
- `cross-target-profile-component` (needs .bog + module-profile join → belongs in `bog-audit.sh`, a different subsystem, not a src lint)
- `api-response-headers` (an ENHANCEMENT to the existing `lint-servlet.sh`, not a new script — handle separately)

## Deferred wiring
Report-module.sh §5.x + BUILD-LOOP pre-gate mentions for the new lints are batched into a final
"wire new lints" PR (matching the kit's #113 precedent, which wired 9 lints at once), so each
lint PR stays minimal and CI-green.

## Progress log
- 2026-09-20: L1 authored — lint-subscribe-without-unsubscribe.sh + bats (7/7), SWU2 guard-pin MATCH, shellcheck clean.
