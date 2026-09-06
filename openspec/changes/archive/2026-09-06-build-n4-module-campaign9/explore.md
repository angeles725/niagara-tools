# Exploration — build-n4-module-campaign9

**Date**: 2026-09-06 | **Phase**: explore | **Status**: complete
**Engram**: `sdd/build-n4-module-campaign9/explore` (observation 8361)
**Kit**: v0.19.0 (C8 close at 1109c0f) | **Client modules**: ColdRoomPan-rt 2.0.7, CompPan-rt 2.0.3, DashboardPan rt/ux 2.1.1
**Tunnel (mini-PC poller/write-server)**: main at 9acb47c (viewer, 2026-09-06); S12-A RED base e4b42b0; viewer v3.3
**Source**: consolidates investigador1's terrain draft (`niagara-research/sources/probes/2026-09-06-c9-explore-draft.md`) + C9 seeds (`campaign9-research-candidates.md`) + S12 plan (`2026-09-06-c9-s12-config-login-audit-plan.md`) + slot-type doctrine draft (`2026-09-06-c8-slot-type-doctrine-draft.md`) + C8 close meta-lessons (`retros/2026-09-06-campaign8-close-process-meta-lessons.md`) + BUILD-LOOP.md §5/§7 + the C8 explore archive + the viewer echo (tunnel main 9acb47c, 2026-09-06).

> Shape mirrors `openspec/changes/archive/2026-09-05-build-n4-module-campaign8/explore.md`. Every claim carries a marker; SHAs are research-repo (`niagara-research`) unless noted `[tunnel]` / `[client]`. Where a C9 seed has no authored RED branch yet it is marked **RED not yet authored**, never given a fabricated tip.

---

## 1. Mandate

### 1.1 Confirmed product decisions (constraints — do NOT re-open)

