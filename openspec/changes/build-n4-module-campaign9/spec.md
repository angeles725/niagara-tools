# Spec: build-n4-module-campaign9

**Status**: spec · **Source**: v0.19.0 (kit `1109c0f`) · **Target**: v0.20.0
**Topic key**: `sdd/build-n4-module-campaign9/spec`
**Based on**: proposal.md + explore.md · research B820-B829 · S12 plan `1ecdf437c`/`80adc279e` · slot-type doctrine draft · `retros/2026-09-06-campaign8-close-process-meta-lessons.md` (lessons 1-11)
**Client**: ColdRoomPan-rt 2.0.7, CompPan-rt 2.0.3, DashboardPan rt/ux 2.1.1 · **Tunnel**: `pancaddia-leon-tunnel` main `9acb47c`
**Row format** (all new scripts): `PASS|FAIL|WARN|SKIP  <check>  <file>:<line>  <detail>`
**Exits** (new standalone scripts): `0` no FAIL · `1` any FAIL · `2` WARN-only (where applicable) · `3` no sources or matrix / env / usage (K20 disjoint)

---

## Cross-cutting discipline (all PRs)

| Rule | Requirement |
|------|-------------|
| CD1 | Every kit-changing push range: retro file + `retros/INDEX.md` row flip + `BUILD-STATE.md` self-envelope in the same push range. |
| CD2 | Doc-only PRs (PR12, PR13 close section) carry zero new bats tests by design. |
| CD3 | QA RED branch tip re-read at apply time before merging (K13). |
| CD4 | Every new rule proven RED against a real-shape fixture copied from the operator's modules with sanitized names (CONTRIBUTING §9). |
| CD5 | Every new toolbelt script named in both `BUILD-LOOP.md` and `skill/SKILL.md` in the PR that lands it (K19). |
| CD6 | `shellcheck 0.10.0` exits 0 on every modified or new `toolbelt/*.sh`. |
| CD7 | No commit body carries an attribution trailer (K11). |
| CD8 | Each child PR diff shows only child commits; rebase until clean. |
| CD9 | Every scanner prunes dot-dirs; a hidden directory is never traversed (D9b). |
| CD10 | Real-tree smoke on **all four** client module roots (ColdRoomPan-rt, CompPan-rt, DashboardPan-rt, DashboardPan-ux) with exact count + subject + absence in every code PR (K22). A smoke that cannot run is a BLOCKER, never an advisory. |
| CD11 | Every code PR records an **OBSERVED** mutation flip (verbatim RED-then-GREEN output); "would flip" prose is not evidence. |

---

## PR1 — S20 CompPan time-slice rotation (R1)

**Branch**: `feat/c9-comppan-rotation` | **QA RED**: `qa/c9-comppan-rotation` — **RED to be authored** | **Repo**: `angeles725/niagara-panccadia-leon` (CompPan-rt) | **Version**: 2.0.3 → 2.1.0

| ID | Requirement |
|----|-------------|
| R1.1 | `BCompressorControl` MUST gain two additive `@NiagaraProperty` slots: `rotationInterval` (`BRelTime`, flags `SUMMARY|OPERATOR`) and `rotationMode` (enum, make-before-break default); no existing slot MAY be dropped or retyped. `[ev: corpus S20]` |
| R1.2 | When `rotationInterval = 0`, the `CompressorControl.step` command trace over a recorded demand sequence MUST be **byte-identical** to the pre-change golden captured before R1 was applied (ROT5). `[ev: corpus S20]` |
| R1.3 | After a compressor runs continuously for ≥ `rotationInterval`, the logic MUST select the idle available unit with the least cumulative hours and execute a make-before-break swap: the incoming unit powers ON, `stageDelay` completes, then the outgoing unit powers OFF (ROT4). `[ev: corpus S20]` |
| R1.4 | No swap MAY occur when `rotationInterval > 0` but elapsed time since the last stage event is < `rotationInterval` (ROT2). |
| R1.5 | No swap MAY occur while the idle candidate is inside its `minOff` guard period (ROT3). |
| R1.6 | No swap MAY occur when `dischargeHigh` protection is active (ROT8). |
| R1.7 | No swap MAY occur when the LP floor condition would be violated by removing the running unit (ROT9). |
| R1.8 | No swap MAY occur when only one compressor unit is available (ROT6). |
| R1.9 | A compressor whose HOA switch is in the OFF position MUST be excluded from rotation candidates; a compressor in HAND MUST be left untouched by rotation (ROT7). |
| R1.10 | `condenserNHours` (the cumulative hours ledger) MUST NOT be modified by a rotation swap; hours continue to accumulate from real run time only (ROT10). `[ev: corpus S20]` |
| R1.11 | `schema-risk.sh` MUST return SAFE against the real `config.bog` snapshot for every slot-touching commit in PR1. |

**QA pin contract (RED to be authored on `qa/c9-comppan-rotation`):**

