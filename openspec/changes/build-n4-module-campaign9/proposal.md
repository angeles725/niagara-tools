# Proposal: build-n4-module-campaign9

**Status**: proposal · **Phase**: propose (post-explore, pre-proposal handoff CONFIRMED)
**Source**: niagara-tools `v0.19.0` (main `f3e86f6`, C8 close at `1109c0f`) · **Target**: kit `v0.20.0` (MINOR)
**Client**: `angeles725/niagara-panccadia-leon` — ColdRoomPan-rt 2.0.7, CompPan-rt 2.0.3, DashboardPan rt/ux 2.1.1
**Tunnel**: `pancaddia-leon-tunnel` main `9acb47c`
**Inputs**: `openspec/changes/build-n4-module-campaign9/explore.md` (gate-passed, §1.1 items 1-7 are constraints) · research B820-B829 · S12 plan `1ecdf437c`/`80adc279e` · slot-type doctrine draft · `retros/2026-09-06-campaign8-close-process-meta-lessons.md` (lessons 1-11)
**Topic key**: `sdd/build-n4-module-campaign9/proposal`
**Delivery**: auto-chain, three fixed waves, 13 PRs · review budget 400 changed lines/PR

---

## 1. Intent

Campaign 8 gave the kit eyes on the deployed station (bog audit, console triage, snapshot, structure/write-path lints). Campaign 9 spends that visibility on the two things the operator can actually feel, and on the one process failure that nearly shipped three blind gates.

**First, the compressors wear unevenly.** `CompressorControl` rotates only at stage events — stage UP picks the least-hours available unit (`:226`), stage DOWN drops the most-hours running unit (`:238`). Under steady demand (`onCount == target`) there is no stage event, so the same compressors run for as long as demand holds and the hours ledger diverges with nothing to correct it. The client asked for time-slice rotation and it is the first client PR of the campaign. `[ev: corpus S20]`

**Second, nobody can answer "who changed this setpoint".** The write-server verifies the operator's JWT and then discards the identity; tunnel main `9acb47c` now inserts a best-effort `public.change_log` row per `/write`, but with JWT-bearer only — no step-up, no explicit logout, no `result`, no session identity. On the station side the picture is worse: the DashboardPan servlet writes with a `null` Context, and `ComplexSlotMap.set:662` gates the `AuditEvent` on `context.getUser() != null`, so the servlet write is **not audited at all** — suppressed, not merely unattributed, even with `AuditHistoryService` installed (B829-G1 CLOSED). The client's ask is one sentence — who / what / old→new / when, with an explicit logout — and today no trail answers it. This is the URGENT wave. `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]`

**Third, every protection trip is silent.** A grep of `ColdRoomPan-rt/src` + `CompPan-rt/src` for `BAlarmSourceExt|BAlarmRecord|BAlarmService` returns ZERO at `fbe9009`. Every protection tops out at a plain `SUMMARY` slot (tier 2); the alarm console (tier 1) is entirely unused, so a freeze trip or a low-suction vacuum event reaches nobody. B827 settled how to fix it legally — two authoring patterns, both schema-SAFE. `[ev: corpus B821 §821.4]` `[ev: corpus B827]`

**Fourth, the C8 close proved the kit's own gates can be fixture-green and real-red.** PR18/PR19/PR20 passed their fixtures and stayed silent on the real client trees; a presence-only smoke pin sat green on `5e21f0e` while the real bog produced CHECK14 ×47 and CHECK19 ×16. The durable fix — ONE module-root/profile convention and smoke pins that assert count + subject + absence — is doctrine work this campaign folds, not advice. `[ev: retro campaign8-close-process-meta-lessons lesson 11]`

---

## 2. Scope

### 2.1 In scope — wave 1 (S20 first, then the two mutation-proven kit lints)

| R-id | Item | Repo | Why now |
|---|---|---|---|
| **R1** | **S20** CompPan time-slice rotation — additive `rotationInterval` (`BRelTime`) + `rotationMode` (enum) on `BCompressorControl`; logic in `CompressorControl.step` after target computation | client | User-explicit FIRST client PR; additive/schema-SAFE; the wear problem is live today |
| **R2** | **S7** demand-in-scope lint (`toolbelt/lint-demand-scope.sh`) | kit | RED authored and mutation-proven (DS2 OBSERVED) |
| **R3** | **S18-lint** silent-protection lint (`toolbelt/lint-silent-protection.sh`) | kit | RED authored and mutation-proven (SP8 OBSERVED); it is the static half of the alarm gap |

### 2.2 In scope — wave 2 (URGENT — S12 config login + unified write audit)

