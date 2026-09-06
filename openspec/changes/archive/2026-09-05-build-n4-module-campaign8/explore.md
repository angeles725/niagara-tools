# Exploration — build-n4-module-campaign8

**Date**: 2026-09-05 | **Phase**: explore | **Status**: complete
**Engram**: `sdd/build-n4-module-campaign8/explore`
**Kit**: v0.18.0 (48fb210) | **Tests**: 179/179 | **Shards read**: comppan, coldroompan, dashboardpan, chihuahua, logs (all 5)
**Orchestrator gate notes** (investigador): CompPan #1 (startingUp not cleared in stopped()) confirmed by direct read (BCompressorControl.java:1760/1864/1799-1805); CompPan #5 (in-place BStatusBoolean.setValue skips propertyChanged) DISPROVED (docSource BStatusBoolean.java:62 → setBoolean → BComplex.set notifies the owner); chihuahua #1 confirmed (BChiDashboardService.java:229/305/314/453-456); the B795 OUTAGE event (PANCCADIA 2026-09-03 17:04, station never started) is [CERT-live]; the production fix for ColdRoomPan (four Clock.schedule sites floored, facets MIN 1 s, v2.0.4) is niagara-panccadia-leon PR #1, independent of this campaign.

---

## 1. Current-State Map — Coverage Matrix

| Theme | lint-timers | slot-coverage | verify --src | report-module | New (RED/candidate) |
|---|---|---|---|---|---|
| Timer floor (<=0 delay at call site) | NONE | NONE | NONE | NONE | A: lint-delays RED (9 pins) |
| Slot-getter with facet MIN=0 passed to Clock | NONE | NONE | partial† | NONE | A: lint-delays RED |
| JDK executor/Thread in BComponent | false-neg | NONE | NONE | NONE | K: lint-timers ext |
| Companion flag uncleared in stopped() | NONE | NONE | NONE | NONE | C: lint-timers ext |
| Clock.schedule in changed() without isRunning guard | NONE | NONE | NONE | NONE | N: lint-timers ext |
| Missing UNITS/PRECISION/RANGE facets | NONE | NONE | NONE‡ | NONE | D: facets lint --src |
| Hardcoded ORD/host literals in rc/ | NONE | NONE | NONE | NONE | F: rc-scan.sh |
| SPA bare catch / null display branch | NONE | NONE | NONE | NONE | F: rc-scan.sh |
| Per-slot lexicon gap / stale keys | NONE | type-level WARN | NONE | via slot-cov | E: slot-coverage ext |
| Station console exception grouping | NONE | NONE | NONE | NONE | B: triage-console RED (6 pins) |
| Schema-risk OUTAGE pre-deploy gate | prose only | NONE | NONE | NONE | BUILD-LOOP mandatory (B795-G1 CERT-live) |

†verify --src catches `BFacets.make(BFacets.MIN|MAX, <raw number>)` but not a slot-getter whose @NiagaraProperty facet has MIN=0.
‡verify --src catches raw-double BFacets.make; does not catch entirely missing facets.

**Audit QA baseline (current tools):** ColdRoomPan ISSUES (BEvaporatorUnit timer FAIL + slot-coverage 50% WARN); DashboardPan ISSUES (plano mismatch); CompPan CLEAN. B801 bug (5× "time <= 0" PANCCADIA) is invisible to every current check. B795 OUTAGE (station NEVER STARTED 2026-09-03 [CERT-live]) closes B795-G1 and upgrades schema-risk.sh from prose-advisory to mandatory pre-deploy gate.

---

## 2. Recurring Themes (count × severity, cross-module)