| Pin | Scenario |
|-----|----------|
| `ROT1 swap-after-interval` | GIVEN a two-compressor CompPan fixture where one unit has run ≥ `rotationInterval` and the idle candidate passes all guards; WHEN `CompressorControl.step` executes; THEN the outgoing unit powers OFF and the incoming unit powers ON, and exactly one swap is recorded. |
| `ROT2 no-swap-below-interval` | GIVEN elapsed time since the last stage event is < `rotationInterval`; WHEN `step` executes; THEN no swap occurs and the run sequence is unchanged. |
| `ROT3 no-swap-while-minOff` | GIVEN the idle candidate is inside its `minOff` guard period; WHEN `step` executes; THEN no swap occurs. |
| `ROT4 make-before-break-order` | GIVEN a swap is triggered; WHEN `step` executes; THEN the incoming unit is commanded ON and the `stageDelay` window completes BEFORE the outgoing unit is commanded OFF. |
| `ROT5 disabled-at-0-golden` | GIVEN `rotationInterval = 0`; WHEN `step` executes over the full recorded demand sequence; THEN the command trace is **byte-identical** to the pre-change golden captured before R1 was applied. |
| `ROT6 no-swap-one-available` | GIVEN only one compressor unit is in the available pool; WHEN `step` executes; THEN no swap occurs. |
| `ROT7 hoa-off-excluded-hand-untouched` | GIVEN one unit has HOA = OFF and another has HOA = HAND; WHEN rotation is evaluated; THEN the OFF unit is excluded from candidates and the HAND unit is not touched by rotation logic. |
| `ROT8 no-swap-on-dischargeHigh` | GIVEN `dischargeHigh` protection is active; WHEN `step` executes; THEN no swap occurs regardless of interval elapsed. |
| `ROT9 lp-floor` | GIVEN the LP floor condition would be violated by removing the running unit; WHEN `step` executes; THEN no swap occurs. |
| `ROT10 hours-ledger-unaffected` | GIVEN a rotation swap executes successfully; WHEN `condenserNHours` values are read after the swap; THEN they reflect only real accumulated run time, not a reset or adjustment caused by the swap. |

Each pin MUST carry a named mutation that restores the defective pre-fix behavior and observably flips that pin from GREEN to RED (OBSERVED flip, verbatim output).

Real-tree smoke (CD10): `lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` each run on all four client module roots with exact count + subject + absence after PR1; `schema-risk.sh` = SAFE.

---

## PR2 — lint-demand-scope.sh (R2)

**Branch**: `feat/c9-demand-scope` | **QA RED**: `qa/c9-demand-in-scope` **`2916954`** `[CERT]` | **Repo**: `niagara-tools` | **Pins**: DS1-DS7 + DS-smoke

| ID | Requirement |
|----|-------------|
| R2.1 | `lint-demand-scope.sh` MUST emit `FAIL` for every `step`/staging body that computes a target value with no zero-demand short-circuit (a zero-demand idle branch that returns or continues before the target computation). `[ev: corpus B820]` |
| R2.2 | A `step` body containing a zero-demand guard before the target computation MUST produce no FAIL row (PASS). |
| R2.3 | Row grammar: `FAIL  lint-demand-scope  <file>:<line>  <reason>`; WARN rows follow the same format. |
| R2.4 | Exit codes: 0 = no FAIL; 1 = any FAIL; 3 = no sources found or usage error (K20 disjoint). |
| R2.5 | The script MUST iterate from the module root through all profiles; a root with no Java sources MUST exit 3, never a silent 0. |
| R2.6 | The script MUST prune dot-directories during traversal (D9b). |
| R2.7 | `lint-demand-scope.sh` MUST be named in both `BUILD-LOOP.md` §5 and `skill/SKILL.md` in PR2 (K19). |
| R2.8 | `report-module.sh` MUST gain a member row for `lint-demand-scope`; a FAIL from this lint MUST surface as an aggregate FAIL. |
| R2.9 | Named mutation: removing the zero-demand guard causes DS2 to flip from PASS to FAIL (OBSERVED). |

**DS-smoke** GIVEN all four client module roots (ColdRoomPan-rt, CompPan-rt, DashboardPan-rt, DashboardPan-ux) at the chain's client tip; WHEN `lint-demand-scope.sh` runs on each; THEN the exact count of findings matches the expected subjects (at minimum `CompressorControl.step` is flagged), named absence assertions hold (no false positives on guardian-exempt paths), and no dot-dir is traversed.

**DS1** GIVEN a `CompressorControl.step` pre-fix fixture with no zero-demand short-circuit; WHEN `lint-demand-scope.sh` runs; THEN exits 1, FAIL row at the target-computation site.

**DS2** (mutation pin — OBSERVED at `2916954`) GIVEN the zero-demand guard is removed from the fixture; WHEN `lint-demand-scope.sh` runs; THEN the FAIL row appears — OBSERVED flip verified.

DS3-DS7: per the RED at tip `2916954`; apply agent re-reads the branch tip at apply (K13).

---

## PR3 — lint-silent-protection.sh (R3)

**Branch**: `feat/c9-silent-protection` | **QA RED**: `qa/c9-silent-protection` **`e38e503`** `[CERT]` | **Repo**: `niagara-tools` | **Pins**: SP1-SP8 + SP-smoke

| ID | Requirement |
|----|-------------|
| R3.1 | `lint-silent-protection.sh` MUST emit `WARN` for every protection-trip code path whose only externally visible surface is a private field or a bare `SUMMARY` slot with no `BAlarmSourceExt` or `BIAlarmSource`. `[ev: corpus B824]` |
| R3.2 | A protection path whose output is an effect slot (a slot that itself drives an actuator or external state) MUST be exempted from the WARN. |
| R3.3 | Pure-model classes MUST be followed to their adapter; the adapter is the authoritative surface for the exemption decision. |
| R3.4 | An allowlisted name pattern (configured in the script) MUST suppress the WARN for known-safe protection names. |
| R3.5 | Row grammar: `WARN  lint-silent-protection  <file>:<line>  <reason>`; exit 0 on WARN-only (WARN is not FAIL); exit 1 on any FAIL; exit 3 on no sources / usage. |
| R3.6 | The script MUST iterate from the module root through all profiles; a root with no Java sources MUST exit 3, never a silent 0. |
| R3.7 | The script MUST prune dot-dirs (D9b). |
| R3.8 | `lint-silent-protection.sh` MUST be named in both `BUILD-LOOP.md` §5 and `skill/SKILL.md` in PR3 (K19). |
| R3.9 | `report-module.sh` MUST gain a member row for `lint-silent-protection`; a FAIL from this lint MUST surface as aggregate FAIL. |
| R3.10 | Named mutation: removing the `BAlarmSourceExt`/`BIAlarmSource` check causes SP8 to flip (OBSERVED at `e38e503`). |

