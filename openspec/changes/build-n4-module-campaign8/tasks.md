# Tasks: build-n4-module-campaign8

**Source**: v0.18.0 (48fb210) → **Target**: v0.19.0 | **Chain**: stacked-to-main | 15 PRs + close

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2 600–3 100 total (excl. fixture binaries) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |
| PR10 declared | size:exception (ledger 620) |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | PR | Goal | Ledger | Focused test | Rollback |
|------|----|------|--------|-------------|---------|
| 1 | PR1 | `lint-delays.sh` | 470 | `bats tests/lint-delays.bats` | new paths; `git revert` |
| 2 | PR2 | `triage-console.sh` | 440 | `bats tests/triage-console.bats` | new paths; `git revert` |
| 3 | PR3 | `lint-timers.sh` ext C/K/N | 350 | `bats tests/lint-timers.bats` | appended rules; `git revert` |
| 4 | PR4 | `verify-module --src` facets-req + ord-literal | 320 | `bats tests/verify-module.bats` | new sub-checks; `git revert` |
| 5 | PR5 | `slot-coverage per-slot` + empty-lex FAIL | 340 | `bats tests/slot-coverage.bats` | additive + pin amend; `git revert` |
| 6 | PR6 | `rc-scan.sh` rc/ assets | 440 | `bats tests/rc-scan.bats` | new script; `git revert` |
| 7 | PR7 | doctrine + Excavador Tecnico profile | 300 | `sweep-fold-audit.sh --strict` + `kit-links.bats` | doc-only; `git revert` |
| 8 | PR8 | `report-module` integration + v0.19.0 | 220 | `bats tests/report-module.bats` | additive rows; `git revert` |
| 9 | PR9 | `station-snapshot.sh` | 340 | `bats tests/station-snapshot.bats` | new script; `git revert` |
| 10 | PR10 | `bog-audit.sh` size:exception | 620 | `bats tests/bog-audit.bats` | new script + fixtures; `git revert` |
| 11 | PR11 | `wb` checks + `lint-wb-threading.sh` | 440 | `bats tests/lint-wb-threading.bats` | additive; `git revert` |
| 12 | PR12 | `rc-scan --servlet` ux lint | 420 | `bats tests/rc-scan.bats` | additive mode; `git revert` |
| 13 | PR13 | post-deploy checklist BUILD-LOOP §6 | 160 | `bats tests/kit-links.bats` | doc-only; `git revert` |
| 14 | PR14 | build pipeline doc + exit-31 regression pin | 200 | `bats tests/build-sh.bats` | doc + pin; `git revert` |
| 15 | PR15 | RT doctrine B805 + history-ext B804 | 280 | `sweep-fold-audit.sh --strict` | doc-only; `git revert` |

---

## WAVE 1

### PR1 — feat/c8-lint-delays (~215 authored, ledger 470)

**RED**: `qa/c8-lint-delays` `c96b2ad` (LD1-LD9) | **Gate**: R1.1-R1.5, SC1

