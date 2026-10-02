# ODD feature — Fold the 23 pending build-n4 retros (2026-09-24 .. 2026-10-01) into the kit core

**Created**: 2026-10-02 · **Owner**: ODD orchestrator · **Repo**: angeles725/niagara-tools (main, PUBLIC)

## Objective
Fold the 147 proposed deltas of the 23 pending retros (integrated into main by PR #183) into the
build-n4-module kit core, as a chain of work-unit PRs grouped by target file and theme.

## Problem / why
The retros carry lessons from commissioning, dashboard frontend reliability, deploy/backup, logic
persistence and process tiers. Until they are folded, `types/*.md`, BUILD-LOOP and the lints do not
carry them, so the next builder repeats the same failures.

## Authorization & delivery
- Operator authorized on 2026-10-02: ODD + RDD (consent pre-granted), commit + push + PR + PR view +
  issues + merge, chained and automatic.
- Delivery: one PR per work unit, each branched from the current `origin/main` and merged after CI
  green (+ RDD when the assessment marks it due) before the next starts. The ~400-line budget is
  advisory.
- Repository is PUBLIC: new text uses generic names (no customer, site, host, IP or user names).

## Fold contract
Same as `odd/tasks/fold-2026-09-21-23-retro-deltas.md` § Fold contract:
`[ev: retro <stem> Δn]` citation on every folded rule; INDEX row + retro marker flip to `folded` only
when ALL of a retro's deltas landed (or are recorded as explicit deferrals in the kit BUILD-STATE
self-envelope `open_issues`); BUILD-STATE self-envelope touched in the same range; trailer
`Retro: promotion (folds <ids> from <stems>)`; gates: sweep-build-state, sweep-fold-audit --strict,
shellcheck 0.10.0, bats, lint-guard-pins --strict when a lint is added.

## Decisions taken (orchestrator, recorded for review)
- HMI browser-floor check (commissioning Δ6 = persistent-config Δ4 = reliability Δ5): WARN by default,
  FAIL under `--strict` or when the module declares `ui_profile: hmi`.
- Toolbelt scripts never call version control (kit-links L2): the client-source-of-truth Δ1/Δ3 checks
  that need `git` live in a `scripts/` helper, not in `toolbelt/`.
- frontend-standard Δ4 is covered by Δ10 (ESLint `max-lines-per-function`, `no-unused-vars`): fold Δ10,
  record Δ4 as folded-by-Δ10.
- Out-of-repo deltas (frontend-standard Δ8, deployment-profiles Δ4 → `tools/dashboard-preview.py` in
  the research repo) are recorded as deferred open_issues, not dropped.
- roll-forward Δ4 pointer target `2026-09-25-panccadia-unknown-unit-outage.md` does not exist; only the
  existing slot-type retro gets the pointer; recorded in progress.

## Work units (tasks)
- [x] WU0 · generated lint index (`toolbelt/INDEX`, BUILD-LOOP §5/§6.b reference it) — kit-meta Δ3.
      Route: delegated direct writer (new script + bats + 4 doc/test files — writer trigger). PARTIAL
      promotion: kit-meta Δ1/Δ2/Δ4/Δ5 owed to later WUs, so its INDEX row stays `pending`; recorded in
      the kit BUILD-STATE self-envelope (open_issue + last_session).
- [ ] WU0b · gen-lint-index advisory follow-ups from the WU0 review: Auto column must match an actual invocation (not a substring/comment); keep ALL Usage/Exit header lines (lint-timers exit 2/3, lint-write-path --strict); document/remove the NR<=5 join cap; treat an awk failure as an error; trim SKILL.md step-5 duplicate enumeration; amend METHODOLOGY K19 + fragment-merge rule to the index.
- [x] WU1 · `types/frontend-standard.md` + dashboard/BUILD-STATE `ui_profile` — frontend-standard Δ1,Δ11-Δ15;
      deployment-profiles Δ1-Δ3; rc-file-split Δ1,Δ3,Δ4. Route: delegated direct writer (new type doc + 5 doc
      files — writer trigger). PARTIAL promotion of all three retros: INDEX rows stay `pending`; owed Δ recorded
      in the kit BUILD-STATE self-envelope open_issue.