**SP-smoke** GIVEN all four client module roots at the chain's client tip; WHEN `lint-silent-protection.sh` runs on each; THEN: `CP-1` (low suction) and `CR-3` (freeze trip) MUST appear in the WARN output (subject-named); `CP-2` and `defrostSkipped` MUST be **absent** from the output (absence pins); exact count matches expected silent-protection findings.

**SP1-SP7**: per the RED at tip `e38e503`; bind at apply (K13).

**SP8** (mutation pin — OBSERVED at `e38e503`): removing the `BAlarmSourceExt` reference check causes SP8 to flip from WARN-present to absent — OBSERVED flip verified.

---

## PR4 — S12-A config login + token gate (R4)

**Branch**: `feat/c9-s12-config-login` | **QA RED**: `qa/c9-s12-write-server` **`24adcba`** (rebase onto **`9acb47c`** at apply) `[CERT via viewer echo]` | **Repo**: `pancaddia-leon-tunnel` | **Pins**: S12A-1..S12A-7

| ID | Requirement |
|----|-------------|
| R4.1 | The write-server MUST expose `buildServer(cfg, deps)` as the exported seam guarding `main()`; the seam receives the config object and injectable dependencies (token store, Supabase client, oBIX client) so unit tests can substitute them without a live station. |
| R4.2 | `POST /config/login` MUST authenticate against Supabase (re-auth, NOT a JSON user+password store), mint a server-held short-TTL config token bound to `(email, purpose="config-write")` with a sliding inactivity expiry; no credentials MAY be persisted to any file. `[ev: corpus B803 §803.6]` |
| R4.3 | `POST /config/logout` MUST revoke the config token immediately; a subsequent `/write` with the revoked token MUST return 401. |
| R4.4 | `POST /write` and `POST /alarms/ack` MUST gate on the config token; a request without a token MUST return 401; an expired token MUST return 401. |
| R4.5 | Read endpoints (viewer data, status) MUST NOT require the config token. |
| R4.6 | A non-allowlisted ORD in a `/write` request MUST return 403. |
| R4.7 | No JSON user+password credential store MAY exist anywhere in the tunnel diff. |
| R4.8 | After rebase onto `9acb47c`, the existing best-effort `change_log` insert (JWT-bearer row) MUST continue to pass the suite; S12A-4 and S12A-6 are re-pinned against the canonical `change_log` schema established in R5. `[ev: S12 plan 1ecdf437c]` |

**S12A-1** GIVEN the write-server is started with the `buildServer` seam; WHEN `POST /write` is called without a config token; THEN returns 401 and no station write is attempted.

**S12A-2** GIVEN `POST /config/login` is called with valid Supabase credentials; WHEN the endpoint returns; THEN a server-held config token exists bound to the requesting email and `purpose="config-write"`.

**S12A-3** GIVEN a valid config token is held; WHEN `POST /config/logout` is called; THEN the token is revoked and a subsequent `POST /write` with that token returns 401.

**S12A-4** (re-pinned per §5 audit-sink decision) GIVEN a successful `POST /write` with a valid config token; WHEN the write completes; THEN exactly ONE row is inserted into `public.change_log` with the canonical schema columns (`ts`, `config_session`, `result`, `surface`, `client_ip`, plus the pre-existing `user_email`, `user_id`, `room`, `slot`, `label`, `old_value`, `new_value`, `area`, `ok`).

**S12A-5** (mutation pin — OBSERVED) GIVEN the config-token check is removed from the `buildServer` seam; WHEN `POST /write` is called without a token; THEN S12A-1 flips to RED — OBSERVED flip.

**S12A-6** (re-pinned per §5 audit-sink decision) GIVEN the `change_log` insert fails (Supabase unavailable); WHEN `POST /write` is called with a valid config token; THEN the endpoint returns the station's HTTP outcome, one error entry is recorded (an `ok=false` row or a spool file entry), and no error caused by the audit failure is returned to the caller.

**S12A-7** GIVEN an expired config token (past TTL); WHEN `POST /write` is called; THEN returns 401.

Named mutation: dropping the config-token check flips S12A-1 and S12A-5 (OBSERVED).

---

## PR5 — S12-A canonical audit schema + local spool (R5)

**Branch**: `feat/c9-s12-audit-schema` | **QA RED**: extends `qa/c9-s12-write-server` (RED-first appended pins) | **Repo**: `pancaddia-leon-tunnel`

| ID | Requirement |
|----|-------------|
| R5.1 | The `public.change_log` migration MUST be additive-only: it adds `ts`, `config_session`, `result`, `surface`, `client_ip` columns and MUST NOT drop or retype any existing column. `[ev: S12 plan 1ecdf437c]` |
| R5.2 | The `old` value MUST be captured by a pre-write GET of the target slot from the station, never taken from the request body. `[ev: S12 plan 1ecdf437c]` |
| R5.3 | On every successful `/write`, exactly ONE row MUST be inserted into `public.change_log` with all canonical columns populated. |
| R5.4 | When the `change_log` insert fails, the write MUST still return the station's HTTP outcome; an `ok=false` error row or a spool file entry MUST be recorded; the endpoint MUST NOT return a caller-visible error caused by the audit failure. `[ev: corpus S12]` |
| R5.5 | The JSON-lines spool file is the failure spool only: written when the `change_log` insert fails; drained on the next successful insert; NOT a parallel primary trail. |
| R5.6 | Write-server rows MUST carry `surface = 'write-server'`. |

**audit-append-failure** GIVEN the Supabase client is injected with a stub that always fails; WHEN `POST /write` is called with a valid config token; THEN the endpoint returns the station's HTTP outcome (200 or the station error code), one error entry is recorded (either an `ok=false` row or a spool entry), and no 5xx error is returned to the caller.