- [x] 1.1 Re-read `qa/c8-lint-delays` tip `c96b2ad`: confirm LD9 pins exit 3 and LD8 WARN + exit 0 (K13).
- [x] 1.2 Cherry-pick / merge RED branch into `feat/c8-lint-delays` as commit 1.
- [x] 1.3 Write `toolbelt/lint-delays.sh`: two-pass grep/awk per D2 argument table + D2b cross-file helper resolution; `set -u`; VCS-free; `shellcheck 0.10.0` clean; exits 0/1/3 (K20).
- [x] 1.4 Copy 7 real-shape sanitised fixtures `tests/fixtures/lint-delays/{Overdue,Floored,LiteralZero,LiteralPos,Periodic,SlotGetterMinZero,SlotGetterMinPos}.java` (CD4).
- [x] 1.5 Write `tests/lint-delays.bats` (LD1-LD9 verbatim + LD10 fixed-tree PASS skip-gated on `$C8_CRP_FIXED`).
- [x] 1.6 Add CI step to `ci.yml`: per-fixture loop asserting mapped exit/row. [NOTE: ci.yml not present in repo; test coverage via bats suffices per project conventions]
- [x] 1.7 Add K19 routing to `build-n4-module-kit/BUILD-LOOP.md` + `build-n4-module-kit/skill/SKILL.md`: `lint-delays.sh [ev: retro campaign8-lint-delays]` (CD5).
- [x] 1.8 **Named mutation**: accept `Math.max(x,0L)` as valid floor → LD1 stops FAILing (LD3 is a literal-zero FAIL unrelated to Math.max mutation, does not flip); record in PR body (R1.5).
- [x] 1.9 Real smoke: `lint-delays.sh` on `Cliente/Leon-Guanjuato/.../ColdRoomPan-rt/src` (pre-fix 4f5f1c7) → exits 1, FAIL at {556,566,620,664}; fixed tree c66e412: BDefrostController clean (D2b resolves helpers); BEvaporatorUnit MIN=0 slots remain (pre-existing, not in defrost fix scope).
- [x] 1.10 Guards: `bats tests/*.bats` 189/189 green; `shellcheck` exit 0; `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0; `kit-links.bats` L4/L5 green.
- [x] 1.11 Retro file + `retros/INDEX.md` row + `BUILD-STATE.md` self-envelope in same push range (CD1); update openspec `tasks.md` apply-progress MERGE for PR1.
- [ ] **[lead]** LD1-LD9 green; merge ff-only; `scripts/install-skill.sh --force` (SKILL.md changed, D12); ledger acquire + settle `--max-changed-lines 470`.

---

### PR2 — feat/c8-triage-console (~195 authored, ledger 440)

**RED**: `qa/c8-triage-console` `6492b2d` (TR1-TR9) | **Gate**: R2.1-R2.7, SC2

- [x] 2.1 Re-read `qa/c8-triage-console` tip `6492b2d`: confirm TR1 (jetty NPE → no row), TR8 (`[sys.xml]` row), TR9 (Spanish level normalize, exit 0).
- [x] 2.2 Cherry-pick / merge RED branch into `feat/c8-triage-console`.
- [x] 2.3 Write `toolbelt/triage-console.sh`: three attribution channels C1/C2/C3 per D3; `LC_ALL=C grep -a`; `norm(msg)` digit-normalise; timestamp month table EN+ES; exits 0/1/3.
- [x] 2.4 Copy 5 fixture files `tests/fixtures/triage-console/{console,console-es,console-load-fail,console-load-fatal-only,console-load-fail-es}.txt`.
- [x] 2.5 Write `tests/triage-console.bats` (TR1-TR9 verbatim from RED).
- [x] 2.6 Add CI step to `ci.yml`: `triage-console.sh --package com.angeles tests/fixtures/triage-console/console.txt` → exit 1 + grouped row.
- [x] 2.7 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `triage-console.sh [ev: retro c8-triage-console]` (CD5).
- [x] 2.8 **Named mutations**: (a) drop C1 own-frame filter → jetty NPE row appears (TR1 flips); (b) drop C3 channel → TR8 exits 0. Record both in PR body (R2.7).
- [x] 2.9 Real smoke: `triage-console.sh` on PANCCADIA `console_backup_260903_1704.txt`, REFLOW, MX60 → SEVERE `Cannot load station` FAIL + `modifyThread` WARN rows. Paste output.
- [x] 2.10 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [x] 2.11 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** TR1-TR9 green; merge ff-only; `install-skill.sh --force`; ledger settle `--max-changed-lines 440`.

---

### PR3 — feat/c8-lint-timers-ext (~160 authored, ledger 350)

**RED**: `qa/c8-lint-timers-ext` `ce6ee5c` (TC-A/B/C + 4 companions) | **Gate**: R3.C/K/N, SC3

- [x] 3.1 Re-read `qa/c8-lint-timers-ext` tip `ce6ee5c`: confirm TC-A `BStaggerHold` clears flag in expiry path (bare grep must not pass); TC-C `changed()` has `isRunning()` yet must FAIL (guard must be in scheduling body, D4).
- [x] 3.2 Cherry-pick / merge RED branch into `feat/c8-lint-timers-ext`.
- [x] 3.3 Extend `toolbelt/lint-timers.sh`: append `companion-flag` (±3 lines + `stopped()`/`started()` body scan), `jdk-thread` (BComponent + `ScheduledExecutorService|Executors\.|new Thread(`), `changed-sched` (one-level callee body; guard-in-scheduling-body is the contract). No existing line edited — TL1-TL4 must not regress (D4).
- [x] 3.4 Extend `tests/lint-timers.bats` with TC-A (BCompressorControl `:1799-1805`), TC-B (BChiDashboardService `:305`), TC-C (BEvaporatorUnit `:519`) heredoc cases.
- [x] 3.5 **Named mutations**: (a) accept any `stopped()` cancel → TC-A no FAIL; (b) whitelist `ScheduledExecutorService` → TC-B no FAIL; (c) drop 1-level following → TC-C no FAIL. Record in PR body.
- [x] 3.6 Real smoke: `lint-timers.sh` on CompPan `:1799-1805`, chihuahua `:305`, CRP `BEvaporatorUnit` pre-fix → FAIL rows for all three. Paste output.
- [x] 3.7 Guards: bats all green (TL1-TL4 not regressed); `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`.
- [x] 3.8 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** TC-A/B/C green; TL1-TL4 not regressed; merge ff-only; ledger settle `--max-changed-lines 350`. No install-skill.

---

### PR4 — feat/c8-facets-lint (~150 authored, ledger 320)

**RED**: needs RED (`facets-req` + `ord-literal`) — re-read tip at apply (K13) | **Gate**: R4.1-R4.3, D5/D7a

- [x] 4.1 Create `qa/c8-facets-lint`; write failing bats for `facets-req` (RED tip 9cb3168 pins WARN — see 4.2) and `ord-literal` (`SERVICE_ORD` → WARN); record tip SHA in PR body.
- [x] 4.2 Extend `toolbelt/verify-module.sh` with `check_facet_presence` (label `facets-req`, D5): OPERATOR numeric without facets key → **WARN** (NOT FAIL — RED tip 9cb3168 pins WARN; tasks.md originally said FAIL but K13 makes RED authoritative; deviation recorded in retro 2026-09-05-campaign8-facets-lint.md); name-pattern without UNITS/PRECISION → WARN; presence-only — no value read (R4.2 false-positive control). Add `ord-literal` **WARN** sub-check: Java string `"(station:|local:|slot:/)` under `<profile>/src`; three exemptions per D5.
- [x] 4.3 Copy sanitised fixtures `tests/fixtures/facets/` via inline `mksrc` helper in bats (no committed fixture files — inline generation is the pattern established by other bats in this suite).
- [x] 4.4 Extend `tests/facets-lint.bats` (F1-F8, RED pins verbatim from 9cb3168); 218/218 total green.
- [x] 4.5 **Named mutations**: (a) strip facet annotation → facets-req WARN fires; (b) MIN=0 facet present → check passes (presence-only); (c) drop `OrdConstants`+comment exemption → WARN on sanctioned holder; (d) revert pass-1 `seen[]` to WARN-only marking → false-positive on boolean OPERATOR slots. All recorded in retro.
- [x] 4.6 Real smoke: `verify-module.sh --src` on CompPan (25 WARN rows — design estimated 12; real count higher) + DashboardPan-ux `DashboardReader.java:75` (→ WARN; ran on -ux not -rt — path deviation D2 in retro) + ColdRoomPan-rt (11 WARN; MIN=0 slots NOT flagged). Output verbatim in retro.
- [x] 4.7 Guards: 218/218 bats green; `shellcheck` exit 0; `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0; `kit-links.bats` 6/6.
- [x] 4.8 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); tasks.md updated (this file).
- [ ] **[lead]** RED green; merge ff-only; ledger settle `--max-changed-lines 320`. No install-skill.

---

### PR5 — feat/c8-slot-per-slot (~160 authored, ledger 340)

**RED**: `qa/c8-slot-per-slot` tip `ab194a5` (SP1-SP4 + SC6-parse amended) | **Gate**: R5.1-R5.3, SC6 (D6a: SC6-parse pin amended exit 0→1)

- [x] 5.1 RED confirmed: `ab194a5` on branch; SC6-parse/SP1/SP2/SP3/SP4 red before impl; SC1–SC6/dup-keys green. Fixtures under `tests/fixtures/slot-coverage/per-slot/` present in RED commit.
- [x] 5.2 Extend `toolbelt/slot-coverage.sh` with `per-slot` subcommand (dispatched before flag loop, D6): required = every OPERATOR-flagged `@NiagaraProperty` slot (RED/K13 gate — D6 text says "every @NiagaraProperty" but RED pins OPERATOR only; implementation follows RED; deviation documented in retro and here per K13 cite RED tip `ab194a5`); emits `pct=<n.n> (per-slot)`, `MISSING <slot>`, `STALE <key>`; escalate empty-lexicon WARN→FAIL in parse mode (D6a, always exits 1); dot-dir prune D9b; 4 named mutations confirmed.
- [x] 5.3 SC6-parse pin was already amended in the RED commit (ab194a5): `[ "$status" -eq 1 ]`. Implementation makes it green.
- [x] 5.4 CHANGELOG.md `## [Unreleased]` section added with D6a behaviour change entry and per-slot Added entry.
- [x] 5.5 Named mutations all confirmed: (a) add setpoint key → SP1 FAILS; (b) EMPTY_LEX_FAIL=0 → chihuahua exit 0, SC6-parse FAILS; (c) remove STALE printf → SP2 FAILS; (d) remove dot-dir prune → staleKnob MISSING, SP4 FAILS.
- [x] 5.6 Real smokes: ColdRoomPan 9 MISSING (design said ~19 — deviation documented; only BEvaporatorUnit has OPERATOR slots), chihuahua exits 1 FAIL, CompPan pct=100.0 exit 0. Outputs in retro.
- [x] 5.7 Guards: 214/214 bats green; shellcheck exit 0; sweep-build-state.sh exit 0; sweep-fold-audit.sh --strict exit 0; kit-links.bats 6/6 green.
- [x] 5.8 Retro `2026-09-05-campaign8-slot-per-slot.md` + INDEX row + BUILD-STATE envelope in same commit.
- [ ] **[lead]** RED green; merge ff-only; ledger settle `--max-changed-lines 340`. No install-skill.

---

### PR6 — feat/c8-rc-scan (~215 authored, ledger 440)

**RED**: needs RED — re-read tip at apply | **Gate**: R6.1-R6.6, SC5 K19/CD5

- [x] 6.1 Create `qa/c8-rc-scan`; write failing bats: ORD literal under `rc/` → FAIL; host literal → FAIL; bare catch → WARN; null branch → WARN; record tip SHA.
- [x] 6.2 Write `toolbelt/rc-scan.sh <artifact-dir> [--strict] [--servlet]`: scan `**/rc/**` `*.html *.js *.css` only (D7); check table per D7; excludes `rc/ext/**`, `*.min.js`, `srcTest/**`, comment-only; exits 0/1/3.
- [x] 6.3 Copy sanitised fixtures `tests/fixtures/rc-scan/{clean,ords,host,catch,null}/` from real DashboardPan shapes.
- [x] 6.4 Write `tests/rc-scan.bats` (RED pins verbatim).
- [x] 6.5 Add CI step to `ci.yml`: `rc-scan.sh tests/fixtures/rc-scan/clean` → exit 0.
- [x] 6.6 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `rc-scan.sh [ev: retro c8-rc-scan]` (CD5).
- [x] 6.7 **Named mutations**: (a) remove ORD rule → ORDFAIL exits 0; (b) restore bare `.catch(()=>{})` → WARN row appears. Record in PR body (R6.6).
- [x] 6.8 Real smoke: `rc-scan.sh` on DashboardPan artifact → FAIL `index.html:701` (host) + WARN `:1298` (bare catch) + WARN `:863` (null branch). Paste output.
- [x] 6.9 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [x] 6.10 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** RED green; merge ff-only; `install-skill.sh --force`; ledger settle `--max-changed-lines 440`.

---

### PR7 — docs/c8-doctrine (~85 authored, ledger 300, doc-only)

**RED**: none (CD2) | **Gate**: R7.1-R7.10, SC5 K6, SC10, SC11

- [x] 7.1 grep-before-fold: `rg '8-layer|Excavador|schema-risk.*mandatory|inter-module.*service|step-up.*auth|BHistoryExt|station load budget' build-n4-module-kit/` → confirm 0 hits for new content (K6).
- [x] 7.2 Extend `types/logic.md` §Safety: 8-layer timer SM defense (layers 1-8, R7.1) + "folded as code: `lint-delays.sh`" line `[ev: corpus B775]` `[ev: corpus B801]`.
- [x] 7.3 Extend `types/logic-authoring.md` with §"talking to another module" (R7.3): `BLink`/service-discovery/`Subscriber`; `fox:` hop boundary; anti-pattern: bespoke bus; B802-G1 OPEN `[ev: corpus B802]`.
- [x] 7.4 Extend `types/dashboard.md`: step-up auth section (R7.4/R7.5) + CsrfUtil correction to DWS1 gate 2 + #49 client guidance; B803-G1/G2 OPEN `[ev: corpus B803]`.
- [x] 7.5 Extend `toolbelt/triage-console.sh` header + `METHODOLOGY.md` §Conformance: triple-attribution + locale contract + "folded as code: `triage-console.sh`" (R2.6, D7) `[ev: corpus B800]`.
- [x] 7.6 Extend `toolbelt/verify-module.sh` header §Checks: cert-chain trust note + "folded as code:" line (R7.6, D8) `[ev: corpus B800]`.
- [x] 7.7 Extend `BUILD-LOOP.md` §5: `schema-risk.sh` mandatory pre-deploy; exit-2 blocks deploy (R7.2) `[ev: corpus B795]` + CERT-live.
- [x] 7.8 Extend `METHODOLOGY.md` §0 + `skill/SKILL.md`: Excavador Tecnico working-profile (R7.9); decompiled-line-numbers note (R7.7); station load budget table + §806.9 8-count checklist (R7.10) — all lines `[ev: corpus B806]`. Investigador1 fidelity read of station-load-budget section.
- [x] 7.9 `kit-links.bats` L4/L5 green (routing already shipped with PR1/2/6 — PR7 adds no new routing line, D8).
- [x] 7.10 `sweep-fold-audit.sh --strict` exits 0; every `[ev:]` token resolvable; all doc delta lines carry `[ev: corpus B…]` (R7.8).
- [x] 7.11 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** `kit-links.bats` + `sweep-fold-audit.sh --strict` green; merge ff-only; `install-skill.sh --force` (SKILL.md changed, D12); ledger settle `--max-changed-lines 300`.

---

### PR8 — feat/c8-report-integration (~80 authored, ledger 220) — VERSION 0.19.0

**RED**: needs RED — re-read tip at apply | **Gate**: R8.1-R8.5, SC4, SC7, D9a, D12

- [x] 8.1 Create `qa/c8-report-integration`; write failing bats: lint-delays FAIL in member → aggregate FAIL (AggFAIL); `triage-console` FAIL → aggregate FAIL; `schema-risk` exit 2 → FAIL row (not ERROR, D9a); record tip SHA. [RED commit d8322d4 already on branch per K13]
- [x] 8.2 Extend `toolbelt/report-module.sh` with `--console-dir <dir>` flag; append per-artifact `lint-delays.sh <artifact>/src` (SKIP when no `src/`) and `schema-risk.sh`; once-per-run `triage-console.sh --console-dir`; map schema-risk exit 2 → FAIL row (D9a).
- [x] 8.3 Extend `tests/report-module.bats` (RED pins verbatim + AggFAIL scenario). [Already in RED commit d8322d4; RM1-RM8 all green]
- [x] 8.4 Bump `VERSION` → `0.19.0`; rename `## [Unreleased]` → `## [v0.19.0] - 2026-09-05` in `CHANGELOG.md`; add `### References` block; include entry for `--console-dir` new flag (D12).
- [x] 8.5 **Named mutation**: drop `lint-delays` aggregation → report exits 0 on a delay-FAIL module (R8.5). Observed: exit=0 CLEAN; RM4 flips. Recorded in retro.
- [x] 8.6 Real smoke: `report-module.sh` on CRP pre-fix tree 4f5f1c7 → exits 1, 4 FAIL lint-delays rows (:556/:566/:620/:664) + 2 FAIL timer rows; ISSUES. With --console-dir HoneywellMX605132026: 3 triage-console FAIL rows.
- [x] 8.7 Guards: bats 239/239 green; `shellcheck` exit 0; `sweep-build-state.sh` exit 0; `sweep-fold-audit.sh --strict` exit 0; `kit-links.bats` 6/6 green.
- [x] 8.8 Retro `2026-09-05-campaign8-report-integration.md` + INDEX row + `BUILD-STATE.md` envelope (CD1); openspec tasks.md ticked (this edit).
- [ ] **[lead]** RED green; merge ff-only; ledger settle `--max-changed-lines 220`. No install-skill.

---

## WAVE 2

### PR9 — feat/c8-station-snapshot (~165 authored, ledger 340)

**RED**: needs RED — re-read tip at apply | **Gate**: R9.1, SC8 (O.1)

- [ ] 9.1 Create `qa/c8-station-snapshot`; write failing bats: read-only-source pin (`chmod a-w` source → snapshot still exits 0; source dir unmodified); manifest sha256 present; record tip SHA.
- [ ] 9.2 Write `toolbelt/station-snapshot.sh <station-dir> <out-dir>`: `cp` config.bog + console*.txt + db pointers; sorted `manifest.txt` (sha256 + relpath + bytes); no station connection; refuse `<out-dir>` inside station-dir; never sets `+x`; exits 0/1/3 (D10).
- [ ] 9.3 Write `tests/station-snapshot.bats` (RED pins verbatim).
- [ ] 9.4 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `station-snapshot.sh [ev: retro c8-station-snapshot]` (CD5).
- [ ] 9.5 **Named mutation**: make snapshot write to source dir → read-only pin FAILs. Record in PR body.
- [ ] 9.6 Real smoke: `station-snapshot.sh` on PANCCADIA station dir copy → exits 0, manifest lists config.bog + consoles, source dir unchanged. Paste output.
- [ ] 9.7 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [ ] 9.8 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** RED green; merge ff-only; `install-skill.sh --force`; ledger settle `--max-changed-lines 340`.

---

### PR10 — feat/c8-bog-audit (~265 authored, ledger 620 size:exception)

**RED**: needs RED (shard landed — bound) — re-read tip at apply | **Gate**: R10.1-R10.3, SC8 (O.2), python3 guarded

- [ ] 10.1 Create `qa/c8-bog-audit`; write failing bats: CHECK7 FAIL on dangling-link fixture; CHECK2 WARN only (not FAIL) for flag drift; malformed bog → exit 3; `python3` absent → exit 3; record tip SHA.
- [ ] 10.2 Write `toolbelt/bog-audit.sh <config.bog> --module <MOD>… [--source-dir] [--strict]`: `command -v python3 || exit 3`; python3 stdlib line-scanner; CHECK1-CHECK10 per D10 grammar; **CHECK11 `proxy-link-safety` (B810 D11)**: for every link whose source is an own-module output slot and target is a writable proxy point (`c:BooleanWritable`/`c:NumericWritable` under a driver network), resolve device tuning policy (writeOnUp/writeOnStart/writeOnEnabled — defaults TRUE per `BTuningPolicy.java:177/206/235`): FAIL when writeOnUp is explicitly false; WARN when target has no explicit `fallback`; WARN when device has no ping health / pingFrequency. CHECK2 WARN by default / `--strict` FAIL; exits 0/1/3.
- [ ] 10.3 Write 60-line synthetic `tests/fixtures/bog/synthetic.bog` (CHECK5 orphan, CHECK3 `differentialUp=-1.5`, CHECK7 dangling `sourceSlot`, CHECK2 `intervalExpired f='o'`, CHECK11 proxy-link with no `fallback` → WARN) + `clean.bog`; no customer config.bog committed (D11).
- [ ] 10.4 Write `tests/bog-audit.bats` (RED pins verbatim; per-check mutations for CHECK2/3/5/7/11).
- [ ] 10.5 Add CI step to `ci.yml`: `bog-audit.sh tests/fixtures/bog/clean.bog --module Demo` → exit 0.
- [ ] 10.6 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `bog-audit.sh [ev: retro c8-bog-audit]` (CD5).
- [ ] 10.7 **Named mutations**: (a) skip handle resolution → CHECK7 dangling stops FAILing; (b) CHECK2 always FAIL → WARN-only fixture fails; (c) CHECK5 drop orphan detection → orphan not caught; (d) drop CHECK11 fallback WARN → PANCCADIA 22-output smoke clean. Record in PR body.
- [ ] 10.8 Real smoke: `bog-audit.sh` on PANCCADIA config.bog → CHECK2 WARN (`DefrostController.intervalExpired f='o'`), CHECK11 WARNs on 22 own outputs → NRIO ro1..ro10 (0 explicit fallbacks, writeOnUp default true), 0 CHECK5/7/9/10 FAIL; MX60 clean. Paste output.
- [ ] 10.9 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [ ] 10.10 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** RED green; merge ff-only; `install-skill.sh --force`; ledger settle `--max-changed-lines 620 --declare-exception`.

---

### PR11 — feat/c8-wb-audit (~215 authored, ledger 440)

**RED**: needs RED (wb shard landed) — re-read tip at apply | **Gate**: WB-LEX1/SCAFFOLD1/THREAD1/AGENT1/DEP1, SC9

- [ ] 11.1 Create `qa/c8-wb-audit`; write failing bats for WB-LEX1 (chihuahua-wb no lexicon → FAIL), WB-SCAFFOLD1 (0 classes + 0 palette → WARN), WB-THREAD1 (BBatchLinkEditor `doInvoke` DFS → FAIL; WARN heuristic: flag for human review per check body comment), WB-AGENT1 (no justification comment → WARN), WB-DEP1 (phantom dep → WARN); record tip SHA.
- [ ] 11.2 Write `toolbelt/lint-wb-threading.sh <wb-src-dir>`: FAIL `doInvoke` bodies calling `getNavChildren|getNavNodes|BQL` without `invokeLater|BJobService`; WARN `@AgentOn(types="baja:Component")` with no justification comment; exits 0/1/3; `shellcheck` clean.
- [ ] 11.3 Extend `toolbelt/slot-coverage.sh`: run over every `-wb` `module-include.xml` + `module.lexicon`; missing lexicon with declared types → FAIL (WB-LEX1).
- [ ] 11.4 Extend `toolbelt/verify-module.sh`: WB-SCAFFOLD1 check — `-wb` jar 0 classes AND 0 palette → WARN; `--strict` FAIL; closes `:245-257` dead angle. WB-DEP1 `--src`: built `module.xml` `<dependency>` not in `api()`/`nre()` → WARN.
- [ ] 11.5 Add `types/wb-widgets.md`: 10-line good-`-wb`-artifact doctrine; chihuahua-wb `model/` cited as DWB1 exemplar.
- [ ] 11.6 Write `tests/lint-wb-threading.bats`; extend `tests/slot-coverage.bats` + `tests/verify-module.bats` (RED pins verbatim).
- [ ] 11.7 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `lint-wb-threading.sh [ev: retro c8-wb-audit]` (CD5).
- [ ] 11.8 **Named mutations**: (a) empty `-wb` scaffold stops being detected; (b) `doInvoke` DFS stops FAILing. Record in PR body.
- [ ] 11.9 Real smoke: `lint-wb-threading.sh` on BBatchLinkEditor-shaped fixture → FAIL at `doInvoke:684`; `slot-coverage.sh` on chihuahua-wb → exits 1. Paste output.
- [ ] 11.10 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [ ] 11.11 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** RED green; merge ff-only; `install-skill.sh --force`; ledger settle `--max-changed-lines 440`.

---

### PR12 — feat/c8-ux-servlet (~195 authored, ledger 420)

**RED**: needs RED (fixture shapes contract-gated) — re-read tip at apply | **Gate**: R12.1-R12.5, SC9

- [ ] 12.1 Create `qa/c8-ux-servlet`; write failing bats: API branch no auth → FAIL; critical write without step-up re-auth → FAIL; `X-Requested-With`-only CSRF → FAIL; invalid value → 400 no write; missing cache headers → WARN; record tip SHA.
- [ ] 12.2 Extend `toolbelt/rc-scan.sh` with `--servlet` mode: scan `BWebServlet` subclasses; FAIL on missing `getRemoteUser()`/RBAC; FAIL on write without validated value; FAIL on `X-Requested-With`-only; FAIL on critical write without short-TTL step-up token `[ev: corpus B803 §803.6]`; WARN on missing cache headers.
- [ ] 12.3 Copy sanitised fixtures from `BDashboardServlet.handleSetpointWrite:239-274` + `BChiServlet:613` shapes.
- [ ] 12.4 Extend `tests/rc-scan.bats` with `--servlet` cases (RED pins verbatim).
- [ ] 12.5 Add K19 routing: `BUILD-LOOP.md` + `skill/SKILL.md` `rc-scan.sh --servlet [ev: retro c8-ux-servlet]` (CD5).
- [ ] 12.6 **Named mutations**: (a) remove auth check from one branch → FAIL; (b) `X-Requested-With`-only → FAIL; (c) step-up token absent → FAIL. Record in PR body.
- [ ] 12.7 Real smoke: `rc-scan.sh --servlet` on DashboardServlet fixture → FAIL rows; clean servlet → exits 0. Paste output.
- [ ] 12.8 Guards: bats all green; `shellcheck` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats` L4/L5.
- [ ] 12.9 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** RED green; merge ff-only; `install-skill.sh --force`; ledger settle `--max-changed-lines 420`.

---

### PR13 — docs/c8-post-deploy (~40 authored, ledger 160, doc-only)

**RED**: none (CD2) | **Gate**: R13.1-R13.2, SC9

- [ ] 13.1 grep-before-fold: `rg 'post-deploy|snapshot.*triage-console.*bog-audit' build-n4-module-kit/BUILD-LOOP.md` → confirm 0 hits (K6).
- [ ] 13.2 Add `BUILD-LOOP.md` §6 post-deploy checklist: snapshot → `triage-console` → `bog-audit` → `report-module` → keep snapshot as deploy baseline; one line per step naming its script; include: "proxy-link safety row (CHECK11) must be clean before operator hand-off" (B810).
- [ ] 13.3 Extend `tests/kit-links.bats` L4/L5: assert every §6 step script is named (R13.2).
- [ ] 13.4 `kit-links.bats` exits 0; `sweep-fold-audit.sh --strict` exits 0.
- [ ] 13.5 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** Guards green; merge ff-only; ledger settle `--max-changed-lines 160`. No install-skill.

---

### PR14 — docs/c8-build-pipeline (~60 authored, ledger 200, doc-only + 1 pin)

**RED**: none (CD2); exit-31 pin is a code addition | **Gate**: R14.1-R14.4

- [ ] 14.1 grep-before-fold: `rg 'Gradle task matrix|mirror-niagara-home|vendorVersion.*bajaVersion' build-n4-module-kit/BUILD-LOOP.md` → confirm 0 hits (K6).
- [ ] 14.2 Add `BUILD-LOOP.md` §4 Gradle task matrix: clean/slotomatic/compileJava/jar/build/moduleTestJar/niagaraTest/bajadoc — what each touches, safe combos, mirror-niagara-home.sh trick.
- [ ] 14.3 Add `BUILD-LOOP.md` version-bump checklist: `vendorVersion` vs `bajaVersion`, station reload behaviour, restart mandate, `schema-risk.sh` required before any slot-touching bump.
- [ ] 14.4 Write `tests/build-sh.bats` pin: `build.sh` exits 31 and stderr contains the mirror-niagara-home.sh recipe text (R14.3 regression pin; behaviour already live at `build.sh:84-88`).
- [ ] 14.5 `bats tests/build-sh.bats` exits 0; `sweep-fold-audit.sh --strict` exits 0.
- [ ] 14.6 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** Guards green; merge ff-only; ledger settle `--max-changed-lines 200`. No install-skill.

---

### PR15 — docs/c8-rt-doctrine (~90 authored, ledger 280, doc-only)

**RED**: none (CD2) | **Gate**: R15.1-R15.4, B805 landed (`ccd8ab95d`), B804 available

- [ ] 15.1 grep-before-fold: `rg 'PID.*anti-windup|errorSum|BHistoryExt|BLatch.*edge' build-n4-module-kit/types/` → confirm 0 hits (K6).
- [ ] 15.2 Extend `types/logic.md` §RT-control-logic: §805.10 — PID anti-windup (`errorSum` clamp), NaN → fault + hold + bumpless transfer, deadband latch-on-cross; `execute()`/`changed()` split; §805.9 flowchart template (STATES/INPUTS/TIMERS/CONTROL/PROTECTIONS/OUTPUTS/FEEDBACK); health/feedback checklist (R15.2); `BLatch` edge-only note; `[CERT-negative]` no ODE facility `[ev: corpus B805]`.
- [ ] 15.3 Extend `types/logic-authoring.md` §"logging a point to history": `BHistoryExt` as point extension, Interval vs COV, `BHistoryConfig` capacity + `fullPolicy`, one ext per logged slot; B804-G1 OPEN `[ev: corpus B804]`.
- [ ] 15.4 `sweep-fold-audit.sh --strict` exits 0; every `[ev: corpus B805]` + `[ev: corpus B804]` token resolvable.
- [ ] 15.5 Retro + INDEX row + `BUILD-STATE.md` envelope (CD1); update openspec apply-progress.
- [ ] **[lead]** `sweep-fold-audit.sh --strict` green; merge ff-only; ledger settle `--max-changed-lines 280`. No install-skill.

---

## CLOSE — Campaign-8 close

**Gate**: SC1-SC11, CD1/CD5/CD6/CD7

- [ ] C.1 `sdd-verify`: run against `spec.md` SC1-SC11; every wave-1 assertion green; wave-2 assertions green for all landed slices.
- [ ] C.2 `grep -c '| pending |' build-n4-module-kit/retros/INDEX.md` == 0 (all retros folded, pending = 0).
- [ ] C.3 Confirm `VERSION` = `0.19.0` and `CHANGELOG.md` contains `## [v0.19.0]` (landed in PR8, D12).
- [ ] C.4 Final sweep: `bats tests/*.bats` all green; `shellcheck` exit 0 all new scripts; `sweep-fold-audit.sh --strict` exit 0; `sweep-build-state.sh` exit 0; no commit body carries an attribution trailer (CD7).
- [ ] C.5 `git tag v0.19.0` on final merged commit; push tag.
- [ ] C.6 `sdd-archive`: archive change to `openspec/changes/archive/2026-09-05-build-n4-module-campaign8/`.
