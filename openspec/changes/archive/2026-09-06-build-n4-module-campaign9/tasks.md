# Tasks: build-n4-module-campaign9

**Source**: v0.19.0 (kit main `d2857d1`, C8 close `1109c0f`) → **Target**: v0.20.0
**Chain**: stacked-to-main | 14 work units (PR1-PR13 + PR6b) across 3 waves

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~3 100–3 500 total (excl. fixture heredocs and generated slot block) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | W1: PR1-PR3 parallel; W2: PR4→PR5→PR6→PR6b→PR7; W3: PR8/PR9 parallel → PR10→PR11→PR12→PR13 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |
| PR1 declared | size:exception — generated slot block (~120 lines) justified by `BRotationMode` frozen-enum + two `@NiagaraProperty` declarations; authored diff ~180 |
| PR6b declared | size:exception — `ConfigLoginGuard` + `ConfigSession` + CLW1-CLW5 wiring; logic is Baja-free core + adapter (~260 lines); justified by audit trail completeness |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | PR | Goal | Est. lines | Focused test | Rollback |
|------|----|------|-----------|-------------|---------|
| 1 | PR1 | S20 rotation: `rotationInterval`/`rotationMode` + step 2b/3b | ~300 | `CompressorRotationTest` 17 pins | `rotationInterval=0` byte-identical — safe feature-flag |
| 2 | PR2 | `lint-demand-scope.sh` + DS1-DS7 + DS-smoke | ~200 | `bats tests/demand-in-scope.bats` | new paths; `git revert` |
| 3 | PR3 | `lint-silent-protection.sh` + SP1-SP8 + SP-smoke | ~220 | `bats tests/lint-silent-protection.bats` | new paths; `git revert` |
| 4 | PR4 | `buildServer(cfg,deps)` seam + `/config/login` token gate | ~280 | `node:test` S12A-1..7 | revert seam; JWT-bearer path intact |
| 5 | PR5 | `change_log` schema extension + failure spool | ~220 | `node:test` audit pins | additive columns; spool append-only |
| 6 | PR6 | `DashboardWriteGuards.evaluate` + real-Context `parent.set` | ~260 | guard1-5 + guard4b + success/one-audit-entry | null-Context revert is `git revert` |
| 6b | PR6b | In-module config login: `ConfigLoginGuard` + `ConfigSession` + guard6 | ~260 | CL1-CL11 + CLW1-CLW5 | session map drops; guard6 reverts to pass-through |
| 7 | PR7 | `audit-mirror.mjs` `runMirror` flag-gated + MIR1-MIR5 | ~180 | `node:test` MIR1-MIR5 | `MIRROR_ENABLED` off by default |
| 8 | PR8 | CR-3 freeze Pattern A: child `BBooleanPoint` + `BAlarmSourceExt` | ~260 | CRA1s/2s/3s + CRA4/CRA5/CRA6 structural | ext not reached → schema revert |
| 9 | PR9 | CP-1 low-suction Pattern B: `AlarmEdge` + `BIAlarmSource` | ~280 | CPB1-CPB4 + CPB6/CPB7 | additive nested class; `git revert` |
| 10 | PR10 | `lint-ext-writable-shape.sh` + EW1-EW10 + EW10-smoke | ~230 | `bats tests/lint-ext-writable-shape.bats` | new paths; `git revert` |
| 11 | PR11 | Write-path matrix: 62 rows in client `docs/write-path-matrix.md` | ~150 | `lint-write-path.sh` exit 1→0 | doc-only; `git revert` |
| 12 | PR12 | Doctrine fold: `types/logic.md` §Protection + BUILD-LOOP + METHODOLOGY | ~200 | `sweep-fold-audit.sh --strict` + `kit-links.bats` | doc-only; `git revert` |
| 13 | PR13 | Close gate: `tests/c9-close.bats` + VERSION 0.20.0 + CHANGELOG | ~180 | `bats tests/c9-close.bats C9_CLOSE=1` | doc-only; `git revert` |

---

## WAVE 1 — PR1, PR2, PR3 (parallel worktrees; merge all before W2 opens)

### PR1 — feat/c9-comppan-rotation (~300, size:exception authored ~180)

**RED**: `qa/c9-comppan-rotation` tip **`cf28572`** (17 pins ROT1-ROT16 + golden; parent `a109249`) — re-read at apply (K13)
**Repo**: `angeles725/niagara-panccadia-leon` · **Profile**: `CompPan-rt`
**Worktree**: `Cliente/Leon-Guanjuato-worktrees/c9-comppan-rotation` (branch off `a109249`)
**D-ids**: D1, D1a, D1b, D1c, D1d · **Gate**: R1.1-R1.11, SC-1, SC-2, SC-13
**Apply-package input**: niagara-research S20 rev 2 `6c3bcf107` — re-read at apply for updated rotation contract details.