**audit-row-count** GIVEN a successful `/write`; WHEN the database is inspected; THEN exactly ONE `change_log` row exists for that write, `surface = 'write-server'`, and `old_value` reflects the slot value captured before the write.

**audit-migration-additive** GIVEN the migration is applied to the live schema; WHEN the schema is inspected; THEN no column has been dropped or retyped; all prior rows remain intact.

Named mutation: forcing the audit insert failure to propagate as a thrown error causes `audit-append-failure` to flip from PASS to FAIL (OBSERVED).

---

## PR6 — S12-B DashboardPan servlet guards + real-Context write (R6)

**Branch**: `feat/c9-s12-servlet-guards` | **QA RED**: `qa/c9-s12-servlet` **`4c18837`** (rebased `a109249`) `[CERT]` | **Repo**: `angeles725/niagara-panccadia-leon` (DashboardPan-ux) | **Pins**: guard1-5 + guard4b + success/one-audit-entry | **Version**: DashboardPan 2.1.1 → 2.2.0

| ID | Requirement |
|----|-------------|
| R6.1 | A new `DashboardWriteGuards.evaluate` seam MUST gate every slot write in the servlet before the `parent.set` call; each guard returns a typed result dispatched by the servlet. |
| R6.2 | The servlet's `parent.set` call MUST use the real request-user Context (`parent.set(prop, toSet, cx)`) rather than `null`, making the write auditable by Niagara's `ComplexSlotMap` gate. `[ev: corpus B829]` |
| R6.3 | guard4 MUST ensure no silent zero: an invalid or unparseable value MUST return 400 with no write; a zero-suppression path MUST NOT silently write 0. |
| R6.4 | An audit-append failure in the servlet path MUST NOT fail the write; the write proceeds and the failure is recorded separately. |
| R6.5 | `schema-risk.sh` MUST return SAFE for PR6 (no slot dropped or retyped). |
| R6.6 | `lint-servlet.sh` MUST exit 0 (clean or WARN-only) on the post-PR6 servlet source. |
| R6.7 | `parent.set(prop, toSet, cx)` MUST carry the real request user so the `AuditEvent` produced by `ComplexSlotMap` is attributed (not suppressed). `[ev: corpus B829]` |

**guard1** GIVEN a request with no valid session or auth; WHEN the servlet evaluates the guards; THEN returns 401 and no `parent.set` is called.

**guard2** GIVEN a request with a non-allowlisted ORD target; WHEN the guards evaluate; THEN returns 403, no write.

**guard3** GIVEN a request where the requesting user lacks OPERATOR rights for the target slot; WHEN the guards evaluate; THEN returns 403, no write.

**guard4** (no-silent-zero regression) GIVEN a request body with an invalid numeric value; WHEN the guards evaluate; THEN returns 400, no write, and the value 0 is NOT written.

**guard4b** GIVEN a request where the coerce step would produce a default zero; WHEN the guards evaluate; THEN returns 400 and does NOT write 0.

**guard5** GIVEN a valid request that passes all guards; WHEN `parent.set(prop, toSet, cx)` is called; THEN the write succeeds with the real user Context and `success` is returned.

**success/one-audit-entry** GIVEN a valid write through the servlet; WHEN the write completes; THEN exactly one audit path is populated: an `AuditEvent` in `/PANCCADIA/AuditHistory` (confirmed at the B829-live gate, which is separate from this PR) or a `change_log` row via the mirror; the servlet returns the station outcome.

Named mutation: removing the real Context from `parent.set` (reverting to `null`) causes the guard5/success audit assertion to flip (OBSERVED).

---

## PR7 — S12-C AuditHistory→change_log mirror (R7)

**Branch**: `feat/c9-s12-audit-mirror` | **QA RED**: `qa/c9-s12-audit-mirror` — **RED to be authored** | **Repo**: `pancaddia-leon-tunnel` + kit doc

| ID | Requirement |
|----|-------------|
| R7.1 | The AuditHistory→`change_log` mirror job MUST be flag-gated OFF by default; live enablement is the B829-live gate only, never a PR gate. |
| R7.2 | The mirror MUST use a dedupe key of `(ts, user, target, old, new)` to prevent double-writes on replay. |
| R7.3 | Replaying the same recorded AuditHistory fixture twice with the mirror enabled MUST produce no additional `change_log` rows beyond the first pass (idempotent). |
| R7.4 | Mirror rows inserted into `change_log` MUST carry `surface = 'servlet'`. |
| R7.5 | The `config_session` column for surface-B (mirror) rows MUST be NULL in this campaign; the mirror MUST NOT fabricate a session value. |
| R7.6 | The reconciliation contract MUST be documented in the kit with one `[ev:]` token per row. `[ev: corpus B829]` `[ev: S12 plan 1ecdf437c]` |

**QA pin contract (RED to be authored on `qa/c9-s12-audit-mirror`):**

| Pin | Scenario |
|-----|----------|
| `MIR1 flag-gated-off` | GIVEN the mirror feature flag is OFF (default); WHEN the mirror job runs; THEN no rows are read from AuditHistory and no rows are inserted into `change_log`. |
| `MIR2 idempotent-replay` | GIVEN a recorded AuditHistory fixture replayed twice with the mirror enabled; WHEN the mirror job runs both times; THEN `change_log` row count after the second run equals the count after the first run. |
| `MIR3 no-duplicate-rows` | GIVEN the mirror has already inserted rows for the fixture; WHEN the same fixture is replayed; THEN no new rows appear in `change_log` (dedupe key `(ts, user, target, old, new)` proven effective). |
| `MIR4 surface-column` | GIVEN a mirror-inserted row; WHEN the row is read from `change_log`; THEN `surface = 'servlet'`. |
| `MIR5 null-config-session` | GIVEN a mirror-inserted row; WHEN the row is read; THEN `config_session IS NULL`. |

