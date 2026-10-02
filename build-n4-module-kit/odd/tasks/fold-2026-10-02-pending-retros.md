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
- [ ] WU0b (→ issue #199, outside this campaign) · gen-lint-index advisory follow-ups from the WU0 review: Auto column must match an actual invocation (not a substring/comment); keep ALL Usage/Exit header lines (lint-timers exit 2/3, lint-write-path --strict); document/remove the NR<=5 join cap; treat an awk failure as an error; trim SKILL.md step-5 duplicate enumeration; amend METHODOLOGY K19 + fragment-merge rule to the index.
- [x] WU1 · `types/frontend-standard.md` + dashboard/BUILD-STATE `ui_profile` — frontend-standard Δ1,Δ11-Δ15;
      deployment-profiles Δ1-Δ3; rc-file-split Δ1,Δ3,Δ4. Route: delegated direct writer (new type doc + 5 doc
      files — writer trigger). PARTIAL promotion of all three retros: INDEX rows stay `pending`; owed Δ recorded
      in the kit BUILD-STATE self-envelope open_issue.
- [x] WU2 · dashboard runtime doctrine (`types/dashboard.md`) — reliability Δ1-Δ4,Δ6,Δ8,Δ9; frontend-standard
      Δ6,Δ7,Δ9; amps-alarms Δ4,Δ5; auto-lock Δ3; rc-file-split Δ2; persistent-config Δ5; servlet-write-audit Δ3;
      deployment-profiles Δ6. Route: delegated direct writer (3 type docs + BUILD-STATE + feature doc — writer
      trigger). PARTIAL promotion of all 8 retros: no INDEX flip; owed Δ in the kit BUILD-STATE open_issue.
- [x] WU3 · `rc-scan.sh` extensions + `lint-spa-poll-no-recovery.sh` success-time rework + bats — reliability Δ1/Δ2/Δ3
      lint halves, Δ5, Δ7 (retro FULLY folded, INDEX row + marker flipped); frontend-standard Δ2,Δ3,Δ5; rc-file-split Δ6;
      commissioning-lessons Δ6; persistent-config Δ4 (rc-scan half). Route: delegated direct writer (2 scripts + 2 bats +
      fixtures + 6 docs — writer trigger).
- [x] WU4 · frontend tooling (`lint-vendor-floor.sh`, ESLint config, `hmi-sweep.js`, client CI doc) — frontend-standard Δ4
      (folded-by-Δ10), Δ10, Δ16 (retro FULLY folded, Δ8 a recorded deferral); amps-alarms Δ3; rc-file-split Δ5 (retro
      FULLY folded); deployment-profiles Δ5; persistent-config Δ4 no-scroll half; report-module `--profile`/`--legacy`
      (WU3 gap). Route: delegated direct writer (3 new tools + report-module + bats + 6 docs — writer trigger).
- [x] WU5a · logic persistence/restart/backup doctrine — restart-seq Δ5,Δ6; pressure-staging Δ2-Δ4; persistent-config
      Δ1 (doc half),Δ2; merged seed rule = auto-lock Δ4 + commissioning Δ5 + station-backup Δ3 (doc half). Route: delegated
      direct writer (logic.md + METHODOLOGY + BUILD-STATE + feature doc — writer trigger). PARTIAL promotion: no INDEX flip.
- [x] WU5b · logic safety/protection/structure doctrine — amps-alarms Δ6,Δ7; restart-seq Δ7-Δ9; continuous-fan Δ2,Δ3;
      commissioning Δ2,Δ13; persistent-config Δ6; auto-lock Δ1,Δ2; alarm-console Δ1; decision-logic Δ1-Δ3. Route: delegated
      direct writer (logic.md + logic-authoring.md + METHODOLOGY + BUILD-STATE + INDEX — writer trigger). decision-logic-decomposition
      FULLY folded (flipped; `lint-size.sh` recorded as a deferred lint candidate); the other retros stay `pending`.
- [x] WU6a · false positives in existing rt lints (verify-module UNITS, lint-delays, lint-arbitrary-ord marker) —
      comppan-fase2-amps-alarms Δ1; panccadia-restart-seq-comp-lockout-hours Δ1, Δ4. Route: delegated direct writer
      (3 scripts + 3 bats + fixtures + 4 docs — writer trigger). PARTIAL promotion of both retros: no INDEX flip.
- [x] WU6b · new rt/servlet checks (link-target flags, CS4, facade slot-coverage, console-only advisory, write audit) —
      commissioning-lessons Δ1, Δ3, Δ9; persistent-config Δ1/Δ3 (lint + dashboard halves); alarm-console Δ3;
      servlet-write-audit Δ1, Δ2 (retro FULLY folded, INDEX row + marker flipped). Route: delegated direct writer
      (1 new lint + 5 scripts + 6 bats + fixtures + 5 docs — writer trigger).
- [x] WU7 · build/preflight/source-of-truth — comppan-fase2-amps-alarms Δ2; panccadia-restart-seq-comp-lockout-hours Δ2, Δ3;
      continuous-fan-post-defrost-delay Δ1; client-source-of-truth Δ1-Δ3 (retro FULLY folded, INDEX row + marker flipped);
      change-tier-time-budgets Δ5 (BUILD-LOOP half); WU4 gap (build.sh --ui-profile/--legacy). Route: delegated direct writer
      (3 toolbelt scripts + 2 new scripts/ helpers + 6 bats + 5 docs — writer trigger).
- [x] WU8 · deploy, backup and release gates — roll-forward-recovery Δ1-Δ4; station-backup Δ1,Δ2; auto-lock Δ5,Δ7;
      restart-seq Δ10; commissioning Δ10; kit-meta Δ4,Δ5; deployment-profiles Δ7; operator-manual Δ2. Route: delegated direct
      writer (BUILD-LOOP + METHODOLOGY + build-verify + distribution + ledgers — writer trigger). FULL: roll-forward-recovery,
      dashboard-deployment-profiles (flipped); the others stay `pending` with owed Δ in the kit BUILD-STATE open_issue.
- [x] WU9 · commissioning and post-deploy triage — commissioning-lessons Δ4,Δ5,Δ8,Δ11; station-backup Δ3; alarm-console Δ2;
      site-fault-triage Δ1,Δ2; ask-dont-assume Δ5; version-defect-ledger Δ4 (merged with commissioning Δ11). Route: delegated
      direct writer (2 scripts + 1 new script + 3 bats + fixtures + 4 docs — writer trigger). `obix-link-audit.sh` kept in WU9
      (142 lines, minimal). PARTIAL promotion of all 6 retros: no INDEX flip.
- [x] WU10 · process: tiers, behavior questions, orchestration — commissioning Δ7, Δ12, Δ14; change-tier Δ1-Δ4, Δ6,
      Δ5 ORCHESTRATION half; behavior-decisions Δ1-Δ4, Δ6; auto-lock Δ1 SKILL half, Δ6; pressure-staging Δ1 (retro FULLY
      folded, flipped); amps-alarms Δ8; kit-meta Δ1 + WU0 K19/fragment-merge leftover (Δ3); site-fault Δ3; rc-file-split Δ5
      METHODOLOGY pointer; persistent-config Δ3 METHODOLOGY row; METHODOLOGY rc-scan list pointer. Route: delegated direct
      writer (4 doc files + BUILD-STATE + INDEX — writer trigger).
- [x] WU11 · close gate and ledgers; final INDEX flips; release. WU11a = remaining deltas (PR #204, merge 9321df3);
      WU11b = ledger reconcile, 13 INDEX flips, disposition markers, close retro, v0.29.0 (PR #206). Route: inline
      (parent writer of WU11; lint + bats + docs, then ledger/INDEX/release files).

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

- 2026-10-02 WU3 (branch `feat/fold-wu3-rc-scan`, chained on `feat/fold-wu2-dashboard-runtime`): `toolbelt/rc-scan.sh`
  gains 8 check ids — browser-floor (CSS `inset:`, non-grid `gap`, `aspect-ratio`, `min(`/`max(`/`clamp(`,
  `backdrop-filter`; JS `??=` `||=` `&&=` `.replaceAll(` `.at(` `structuredClone(` column-0 `await`; WARN, FAIL under
  `--strict` or `--profile hmi|both`), disabled-gate (login-gated files), fetch-no-signal, setinterval-async,
  innerhtml-server, datauri-budget (FAIL > 20480 chars, WARN with `--legacy`), orphan-page (cross-file pass),
  inline-block-size (> 300 lines) — plus `--profile`/`--legacy` flags, the `rc-scan: allow <id>` marker and
  `--strict` promoting every WARN. The profile is passed by the caller; rc-scan never reads BUILD-STATE.
  `lint-spa-poll-no-recovery.sh` reworked to the success-time doctrine (failure-count or reload-on-error recovery
  WARNs; `lastOk*` clock timestamp + compare + reload is clean). Docs: `types/dashboard.md` (HMI kiosk browser
  floor, critical-write disabled-gate rule, enforcement pointers), `types/frontend-standard.md` § Enforcement,
  BUILD-LOOP §5 rc-scan line, regenerated `toolbelt/INDEX.md`. RC5 `clean` fixture now uses `fetchT` (doctrine change).
  Evidence:
  - RED: `bats tests/rc-scan.bats` 10 not ok (RC11, RC12, RC13, RC16, RC18, RC20, RC22, RC24, RC26, RC28
    positives; the negatives pass vacuously before the checks exist); `bats tests/lint-spa-poll-no-recovery.bats`
    SPR2 + SPR5 not ok (old lint accepted the failure-count and reload-on-error shapes).
  - GREEN: `bats tests/rc-scan.bats` 29/29 ok; `bats tests/lint-spa-poll-no-recovery.bats` 11/11 ok.
  - Observed mutations (each flips its named test, then restored byte-identical): RC11, RC13, RC16, RC18, RC20,
    RC22, RC24, RC26, RC28 (rc-scan.sh), SPR2, SPR8 (lint-spa-poll-no-recovery.sh).
  - Real smoke (local client checkout, read-only): rc-scan on the deployed dashboard `-ux` reports the retro's known
    defects (orphan `page-graficas`, `setInterval(poll)`, 9 fetch-no-signal, 4 innerhtml-server, 7 datauri-budget,
    20 flex-gap) in ~4 s on a 3.6 MB page.
  - `bats tests/*.bats`: 735 ok / 0 not ok (58 env skips), exit 0.
  - shellcheck 0.11.0 (CI pins 0.10.0): only pre-existing SC2329 info in `lint-config-sanity.sh:58`.
  - `lint-guard-pins.sh --strict .` exit 0 (SPR2, SPR8 MATCH); `gen-lint-index.sh --check` exit 0;
    `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0 (172 folded, 172 cited).
  - Owed (recorded in the kit BUILD-STATE open_issue): `report-module.sh` passing `--profile`/`--legacy` to rc-scan
    (WU4/WU7); METHODOLOGY.md:131 rc-scan check list (WU10/WU11).

- 2026-10-02 WU4 (branch `feat/fold-wu4-frontend-tooling`, chained on `feat/fold-wu3-rc-scan`): new
  `toolbelt/lint-vendor-floor.sh` (node + acorn: ES2020 classic-script parse FAIL, ES-module-only WARN, token-level API
  scan WARN — comments/strings never flagged —, `--strict`; exit 4 + SKIP row naming node/acorn when missing); kit ESLint
  flat config `toolbelt/eslint.config.mjs` (no imports; ecmaVersion 2020, script, inline browser globals, no-unused-vars
  locals, max-lines-per-function 60 warn, no-console except error, eqeqeq) + `toolbelt/eslint/rows-formatter.cjs` +
  pinned `toolbelt/eslint/package.json` (eslint 10.11.0, acorn 8.18.0; node_modules/lock gitignored); `report-module.sh`
  `--profile hmi|lan|both|unknown` + `--legacy` passed to rc-scan, ESLint relay on each `-ux` artifact's own rc js,
  lint-vendor-floor relay on rc/vendor (missing tool = SKIP row, not an env fault); new `toolbelt/hmi-sweep.js`
  (generic rewrite of a client-repo sweep: app-agnostic `.nav-item` + `--subtab` enumeration, `--scenario` with
  `--api-match` forwarding, document no-scroll FAIL, unnamed inner scroller WARN, occluded target FAIL; exit 4 when
  puppeteer-core/Chrome missing). Docs: `build-verify.md` § Frontend verify step + § Client repository CI;
  `types/frontend-standard.md` § Vendored libraries verdict rule + § Enforcement rewritten (owed note closed);
  `types/dashboard.md` HMI kiosk sweep pointer + behavior-neutral split rule (rc-file-split Δ5 placed next to DJS1, not
  METHODOLOGY.md as proposed — outside this WU's surface); BUILD-LOOP §5 tool line + report-module usage; regenerated
  `toolbelt/INDEX.md`.
  Evidence:
  - RED: `bats tests/lint-vendor-floor.bats tests/hmi-sweep.bats` 16/16 not ok (scripts absent, exit 127);
    `bats tests/report-module.bats` RM24, RM25, RM27-RM31 not ok (RM26 negative passes vacuously: unknown flag exit 2).
  - GREEN: lint-vendor-floor 9/9, hmi-sweep 7/7 (HS6/HS7 real headless Chrome sweeps of the file:// fixtures with
    puppeteer-core 25.12.0), report-module 35/35 (RM33 against real eslint 10.11.0). RM32/RM33 were added after the
    implementation (config-load and end-to-end pins), not RED-first.
  - Observed mutations (each flips its pin, restored byte-identical): VF3, VF5, VF6 (lint-vendor-floor.sh), RM24
    (drop --profile pass-through), RM25 (drop --legacy), RM27 (relay no eslint rows).
  - CI parity: without node_modules the acorn/eslint/browser pins skip with a reason; the degrade pins VF3, VF3b, VF4,
    HS5, RM28, RM30 run wherever node runs (VF3 needs no node at all).
  - Verification commands: see the WU4 commit / PR body.
  - Gaps: `build.sh` calls report-module without `--profile`/`--legacy` (WU7); review finding R3-spa-init-only-lastok
    (lint-spa-poll-no-recovery.sh accepts a `lastOk` assigned only at load time) — not fixed here.
- 2026-10-02 WU5a (branch `feat/fold-wu5-logic-doctrine`, from origin/main a392aac): `types/logic.md` new § Restart &
  persistence (state→mechanism table; debounced station save Δ2; file-backed accumulator backup Δ5; Windows-safe
  dest/.bak/.tmp replace Δ6; revision/recency restore via a pure selector pressure-staging Δ2; four proofs Δ3; ONE merged
  seed-before-first-execute rule extending `hoursSeeded` — auto-lock Δ4 + commissioning Δ5 + station-backup Δ3);
  § Staging & interlocks HOA bullet REWRITTEN in place (persisted, never TRANSIENT — persistent-config Δ1 supersedes the
  old TRANSIENT/restart→Auto rule); `METHODOLOGY.md` rt checklist adapter-default pin (pressure-staging Δ4) + Schema
  seed-order pin. Owed halves recorded in the kit BUILD-STATE open_issue (Δ1 lint → WU6b; §6.b rows → WU9).
  Evidence (passive docs, no RED applicable): kit-links + build-retro-sync 36 ok / 0 not ok; sweep-build-state exit 0;
  sweep-fold-audit --strict exit 0 (171/171); gen-lint-index --check fresh; public-repo grep 0 hits outside citation
  tokens. Commit: WU5a work-unit commit on this branch.
- 2026-10-02 WU5b (same branch): `types/logic.md` § Safety fail-modes — ONE release-point-gate rule (restart-seq Δ8 +
  continuous-fan Δ2,Δ3: never assume prior output state; audit every release-point caller; branch on the output's state
  before release), observed-flag predicate (Δ9), opt-in AUTO-only proof-fault lockout cross-linked to HOA-OFF dominance (Δ7),
  wear-counter rotation-liveness + proof-fault rotation regression test (amps-alarms Δ6 + commissioning Δ13); § RT control
  logic field-terms restatement + glossary (commissioning Δ2); § Protection anatomy auto-restarting protection (persistent-config
  Δ6); new § Flag vs console alarm (alarm-console Δ1); § Pure-class extraction named-phase `step()` (decision-logic Δ1);
  § Composition per-concern size smells (Δ2). `types/logic-authoring.md` new § slotomatic last-comment-line Javadoc
  (amps-alarms Δ7). `METHODOLOGY.md` automatic-action latched trace (auto-lock Δ1), negative-criterion test enabled +
  near-miss (auto-lock Δ2), advisory size smell (decision-logic Δ2), no-restructure of deployed modules (decision-logic Δ3).
  INDEX + marker flip: decision-logic-decomposition → `folded`. Evidence in the WU5b commit body / PR (bats, sweeps,
  gen-lint-index, public-repo grep). Commit: WU5b work-unit commit on this branch.
- 2026-10-02 WU6a (branch `feat/fold-wu6a-rt-lint-fp`, from `origin/main`): `toolbelt/verify-module.sh` facets-req treats
  `BFacets.makeNumeric(<unit>, ...)` as a unit and any `makeNumeric(` as a precision (precision-only `makeNumeric(<int>)` still
  WARNs missing UNITS); `toolbelt/lint-delays.sh` reads `makeSeconds`/`makeMinutes`/`makeHours`/`makeDays` in delay arguments and
  `BFacets.MIN` facets (one `facet_min` helper replaces the four duplicated MIN parsers) and accumulates a multi-line
  `Clock.schedule*(` call to its closing paren (12-line cap); `toolbelt/lint-arbitrary-ord.sh` honors
  `// lint-arbitrary-ord: reviewed <reason>` on the same or the preceding comment line, reason mandatory. Docs: `types/logic.md`
  § Safety fail-modes & timers, `build-verify.md` § Verify; regenerated `toolbelt/INDEX.md`; BUILD-STATE WU6a open_issue.
  Evidence:
  - RED: `bats -f VMN tests/verify-module.bats` VMN1, VMN2 not ok (VMN3 negative control passes); `bats -f "LD1[3-7]"
    tests/lint-delays.bats` 5/5 not ok; `bats tests/lint-arbitrary-ord.bats` AO4, AO5, AO6 not ok (AO7 negative passes).
  - GREEN: lint-delays 16 ok + 1 env skip (LD10); lint-arbitrary-ord 8/8; VMN 3/3; facets-lint unchanged green.
  - Observed mutations (each flips its named test, restored from a scratch copy): VMN1, VMN3 (verify-module.sh, documentary),
    LD13, LD16 (also flips LD17), AO4 (also flips AO5), AO6.
  - Real smoke (client tree, read-only): lint-delays output byte-identical before/after on the deployed client source
    (7 rows) — the author had already worked around both gaps by hand.
  - `bats tests/*.bats`: 747 ok / 0 not ok (58 env skips), exit 0. shellcheck 0.11.0: only the pre-existing SC2329 info in
    `lint-config-sanity.sh:58`; no new `A && B || C` (SC2015) in added lines. `lint-guard-pins.sh --strict .` exit 0 (AO4, AO6,
    LD13, LD16 MATCH); `gen-lint-index.sh --check` exit 0; `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0
    (172 folded, 172 cited).
  - No retro flipped: comppan-fase2-amps-alarms owes Δ2/Δ3/Δ6/Δ7/Δ8; restart-seq-comp-lockout-hours owes Δ2/Δ3/Δ5-Δ10.
- 2026-10-02 WU9 (branch `feat/fold-wu9-commissioning-triage`, from `origin/main` ad928b0): new `toolbelt/obix-link-audit.sh`
  (wiring-map source column vs the live links of one component; `--xml` offline or `--obix <base> --component <path>` one
  read-only GET with credentials from `OBIX_USER`/`OBIX_PASS` fed to `curl -K -`, `--insecure` opt-in; `--table 1|2`;
  MATCH/MISMATCH/MISSING/SKIP + summary; exit 0/1/3); `toolbelt/commissioning-verify.sh` `--values-owed <file>` (default
  `<module-root>/docs/values-owed.md`; MANUAL row per owed slot, PASS when all provided, MANUAL ask when absent, exit 3 on a
  missing named file) + MANUAL rows persisted-state-restart, alarm-routing, consumer-impact, link-source-audit;
  `toolbelt/triage-console.sh --site` (SITE rows for device-offline / duplicate-device-id / comm-timeout / history-flood on
  blocks no own channel claims; count, first/last seen, distinct raw messages; after own rows, count desc; never changes exit).
  Docs: BUILD-LOOP §6.a attribute-before-fixing + scenario live check; §6.b item 1 rewritten to obix-link-audit, new items
  6-9 (persisted state across first restart, alarm routing, consumer impact, values owed by the field); commissioning-verify
  paragraph; METHODOLOGY Consumer impact table rule; `types/dashboard.md` wiring-map → obix-link-audit cross-reference.
  Fixtures: new commissioning ones live in the existing `tests/fixtures/commissioning/values-owed/` (not
  `tests/fixtures/commissioning-verify/`); triage ones in `tests/fixtures/triage-console/`.
  Evidence:
  - RED: `bats tests/triage-console-site.bats` TCS1, TCS3, TCS4, TCS6 not ok (unknown option, exit 3; TCS2/TCS5 negatives
    vacuous); `bats tests/commissioning-verify.bats` CV-owed1-4 + CV-manual2 not ok (CV-owed5 vacuous: unknown option exit 3);
    `bats tests/obix-link-audit.bats` OLA1-OLA6 not ok (script absent, exit 127). OLA7 added after GREEN (empty-link-set pin).
  - GREEN: triage-console-site 6/6 + triage-console 11/11; commissioning-verify 11/11; obix-link-audit 7/7.
  - Observed mutations (each flips its pin, restored byte-identical): drop the `if (SITE)` guard → TCS5; drop the offline
    override → TCS1, TCS4, TCS6; drop the filled/provided skip → CV-owed1, CV-owed3; `NR == FNR` instead of the FILENAME
    guard → OLA7; match any source → OLA1, OLA5; credentials on argv (`-u`) → OLA5.
  - `bats tests/*.bats` (serial): 791 ok / 1 not ok (kit-links L1 — a backticked path to the not-yet-existing link-target
    lint in BUILD-STATE; reworded) → `bats tests/kit-links.bats` 11/11 after the fix; 66 env skips.
  - shellcheck 0.11.0 (CI pins 0.10.0): only the pre-existing SC2329 info in `lint-config-sanity.sh:58`; no `A && B || C`
    in added lines. `lint-guard-pins.sh --strict .` exit 0; `gen-lint-index.sh --check` fresh (no lint header changed);
    `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0 (175 folded, 175 cited).
  - Gap: the link-target-flags lint (commissioning Δ1) is WU6b and absent on this base — `commissioning-verify.sh` does not
    pass it `--wiring-map` yet. Commissioning Δ9 (wiring-map display-name column, slot-coverage facade FAIL) stays WU6b.
  - Commit: the WU9 work-unit commit on this branch.

- 2026-10-02 WU6b (branch `feat/fold-wu6b-rt-checks`, from `origin/main`): new `toolbelt/lint-link-target-flags.sh`
  (LTF1 READONLY on a link-in target — comment-block convention or `--wiring-map` Table 2; LTF2 TRANSIENT OPERATOR
  `*Mode`/`*Hoa` slot, placed in the same lint because both are flags on a link endpoint), `lint-config-sanity.sh` CS4
  WARN, `slot-coverage.sh` facade default-FAIL (`*Panel`/`*Facade`, `--facade`), `generate-wiring-map.sh` display-name +
  full-ord columns (`--lexicon`, auto-discovery; also fixed single-line annotations being dropped), `lint-silent-protection.sh`
  `ADVISORY … console-only:` row (own severity so golden G-adapter "0 WARN" holds; never arms `--strict`), `lint-write-path.sh` audit-call WARN (`--strict` promotes);
  docs: `types/dashboard.md`, `types/security.md` § 1.3, `types/logic.md`, BUILD-LOOP tool lines, regenerated INDEX.
  Evidence:
  - RED (before each change): LTF-* 6/6 not ok (script absent, exit 127); LCS-floor not ok; SC-facade + SC-facade-flag not ok;
    GWM-display, GWM-display-auto, GWM-columns not ok; SP-console + SP-console-strict not ok; WP-audit + WP-audit-strict not ok.
  - GREEN: lint-link-target-flags 6/6, lint-config-sanity 7/7, slot-coverage 18/18, generate-wiring-map 15/15,
    lint-silent-protection 16/16, lint-write-path 25/25.
  - Observed mutations (each flips its named test, restored byte-identical): LTF-comment, LTF-map, LTF-transient, LCS-floor,
    SC-facade, GWM-display, SP-console, WP-audit, WP-audit-ok.
  - Real smoke (read-only client trees): LTF2 flags the TRANSIENT HOA slots of the pre-fix facade + evaporator unit and LTF1
    the READONLY defrost-skip link-in mirrors; CS4 flags the shipped floor/cutout combination on a copy with the floor set to 1
    (clean on the fixed default 0); console-only flags exactly the two alarm-only trips (SP-smoke re-pinned); the audit
    check is clean on the real servlet (audited); wiring map shows 170/192 rows with no lexicon key on the real facade.
  - `bats tests/*.bats`: 756 ok / 0 not ok (58 env skips), exit 0 (first run caught golden-parser G-adapter pinning
    "0 WARN" for an alarm-only trip → the console-only row got its own ADVISORY severity instead of WARN).
  - shellcheck 0.11.0 (CI pins 0.10.0): only pre-existing SC2329 info in `lint-config-sanity.sh`; no new `A && B || C`.
  - `lint-guard-pins.sh --strict .` exit 0; `gen-lint-index.sh --check` exit 0; `sweep-build-state.sh` exit 0;
    `sweep-fold-audit.sh --strict` exit 0 (173 folded, 173 cited).
  Gaps: report-module.sh wiring (new lint, facade FAIL, console-only) → WU7; logic.md "HOA is TRANSIENT" bullet contradicts
  LTF2 → WU5a (persistent-config Δ1 doc half); METHODOLOGY conformance row (persistent-config Δ3) → WU10/WU11.
- 2026-10-02 WU8 (branch `feat/fold-wu8-deploy-gates`, from origin/main 32bff11): `BUILD-LOOP.md` §4.c pre-built revert
  build (roll-forward Δ2); §6 new MANDATORY pre-deploy gates — target-distribution boot smoke (commissioning Δ10), full station
  backup before any -rt install (station-backup Δ1; the `--no-backup` bullet rewritten in place), versioned jar archive + sha256
  (roll-forward Δ3), deploy checklist names the opt-in enabling slot + live value (auto-lock Δ5) and every BComponent-only flag
  lifecycle as a station-smoke item (restart-seq Δ10), observed two-checkout handoff (auto-lock Δ7); roll-forward recovery
  doctrine (roll-forward Δ1); remote-access preconditions (kit-meta Δ5); `niagara-tools/scripts/ng-deploy.sh` full path at first
  use (kit-meta Δ4). `METHODOLOGY.md` 4-layer-stack cold-boot bullet REWRITTEN in place (target distribution/version,
  commissioning Δ10; same in `build-verify.md` item 3) + Schema / upgrade safety roll-forward bullet. `types/distribution.md` §10
  Downgrade row REWRITTEN in place (opposite-polarity rule) + new § 12 station backup/provisioning/fleet (station-backup Δ2, corpus
  B39), § 13 release package (deployment-profiles Δ7), § 14 client documentation deliverable (operator-manual Δ2).
  roll-forward Δ4: pointer added only on the slot-type-change retro's `retros/INDEX.md` row (retro bodies untouched); the outage
  retro target does not exist — recorded as a note. FULL promotion + flip: roll-forward-recovery, dashboard-deployment-profiles.
  No script: no WU8 delta asked for one. Evidence (passive docs, no RED applicable): see the WU8 commit body.
  Gap (outside surface): `types/issues-and-gotchas.md` station-stuck-at-boot list still names Downgrade without the roll-forward
  caveat (WU11).

- 2026-10-02 WU7 (branch `feat/fold-wu7-build-preflight`, from origin/main): `toolbelt/build.sh` repo-root hint (exit 10 lists
  gradle roots up to 3 levels below the argument), post-jar copy lock (`FileAlreadyExistsException` -> exit 32 naming each
  `build/libs` jar rebuilt this run, stale ones "not rebuilt"), `--plugin-version`/env forwarded to `preflight.sh --plugin-version`
  (and used by the m2 WARN), `--ui-profile hmi|lan|both|unknown` + `--legacy` forwarded to `report-module.sh`, header-only
  `--help` via awk; `toolbelt/preflight.sh --plugin-version` (env fallback); `toolbelt/lint-structure.sh` L14 (hardcoded
  `gradlePluginVersion` literal FAILs; pin LS14); new `scripts/check-client-source.sh` (git half of Δ1; kit-links L2 keeps git
  out of toolbelt) and `scripts/check-skill-drift.sh` (Δ3; a scripts/ helper, not sweep-build-state.sh: CI has no installed
  skill); `source_of_truth` DECLARED field (How to read + every module envelope, `unknown` where unconfirmed); kit-links L12
  routes `scripts/check-*.sh` through BUILD-LOOP; BUILD-LOOP §0.b/§4/§7, build-verify.md, skill/SKILL.md step 4; regenerated
  `toolbelt/INDEX.md`.
  Evidence:
  - RED: `bats tests/build-sh.bats` BS-repo-root-hint, BS-copy-lock, BS-copy-lock-stale, BS-preflight-plugin,
    BS-preflight-plugin-env, BS-ui-profile not ok; `tests/preflight.bats` PF-plugin-override, PF-plugin-env,
    PF-plugin-override-missing not ok; `tests/lint-structure.bats` LS14 not ok; `tests/check-client-source.bats` 14/14 and
    `tests/check-skill-drift.bats` 6/7 not ok (scripts absent; SD7 vacuous); kit-links L12 not ok once the helpers existed.
  - GREEN: build-sh 38/38, lint-structure 17/17, PF-plugin* 5/5, check-client-source 14/14, check-skill-drift 7/7, kit-links ok.
  - Observed mutations (restored byte-identical): LS14 (L14 row disabled -> LS14 not ok), BS-copy-lock (signature regex
    broken -> not ok), BS-ui-profile (--profile pass-through dropped -> not ok).
  - Real smoke (read-only): check-client-source on the ColdRoomPan module root -> PASS (declared path, commit unknown);
    on a non-git client tree -> WARN; check-skill-drift on this machine -> FAIL (installed launcher differs from the tracked one).
  - `bats tests/*.bats`: 800 ok / 1 not ok (L1 dangling `toolbelt/lint-link-target-flags.sh` named in the new BUILD-STATE
    entry; reworded, kit-links re-run 0 not ok), 66 skips.
  - shellcheck 0.11.0 (CI pins 0.10.0): only pre-existing SC2329 info in `lint-config-sanity.sh:58`; `&& .* ||` in the diff: 0.
  - `lint-guard-pins.sh --strict .` exit 0 (LS14 MATCH); `gen-lint-index.sh --check` fresh; `sweep-build-state.sh` exit 0;
    `sweep-fold-audit.sh --strict` exit 0.
  - Owed: continuous-fan-post-defrost-delay Δ2/Δ3 (WU5b); comppan-fase2-amps-alarms Δ1, Δ6-Δ8; restart-seq Δ1, Δ4-Δ10;
    change-tier-time-budgets Δ1-Δ4, Δ6 + Δ5 ORCHESTRATION half (WU10); report-module.sh wiring handed off by WU6b
    (link-target-flags relay, slot-coverage facade FAIL mapping, silent-protection ADVISORY rows) — outside the WU7 surface,
    recorded in the kit BUILD-STATE open_issue.

- 2026-10-02 WU10 (branch `feat/fold-wu10-process`, from `origin/main` 3bdbb02): `BUILD-LOOP.md` new §0.c blast-radius tier
  table P0-P3 (ONE rule = commissioning Δ14 + change-tier Δ1; both citations) + RDD candidate sizing (pressure-staging Δ1); §1
  Behavior decisions gate (behavior Δ1); §2 recipe pointer (change-tier Δ2); §4.a tier-scoped loop + §5 non-skippable floor (ONE
  rule = change-tier Δ3 + commissioning Δ12); §7 assumption register (behavior Δ4). `ORCHESTRATION.md` §3.a-§3.d (tier topology,
  design-shard checklist commissioning Δ7, writer prompt clauses change-tier Δ5 / behavior Δ3 / amps-alarms Δ8, shared working-tree
  git discipline kit-meta Δ1), §4 estimate pricing (change-tier Δ6), §7 candidate shaping (Δ4), §8 per-phase timing (auto-lock Δ6),
  new §9 Incident journal (site-fault Δ3). `skill/SKILL.md` behavior gate + classify rule, step 1c hard gate, step 1d question
  format (behavior Δ6), auto-lock Δ1 design-checklist item, § Recipe (change-tier Δ2). `METHODOLOGY.md` behavior question catalog
  (behavior Δ2), floor checkbox, K19 + fragment-merge rewritten to the generated-index doctrine (kit-meta Δ3), rc-scan list
  pointer, persistent-config Δ3 lintable row, rc-file-split Δ5 pointer. Retro flipped: comppan-pressure-staging. Evidence
  (passive docs, no RED applicable): see the WU10 commit / PR body. Owed: change-tier Δ5 BUILD-LOOP half (WU7), behavior Δ5
  (WU9), auto-lock Δ5/Δ7 (WU8), amps-alarms Δ2 (WU7), kit-meta Δ2 (WU11)/Δ4/Δ5 (WU8), site-fault Δ1/Δ2 (WU9); persistent-config
  is now fully folded but its marker is outside the WU10 surface → WU11 flip.
  - Merge of origin/main (WU7/WU8/WU9 merged first): conflicts in `BUILD-STATE.md`, `METHODOLOGY.md` and this doc resolved by
    hand at entry granularity (ledger: main's entries kept, WU10's "LANDED WU10" edits re-applied per entry by a 3-way word
    merge, WU10 entry added once; METHODOLOGY: both sides' bullets kept). The owed items above that WU7/WU8/WU9 carried are
    now LANDED on main; only kit-meta Δ2 (WU11) stays owed. WU7/WU8/WU9 entries' "(WU10)" owed notes flipped to LANDED.
  - Checks after the merge: `bats tests/*.bats` 856 ok / 0 not ok (66 env skips); mcp-n4-kit unittest 456 OK; shellcheck
    0.11.0 only the existing SC2329 info message; `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` 181/181 cited;
    `lint-guard-pins.sh --strict` exit 0; `gen-lint-index.sh --check` fresh.
  - Review: native RDD on the merge commit — consent envelope returned, granted; one lens (reliability), APPROVED and
    acknowledged (authority burned). 4 informational, non-blocking findings for WU11: R3-tier-p0-no-feature-doc (§0.c says the
    tier goes in the feature doc, but P0 has none), R3-floor-vs-p0-ceremony (§5 floor RED/GREEN vs the P0 structural-only
    ceremony), R3-k19-guard-claim (K19 names only kit-links L5/L11; L4 coverage not shown), R3-ledger-wu2-stale-paren.
  - PR #202 (https://github.com/angeles725/niagara-tools/pull/202); merge commit recorded in the WU11 entry.

- 2026-10-02 WU11a (branch `feat/fold-wu11-close`, from `origin/main` a0a9b90 = WU10 merge of PR #202): remaining
  deltas. New `templates/VERSION-LEDGER.md` (client scaffold: versions table, root-cause classes 1-8, repeated-class
  escalation, Open items); `BUILD-LOOP.md` §0 client-ledger read (version-defect-ledger Δ2), §7 ledger close gate (Δ1),
  repeated-class escalation (Δ3), operator-manual lockstep close gate `manual: updated|n/a` (operator-manual Δ1) +
  pointer to the new advisory `toolbelt/lint-manual-labels.sh` (Δ3); `corpus-index.md` retitled to its real range
  B29–B1027 + rows B33/B34/B39/B48 (kit-meta Δ2; BUILD-LOOP §2, METHODOLOGY top and skill/SKILL.md range text
  updated); `types/issues-and-gotchas.md` G1 Downgrade roll-forward caveat; WU10 review advisories reconciled (§0.c P0
  records the tier in the commit body; §5 floor RED/GREEN `n/a (no behavior change)` for a no-behavior P0 and the P0
  ceremony cell says the floor still runs; K19 names L5/L11/L12 and states L4 guards types docs, not scripts); the
  dated token in two bats comments made date-less (the .md citations were already date-less, so fold-audit credits
  kit-meta). Route: inline (the parent writer of WU11; lint + bats + 8 doc files).
  Evidence:
  - RED: `bats tests/lint-manual-labels.bats` 7/7 not ok (script absent). GREEN: 8/8 ok (ML8 added with the
    word-boundary prefix rule after the first real smoke, not RED-first; pinned by its observed mutation).
  - Observed mutations (restored byte-identical): ML2 (not-found check dropped), ML4 (`\uNNNN` decode dropped), ML8
    (word-boundary test dropped) — each flips its pin.
  - Real smoke (read-only, a client DashboardPan tree): the manual BEFORE the cut-out rename commit WARNs on the old
    label `"Temp. de corte (serpentín)"`; the manual after it does not; 49 other WARNs are prose quotes/paraphrases
    (advisory by design). 0.5 s.
  - `bats tests/*.bats` (serial): 864 ok / 0 not ok (66 env skips); mcp-n4-kit unittest 459 OK; shellcheck 0.11.0 only
    the pre-existing SC2329 info (`lint-config-sanity.sh`); `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh
    --strict` 181/181; `lint-guard-pins.sh --strict .` exit 0 (ML2/ML4/ML8 MATCH); `gen-lint-index.sh --check` fresh.

- WU11a review: native RDD, consent granted, 4 lenses (risk, resilience, readability, reliability) APPROVED and
  acknowledged (lineage review-95af45cf0754b8ad, authority burned). 7 informational findings → issue #199:
  R2-escalation-rule-divergent-copy (template vs BUILD-LOOP §7 wording), R2-ledger-class-map-drift (BUILD-LOOP §0 class
  map vs template table), R2-row-format-doc-mismatch (lint-manual-labels header row format), R2-unexplained-exit-sentinel
  (python exit 10), R3-ml-haystack-overbroad (*.json/*.xml in the haystack), R3-ml-strict-clean-unpinned (no `--strict`
  clean pin), R3-ml-unreadable-file-aborts (an unreadable UI file aborts with exit 3). PR #204 CI green, merged 9321df3.

- 2026-10-02 WU11b (branch `feat/fold-wu11b-release`, from 9321df3, merged with origin/main ac41750 = mcp-n4 #203/#205):
  scripted check over all 23 campaign retros — every Δ1..Δn cited as `[ev: retro <stem> Δn]` in the kit core, 0
  missing — so the 13 still-`pending` retros flip `folded` (INDEX rows + line-1 markers): comppan-fase2-amps-alarms,
  panccadia-restart-seq-comp-lockout-hours, panccadia-commissioning-lessons, change-tier-time-budgets,
  comppan-auto-lock-indicator, behavior-decisions-ask-dont-assume, panccadia-version-defect-ledger,
  panccadia-persistent-config-hoa, operator-manual-lockstep, station-backup-before-deploy, alarm-console-design,
  site-fault-triage-and-incident-journal, kit-meta-hygiene-2026-10-01. CXF proposal (#122) and the 2026-09-20 apply
  worklist: INDEX `<!-- disposition: … -->` + a line-2 disposition comment; `sweep-build-state.sh --age` skips such
  rows (RED D7 not ok → GREEN; mutation "skip dropped" flips D7). Ledger (`BUILD-STATE.md`, hand-edited per entry):
  stale duplicate WU1 entry removed, WU2 stray parenthesis fixed, RECONCILED clause on the 13 campaign entries, CAMPAIGN
  CLOSED + DEFERRED out-of-repo preview entries, module `retro_pending` cleared (ColdRoomPan, CompPan, DashboardPan,
  UmbrellaDashboard — their retros are folded), kit envelope version/last_change/last_commit/last_session + index row.
  Sanity: every `[ev:` closed, no duplicate list items, no entry outside a `build-state.v1` block. Close retro
  `retros/2026-10-02-retro-fold-campaign-close.md` (4 proposed Δ). Release: VERSION 0.28.1 → 0.29.0 (the brief's
  0.22.0 → 0.23.0 was stale: v0.23.0..v0.28.1 already exist), CHANGELOG v0.29.0 with WU0-WU11 and mcp-n4 #201/#203/#205.

## Delivery record
| WU | PR | Merge |
|---|---|---|
| retros landed | #183 | ed4a27b |
| WU0 | #185 | bfeb9c4 |
| WU1 | #186 | af2a4db |
| WU2 | #188 | 6289617 |
| WU3 | #191 | f4c8fec |
| WU4 | #192 | e64187f |
| WU5a/WU5b | #193 | 32bff11 |
| WU6a | #194 | ad928b0 |
| WU6b | #195 | 3bdbb02 |
| WU8 | #196 | 2354eb0 |
| WU7 | #197 | 5608866 |
| WU9 | #198 | 8112c58 |
| WU10 | #202 | a0a9b90 |
| WU11a | #204 | 9321df3 |
| WU11b | #206 | tag v0.29.0 |

## Status: complete
Campaign closed 2026-10-02. Open follow-ups: issue #199 (review advisories incl. WU0b), the close retro's 4 proposed
deltas, the DEFERRED BUILD-STATE entries.
