# Exploration — build-n4-module-campaign10

**Date**: 2026-09-07 | **Phase**: explore | **Status**: complete
**Engram**: `sdd/build-n4-module-campaign10/explore` (observation 8389)
**Kit**: v0.20.0 (tag 1fb63d6; C9 archive at df8c7ec) | **Client modules** (client main ff1b659): CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan rt/ux 2.2.0
**Tunnel**: PR #1/#2/#3 (config login, change_log, audit mirror) blessed, awaiting Cristian's merge; tunnel main 872c64c
**Station (PANCCADIA)**: still running ColdRoomPan-rt 2.0.3 / CompPan-rt 2.0.1 / DashboardPan 2.0 (the RARs) — four repo versions behind
**Source**: consolidates companero probe `2026-09-07-c10-explore-draft.md` (45d550dac) + `2026-09-07-c10-lint-refinement-apply-packages.md` (2f710626d) + `2026-09-07-before-c10-checklist.md` + investigador1 B831 (686b54ac5) + QA C10 RED branches + C9 archive `explore.md`/`proposal.md` + `retros/2026-09-06-campaign9-close-process-meta-lessons.md` (25 lessons) + kit issue #89.

> Shape mirrors `openspec/changes/archive/2026-09-06-build-n4-module-campaign9/explore.md`. Every claim carries an `[ev:]` token. SHAs are research-repo (niagara-research) unless noted `[client]`, `[kit]` or `[tunnel]`.

---

## 1. Mandate

### 1.1 Confirmed product decisions (constraints — do NOT re-open)

1. **Keep improving.** C10 is authorized. The campaign opens immediately with the kit lint-precision wave (W1 = S21 + S22 + S23 plus S24/S25/S26 hygiene), which is WSL-only and needs nothing from Cristian. `[ev: Cristian 2026-09-06 "Sigan trabajando para continuar con las mejoras"]`
2. **Deploy chain and niagaraTest harness session are PREREQUISITES, not C10 work.** The pending client jars (ColdRoomPan 2.0.7 / CompPan 2.0.3 / DashboardPan 2.1.1) plus the C9 bumps (CompPan 2.2.0, ColdRoomPan 2.1.0, DashboardPan 2.2.0) must be deployed before any C10 client jar stacks on them. The niagaraTest harness session (C9 harness-only pins CRA1/2/3-live, CPB5, R14 lockout) must run and its result recorded before any C10 client jar that depends on alarm/adapter behavior. Kit lanes (S21/S22/S23) need neither. `[ev: 2026-09-07-before-c10-checklist.md §A-§C]`
3. **Product seeds P1-P5 are CONDITIONAL.** W2 is gated behind: (a) tunnel merge + deploy chain + harness session; (b) Cristian's three station answers (defrost trial / intercambiador Cuarto 3 / coolOnSensorFault link). These are gated slices, not committed scope. `[ev: 2026-09-07-before-c10-checklist.md §D]`
4. **Research is SELECTED for W1 and DELIVERED.** investigador1's B831 (`niagara-mental-model-bloque831.md`, 686b54ac5) is the single source-backed block for the three rules: S21 field-vs-local + same-method-body scoping; S22 action-writes-THIS-slot detection; S23 Pattern B surface tokens. Propose may start; B831-G1/G2 stay open gates. `[ev: B831 686b54ac5]`
5. **Process constraints from C9 carry over (chain-wide).** K11 no AI attribution trailers; K12 workers write only in their own worktrees; K13 RED cited by branch, tip re-read at apply, workers never edit QA REDs; K14 metric names state what they measure; D9b every scanner prunes dot-dirs; K19 every toolbelt script routed in BUILD-LOOP + SKILL; K20 disjoint exit codes; one `[ev:]` per paragraph/row; workers run real-tree smokes on a blessed-tip worktree (read tree = `Leon-Guanjuato-worktrees/main-ff1b659` or a C10 worktree at that tip, never the stale main checkout); smoke pins assert exact count + subject + absence; strip `//` and `/* */` before regex pins; every check must bite with an OBSERVED mutation flip. `[ev: retros/2026-09-06-campaign9-close-process-meta-lessons.md lessons K11-K14/K19/K20/D9b + lessons 21/24/25]`