| R-id | Item | Repo | Why now |
|---|---|---|---|
| **R4** | **S12-A.1** `buildServer(cfg,deps)` seam + `POST /config/login` step-up + `/config/logout` + token gate on `/write` and `/alarms/ack` | tunnel | The client's ask; RED authored |
| **R5** | **S12-A.2** canonical audit schema — extend `public.change_log`, write `config_session`/`result`/`ts`/`surface`, capture `old` by a pre-write GET, local spool on insert failure | tunnel | Makes the 9acb47c best-effort row into a real trail |
| **R6** | **S12-B** `DashboardWriteGuards.evaluate` seam + B829-G2 real-Context `parent.set(prop, toSet, cx)` | client | Turns the servlet write from unaudited into Niagara-audited |
| **R7** | **S12-C** reconciliation contract + flag-gated AuditHistory→`change_log` mirror (unit-tested against a recorded fixture) | tunnel + kit doc | The word "unified" has to mean one queryable table |

### 2.3 In scope — wave 3 (alarm PoC + remaining lints + doctrine + close)

| R-id | Item | Repo | Why now |
|---|---|---|---|
| **R8** | **S18/S13** alarm PoC Pattern A — CR-3 freeze: child `BBooleanPoint` + `BAlarmSourceExt` + `BBooleanChangeOfStateAlgorithm` | client (ColdRoomPan-rt) | B827 §827.3 copy-ready; schema-SAFE additive |
| **R9** | **S18/S13** alarm PoC Pattern B — CP-1 low suction: `BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` on the offnormal EDGE | client (CompPan-rt) | B827 §827.4; one source scales to CompPan's ~5 trips |
| **R10** | **S19** ext-writable-shape lint | kit | RED authored (EW1-EW10, EW10 = real `BRoomPanel.setpoint` WARN) |
| **R11** | **S5-cont** write-path matrix rows W14-W22 against the real writable set | kit | `lint-write-path.sh` shipped in C8 PR19 and has no rows past W13 |
| **R12** | **Doctrine fold** — slot types for externally written values, alarm authoring patterns A/B, BUILD-LOOP §5 one-module-root, METHODOLOGY K22 | kit | The durable half of R3/R8/R9/R10 and of C8 lesson 11 |
| **R13** | **Close** — `tests/c9-close.bats` (`C9_CLOSE=1`), VERSION `0.20.0`, CHANGELOG, retro fold to pending = 0, BUILD-STATE envelope | kit | Hard close gate (BUILD-LOOP §7) |

### 2.4 Out of scope — explicit

| Item | Reason |
|---|---|
| **Setpoint RETYPE** (`BStatusNumeric`↔`BDouble` on `BRoomPanel.setpoint`) | REJECTED by the user. The B800 retype is an OUTAGE class proven live 2026-09-03 (`Cannot load station`). The shipped write form is the child-leaf ORD `…/setpoint/value` bare `<real>` PUT (B826-G1/G2 `[CERT-live]`); the wrapped-`obj`-to-parent-slot form (B825) is the fallback only. `[ev: corpus B826]` `[ev: corpus B800 §800.8]` |
| **`airDefrost` module flag** (rooms 1/2/4) | STATION-ONLY client trial this cycle — OR of `evapOut`/`resistanceOut` into the fan relay via links, produced by companero with bog-nav outside SDD. No module change in C9; the flag stays a deferred seed. `[ev: corpus S16]` |
| **S15 relay `fallback=false` on the 22 null-fallback relays** | Station-side wiring fix, not a module change. `bog-audit` CHECK11 already reports it. `[ev: corpus B810 §810.8]` |
| **Any live station WRITE** | Only with Cristian's direct authorization and routed through him — the viewer session is his. Requires-execution gates (B822-G1, B827-G2, B829 live) are paired read-only-first and never block a PR. |
| S1-S4, S6, S8-S11, S14, S17 seeds | Not selected this campaign; they stand for C10. |

---

## 3. Capabilities

> `openspec/specs/` does not exist in this repo. Per the campaign-6/7/8 convention, `sdd-spec` writes one change-local `spec.md`.

### New Capabilities

