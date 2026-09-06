# C9 verify runbooks — execute-only (QA, VERIFY authority)

Fourteen work units (PR1-PR13 + R14), branch/worktree names from companero's map
(`niagara-research sources/probes/2026-09-06-c9-branch-worktree-map.md` @ 8ca61da72; R14 branch per the lead:
`feat/c9-s12-hmi-config-login`). Each section is the exact sequence run at GREEN. The RED files are the contract;
where tasks.md prose differs from a RED, the RED wins (retro pin RP4). English artifact.

## 0. Conventions (every PR)

```
KIT=~/modulos_niagara_n4/niagara-tools                 # kit origin main (C9 base c0447c2; docs tip b27e79d)
KWT=~/modulos_niagara_n4/niagara-tools-worktrees
CLI=~/modulos_niagara_n4/Cliente/Leon-Guanjuato        # LOCAL CHECKOUT IS STALE @4f5f1c7 — fetch here, never read
CWT=~/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees   # read tree main-a109249 (origin/main == a109249 at authoring)
TUN=~/tunnel/clientes/Leon-Guanajuato/Pancaddia        # tunnel main 9acb47c
TWT=~/modulos_niagara_n4/pancaddia-tunnel-wt           # (map says …/Pancaddia-worktrees; any throwaway dir works)
QA=$KIT/qa; TB=$KIT/build-n4-module-kit/toolbelt
```
Tool versions on this machine: bats 1.14.0 · shellcheck 0.11.0 (tasks say 0.10.0; exit 0 on either) · node v24.19.0 ·
openjdk 21.0.12.1 · N4.14.0.162 jars (`$N4/modules/{baja,control-rt,alarm-rt}.jar`, `$N4/bin/ext/nre.jar`).