Each pin MUST carry a named mutation (e.g., removing the dedupe-key check causes MIR2 and MIR3 to flip from PASS to FAIL — OBSERVED).

---

## PR8 — Alarm PoC Pattern A: CR-3 freeze (R8)

**Branch**: `feat/c9-alarm-cr3` | **QA RED**: `qa/c9-alarm-cr3` — **RED to be authored** (from B827 §827.3) | **Repo**: `angeles725/niagara-panccadia-leon` (ColdRoomPan-rt) | **Version**: 2.0.7 → 2.1.0

| ID | Requirement |
|----|-------------|
| R8.1 | The CR-3 freeze protection trip MUST gain a child `BBooleanPoint` with a `BAlarmSourceExt` carrying a `BBooleanChangeOfStateAlgorithm`; both are additive child components — no existing slot dropped or retyped. `[ev: corpus B827 §827.3]` |
| R8.2 | On a freeze trip (alarm condition becomes true), the `BAlarmSourceExt` MUST route a `BAlarmRecord` with `sourceState = offnormal`. |
| R8.3 | On recovery (alarm condition returns false), the `BAlarmSourceExt` MUST route the corresponding normal/clear notification. |
| R8.4 | `schema-risk.sh` MUST return SAFE (additive child point; no existing slot affected). |
| R8.5 | The alarm record's `sourceState = offnormal` MUST make it selectable by the DashboardPan alarm bql at `:502`. `[ev: corpus B827]` |
| R8.6 | The PR MUST carry an OBSERVED mutation flip on at least one alarm-fires pin. |

**QA pin contract (RED to be authored on `qa/c9-alarm-cr3`):**

| Pin | Scenario |
|-----|----------|
| `CRA1 alarm-fires-on-freeze-trip` | GIVEN the freeze trip condition becomes true in the ColdRoomPan fixture; WHEN the component executes; THEN `BAlarmSourceExt` routes exactly one `BAlarmRecord` with `sourceState = offnormal`. |
| `CRA2 alarm-clears-on-recovery` | GIVEN the alarm is active and the freeze trip condition clears; WHEN the component executes; THEN the alarm transitions to normal/clear. |
| `CRA3 source-state-offnormal` | GIVEN a routed alarm record from the Pattern-A source; WHEN the record is inspected; THEN `sourceState == offnormal`. |
| `CRA4 schema-risk-safe` | GIVEN the after-snapshot of ColdRoomPan-rt slots; WHEN `schema-risk.sh` runs; THEN verdict = SAFE. |
| `CRA5 observed-mutation-flip` | GIVEN the `BAlarmSourceExt` is removed from the child point; WHEN the alarm-fires assertion runs; THEN `CRA1` flips from GREEN to RED (OBSERVED). |
| `CRA6 version-bump` | GIVEN PR8 merges; WHEN the module version is inspected; THEN `vendorVersion = 2.1.0`. |

Real-tree smoke (CD10): all four client module roots after PR8; `schema-risk.sh` = SAFE.

---

## PR9 — Alarm PoC Pattern B: CP-1 low suction (R9)

**Branch**: `feat/c9-alarm-cp1` | **QA RED**: `qa/c9-alarm-cp1` — **RED to be authored** (from B827 §827.4/§827.6) | **Repo**: `angeles725/niagara-panccadia-leon` (CompPan-rt) | **Version**: 2.1.0 → 2.2.0

| ID | Requirement |
|----|-------------|
| R9.1 | The CP-1 low-suction protection trip MUST be wired via `BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` invoked **on the offnormal EDGE only** (normal→offnormal transition), never level-triggered. `[ev: corpus B827 §827.4]` `[ev: corpus B827 B827-G1]` |
| R9.2 | A boolean `wasOffnormal[]` state array MUST track per-trip offnormal state; the alarm fires only when `!wasOffnormal[trip]` transitions to `true`. |
| R9.3 | On recovery (condition returns to normal), `AlarmSupport.toNormal` MUST be invoked and `wasOffnormal[trip]` MUST reset to `false`. |
| R9.4 | `BComponent.started()` MUST re-seed `wasOffnormal[]` from current condition state so a restart does not re-fire already-active alarms. |
| R9.5 | Multiple successive executions of `step` while the condition is offnormal MUST NOT produce more than one alarm record (no re-fire across repeated executes). `[ev: corpus B827 §827.6]` |
| R9.6 | The routed `BAlarmRecord` MUST carry `sourceState = offnormal`. |
| R9.7 | `schema-risk.sh` MUST return SAFE (additive `BIAlarmSource` references; no slot dropped or retyped). |
| R9.8 | The PR MUST carry an OBSERVED mutation flip on the edge-only pin. |

**QA pin contract (RED to be authored on `qa/c9-alarm-cp1`):**

| Pin | Scenario |
|-----|----------|
| `CPB1 alarm-fires-on-edge-only` | GIVEN the CP-1 condition transitions from normal to offnormal; WHEN `step` executes; THEN exactly ONE `BAlarmRecord` with `sourceState = offnormal` is produced. |
| `CPB2 no-refire-across-executes` | GIVEN the CP-1 condition remains offnormal; WHEN `step` executes N > 1 times; THEN no additional alarm records are produced after the first. |
| `CPB3 to-normal-on-recovery` | GIVEN the CP-1 condition returns to normal; WHEN `step` executes; THEN `toNormal` is invoked and `wasOffnormal[trip]` resets to false. |
| `CPB4 started-reseeds-wasOffnormal` | GIVEN the component restarts while CP-1 is in offnormal state; WHEN `started()` runs; THEN `wasOffnormal[trip]` is seeded true and the next `step` does NOT re-fire the alarm. |
| `CPB5 source-state-offnormal` | GIVEN a routed alarm record from the Pattern-B source; WHEN the record is inspected; THEN `sourceState == offnormal`. |
| `CPB6 schema-risk-safe` | GIVEN the after-snapshot of CompPan-rt slots; WHEN `schema-risk.sh` runs; THEN verdict = SAFE. |
| `CPB7 observed-mutation-flip` | GIVEN edge-detection is removed (level-triggered path restored); WHEN the suite runs over N repeated executes while offnormal; THEN `CPB2` flips from GREEN to RED (OBSERVED). |