- `module-demand-scope-lint`: `lint-demand-scope.sh` contract — a staged process must decide WHETHER to run (a zero-demand idle branch) before HOW; FAIL a `step`/staging body that computes a target with no zero-demand short-circuit; rows `FAIL|WARN <file>:<line> <reason>`; exits 0/1/3 (K20 disjoint). `[ev: corpus B820]`
- `module-silent-protection-lint`: `lint-silent-protection.sh` contract — WARN a protection trip whose only surface is a private field or a bare `SUMMARY` slot with no `BAlarmSourceExt`/`BIAlarmSource`; effect-slot exemption, pure-model→adapter follow, name allowlist; exits 0/1/3. `[ev: corpus B824]`
- `module-ext-writable-shape-lint`: `lint-ext-writable-shape.sh` contract — WARN a `Flags.OPERATOR` complex property (`BStatusNumeric`/`BStatusBoolean`/`BStatusEnum`) with no writing action; plain `double`/`boolean`/`BRelTime`, or complex-with-`@NiagaraAction`, is clean. `[ev: corpus B823]`
- `compressor-time-slice-rotation` *(client)*: `rotationInterval`/`rotationMode` contract — swap a running compressor after a continuous run ≥ `rotationInterval` for the idle available unit with the least hours; gated by `minOn`/`minOff`/`stageDelay`; HOA OFF excluded, HAND untouched, no swap on `dischargeHigh`, no swap below the LP floor, no swap with a single available unit; `rotationInterval = 0` is byte-identical to today. `[ev: corpus S20]`
- `write-config-step-up` *(tunnel)*: `/config/login` mints a server-held short-TTL token bound to `(email + purpose="config-write")` with sliding inactivity expiry; `/config/logout` revokes immediately; `/write` and `/alarms/ack` gate on it; read endpoints do not. NO JSON user+password store. `[ev: corpus S12]` `[ev: corpus B803 §803.6]`
- `unified-write-audit` *(tunnel + client)*: one canonical `public.change_log` schema written by both surfaces (§5); `old` captured by a pre-write GET; an audit-append failure NEVER fails the write. `[ev: S12 plan 1ecdf437c]`
- `protection-alarm-surface` *(client)*: patterns A and B both route a `BAlarmRecord` with `sourceState = offnormal`, which the DashboardPan alarm bql (`BDashboardServlet.java:502`) already selects. `[ev: corpus B827]`

### Modified Capabilities

- `kit-module-report`: `report-module.sh` gains member rows for the three new lints; a member FAIL must surface as an aggregate FAIL.
- `build-n4-module-kit-doctrine`: `types/logic-authoring.md` gains "Slot types for externally written values"; `types/logic.md` gains alarm authoring patterns A/B; `types/dashboard.md` gains the slot-type one-liner; `BUILD-LOOP.md` §5 gains the ONE module-root/profile convention; `METHODOLOGY.md` gains K22.
- `module-write-path-matrix`: `docs/write-path-matrix.md` extends to rows W14-W22 against the real writable set.
- `dashboard-servlet-write` *(client)*: the servlet write moves from `parent.set(prop, toSet, null)` to a real request-user Context and gains an explicit `DashboardWriteGuards.evaluate` seam.

---

## 4. Approach — three waves of chained PRs, RED first

Wave order is fixed by the user. Each PR is one destination-file group. Every code PR merges its QA RED branch first and goes green against it (CONTRIBUTING §2); doc-only PRs carry zero tests by design. Workers write only inside their worktree (K12); RED branches are cited by branch with the tip re-read at apply (K13); commit bodies are grepped for attribution trailers before publishing (K11). Parallel branches off the same base rebase onto main BEFORE the QA ping, and the lead verifies `git log -1` equals the blessed tip BEFORE settling the ledger (C8 lesson 10).