- [x] WU2 · dashboard runtime doctrine (`types/dashboard.md`) — reliability Δ1-Δ4,Δ6,Δ8,Δ9; frontend-standard
      Δ6,Δ7,Δ9; amps-alarms Δ4,Δ5; auto-lock Δ3; rc-file-split Δ2; persistent-config Δ5; servlet-write-audit Δ3;
      deployment-profiles Δ6. Route: delegated direct writer (3 type docs + BUILD-STATE + feature doc — writer
      trigger). PARTIAL promotion of all 8 retros: no INDEX flip; owed Δ in the kit BUILD-STATE open_issue.
- [ ] WU3 · `rc-scan.sh` extensions + `lint-spa-poll-no-recovery.sh` success-time rework + bats.
- [ ] WU4 · frontend tooling (`lint-vendor-floor.sh`, ESLint config, `hmi-sweep.js`, client CI doc).
- [ ] WU5a · logic persistence/restart/backup doctrine.
- [ ] WU5b · logic safety/protection/structure doctrine.
- [ ] WU6a · false positives in existing rt lints (verify-module UNITS, lint-delays, lint-arbitrary-ord marker).
- [ ] WU6b · new rt/servlet checks (link-target flags, CS4, facade slot-coverage, console-only WARN, write audit).
- [ ] WU7 · build/preflight/source-of-truth.
- [ ] WU8 · deploy, backup and release gates.
- [ ] WU9 · commissioning and post-deploy triage (+ `obix-link-audit.sh` possibly split out).
- [ ] WU10 · process: tiers, behavior questions, orchestration.
- [ ] WU11 · close gate and ledgers; final INDEX flips; release.

## Progress / evidence
- 2026-10-02: PR #183 merged (ed4a27b) — 25 pending retros on main; #156 closed superseded, #157 auto-merged.
- Delta inventory produced by a read-only explorer (147 deltas, duplicates listed in the decisions above).