| Rank | Theme | Modules (refs) | Sev | Real-shape fixture (file:line) | Named mutation |
|---|---|---|---|---|---|
| 1 | Timer floor / <=0 delay passed to Clock.schedule | CRP F1-F3 + DPan #5 + PANCCADIA ×5 | BUG [CERT-live B801] | BDefrostController.java:566,622,664 getInterval/getDuration/getStaggerDelay raw (pre-fix tree) | comment out `max(1,…)` → lint-delays FAIL |
| 2 | JDK executor/Thread instead of Clock | Chi #1-2 + logs ×21 MX60+REFLOW | BUG [CERT-live] | BChiDashboardService.java:305,456 ScheduledExecutorService | replace Clock.schedulePeriodically with Executors.new* → FAIL |
| 3 | Clock.schedule in changed() without isRunning guard | CRP NotRunning ×6 PANCCADIA logs | BUG [CERT-live] | BEvaporatorUnit.java:519 (pre-fix; fix deployed at :887) | remove isRunning() check from changed() → FAIL |
| 4 | Companion flag uncleared in stopped() | CompPan #1 CONFIRMED + CRP F6 VERIFY | BUG | BCompressorControl.java:1799-1805 startingUp | remove startingUp=false from stopped() → FAIL |
| 5 | Missing UNITS/PRECISION/RANGE facets on OPERATOR slots | CRP F7-F8 + CompPan #7-8 | hygiene | BCompressorControl: 12 OPERATOR doubles no MIN/MAX | strip facet annotation → FAIL |
| 6 | Hardcoded ORD/host literals + SPA bare catch/null branch | DPan #3/#9/#12, Chi #5 | bug | DashboardReader.java:75 SERVICE_ORD + index.html catch(()=>{}) | remove ORD constant → FAIL |
| 7 | Per-slot lexicon gap / empty lexicon | CRP F9 + DPan #14-15 + Chi #11 | hygiene | chihuahua module.lexicon (EMPTY) | drop slot key → FAIL |
| 8 | Station-load hotspots (O(n) scans, ~100 KB persisted audit string, executor threads) | DPan #11 + Chi #6/#8 + logs | robustness | BChiDashboardService.java:802-810 per-tick scan | note-only until B806+ lands |

---

## 3. Backlog

| ID | Item | Value | Size | Biting test? | Blocked? |
|---|---|---|---|---|---|
| A | `lint-delays.sh` — static <=0 + slot-getter-facet-MIN=0 for Clock.schedule/schedulePeriodically [ev: B801] | HIGH | S | YES — RED qa/c8-lint-delays (9 pins; real CRP FAIL included) | no |
| B | `triage-console.sh` — own-frame exception groups (count/first/last), SEVERE "Cannot load station" + [sys.xml] type names without own frame, Spanish level normalize (ADVERTENCIA→WARNING GRAVE→SEVERE), `LC_ALL=C grep -a`, exits 0/1/3 | HIGH | S | YES — RED qa/c8-triage-console (6 pins + SEVERE expansion needed) | no |
| C | `lint-timers` ext: companion flag uncleared in stopped() — flag set beside Clock.schedule must be cleared in stopped() | HIGH | S | needs RED; CompPan #1 confirmed fixture | no |
| K | `lint-timers` ext: JDK Executors.*/ScheduledExecutorService/new Thread in BComponent → FAIL; doctrine: use Clock.schedulePeriodically(BComponent, BRelTime, Action, BValue) + Ticket.cancel in stopped() only | HIGH | S | needs RED; chihuahua [CERT-live] fixture at :305 | no |
| N | `lint-timers` ext: Clock.schedule* inside changed()/started() body without isRunning() guard → FAIL | HIGH | S | needs RED; BEvaporatorUnit pre-fix :519 is real-shape fixture | no |
| D | Facets lint `--src`: UNITS/PRECISION/RANGE absent on OPERATOR numeric/enum slots | MED | M | needs RED; CRP/CompPan fixtures | no |
| E | `slot-coverage` per-slot: @NiagaraProperty(name) vs lexicon key diff + stale key detection; empty lexicon → FAIL (not just WARN) | MED | M | needs RED; CRP/DPan/Chi fixtures | no |
| F | `rc-scan.sh`: hardcoded ORD literals + `http://`/IPv4 host + SPA bare catch `()=>{}` + null display branch in rc/ HTML/JS | MED | M | needs RED; DPan/Chi fixtures | no |
| G | Doctrine: timer-driven SM defense-in-depth (8 layers: facet floor + server rejection, call-site floor `max(1,…)`, fail-loud status/lastError, self-healing monitor tick, single idempotent ensureArmed, nextFire+overdue TRANSIENT exposure, OPERATOR force action, kit lints + console triage) → types/logic.md [ev: corpus B775/B801] | HIGH | S | none (doc-only) | no |
| H | `report-module.sh` integration: add lint-delays (A) row + triage-console --console-dir (B) row + schema-risk explicit row | MED | S | needs RED (integration row) | needs A+B |
| I | Research folds when B802+ (inter-module comms), history-ext block, B800 (console census) land → types/ doc PRs | MED | S | none (doc-only) | B802+/B803+ in progress |
| J | Client-fix guidance for #49 (DashboardPan gate 4, out of kit scope) + cert-chain kit rule (REFLOW: 7× "Could not validate cert chain" for BChiDashboardService.class, 25 lines across chihuahua-rt entries, per B800 §800.8 — trust-store import guidance, doc-only) | LOW | XS | none | no |
| L | Station/JACE load budget: doctrine in METHODOLOGY/types/logic.md (engine-thread callbacks, timer/poll density, persisted large String slots per write, JDK threads, servlet polling without backoff) + per-module viability checklist; future `station-load.sh` probe (out of campaign); skeleton draftable from findings now [ev: B806+ for full doctrine] | MED | S/doc | none (skeleton only) | B806+ for doctrine |
| M | Critical-write step-up auth: types/dashboard.md section + contract-test RED (endpoint without fresh re-auth → 401/403) + client follow-up DPan/Chi; BDashboardServlet.handleSetpointWrite:239-274 + BChiServlet canWrite are real shapes [ev: research B803/B804] | HIGH | M | RED candidate | B803/B804 |