| # | R-id | Branch / work unit | Repo | RED branch + tip | Est. changed | Lead gate (must all pass before merge) |
|---|---|---|---|---|---|---|
| **PR1** | R1 | `feat/c9-comppan-rotation` | client (CompPan-rt) | **RED to be authored** — `qa/c9-comppan-rotation` (QA lane, RED-first) | ~300 (incl. generated slot block) | `rotationInterval=0` golden byte-identical vs pre-change `step` trace; pins swap-after-interval / NO-swap-below-interval / NO-swap-inside-`minOff` / make-before-break ordering / single-available-no-swap / hours-ledger-unaffected, each with count + subject; OBSERVED mutation flip; `schema-risk.sh` = SAFE against the real bog; `vendorVersion` 2.0.3 → **2.1.0** |
| **PR2** | R2 | `feat/c9-demand-scope` | kit | `qa/c9-demand-in-scope` **`2916954`** `[CERT]` | ~200 | DS1-DS7 + DS-smoke green; OBSERVED DS2 flip; real-tree smoke on ALL four client module roots with exact counts + subjects + absence; K19 routing in BUILD-LOOP §5 + SKILL; `kit-links.bats` green; shellcheck 0; 0 trailers |
| **PR3** | R3 | `feat/c9-silent-protection` | kit | `qa/c9-silent-protection` **`e38e503`** `[CERT]` | ~220 | SP1-SP8 + SP-smoke; OBSERVED SP8 flip; real-tree smoke flags CP-1 and CR-3 and does NOT flag CP-2/`defrostSkipped` (absence pin); K19 routing; `kit-links.bats`; 0 trailers |
| **PR4** | R4 | `feat/c9-s12-config-login` | tunnel (rebase onto `9acb47c`) | `qa/c9-s12-write-server` **`24adcba`** (base `e4b42b0`) `[CERT via viewer echo]` | ~280 | S12A-1..S12A-7 green ON the rebased tip; the existing `change_log` insert on `9acb47c` still passes; OBSERVED flip = dropping the config-token check flips S12A-1/S12A-5; no user+password store anywhere in the diff |
| **PR5** | R5 | `feat/c9-s12-audit-schema` | tunnel | extends `qa/c9-s12-write-server` (QA appends the schema pins RED-first) | ~220 | Named pin **audit-append failure never fails the write** (write lands 200 + an error row + a spool entry); `old` comes from a pre-write GET, not from the request; exact-count pin = exactly ONE row per successful write; migration is additive-only (no column drop/retype) |
| **PR6** | R6 | `feat/c9-s12-servlet-guards` | client (DashboardPan-ux) | `qa/c9-s12-servlet` **`4c18837`** (rebased `a109249`) `[CERT]` | ~260 | guard1-5 + guard4b + success/one-audit-entry (7 pins); guard4 = no-silent-zero regression; `parent.set(prop, toSet, cx)` carries the real request user; `schema-risk.sh` = SAFE; `lint-servlet.sh` clean; audit-fail-never-fails-the-write pin; DashboardPan 2.1.1 → **2.2.0** |
| **PR7** | R7 | `feat/c9-s12-audit-mirror` | tunnel + kit doc | RED to be authored — mirror unit tests against a recorded AuditHistory fixture | ~180 | Mirror is flag-gated OFF by default; dedupe key proven idempotent over a replayed fixture; the reconciliation contract is documented in the kit with `[ev:]` per row; enabling it live is the B829-live gate, never a PR gate |
| **PR8** | R8 | `feat/c9-alarm-cr3` | client (ColdRoomPan-rt) | **RED to be authored** — `qa/c9-alarm-cr3` from B827 §827.3 | ~260 | Alarm fires on the freeze trip and clears on recovery; `sourceState = offnormal`; `schema-risk.sh` = SAFE (additive child point); OBSERVED flip; ColdRoomPan-rt 2.0.7 → **2.1.0** |
| **PR9** | R9 | `feat/c9-alarm-cp1` | client (CompPan-rt) | **RED to be authored** — `qa/c9-alarm-cp1` from B827 §827.4/§827.6 | ~280 | EDGE-only pin (B827-G1): fires once on normal→offnormal, `toNormal` on recovery, `started()` re-seeds `wasOffnormal[]`, no re-fire across repeated executes; `schema-risk.sh` = SAFE; CompPan-rt 2.1.0 → **2.2.0** |
| **PR10** | R10 | `feat/c9-ext-writable-shape` | kit | `qa/c9-ext-writable-shape` **`3726722`** `[CERT]` | ~230 | EW1-EW10 green; EW10 = the real `BRoomPanel.setpoint` WARN with subject asserted; real-tree smoke on all four module roots; K19 routing; `kit-links.bats` |
| **PR11** | R11 | `docs/c9-write-path-w14-w22` | kit | `qa/c8-write-path` **`5e357d1`** `[CERT]` (extends to W14-W22) | ~150 | `lint-write-path.sh` exits 0 on every client module root after the rows land, and exit **3** (never a silent 0) on a root with no matrix; each row cites the real writable slot |
| **PR12** | R12 | `docs/c9-doctrine` | kit | none (doc-only) | ~200 | `sweep-fold-audit.sh --strict` green; `kit-links.bats` green; one `[ev:]` per paragraph/row; every folded lesson names its retro anchor |
| **PR13** | R13 | `chore/c9-close` | kit | none | ~180 | `tests/c9-close.bats` green under `C9_CLOSE=1`; `sweep-build-state.sh` green; `retros/INDEX.md` pending = 0; VERSION `0.20.0` + CHANGELOG per CONTRIBUTING §4-5; 0 attribution trailers in the whole PR range |

**Method invariants.** (1) Every new rule is proven RED against a real-shape fixture copied from the operator's modules with sanitized names — never an invented snippet. (2) Every named mutation is recorded as an **OBSERVED** flip with verbatim RED-then-GREEN output; a "would flip" prose claim is not evidence (C8 lesson 7). (3) Every smoke pin asserts **exact count + subject + absence**; a smoke that cannot run is a BLOCKER, never an advisory (C8 lesson 11a/11b). (4) Every lint iterates profiles from the module root; a root with no sources or no matrix exits **3**, never a silent 0 (C8 lesson 11c). (5) K19 — every new script is named in both `BUILD-LOOP.md` and `skill/SKILL.md` in the PR that lands it. (6) The four always-conflict files (BUILD-LOOP routing line, SKILL toolbelt line, `retros/INDEX.md` row, `BUILD-STATE.md` envelope) merge by FRAGMENT: append, keep both rows, dedupe by script name, never overwrite; merge after each wave before opening the next (C8 lesson 2). (7) Drafts land in a repo path (`sources/probes/`), never only in `/tmp` (C8 lesson 4).

