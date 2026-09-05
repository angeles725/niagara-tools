# Spec: build-n4-module-campaign8

**Status**: spec · **Source**: v0.18.0 (48fb210) · **Target**: v0.19.0
**Topic key**: `sdd/build-n4-module-campaign8/spec`
**Based on**: proposal.md + explore.md + B800/B801/B802/B803 + audit evidence (2026-09-05)
**Row format** (all new scripts): `PASS|FAIL|WARN|SKIP  <check>  <file>:<line>  <detail>`
**Exits** (new standalone scripts): `0` no FAIL · `1` any FAIL · `3` env/usage (K20 disjoint ranges)

---

## Cross-cutting discipline (all PRs)

| Rule | Requirement |
|------|-------------|
| CD1 | Every kit-changing push range: retro file + INDEX row flip + `BUILD-STATE.md` self-envelope in the same push range. |
| CD2 | Doc-only PRs (PR7, PR13) carry zero new bats tests. |
| CD3 | QA RED branch tip re-read at apply time before merging (K13). |
| CD4 | Every new rule proven RED against a real-shape fixture copied from the operator's modules with sanitized names (CONTRIBUTING §9). |
| CD5 | Every new toolbelt script named in both `BUILD-LOOP.md` and `skill/SKILL.md` in the PR that lands it (K19). |
| CD6 | `shellcheck 0.10.0` exits 0 on every modified or new `toolbelt/*.sh`. |
| CD7 | No commit body carries an attribution trailer (K11). |
| CD8 | Each child PR diff shows only child commits; rebase until clean. |

---

## PR1 — lint-delays.sh (A)

**Branch**: `feat/c8-lint-delays` | **QA RED**: `qa/c8-lint-delays` (LD1–LD9, 9 pins)

| ID | Requirement |
|----|-------------|
| R1.1 | MUST emit `FAIL` for every `Clock.schedule*`/`schedulePeriodically` call site that passes a delay/period with no `>0` floor (`Math.max(x, k)` with `k>0`, or a literal `≥1 ms`). |
| R1.2 | `Math.max(x, positive)` and literal `≥1 ms` floors MUST produce PASS (no FAIL row). |
| R1.3 | A slot-getter whose `@NiagaraProperty` facet has `MIN≥1` MUST produce WARN; `MIN=0` MUST produce FAIL. |
| R1.4 | Row: `FAIL  lint-delays  <file>:<line>  <reason>`; WARN same columns. |
| R1.5 | Named mutation: accepting `Math.max(x,0L)` as valid causes LD1/LD3 to stop FAILing. |

**LD-FAIL** GIVEN BDefrostController.java pre-fix tree 4f5f1c7 (getInterval() raw, facet MIN=0, no max floor); WHEN `lint-delays.sh` runs; THEN exits 1, FAIL rows at `:566`, `:622`, `:664`.

**LD-PASS** GIVEN ColdRoomPan fixed tree 509bef2 (floors ≥1 ms applied); WHEN `lint-delays.sh` runs; THEN exits 0, no FAIL rows.

**LD-WARN** GIVEN slot-getter with `@NiagaraProperty(min=1)` facet; WHEN `lint-delays.sh` runs; THEN WARN emitted, exit 0.

---

## PR2 — triage-console.sh (B)

**Branch**: `feat/c8-triage-console` | **QA RED**: `qa/c8-triage-console` (TR1–TR6 + SEVERE expansion TR7–TR8)