Real-tree smoke (CD10): all four client module roots after PR9; `schema-risk.sh` = SAFE.

---

## PR10 — lint-ext-writable-shape.sh (R10)

**Branch**: `feat/c9-ext-writable-shape` | **QA RED**: `qa/c9-ext-writable-shape` **`3726722`** `[CERT]` | **Repo**: `niagara-tools` | **Pins**: EW1-EW10

| ID | Requirement |
|----|-------------|
| R10.1 | `lint-ext-writable-shape.sh` MUST emit `WARN` for every `@NiagaraProperty` slot with `Flags.OPERATOR` whose declared type is a complex (`BStatusNumeric`, `BStatusBoolean`, `BStatusEnum`) with no associated `@NiagaraAction` writing action. `[ev: corpus B823]` |
| R10.2 | A `Flags.OPERATOR` slot whose type is a plain value (`double`, `boolean`, `BRelTime`) MUST produce no WARN. |
| R10.3 | A complex `Flags.OPERATOR` slot that carries an associated `@NiagaraAction` for writing MUST produce no WARN. |
| R10.4 | EW10 MUST produce a WARN on the real `BRoomPanel.setpoint` slot (production fixture confirming real-module coverage). `[ev: corpus B823]` |
| R10.5 | Row grammar: `WARN  lint-ext-writable-shape  <file>:<line>  <reason>`; exit 0 on WARN-only; exit 1 on any FAIL; exit 3 on no sources / usage. |
| R10.6 | The script MUST iterate from the module root through all profiles; a root with no Java sources MUST exit 3, never a silent 0. |
| R10.7 | Dot-dirs MUST be pruned during traversal (D9b). |
| R10.8 | `lint-ext-writable-shape.sh` MUST be named in both `BUILD-LOOP.md` §5 and `skill/SKILL.md` in PR10 (K19). |
| R10.9 | `report-module.sh` MUST gain a member row for `lint-ext-writable-shape`; a FAIL from this lint MUST surface as aggregate FAIL. |

**EW10-smoke** GIVEN all four client module roots at the chain's client tip; WHEN `lint-ext-writable-shape.sh` runs; THEN `BRoomPanel.setpoint` appears in the WARN output with subject name (EW10 assertion), exact count matches the expected number of writable-shape findings, and no unexpected subjects appear (absence assertion).

**EW1-EW9**: per the RED at tip `3726722`; apply agent re-reads the tip at apply (K13).

Named mutation: removing the `Flags.OPERATOR` complex-type detection causes EW1 to flip from WARN-present to absent (OBSERVED).

---

## PR11 — write-path matrix W14-W22 (R11)

**Branch**: `docs/c9-write-path-w14-w22` | **QA RED**: `qa/c8-write-path` **`5e357d1`** `[CERT]` (extended to W14-W22) | **Repo**: `niagara-tools`

| ID | Requirement |
|----|-------------|
| R11.1 | `docs/write-path-matrix.md` MUST gain rows W14-W22 covering the real writable slots beyond W13; each row cites the real writable slot name, its declaring class, and the write mechanism. `[ev: corpus S5]` |
| R11.2 | After W14-W22 land, `lint-write-path.sh` MUST exit 0 on every client module root. |
| R11.3 | A client module root with no `docs/write-path-matrix.md` MUST cause `lint-write-path.sh` to exit **3** (never a silent 0). |
| R11.4 | Each W14-W22 row corresponds to a real slot verified against the actual module source at the chain's client tip. |

**W14-W22-smoke** GIVEN `lint-write-path.sh` runs on each of the four client module roots after PR11 merges; WHEN the exit code is checked; THEN all roots with a matrix exit 0; a root with no matrix exits 3 (never a silent 0).

W14-W22 row shapes: per the RED at tip `5e357d1` extended; apply agent re-reads the tip at apply (K13).

---

## PR12 — Doctrine fold (R12)

**Branch**: `docs/c9-doctrine` | **Type**: doc-only · zero new bats tests | **Repo**: `niagara-tools`

| ID | Requirement |
|----|-------------|
| R12.1 | `types/logic-authoring.md` MUST gain §"Slot types for externally written values": a slot written by external clients MUST be a simple value type or carry a `@NiagaraAction` writing action; a bare `Flags.OPERATOR` complex either rejects the write or silently writes a default; `[ev: corpus B823]` + `[ev: corpus B826]`. |
| R12.2 | `types/logic.md` MUST gain §"Protection anatomy" covering alarm authoring patterns A and B: Pattern A (declarative child `BBooleanPoint` + `BAlarmSourceExt`); Pattern B (`BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` on the offnormal EDGE only); both route `sourceState = offnormal`; both are schema-SAFE additive; `[ev: corpus B827]` + `[ev: corpus B821 §821.4]`. |
| R12.3 | `types/dashboard.md` MUST gain a one-liner cross-reference to the slot-type doctrine section in `types/logic-authoring.md`. |
| R12.4 | `BUILD-LOOP.md` §5 MUST state the ONE module-root/profile convention: the module root is the directory containing profiles; every lint MUST iterate the profiles; a root with no sources or no matrix is an ERROR (exit 3), never a silent 0. `[ev: retro campaign8-close-process-meta-lessons lesson 11c]` |
| R12.5 | `METHODOLOGY.md` MUST carry K22: a real-tree smoke on every client module root is part of the lead gate; smoke pins assert exact count + subject + absence; a skipped smoke is a BLOCKER, never an advisory. `[ev: retro campaign8-close-process-meta-lessons lesson 11a/11b]` |
| R12.6 | `METHODOLOGY.md` MUST state that mutation tables record OBSERVED flips (verbatim RED-then-GREEN output), never "would flip" prose. `[ev: retro campaign8-lint-timers-ext]` |
| R12.7 | `METHODOLOGY.md` MUST gain the four-always-conflict-files + fragment-merge rule as the standing multi-session merge protocol (append, keep both rows, dedupe by script name, never overwrite). `[ev: retro campaign8-close-process-meta-lessons lesson 2]` |
| R12.8 | `BUILD-LOOP.md` §7 MUST state the lead merge/settle order: merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger; rebase a parallel branch before the QA ping. `[ev: retro campaign8-close-process-meta-lessons lesson 10]` |
| R12.9 | `types/dashboard.md` and `BUILD-LOOP.md` §6 MUST document the unified write-audit reconciliation contract (§5 decision): one canonical sink, two writers, dedupe key, audit-fail never fails the write; `[ev: corpus B829]` + `[ev: S12 plan 1ecdf437c]`. |
| R12.10 | Every new doctrine paragraph MUST carry one `[ev:]` token; `sweep-fold-audit.sh --strict` MUST exit 0; `kit-links.bats` MUST exit 0 after PR12. |

