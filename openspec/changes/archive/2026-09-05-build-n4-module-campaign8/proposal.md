# Proposal: build-n4-module-campaign8

**Status**: proposal · **Phase**: propose (post-explore, pre-proposal handoff CONFIRMED)
**Source**: niagara-tools `v0.18.0` (main `48fb210`, bats 179/179) · **Target**: `v0.19.0` (MINOR — new toolbelt scripts + flags, CONTRIBUTING §4-5)
**Inputs**: `openspec/changes/build-n4-module-campaign8/explore.md` (gate-passed) · research B800 / B801 / B802 (landed), B775 §775.6 · B803 / B804 / B806+ in progress (gate the OUT items only)
**Topic key**: `sdd/build-n4-module-campaign8/proposal`
**Delivery**: auto-chain, two waves — wave 1 = 8 chained PRs (no pending input), wave 2 = O/P/Q/R appended after PR8 as their inputs land — fixed merge order, ~400 authored lines/PR

---

## 1. Intent

Campaign 7 shipped the scaffolder, the schema-risk classifier and the aggregate report; the operator's stations then failed five ways the kit still cannot see. The `ColdRoomPan` defrost interval died silently 5× on PANCCADIA because `BDefrostController.armTrigger` floors its delay at `0L` and `Clock.schedule` rejects a non-positive delay (`IllegalArgumentException: time <= 0`, `[CERT]` B801 §801.2, `[CERT-live]` §801.4). `chihuahua` drives its dashboard from a `ScheduledExecutorService` instead of `Clock` (`BChiDashboardService.java:305,456`), producing 21 `modifyThread failed to unschedule controlTick` warnings across MX60 and REFLOW. `BEvaporatorUnit` scheduled from `changed()` before steady state, throwing `NotRunningException` 6× per boot. `BCompressorControl` leaves `startingUp` set in `stopped()` (`:1799-1805`). And on 2026-09-03 17:04 the PANCCADIA station **never started** after a `ColdRoomPan-rt` redeploy — `Cannot load station: BRelTime cannot be cast to BComplex`, the exact B795 slot-retype OUTAGE class, live.

Every one of those is invisible to `lint-timers`, `slot-coverage`, `verify --src` and `report-module` today. Worse, the only evidence that any of them happened lives in a station console the kit never reads. This campaign closes that class: two new detectors from production shapes, three `lint-timers` rules, the console reader, the facet/lexicon/rc hygiene gaps, the eight-layer timer doctrine, and the promotion of `schema-risk.sh` from prose advice to a mandatory pre-deploy BUILD-LOOP step now that its failure mode is `[CERT-live]`.

The same evidence exposes a second blind spot: the loop **ends at deploy**. Link wiring, operator overrides, un-hidden actions and bog-vs-source schema drift only exist in the deployed station, and the kit has never read a `config.bog` or kept a deploy baseline. Wave 2 adds that half — a read-only station snapshot, an offline bog audit, the `wb` and `ux`-servlet checks, and a post-deploy checklist — as chained slices that append after wave 1 so the chain never waits on an input still in production.

---

## 2. Scope

### 2.1 In scope