**Versions**: kit `VERSION` + `CHANGELOG.md` → **`0.20.0`** in PR13. Client `vendorVersion` bumps per PR: CompPan-rt 2.0.3 → 2.1.0 (PR1, additive slots) → 2.2.0 (PR9, additive alarm source); ColdRoomPan-rt 2.0.7 → 2.1.0 (PR8, additive child point); DashboardPan 2.1.1 → 2.2.0 (PR6). Every slot-touching bump runs `schema-risk.sh` first (BUILD-LOOP §4.b/§6). **Gate exit**: every kit PR is a kit-changing push → close-gate exit (a) NEW RETRO, except PR12 which is exit (c) PROMOTION with its `retros/INDEX.md` anchor.

---

## 5. Decision — audit-sink reconciliation (S12)

**Recommendation: ONE canonical sink — Supabase `public.change_log`, extended in place. Drop the JSON-lines file as a primary trail.**

| Aspect | Decision | Why |
|---|---|---|
| Canonical table | `public.change_log` (already live on tunnel `9acb47c` with `user_email, user_id, room, slot, label, old_value, new_value, area, ok`) | It already exists, already receives every successful `/write`, and is already reachable from the viewer. Introducing a second `audit` table would split the very trail the client asked to unify, and would orphan the rows written since `9acb47c`. `[CERT via viewer echo]` |
| Additive columns | `ts` (server time), `config_session` (the step-up session id), `result` (station HTTP status), `surface` (`write-server` \| `servlet`), `client_ip` | Exactly the S12 plan's `{ts, email, ord, old, new, result, ip, config_session}` mapped onto the existing column names. Additive-only: no drop, no retype — the same discipline the module side owes `schema-risk`. `[ev: S12 plan 1ecdf437c]` |
| `old` value | Captured by a pre-write GET of the slot, never taken from the request body | The audit must record the true prior value; a client-supplied `old` is a claim, not a reading. `[ev: S12 plan 1ecdf437c]` |
| JSON-lines file | Demoted from a parallel trail to a **local failure spool**: written only when the `change_log` insert fails, drained on the next successful insert | Two independent primary trails guarantee they will disagree. One canonical table + a replay spool keeps a single answer while losing nothing during a Supabase outage. |
| Audit-append failure | The write LANDS; an error row (`ok=false`, `result` carrying the station status) plus a spool entry is recorded; the endpoint still returns the station's outcome | Named pin in both R5 and R6. A RED that fails the write on audit-fail tests the wrong contract. `[ev: corpus S12]` |
| Surface B reconciliation | The servlet's authoritative station-side record is the **native** `AuditEvent` produced by the real-Context `parent.set` (B829-G2) landing in `/PANCCADIA/AuditHistory`. It reaches `change_log` through the mini-PC by a **flag-gated mirror job** (R7) that reads AuditHistory and inserts rows with `surface='servlet'` and a dedupe key of `(ts, user, target, old, new)` | The station must not gain a new outbound internet dependency, and the servlet must not own a Supabase credential. The mirror also survives the servlet failing, because it reads the trail Niagara itself wrote. `[ev: corpus B829]` |
| `config_session` on surface B | NULL in this campaign, honestly | Surface-B step-up (S12 plan Part 3 — re-auth modal + real `x-niagara-csrfToken`) is NOT in C9. Surface B's identity comes from the Niagara request user via the real Context; the session column stays empty rather than being faked. |
| Answer to the client's ask | `SELECT` on one table: `user_email` (who) · `room`/`slot`/`label` (what) · `old_value`→`new_value` (old→new) · `ts` (when) · `config_session` + `/config/logout` revocation (explicit session boundary) | One query, both surfaces, no manual merge. |

`schema-risk.sh` has no jurisdiction over a Postgres table, so R5 carries its own additive-only migration pin: the migration adds columns and never drops or retypes one.

---

## 6. Doctrine deltas this campaign folds

| Delta | Target § | Evidence token |
|---|---|---|
| Slot types for externally written values — a slot external clients write must be a SIMPLE value or carry a writing `@NiagaraAction`; a bare `Flags.OPERATOR` complex either rejects the write or silently writes a default | `types/logic-authoring.md` §"Slot types for externally written values" + a one-liner in `types/dashboard.md` | `[ev: corpus B823]` `[ev: corpus B826]` |
| Alarm authoring patterns A (declarative child point + `BAlarmSourceExt`) and B (`BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` on the offnormal EDGE); both route `sourceState = offnormal`; both are schema-SAFE | `types/logic.md` §"Protection anatomy" | `[ev: corpus B827]` `[ev: corpus B821 §821.4]` |
| ONE module-root/profile convention — module root = the directory containing the profiles; every lint ITERATES the profiles; a root with no sources or no matrix is an ERROR (exit 3), never a silent 0 | `BUILD-LOOP.md` §5 | `[ev: retro campaign8-close-process-meta-lessons lesson 11c]` |
| **K22** — a real-tree smoke on every client module root is part of the lead gate, and a smoke pin asserts exact counts AND subjects AND absence; a skipped smoke BLOCKS, never downgrades to advisory | `METHODOLOGY.md` §Kit maintenance | `[ev: retro campaign8-close-process-meta-lessons lesson 11a/11b]` |
| Mutation tables record OBSERVED flips (verbatim RED-then-GREEN), never "would flip" prose | `METHODOLOGY.md` §Kit maintenance | `[ev: retro campaign8-lint-timers-ext]` |
| Four always-conflict files + the fragment-merge rule as the standing multi-session merge protocol | `METHODOLOGY.md` §Multi-session (extends K12) | `[ev: retro campaign8-close-process-meta-lessons lesson 2]` |
| Lead merge/settle order — merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger; rebase a parallel branch before the QA ping | `BUILD-LOOP.md` §7 | `[ev: retro campaign8-close-process-meta-lessons lesson 10]` |
| Unified write-audit reconciliation contract (§5) — one canonical sink, two writers, dedupe key, audit-fail never fails the write | `types/dashboard.md` + `BUILD-LOOP.md` §6 note | `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]` |