---

## 2. Current State

| Axis | State at C10 open | Token |
|---|---|---|
| Kit | v0.20.0 (1fb63d6; main df8c7ec = C9 archive chore, lint tools byte-identical) — all C9 lints landed: lint-timers (companion-flag STILL FP on anyNoHardware — issue #89), lint-silent-protection (Pattern-A-only — STILL WARNs CompressorControl.java:294 post-PR9), lint-ext-writable-shape (class-level has_action — STILL FN on faultReset), lint-demand-scope, lint-write-path, lint-structure, lint-servlet, lint-wb-threading, rc-scan, verify-module, report-module, schema-risk, triage-console, station-snapshot, bog-audit | `[kit]` `[ev: BUILD-STATE.md last_session 2026-09-06]` |
| Client — CompPan-rt | 2.2.0 (ff1b659) — S20 rotation shipped (PR #12); CP-1 alarm Pattern B shipped (PR #15) | `[client]` `[ev: Compresores/build.gradle.kts:33]` |
| Client — ColdRoomPan-rt | 2.1.0 (ff1b659) — CR-3 freeze alarm Pattern A shipped (PR #11) | `[client]` `[ev: Paccadia/build.gradle.kts:33]` |
| Client — DashboardPan rt/ux | 2.2.0 (ff1b659) — servlet guards + config login shipped (PR #10/#13) | `[client]` `[ev: Dashboard/build.gradle.kts:33]` |
| Tunnel (poller + write-server) | PRs #1/#2/#3 blessed, NOT merged; config step-up + unified change_log + audit mirror live only on those branches; `sql/2026-09-06-change-log-extend.sql` NOT applied | `[tunnel]` `[ev: 2026-09-07-before-c10-checklist.md §A]` |
| Station (PANCCADIA) | Running ColdRoomPan-rt 2.0.3 / CompPan-rt 2.0.1 / DashboardPan 2.0 (RARs); pending 2.0.7/2.0.3/2.1.1 then the C9 jars; deploy runbook not executed | `[ev: Cristian 2026-09-06 RARs Cliente/Leon-Guanjuato/{Compresores,Leon-Guanjuato}.rar]` |
| Kit issue #89 | lint-timers companion-flag FP on anyNoHardware filed at C9 close; S21 fixes it | `[kit]` `[ev: issue #89]` |

---

## 3. Evidence Inventory

### 3.1 Research blocks

| Block | SHA | One-line finding | Token |
|---|---|---|---|
| B820 | `16b635f0f` | demand-in-scope lint — a staged process decides WHETHER before HOW | `[CERT]` |
| B821 | `f960f2997` | protection anatomy — RT modules zero alarm-console events; 22 trips classified | `[CERT]` |
| B822 | `9d1a336b1` | additive-code setpoint write; retype = OUTAGE | `[CERT]` |
| B824 | `b5060f60b` | silent-protection lint — effect-slot exemption + pure-model→adapter follow | `[CERT]` |
| B825 | `c5ac2ca57` | propagation mechanism — oBIX write synchronous; lag is the reader poll | `[CERT-live]` |
| B826 | `b53fdea9d` | child-ORD routability — `/value` resolvable and writable; child bare-`<real>` = preferred | `[CERT-live]` |
| B827 | `ff0ce3f5b` | alarm authoring — Pattern A child-point / Pattern B BIAlarmSource+AlarmSupport | `[CERT]` |
| B828 | `e1d58acc7` | HOA frozen enum — BFrozenEnum carries range intrinsically; §828.7 cross-module-link carve-out | `[CERT]` |
| B829 | `d26305d21` | audit trail — null-Context set NOT audited; AuditHistoryService installed | `[CERT]` |
| B830 | `778d3b64b` | Niagara auth API — BUser implements Context; lockout via authenticateFailed; 401 on unrecognized scheme; TTL must be compared on every request | `[CERT]` |
| **B831** | `686b54ac5` | **C10 lint precision (S21/S22/S23) at client ff1b659** — S21: `anyNoHardware` is a method-LOCAL at BDefrostController.java:718 inside requestDefrostCycle (:713, no Clock.schedule); the schedules live at :808/:810/:850 in other methods → rule needs class FIELD + same-method schedule. S22: `faultReset` is BStatusBoolean SUMMARY\|OPERATOR at BCompressorControl.java:1612; no @NiagaraAction (:435/:437/:439) writes it; the only write is setFaultReset(false) at :2025 inside execute/changed → rule follows setX/.set only inside @NiagaraAction bodies (excluding execute/changed/generated setters). S23: CP-1 shed at CompressorControl.java:294 is alarmed via BIAlarmSource (BCompressorControl.java:447) + AlarmSupport (:1882) + newOffnormalAlarm (:2093); Pattern A token = BAlarmSourceExt child (BEvaporatorUnit.java:193). Open: B831-G1 doAckAlarm indirection; B831-G2 build the three fixtures | `[CERT]` |

### 3.2 QA RED branches (kit, origin) — the contract per K13

| Branch | Tip | Base | Mapping | Status |
|---|---|---|---|---|
| `qa/c10-lint-timers-fp` | `52ebd11` | df8c7ec | S21 — `tests/lint-timers.bats`: S21-neg (method-local bool + Clock.schedule, FP today → clean), S21-pos (FIELD flag cleared off-lifecycle, FAIL stays as guard), S21-smoke (ColdRoomPan-rt @ ff1b659 anyNoHardware exit 1 → 0) | RED verified RED-for-the-right-reason on df8c7ec |
| `qa/c10-ext-writable-per-slot` | `00e31ae` | df8c7ec | S22 — `tests/ext-writable-shape.bats`: EW-s22-pos (action writes the slot → clean), EW-s22-neg (unrelated action → WARN; FAILs today), EW10 real-tree re-cut CompPan-rt 0 → **1** (faultReset) | RED; **CONTRACT CHANGE** carried inside this branch (supersedes `qa/c9-ext-writable-shape` 3726722 EW10) |
| `qa/c10-silent-protection-surfaces` | `f981754` | df8c7ec | S23 — `tests/lint-silent-protection.bats`: S23-pos (real SP1 `Math.min(target, onCount-1)` trip + Pattern-B adapter → clean; WARNs today), S23-neg (trip, no surface → WARN guard), SP-smoke CompPan-rt 1 → **0** (:294 surfaced) | RED; first fixture cut was vacuous, re-done (RP15) |
| `qa/c10-close-checklist` | `41bca42` | df8c7ec | `tests/c10-close.bats` skeleton — C10_CLOSE-guarded, inert 12/12 skip, BASE 1fb63d6, C10_CLOSE_COMMIT param, VERSION/tag/SC-13 TODO(freeze), tool-pins = C9 set (no new tool files), CLOSE-harness-run carried as pending | skeleton |
| `qa/c9-verify-runbooks` | `18420d9` | — | `qa/c9-harness-runsheet.md` — one-page executable sheet for Cristian's Windows box; running it closes CLOSE-harness-run (C9 gate 14/14) | owed to Cristian |
| `qa/c9-silent-protection` | `e38e503` | — | C9 SP1-SP8 + SP-smoke (Pattern-A-only) | reference only |

`[ev: git ls-remote origin qa/c10-* qa/c9-verify-runbooks 2026-09-07]`

### 3.3 C10 apply-packages (companero, 2f710626d — cited against kit df8c7ec)

- **S21**: `toolbelt/lint-timers.sh` companion-flag — anchors :135-212 (Pass 1 :135-168 scans forward from the assignment, never checks field-vs-local). Two guards required: (1) identifier is a class-scope field; (2) Clock.schedule in the SAME enclosing method body as the assignment. FAIL OBSERVED at df8c7ec exit 1 on ColdRoomPan-rt @ ff1b659. `[CERT]`
- **S22**: `toolbelt/lint-ext-writable-shape.sh` — anchors :82-91 (class-level has_action counts ANY @NiagaraAction, including HIDDEN unrelated ones). Replace with per-slot body-follow (reuse lint-silent-protection :124-165 slot→writer follow). faultReset exempt at ff1b659 (0 WARN OBSERVED). CONTRACT CHANGE: EW10 CompPan-rt 0→1. `[CERT]`
- **S23**: `toolbelt/lint-silent-protection.sh` — anchors :222/:424 (Pattern-A-only: BAlarmSourceExt / BAlarmRecord). Extend the surface recogniser with BIAlarmSource + newOffnormalAlarm / AlarmSupport and the adapter→pure follow (B`<PureClass>` naming pair). WARN OBSERVED at ff1b659 CompressorControl.java:294 post-PR9. `[CERT]`
- **Kit issue #89**: companion-flag FP on anyNoHardware; root cause confirmed (scan-forward + no field-vs-local check); designated C10 S21. `[kit]`

### 3.4 Companion probes (research repo)

- `2026-09-06-c10-hmi-per-operator-login-options.md` (3d212e746): R14 (shipped) already answers "who changed X" and is SAFER on a shared touch panel than a per-operator kiosk session; options B/C only add per-operator VIEW/RBAC, a NEW requirement layered UNDER R14. One question for Cristian: attribution (done) or per-operator screens (new)? `[ev: 3d212e746]`
- `2026-09-06-c10-live-gate-plan-v2.md` (3d212e746): supersedes the C9 v1 — P0 deploy chain first, P1 tunnel merge before any surface-A gate, Part 1 read-only R1-R5, Part 2 minimum writes W1-W5, Part 3 Windows harness H1-H3, after-steps A1-A4; names what it cannot close (B830-G2/G3, B828-G2, B831-G2). `[ev: 3d212e746]`

---

## 4. Candidate Slices — Waves

### Wave 1: Lint-precision cluster (S21 + S22 + S23) + hygiene (S24/S25/S26)

**Rationale**: WSL-only, high tractability, closes issue #89, removes two verified false results and the C9-shipped false alarm. Needs nothing from Cristian. REDs already authored by QA; research delivered (B831).

| Slice | Class | Repo | Seam | RED status | `[ev:]` |
|---|---|---|---|---|---|
| **S21** lint-timers companion-flag: class-FIELD + same-method-body scope | KIT | `niagara-tools` | `toolbelt/lint-timers.sh` :135-212; Pass 1 :135-168 — add (1) class-field-only guard via a class-scope pass, (2) enclosing-method-body anchor for the schedule pairing | RED `qa/c10-lint-timers-fp` 52ebd11 (S21-neg/pos/smoke) | `[ev: apply-package S21 2f710626d]` `[ev: B831 §S21]` |
| **S22** lint-ext-writable-shape per-slot writing-action | KIT | `niagara-tools` | `toolbelt/lint-ext-writable-shape.sh` :82-91; per-slot body-follow inside @NiagaraAction bodies (excl. execute/changed/generated setters) replacing class-level has_action | RED `qa/c10-ext-writable-per-slot` 00e31ae; **CONTRACT CHANGE** EW10 CompPan-rt 0→1 (faultReset, BCompressorControl.java:1612); open gate B831-G1 (write inside `doX` body must count for action `x`) | `[ev: apply-package S22 2f710626d]` `[ev: B831 §S22]` |
| **S23** lint-silent-protection Pattern B surface | KIT | `niagara-tools` | `toolbelt/lint-silent-protection.sh` :222/:424; surface tokens BIAlarmSource / AlarmSupport / newOffnormalAlarm + B`<PureClass>` adapter→pure follow | RED `qa/c10-silent-protection-surfaces` f981754 (S23-pos/neg, SP-smoke 1→0) | `[ev: apply-package S23 2f710626d]` `[ev: B831 §S23]` |
| **S24** cwd-independent structural REDs | KIT/test | `niagara-tools` | run-pure-test.sh / structural test classes — resolve src from the test's own location, not cwd; land before more structural REDs are authored in C10 | RED not yet authored — one from-any-cwd harness pin (QA) | `[ev: explore-draft 45d550dac §S24]` |
| **S25** `lint-write-path --strict` flag | KIT | `niagara-tools` | `toolbelt/lint-write-path.sh` — add `--strict` exit 1; WARN-only default unchanged (C9 design assumed the flag existed) | RED not yet authored — lint-write-path.bats --strict exit-1 pin (QA) | `[ev: explore-draft 45d550dac §S25]` |
| **S26** gitignore client build caches | CLIENT hygiene | `angeles725/niagara-panccadia-leon` | `.gitignore` — stop `build/tmp` + `*.class` churn; NOTE the module jars under `build/` are tracked by convention — the ignore must not cover them | No RED (chore); needs a diff-shows-no-jar check | `[ev: explore-draft 45d550dac §S26]` |

### Wave 2: Product seeds (GATED — not committed scope)

**Gate conditions**: tunnel PRs #1-#3 merged + deploy chain executed + harness session recorded + Cristian's three station answers.

| Slice | Class | Repo | Gate | `[ev:]` |
|---|---|---|---|---|
| **P1** viewer per-user re-auth + configurator role list (write-server) | PRODUCT (tunnel) | `pancaddia-leon-tunnel` | Tunnel merge + Supabase operator re-check + role table decision | `[ev: explore-draft 45d550dac §P1]` |
| **P2** HMI per-operator kiosk login (DashboardPan-ux) | PRODUCT (client -ux) | `angeles725/niagara-panccadia-leon` | Cristian's answer: attribution (R14, done) vs per-operator screens (new RBAC) + harness session green | `[ev: 3d212e746 hmi-per-operator-login-options]` |
| **P3** `airDefrost` module flag (rooms 1/2/4) | PRODUCT (client -rt) | `angeles725/niagara-panccadia-leon` | Cristian's defrost trial green light | `[ev: explore-draft 45d550dac §P3]` |
| **P4** intercambiador Cuarto 3 control point | PRODUCT (station + client) | station + client | Cristian confirms it is on a Niagara output | `[ev: explore-draft 45d550dac §P4]` |
| **P5** `coolOnSensorFault` station-side link (all rooms) | STATION | station (Workbench) | Cristian approves the station-side link | `[ev: explore-draft 45d550dac §P5]` |

---

## 5. Risks

1. **S22 contract change — EW10 CompPan-rt 0→1.** The re-cut lives in `qa/c10-ext-writable-per-slot` 00e31ae; a worker that reads the C9 tip (3726722) ships the wrong contract. K13: the RED is the contract; exact-count pin = 1 WARN, subject `faultReset` (pinned by NAME, not line — the lint anchors the `@NiagaraProperty(` open at BCompressorControl.java:381; :1612 is the generated `Property faultReset` the lint never cites), no other CompPan-rt slot flips (module-find: 1 OPERATOR complex property in CompPan-rt). `[ev: QA 2026-09-07 max-WARN run on ff1b659]`
2. **B831-G1 (doAckAlarm indirection) is the action→`do<Action>` mapping.** The rule must map `@NiagaraAction(name="x")` to `doX()` and scan THAT body for the slot write; EW-s22-pos already uses this shape (`bumpSetpoint` → `doBumpSetpoint`), and `doAckAlarm` writes the alarm ack, not faultReset, so faultReset stays unexempted. An extra pin (a `do`-body writing a DIFFERENT slot must not exempt faultReset) is requested from QA. `[ev: B831-G1; QA 2026-09-07 EW-s22-pos]`
3. **Stale-checkout reads recur as the campaign's defect class (C9 lesson 1).** Every file cite names the worktree and commit. Smoke tree = `Leon-Guanjuato-worktrees/main-ff1b659` or a C10 worktree at that tip — never the stale main checkout. `[ev: retros/2026-09-06-campaign9-close-process-meta-lessons.md lesson 1]`
4. **Comment-satisfiable pins (C9 lesson 21).** Strip `//` and `/* */` before any identifier or WARN-string assertion; S21/S22/S23 fixtures must include a comment-only decoy. `[ev: retros/2026-09-06-campaign9-close-process-meta-lessons.md lesson 21]`
5. **Harness dependency for P2/P3/P4.** Without the niagaraTest harness session, alarm/adapter REDs stay structural + SKIP — a SKIP is not a pass. `[ev: 2026-09-07-before-c10-checklist.md §C]`
6. **Always-conflict files across parallel kit PRs.** BUILD-LOOP routing, SKILL toolbelt line, retros/INDEX.md row, BUILD-STATE.md envelope: append, keep both rows, dedupe by script name, never overwrite; merge after each wave. `[ev: retros/2026-09-06-campaign8-close-process-meta-lessons.md lesson 2]`
7. **P1-P5 scope creep if station gates resolve mid-W1.** Queue P-slices for W2 after W1 closes; do not interrupt the lint wave. `[ev: explore-draft 45d550dac §3]`

---

## 6. Open Questions — Requires-Execution Gates + Cristian Decisions

### 6.1 Requires-execution (WSL — resolve during W1 apply)

| Gate | Question |
|---|---|
| S21-smoke | `lint-timers.sh` on ColdRoomPan-rt/src at ff1b659 exits 0 after the fix; anyNoHardware no longer flagged; synthetic FIELD flag still FAILs with an OBSERVED flip |
| S22-smoke | `lint-ext-writable-shape.sh` on CompPan-rt/src at ff1b659 → exactly 1 WARN (faultReset); DashboardPan-rt unchanged; ColdRoomPan-rt 0 |
| S23-smoke | `lint-silent-protection.sh` on CompPan-rt/src at ff1b659 → 0 WARN (CP-1 recognised via Pattern B); ColdRoomPan-rt 0; DashboardPan 0 |
| S24-smoke | Structural test passes from any cwd, not only from the profile dir |
| B831-G1 | `doX` body counts as the action body for `x` (fixture + OBSERVED flip) |

### 6.2 Cristian's decisions (block W2, never W1)

| Decision | Unblocks |
|---|---|
| Tunnel PRs #1-#3 merge (viewer must send x-config-token) | P1/P2 |
| Deploy chain execution (2.0.7/2.0.3/2.1.1 then the C9 jars) | ALL C10 client jars |
| niagaraTest harness session on the Windows box (`qa/c9-harness-runsheet.md`) | C9 gate 14/14; P2/P3/P4 alarm REDs |
| P2 need: attribution (done by R14) vs per-operator screens (new RBAC) | P2 |
| Defrost trial green light (rooms 1/2/4) | P3 |
| Intercambiador Cuarto 3 on a Niagara output? YES/NO | P4 (YES) or demote the panel control (NO) |
| coolOnSensorFault station-side link approval | P5 |

---

## 7. Research Gate Answer

**Research**: SELECTED for W1 — DELIVERED as B831 (686b54ac5) with `[CERT]` anchors at client ff1b659 and two named open gaps (B831-G1, B831-G2). Propose may start.

**Product decisions**: W1 CONFIRMED (lint precision + hygiene). W2 (P1-P5) NOT confirmed — gated slices only.

**Evidence completeness**: All W1 FP/FN claims carry `[CERT]` (OBSERVED lint runs at df8c7ec / ff1b659; QA REDs verified RED-for-the-right-reason). W2 claims are conditional.

**Next recommended**: `sdd-propose`

---

## Key Learnings

1. S21/S22/S23 share one defect class — coarse class-level heuristics instead of per-slot / per-method-body matching — so they land as a single lint-precision wave without cross-contaminating each other's contract.
2. S22 is the only contract change in W1 (EW10 CompPan-rt 0→1, faultReset); QA cut the re-pin inside the C10 RED branch so the worker never touches the C9 RED (K13).
3. The blessed smoke tree is `Leon-Guanjuato-worktrees/main-ff1b659` or a C10 worktree at that tip; the stale main checkout is never the smoke target (C9 lesson 1).
4. Research landed before propose as one combined block (B831) rather than three, which keeps the anchors consistent across the three lints.
5. Product seeds P1-P5 stay conditional on seven pending Cristian decisions; listing them as gated slices preserves W1's WSL-only autonomy.