| ID | Item | Why now |
|---|---|---|
| **A** | `toolbelt/lint-delays.sh` — static non-positive delay/period at `Clock.schedule*` call sites + slot-getter whose `@NiagaraProperty` facet `MIN=0` | The B801 bug class; QA RED `qa/c8-lint-delays` already pins 9 cases |
| **B** | `toolbelt/triage-console.sh` — own-frame exception groups (count/first/last), plus SEVERE `Cannot load station` and `[sys.xml]` lines naming our types **without** an own frame; Spanish level normalize (ADVERTENCIA→WARNING, GRAVE→SEVERE); `LC_ALL=C grep -a`; exits 0/1/3 | The kit reads no station evidence at all; B800 §800.5 is the row contract |
| **C+K+N** | `lint-timers.sh` extensions: companion flag set beside a schedule and never cleared in `stopped()` (C); `Executors.*` / `ScheduledExecutorService` / `new Thread` inside a `BComponent` (K); `Clock.schedule*` in a `changed()`/`started()` body without an `isRunning()`/steady-state guard (N) | Three confirmed production shapes, one script, one bats file |
| **D** | Facets presence in `verify-module.sh --src`: UNITS/PRECISION/RANGE absent on OPERATOR numeric/enum slots | `verify --src` catches raw-double `BFacets.make`, never a missing facet |
| **E** | `slot-coverage.sh` per-slot mode: `@NiagaraProperty(name)` vs lexicon key diff, stale-key detection, empty lexicon → FAIL (not WARN) | `chihuahua` ships an empty `module.lexicon` and passes today |
| **F** | `toolbelt/rc-scan.sh` — hardcoded ORD literals, `http://`/IPv4 hosts, SPA bare `catch(()=>{})`, null display branch under `rc/` | `DashboardReader.java:75` + `index.html`; no check reads `rc/` at all |
| **G** | Doctrine: eight-layer timer-driven state-machine defense in `types/logic.md`; `schema-risk.sh` becomes a **mandatory** BUILD-LOOP pre-deploy step; J client/cert-chain note; B802 inter-module line | The doctrine is the durable half of the fix |
| **H** | `report-module.sh` integration: `lint-delays` row, `triage-console --console-dir` row, explicit `schema-risk` row | The aggregate report is where an operator actually looks |

### 2.1b In scope — wave 2 (chained after PR8; never blocks wave 1)

| ID | Item | Why now | Pending input |
|---|---|---|---|
| **O** | `toolbelt/station-snapshot.sh` (read-only copy of `config.bog` + `console*.txt` + history/alarm db pointers from a station dir or a mounted station copy — **no live station required**) and `toolbelt/bog-audit.sh <config.bog> --module <MOD>`: offline analysis of instances of our types, every `BLink` into/out of them resolved by handle (dangling or misnamed slot → FAIL), action flags changed from the module defaults (`<a n="intervalExpired" f="o"/>` = un-hidden action → WARN), property values outside declared facets, properties present in the bog but absent from `module-include`/`@NiagaraProperty` and vice versa (schema drift, pairs with `schema-risk`), HOA/override leftovers, TRANSIENT values persisted | The deployed station is the only place where link wiring, operator overrides and schema drift are actually observable; nothing in the kit reads a bog | PANCCADIA `config.bog` contract shard **in production now** — treat as pending input, not as an assumption |
| **P** | `wb` artifacts audit → `wb` checks: palette/lexicon presence, agent registration (`@AgentOn`), `wb`-only APIs leaking into `rt`, empty `wb` scaffolds (e.g. `DashboardPan-wb`) | The `wb` half of every operator module is unchecked today | Audit shard running now |
| **Q** | `ux` servlet contract lint over `BWebServlet` subclasses + `rc-scan`: missing/invalid value → 400 and **never** a write, server-side facet enforcement, an auth check on **every** API branch, cache headers | The write surface is where an operator error becomes a station action; `BDashboardServlet.handleSetpointWrite` is the real shape | None (independent of B803/B804 step-up auth, which stays OUT) |
| **R** | Doctrine: post-deploy checklist in `BUILD-LOOP.md` §6 — *after the module loads: snapshot the station → `triage-console` → `bog-audit` → `report-module` → keep the snapshot as the deploy baseline* | The loop currently ends at deploy; every campaign-8 defect surfaced **after** it | Needs O to exist |

**Two waves.** Wave 1 = PR1-PR8 (already planned, no pending input). Wave 2 = O/P/Q/R plus the research doc PRs I/L/M **if** their blocks land. The chain never waits on research or on a shard: wave 2 slices append after PR8 in delivery order as their inputs arrive, and any slice whose input has not landed by close rolls to campaign 9 without holding the version bump.

### 2.2 Judged OUT / research-gated (with reason)