| ID | Requirement |
|----|-------------|
| R2.1 | MUST group exceptions by `(exception_class, normalized_message, own_frame)`; each group emits exactly one row with count, first timestamp, last timestamp. |
| R2.2 | MUST attribute via BOTH channels: `com.angeles.*` stack frame AND module logger tag `[coldRoomPan|dashboardpan|chihuahua]`; a frame-only tool MUST NOT be accepted (B800 §800.1). |
| R2.3 | SEVERE `Cannot load station` lines naming a `com.angeles.*` type AND `[sys.xml]` entries naming our types MUST produce dedicated FAIL rows even without an own frame (third attribution channel). |
| R2.4 | Spanish levels MUST be normalized before grouping: ADVERTENCIA→WARNING, GRAVE→SEVERE, INFORMACIÓN→INFO. |
| R2.5 | MUST read console files with `LC_ALL=C grep -a`; MUST NOT fail on mojibake bytes. |
| R2.6 | `triage-console.sh` script header and `METHODOLOGY.md` MUST document the triple-attribution contract and locale/encoding requirement, each carrying `[ev: corpus B800]` (D7). |
| R2.7 | Named mutation: dropping own-frame filter causes jetty-only NPE row to appear (TR1 flips). |

**TR-DUAL** GIVEN MX60 console (chihuahua `modifyThread` lines, no `com.angeles` frame, only `[chihuahua]` tag); WHEN `triage-console.sh` runs; THEN 9 findings attributed via logger-tag, exit 1.

**TR-SEVERE** GIVEN PANCCADIA console with `Cannot load station: BRelTime cannot be cast to BComplex`; WHEN `triage-console.sh` runs; THEN one SEVERE FAIL row emitted, exit 1.

**TR-ENCODING** GIVEN HoneywellMX console with mojibake Spanish bytes; WHEN `triage-console.sh` runs; THEN completes without error; ADVERTENCIA rows normalized to WARNING.

---

## PR3 — lint-timers ext (C+K+N)

**Branch**: `feat/c8-lint-timers-ext` | **QA RED**: needs RED (TC-A / TC-B / TC-C)

| ID | Requirement |
|----|-------------|
| R3.C | A companion flag set beside a `Clock.schedule*` call that is not cleared in `stopped()` MUST produce FAIL. |
| R3.K | `Executors.*` / `ScheduledExecutorService` / `new Thread(…)` inside a `BComponent` subclass MUST produce FAIL. |
| R3.N | `Clock.schedule*` inside a `changed()` or `started()` body without an `isRunning()` guard MUST produce FAIL. |
| R3.4 | Rows MUST follow existing `lint-timers.sh` format: `PASS|FAIL  lint-timers  <file>: <detail>`; exits 0/1/2(usage)/3(env). |

**TC-A** GIVEN BCompressorControl.java `:1799–1805` (`startingUp` set near schedule, not cleared in `stopped()`); WHEN `lint-timers.sh` runs; THEN FAIL on companion-flag check, exit 1.

**TC-B** GIVEN BChiDashboardService.java`:305` (`ScheduledExecutorService` in BComponent); WHEN `lint-timers.sh` runs; THEN FAIL on JDK-executor check, exit 1.

**TC-C** GIVEN BEvaporatorUnit.java`:519` pre-fix (`Clock.schedule` in `changed()` without `isRunning()` guard); WHEN `lint-timers.sh` runs; THEN FAIL on changed()-guard check, exit 1.

---

## PR4 — facets lint --src (D)

**Branch**: `feat/c8-facets-lint` | **QA RED**: needs RED

| ID | Requirement |
|----|-------------|
| R4.1 | `verify-module.sh --src` MUST emit FAIL when an OPERATOR numeric or enum slot lacks UNITS, PRECISION, or RANGE facets in its `@NiagaraProperty` annotation. |
| R4.2 | A valid `MIN=0` on an enum slot MUST NOT be a false FAIL. |
| R4.3 | Named mutation: stripping facet annotation from one OPERATOR double causes FAIL. |

**FacetFAIL** GIVEN CompPan source with 12 OPERATOR doubles without MIN/MAX facets; WHEN `verify-module.sh --src` runs; THEN FAIL rows emitted for missing facets.

**FacetFalsePos** GIVEN enum slot with `@NiagaraProperty(min=0)`; WHEN `verify-module.sh --src` runs; THEN no FAIL row for that slot.