One `[ev:]` token per paragraph/row is mandatory; `sweep-fold-audit.sh --strict` enforces it at PR12.

---

## 7. Affected Areas

| Area | Impact | Description |
|---|---|---|
| `build-n4-module-kit/toolbelt/{lint-demand-scope,lint-silent-protection,lint-ext-writable-shape}.sh` | New | Three static lints (R2, R3, R10) |
| `build-n4-module-kit/toolbelt/report-module.sh` | Modified | Three new member rows; member FAIL → aggregate FAIL |
| `build-n4-module-kit/{BUILD-LOOP.md,METHODOLOGY.md,types/logic.md,types/logic-authoring.md,types/dashboard.md,skill/SKILL.md}` | Modified | §6 doctrine deltas + K19 routing |
| `build-n4-module-kit/docs/write-path-matrix.md` | Modified | Rows W14-W22 (R11) |
| `tests/*.bats` + `tests/fixtures/**`, `tests/c9-close.bats` | New/Modified | Per-lint suites + the `C9_CLOSE=1` close gate |
| `build-n4-module-kit/retros/INDEX.md`, `BUILD-STATE.md`, `VERSION`, `CHANGELOG.md` | Modified | One retro per kit-changing push; `0.19.0 → 0.20.0` |
| client `CompPan-rt/src/**/{BCompressorControl,CompressorControl}.java` + `srcTest` | Modified | R1 rotation slots + logic; R9 `BIAlarmSource` |
| client `ColdRoomPan-rt/src/**/BEvaporatorUnit*.java` + `srcTest` | Modified | R8 child point + `BAlarmSourceExt` |
| client `DashboardPan-ux/src/**/{BDashboardServlet,DashboardWriteGuards}.java` + `srcTest` | New/Modified | R6 guard seam + real-Context `set` |
| tunnel `instalacion/pipeline/{write-server.mjs,test/**}` + migration | Modified/New | R4 step-up, R5 canonical schema + spool, R7 mirror |

---

## 8. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| RK1 | A new lint is fixture-green and real-red (C8 lesson 1/11) | High | Every code PR carries a real-tree smoke on ALL four client module roots with count + subject + absence pins; a lead-found defect becomes a `mktemp` pre-fix RED pin before the fix lands |
| RK2 | R1 and R8/R9 have **no authored RED yet** | High | RED-first is a hard precondition: QA authors and mutation-proves (OBSERVED flip) before apply opens. If a RED is not blessed, its PR does not open — it does not proceed on fixtures |
| RK3 | The S12-A RED sits on `e4b42b0` while tunnel main is `9acb47c`; the existing `change_log` insert may break the suite | Med | Rebase onto `9acb47c` BEFORE apply (K13: re-read the tip), then re-bless; the `buildServer(cfg,deps)` seam contract is unchanged by the rebase |
| RK4 | R1 exceeds the 400-line budget once the generated slot block is counted | Med | Split the generated slot block into its own commit; if the PR still exceeds 400 changed lines, open it with a per-PR `size:exception` justified by the generated-code share. Authored risk stays ~180 |
| RK5 | Pattern B (R9) is implemented level-triggered and re-fires every execute (B827-G1) | Med | The RED encodes `wasOffnormal[]` per-trip state + `toNormal` on recovery + `started()` re-seed; the edge-only pin is named, not implied |
| RK6 | The real-Context `parent.set` (R6) changes servlet audit behavior in a way only a live session can confirm | Med | `schema-risk.sh` SAFE pin plus the off-station guard suite land in C9; the end-to-end `/PANCCADIA/AuditHistory` confirm is the B829-live gate, paired read-only-first with Cristian — it gates the mirror's live enablement, never PR6 |
| RK7 | The AuditHistory mirror (R7) double-writes rows on replay | Med | Idempotent dedupe key `(ts, user, target, old, new)` proven over a replayed fixture; the mirror ships flag-gated OFF |
| RK8 | HOA frozen-enum retrofit trap — applying B828 frozen-enum doctrine to a cross-module-LINKED HOA deletes the linked type (live "Missing class") | Low | Frozen-enum is NEW-modules-only. `rotationMode` is a new, non-linked enum. Any C9 work touching an existing HOA slot audits for cross-module links first `[ev: corpus B828 §828.7]` |
| RK9 | Stale-tree client cites (B821 read v2.0.0 while main was `fbe9009`; B823 cited `deed38c`) | Med | Re-anchor every load-bearing client cite at the actual tip the chain builds on before promoting to `[CERT]` |
| RK10 | Parallel worktrees conflict on the four always-conflict files every kit PR | High | Fragment-merge rule (append, keep both rows, dedupe by script name, never overwrite); merge after each wave before opening the next |
| RK11 | The ledger settles on a reported-but-unverified merge (C8 lesson 10) | Med | BUILD-LOOP §7 order: merge ff-only → verify `git log -1` equals the blessed tip → THEN settle |
| RK12 | A wave-3 RED is not authored before close | Med | R8/R9 roll to C10 without holding the kit version bump — PR13 depends on R10/R11/R12 only |