C, K, N bundle into PR3 (all extend lint-timers.sh; share one bats file).

---

## 4. Proposed Campaign Structure

| PR | Branch | Type | QA RED | Est. lines | Notes |
|---|---|---|---|---|---|
| PR1 | feat/c8-lint-delays | code | qa/c8-lint-delays (exists, 9 pins) | ~200 | A: lint-delays.sh + tests; real CRP smoke required |
| PR2 | feat/c8-triage-console | code | qa/c8-triage-console (exists, 6 pins + SEVERE expansion) | ~150 | B: triage-console.sh + tests; re-read branch tip before writing |
| PR3 | feat/c8-lint-timers-ext | code | needs RED (3 checks: companion, executor, isRunning) | ~160 | C+K+N: 3 lint-timers.sh extensions in one bats file |
| PR4 | feat/c8-facets-lint | code | needs RED | ~100 | D: facets presence check in verify-module.sh --src |
| PR5 | feat/c8-slot-per-slot | code | needs RED | ~160 | E: slot-coverage per-slot mode |
| PR6 | feat/c8-rc-scan | code | needs RED | ~200 | F: rc-scan.sh new script |
| PR7 | docs/c8-doctrine | doc-only | none | ~70 | G: types/logic.md 8-layer SM doctrine + J client/cert note + schema-risk BUILD-LOOP mandatory + B802 inter-module line |
| PR8 | feat/c8-report-integration | code | needs RED (integration row) | ~60 | H: report-module lint-delays + console dir + schema-risk row |

Decision needed before apply: No. Chained PRs recommended: Yes. 400-line budget risk: Low (all <=200 authored lines).
Research-gated (not in chain): I (B803+/history), L full doctrine (B806+), M (B803/B804) — each lands as a doc PR when its block is delivered.

---

## 5. Risks

1. **Real-module smoke per code PR**: every new check needs a real-module fixture that fails without the fix (campaign-7 meta-lesson 1). PR1 has CRP smoke in RED (note: the fixed ColdRoomPan tree on branch fix/defrost-overdue-delay must PASS — both trees are available); PR2 needs a SEVERE fixture covering "Cannot load station"; PR3 needs the BEvaporatorUnit pre-fix :519 shape.
2. **Unverified audit findings before fixture**: ColdRoomPan F4/F6, chihuahua #10, CompPan #3 are VERIFY — a second independent read is required before they become bats fixtures; wrong-shape mutations produce tests that pass trivially.
3. **triage-console scope expansion (B)**: adding SEVERE/[sys.xml] attribution to the existing 6-pin RED (qa/c8-triage-console a4553df) changes the bats contract — QA extends the RED first; the worker re-reads the branch tip before writing.
4. **--src facets annotation heuristic (D)**: slot-getter-with-MIN=0 requires grepping the @NiagaraProperty facets annotation string, not just the BFacets.make call site — prove the heuristic against real CRP source before writing tests; rule out false positives on valid MIN=0 enum facets.
5. **report-module row contract (H)**: lint-delays.sh and triage-console.sh must emit PASS|FAIL|WARN|SKIP rows in the canonical format report-module.sh expects; mismatch breaks H without biting any existing test.
6. **schema-risk mandatory gate (PR7)**: upgrading from prose-advisory to mandatory BUILD-LOOP step is doc-only but changes operator workflow — flag in PR description; no code change needed since schema-risk.sh already exits 2 on OUTAGE.