---

## PR5 — slot-coverage per-slot + stale + empty-lexicon FAIL (E)

**Branch**: `feat/c8-slot-per-slot` | **QA RED**: needs RED

| ID | Requirement |
|----|-------------|
| R5.1 | `slot-coverage.sh` MUST support per-slot mode: compare `@NiagaraProperty(name)` values against lexicon keys and report the diff. |
| R5.2 | A key present in the lexicon with no matching `@NiagaraProperty` MUST be reported as a stale key. |
| R5.3 | An empty `module.lexicon` when `module-include.xml` declares at least one type MUST exit 1 (FAIL), not 0 (WARN). |

**SlotDiff** GIVEN ColdRoomPan with 19 missing per-slot lexicon keys; WHEN per-slot mode runs; THEN 19 missing slots listed, exit 1.

**EmptyLex** GIVEN chihuahua `module.lexicon` (empty) with types in module-include.xml; WHEN `slot-coverage.sh` runs; THEN exits 1.

**StaleKey** GIVEN lexicon key with no matching `@NiagaraProperty`; WHEN per-slot mode runs; THEN stale-key row emitted.

---

## PR6 — rc-scan.sh (F)

**Branch**: `feat/c8-rc-scan` | **QA RED**: needs RED

| ID | Requirement |
|----|-------------|
| R6.1 | Hardcoded ORD string literals under `rc/` MUST produce FAIL rows. |
| R6.2 | `http://` or IPv4 host literals under `rc/` MUST produce FAIL rows. |
| R6.3 | Bare `catch(()=>{})` patterns MUST produce WARN rows (not FAIL). |
| R6.4 | Null display branches (missing null-check before DOM write) MUST produce WARN rows. |
| R6.5 | Row: `FAIL|WARN  rc-scan  <file>:<line>  <reason>`; exits 0/1/3. |
| R6.6 | Named mutation: removing the ORD constant / restoring bare catch causes FAIL/WARN flip. |

**ORDFAIL** GIVEN DashboardReader.java`:75` `SERVICE_ORD` constant; WHEN `rc-scan.sh` runs; THEN FAIL row, exit 1.

**BareCatch** GIVEN `index.html` with `catch(()=>{})` pattern; WHEN `rc-scan.sh` runs; THEN WARN row, exit 0.

---

## PR7 — Doctrine (G + J + M)

**Branch**: `docs/c8-doctrine` | **Files**: `types/logic.md`, `types/dashboard.md`, `BUILD-LOOP.md`, `METHODOLOGY.md` | **Type**: doc-only