- [x] 1.1 Re-read RED tip `cf28572` (K13): confirm compile symbols `Cfg.rotationIntervalMs` (long), `Cfg.rotationMode` (int), `ROTATION_MAKE_BEFORE_BREAK` (static final int), `ctl.swaps` (int, package-visible); confirm ROT1/ROT4 include N=3 case (D1b, D1c); confirm `pickLeastHoursOffAuto` skip `modes[k] != MODE_AUTO`.
- [x] 1.2 Cherry-pick / merge RED branch into `feat/c9-comppan-rotation` as commit 1 (suite is RED before impl).
- [x] 1.3 Add `BRotationMode.java`: new frozen enum `{makeBefore(0), breakBefore(1)}`; ordinals match `ROTATION_MAKE_BEFORE_BREAK = 0` / `ROTATION_BREAK_BEFORE_MAKE = 1` (D1b).
- [x] 1.4 Add two `@NiagaraProperty` slots to `BCompressorControl.java`: `rotationInterval` (`BRelTime`, `SUMMARY|OPERATOR`, min 0, max 24h) and `rotationMode` (`BRotationMode`, `SUMMARY|OPERATOR`); shape precedent: `powerOnDelay` at `:383-388`; MIN and MAX both supplied (D1).
- [x] 1.5 Add adapter wiring beside the `minOnMs/minOffMs/stageDelayMs` trio at `:1907-1909`: `cfg.rotationIntervalMs = getRotationInterval().getMillis();` and `cfg.rotationMode = getRotationMode().getOrdinal();` (D1).
- [x] 1.6 Optionally surface `setRotationSwaps(ctl.swaps)` as a READONLY slot; if added, include its write-path matrix row (D1). — SKIPPED (not added, optional).
- [x] 1.7 In `CompressorControl.java`: add transient fields `rotOut = -1`, `rotArmedMs`, `rotSinceMs[]`, `swaps`; mode constants `ROTATION_MAKE_BEFORE_BREAK = 0`, `ROTATION_BREAK_BEFORE_MAKE = 1` (D1b, D1d).
- [x] 1.8 Clear `rotOut`, `rotArmedMs`, `rotSinceMs[]`, `swaps` in `resetTransient()` at `:328-335`; re-seed `rotSinceMs[]` alongside `cmdSince[]` in `seedRestart(now)` at `:346-349` (D1d).
- [x] 1.9 Stamp `rotSinceMs[k] = now` in the ordinary stage-up write at `:229` (stamped whenever a unit is commanded ON by staging or by rotation arm) (D1). — DEVIATION: stamp is via feedback loop (lazy, 1 stageDelay after first cmd), not at stage-up write; preserves ROT5 golden.
- [x] 1.10 Add `pickLeastHoursOffAuto(now, minOffMs)`: skip condition `cmd[k] || modes[k] != MODE_AUTO`; do NOT modify existing `pickLeastHoursOff` (ROT5 golden requires the stage-up path unchanged) (D1).
- [x] 1.11 Insert **step 2b** (rotation COMPLETION) after `:217` clamp-end, before `:219` step-3 comment: when `rotOut >= 0 && (now - rotArmedMs) >= stageDelayMs`, execute completion — `cmd[rotOut] = false; cmdSince[rotOut] = now; lastStageMs = now; rotOut = -1; swaps++` (make-before-break default); handle edge cases ROT11-ROT15 per D1c (demand-rise cancel, demand-fall drop-rotOut-first, dischargeHigh mid-window, HAND/OFF flip mid-window).
- [x] 1.12 Insert **step 3b** (rotation ARM) after `:246` stage-move close, before `:248` step-4 comment: evaluate 10 gates in order (D1 gate table, gates 1-10); arm action: `cmd[in] = true; cmdSince[in] = rotSinceMs[in] = now; lastStageMs = now; rotOut = out; rotArmedMs = now` (D1).
- [x] 1.13 Handle `rotationMode = ROTATION_BREAK_BEFORE_MAKE`: invert completion order (drop `rotOut` first, add `in` after `stageDelay`).
- [x] 1.14 Bump `Compresores/build.gradle.kts:33` from `2.0.3` to **`2.1.0`** — fragment-merge constraint: PR9 bumps this same line to 2.2.0 later; PR1 sets ONLY 2.1.0.
- [x] 1.15 Add two write-path matrix rows in `docs/write-path-matrix.md` for `rotationInterval` and `rotationMode` (D11c decision: PR1 owns its 2 S20 rows; PR11 owns the remaining 62).
- [x] 1.16 **Named mutations** (all OBSERVED): (a) drop `!= MODE_AUTO` from `pickLeastHoursOffAuto` → ROT7b flips RED (1 failure); (b) delete completion `lastStageMs = now` → ROT7b+ROT11+ROT14+ROT15+ROT16 flip RED (7 failures); (c) base clock on `cmdSince[k]` instead of `rotSinceMs[k]` → ROT9+ROT7b+ROT16 flip RED (4 failures); (d) treat `rotationIntervalMs=0` as tiny interval (`>=0`) → ROT5 golden + ROT16 flip RED (2 failures).
- [x] 1.17 `schema-risk.sh` SAFE on real `config.bog` snapshot (R1.11). — verdict=SAFE: add_slot only (rotationInterval, rotationMode).
- [x] 1.18 Real-tree smoke (CD10): `lint-demand-scope.sh` (no output = no false positives), `lint-silent-protection.sh` (2 pre-existing WARNs — not false positives, pre-existing on base), `lint-ext-writable-shape.sh` (script not yet built, PR10 scope — SKIP); `schema-risk.sh` = SAFE for CompPan-rt.
- [x] 1.19 Guards: `javac`/`slotomatic` exits 0 (BUILD SUCCESSFUL 11s); 0 attribution trailers in commit bodies (K11/CD7 verified); rebase not needed (feat/c9-comppan-rotation is 3 commits ahead of a109249 on same branch).
- [x] **[lead]** ROT1-ROT16 + golden green (17/17); OBSERVED flips for all 4 named mutations; `schema-risk.sh` SAFE; `vendorVersion = 2.1.0` confirmed in `CompPan-rt/module.xml`; tip commit `57a15d2`.

**Harness-only**: none for PR1 — all ROT pins are off-station pure Java.

---

### PR2 — feat/c9-demand-scope (~200)