- 2026-10-02 WU0 (branch `feat/fold-wu0-lint-index`): new `toolbelt/gen-lint-index.sh` writes
  `toolbelt/INDEX.md` from each `lint-*.sh` header (description, Usage, Exit contract, Auto = run by
  `report-module.sh`, `[ev:]` tags); `--check` exits 1 on a stale/missing index; header-contract
  violation exits 1. BUILD-LOOP §5 pre-gate enumeration replaced by the index pointer + non-lint tools +
  lint doctrine + frozen provenance tags; report-module bullet and §6.b point at the index; skill/SKILL.md
  step 5 points at it; kit-links L5 accepts the index for `lint-*.sh`, new L11; CI step
  `gen-lint-index --check`. Evidence:
  - RED: `bats tests/gen-lint-index.bats` 13/13 not ok before the script existed (exit 127).
  - GREEN: `bats tests/gen-lint-index.bats tests/kit-links.bats` 24/24 ok.
  - Observed mutation GLI-stale: compare replaced by `if false` → GLI-stale not ok; restored → ok.
  - Observed L5 bite: a row deleted from INDEX.md → L5 not ok naming the lint, `--check` exit 1.
  - `bats tests/*.bats`: 713 ok / 0 not ok (58 env skips), exit 0.
  - shellcheck 0.11.0 (CI pins 0.10.0): only finding is pre-existing SC2329 (0.11-only info) in
    `lint-config-sanity.sh:58`, untouched by WU0.
  - `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0 (171 folded, 171 cited);
    `lint-guard-pins.sh --strict .` exit 0; `gen-lint-index.sh --check` exit 0.
  - Not run: `.githooks/pre-push` (needs a push range).
  - Follow-up outside WU0 surface: METHODOLOGY K19 + fragment-merge rule still name BUILD-LOOP §5 as
    the lint routing line; amend to name `toolbelt/INDEX.md` (recorded as kit open_issue).

- WU0 RDD: assessed high (process_boundary, shell_source); consent pre-granted; 4-lens review APPROVED and acknowledged (lineage review-1e008815decc0963); 9 advisory findings → WU0b.

- 2026-10-02 WU1 (branch `feat/fold-wu1-frontend-standard`, chained on `feat/fold-wu0-lint-index`): new
  `types/frontend-standard.md` (profiles hmi/lan/both with a tag on every rule, 11-section rule set, rc/ component
  layout, technology table, vendored-library rule, vendor catalog, LAN profile, prototype→module, owed-enforcement
  note); `types/dashboard.md` cross-links only (ux pointer + `ui_profile`, DJS1 rc/ tree, no-Java split precondition,
  vendor refinement, framework-decision pointer, prototype decomposition, panel classic scripts, new § LAN browser
  access); `BUILD-STATE.md` How to read `ui_profile` + kit envelope open_issue/log line; skill/SKILL.md dashboard row
  and README types list name the new file (kit-links L1/L4). Citation tokens are date-less
  (`[ev: retro dashboard-frontend-standard Δn]`) because sweep-fold-audit strips the date from the stem.
  Evidence (passive docs, no RED applicable):
  - `bats tests/kit-links.bats tests/build-retro-sync.bats`: 36 ok / 0 not ok.
  - `bats tests/*.bats`: 713 ok / 0 not ok (58 env skips), exit 0.
  - `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0 (171 folded, 171 cited);
    `gen-lint-index.sh --check` exit 0 (fresh).
  - Public-repo grep (customer/site/host/IP names) over added text: 0 hits.
  - Commit: the WU1 work-unit commit on `feat/fold-wu1-frontend-standard` (sha in the PR).
  - Gap for WU11: WU0 cited `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]` (dated token), which
    sweep-fold-audit will NOT credit to stem `kit-meta-hygiene-2026-10-01` when that row flips to folded.

- 2026-10-02 WU2 (branch `feat/fold-wu2-dashboard-runtime`, chained on `feat/fold-wu1-frontend-standard`):
  `types/dashboard.md` new § Poll loop reliability (fetchT timeout Δ1, self-scheduled polls Δ2, one timing
  config Δ8, success-time watchdog Δ3 REWRITING the failure-count watchdog text in place — the
  panccadia-defrost-sequencing Δ1 citation kept —, fail-visible read path = amps-alarms Δ5 + auto-lock Δ3
  merged); `setInterval` mentions in the live-data matrix and the REST-poll bullet retargeted; facade
  live-value-before-link (persistent-config Δ5); write confirmation + recent-changes view (frontend-standard
  Δ7 + servlet-write-audit Δ3 merged); interaction safety (reliability Δ6); cache-bust `?v=` (rc-file-split Δ2);
  `/api/version` (deployment-profiles Δ6, rt half in new `types/structure.md` § Deployed version
  publication); slot-key contract (frontend-standard Δ6); HMI layout shell (frontend-standard Δ9), stale-data
  visibility (reliability Δ9), recovery ladder (reliability Δ4); scenario-forwarding preview mock
  (amps-alarms Δ4). `types/frontend-standard.md`: § 5/§ 7 pointers + owed-enforcement note updated.
  Evidence (passive docs, no RED applicable):
  - `bats tests/kit-links.bats tests/build-retro-sync.bats`: 36 ok / 0 not ok.
  - `bats tests/*.bats`: 713 ok / 0 not ok (58 env skips), exit 0.
  - `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0 (171 folded, 171 cited);
    `gen-lint-index.sh --check` exit 0 (fresh).
  - Public-repo grep over added lines: only retro-stem citation tokens match; 0 customer/site/host/IP hits.
  - Commit: the WU2 work-unit commit on `feat/fold-wu2-dashboard-runtime` (sha in the PR).
  Owed lint halves to WU3: rc-scan fetch-without-signal + `setInterval(async)` WARNs,
  `lint-spa-poll-no-recovery.sh` success-time rework.

## Next step
- WU3 (`rc-scan.sh` extensions + `lint-spa-poll-no-recovery.sh` success-time rework + bats) after WU2 merges.