| ID | Requirement |
|----|-------------|
| R7.1 | `types/logic.md` MUST gain the 8-layer timer SM defense section `[ev: corpus B775 §775.6, B801 §801.2–801.4]`: (1) slot facet MIN≥1, (2) call-site `Math.max` floor, (3) fail-loud status/lastError, (4) self-healing monitor tick, (5) single idempotent ensureArmed, (6) nextFire+overdue TRANSIENT exposure, (7) OPERATOR force action, (8) kit lints + console triage. |
| R7.2 | `BUILD-LOOP.md` MUST state `schema-risk.sh` as a mandatory pre-deploy step; exit 2 (OUTAGE) MUST block deploy `[ev: corpus B795 §795.4, 2026-09-03 CERT-live OUTAGE]`. |
| R7.3 | `types/logic-authoring.md` MUST gain a "talking to another module" section `[ev: corpus B802]`: within a station, `BLink`/service-discovery/`Subscriber` are module-agnostic; the only real boundaries are the compile-time `<dependency>` and the `fox:` remote hop; anti-pattern: bespoke inter-module bus. Requires-execution gap B802-G1 cited as OPEN (not folded as closed). |
| R7.4 | `types/dashboard.md` MUST gain a "critical-write step-up auth" section `[ev: corpus B803 §803.6]`: a critical endpoint MUST require a fresh re-auth token bound to session+user+target ORD (short-TTL); CSRF MUST use CsrfUtil `x-niagara-csrfToken`/`csrfToken`, not only `X-Requested-With` (updates DWS1 gate 2); SAML mid-session re-verify limitation documented as a constraint. Requires-execution gaps B803-G1/G2 cited as OPEN. |
| R7.5 | `types/dashboard.md` MUST carry client-fix guidance for issue #49 `[ev: corpus B803]`. |
| R7.6 | `toolbelt/verify-module.sh` MUST add a cert-chain trust note: `META-INF/NIAGARA4.SF` presence does NOT imply the signing cert is trusted by the station — a mismatched cert loads as UNSIGNED `[ev: corpus B800 §800.8, CERT-live REFLOW]`. |
| R7.7 | `METHODOLOGY.md` MUST note that decompiled line numbers are build-specific; `[ev:]` cites MUST name the build string `[ev: corpus B801 §801.4]`. |
| R7.8 | All doc delta lines carry `[ev: corpus B…]` tokens; `sweep-fold-audit.sh --strict` exits 0; `kit-links.bats` exits 0. |
| R7.9 | `skill/SKILL.md` and `METHODOLOGY.md` §0 MUST gain an "Excavador Técnico" working-profile section binding each trait to an enforceable kit mechanism: first-principles ↔ `[CERT]` cites to Tridium source before any rule; rigor ↔ RED-first + named mutation + real-module smoke + console triage after deploy; systems-thinking ↔ schema-risk/bog-audit/report-module trace from slot to station. Real motivating case: `Math.max(delayMs, 0L)` "fix" that left `time<=0` alive because first principles were skipped `[ev: corpus B801]`. After `skill/SKILL.md` changes, orchestrator re-installs via `scripts/install-skill.sh`. |
| R7.10 *(L, B806 landed)* | `METHODOLOGY.md` MUST gain a "Station load budget" section: the B806 resource-budget table (`| Dimension | Budget/limit | Cite |`: engine = 1 thread / 20 ms tick, engine watchdog 3 min, no per-callback slow threshold (open), Clock delay floor > 0, timer queues unbounded, fox circuit pool 2, persisted String slot = full config.bog re-save per write, audit ring ~100 KB NOT transient, auto-save 24 h, typical JACE capacity 25 dev / 500 pt / 400 link / 125 hist with > 100 % warn and > 110 % no boot, browser poll 5 s without backoff, JDK executors forbidden) plus the §806.9 per-module viability checklist (8 counts) — every line `[ev: corpus B806]`; open items stay marked open. |

---

## PR8 — report-module integration (H)

**Branch**: `feat/c8-report-integration` | **QA RED**: needs RED

| ID | Requirement |
|----|-------------|
| R8.1 | `report-module.sh` MUST invoke `lint-delays.sh` per artifact; FAIL rows from lint-delays MUST surface as aggregate FAIL. |
| R8.2 | MUST accept `--console-dir <dir>` and invoke `triage-console.sh`; its FAIL MUST surface as aggregate FAIL. |
| R8.3 | MUST invoke `schema-risk.sh` and emit an explicit `schema-risk` row; exit 2 (OUTAGE) MUST surface as aggregate FAIL. |
| R8.4 | Row format and exit codes unchanged (`<artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>`; exits 0/1/3). |
| R8.5 | Named mutation: dropping lint-delays aggregation causes report to exit 0 on a delay-FAIL module. |

**AggFAIL** GIVEN ColdRoomPan-rt pre-fix tree 4f5f1c7 (lint-delays FAIL on BDefrostController); WHEN `report-module.sh` runs; THEN aggregate exits 1, FAIL row for `lint-delays`.

---

## Wave 2 — PR9–PR13

### PR9 — station-snapshot.sh (O.1)