---

## 9. Rollback Plan

| Slice | Rollback |
|---|---|
| PR2 / PR3 / PR10 | `git revert` — new script + tests + fixtures on new paths; nothing references them until the report-module rows land |
| PR11 / PR12 / PR13 | `git revert` — doc-only; retro files are never deleted (propose-never-apply); `retros/INDEX.md` rows revert to `pending`, VERSION returns to `0.19.0` |
| PR1 / PR8 / PR9 (client, slot-touching) | `git revert` + rebuild + redeploy the previous jar. Because every slot is ADDITIVE, reverting is a slot REMOVE against a station bog that now holds the new slots — so a revert-after-deploy runs `schema-risk.sh` first and, if the verdict is LOSSY/OUTAGE, the safe rollback is `rotationInterval = 0` / the alarm ext disabled, NOT a jar downgrade. This is the reason `rotationInterval = 0` must be byte-identical to today |
| PR6 (client, no slot change) | `git revert` — schema-neutral; the servlet returns to the null-Context write |
| PR4 / PR5 / PR7 (tunnel) | `git revert` the server code; the `change_log` migration is additive-only and is LEFT in place (dropping a column is the destructive direction). Feature flags off = the mirror stops; the step-up gate reverts to JWT-bearer-only |

No station write, no operator data mutation, and no live jar deploy is performed by any SDD slice. Deploys stay with the operator under BUILD-LOOP §6.

---

## 10. Dependencies