---

## PR13 — Close (R13)

**Branch**: `chore/c9-close` | **Type**: close gate | **Repo**: `niagara-tools` | **Pins**: `tests/c9-close.bats` under `C9_CLOSE=1`

| ID | Requirement |
|----|-------------|
| R13.1 | `tests/c9-close.bats` under `C9_CLOSE=1` MUST assert: `VERSION = 0.20.0`; `CHANGELOG.md` has a `## 0.20.0` section conforming to CONTRIBUTING §4-5; `retros/INDEX.md` has `pending = 0`; `sweep-build-state.sh` exits 0; `sweep-fold-audit.sh --strict` exits 0. |
| R13.2 | `shellcheck 0.10.0` MUST exit 0 on every new or modified `toolbelt/*.sh` in the PR1-PR13 range. |
| R13.3 | No commit body in the **entire PR1-PR13 range** MAY carry an attribution trailer (K11); the close gate MUST scan the full range and assert zero trailers. |
| R13.4 | `retros/INDEX.md` `pending` MUST be 0 at close; every kit-changing push (PR2, PR3, PR4-PR7, PR8-PR10) MUST have its retro filed and its INDEX row updated before PR13 opens. |
| R13.5 | `VERSION` file MUST read `0.20.0`; `CHANGELOG.md` MUST contain a `## 0.20.0` section with an entry per CONTRIBUTING §4-5. |
| R13.6 | `kit-links.bats` MUST exit 0 after PR13, verifying that every new script (`lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh`) is reachable from both `BUILD-LOOP.md` §5 and `skill/SKILL.md`. |

---

## Non-functional requirements (all PRs)

| Rule | Requirement |
|------|-------------|
| NF1 | `schema-risk.sh` verdict MUST be SAFE for every client PR that touches slots (PR1, PR6, PR8, PR9). |
| NF2 | Audit-append failure MUST NEVER fail the write on either surface (R5 write-server, R6 servlet): the write lands; an `ok=false` error row or a spool entry is recorded. |
| NF3 | `rotationInterval = 0` MUST produce a byte-identical `CompressorControl.step` trace to the pre-change golden (ROT5). |
| NF4 | The AuditHistory mirror (R7) ships flag-gated OFF by default; live enablement is the B829-live gate only. |
| NF5 | No live station write occurs within any SDD slice; deploys remain with the operator under BUILD-LOOP §6. |
| NF6 | No JSON user+password credential store MAY exist anywhere in the tunnel diff (R4, R5). |

---

## Success Criteria