| R9.1 | MUST produce read-only capture set (`config.bog`, `console*.txt`, history/alarm db pointers) from a station dir or mounted copy; MUST NOT write to source; read-only assertion is the named mutation; exits 0/1/3. |

### PR10 — bog-audit.sh (O.2) — contract-gated (PANCCADIA bog shard pending)

| R10.1 | Handle-resolved `BLink` with unresolvable or misnamed target slot MUST FAIL. |
| R10.2 | Action-flag drift from module defaults MUST WARN (never FAIL alone). HOA/override leftovers and persisted TRANSIENT values MUST WARN. |
| R10.3 | Fixture shapes bind when the PANCCADIA `config.bog` contract shard lands; do not invent. |

### PR11 — wb checks (P) — NOW CONCRETE (wb shard landed)

| ID | Requirement |
|----|-------------|
| WB-LEX1 | `slot-coverage.sh` MUST run on every `-wb` `module-include.xml` + `module.lexicon`; missing lexicon file with declared types MUST FAIL (real fixture: chihuahua-wb, no lexicon, 1 declared type). |
| WB-SCAFFOLD1 | `verify-module.sh`: a `-wb` jar with 0 `.class` entries AND 0 palette entries MUST WARN; `--strict` MUST FAIL (real fixture: DashboardPan-wb; closes dead angle at `verify-module.sh:245–257`). |
| WB-THREAD1 | `lint-wb-threading.sh`: `doInvoke` bodies calling `getNavChildren`/`getNavNodes`/BQL without `invokeLater`/`BJobService` MUST FAIL (real fixture: `BBatchLinkEditor.java:684–720`). |
| WB-AGENT1 | `@AgentOn(types="baja:Component")` without a justification comment MUST WARN (real fixture: `BBatchLinkEditor.java:66–67`). |
| WB-DEP1 | `verify --src`: built `module.xml` `<dependency>` not declared as `api()`/`nre()` in profile `gradle.kts` MUST WARN (real fixture: chihuahua-wb schedule-rt phantom dep). |
| WB-DOC | `types/wb-widgets.md` MUST gain the 10-line good-`-wb`-artifact doctrine sketch; chihuahua-wb `model/` package cited as DWB1 exemplar. |

**WB-LEX1-Scenario** GIVEN chihuahua-wb with no `module.lexicon` and 1 type in module-include.xml; WHEN slot-coverage runs; THEN exits 1 (FAIL).

**WB-THREAD1-Scenario** GIVEN `BBatchLinkEditor.java:684` DFS inside `doInvoke`; WHEN `lint-wb-threading.sh` runs; THEN FAIL row, exit 1.

### PR12 — ux servlet lint (Q) — doc requirement unblocked (B803); fixture shapes contract-gated

| R12.1 | `BWebServlet` API branch with no auth check MUST FAIL. |
| R12.2 | A critical write endpoint reachable without a fresh re-auth token MUST FAIL `[ev: corpus B803 §803.6]`; token MUST be bound to session+user+target ORD, short-TTL. |
| R12.3 | CSRF MUST be validated with CsrfUtil (`x-niagara-csrfToken`/`csrfToken` param); `X-Requested-With`-only MUST FAIL. |
| R12.4 | Missing/invalid value reaching a write MUST produce 400 with no write MUST FAIL. |
| R12.5 | Missing cache headers MUST produce WARN. |

**StepUpAuth** GIVEN a critical endpoint called without a fresh re-auth token; WHEN the servlet lint runs; THEN FAIL, no component write.

**CsrfOnly** GIVEN a branch validating only `X-Requested-With`, no CsrfUtil; WHEN the servlet lint runs; THEN FAIL.

### PR13 — post-deploy checklist (R)

| R13.1 | `BUILD-LOOP.md` §6 MUST list: snapshot → triage-console → bog-audit → report-module → keep snapshot as deploy baseline. |
| R13.2 | `kit-links.bats` MUST name each step's script; all links resolve after PR13 merges. |