**V0 fresh detached worktree at the tip** (never the worker's worktree, never a real checkout):
```
git -C $REPO fetch -q origin && git -C $REPO worktree add --detach $WT/v-<pr> origin/<branch> && cd $WT/v-<pr>
TIP=$(git rev-parse HEAD); echo "verifying <pr> @ $TIP"
```
**V1 base + hygiene** (all must hold; PIPESTATUS, never the pipe's tail):
```
git merge-base --is-ancestor origin/main HEAD && echo base-ok        # contains CURRENT main (C8 PR13/PR16 lesson)
git log --format=%P origin/main..HEAD | awk 'NF>1' | wc -l             # 0 (no merge commits)
git diff origin/main..HEAD | grep -cE '^\+(<<<<<<<|=======|>>>>>>>)'  # 0 conflict markers
git log --format=%B origin/main..HEAD | grep -ciE 'co-authored|generated with|claude-session|noreply@anthropic'   # 0
git diff --stat origin/main..HEAD | tail -1                             # changed lines vs the ledger budget
```
**V2 the RED still applies byte-identically:** `git diff <RED tip> HEAD -- <RED test files>` → empty. A PR may only ADD
tests. If a RED file changed, stop and diff: the RED wins unless the lead re-pinned it.
**V3 gate** — WSL-countable pins only. Harness-only pins (listed per PR) are EXCLUDED from the count; a SKIP is not a PASS.
**V4 named mutations** (post-green, same detached worktree):
```
$QA/mutate.sh --worktree $WT/v-<pr> --table $QA/c9-mutations.tsv --pr <pr> --out $WT/v-<pr>-mut
```
every `kind=sed` row → `OBSERVED`; `MANUAL` rows: apply by hand on the SAME worktree, run the same suite, paste the
`not ok` / `N)` lines verbatim, `git checkout -- <file>`, `git status --porcelain` empty.
**V5 version key** — GROUP `build.gradle.kts` only (never `vendorVersion`, never `module.xml`, never the module `.kts`):
```
v() { grep -ohE 'defaultModuleVersion\("[0-9.]+"\)' "$1" | head -1 | grep -oE '[0-9.]+'; }
```
**V6 verdict:** `BLESS <pr> @ $TIP` or `REJECT <pr> @ $TIP: <reason>` (English, to the lead).
**V7 after the lead's ff merge:** `git -C $REPO fetch -q && [ "$(git -C $REPO rev-parse origin/main)" = "$TIP" ] && echo "tip == blessed"`
— if main != TIP the lead merged something else or squashed: do not settle.

**Kit common gate** (PR2/PR3/PR10/PR12/PR13; `REPO=$KIT WT=$KWT`):
```
bats tests/                                     # 0 failures (32 files on main + this PR's file(s))
shellcheck -S style build-n4-module-kit/toolbelt/<new>.sh      # exit 0
K=build-n4-module-kit
$K/toolbelt/sweep-fold-audit.sh --strict $K/retros/INDEX.md $K                       # exit 0
bash $K/toolbelt/sweep-build-state.sh $K/BUILD-STATE.md $K/retros $K/retros/INDEX.md   # exit 0
bats tests/kit-links.bats                                       # 0 failures (K19 routing ×2 for a new lint)
scripts/install-skill.sh --dry-run                              # exit 0
```
**Client common gate** (`REPO=$CLI WT=$CWT`): `$QA/junit-run.sh <profile-dir> <fq.TestClass>` (cwd = profile dir, Baja-free
sources auto-picked) → `OK (N tests)`; every PRE-EXISTING srcTest class of the touched module still `OK`
(a109249 set: CompPan-rt `CompressorControlTest`, `CompressorWritePathTest`; ColdRoomPan-rt `ColdRoomControlTest`,
`ColdRoomWritePathTest`, `ColdRoomControlDelayTest`, `ResistanceLockoutTest`, `ColdRoomControlSequenceTest`;
DashboardPan-ux `DashboardDispatchTest`, `JsonUtilTest`). Schema: `$TB/schema-risk.sh $CWT/main-a109249/<profile-dir> $WT/v-<pr>/<profile-dir>`
→ verdict `SAFE` (add_slot only; snapshot dir = module-include.xml + java tree, see tool header).
**Tunnel common gate** (`REPO=$TUN WT=$TWT`): `node --test --test-reporter=tap --test-force-exit <test file>` → `# fail 0`;
`node --check` on every touched `.mjs`; no port bound after the run (`ss -ltnp | grep -c node` → 0; tests set WRITE_PORT=0).
**Real-tree smokes:** the three lint REDs read `C9_CLIENT_ROOT` (default `$CWT/main-a109249`, the blessed tip; RP1 done
2026-09-06). At verify export `C9_CLIENT_ROOT=$CWT/<current-main-worktree>` so the smoke reads the post-merge tree, never the
local working copy `$CLI` (stale @4f5f1c7, carries uncommitted docs).

---

## PR1 — `feat/c9-comppan-rotation` (client CompPan-rt, base a109249) · RED `qa/c9-comppan-rotation` cf28572 · worktree `c9-rotation`
- V2: `git diff cf28572 HEAD -- Compresores/CompPan/CompPan-rt/srcTest/test/com/angeles/CompPan/CompressorRotationTest.java` → empty.
- V3: `$QA/junit-run.sh Compresores/CompPan/CompPan-rt com.angeles.CompPan.CompressorRotationTest` → **OK (17 tests)**
  (ROT1-ROT16, ROT7b, ROT5 golden `100 ×20 110 ×15 111 ×10 011 ×10 001 ×25 000 ×5 010 110 ×14 010 ×20` at rotationInterval=0).
  Pre-existing: `CompressorControlTest`, `CompressorWritePathTest` → OK, same counts as a109249.
- Contract symbols present and only these: `Cfg.rotationIntervalMs`, `Cfg.rotationMode`, `CompressorControl.ROTATION_MAKE_BEFORE_BREAK`, `ctl.swaps`.
- Schema: `schema-risk.sh` → SAFE; new slots exactly `rotationInterval`, `rotationMode` (+ `BRotationMode` frozen enum): `git diff origin/main..HEAD -- '*.java' | grep -cE '^\+.*@NiagaraProperty'` → 2.
- Matrix: PR1 owns its 2 rows: `grep -cE '^\| *(rotationInterval|rotationMode)' docs/write-path-matrix.md` → 2. `$TB/lint-write-path.sh Compresores/CompPan/CompPan-rt --matrix docs/write-path-matrix.md` may still exit 1 until PR11 (record the FAIL count; it must not include the two S20 slots).
- V4: M1a, M1b, M1c (all MANUAL; flips ROT7_/ROT7b_, ROT4_/ROT11_, ROT16_).
- V5: `v Compresores/build.gradle.kts` → **2.1.0** (2.0.3 → 2.1.0; PR9 bumps to 2.2.0 later — PR1 sets ONLY 2.1.0).
- Harness-only: none. Coupling: PR9 requires this PR merged first; PR11 requires its two rows.
- V6/V7.

## PR2 — `feat/c9-demand-scope` (kit) · RED `qa/c9-demand-in-scope` 37ce005 · file `tests/demand-in-scope.bats` · tool `toolbelt/lint-demand-scope.sh` · worktree `c9-demand-scope`
- V2: `git diff 37ce005 HEAD -- tests/demand-in-scope.bats` → empty.
- V3: kit common gate; `bats tests/demand-in-scope.bats` → **9/9** (DS1-DS7 + DS-smoke + DS9). The client tree is on this machine, so a DS-smoke `skip` = REJECT.
- Exit taxonomy: `$TB/lint-demand-scope.sh` → 3 (DS6); empty/no-Java dir → 3 + `ERROR` row, no WARN (DS9); `--strict` on a WARNing tree → 1 (DS5); WARN without `--strict` → 0.
- Real-tree rows (record verbatim, WARN-only tool):
  ```
  for m in Compresores/CompPan/CompPan-rt Paccadia/ColdRoomPan/ColdRoomPan-rt Dashboard/DashboardPan/DashboardPan-rt; do
    $TB/lint-demand-scope.sh $CWT/main-a109249/$m/src; echo "exit=$?"; done
  ```
  expected: exit 0 each; `grep -c 'WARN.*step'` → 0 for CompPan (demandCount gate in scope); other rows recorded, none is a gate.
- K19: routing lines in `build-n4-module-kit/BUILD-LOOP.md` §5 and `skill/SKILL.md` (kit-links); `ci.yml` runs the bats; CHANGELOG `[Unreleased]` names `lint-demand-scope.sh`.
- V4: M2a (DS3), M2b (DS4) — MANUAL. tasks 2.7 says DS2: DS2 is the fixture-level mutant already executable.
- Harness-only: none. V6/V7.

## PR3 — `feat/c9-silent-protection` (kit) · RED `qa/c9-silent-protection` 3b281e0 (SP4 detected-trip shape) · `tests/lint-silent-protection.bats` · tool `toolbelt/lint-silent-protection.sh` · worktree `c9-silent-protection`
- V2: `git diff 3b281e0 HEAD -- tests/lint-silent-protection.bats` → empty (unless the R3↔R8 re-pin below applies).
- V3: kit common gate; `bats tests/lint-silent-protection.bats` → **10/10** (SP1-SP8 + SP-smoke + SP9 no-sources → exit 3 + ERROR).
- SP-smoke is EXACT at a109249 (`C9_CLIENT_ROOT`): CompPan-rt exactly 1 WARN whose subject is `CompressorControl.java:215` (CP-1),
  ColdRoomPan-rt exactly 1 whose subject is `BEvaporatorUnit.java:1287` (CR-3), DashboardPan-rt 0, DashboardPan-ux 0; never `dischargeHighAlarm`,
  `defrostSkipped` or `BCompressorControl` (getters). Total 2 rows on the four roots. (First GREEN c315aae produced 113 — rejected by the lead.)
- **Coupling R3↔R8:** if PR8 is already in client main, the CR-3 row must be ABSENT and the SP-smoke assertion must say so
  (whichever merges second updates the pin). Check `git -C $CLI log --oneline origin/main | grep -c 'c9-alarm-cr3\|CR-3'`.
- V4: M3a (SP8 + SP-smoke), M3b (SP-smoke) — MANUAL.
- Harness-only: none. V6/V7.

## PR4 — `feat/c9-s12-config-login` (tunnel, base 9acb47c) · RED `qa/c9-s12-write-server` e7e6615 · worktree `c9-s12-config-login`
- V0: `git -C $TUN fetch -q origin && git -C $TUN worktree add --detach $TWT/v-pr4 origin/feat/c9-s12-config-login && cd $TWT/v-pr4`; base check against `origin/main` (9acb47c at authoring).
- V2: `git diff e7e6615 HEAD -- instalacion/pipeline/test/write-server.config-login.test.mjs` → empty.
- V3: `node --test --test-reporter=tap --test-force-exit instalacion/pipeline/test/write-server.config-login.test.mjs`
  → `# pass N`, `# fail 0` on the PR4 set. Required `ok` at PR4: **S12A-1, S12A-2, S12A-3, S12A-5, S12A-7** (token gate + seam + §10 bare `<real>` PUT).
  S12A-4, S12A-6, S12A-8, S12A-9 belong to PR5 (record their state; all **9** `ok` if PR4 and PR5 land together).
- Seam facts the RED drives (`startServer` helper, test :55-72): `import('../write-server.mjs')` must not bind or exit (dummy `OBIX_BASE/OBIX_USER/OBIX_PASS/SUPABASE_URL`, `WRITE_PORT=0` set before import; main() guarded by an `import.meta.url` check);
  `export function buildServer(cfg, deps)` returns a non-listening `http.Server`; `cfg = {WRITE_PORT:0, CONFIG_PASSWORD, AUDIT_SPOOL}`; `deps = {obix(method, path, body), verifyToken(bearer)->{email,sub}, changeLog(row)->{ok}}`.
  Header `x-config-token` (proposal f610d21); `/config/login {password}` → 200 `{token}` | 401 no token; `/write` without/with stale token → 403; `/config/logout` revokes.
- `node --check instalacion/pipeline/write-server.mjs`; after the run `ss -ltnp | grep -c node` → 0.
- V4: `$QA/mutate.sh --worktree $TWT/v-pr4 --table $QA/c9-mutations.tsv --pr PR4` → M4s (seam absent → all 9 fail), M4a (token gate dropped → S12A-1, S12A-5), M4b (header name drifted → S12A-4 + the 403 pins). Content-anchored on the RED contract; an ANCHOR-MISSING row is hand-mutated on the same worktree.
- No real smoke possible (Supabase/oBIX injected). Harness-only: none. V6/V7 (`origin/main` of the TUNNEL repo == TIP).

## PR5 — `feat/c9-s12-audit-schema` (tunnel) · same RED e7e6615 · worktree `c9-s12-audit-schema` · after/with PR4
- V0/V2 as PR4 (`$TWT/v-pr5`, branch `feat/c9-s12-audit-schema`). V3: the same `node --test` line → **9/9** `ok`, `# fail 0`.
  Rows the RED pins: S12A-4 exactly ONE `change_log` row per successful write with `ts, config_session, result, surface:'write-server', client_ip` + `user_email, room, slot, old_value, new_value`, and `AUDIT_SPOOL` absent/empty;
  S12A-6 `changeLog` throws → still 200 + exactly ONE JSON line in `cfg.AUDIT_SPOOL`; S12A-8 `deps.obix` → `{status:500}` → HTTP 502 + ONE row `ok:false, result:502`;
  S12A-9 `replaySpool({AUDIT_SPOOL}, {changeLog})` → first `{replayed:1, remaining:0}`, second inserts nothing.
- Migration: `grep -rlE 'config_session' instalacion --include='*.sql'` → the additive migration naming `ts config_session result surface client_ip` (no DROP).
- V4: `--pr PR5` → M5a (spool on success → S12A-4), M5b (spool append no-op → S12A-6, S12A-9), M5c (failed write audited ok:true → S12A-8), M5d (replaySpool seam absent → S12A-9). Harness-only: none. V6/V7.

## PR6 — `feat/c9-s12-servlet-guards` (client DashboardPan-ux) · RED `qa/c9-s12-servlet` 4c18837 · worktree `c9-s12-servlet`
- V2: `git diff 4c18837 HEAD -- Dashboard/DashboardPan/DashboardPan-ux/srcTest/test/com/angeles/DashboardPan/ux/DashboardWriteGuardsTest.java` → empty.
- V3: `$QA/junit-run.sh Dashboard/DashboardPan/DashboardPan-ux com.angeles.DashboardPan.ux.DashboardWriteGuardsTest` → **OK (7 tests)**
  (guard1 302, guard2 401, guard3 403 fail-closed, guard4 400 no-silent-zero, guard4b empty/NaN 400, guard5 ORD traversal 400, success 200 + exactly one audit entry).
  Pre-existing `DashboardDispatchTest`, `JsonUtilTest` → OK.
- Servlet: `$TB/lint-servlet.sh Dashboard/DashboardPan/DashboardPan-ux/src --strict` → exit 0. The null-Context write
  `parent.set(prop, toSet, null)` (a109249 :291) is GONE: `grep -c 'parent.set(prop, toSet, null)' Dashboard/DashboardPan/DashboardPan-ux/src/com/angeles/DashboardPan/ux/BDashboardServlet.java` → 0.
- V4: M6a (guard4_, guard4b_) — sed, anchor `parseFiniteDouble`.
- V5: `v Dashboard/build.gradle.kts` → **2.2.0** (2.1.1 → 2.2.0). Fragment-merge: `git log --oneline origin/main..HEAD -- Dashboard/build.gradle.kts | wc -l` → 1 (PR6 owns the bump; R14 must show 0).
- Harness-only: AuditEvent naming the operator in AuditHistory → harness H3 (`qa/c9-harness-procedure.md`). V6/V7.

## R14 — `feat/c9-s12-hmi-config-login` (client DashboardPan-ux) · RED `qa/c9-s12-config-login` cc1c948 · worktree `c9-config-login` · after PR6
- V2: `git diff cc1c948 HEAD -- Dashboard/DashboardPan/DashboardPan-ux/srcTest/test/com/angeles/DashboardPan/ux/ConfigLoginGuardTest.java Dashboard/DashboardPan/DashboardPan-ux/srcTest/test/com/angeles/DashboardPan/ux/ConfigLoginWiringTest.java` → empty.
- V3: `… com.angeles.DashboardPan.ux.ConfigLoginGuardTest` → **OK (11 tests)** CL1-CL11 (CL6 = **401**, lead-confirmed; B830's 400 superseded);
  `… com.angeles.DashboardPan.ux.ConfigLoginWiringTest` → **OK (6 tests)** CLW1-CLW5 + SC13. PR6's `DashboardWriteGuardsTest` still OK (7).
- Wiring facts: `/api/config/login` + `/api/config/logout` routed (CLW1); legal re-auth path `BUserService.getUser → canLogin → BPasswordCache.validate → authenticateOk/Failed` (CLW2);
  `parent.set(prop, toSet, <user>)` and no null-Context (CLW3); `catch (PermissionException …) → SC_FORBIDDEN` (CLW4); `config_login_required` 403 before the write (CLW5).
- V4: M6b1 (CLW4_), M6b2 (CLW3_), M6b3 (CL3_) — all sed, content-anchored.
- V5 fragment-merge: `v Dashboard/build.gradle.kts` → 2.2.0 (already from PR6) and `git log --oneline origin/main..HEAD -- Dashboard/build.gradle.kts | wc -l` → 0 (no double bump). If PR6 and R14 are verified together, exactly ONE commit in the union touches the line and the value is 2.2.0.
- Greps (SV=`Dashboard/DashboardPan/DashboardPan-ux/src/com/angeles/DashboardPan/ux/BDashboardServlet.java`): CLW3 removal `grep -c 'parent.set(prop, toSet, null)' $SV` → 0 and `grep -cE 'parent\.set\(prop, toSet, [A-Za-z_]+\)' $SV` → ≥1; CLW4 `grep -A4 'catch (PermissionException' $SV | grep -c 'SC_FORBIDDEN'` → 1 and no flat `catch (Exception` between the write and that catch (`grep -nE 'catch \((Exception|Throwable)' $SV` reviewed by hand: none may swallow the PermissionException first); CLW5 `grep -c 'config_login_required' $SV` → ≥1.
- **Coupling MIR5:** after R14 merges, MIR5 in `qa/c9-s12-audit-mirror` re-pins from `config_session IS NULL` to the station username (PR7's owner). If PR7 already merged, file the re-pin as a follow-up and note it in the BLESS.
- Harness-only: real station lockout (5 / 30 s / 10 s) and AuditEvent attribution → H3. V6/V7.

## PR7 — `feat/c9-s12-audit-mirror` (tunnel + kit doc line via PR12) · RED `qa/c9-s12-audit-mirror` 0a14df8 · worktree `c9-s12-audit-mirror` · after PR5
- V2: `git diff 0a14df8 HEAD -- instalacion/pipeline/test/audit-mirror.test.mjs` → empty (unless the MIR5 re-pin applies after R14).
- V0: `git -C $TUN worktree add --detach $TWT/v-pr7 origin/feat/c9-s12-audit-mirror`. V3: `node --test --test-reporter=tap --test-force-exit instalacion/pipeline/test/audit-mirror.test.mjs` → **5/5** MIR1-MIR5, `# fail 0`; `node --check instalacion/pipeline/audit-mirror.mjs`.
  Injection points (test :34-38, :50-90): `runMirror(cfg, deps)`; `cfg.MIRROR_ENABLED` absent/false → `{read:0, inserted:0}` and `readAuditHistory` never called; `deps.readAuditHistory()` → fixture of 3 records, two sharing a `ts`; `deps.changeLog = {rows, insert(row), has(key)}` with `key = [ts,user,target,old,new].join('|')`.
  `runMirror(cfg, deps) -> {read, inserted, skipped}`; `cfg.MIRROR_ENABLED` default OFF (MIR1 reads nothing); dedupe key `(ts,user,target,old,new)`; rows carry `surface:'servlet'`, `config_session:null` (MIR5 as pinned).
- V4: `--pr PR7` → M7s (seam absent → all 5), M7a (key → ts only → MIR3), M7b (`cfg.MIRROR_ENABLED` → true → MIR1), M7c (`changeLog.has(` → false → MIR2, MIR3), M7d (surface mislabeled → MIR4), M7e (config_session fabricated → MIR5). All content-anchored seds.
- Harness-only: none. V6/V7.

## PR8 — `feat/c9-alarm-cr3` (client ColdRoomPan-rt) · RED `qa/c9-alarm-cr3` 70a357b · worktree `c9-alarm-cr3`
- V2: `git diff 70a357b HEAD -- Paccadia/ColdRoomPan/ColdRoomPan-rt/srcTest/test/com/angeles/ColdRoomPan/FreezeAlarmWiringTest.java` → empty.
- V3: `$QA/junit-run.sh Paccadia/ColdRoomPan/ColdRoomPan-rt com.angeles.ColdRoomPan.FreezeAlarmWiringTest` → **OK (5 tests)**
  CRA1s (child `freezeAlarmPt` BBooleanPoint), CRA3s (`BAlarmSourceExt` + `BBooleanChangeOfStateAlgorithm`, alarmValue=true), CRA2s (`recomputeFreeze` drives it from `freezeTripped`), CRA4 (additive-only vs the 21-slot a109249 baseline), CRA6 (2.1.0).
  Pre-existing five ColdRoomPan test classes → OK.
- Schema: `schema-risk.sh` → SAFE (add_slot `freezeAlarmPt` only).
- V4: M8a (CRA3s_) — sed `/alarmValue/s/true/false/`.
- V5: `v Paccadia/build.gradle.kts` → **2.1.0**.
- **Coupling R3↔R8:** SP-smoke's CR-3 row disappears once this lands; if PR3 already merged, this PR (or a kit follow-up the same day) updates `tests/lint-silent-protection.bats` SP-smoke to assert ABSENT.
- Harness-only: CRA1/CRA2/CRA3 live routing → H1. V6/V7.

## PR9 — `feat/c9-alarm-cp1` (client CompPan-rt) · RED `qa/c9-alarm-cp1` 8b43488 · worktree `c9-alarm-cp1` · AFTER PR1
- Pre-check: `git merge-base --is-ancestor <PR1 blessed tip> HEAD` → true.
- V2: `git diff 8b43488 HEAD -- Compresores/CompPan/CompPan-rt/srcTest/test/com/angeles/CompPan/CompressorAlarmEdgeTest.java Compresores/CompPan/CompPan-rt/srcTest/test/com/angeles/CompPan/CompressorAlarmWiringTest.java` → empty.
- V3: `… com.angeles.CompPan.CompressorAlarmEdgeTest` → **OK (5 tests)** CPB1-CPB4 + per-trip independence (`AlarmEdge(int trips)`, `int decide(trip, nowOffnormal, recoveredPastDeadband) -> FIRE|CLEAR|NONE`, `reseed(boolean[])`, `wasOffnormal(trip)`);
  `… com.angeles.CompPan.CompressorAlarmWiringTest` → **OK (5 tests)** W1 `implements BIAlarmSource`, W2 `new AlarmSupport(` in started(), W3 delegation FIRE→newOffnormalAlarm / CLEAR→toNormal, W4 `reseed(` in started(), SC13 (2.2.0).
  `CompressorRotationTest` (17) and the two pre-existing classes still OK.
- Schema: SAFE (no slot change expected: `git diff origin/main..HEAD -- '*.java' | grep -cE '^\+.*@NiagaraProperty'` → 0).
- V4: M9a (CPB2_) — MANUAL; M9b (CPB_W4_) — sed `/\.reseed\(/d` (tasks names CPB4; executable pin is CPB_W4).
- V5: `v Compresores/build.gradle.kts` → **2.2.0**.
- Harness-only: CPB5 `sourceState` on the routed record → H2. V6/V7.

## PR10 — `feat/c9-ext-writable-shape` (kit) · RED `qa/c9-ext-writable-shape` **269be48** (path + root + EW11 + exact EW10 re-issues) · `tests/ext-writable-shape.bats` · tool `toolbelt/lint-ext-writable-shape.sh` · worktree `c9-ext-writable-shape`
- V2: `git diff 269be48 HEAD -- tests/ext-writable-shape.bats` → empty.
- V3: kit common gate; `bats tests/ext-writable-shape.bats` → **11/11** EW1-EW11 (EW10 is EXACT at a109249: DashboardPan-rt 1 WARN `BRoomPanel.setpoint`, CompPan-rt 0 —
  `faultReset` has an action —, ColdRoomPan-rt 0, DashboardPan-ux 0; the client tree is present, a `skip` = REJECT).
- Exit taxonomy: no arg → 3 (EW9); empty/no-Java dir → 3 + `ERROR` row (EW11); `--strict` with a WARN → 1 (EW7); WARN alone → 0.
- Real-tree rows: `$TB/lint-ext-writable-shape.sh $CWT/main-a109249/Dashboard/DashboardPan/DashboardPan-rt/src` → one WARN row for
  `BRoomPanel.setpoint` (BStatusNumeric SUMMARY|OPERATOR, no `@NiagaraAction`) carrying the child `…/value` note; run CompPan-rt and ColdRoomPan-rt too and record rows
  (cross-check with `python3 ~/niagara-research/tools/module-find.py <src> ext-writable`).
- K19 routing ×2, `ci.yml`, CHANGELOG entry `lint-ext-writable-shape.sh`.
- V4: M10a (EW3; tasks says EW1 — EW1 is the WARN pin, EW3 is the clean-with-action pin that flips), M10b (EW1, EW6) — MANUAL.
- Harness-only: none. V6/V7.

## PR11 — `docs/c9-write-path-measured-rows` (client docs, after PR1) · pin = SC-9 exit 0 (extends `qa/c8-write-path` 5e357d1) · worktree `c9-write-path-rows`
- Pre-check: PR1's two rows present (`grep -cE '^\| *(rotationInterval|rotationMode)' docs/write-path-matrix.md` → 2).
- V3:
  ```
  for m in Compresores/CompPan/CompPan-rt Paccadia/ColdRoomPan/ColdRoomPan-rt Dashboard/DashboardPan/DashboardPan-rt Dashboard/DashboardPan/DashboardPan-ux; do
    $TB/lint-write-path.sh $m --matrix docs/write-path-matrix.md; echo "$m exit=$?"; done
  ```
  → every `exit=0`, zero `FAIL lint-write-path` rows. Row count: `grep -cE '^\| ' docs/write-path-matrix.md` minus the 2 header lines = **84** (20 C8 + 2 PR1 + 62); record the exact number.
- Doc-only: `git diff --stat origin/main..HEAD -- '*.java' '*.kts'` → empty.
- V4: M11a — sed `/rotationInterval/d` on the matrix, fmt `exit` → `lint-write-path.sh` exits 1 naming `rotationInterval`.
- Kit side: `tests/lint-write-path.bats` WP-smoke reads `C8_WRITEPATH_ROOT` (a deed38c root) — unaffected. Harness-only: none. V6/V7.

## PR12 — `docs/c9-doctrine` (kit, no RED) · worktree `c9-doctrine` · after PR3/PR10 and the campaign9 retros
- V3: kit common gate (`sweep-fold-audit.sh --strict` → 0, `kit-links.bats` → 0). No script changes: `git diff --stat origin/main..HEAD -- build-n4-module-kit/toolbelt` → empty.
- Presence checks: `grep -c '^- \*\*K22 —' build-n4-module-kit/METHODOLOGY.md` → **1** (never re-folded);
  `grep -n 'Protection anatomy' build-n4-module-kit/types/logic.md` → 1 section; `grep -n 'Slot types for externally written values' build-n4-module-kit/types/logic-authoring.md` → 1;
  `types/dashboard.md` cross-reference + audit reconciliation contract; `BUILD-LOOP.md` §5 one K22 cross-ref line, §6 reconciliation, §7 merge-ff → `git log -1` == blessed → settle order; `METHODOLOGY.md` §Kit maintenance OBSERVED-flips + fragment-merge rule; every new paragraph carries `[ev:]` (the sweep proves it).
- Retro file + `retros/INDEX.md` row + `BUILD-STATE.md` envelope. Harness-only: none. V6/V7.

## PR13 — `chore/c9-close` (kit, LAST) · gate file `tests/c9-close.bats` (branch `qa/c9-close-checklist`) · worktree `c9-close`
- Pre-checks: PR1-PR12 + R14 all `tip == blessed`; `qa/c9-harness-run.md` present with three `Failures: 0` runs (harness H1-H3 done).
- V3: `C9_CLOSE=1 bats tests/c9-close.bats` → every test `ok` except `CLOSE-tag` until the lead tags; after the tag re-run with
  `C9_CLOSE=1 C9_CLOSE_COMMIT=<close sha> bats tests/c9-close.bats` → all `ok`.
  Pins: VERSION 0.20.0 · CHANGELOG `[v0.20.0]` naming the three lint scripts · `retros/INDEX.md` pending 0 · sweeps · shellcheck over `toolbelt/*.sh` · 0 trailers over `c0447c2..HEAD` · tool-pins loop (19 bats files incl. `ext-writable-shape`, `lint-silent-protection`, `demand-in-scope`) · SC-13 versions 2.2.0 / 2.1.0 / 2.2.0 read from `C9_CLIENT_REPO` (default the a109249 worktree — point it at the post-wave-3 client main worktree: `C9_CLIENT_REPO=$CWT/<current-main-wt>`) · K22 exactly once · harness run record.
- V6/V7; then the lead tags `v0.20.0` on the close commit and the post-hoc run above must be all `ok`.