| ID | Assertion |
|----|-----------|
| SC-1 | `CompressorControlTest` proves `rotationInterval = 0` produces a byte-identical step trace to the pre-change golden (ROT5 pin green). Named mutation for each of ROT1-ROT10 lands as an OBSERVED flip. `schema-risk.sh` = SAFE. `vendorVersion` = 2.1.0 after PR1. |
| SC-2 | Rotation pins ROT1-ROT10 green with count + subject assertions: swap after the interval, NO swap below it, NO swap while `minOff` active, make-before-break ordering, single-available no swap, HOA OFF excluded, HAND untouched, no swap on `dischargeHigh`, no swap below LP floor, `condenserNHours` unaffected. |
| SC-3 | Audit-append failure pins green on both surfaces (R5 + R6): with the sink forced to fail, `/write` returns the station outcome; `change_log` receives an `ok=false` error row or the spool receives the entry; the servlet path behaves identically; no 5xx error reaches the caller. |
| SC-4 | S12A-1..S12A-7 green on the rebased tip (`9acb47c`): `/config/login` mints a token; `/write`+`/alarms/ack` without a token → 401; expired token → 401; non-allowlisted ORD → 403; `/config/logout` revokes; a successful write inserts exactly ONE canonical `change_log` row with `old` from a pre-write GET. No JSON user+password store in the tunnel diff. |
| SC-5 | guard1-5 + guard4b + success/one-audit-entry green on the rebased servlet tip (`a109249`): `parent.set(prop, toSet, cx)` carries the real request user; guard4 no-silent-zero holds; `schema-risk.sh` = SAFE; `lint-servlet.sh` exits 0. `vendorVersion` = 2.2.0 after PR6. |
| SC-6 | MIR1-MIR5 green: mirror ships flag-gated OFF; dedupe key idempotent over a replayed AuditHistory fixture (no duplicate rows); mirror rows carry `surface='servlet'` and `config_session IS NULL`; reconciliation contract documented with one `[ev:]` per row. |
| SC-7 | CRA1-CRA6 green (Pattern A CR-3 freeze) and CPB1-CPB7 green (Pattern B CP-1 low suction): each raises a `BAlarmRecord` with `sourceState = offnormal`; Pattern B fires once on the edge, sends `toNormal` on recovery, re-seeds on `started()`; both verdicts `schema-risk.sh` = SAFE. `vendorVersion`: ColdRoomPan-rt = 2.1.0, CompPan-rt = 2.2.0. |
| SC-8 | DS1-DS7 + DS-smoke green, SP1-SP8 + SP-smoke green, EW1-EW10 green: each lint runs on all four client module roots; smoke pins assert exact count + subject + absence; `lint-silent-protection` flags CP-1 and CR-3 and does NOT flag CP-2/`defrostSkipped`; `lint-ext-writable-shape` WARNs the real `BRoomPanel.setpoint` (EW10). |
| SC-9 | Every lint exits 3 (never a silent 0) on a module root with no sources or no matrix; `lint-write-path.sh` exits 0 on all client module roots once W14-W22 land; a root without a matrix exits 3. |
| SC-10 | Every code PR (PR1-PR11) records an OBSERVED mutation flip (verbatim RED-then-GREEN); no PR merges on a "would flip" prose claim; no skipped smoke is downgraded to advisory. |
| SC-11 | `BUILD-LOOP.md` §5 carries the one module-root/profile convention; `METHODOLOGY.md` carries K22; `types/logic.md` carries alarm patterns A/B; `types/logic-authoring.md` carries the slot-type doctrine; `BUILD-LOOP.md` §7 carries the lead merge/settle order; `METHODOLOGY.md` carries the fragment-merge rule and OBSERVED-flip requirement; `kit-links.bats` green with every new script routed in both `BUILD-LOOP.md` and `skill/SKILL.md` (K19). |
| SC-12 | `tests/c9-close.bats` green under `C9_CLOSE=1`; `sweep-build-state.sh` and `sweep-fold-audit.sh --strict` green; `retros/INDEX.md` pending = 0; `VERSION` = `0.20.0` with a `CHANGELOG.md` `## 0.20.0` section per CONTRIBUTING §4-5; `shellcheck 0.10.0` exits 0; no commit body in PR1-PR13 carries an attribution trailer. |
| SC-13 | Client `vendorVersion` bumps schema-risk-cleared: CompPan-rt → 2.2.0 (PR1: 2.0.3→2.1.0, PR9: 2.1.0→2.2.0); ColdRoomPan-rt → 2.1.0 (PR8); DashboardPan-ux → 2.2.0 (PR6). |

---

## Pin map (corrections bound to executable REDs)

Where spec and QA RED branches disagree, the RED wins (CD3); design records deviations.

| Pin | Maps to |
|-----|---------|
| DS1 | R2.1 — `CompressorControl.step` fixture FAIL (no zero-demand short-circuit) |
| DS2 | R2.9 — named mutation: removing the demand guard causes FAIL (OBSERVED at `2916954`) |
| DS3-DS7 | R2.1-R2.4 — per RED `2916954`; bind at apply |
| DS-smoke | R2.8 + CD10 — all four client module roots with count + subject + absence |
| SP1-SP7 | R3.1-R3.4 — per RED `e38e503`; bind at apply |
| SP8 | R3.10 — named mutation: removing alarm-ext check causes flip (OBSERVED at `e38e503`) |
| SP-smoke | R3.9 + CD10 — flags CP-1/CR-3; absent CP-2/`defrostSkipped`; exact count |
| S12A-1 | R4.4 — `/write` without token → 401 |
| S12A-2 | R4.2 — `/config/login` mints token bound to email + purpose |
| S12A-3 | R4.3 — `/config/logout` revokes; next write → 401 |
| S12A-4 | R5.3 (re-pinned per §5) — exactly ONE canonical `change_log` row per successful write |
| S12A-5 | R4.4 / mutation — dropping token check flips S12A-1 (OBSERVED) |
| S12A-6 | R5.4 (re-pinned per §5) — audit-append failure NEVER fails the write |
| S12A-7 | R4.4 — expired token → 401 |
| guard1 | R6.1 — auth gate: 401 on no session |
| guard2 | R6.1 — ORD gate: 403 on non-allowlisted |
| guard3 | R6.1 — OPERATOR rights: 403 on insufficient |
| guard4 | R6.3 — no-silent-zero: 400 on invalid value |
| guard4b | R6.3 — no-silent-zero: 400 on coerced zero |
| guard5 / success/one-audit-entry | R6.2 / R6.7 — real-Context `parent.set` + one audit entry |
| EW1-EW9 | R10.1-R10.3 — per RED `3726722`; bind at apply |
| EW10 | R10.4 — real `BRoomPanel.setpoint` WARN (production fixture) |
| W14-W22 | R11.1-R11.4 — per RED `5e357d1` extended; bind at apply |
| ROT1-ROT10 | R1.2-R1.10 — rotation contract pins; RED `qa/c9-comppan-rotation` to be authored |
| MIR1-MIR5 | R7.1-R7.6 — mirror pins; RED `qa/c9-s12-audit-mirror` to be authored |
| CRA1-CRA6 | R8.1-R8.6 — Pattern A alarm pins; RED `qa/c9-alarm-cr3` to be authored |
| CPB1-CPB7 | R9.1-R9.8 — Pattern B alarm pins; RED `qa/c9-alarm-cp1` to be authored |

REDs not yet authored at spec time: `qa/c9-comppan-rotation` (ROT1-ROT10), `qa/c9-s12-audit-mirror` (MIR1-MIR5), `qa/c9-alarm-cr3` (CRA1-CRA6), `qa/c9-alarm-cp1` (CPB1-CPB7). R5 schema pins are appended to `qa/c9-s12-write-server` as RED-first additions. Apply agent re-reads ALL branch tips at apply (K13).