### PR14 — module versioning + build pipeline doctrine (S) — research-gated (block TBD)

| R14.1 | `BUILD-LOOP.md` §4 MUST gain a Gradle task matrix covering: `clean`, `slotomatic`, `compileJava`, `jar`, `build`, `moduleTestJar`, `niagaraTest`, `bajadoc` — what each touches, when required, safe combinations, and the mirror-niagara-home.sh trick for station locks. |
| R14.2 | `BUILD-LOOP.md` MUST document the version-bump checklist: `vendorVersion` vs `bajaVersion`, what the station does on reload, when a restart is mandatory, and `schema-risk.sh` MUST be run before any bump that touches slots. |
| R14.3 | `build.sh` MUST print the `mirror-niagara-home.sh` recipe to stderr on exit 31 (small code change; verified today on the fix-defrost-overdue worktree). |
| R14.4 | Doc requirement is blocked on the block number being assigned; specced at this level; fixtures and RED bind when the block lands. |

### PR15 — RT control-logic doctrine (T) — research-gated (block TBD; B805 in flight)

| R15.1 | `types/logic.md` MUST gain an RT control-logic section covering: PID/loop anti-windup, latches/SR, hysteresis, protection-layer ownership (who raises, who watches), the `execute()`/`changed()` split rule, and a flowchart template `[ev: corpus B805 when it lands]`. |
| R15.2 | `types/logic.md` MUST gain a "health/feedback surface" checklist for rt components: required `status`/`lastError`/`health` slots, how each is set, and the rule that a silent FAIL with no status update is forbidden `[ev: companero's block, TBD]`. |
| R15.3 | `types/logic-authoring.md` MUST gain a "logging a point to history" section: `BHistoryExt` as a point extension, Interval vs COV selection, `BHistoryConfig` capacity + fullPolicy (`roll`/`stop`), one ext per logged slot `[ev: corpus B804]`. Requires-execution gap B804-G1 cited as OPEN. |
| R15.4 | Doc requirement blocked on B805 (R15.1/R15.2) and B804 (R15.3) landing; specced at this level; bind when blocks land. |

---

## Success Criteria

| ID | Assertion |
|----|-----------|
| SC1 | `lint-delays.sh` exits 1 on pre-fix ColdRoomPan tree `4f5f1c7` (BDefrostController.java FAIL rows exactly at `:556` (Math.max(delayMs, 0L)), `:566`, `:620`, `:664` (raw slot getters whose facets declare MIN = 0 s); `:622`/`:641` schedule the same-file constant `POLL = BRelTime.make(5000)` and MUST NOT appear — same-file `static final BRelTime` constants are resolved); exits 0 on the fixed tree (client repo main `c66e412`, or `509bef2`); LD1–LD9 green. |
| SC2 | `triage-console.sh` reproduces B800 §800.2 exception rows on PANCCADIA/REFLOW/MX60 consoles including the fatal SEVERE `Cannot load station` row; TR1–TR8 green. |
| SC3 | `lint-timers.sh` emits FAIL for BChiDashboardService (TC-B), BCompressorControl`:1799–1805` (TC-A), BEvaporatorUnit`:519` pre-fix (TC-C); TC-A/B/C bats pins green. |
| SC4 | `report-module.sh` aggregates FAIL from lint-delays, triage-console, and schema-risk; named mutation for each flips the aggregate. |
| SC5 | `BUILD-LOOP.md` states `schema-risk.sh` as mandatory; `kit-links.bats` L4/L5 green; all new wave-1 scripts routed in both `BUILD-LOOP.md` and `skill/SKILL.md`; `scripts/install-skill.sh` run after `skill/SKILL.md` changes. |
| SC6 | bats total grows only by biting tests; `retros/INDEX.md` pending = 0 at close; `sweep-fold-audit.sh --strict` exits 0. |
| SC7 | `VERSION` = `0.19.0`; `CHANGELOG.md` entry per CONTRIBUTING §4-5; `shellcheck 0.10.0` exits 0; no commit body carries an attribution trailer. |
| SC8 *(wave 2, O)* | `station-snapshot.sh` zero writes to source, proven under read-only source dir. `bog-audit.sh` FAIL on dangling-link fixture; WARN on PANCCADIA `config.bog` for un-hidden `intervalExpired` action (no FAIL for flag drift alone). |
| SC9 *(wave 2, P–R)* | WB-LEX1/SCAFFOLD1/THREAD1/AGENT1/DEP1 bats pins green; servlet lint FAIL on branchless auth and on critical write without step-up re-auth; `BUILD-LOOP.md` §6 names each post-deploy step's script. |
| SC11 | `METHODOLOGY.md` contains the "Station load budget" section with the B806 table and the 8-count viability checklist, all lines cite `[ev: corpus B806]`. |
| SC10 | `skill/SKILL.md` and `METHODOLOGY.md` §0 contain the "Excavador Técnico" working-profile section citing `[ev: corpus B801]`; `kit-links.bats` green; `sweep-fold-audit.sh --strict` green after PR7. |