- `bats-core`, `shellcheck 0.10.0` (pinned in `ci.yml`), the live pre-push hook — installed.
- Research landed: **B820** (demand-in-scope), **B821** (protection anatomy), **B822** (additive setpoint), **B823** (no-code channels), **B824** (silent-protection), **B825/B826** (`[CERT-live]` child-ORD write form), **B827** (alarm authoring), **B828** (frozen enum), **B829** (audit trail, G1 CLOSED). No further research-sdd iteration before propose.
- QA RED branches, blessed at the tip the chain builds on: `qa/c9-demand-in-scope` `2916954`, `qa/c9-silent-protection` `e38e503`, `qa/c9-ext-writable-shape` `3726722`, `qa/c9-s12-servlet` `4c18837`, `qa/c9-s12-write-server` `24adcba` (rebase to `9acb47c`), `qa/c8-write-path` `5e357d1`. **To be authored**: `qa/c9-comppan-rotation`, `qa/c9-alarm-cr3`, `qa/c9-alarm-cp1`, plus the R5 schema pins and the R7 mirror pins.
- Real trees for smokes: all four client module roots at the chain's client tip (`a109249` or later) — ColdRoomPan-rt, CompPan-rt, DashboardPan-rt, DashboardPan-ux.
- Requires-execution gates, none of which block a PR: **B822-G1** (`applySetpoint` invoke — informational, phase 2 only), **B827-G2** (routed alarm reaches the console + panel — closes the alarm PoC), **B828-G2** (frozen-enum on a NEW non-linked deploy — informational for `rotationMode`), **B829 live** (both trails merge on the shared schema — gates the R7 mirror's live enablement). All run read-only-first, only with Cristian's direct authorization, routed through him.
- Chain-start confirmation: the exact K-number for the conventions delta is `[INFER]` K22 — settle at apply orient.

---

## 11. Success Criteria

- [ ] **SC-1** — `CompressorControlTest` proves `rotationInterval = 0` is **byte-identical to the pre-change behavior**: the `step` command trace over the recorded demand sequence is equal byte-for-byte to the golden captured before the change. Named pin.
- [ ] **SC-2** — rotation pins green with count + subject: swap after the interval, NO swap below it, NO swap while the idle candidate is inside `minOff`, make-before-break ordering (incoming ON → `stageDelay` → outgoing OFF), no swap with a single available unit, HOA OFF excluded, HAND untouched, no swap on `dischargeHigh`, no swap below the LP floor, `condenserNHours` unaffected by a swap.
- [ ] **SC-3** — **an audit-append failure never fails the write**: with the sink forced to fail, `/write` still returns the station outcome, `change_log` receives an `ok=false` error row (or the spool receives the entry), and the servlet path behaves identically. Named pin in R5 and R6.
- [ ] **SC-4** — `/config/login` mints a token; `/write` and `/alarms/ack` WITHOUT a token → 401; an EXPIRED token → 401; a non-allowlisted ORD → 403; `/config/logout` makes the next write fail; a successful write appends **exactly one** `change_log` row with `old` captured by a pre-write GET. No user+password store exists anywhere in the tunnel diff.
- [ ] **SC-5** — the servlet write carries a real request-user Context (`parent.set(prop, toSet, cx)`); guard1-5 + guard4b green; guard4 no-silent-zero holds; `schema-risk.sh` = SAFE.
- [ ] **SC-6** — the AuditHistory→`change_log` mirror is idempotent over a replayed fixture (no duplicate rows) and ships flag-gated OFF; the reconciliation contract is documented with one `[ev:]` per row.
- [ ] **SC-7** — CR-3 freeze (Pattern A) and CP-1 low suction (Pattern B) each raise a `BAlarmRecord` with `sourceState = offnormal`; Pattern B fires **once** on the normal→offnormal edge, sends `toNormal` on recovery, and re-seeds on `started()`; both verdicts `schema-risk.sh` = SAFE.
- [ ] **SC-8** — each of the three new lints runs on **all four** client module roots and its smoke pin asserts exact count + subject + absence; `lint-silent-protection` flags CP-1 and CR-3 and does not flag CP-2/`defrostSkipped`; `lint-ext-writable-shape` WARNs the real `BRoomPanel.setpoint` (EW10).
- [ ] **SC-9** — every lint exits **3** (never a silent 0) on a module root with no sources or no matrix; `lint-write-path.sh` exits 0 on every client module root once W14-W22 land.
- [ ] **SC-10** — every code PR records an **OBSERVED** mutation flip (verbatim RED-then-GREEN); no PR merges on a "would flip" claim; no skipped smoke is downgraded to advisory.
- [ ] **SC-11** — `BUILD-LOOP.md` §5 states the one module-root/profile convention; `METHODOLOGY.md` carries K22; `types/logic.md` carries alarm patterns A/B; `types/logic-authoring.md` carries the slot-type doctrine; `kit-links.bats` green with every new script routed in both `BUILD-LOOP.md` and `skill/SKILL.md` (K19).
- [ ] **SC-12** — `tests/c9-close.bats` green under `C9_CLOSE=1`; `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` green; `retros/INDEX.md` pending = 0; `VERSION` = `0.20.0` with a CHANGELOG entry per CONTRIBUTING §4-5; `shellcheck` exit 0; **no commit body in the whole PR range carries an attribution trailer**.
- [ ] **SC-13** — client `vendorVersion` bumps landed and schema-risk-cleared: CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan 2.2.0.

---

## 12. Review Workload Note

`delivery_strategy` = **auto-chain**.
`Decision needed before apply: No`
`Chained PRs recommended: Yes` — three fixed waves, 13 slices; wave 1 = PR1-PR3, wave 2 = PR4-PR7, wave 3 = PR8-PR13. Merge each wave before opening the next.
`400-line budget risk: Medium` — every kit PR is under 400 by construction (≤230 changed). The client PRs are the pressure: PR1 (~300 with the generated slot block), PR9 (~280), PR6 (~260), PR8 (~260). PR1 is the only likely `size:exception`, justified per-PR by the generated-code share; the generated slot block lands as its own commit so the authored diff stays reviewable.

---

## 13. Next Phases

- `sdd-spec` — wave 1 first: the rotation contract (gates, ordering, the `0 = disabled` byte-identity requirement), the demand-in-scope and silent-protection verdict domains and row grammar. Then the S12 surface contracts (`/config/login`, `/config/logout`, the token gate, the canonical `change_log` schema and its writers) and the alarm authoring contracts for both patterns.
- `sdd-design` (parallel with `sdd-spec`) — the `cmdSince[]`-based rotation clock and make-before-break sequencing; the `buildServer(cfg,deps)` seam and token store; the migration and spool mechanics; the mirror dedupe key; fixture layout and sanitization for the real-shape client fixtures; the three-wave chain and fragment-merge mechanics.
- Then `sdd-tasks` → `sdd-apply` (RED-first on every code slice) → `sdd-verify` → `sdd-archive`.