**RED**: `qa/c9-demand-in-scope` tip **`d0f5942`** (DS1-DS7 + DS-smoke; spec cites `2916954` as authoring tip; re-read at apply K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools/c9-demand-scope` (branch off kit main `d2857d1`)
**D-ids**: D2, D2a, D2b, D3 · **Gate**: R2.1-R2.9, SC-8, SC-10
**Apply-package input**: `950a1da06` (S7 package; cites RED tip `d0f5942`) — re-read at apply.

- [x] 2.1 Re-read RED tip `d0f5942` (K13): confirm row column order `WARN  lint-demand-scope  <file>:<line>  <reason>` (D2a `STATUS check subject detail`); confirm DS2 = WARN row + exit 0 NOT exit 1 (D2b); confirm CLI shape `[--strict] <java-src-dir>` (D2, lint-delays shape); confirm inline heredoc fixtures — no `tests/fixtures/` dir.
- [x] 2.2 Cherry-pick / merge RED branch into `feat/c9-demand-scope` as commit 1.
- [x] 2.3 Write `toolbelt/lint-demand-scope.sh`: two-pass grep/awk (Pass 1 control-decision methods, Pass 2 demand-shaped inputs in scope); `LC_ALL=C`; `set -u`; dot-dirs pruned (D9b/CD9); exits 0 (WARN-only or clean) / 1 (FAIL under `--strict`) / 3 (usage or no source, K20 disjoint); no `eval`; `shellcheck 0.10.0` clean (CD6).
- [x] 2.4 Write `tests/demand-in-scope.bats` (DS1-DS7 + DS-smoke verbatim from RED; inline heredoc fixtures written into `$BATS_TEST_TMPDIR`; D2 shape).
- [x] 2.5 Add K19 routing: one line in `BUILD-LOOP.md` §5 + one line in `skill/SKILL.md` for `lint-demand-scope.sh [ev: retro c9-demand-scope]` (CD5, R2.7).
- [x] 2.6 Extend `toolbelt/report-module.sh`: append member row for `lint-demand-scope`; a FAIL from this lint surfaces as aggregate FAIL (R2.8).
- [x] 2.7 **Named mutation** (OBSERVED): drop the field-scan demand-input check → a class-field demand input stops counting; DS2 flips PASS→FAIL. Record verbatim output (R2.9).
- [x] 2.8 DS-smoke (CD10): `lint-demand-scope.sh` on ColdRoomPan-rt · CompPan-rt · DashboardPan-rt · DashboardPan-ux at the chain's client tip — exact count + at minimum `CompressorControl.step` flagged as named subject + absence assertions (no false positives on guardian-exempt paths); no dot-dir traversed.
- [x] 2.9 Guards: `bats tests/demand-in-scope.bats` all green; `shellcheck 0.10.0` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats`; 0 attribution trailers (K11); rebase onto kit main before QA ping; verify `git log -1` before settle.
- [x] 2.10 Retro file + `retros/INDEX.md` row + `BUILD-STATE.md` self-envelope in same push range (CD1).
- [x] **[lead]** DS1-DS7 + DS-smoke green; OBSERVED DS2 flip recorded in PR body; K19 routing in both `BUILD-LOOP.md` + `skill/SKILL.md`; `kit-links.bats` green; merge ff-only; ledger acquire + settle `--max-changed-lines 200`.

---

### PR3 — feat/c9-silent-protection (~220)

**RED**: `qa/c9-silent-protection` tip **`e38e503`** (SP1-SP8 + SP-smoke; `[CERT]`) — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools/c9-silent-protection` (branch off kit main `d2857d1`)
**D-ids**: D2, D2a, D2b, D4 · **Gate**: R3.1-R3.10, SC-8, SC-10
**Apply-package input**: `950a1da06` (S18-lint package; shares the same apply-package with PR2) — re-read at apply.
**Cross-PR**: R3↔R8 COUPLED on CR-3 smoke pin — whichever of PR3/PR8 merges second must update the SP-smoke CR-3 subject assertion (absent once Pattern A ext is wired in PR8).

- [x] 3.1 Re-read RED tip `e38e503` (K13): confirm SP1/SP3/SP8 each emit exactly ONE WARN (dedupe per trip site — a trip reached by several guard branches is one row, not one per branch); confirm SP2 is the cross-file one-level field→slot follow resolved within the same source directory; confirm SP8 pins that a private field is never a surface; confirm row grammar `WARN  lint-silent-protection  <file>:<line>  <reason>` (D2a).
- [x] 3.2 Cherry-pick / merge RED branch into `feat/c9-silent-protection` as commit 1.
- [x] 3.3 Write `toolbelt/lint-silent-protection.sh`: two-part scan per D4 — TRIPS detection + SURFACES resolution; one-level field→adapter follow (SP2); effect-slot exemption (SP3); name allowlist advisory seam; `LC_ALL=C`; `set -u`; dot-dirs pruned (D9b); exits 0/1/3 (K20); no `eval`; `shellcheck 0.10.0` clean (CD6).
- [x] 3.4 Write `tests/lint-silent-protection.bats` (SP1-SP7 verbatim from RED; SP8 mutation pin; SP-smoke; inline heredoc fixtures into `$BATS_TEST_TMPDIR`).
- [x] 3.5 Add K19 routing: `BUILD-LOOP.md` §5 + `skill/SKILL.md` for `lint-silent-protection.sh [ev: retro c9-silent-protection]` (CD5, R3.8).
- [x] 3.6 Extend `toolbelt/report-module.sh`: append member row for `lint-silent-protection`; FAIL surfaces as aggregate FAIL (R3.9).
- [x] 3.7 **Named mutations** (OBSERVED): (a) treat a private field as a surface → CR-3 (`freezeTripped` private at `BEvaporatorUnit.java:1287`) WARN disappears; SP8 flips; (b) drop the field→slot follow → CP-2 `dischargeHighAlarm` (written at `:1994`) starts WARNing — false positive. Record verbatim output for each (R3.10).
- [x] 3.8 SP-smoke (CD10): `lint-silent-protection.sh` on ColdRoomPan-rt · CompPan-rt · DashboardPan-rt · DashboardPan-ux — CP-1 and CR-3 MUST appear as named subjects; CP-2 and `defrostSkipped` MUST be absent; exact count matches expected. **If PR8 has already merged** (CR-3 alarm ext wired), update the pin: CR-3 is now absent from the smoke; record the updated expectation.
- [x] 3.9 Guards: `bats tests/lint-silent-protection.bats` all green; `shellcheck 0.10.0` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats`; 0 attribution trailers; rebase onto kit main before QA ping; verify `git log -1` before settle.
- [x] 3.10 Retro file + `retros/INDEX.md` row + `BUILD-STATE.md` self-envelope (CD1).
- [x] **[lead]** SP1-SP8 + SP-smoke green (CR-3 pin updated if PR8 merged first); OBSERVED SP8 flip recorded; K19 routing; `kit-links.bats` green; merge ff-only; ledger acquire + settle `--max-changed-lines 220`.

---

## WAVE 2 — PR4 → PR5 → PR6 → PR6b → PR7 (sequential; all merge before W3 opens)

### PR4 — feat/c9-s12-config-login (~280)

**RED**: `qa/c9-s12-write-server` tip **`e7e6615`** (9 pins S12A-1..S12A-9; parent `55d6797`, tunnel base `9acb47c`; re-read at apply K13; spec cites `24adcba` as authoring tip on base `e4b42b0`)
**Repo**: `pancaddia-leon-tunnel` · **Worktree**: tunnel branch off `9acb47c`
**D-ids**: D6a, D6b, D6c · **Gate**: R4.1-R4.8, SC-4
**Apply-package input for PR1**: niagara-research S20 rev 2 `6c3bcf107` — re-read at apply.
**Product decision (viewer step-up)**: viewer step-up uses ONE shared `cfg.CONFIG_PASSWORD` bound to the JWT identity via the `x-config-token` header (proposal `f610d21`); TTL defaults HMI 5 min / viewer 10 min sliding — both injected via a `Clock` interface so the pins are clock-injectable (not wall-clock sleeping).

- [x] 4.1 Re-read RED tip `e7e6615` (K13; parent `55d6797`, rebased onto `9acb47c`): confirm S12A-1..9 pin text; confirm S12A-8 — a FAILED station write (e.g. 502) still produces exactly ONE `change_log` row with `ok:false` and `result = failing HTTP status`; confirm S12A-9 — `replaySpool` is pinned (see PR5 step 5.5); grep the diff for any `user+password` JSON store — must be zero (R4.7).
- [x] 4.2 Cherry-pick / merge RED `e7e6615` into `feat/c9-s12-config-login` as commit 1.
- [x] 4.3 Refactor `instalacion/pipeline/write-server.mjs`: extract `buildServer(cfg, deps)` returning the handler without binding a port; guard `main()` with `import.meta.url` guard at `:306-309` (D6c); `deps = { obix, verifyToken, changeLog }` (RED shape wins over design's `{changeLog, station, clock, spool}`).
- [x] 4.4 Implement server-held token store: mints a random opaque token (crypto.randomBytes 32) bound to viewer JWT email; no credentials persisted to file (R4.2, D6c).
- [x] 4.5 Implement `POST /config/login`: validates ONE shared `cfg.CONFIG_PASSWORD` (NOT a JSON user+password store); on success mints config token; returns opaque handle to client (R4.2).
- [x] 4.6 Implement `POST /config/logout`: deletes token immediately; subsequent `/write` with revoked token → 403 (R4.3; 403 not 401 — the seam returns 403 config_login_required).
- [x] 4.7 Gate `POST /write` and `POST /alarms/ack` on the config token; read endpoints (`/health`) unchanged and ungated (R4.5). Token gate returns 403 when header missing or session absent.
- [x] 4.8 Confirm non-allowlisted ORD path returns **400** (not 403 — D6b decision; allowlist re-homed behind the token gate, existing 400 preserved).
- [x] 4.9 Verify the existing best-effort `change_log` insert on `9acb47c` still passes after refactor (R4.8); S12A-4 and S12A-6 re-pinned to land in PR5.
- [x] 4.10 **Named mutation** (OBSERVED): drop the config-token check from the `buildServer` seam → S12A-1 flips ✔→✖ (was 403, now 200 since no token gate) and S12A-5 flips ✔→✖ (stale token is no longer 403). Verbatim: `✖ S12A-1` + `✖ S12A-5` appear; pass count drops from 5 to 3.
- [x] 4.11 Guards: S12A-1/2/3/5/7 GREEN (5/9); S12A-4/6/8/9 RED (PR5 scope — expected); `node --check write-server.mjs` SYNTAX OK; 0 attribution trailers (confirmed `grep -ciE 'co-authored|generated with'` = 0).
- [ ] **[lead]** S12A-1..9 green; OBSERVED token-check mutation flip; no JSON credential store; merge ff-only; ledger acquire + settle `--max-changed-lines 280`.

---

### PR5 — feat/c9-s12-audit-schema (~220)

**RED**: extends `qa/c9-s12-write-server` tip **`e7e6615`** (RED-first schema pins appended: `audit-append-failure`, `audit-row-count`, `audit-migration-additive` + S12A-8 + S12A-9 `replaySpool`; re-read at apply K13)
**Repo**: `pancaddia-leon-tunnel` · **Worktree**: tunnel branch off PR4 merged tip (or `9acb47c`)
**D-ids**: D7 · **Gate**: R5.1-R5.6, SC-3, SC-4
**Apply-package input**: `ffcf04fac` — re-read at apply (companero's R5 package F4 "spool replay" is now pinned as S12A-9).

- [x] 5.1 Re-read RED tip `e7e6615` (K13): confirm `audit-append-failure` pin — forced-fail sink → endpoint returns 200 + ONE spool row + NO 5xx (R5.4); confirm `audit-row-count` — ONE row per successful write, `surface = 'write-server'`, `old_value` from a pre-write GET not from request body (R5.2, R5.3, R5.6); confirm `audit-migration-additive` — no column dropped or retyped; confirm S12A-8 — a FAILED station write produces ONE row with `ok:false` + `result = failing HTTP status`; confirm S12A-9 — `replaySpool` exports as `async function replaySpool(cfg, deps) -> { replayed, remaining }` (apply-package `ffcf04fac`).
- [x] 5.2 Write and apply Supabase migration: add columns `ts`, `config_session`, `result`, `surface`, `client_ip` to `public.change_log` — additive-only, no drop, no retype (R5.1).
- [x] 5.3 Move `auditChange` from module-scope to `deps.changeLog` injection point (D7); move the audit call OUT of the success-only branch (`:270-283`) so a failed write also records an `ok=false` error row (D7).
- [x] 5.4 Implement `cfg.AUDIT_SPOOL` failure spool: JSON-lines, append-only to a configured path; a sink failure → spool entry + endpoint still returns the station's HTTP outcome (R5.5, D7).
- [x] 5.5 Implement `export async function replaySpool(cfg, deps) -> { replayed, remaining }` (S12A-9 pin, apply-package `ffcf04fac`): reads `cfg.AUDIT_SPOOL` and drains entries into `deps.changeLog`; idempotent on a second call (remaining = 0 after a fully successful first call); a failed individual insert leaves the entry in the spool and increments `remaining`; never marks an entry as replayed until the insert confirms.
- [x] 5.6 Confirm `old_value` is captured by a pre-write GET of the target slot from the station — the path at `:266-268` already does this `[CERT]`; ensure it never reads from the request body (R5.2).
- [x] 5.7 **Named mutations** (OBSERVED): (a) spool on success too → S12A-4 zero-spool flips; (b) swallow sink error without spooling → S12A-6 flips; (c) audit only on 2xx → S12A-8 flips (0 rows); (d) skip drain → S12A-9 second replay inserts duplicate.
- [x] 5.8 Guards: `node:test` audit pins + S12A-8 + S12A-9 green; `audit-migration-additive` assertion green; 0 attribution trailers; verify `git log -1` before settle.
- [ ] **[lead]** All audit pins + S12A-8 + S12A-9 green; ONE canonical row per write verified; `replaySpool` idempotency proven; OBSERVED mutation flips; merge ff-only; ledger acquire + settle `--max-changed-lines 220`.

---

### PR6 — feat/c9-s12-servlet-guards (~260)

**RED**: `qa/c9-s12-servlet` tip **`4c18837`** (rebased `a109249`; guard1-5 + guard4b + success/one-audit-entry; `[CERT]`) — re-read at apply (K13)
**Repo**: `angeles725/niagara-panccadia-leon` · **Profile**: `DashboardPan-ux`
**Worktree**: `Cliente/Leon-Guanjuato-worktrees/c9-s12-servlet-guards` (branch off `a109249`)
**D-ids**: D8, D8a, D8b · **Gate**: R6.1-R6.7, SC-3, SC-5
**Cross-PR**: `Dashboard/build.gradle.kts:33` edited here (2.1.1→2.2.0) AND by PR6b — fragment-merge; PR6b must NOT re-bump this line.

- [x] 6.1 Re-read RED tip `4c18837` (K13): confirm guard order 1→2→3→6→4→5; confirm guard4 rejects `""`, `"abc"`, `"NaN"`, `"Infinity"`, `null` with 400 and no `parent.set`; confirm named mutation deletes `:274-288` block → silent-zero returns; confirm guard6 is the config-session slot (wired as a pass-through stub here; real impl in PR6b).
- [x] 6.2 Cherry-pick / merge RED into `feat/c9-s12-servlet-guards` as commit 1 (cherry-picked 4c18837 → 8d53be7).
- [x] 6.3 Create `DashboardWriteGuards.java`: pure static evaluate(xhr, user, operatorWrite, ord, value, AuditSink) with guard order 1→2→3→6(stub)→4→5; guard6 pass-through stub (real impl in PR6b); returns 302/401/403/400/200; no Baja imports.
- [x] 6.4 Change `parent.set(prop, toSet, null)` at `:291` to `parent.set(prop, toSet, op)` where `op = DashboardRbacHelper.resolveUser(kioskName)`; added explicit `catch (PermissionException pe) { resp.setStatus(SC_FORBIDDEN); return; }` before outer catch — NOT a catch-all.
- [x] 6.5 Guard4 numeric block `:274-288` preserved as belt-and-braces D8a AFTER evaluate()'s 200; evaluate() is PRIMARY call (not inside numeric block) — boolean/enum/string slots NOT affected (D8a regression protection intact).
- [x] 6.6 Bumped `Dashboard/build.gradle.kts:33` from `2.1.1` to `2.2.0` — fragment-merge: PR6b reads this same value, does NOT re-bump.
- [x] 6.7 `lint-servlet.sh` exits 0 (WARN-only): 4 WARNs — 2x catch-no-400 (pre-existing), cache-nofinger (pre-existing), csrf-xrw-only (pre-existing). All pre-existing; no new FAILs.
- [x] 6.8 `schema-risk.sh` SAFE — only -ux files changed; no -rt slots touched. verdict=SAFE exit=0 confirmed on source snapshots.
- [x] 6.9 **Named mutation OBSERVED**: (1) replace `if (!isFiniteNumber(value)) return 400;` with comment → WG4+WG4b flip 400→200; (2) skip `sink.append(user, ord, null, value)` → success pin flips (expected:<1> but was:<0>); (3) swallow `if (!operatorWrite) return 403;` → WG3 flips 403→200. All 3 mutations observed and recorded.
- [ ] 6.10 Real-tree smoke (CD10): `lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` on all four client module roots; exact count + subject + absence; `schema-risk.sh` = SAFE. [DEFERRED — PR2/PR3/PR10 tools not yet landed in this session; smoke runs when those PRs merge]
- [x] 6.11 ALL GATES PASSED: 7/7 pure pins green; DashboardDispatchTest 14/14 green (pre-existing, unmodified); build DashboardPan-rt + -ux via Niagara 4.14 mirror; verify-module 14 passed 0 failed; lint-servlet exit 0 (4 pre-existing WARNs); lint-structure 3 pre-existing FAILs (L10 abs-path, DashboardPan-wb L6/L9 empty skeleton); schema-risk SAFE; grep null-literal=0, catch(PermissionException)=1; 0 attribution trailers; tip 19b70b25ee9e0da5bfe26a2cbdc19bf66b1cbfe5; seam PRIMARY path confirmed; resolveOperatorWrite package-private.
- [x] **[lead]** guard1-5 + guard4b + success/one-audit-entry green; 3 OBSERVED mutations; `schema-risk.sh` SAFE; `DashboardPan vendorVersion = 2.2.0` confirmed in build.gradle.kts:33; evaluate() PRIMARY seam wiring confirmed (not belt-and-braces); build + verify-module passed from Niagara 4.14 WSL mirror; tip **19b70b2**; merge ff-only.

**Harness-only**: `AuditEvent` naming the operator in `/PANCCADIA/AuditHistory` — confirmed at the B829-live gate only, never a PR gate.

---

### PR6b — feat/c9-config-login-servlet (~260, size:exception)

**RED**: `qa/c9-s12-config-login` tip **`cc1c948`** (CL1-CL11 pure core + CLW1-CLW5 wiring + SC-13; parent `a109249`) — re-read at apply (K13)
**Repo**: `angeles725/niagara-panccadia-leon` · **Profile**: `DashboardPan-ux`
**Worktree**: `Cliente/Leon-Guanjuato-worktrees/c9-config-login-servlet` (branch off PR6 merged tip)
**D-ids**: D8c, D8c-1 · **Gate**: R14 (all sub-requirements), SC-13
**Apply-package input**: R14 rev 3 `5248a9c58` — re-read at apply; supersedes any earlier R14 rev; treat this as the authoritative shape for CL/CLW pins.
**Depends on**: PR6 merged (guard6 slot in `DashboardWriteGuards` must exist as a stub before PR6b replaces it).
**Cross-PR**: `Dashboard/build.gradle.kts:33` — already set to 2.2.0 by PR6; PR6b must NOT re-bump.
**Cross-PR**: MIR5 re-pin — when PR6b lands, MIR5 in `qa/c9-s12-audit-mirror` changes from `config_session IS NULL` to `config_session = station username`; coordinate with PR7 author.

- [ ] 6b.1 Re-read RED tip `cc1c948` (K13): confirm CLW3 is a REMOVAL pin (assert no `parent.set(…, null)` anywhere in the servlet source after PR6b); confirm CLW4 is explicit `catch (PermissionException)`; confirm CL6 returns 401 (NOT 400) for non-`BPasswordCache` authenticator — deliberate divergence from B830 §830.7 (D8c-1); confirm guard order 1→2→3→**6**→4→5 in full.
- [ ] 6b.2 Cherry-pick / merge RED into `feat/c9-config-login-servlet` as commit 1.
- [ ] 6b.3 Create Baja-free interfaces `Clock`, `UserLookup`, `UserHandle` with the exact symbols from D8c-1; no Baja imports in interface files.
- [ ] 6b.4 Create `ConfigSession(Clock, ttlMs)`: `issue(httpSessionId, username)`, `userFor(httpSessionId)` (touches on hit — sliding window lives here), `revoke(httpSessionId)`; stores station username and timestamps only — no credentials (D8c-1).
- [ ] 6b.5 Create `ConfigLoginGuard(users, sessions)`: `login(...)` → 200 | 401; `logout(...)`; `requireSession(...)` → 200 (renews) | 403 `config_login_required`; `statusForWrite(permissionDenied, auditFailed)` → 403 | 200 truth table (D8c-1); implement CL2/CL5/CL6 gating: `getUser` null → 401, `canLogin` false → 401, `instanceof BPasswordCache` false → 401 (no cast, no validate).
- [ ] 6b.6 Wire Baja adapter: `UserLookup.find → svc.getUser`; `UserHandle.canLogin → svc.canLogin(u)`; `isPasswordCache → authenticator instanceof BPasswordCache`; `validate → BPasswordCache.validate`; `authenticateOk → u.authenticateOk(svc)`; `authenticateFailed → u.authenticateFailed(svc)` (D8c); never call `validate` before `canLogin` (CL5); always call `authenticateFailed` on a wrong password (CL3).
- [ ] 6b.7 Add servlet routes in `DashboardDispatch`: `POST /api/config/login` + `POST /api/config/logout` (CLW1).
- [ ] 6b.8 Replace guard6 stub in `DashboardWriteGuards.evaluate` with real `requireSession(...)` call; on 403, reason is `config_login_required` (CLW5).
- [ ] 6b.9 Each write resolves the `BUser` per request: `svc.getUser(configSessions.userFor(token))` — never cache a `BUser` across requests (stale against a disabled/locked account); call `parent.set(prop, toSet, op)` with the re-authenticated operator's `BUser` as Context (CL9, D8c).
- [ ] 6b.10 Verify `Dashboard/build.gradle.kts:33` reads `2.2.0` (set by PR6); do NOT bump.
- [ ] 6b.11 Confirm `/alarms/ack` is also gated by `requireSession` (additive verify pin): a request without a config session on the `/alarms/ack` path returns 403 `config_login_required`.
- [ ] 6b.12 MIR5 re-pin coordination: record in the PR body that MIR5 in `qa/c9-s12-audit-mirror` must be updated from `config_session IS NULL` to `config_session = station_username` after PR6b merges (D7a re-pin, not supersession); if PR7 has already merged, open a follow-up commit on `qa/c9-s12-audit-mirror` updating MIR5.
- [ ] 6b.13 `schema-risk.sh` SAFE — PR6b touches no slot (schema-neutral, D8c).
- [ ] 6b.14 **Named mutations** (OBSERVED): (a) replace explicit `catch (PermissionException)` with a catch-all → CLW4 flips (403 becomes 500); (b) leave null-Context `parent.set` on any branch → CLW3 flips (REMOVAL pin fails); (c) drop `authenticateFailed` call → CL3 flips (lockout oracle). Record verbatim output for each.
- [ ] 6b.15 Real-tree smoke (CD10): all four client module roots; `schema-risk.sh` = SAFE; exact count + subject + absence.
- [ ] 6b.16 Guards: CL1-CL11 + CLW1-CLW5 green; `schema-risk.sh` SAFE; 0 attribution trailers; rebase onto client main before QA ping; verify `git log -1` before settle.
- [ ] **[lead]** CL1-CL11 + CLW1-CLW5 green; 3 OBSERVED mutation flips; `schema-risk.sh` SAFE; MIR5 re-pin coordinated; merge ff-only; ledger acquire + settle `--max-changed-lines 260 --declare-exception`.

**Harness-only (never green from WSL)**: real station lockout after 5 bad logins (CL3 asserts `authenticateFailed` called on the fake `UserHandle` — that assertion is WSL-ok; that the STATION enforces lockout is Windows `niagaraTest` only); `AuditEvent` naming the second operator in AuditHistory (B830 — harness-only; never count as green from WSL).

---

### PR7 — feat/c9-s12-audit-mirror (~180)

**RED**: `qa/c9-s12-audit-mirror` tip **`0a14df8`** (MIR1-MIR5; parent `9acb47c`) — re-read at apply (K13)
**Repo**: `pancaddia-leon-tunnel` + kit doc · **Worktree**: tunnel branch off `9acb47c` (or after PR5 merged)
**D-ids**: D7a · **Gate**: R7.1-R7.6, SC-6

- [x] 7.1 Re-read RED tip `0a14df8` (K13): confirm MIR1 shape — "flag off = `readAuditHistory` was NEVER CALLED" (not merely zero rows inserted); confirm MIR3 dedupe key is the full 5-tuple `(ts, user, target, old, new)` — NOT just `ts`; confirm `runMirror` returns `{read, inserted, skipped}`.
- [x] 7.2 Cherry-pick / merge RED into `feat/c9-s12-audit-mirror` as commit 1 (a08fa84; rebased onto 08e3ed6 per coordinator mid-task).
- [x] 7.3 Create `instalacion/pipeline/audit-mirror.mjs`: export `async runMirror(cfg, deps)` → `{read, inserted, skipped}`; when `cfg.MIRROR_ENABLED` is falsy, return `{read:0, inserted:0, skipped:0}` immediately WITHOUT calling `deps.readAuditHistory` (MIR1).
- [x] 7.4 Implement 5-tuple dedupe: `deps.changeLog.has(key)` where key = `[ts,user,target,old,new].join('|')`; do NOT key on `ts` alone — two same-tick records must NOT collapse (MIR3, D7a).
- [x] 7.5 Inserted rows carry `surface: 'servlet'` and `config_session: null` (MIR4, MIR5 current pin — MIR5 is re-pinned to station username when PR6b lands, D7a).
- [x] 7.6 Document the reconciliation contract in the kit — [ev:] per row in audit-mirror.mjs header + sql/2026-09-06-change-log-mirror-index.sql; R7.6 ev refs in commit body. (Kit doc line deferred to PR12 per apply-package §3 F6.)
- [x] 7.7 **Named mutations** (OBSERVED): (a) key the dedupe on `ts` alone → MIR3 flips; (b) ignore flag → MIR1 flips; (c) drop has() check → MIR2+MIR3 flip; (d) mislabel surface → MIR4 flips; (e) fabricate config_session → MIR5 flips. All 5 OBSERVED verbatim.
- [x] 7.8 Guards: `node:test` MIR1-MIR5 green (15/15 total); `node --check` SYNTAX OK; import-only probe prints runMirror/mapRecord/isMirrorable; 0 attribution trailers; rebased onto 08e3ed6; tip 9d76b3c.
- [ ] **[lead]** MIR1-MIR5 green; OBSERVED mutation flips; kit doc merged; merge ff-only; ledger acquire + settle `--max-changed-lines 180`.

---

## WAVE 3 — PR8/PR9 parallel → PR10 → PR11 → PR12 → PR13

### PR8 — feat/c9-alarm-cr3 (~260)

**RED**: `qa/c9-alarm-cr3` tip **`70a357b`** (CRA1s/2s/3s/CRA4/CRA5/CRA6; structural + harness-only live routing; parent `a109249`) — re-read at apply (K13)
**Repo**: `angeles725/niagara-panccadia-leon` · **Profile**: `ColdRoomPan-rt`
**Worktree**: `Cliente/Leon-Guanjuato-worktrees/c9-alarm-cr3` (branch off `a109249`)
**D-ids**: D9, D9a · **Gate**: R8.1-R8.6, SC-7
**Cross-PR**: R3↔R8 COUPLED — CR-3 WARN in SP-smoke disappears once PR8 lands; whichever of PR3/PR8 merges second must update the SP-smoke CR-3 pin to "absent".

- [ ] 8.1 Re-read RED tip `70a357b` (K13): confirm CRA4 embeds the `a109249` **21-slot baseline** for `BEvaporatorUnit` (additive-only proof); confirm structural pin set CRA1s/2s/3s + CRA4; confirm harness-only live-routing halves are `skip`-gated in WSL (D9a, retro campaign7 D9 skip-is-not-pass).
- [ ] 8.2 Cherry-pick / merge RED into `feat/c9-alarm-cr3` as commit 1.
- [ ] 8.3 In `BEvaporatorUnit.java`: declare child `BBooleanPoint freezeAlarmPt` with a child `BAlarmSourceExt` carrying a `BBooleanChangeOfStateAlgorithm` (`alarmValue = true`); use additive `add_slot` — do NOT drop or retype any existing slot (R8.1, D9).
- [ ] 8.4 In `recomputeFreeze()` at `:1092`: add one line to drive `freezeAlarmPt.out` from the `freezeTripped` field on change (CRA2s).
- [ ] 8.5 Bump `Paccadia/build.gradle.kts:33` from `2.0.7` to **`2.1.0`** (CRA6).
- [ ] 8.6 `schema-risk.sh` SAFE on real `config.bog` snapshot — additive child point, no retype (R8.4, CRA4).
- [ ] 8.7 R3↔R8 coupling: if PR3 has already merged, coordinate with PR3 author to update the SP-smoke CR-3 assertion to "absent" (CR-3 WARN now resolved by the ext); if PR3 has not yet merged, record the expected post-PR8 SP-smoke state in the PR body so PR3 can update accordingly.
- [ ] 8.8 **Named mutation** (OBSERVED): drop `alarmValue = true` (or remove the `BAlarmSourceExt` entirely) → CRA3s flips GREEN→RED. Record verbatim output (R8.6).
- [ ] 8.9 Real-tree smoke (CD10): all four client module roots; exact count + subject + absence; `schema-risk.sh` = SAFE.
- [ ] 8.10 Guards: CRA1s/2s/3s + CRA4/CRA5/CRA6 green; `schema-risk.sh` SAFE; 0 attribution trailers; rebase before QA ping; verify `git log -1` before settle.
- [ ] **[lead]** Structural pins green; OBSERVED mutation flip; `schema-risk.sh` SAFE; `ColdRoomPan-rt vendorVersion = 2.1.0` confirmed; merge ff-only; ledger acquire + settle `--max-changed-lines 260`.

**Harness-only (never green from WSL)**: CRA1/CRA2/CRA3 live-routing halves — `BAlarmSourceExt` routes only inside a running station; `skip`-gated in WSL; must be run on Windows `niagaraTest` and recorded as a real harness run (a SKIP is not a PASS — retro campaign7 D9).

---

### PR9 — feat/c9-alarm-cp1 (~280)

**RED**: `qa/c9-alarm-cp1` tip **`8b43488`** (pure `AlarmEdge` contract + wiring pins; CPB5 harness-only; parent `a109249`) — re-read at apply (K13)
**Repo**: `angeles725/niagara-panccadia-leon` · **Profile**: `CompPan-rt`
**Worktree**: `Cliente/Leon-Guanjuato-worktrees/c9-alarm-cp1` (branch off `a109249`)
**D-ids**: D10, D10a · **Gate**: R9.1-R9.8, SC-7
**Cross-PR**: `Compresores/build.gradle.kts:33` — PR1 set to 2.1.0; PR9 bumps to 2.2.0 (fragment-merge; verify PR1 merged before PR9 bumps).

- [ ] 9.1 Re-read RED tip `8b43488` (K13): confirm `CompressorControl.AlarmEdge` nested class with `decide(trip, nowOffnormal, recoveredPastDeadband)` → `FIRE | CLEAR | NONE` (static final int); confirm `reseed(boolean[] current)` and `wasOffnormal(int trip)`; confirm CPB5 is `skip`-gated (harness-only); confirm wiring pins structural (D10a).
- [ ] 9.2 Cherry-pick / merge RED into `feat/c9-alarm-cp1` as commit 1.
- [ ] 9.3 Add static nested class `CompressorControl.AlarmEdge(trips)` in `CompressorControl.java`: constants `FIRE`, `CLEAR`, `NONE`; `decide` implementing the 3-way edge rule (D10a); `reseed` seeds from current condition without firing; `wasOffnormal` accessor.
- [ ] 9.4 In `BCompressorControl.java`: implement `BIAlarmSource`; declare `@NiagaraAction BBoolean ackAlarm(BAlarmRecord)`; create transient `AlarmSupport support = new AlarmSupport(this, "defaultAlarmClass")` in `started()`; call `support.reseed(currentConditions)` in `started()` (R9.4); in `step`: call `alarmEdge.decide(trip, condition, recoveredPastDeadband)` and map `FIRE → support.newOffnormalAlarm(dataFor(trip))`, `CLEAR → support.toNormal(BFacets.DEFAULT, null)` (D10a).
- [ ] 9.5 Bump `Compresores/build.gradle.kts:33` from `2.1.0` to **`2.2.0`** — confirm PR1 merged first (2.1.0 is there); do NOT regress to 2.1.0.
- [ ] 9.6 `schema-risk.sh` SAFE — additive `BIAlarmSource` references + transient field; no slot dropped or retyped (R9.7).
- [ ] 9.7 **Named mutations** (OBSERVED): (a) make `decide` level-triggered (return `FIRE` whenever `nowOffnormal`) → CPB2 re-fire pin flips GREEN→RED; (b) drop `started()` `reseed` call → CPB4 restart-re-seeds pin flips (R9.8). Record verbatim output for each.
- [ ] 9.8 Real-tree smoke (CD10): all four client module roots; exact count + subject + absence; `schema-risk.sh` = SAFE.
- [ ] 9.9 Guards: CPB1-CPB4 + CPB6/CPB7 green; `schema-risk.sh` SAFE; 0 attribution trailers; rebase before QA ping; verify `git log -1` before settle.
- [ ] **[lead]** Pattern B pins green; OBSERVED edge-only + reseed mutation flips; `schema-risk.sh` SAFE; `CompPan-rt vendorVersion = 2.2.0` confirmed; merge ff-only; ledger acquire + settle `--max-changed-lines 280`.

**Harness-only (never green from WSL)**: CPB5 — `sourceState` on the routed `BAlarmRecord` can only be verified inside a running station; `skip`-gated in WSL; must be recorded as a Windows `niagaraTest` harness run.

---

### PR10 — feat/c9-ext-writable-shape (~230)

**RED**: `qa/c9-ext-writable-shape` tip **`3726722`** (EW1-EW10; `[CERT]`) — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools/c9-ext-writable-shape` (branch off kit main `d2857d1`)
**D-ids**: D2, D2a, D2b, D5, D5a · **Gate**: R10.1-R10.9, SC-8, SC-9

- [ ] 10.1 Re-read RED tip `3726722` (K13): confirm EW10 asserts real `BRoomPanel.setpoint` WARN with subject name; confirm plain `double`/`boolean`/`BRelTime` produces no WARN; confirm complex-with-`@NiagaraAction` produces no WARN; confirm row grammar `WARN  lint-ext-writable-shape  <file>:<line>  <reason>` (D2a).
- [ ] 10.2 Cherry-pick / merge RED into `feat/c9-ext-writable-shape` as commit 1.
- [ ] 10.3 Write `toolbelt/lint-ext-writable-shape.sh`: parse `@NiagaraProperty` blocks using the `lint-write-path.sh:310-343` awk technique (D5); WARN when `Flags.OPERATOR` (or `"o"`) + complex type in `{BStatusNumeric, BStatusBoolean, BStatusEnum}` + no `@NiagaraAction` writing action in the declaring class; exemptions per D5; `LC_ALL=C`; `set -u`; dot-dirs pruned (D9b); exits 0/1/3 (K20); `shellcheck 0.10.0` clean (CD6).
- [ ] 10.4 Write `tests/lint-ext-writable-shape.bats` (EW1-EW10 verbatim from RED; EW10 uses real `BRoomPanel.setpoint` production fixture; inline heredocs into `$BATS_TEST_TMPDIR`).
- [ ] 10.5 Add K19 routing: `BUILD-LOOP.md` §5 + `skill/SKILL.md` for `lint-ext-writable-shape.sh [ev: retro c9-ext-writable-shape]` (CD5, R10.8).
- [ ] 10.6 Extend `toolbelt/report-module.sh`: append member row for `lint-ext-writable-shape`; FAIL → aggregate FAIL (R10.9).
- [ ] 10.7 **Named mutation** (OBSERVED): drop the `@NiagaraAction` exemption → a clean complex-with-action slot starts WARNing; EW1 flips. Record verbatim output.
- [ ] 10.8 EW10-smoke (CD10): `lint-ext-writable-shape.sh` on all four client module roots — `BRoomPanel.setpoint` appears in WARN output with subject name asserted (EW10); exact count matches expected; no unexpected subjects appear (absence assertion); no dot-dir traversed. NOTE: CompPan-rt is a natural absence pin — every OPERATOR slot there is plain (D5 `[CERT]`); assert ZERO rows for CompPan-rt.
- [ ] 10.9 Guards: `bats tests/lint-ext-writable-shape.bats` all green; `shellcheck 0.10.0` exit 0; `sweep-build-state.sh`; `sweep-fold-audit.sh --strict`; `kit-links.bats`; 0 attribution trailers; rebase before QA ping; verify `git log -1` before settle.
- [ ] 10.10 Retro file + `retros/INDEX.md` row + `BUILD-STATE.md` self-envelope (CD1).
- [ ] **[lead]** EW1-EW10 green; EW10-smoke green with `BRoomPanel.setpoint` subject + CompPan-rt zero-row absence; OBSERVED mutation flip; K19 routing; merge ff-only; ledger acquire + settle `--max-changed-lines 230`.

---

### PR11 — docs/c9-write-path-rows (~150)

**RED**: `qa/c8-write-path` tip **`5e357d1`** (extended to uncovered slots; re-read at apply K13)
**Repo**: `angeles725/niagara-panccadia-leon` · **File**: `docs/write-path-matrix.md` (client repo root — D11a)
**Worktree**: `Cliente/Leon-Guanjuato-worktrees/c9-write-path-rows` (branch off `a109249`)
**D-ids**: D11, D11a, D11b, D11c · **Gate**: R11.1-R11.4, SC-9
**Depends on**: PR1 merged (PR1 carries 2 S20 matrix rows; PR11 authors remaining 62; total 64).

- [ ] 11.1 Re-read RED tip `5e357d1` extended (K13): confirm SC-9 pin is exit **1→0** (NOT 3→0 — the matrix exists at `a109249` with 20 rows, so `lint-write-path.sh` exits 1 today; D11b); confirm each W row must cite real slot name, declaring class, and write mechanism.
- [ ] 11.2 Confirm PR1 merged; re-measure uncovered slots on the merged tip — expect `lint-write-path.sh` exits 1 with FAIL rows for ~62 uncovered slots (20 original + 2 S20 rows now covered = 22 covered of ~84 total OPERATOR slots, per D11c; DashboardPan-rt is the bulk with 41 uncovered).
- [ ] 11.3 Author 62 rows in `docs/write-path-matrix.md` (extend from current 22): ColdRoomPan-rt 6 + CompPan-rt 15 + DashboardPan-rt 41 + DashboardPan-ux 0 (D11c measured at `a109249`); each row cites the real writable slot name, its declaring class, and the write mechanism (R11.4); verify against actual module source at the chain's client tip.
- [ ] 11.4 **Named mutation** (OBSERVED): delete one row → that slot FAILs again; `lint-write-path.sh` exits 1. Record verbatim output.
- [ ] 11.5 W14-W22 smoke (R11.2-R11.3): `lint-write-path.sh` on all four client module roots — all roots with a matrix exit 0; a root with no matrix exits 3 (never a silent 0).
- [ ] 11.6 Guards: `lint-write-path.sh` exits 0 on all client module roots; 0 attribution trailers; rebase before QA ping; verify `git log -1` before settle.
- [ ] **[lead]** Exit 1→0 confirmed; all rows cite real slots; OBSERVED deletion mutation; merge ff-only; ledger acquire + settle `--max-changed-lines 150`.

---

### PR12 — docs/c9-doctrine (~200, doc-only)

**RED**: none (CD2 — zero new bats tests by design)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools/c9-doctrine` (branch off kit main `d2857d1`)
**D-ids**: D12 · **Gate**: R12.1-R12.10, SC-11
**K22 already present at `METHODOLOGY.md:86`** — PR12 adds idempotent presence guard ONLY; NEVER re-folds or duplicates K22.

- [x] 12.1 `grep-before-fold` (K6): `rg 'Protection anatomy|BIAlarmSource|AlarmSupport|fragment-merge|OBSERVED flip|module-root.*profile' build-n4-module-kit/` → confirm 0 hits for genuinely new content; `grep -n 'K22' METHODOLOGY.md` → expect exactly `:86` (K22 exists; do NOT author it again).
- [x] 12.2 Add `types/logic.md` §"Protection anatomy" (NEW §): Pattern A (declarative child `BBooleanPoint` + `BAlarmSourceExt`); Pattern B (`BIAlarmSource` + `AlarmSupport.newOffnormalAlarm` on the offnormal EDGE only); both route `sourceState = offnormal`; both `add_slot`-SAFE; "folded as code: `lint-silent-protection.sh`" prose line; `[ev: corpus B827]` `[ev: corpus B821 §821.4]` per R12.2.
- [x] 12.3 VERIFY ONLY — slot-type doctrine already at `types/logic-authoring.md:62-70` (C8 PR15); added lint-ext-writable-shape.sh cross-ref at end of section (PR10 landed). No duplicate text authored.
- [x] 12.4 Add `types/dashboard.md` one-canonical-sink unified write audit doctrine (B829/B830); cross-reference to `types/logic-authoring.md` §Slot types already in the slot-type line at :33 (R12.3/R12.9).
- [x] 12.5 Add `BUILD-LOOP.md` §5 ONE K22 cross-reference bullet (module-root/profile convention; exit 3 no silent 0). K22 text at METHODOLOGY.md:86 untouched (R12.4).
- [x] 12.6 Add `METHODOLOGY.md` §Kit maintenance: (a) OBSERVED-flip mutation table rule [ev: retro campaign8-close-process-meta-lessons]; (b) fragment-merge protocol for the four always-conflict kit files [ev: retro campaign8-close-process-meta-lessons] (R12.6/R12.7).
- [x] 12.7 Add `BUILD-LOOP.md` §7 lead merge/settle order: merge ff-only → verify `git log -1` = blessed tip → THEN settle; rebase parallel workers before QA ping (R12.8).
- [ ] 12.8 Add presence guard to `tests/c9-close.bats`: assert exactly ONE `**K22 —` line in `METHODOLOGY.md` — DEFERRED to PR13 (PR12 = CD2 zero new bats tests; this guard belongs in the close-gate suite).
- [x] 12.9 `sweep-fold-audit.sh --strict` exits 0 (77 folded, 77 cited, 0 uncited); `kit-links.bats` exits 0 (L1-L8 all pass); dangling `campaign9-s12-write-audit` = 0; K19 routing lines NOT re-added (already in PR2/PR3/PR10) (R12.10).
- [x] 12.10 Guards: `sweep-fold-audit.sh --strict` exit 0; `kit-links.bats` 8/8; 0 attribution trailers; bats 368/368 green; tip commit `41d1003` on branch `feat/c9-pr12`.
- [x] 12.11 Retro `campaign9-doctrine-fold` filed (7 deltas); `retros/INDEX.md` row appended; `BUILD-STATE.md` retro_pending: true (CD1). C10 seeds S21/S22 in retro body.
- [x] **[lead]** Doctrine sections land with `[ev:]` per paragraph; `sweep-fold-audit.sh --strict` exit 0; `kit-links.bats` 8/8; K22 NOT duplicated (presence at :86 verified). Tip: `41d1003`. Gate outputs below.

---

### PR13 — chore/c9-close (~180)

**RED**: `tests/c9-close.bats` under `C9_CLOSE=1` — tip **`30e22f9`** — re-read at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools/c9-close` (branch off kit main `d2857d1`)
**D-ids**: D13 · **Gate**: R13.1-R13.6, SC-12

- [ ] 13.1 Re-read close bats tip `30e22f9` (K13): confirm it asserts `VERSION = 0.20.0`; `CHANGELOG.md` has `## 0.20.0` section; `retros/INDEX.md` pending = 0; `sweep-build-state.sh` exits 0; `sweep-fold-audit.sh --strict` exits 0; K22 exactly one occurrence; `shellcheck 0.10.0` on all `toolbelt/*.sh`; zero attribution trailers across the full PR1-PR13 range.
- [ ] 13.2 Write `tests/c9-close.bats` (gated on `C9_CLOSE=1`; mirror `tests/c8-close.bats` shape): assertions per R13.1; `shellcheck 0.10.0` assertion over `toolbelt/*.sh`; attribution trailer scan over the PR1-PR13 range (0 trailers — K11/CD7); K22 idempotent guard (exactly ONE `**K22 —` line in `METHODOLOGY.md`).
- [ ] 13.3 Bump `VERSION` to `0.20.0`; update `CHANGELOG.md` — rename `## [Unreleased]` → `## [0.20.0] - {close-date}` and add `### References` block; entries for `lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` (R13.5).
- [ ] 13.4 Confirm `retros/INDEX.md` pending = 0 (R13.4): every kit-changing PR (PR2, PR3, PR10 — and PR7 if the tunnel reconciliation doc was considered kit-touching) has its retro row filed and indexed.
- [ ] 13.5 `kit-links.bats` exits 0 (R13.6): all three new scripts (`lint-demand-scope.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh`) are reachable in both `BUILD-LOOP.md` §5 and `skill/SKILL.md`.
- [ ] 13.6 Guards: `bats tests/c9-close.bats C9_CLOSE=1` green; `sweep-build-state.sh` green; `sweep-fold-audit.sh --strict` green; `shellcheck 0.10.0` exits 0 on all modified `toolbelt/*.sh` across PR1-PR13; 0 attribution trailers in full range; rebase before QA ping; verify `git log -1` before settle.
- [ ] **[lead]** Close bats green under `C9_CLOSE=1`; `retros/INDEX.md` pending = 0; `VERSION = 0.20.0`; merge ff-only; ledger acquire + settle `--max-changed-lines 180`.

---

## Cross-PR Constraint Registry

| Constraint | PRs | Rule |
|---|---|---|
| `Compresores/build.gradle.kts:33` fragment-merge | PR1 (2.0.3→2.1.0), PR9 (2.1.0→2.2.0) | PR1 sets 2.1.0; PR9 bumps to 2.2.0; never regress; verify PR1 merged before PR9. |
| `Dashboard/build.gradle.kts:33` fragment-merge | PR6 (2.1.1→2.2.0), PR6b (verify same) | PR6 sets 2.2.0; PR6b reads 2.2.0 — NOT re-bumped. |
| CR-3 smoke pin coupling | PR3 (SP-smoke: CR-3 WARN), PR8 (removes WARN) | Whichever merges second updates the SP-smoke CR-3 pin to "absent". |
| PR11 sequenced after PR1 | PR1, PR11 | PR1's 2 S20 matrix rows must be in the matrix before PR11 is verified; exit 1→0 requires them. |
| PR6b depends on PR6 | PR6, PR6b | guard6 stub in `DashboardWriteGuards` must exist before PR6b replaces it with real config-session logic. |
| MIR5 re-pin | PR7 (MIR5 = null), PR6b (triggers re-pin) | After PR6b merges, update MIR5 in `qa/c9-s12-audit-mirror` from `config_session IS NULL` to station username; if PR7 already merged, open a follow-up commit. |
| SC-4 amendment | PR6 body, PR12 §6 note | The proposal's SC-4 "403 for non-allowlisted ORD" is corrected to "400" per D6b; PR6 records this in its PR body; PR12 doctrine cross-references D6b. |
| K22 idempotent guard | PR12, PR13 | `METHODOLOGY.md:86` already has K22; PR12 adds presence guard; PR13 close bats verifies exactly one occurrence. |
| `/alarms/ack` requireSession | PR6b | Additive verify pin: a request to `/alarms/ack` without a config session must return 403. |
| Harness-only never green from WSL | PR8 (CRA1/2/3), PR9 (CPB5), PR6b (AuditEvent + real lockout) | These pins are `skip`-gated in WSL; they require Windows `niagaraTest`; a SKIP is not a PASS. |

> Pin-name note (RED wins, per QA 2026-09-06): the field-scan mutation is **DS3** (not DS2), the reseed-drop mutation is **CPB_W4**, the `@NiagaraAction` exemption is **EW3**; the S7 bats file is `tests/demand-in-scope.bats`; the client version key is `defaultModuleVersion(...)` in the GROUP `build.gradle.kts`. QA's executable mutation table (`qa/c9-verify-runbooks` a406f6e, `c9-mutations.tsv`) names the authoritative pins. The lint REDs' real-tree smoke root is `C9_CLIENT_ROOT` (default: the a109249 worktree), never the stale `Cliente/Leon-Guanjuato` checkout.