---

## Corrections bound to the executable REDs

Where spec and QA RED branches disagree, the RED wins (CD3); design records deviations.

| Pin | Maps to |
|-----|---------|
| LD1 | R1.1 — BDefrostController.java`:566` (no-floor FAIL, first PANCCADIA `time<=0`) |
| LD2, LD3 | R1.1 — BDefrostController.java`:622`, `:664` (same no-floor family) |
| LD4–LD9 | R1.3 — slot-getter facet boundary (MIN=0 FAIL / MIN≥1 WARN) |
| TR1–TR6 | R2.1 / R2.2 — own-frame grouped rows (count/first/last) |
| TR7 | R2.3 — SEVERE `Cannot load station` + our type name without own frame |
| TR8 | R2.3 — `[sys.xml]` attribution row for our types |
| TC-A | R3.C — companion flag uncleared in `stopped()` (BCompressorControl`:1799–1805`) |
| TC-B | R3.K — JDK `ScheduledExecutorService` in BComponent (BChiDashboardService`:305`) |
| TC-C | R3.N — `Clock.schedule` in `changed()` without `isRunning()` guard (BEvaporatorUnit`:519`) |

PR3–PR8 REDs are **RED pending** at spec time; apply agent re-reads branch tip (K13) before writing.
Wave-2 fixture REDs (PR10 bog, PR12 servlet) are **contract-gated**; bind when their shards land.


## Design-phase corrections (lead, after design D2a/D6a/D7a/D9a/R14.3)

- D2a: SC1 line set corrected above; POLL constant = false-positive control.
- D6a: `tests/slot-coverage.bats` SC6-parse pin (empty lexicon + types -> exit 0) is AMENDED by PR5 to exit 1 (R5.3 wins); the amendment is a contract change and carries a CHANGELOG line.
- D7a: rc-scan.sh scans rc/ only; Java hardcoded station ORD literals (`"station:|slot:/`) are a `--src` WARN sub-check in PR4.
- D9a: report-module maps schema-risk exit 2 (OUTAGE) to a FAIL row, never to the ERROR/exit-3 env branch.
- R14.3: already satisfied at 48fb210 (build.sh:84-88 prints the mirror recipe before exit 31) -> PR14 adds a regression pin only.
- Fixed-tree smoke: the client repo main is now c66e412 (PR #1 + PR #2 merged); PR1's worker MAY create a detached worktree of ~/modulos_niagara_n4/Cliente/Leon-Guanjuato at c66e412 under ~/modulos_niagara_n4/Leon-Guanjuato-worktrees/ (lead-authorized) for the PASS half; the pre-fix tree is the local checkout at 4f5f1c7.
- PR10 python3 dependency accepted (command -v guarded; SKIP rows with reason when absent).