| Item | Reason |
|---|---|
| **I** research folds (history-ext, further B800 census) | Blocks still in flight; lands as its own doc PR appended at the end of the chain if delivered before close |
| **L** full station/JACE load doctrine | Needs B806+; only note-only findings exist. `station-load.sh` probe is out of campaign entirely |
| **M** critical-write step-up auth (`types/dashboard.md` + contract RED) | Gated on B803/B804; appended as its own doc PR if the blocks land before close |
| Fixes inside client module repos (DashboardPan gate 4, ColdRoomPan production fix) | Separate repos; `niagara-panccadia-leon` PR #1 already carries the ColdRoomPan fix. The kit cites, never edits |
| **#50** station-required checks | Still no station in CI/WSL — a fake PASS. Exception: **B795-G1 is now closed** by the 2026-09-03 `[CERT-live]` OUTAGE, so the schema-risk mandate lands on evidence, not a live probe |

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. Per the campaign-6/7 convention, `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- `module-delay-floor-lint`: `lint-delays.sh` contract — FAIL on a `Clock.schedule*` delay/period with no visible `>0` floor (`Math.max(x,0L)`, literal `BRelTime.make(0)`), PASS on a `>=1` literal floor, WARN when the floor lives in a slot facet with `MIN>=1`, FAIL when facet `MIN=0`; rows `FAIL|WARN <file>:<line> <reason>`; exits 0/1/3 (K20 disjoint ranges).
- `station-console-triage`: `triage-console.sh` contract — own-package frame attribution, digit-normalized grouping with count/first/last, stack-less own-logger warning rows, mandatory SEVERE `Cannot load station` + `[sys.xml]` type-named rows without an own frame, Spanish level normalization, byte-safe (`LC_ALL=C grep -a`) reading, exits 0/1/3.
- `module-rc-scan`: `rc-scan.sh` contract — ORD/host literal detection, bare-catch and null-display-branch detection under `rc/`, PASS/FAIL/WARN rows, exits 0/1/3.
- `module-timer-conformance-lint`: formalizes `lint-timers.sh` (undocumented since its retro code-fold) plus the three new rules C/K/N and the doctrine exception list.
- `station-snapshot` *(wave 2, O)*: `station-snapshot.sh` contract — read-only capture set (`config.bog`, `console*.txt`, history/alarm db pointers), source = station dir or mounted copy, never a live write, deterministic snapshot layout, exits 0/1/3.
- `station-bog-audit` *(wave 2, O)*: `bog-audit.sh` contract — per-module instance enumeration, handle-resolved `BLink` integrity (dangling/misnamed slot → FAIL), action-flag drift from module defaults (→ WARN), facet-range violations, bog↔`module-include`/`@NiagaraProperty` schema drift in both directions, HOA/override leftovers, persisted TRANSIENT values; verdict rows compatible with `report-module.sh`; exits 0/1/3.
- `module-wb-conformance` *(wave 2, P)*: palette/lexicon presence, `@AgentOn` registration, `wb`-only API leakage into `rt`, empty-`wb`-scaffold detection.
- `module-ux-servlet-contract` *(wave 2, Q)*: `BWebServlet` subclass lint — invalid/missing value → 400 with no write, server-side facet enforcement, auth on every API branch, cache headers.

### Modified Capabilities

- `module-verify-gate`: `--src` gains OPERATOR-slot facet-presence checking (D).
- `kit-preflight-and-coverage`: `slot-coverage.sh` gains per-slot diff + stale keys; empty lexicon escalates WARN → FAIL (E).
- `kit-module-report`: three new member rows (`lint-delays`, `triage-console`, `schema-risk`); a member FAIL must surface as an aggregate FAIL (H).
- `build-n4-module-kit-doctrine`: `types/logic.md` gains the eight-layer timer SM doctrine; BUILD-LOOP makes `schema-risk.sh` a mandatory pre-deploy step (exit 2 = OUTAGE blocks deploy); cert-chain and client-fix notes; B802 inter-module line; **wave 2 (R)** adds the BUILD-LOOP §6 post-deploy checklist (snapshot → triage-console → bog-audit → report-module → keep the snapshot as the deploy baseline).

---

## 4. Approach — two waves of chained PRs, RED first

Each PR is one destination-file group; PR1 targets `main`, each later PR branches only after its predecessor merges. Every code PR merges its QA RED branch first and goes green against it (CONTRIBUTING §2). Doc-only PRs carry zero tests by design. Workers write only inside their worktree (K12); commit bodies are grepped for attribution trailers before publishing (K11); QA REDs are cited by branch name with the tip re-read at apply time (K13).

| # | Branch / work unit | Files | Est. authored | Named mutation | QA RED |
|---|---|---|---|---|---|
| **PR1** | `feat/c8-lint-delays` (A) | `toolbelt/lint-delays.sh`, `tests/lint-delays.bats`, `tests/fixtures/lint-delays/**` | ~200 | Accept `Math.max(x,0L)` as a valid floor → LD1/LD3 stop failing | `qa/c8-lint-delays` (9 pins) |
| **PR2** | `feat/c8-triage-console` (B) | `toolbelt/triage-console.sh`, `tests/triage-console.bats`, fixtures | ~150 | Drop the own-frame filter → the jetty-only NPE row appears (TR1 flips) | `qa/c8-triage-console` (6 pins + SEVERE expansion) |
| **PR3** | `feat/c8-lint-timers-ext` (C+K+N) | `toolbelt/lint-timers.sh`, `tests/lint-timers.bats` | ~160 | Remove `startingUp=false` from `stopped()`; swap `Clock.schedulePeriodically` for `Executors.new*`; drop the `isRunning()` guard from `changed()` — each must FAIL | needs RED |
| **PR4** | `feat/c8-facets-lint` (D) | `toolbelt/verify-module.sh`, `tests/verify-module.bats` | ~100 | Strip the facet annotation from an OPERATOR double → FAIL | needs RED |
| **PR5** | `feat/c8-slot-per-slot` (E) | `toolbelt/slot-coverage.sh`, `tests/slot-coverage.bats` | ~160 | Drop one slot key from the lexicon → FAIL; empty lexicon → FAIL not WARN | needs RED |
| **PR6** | `feat/c8-rc-scan` (F) | `toolbelt/rc-scan.sh`, `tests/rc-scan.bats`, fixtures | ~200 | Remove the ORD constant / restore the bare catch → FAIL | needs RED |
| **PR7** | `docs/c8-doctrine` (G) | `types/logic.md`, `BUILD-LOOP.md`, `METHODOLOGY.md` | ~70 | None (doc-only). Guard: `kit-links.bats` green, `sweep-fold-audit.sh --strict` green, every line cites `[ev: corpus B…]` | none |
| **PR8** | `feat/c8-report-integration` (H) | `toolbelt/report-module.sh`, `tests/report-module.bats` | ~60 | Drop aggregation of the new rows → report says PASS while a member FAILs | needs RED |
| **PR9** | `feat/c8-station-snapshot` (O.1) | `toolbelt/station-snapshot.sh`, `tests/station-snapshot.bats` | ~150 | Make the snapshot writable/mutating → the read-only assertion FAILs | needs RED |
| **PR10** | `feat/c8-bog-audit` (O.2) | `toolbelt/bog-audit.sh`, `tests/bog-audit.bats`, bog fixtures | ~250 | Skip handle resolution → the dangling-link fixture stops FAILing | needs RED (gated on the PANCCADIA bog contract shard) |
| **PR11** | `feat/c8-wb-audit` (P) | `toolbelt/verify-module.sh` or `wb` check + tests | ~150 | Empty `wb` scaffold stops being detected | needs RED (gated on the `wb` audit shard) |
| **PR12** | `feat/c8-ux-servlet` (Q) | `toolbelt/rc-scan.sh` + servlet lint, tests | ~180 | Remove the auth check from one API branch → must FAIL | needs RED |
| **PR13** | `docs/c8-post-deploy` (R) | `BUILD-LOOP.md` §6 | ~30 | None (doc-only). Guard: `kit-links.bats` names every step's script | none |

PR9-PR13 are **wave 2**: they append after PR8 in that order, each landing when its pending input (bog contract shard, `wb` audit shard) arrives. Research doc PRs I/L/M append last if B803/B804/B806+ deliver. Wave 1 never waits on any of them.

**Method invariants.** (1) Every new rule is proven RED against a **real-shape fixture copied from the operator's modules with sanitized names**, never an invented snippet. (2) Where a rule has an enumerable domain, it is **table-driven from an oracle file** and the heredoc-vs-oracle byte equality is itself a named mutation (the `schema-risk` precedent, CONTRIBUTING §2). (3) `lint-delays.sh` and `triage-console.sh` emit `PASS|FAIL|WARN|SKIP` rows in the canonical format `report-module.sh` parses, agreed in PR1/PR2 and consumed in PR8. (4) K19: every new script is named in **both** `BUILD-LOOP.md` and `skill/SKILL.md` in the PR that lands it. (5) Each kit-changing push pairs its retro + `retros/INDEX.md` row + `BUILD-STATE.md` self-envelope in the same push range.

**Version**: `VERSION` + `CHANGELOG.md` → `0.19.0` in the last merged **wave-1** PR, so a delayed wave-2 slice never holds the release; wave-2 slices land as further MINOR entries. **Gate exits**: every PR here is a kit-changing push → close-gate exit **(a) NEW RETRO**.

---

## 5. Affected Areas

| Area | Impact | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/{lint-delays,triage-console,rc-scan}.sh` | New | Delay-floor lint, console triage, `rc/` scan |
| `build-n4-module-kit/toolbelt/{station-snapshot,bog-audit}.sh` | New (wave 2) | Read-only station capture and offline `config.bog` audit |
| `build-n4-module-kit/toolbelt/{lint-timers,verify-module,slot-coverage,report-module,rc-scan}.sh` | Modified | C/K/N rules, `--src` facets, per-slot mode, new report rows, `wb` checks (P) and the servlet contract lint (Q) |
| `build-n4-module-kit/types/logic.md`, `BUILD-LOOP.md`, `METHODOLOGY.md` | Modified | Eight-layer doctrine, mandatory schema-risk step, K-rules from campaign-8 retros |
| `build-n4-module-kit/skill/SKILL.md` | Modified | K19 routing entries for every new script |
| `tests/*.bats` + `tests/fixtures/**` | New/Modified | ~30 new cases, each with a named mutation; real-shape sanitized fixtures |
| `build-n4-module-kit/retros/INDEX.md`, `BUILD-STATE.md` | Modified | One retro per kit-changing push; pending → 0 at close |
| `CONTRIBUTING.md`, `VERSION`, `CHANGELOG.md` | Modified | `0.18.0 → 0.19.0` |

---

## 6. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| R1 | A new check has no real-module smoke and only bites synthetic fixtures (campaign-7 meta-lesson 1) | High | Every code PR carries a real-module smoke: PR1 the ColdRoomPan pre-fix tree, PR2 the three real consoles, PR3 the `BEvaporatorUnit:519` and `BCompressorControl:1799-1805` shapes |
| R2 | **VERIFY-marked audit findings** (ColdRoomPan F4/F6, chihuahua #10, CompPan #3) become fixtures on a wrong shape and pass trivially | High | Two independent reads (CONTRIBUTING §9) before any VERIFY finding becomes a fixture; a finding that fails the second read is dropped, not softened |
| R3 | PR2 scope expansion changes the `qa/c8-triage-console` contract mid-flight | Med | QA extends the RED **first**; the worker re-reads the branch tip at apply time (K13) |
| R4 | The `--src` facet heuristic (D) needs the `@NiagaraProperty` facets **annotation string**, not the `BFacets.make` call site; valid `MIN=0` enum facets are false positives | Med | Prove the heuristic against real ColdRoomPan/CompPan source before writing tests; K2 — scoped to the reported defect, mutation-proven to bite only a real case |
| R5 | Row-contract mismatch between PR1/PR2 output and `report-module.sh` breaks H silently | Med | The row format is fixed in PR1 and asserted by PR8's integration RED, not by review opinion |
| R6 | The mandatory schema-risk step changes operator workflow with no code change | Low | Doc-only and flagged in the PR body; `schema-risk.sh` already exits 2 on OUTAGE |
| R7 | B803/B804/B806+ do not land before close | Med | They gate no chain PR. If they land, each appends its own doc PR at the end; if not, they roll to campaign 9 |
| R8 | Chain drift — a child PR diff shows its parent's commits | Med | Branch only after the parent merges; rebase until the child diff is clean |
| R9 | **Wave 2 pending inputs** (PANCCADIA bog contract shard, `wb` audit shard) do not arrive, or arrive with a different bog grammar than assumed | Med | Wave 2 is appended, never interleaved: PR9-PR13 hold no wave-1 slice and no version bump. `bog-audit.sh` parses only what the delivered shard documents; an undelivered shard rolls PR10 to campaign 9 |
| R10 | A snapshot copied from a live station directory races the running station or is mistaken for a station write | Med | `station-snapshot.sh` is read-only by construction (copy-out only, no station connection, no write path) and the read-only assertion is a named mutation; live-verify safety (METHODOLOGY) forbids any state-changing write during verification |
| R11 | `bog-audit` action-flag drift produces noise: a legitimately un-hidden action is flagged | Med | Flag drift is **WARN, never FAIL** (K3: a trace over a silent removal); only handle-unresolvable links FAIL |

---

## 7. Rollback Plan

| Slice | Rollback |
|---|---|
| PR1 / PR2 / PR6 | `git revert` — new script + tests + fixtures on new paths; nothing else references them until PR8 |
| PR3 / PR4 / PR5 / PR8 | `git revert` — the modified scripts lose only the new rules/rows; existing bats cases are untouched |
| PR7 / PR13 | `git revert` — doc-only; retro files are never deleted (propose-never-apply), INDEX rows revert to `pending` |
| PR9 / PR10 (wave 2) | `git revert` — new scripts, tests and bog fixtures on new paths; snapshots are produced outside the repo and are never committed |
| PR11 / PR12 (wave 2) | `git revert` — the modified scripts lose only the `wb` and servlet rules |

No station, no deployed jar and no operator data is touched by any slice. Rollback risk is limited to the developer workstation and CI.

---

## 8. Dependencies

- `bats-core`, `shellcheck 0.10.0` (pinned in `ci.yml`), live pre-push hook — already installed.
- Research landed: **B800** (console census + row contract), **B801** (`Clock` `>0` floor, `[CERT]` + `[CERT-live]`), **B802** (inter-module comms), **B775 §775.6**. In flight: B803/B804 (step-up auth), B806+ (station load) — OUT items only.
- Real trees for smokes: `~/modulos_niagara_n4/Cliente/Leon-Guanjuato/…/ColdRoomPan` pre-fix tree **and** branch `fix/defrost-overdue-delay`; PANCCADIA / REFLOW / MX60 consoles.
- QA RED branches merged into their PR branch before it goes green: `qa/c8-lint-delays`, `qa/c8-triage-console`, plus new REDs for PR3-PR6, PR8 and every wave-2 code slice.
- **Wave-2 pending inputs** (not assumptions): the PANCCADIA `config.bog` contract shard (gates PR10 and its fixtures) and the `wb` artifacts audit shard (gates PR11). Both are in production now; neither gates wave 1.

---

## 9. Research & product-decision record

| Doc line / rule | Cites |
|---|---|
| Eight-layer timer SM doctrine (`types/logic.md`) | `[ev: corpus B775 §775.6]` `[ev: corpus B801 §801.2-801.4]` |
| `lint-delays.sh` rule set + facet floor | `[ev: corpus B801 §801.2, §801.3]` |
| `triage-console.sh` row shape + SEVERE/`[sys.xml]` requirement | `[ev: corpus B800 §800.2, §800.5]` |
| JDK executor → `Clock` doctrine (K) | `[ev: corpus B800 §800.3]` |
| Mandatory `schema-risk.sh` BUILD-LOOP step | `[ev: corpus B795 §795.4]` + PANCCADIA 2026-09-03 `[CERT-live]` OUTAGE (closes B795-G1) |
| Inter-module comms line | `[ev: corpus B802 §802.1-802.5]` |
| `bog-audit.sh` rule set + BUILD-LOOP §6 post-deploy checklist (O, R) | PANCCADIA `config.bog` contract shard (pending) + the 2026-09-03 `[CERT-live]` OUTAGE + `[ev: corpus B800 §800.4]` boot/reload history |
| `wb` checks (P) / servlet contract lint (Q) | `wb` audit shard (pending) / `BDashboardServlet.handleSetpointWrite:239-274` and `BChiServlet` real shapes |

Product decisions are the operator's standing mandate: improve the kit from real bugs and audits, tests that bite (no padding), chained PRs, doc-only PRs with zero tests, workers in worktrees, no AI attribution trailers.

---

## 10. Success Criteria

- [ ] `lint-delays.sh` **FAILs** the pre-fix ColdRoomPan tree naming `BDefrostController`, and **PASSes** the fixed tree on branch `fix/defrost-overdue-delay` of `~/modulos_niagara_n4/Cliente/Leon-Guanjuato`; all 9 `qa/c8-lint-delays` pins green.
- [ ] `triage-console.sh` reproduces the B800 census rows on the real PANCCADIA / REFLOW / MX60 consoles, **including** the fatal SEVERE `Cannot load station` row, and survives `LC_ALL=C` over mojibake bytes.
- [ ] `lint-timers.sh` FAILs `chihuahua` `BChiDashboardService` (executor), CompPan `BCompressorControl` (`startingUp` uncleared) and the pre-fix `BEvaporatorUnit` `changed()` shape.
- [ ] Every code PR records a **named mutation** and a **real-module smoke**; the bats total grows only by biting tests (no padding cases).
- [ ] `report-module.sh` surfaces a member FAIL from `lint-delays`, `triage-console` and `schema-risk` as an aggregate FAIL.
- [ ] `BUILD-LOOP.md` states `schema-risk.sh` as a mandatory pre-deploy step; `kit-links.bats` L4/L5 green with every toolbelt script — including the three new ones — routed in both `BUILD-LOOP.md` and `skill/SKILL.md` (K19).
- [ ] *(wave 2, O)* `bog-audit.sh` **FAILs** a fixture whose `BLink` handle resolves to a missing/renamed slot, and **WARNs** on the real PANCCADIA `config.bog` for the un-hidden `intervalExpired` action (`<a n="intervalExpired" f="o"/>`), with no FAIL on that same file for flag drift alone.
- [ ] *(wave 2, O)* `station-snapshot.sh` produces a complete capture set from a mounted station copy with **zero writes** to the source, proven under a read-only source directory.
- [ ] *(wave 2, P/Q/R)* the `wb` checks FAIL an empty `wb` scaffold; the servlet lint FAILs an API branch without an auth check and a write reached with an invalid value; `BUILD-LOOP.md` §6 names each post-deploy step's script.
- [ ] `retros/INDEX.md` folded to **pending = 0** at close; `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` green.
- [ ] `VERSION` = `0.19.0` with a `CHANGELOG.md` entry per CONTRIBUTING §4-5; `shellcheck` exit 0; no commit body carries an attribution trailer.
- [ ] Each PR merges in fixed order with a clean child diff.

---

## 11. Review Workload Note

`delivery_strategy` = **auto-chain**.
`Decision needed before apply: No`
`Chained PRs recommended: Yes` — wave 1 = 8 slices in fixed merge order; wave 2 = up to 5 more appended after PR8 as their inputs land
`400-line budget risk: Low` for wave 1 (every slice ≤200 authored lines) and **Medium** for PR10 (~250, bog fixtures as their own commit); no `size:exception` is expected.

---

## 12. Next Phases

- `sdd-spec` — wave 1 first: formalize the `lint-delays` verdict domain and facet rule, the `triage-console` row grammar and attribution rule, the three `lint-timers` predicates, the facet-presence and per-slot coverage rules, the `rc-scan` patterns, and the aggregate-report row contract. Wave 2 (snapshot capture set, bog-audit verdict domain, `wb` and servlet rules) is specced when its pending shards land — the wave-1 spec must not block on them.
- `sdd-design` — fixture layout and sanitization for real-shape sources, the oracle-file/table-driven mechanics, the shared row format between PR1/PR2 and PR8, the doc placement with `[ev:]` citations, the read-only snapshot mechanics and bog-fixture sanitization, and the two-wave chain mechanics. Can run in parallel with `sdd-spec`.
- Then `sdd-tasks` → `sdd-apply` (strict TDD on every code slice) → `sdd-verify` → `sdd-archive`.