1. **S20 CompPan time-slice rotation is the FIRST client PR of C9.** `BCompressorControl` gains two additive `@NiagaraProperty`: `rotationInterval` (`BRelTime`, `SUMMARY|OPERATOR`; 0 = disabled, byte-identical to today's behavior) and `rotationMode` (enum, make-before-break default). Logic in `CompressorControl.step` after target computation; gated by `minOn`/`minOff`/`stageDelay`; HOA OFF excluded; HAND untouched; no swap on `dischargeHigh`; no swap below the LP floor; no swap when only one compressor is available. Per-compressor rotation clock uses `cmdSince[]` (`:71`, already tracked). Schema-risk SAFE (additive, no retype). `[ev: corpus S20 (campaign9-research-candidates.md)]` `[client]`
2. **S12 config login + unified write audit is URGENT — two surfaces.** Surface A = write-server (tunnel repo, Node.js): config step-up token (`POST /config/login` re-auths against Supabase, server-held TTL token, sliding inactivity expiry), `/config/logout` revokes immediately; token required on `/write` and `/alarms/ack`; audit rows `{ts, email, ord, old, new, result, ip, config_session}`. Surface B = DashboardPan servlet: `DashboardWriteGuards.evaluate` seam + B829-G2 real-Context `parent.set(prop, toSet, cx)` so the servlet write becomes Niagara-audited. NO JSON user+password store (path retracted). Station-side step-up is NOT the primary path (primary = Supabase re-auth). Audit-append failure must NEVER fail the write — the write lands + an error/alarm row. `[ev: corpus S12]` `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]`
3. **Setpoint RETYPE is REJECTED.** The B800 incident (station never started after a `BStatusNumeric`↔`BDouble` retype, 2026-09-03 `[CERT-live]`) makes retype an OUTAGE-class change. The shipped write form is the child-leaf ORD `…/setpoint/value` bare `<real>` PUT (write-server e4b42b0; B826-G1/G2 closed `[CERT-live]`). The wrapped-`obj`-to-parent-slot form (B825) is the proven fallback only. `[ev: corpus B826]` `[ev: corpus B822]` `[ev: corpus B800 §800.8]`
4. **airDefrost for rooms 1/2/4 is a STATION-ONLY client trial.** OR of `evapOut`/`resistanceOut` into the fan relay, AND fan-mode≠OFF, per unit, `hasDefrost=true`, fallback=false. NO module change in C9; the `airDefrost` module flag stays a deferred seed. companero produces the link-list with bog-nav outside SDD. `[ev: corpus S16]`
5. **Kit lints with existing QA RED branches are in scope.** S7 demand-in-scope (`qa/c9-demand-in-scope` `2916954`), S18-lint silent-protection (`qa/c9-silent-protection` `e38e503`), S19 ext-writable-shape (`qa/c9-ext-writable-shape` `3726722`). S12-B RED `qa/c9-s12-servlet` `4c18837` (client `[CERT]`); S12-A RED `qa/c9-s12-write-server` `24adcba` (tunnel, base e4b42b0, `[CERT]` via viewer echo). `[ev: git ls-remote origin qa/c9-* 2026-09-06]`
6. **S18/S13 alarm PoC in scope; RED to be authored by QA first.** `BAlarmSourceExt` on CR-3 freeze + CP-1 low suction (B827 patterns A/B). Schema-SAFE (additive child point or additive `BIAlarmSource` + transient `AlarmSupport`). S5-cont write-path matrix W14-W22 in scope (kit doc, extending `qa/c8-write-path` `5e357d1`). Kit conventions delta K22 + BUILD-LOOP §5 one-module-root convention in scope. `[ev: corpus B827]` `[ev: corpus S5]` `[ev: retro campaign8-close-process-meta-lessons lesson 11]`
7. **Process constraints carry over from C8 (chain-wide).** K11 no AI attribution trailers; K12 workers write only in worktrees; K13 RED cited by branch, tip re-read at apply; K14 metric names state what they measure; D9b every scanner prunes dot-dirs; K19 every toolbelt script routed in BUILD-LOOP + SKILL; K20 disjoint exit codes; one `[ev:]` per paragraph/row; lead runs real-tree smokes on every client module root; rebase a parallel branch onto main BEFORE the QA ping and verify the tip before settling the ledger; smoke pins assert exact counts + subjects + absences (lesson 11); every check must bite — mutation flip OBSERVED (lessons 1/7). `[ev: retro campaign8-close-process-meta-lessons]`

### 1.2 Protection-surface baseline (drives S18/S13 and S18-lint)

The RT modules (`ColdRoomPan-rt/src`, `CompPan-rt/src`) raise ZERO alarm-console events — a clean grep of `BAlarmSourceExt|BAlarmRecord|BAlarmService` returns nothing; every protection tops out at a plain `SUMMARY` slot (tier 2). Verified at client `fbe9009`. The alarm fix is spec'd via B827: two legal authoring patterns (A: declarative child point + `BAlarmSourceExt`; B: `BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` on the offnormal edge). `[ev: corpus B821 §821.4]` `[ev: corpus B827]`

---

## 2. Current State

| Axis | State at C9 open | Token |
|---|---|---|
| Kit | v0.19.0 (C8 close at 1109c0f) — lint-delays, lint-timers(+ext), lint-servlet, lint-structure, lint-write-path, lint-wb-threading, rc-scan, verify-module (+facets-req/ord-literal), report-module, schema-risk mandatory gate, triage-console, station-snapshot, bog-audit CHECK1-19, sweep-fold-audit, new-retro, kit-ticket | `[ev: corpus C8 close]` |
| Client — ColdRoomPan-rt | 2.0.7 | `[client]` |
| Client — CompPan-rt | 2.0.3 | `[client]` |
| Client — DashboardPan rt/ux | 2.1.1 | `[client]` |
| Tunnel (poller + write-server) | main 9acb47c (2026-09-06) — adds a best-effort write audit into Supabase `public.change_log` (JWT-bearer only, NO config step-up); S12-A RED base e4b42b0; write-server ships the child-ORD `${ord}/value` bare-`<real>` setpoint path (e4b42b0) | `[CERT via viewer echo]` `[tunnel]` `[ev: corpus B826]` |
| Station (PANCCADIA) | `AuditHistoryService` INSTALLED (`/PANCCADIA/AuditHistory`, bog read `[CERT]`, B829-G1 CLOSED); 22 relays still null-fallback (S15 STATION fix open); setpoint write form = child bare-`<real>` via write-server | `[ev: corpus B829]` `[ev: corpus B810 §810.8]` |

---

## 3. Evidence Inventory

### 3.1 C9 seeds (campaign9-research-candidates.md, `a5a2e5cba`+)

S1 rt component-lifecycle testable seam (exec: YES) · S2 protection-latch seam (PoC `e31bd60a1`) · S3 heartbeat/liveness monitor (PoC `fc9caa1ff`) · S4 HOA-precedence seam (PoC `5a9020fd6`, client `20f74f8`) · **S5** write-path coverage lint + W-matrix (PR19 `0590c2b7f`) · S6 structure lint L1-L11 (PR18 `f7a4521ee`) · **S7** demand-in-scope lint (B820) · S8 `station-load.sh` (exec: YES) · S9 commit-msg AI-trailer hook (K11) · S10 schema-risk slot→declaring-class · S11 lint-delays cross-module helper · **S12** config step-up + write audit (TOP-RANKED) · **S13** health surface tier-1 per-module (B821 §821.6) · S14 tag dictionary `angeles` (exec: YES) · S15 `fallback=false` on 22 relays (STATION) · S16 Cuarto-1 links + bog-audit CHECK12 (STATION) · S17 persist drafts to repo · **S18** protection trips never reach console (CROSS-CUTTING; B827 spec'd) · **S19** ext-writable-shape lint (B823) · **S20** CompPan time-slice rotation (user-explicit 2026-09-06). `[ev: corpus campaign9-research-candidates.md]`

### 3.2 Delivered research blocks (B820-B829)

| Block | SHA | One-line finding | Token |
|---|---|---|---|
| B820 | `16b635f0f` | demand-in-scope lint — a staged process decides WHETHER before HOW (zero-demand idle) | `[CERT]` |
| B821 | `f960f2997` (re-anchored `fbe9009`) | protection anatomy — RT modules raise ZERO alarm-console events; 22 trips classified on three axes; none latch first-out | `[CERT]` |
| B822 | `9d1a336b1` | additive-code setpoint write — `applySetpoint(BDouble)` action oBIX-native (recommended); `setpointCmd` double (fallback); retype = OUTAGE | `[CERT]` |
| B824 | `b5060f60b` | silent-protection lint — effect-slot exemption + pure-model→adapter follow + name allowlist; CP-1 FLAG, CP-2/defrostSkipped clean | `[CERT]` |
| B825 | `c5ac2ca57` (upd `3e8dc8b45`) | propagation mechanism — oBIX write = top-slot replacement, SYNCHRONOUS propagation; lag is the reader poll; `[CERT-live]` via B826-G2 | `[CERT]` |
| B826 | `b53fdea9d` (upd `3e8dc8b45`) | child-ORD routability — child `/value` NOT advertised (agent leaf-collapse) but RESOLVABLE and `writable="true"`; both gaps CLOSED `[CERT-live]`; child bare-`<real>` = PREFERRED write form | `[CERT-live]` |
| B827 | `ff0ce3f5b` | alarm authoring — `BAlarmSourceExt` needs a `BControlPoint` parent → child-point OR programmatic `BIAlarmSource`+`AlarmSupport`; corrects B8 §8.1.4/§8.1.5 | `[CERT]` |
| B828 | `e1d58acc7` (§828.7 `c1cdef272`) | HOA frozen enum — `BFrozenEnum` carries range intrinsically; §828.7 carve-out: cross-module-LINKED HOA stays double | `[CERT]` |
| B829 | `d26305d21` (G1 CLOSED `7218fdad7`) | audit trail — servlet null-Context set NOT audited (`ComplexSlotMap:662` gate); oBIX PUT audited to the oBIX login user; AuditHistoryService installed | `[CERT]` |

### 3.3 QA RED branches (by branch + tip, verified 2026-09-06)

**Five C9 REDs:**

| Seed | Branch | Tip | Repo | Pins | Seam it needs | Token |
|---|---|---|---|---|---|---|
| S19 | `qa/c9-ext-writable-shape` | `3726722` | niagara-tools (origin) | EW1-EW10 (EW10 = real `BRoomPanel.setpoint` WARN) | none (static lint) | `[CERT]` |
| S18-lint | `qa/c9-silent-protection` | `e38e503` | niagara-tools (origin) | SP1-SP8 + SP-smoke (mutation pin SP8 OBSERVED; smoke flags CP-1/CR-3, not CP-2) | none (static lint) | `[CERT]` |
| S7 | `qa/c9-demand-in-scope` | `2916954` | niagara-tools (origin) | DS1-DS7 + DS-smoke (mutant pin DS2 OBSERVED; smoke on real `CompressorControl.step`) | none (standalone script) | `[CERT]` |
| S12-A | `qa/c9-s12-write-server` | `24adcba` | pancaddia-leon-tunnel (base e4b42b0; target rebase 9acb47c) | S12A-1..S12A-7; 1 file `instalacion/pipeline/test/write-server.config-login.test.mjs`; expects `buildServer(cfg,deps)` seam guarding `main()`; named mutation: dropping the config-token check flips S12A-1/S12A-5 | `buildServer` seam (config-token store, `/config/logout`, audit rows) | `[CERT via viewer echo]` |
| S12-B | `qa/c9-s12-servlet` | `4c18837` | niagara-panccadia-leon (rebased `a109249`, origin) | guard1-5 + guard4b + success/one-audit-entry (7 total); guard4 = no-silent-zero regression pin | `DashboardWriteGuards.evaluate` seam + B829-G2 real-Context | `[CERT]` |

**Two C8 QA REDs mapping to standing C9 seeds:**

| Branch | Tip | C9-seed mapping | Token |
|---|---|---|---|
| `qa/c8-write-path` | `5e357d1` | **S5-cont** — write-path coverage lint RED (W-matrix; extends to W14-W22) | `[CERT]` |
| `qa/c8-structure` | `c32cb5a` | **S6** — structure lint L1-L11 RED (shipped in PR18; standing reference) | `[CERT]` |

`[ev: git ls-remote origin qa/c9-* 2026-09-06]` `[ev: viewer echo tunnel main 9acb47c 2026-09-06]`

### 3.4 Plans and live records

- **S12 config-login + write-audit plan** — `1ecdf437c` (fixed `80adc279e`): config-login step-up + audited setpoint write; `DashboardDispatch` executable guard `:121-126`, `SetpointWrite` `:59-60`; servlet `handleSetpointWrite:195`, `parent.set:291`, `coerceValue:357`, `appendAudit:312`; write-server acquires the old value via GET before PUT; guards XHR-302 / auth-401 / OPERATOR_WRITE-403 / invalid-400; B829 null-Context confirmed as SUPPRESSION (not just un-attribution) even with the service installed. `[ev: corpus S12 plan 1ecdf437c]`
- **Tunnel main 9acb47c audit surface (viewer echo, 2026-09-06):** write-server now inserts a best-effort row into Supabase `public.change_log` `{user_email, user_id, room, slot, label, old_value, new_value, area, ok}` on every successful `/write` (JWT-bearer only). This is NOT config step-up and NOT the full audit design (no `/config/login`, no `/config/logout`). The S12-A RED (`24adcba`) sits on base e4b42b0 and must be rebased onto 9acb47c at apply; the `buildServer(cfg,deps)` seam contract in the RED remains valid. `change_log` vs the RED's JSON-lines/`audit` sink is an open **audit-sink reconciliation** for the design phase. `[CERT via viewer echo]` `[tunnel]`
- **Slot-type doctrine draft** (`2026-09-06-c8-slot-type-doctrine-draft.md`): doctrine for `types/logic-authoring.md` §"Slot types for externally written values" + one-liner in `types/dashboard.md`; grounded in B823/B825/B826 `[CERT-live]`. The enforcing lint is S19 (C9). `[ev: corpus B823]` `[ev: corpus B826]`
- **Live oBIX probe record** — `aa7054702` (§1-11): the viewer's live probes closed B823-G1, B826-G1/G2 (child `/value` served + `writable="true"`, propagates ~1.5 s). `[CERT-live]`
- **C8 close meta-lessons** (`retros/2026-09-06-campaign8-close-process-meta-lessons.md`): 11 lessons; lesson 11 → BUILD-LOOP §5 one-module-root/profile convention + METHODOLOGY K22 (smoke pin asserts count + subject + absence; a skipped smoke blocks, never advisory). `[ev: retro campaign8-close-process-meta-lessons]`

---

## 4. Candidate Slices — Waves

### Wave 1: S20 client PR + S7 + S18-lint kit lints

Rationale: S20 is the user's explicit first PR; S7 and S18-lint are kit lints with full REDs already authored and mutation-proven (OBSERVED flips). Three slices run in parallel (S7 and S18-lint in separate kit worktrees; S20 in a client worktree). No dependency on wave 2 or 3.

| Slice | Class | Repo | Seam | RED Status | `[ev:]` |
|---|---|---|---|---|---|
| **S20** CompPan time-slice rotation | CLIENT | `angeles725/niagara-panccadia-leon` (CompPan-rt) | `CompressorControl.step` — two additive `@NiagaraProperty` (`rotationInterval` BRelTime, `rotationMode` enum); rotation gate uses `cmdSince[]` (`:71`) per compressor; schema-risk SAFE (additive) | RED not yet authored (QA lane); pins: swap-after-interval, NO-swap-below-interval, NO-swap-while-minOff, make-before-break ordering (incoming ON → stageDelay → outgoing OFF), disabled-at-0 byte-identical golden, hours ledger unaffected | `[ev: corpus S20]` `[ev: corpus B822 (additive-code)]` `[ev: corpus B795]` `[client]` |
| **S7** demand-in-scope lint | KIT | `niagara-tools` (origin) | standalone script; fixture defined in B820; `CompressorControl.step` real-tree smoke | RED `qa/c9-demand-in-scope` `2916954` `[CERT]`; DS1-DS7 + DS-smoke; mutation pin DS2 OBSERVED | `[ev: corpus B820]` |
| **S18-lint** silent-protection lint | KIT | `niagara-tools` (origin) | static lint; effect-slot exemption + pure-model→adapter follow + name allowlist from B824 | RED `qa/c9-silent-protection` `e38e503` `[CERT]`; SP1-SP8 + SP-smoke; mutation pin SP8 OBSERVED; smoke flags CP-1/CR-3 | `[ev: corpus B824]` |

### Wave 2: S12-A + S12-B (config login + write audit)

Rationale: both surfaces implement the same user-visible feature and share the same audit schema; they run in separate worktrees (tunnel repo vs client repo). S12-A carries the open audit-sink reconciliation (design-phase decision). The RED `24adcba` is on base e4b42b0; the apply worker rebases onto 9acb47c before writing. Neither surface depends on wave 1 output.

| Slice | Class | Repo | Seam | RED Status | `[ev:]` |
|---|---|---|---|---|---|
| **S12-A** config step-up + audit — write-server (mini-PC) | TUNNEL | `pancaddia-leon-tunnel` (base e4b42b0; target rebase 9acb47c) | `buildServer(cfg,deps)` seam — `POST /config/login` mints a server-held TTL token bound to `(email+purpose)`; `/write` and `/alarms/ack` gate on the token; `/config/logout` revokes; audit row with old/new (pre-write GET); NO JSON user+password store; **audit-sink reconciliation open**: `public.change_log` (9acb47c, JWT-bearer only, no step-up) vs the RED's `audit` design — design decides the canonical sink | RED `qa/c9-s12-write-server` `24adcba` `[CERT via viewer echo]`; S12A-1..S12A-7; audit-fail never fails the write (write lands + error row) | `[ev: corpus S12]` `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]` `[tunnel]` |
| **S12-B** config step-up + audit — DashboardPan servlet (HMI) | CLIENT | `angeles725/niagara-panccadia-leon` (DashboardPan-ux) | `DashboardWriteGuards.evaluate` seam (new class gating the servlet write) + B829-G2: `parent.set(prop, toSet, cx)` with the real request user instead of `null` — the servlet write becomes Niagara-audited; DashboardDispatch executable guard `:121-126`, SetpointWrite `:59-60` | RED `qa/c9-s12-servlet` `4c18837` `[CERT]` (rebased `a109249`); guard1-5 + guard4b + success/one-audit-entry (7 pins); guard4 = no-silent-zero regression pin; audit-fail never fails the write | `[ev: corpus S12]` `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]` `[client]` |

### Wave 3: S18/S13 alarm PoC + S19 + S5-cont + K22/§5 docs

Rationale: S18/S13 needs a QA-authored RED first (B827 sketches available; additive and schema-SAFE). S19 and S5-cont are static/doc work with REDs authored. K22/§5 is docs-only. All four can run in parallel in separate worktrees; no hard dependency on wave 2.

| Slice | Class | Repo | Seam | RED Status | `[ev:]` |
|---|---|---|---|---|---|
| **S18/S13** alarm PoC — `BAlarmSourceExt` on CR-3 freeze + CP-1 low suction | CLIENT | `angeles725/niagara-panccadia-leon` (CompPan-rt, ColdRoomPan-rt) | Pattern A (CR-3 freeze): child `BBooleanPoint` + `BAlarmSourceExt` with `BBooleanChangeOfStateAlgorithm`; Pattern B (CP-1 low suction): `BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` on the offnormal EDGE only (`wasOffnormal[]` state + `toNormal` on recovery + `started()` re-seed); both route `offnormal` sourceState — already selected by the DashboardPan alarm bql `:502` | RED not yet authored — QA authors first from B827 §827.3/§827.4/§827.6; B827-G1 edge-detection invariants explicit in the RED | `[ev: corpus B827]` `[ev: corpus B821 §821.4/§821.6]` `[client]` |
| **S19** ext-writable-shape lint | KIT | `niagara-tools` (origin) | static lint; WARN on a `Flags.OPERATOR` complex property (`BStatusNumeric`/`BStatusBoolean`/`BStatusEnum`) with no writing action; plain `double`/`boolean`/`BRelTime` or complex-with-`@NiagaraAction` → clean | RED `qa/c9-ext-writable-shape` `3726722` `[CERT]`; EW1-EW10; EW10 = real `BRoomPanel.setpoint` WARN | `[ev: corpus B823]` `[ev: retro obix-statusnumeric-wrapped-put]` |
| **S5-cont** write-path matrix W14-W22 | KIT | `niagara-tools` (origin) | author rows W14-W22 in `docs/write-path-matrix.md` against the real writable set; `lint-write-path.sh` shipped in PR19 | RED `qa/c8-write-path` `5e357d1` `[CERT]` (standing C8 RED, extends to W14-W22) | `[ev: corpus S5]` `[ev: corpus B816 §816.6]` |
| **K22/§5 docs** kit conventions delta | KIT | `niagara-tools` (origin) | `BUILD-LOOP.md §5`: ONE module-root/profile convention (every lint iterates profiles; a root with no sources = exit 3, never a silent 0); `METHODOLOGY.md K22`: real-tree smoke on every client module root + smoke pins assert count + subject + absence; doc-only | doc-only; no RED — lesson 11 anchor is the PR18/PR19/PR20 fix commits | `[ev: retro campaign8-close-process-meta-lessons lesson 11]` |

---

## 5. Risks

1. **Real-module smoke per code PR (lessons 1/7/11).** Every new lint/seam needs a fixture that FAILS without the fix AND a real-tree smoke asserting exact counts + subjects + absence. S7/S18-lint/S19 have authored REDs with OBSERVED mutation flips. The alarm PoC (S18/S13) and S20 do not yet — author RED-first, mutation-prove with an OBSERVED flip before apply. `[ev: retro campaign8-close-process-meta-lessons lessons 1/7/11]`
2. **S12-A audit-sink reconciliation (change_log vs the RED's sink).** Tunnel main 9acb47c already ships a best-effort `public.change_log` insert on each `/write` (JWT-bearer, no step-up). The S12-A RED expects an audit row with old/new values and config-session identity. Design must decide: unify into one canonical table, keep both, or extend `change_log` with the richer schema. NOT a propose blocker — a design-phase input. `[CERT via viewer echo]` `[tunnel]`
3. **S12 audit-fail semantics — the write must survive.** An audit APPEND FAILURE must never fail the write (DWS1 gate 5 in the S12 plan). A RED that fails the write on audit-fail tests the wrong contract; this must be an explicit named pin in both S12-A and S12-B. `[ev: corpus S12]` `[ev: S12 plan 1ecdf437c]`
4. **S12-A RED rebase onto 9acb47c.** The RED `24adcba` sits on base e4b42b0. At apply the worker rebases onto 9acb47c before writing — the `buildServer(cfg,deps)` seam contract remains valid, but the existing `change_log` insert on main must not silently break the suite. K13 applies: re-read the branch tip at apply. `[CERT via viewer echo]` `[tunnel]`
5. **B829-G2 real-Context change (S12-B surface B).** Passing the real user Context to `parent.set` is schema-neutral but changes the servlet's audit behavior — pin `schema-risk.sh` SAFE and confirm the `AuditEvent` actually reaches `/PANCCADIA/AuditHistory` in a live session. B829-G1 is CLOSED (service present `[CERT]`); the only gap is the null-Context gate. `[ev: corpus B829]`
6. **B827-G1 alarm edge detection (S18/S13 PoC).** Pattern B must fire `newOffnormalAlarm` only on the normal→offnormal transition; a naive level-triggered implementation re-fires every execute. The RED must encode `wasOffnormal[]` per-trip state + `toNormal` on recovery + `started()` re-seed — QA authors these invariants before apply. `[ev: corpus B827]`
7. **HOA frozen-enum retrofit trap (B828 §828.7).** Applying the frozen-enum doctrine to a cross-module-LINKED HOA deletes the linked type → live "Missing class" crash. Frozen-enum is NEW-modules-only. Any C9 work touching HOA slots must audit for cross-module links first. `[ev: corpus B828 §828.7]`
8. **Stale-tree reads on client cites.** The client tip moves (B821 read v2.0.0 while main was `fbe9009`; B823 cited `deed38c` vs main `fbe9009`). Re-anchor every load-bearing client cite at the actual tip the chain builds on before promoting to `[CERT]`. `[ev: corpus B821 re-anchor]` `[ev: corpus B823]`
9. **Parallel worktree merge conflicts on the four always-conflict files.** Every kit PR touches the BUILD-LOOP routing line, the SKILL toolbelt line, a `retros/INDEX.md` row, and the `BUILD-STATE.md` envelope. The fragment-merge rule (append, keep both rows, dedupe by script name, never overwrite) applies; merge after each wave before opening the next. `[ev: retro campaign8-close-process-meta-lessons lesson 2]`

---

## 6. Open Questions — Requires-Execution Gates

These are NOT blockers for propose; they are live-session confirmations to pair before or during apply (station access only with Cristian's direct authorization, routed through him — the viewer session is his).

| Gate | Question | Token |
|---|---|---|
| B822-G1 | Live smoke: `POST /obix/config/…/applySetpoint` with `<real val=".."/>` INVOKES and the oBIX login user's `OPERATOR_INVOKE` gates it. Settles the phase-2 additive action path (not needed while the child-leaf PUT is sufficient). | `[ev: corpus B822]` |
| B827-G2 | Live: a routed `defaultAlarmClass` alarm reaches the PANCCADIA console + the DashboardPan panel end-to-end (= B821-G2 tier-1 confirm). Required to close the alarm PoC (S18/S13). | `[ev: corpus B827]` |
| B828-G2 | Live confirm of the frozen-enum behavior on a NEW-module deploy (range serialization, no missing-class on a NON-linked HOA). Informational for S20 (`rotationMode` is a new, non-linked enum). | `[ev: corpus B828]` |
| B829 live | End-to-end: surface-A oBIX PUT audits to the station login user; surface-B servlet real-Context `parent.set` produces an `AuditEvent` in `/PANCCADIA/AuditHistory`; the two trails merge on the shared schema. Required to close S12-A/S12-B. | `[ev: corpus B829]` |

Chain-start confirmations (resolve at apply orient): the exact K-number for the conventions delta (`[INFER]` K22 from the close retro); S12-B's `DashboardWriteGuards` seam authored at the client tip the chain builds on (re-anchor at `a109249`).

---

## 7. Research Gate Answer

**Research**: UNSELECTED — already executed via research-sdd lanes (blocks B820-B829, the S12 plan, the slot-type doctrine, the live oBIX probe record, the viewer echo of tunnel main 9acb47c). No further research-sdd iteration before propose.

**Product decisions**: CONFIRMED (§1.1 items 1-7). All seven carry user-explicit authorization; none re-open.

**Evidence completeness**: all load-bearing claims carry `[CERT]`/`[CERT-live]`/`[INFER]` markers. S12-A upgraded to `[CERT via viewer echo]`. The only `[INFER]` among chain-start confirmations is the K22 exact K-number — not a propose blocker.

**Next recommended**: `sdd-propose`
