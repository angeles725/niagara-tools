# C10 verify runbooks — execute-only (QA)

Five lint-precision fixes (S21-S25) + the PR6 client concept-marking + the close. Base kit main **df8c7ec**
(the RED base; kit main is now 2ff4a6e — the archive/spec commits are doc-only, the lints are byte-identical).
RED tips: S21 52ebd11 · S22 954ebd7 · S23 f981754 · S24 a792d7a · S25 a56a72e. PR map (natural order; confirm
at freeze): PR1=S21 PR2=S22 PR3=S23 PR4=S24 PR5=S25 PR6=client-concept PR7=close.

## Conventions (every kit PR)
```
KIT=~/modulos_niagara_n4/niagara-tools ; KWT=~/modulos_niagara_n4/niagara-tools-worktrees
B=~/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659   # matrix/covered read tree (post-C9)
```
- V0 fresh detached worktree at the tip; V1 base (`merge-base --is-ancestor origin/main HEAD`), 0 merges/markers,
  0 attribution trailers (`git log <base>..HEAD --format=%B | grep -ciE 'co-authored|generated with|claude'`).
- V2 the RED bats file is byte-identical to the cited RED tip (a PR may only ADD/GREEN, never edit the RED).
- V3 `bats tests/` all green; the S-item's new pins are now GREEN; shellcheck 0; sweeps 0/0; kit-links 8/8.
- V4 the named mutation (qa/c10-mutations.tsv) on a mktemp copy of the FIXED toolbelt/<lint>.sh → the flip OBSERVED, restore.
- V6 BLESS @ tip; V7 `origin/main == tip` after the ff merge.

## PR1 — S21 lint-timers companion-flag (FIELD not method-local) · RED 52ebd11
- V3: `bats tests/lint-timers.bats` all green incl. S21-neg (method-local → clean), S21-pos (field flag → FAIL), S21-smoke.
- Real-tree: `$KIT/build-n4-module-kit/toolbelt/lint-timers.sh $B/Paccadia/ColdRoomPan/ColdRoomPan-rt/src` → **exit 0**, no companion-flag (was exit 1 on `anyNoHardware` at df8c7ec).
- V4: M-S21 (drop the field-scope check) → S21-neg FAILs + smoke exit 1.

## PR2 — S22 ext-writable per-slot @NiagaraAction · RED 954ebd7
- V3: `bats tests/ext-writable-shape.bats` green incl. EW-s22-pos (writer exempt), EW-s22-neg (unrelated action → WARN), EW-s22-neg2 (doAckAlarm → faultReset WARNs), EW10 (CompPan-rt 1).
- Real-tree exact at ff1b659: `lint-ext-writable-shape.sh $B/Dashboard/DashboardPan/DashboardPan-rt/src` → 1 WARN (BRoomPanel.setpoint); `…/Compresores/CompPan/CompPan-rt/src` → **1 WARN faultReset** (annotation :381; pin by NAME, count 1); ColdRoomPan-rt & DashboardPan-ux → 0.
- V4: M-S22 (skip do<Action> mapping) → EW-s22-neg2 flips clean + EW10 CompPan back to 0.

## PR3 — S23 silent-protection Pattern-B surface · RED f981754
- V3: `bats tests/lint-silent-protection.bats` green incl. S23-pos (trip + BIAlarmSource adapter → clean), S23-neg (trip, no surface → WARN), SP-smoke.
- Real-tree exact at ff1b659: `lint-silent-protection.sh $B/Compresores/CompPan/CompPan-rt/src` → **0 WARN** (CP-1 :294 surfaced by the adapter; was 1); ColdRoomPan-rt 0, DashboardPan-rt/ux 0.
- V4: M-S23 (remove the BIAlarmSource/newOffnormalAlarm token from file_has_alarm) → SP-smoke CompPan back to 1, S23-pos WARNs.

## PR4 — S24 run-pure-test cwd-independence · RED a792d7a
- V3: `bats tests/run-pure-test.bats` green incl. S24-cwd (structural test via the runner from a non-profile cwd → pass), S24-cwd-regression.
- Verify the fix: the runner `cd "$rt"` before the java line; a real structural test works from any cwd:
  `( cd / && $KIT/build-n4-module-kit/toolbelt/run-pure-test.sh $B/Paccadia/ColdRoomPan/ColdRoomPan-rt com.angeles.ColdRoomPan.FreezeAlarmWiringTest )` → OK (5 tests) [needs the client srcTest + junit jars].
- V4: M-S24 (remove the cd) → S24-cwd FAILs.

## PR5 — S25 lint-write-path STALE + --strict · RED a56a72e
- V3: `bats tests/lint-write-path.bats` green incl. WP-stale-neg/-strict/-regression/-concept/-concept-decoy/-perrow/-prose/-action/-summary/-smoke + WP-uncovered-strict; WP1/WP2 unchanged.
- Real-tree exact at ff1b659 (STALE is matrix-root-scoped): `lint-write-path.sh --strict $B/Paccadia/ColdRoomPan/ColdRoomPan-rt` → **exit 1, exactly 5 STALE** (hoaMode ×3 :31/:32/:52, inhibit :33, freezeEnabled :36); a SECOND root (`…/Compresores/CompPan/CompPan-rt`) → same 5. Without `--strict` the exit is the per-module FAIL contract (0 covered / 1 uncovered), unchanged.
  Reference (companero bcd02efe6, VERIFIED here): covered = 177 names via the MULTI-LINE-safe `name = "X"` field match (a single-line @Niagara regex under-counts). :40 first backtick is `setpoint` (covered); :64/:65 intervalExpired/forceDefrost are actions (covered); inhibit is NOT --bog-traced.
- V4: M-S25a (OPERATOR-only harvest) → smoke 7 not 5; M-S25b (skip comment-strip) → decoy exempts; M-S25c (dedupe by name) → per-row collapses.

## PR6 — client concept-marking (the STALE 5→0 flip) · client repo
- The five STALE rows get the literal `[concept]` token in `docs/write-path-matrix.md` (:31/:32/:52 hoaMode, :33 inhibit, :36 freezeEnabled). No source change.
- OBSERVED flip: on the pre-PR6 tree `lint-write-path.sh --strict <root>` → 5 STALE; on the PR6 tree → **0 STALE**, exit 0.
- Tracked-jar proof (if PR6 also rebuilds/commits jars per repo convention): `git ls-files '*/build/libs/*.jar'` before vs after — the same jar paths tracked, only content changed; `git diff --stat` shows only the matrix + the jars, no source `.java` change.

## PR7 — close · c10-close.bats (qa/c10-close-checklist 41bca42, C10_CLOSE-guarded)
- `C10_CLOSE=1 C10_CLOSE_COMMIT=<close sha> C9_CLIENT_REPO=$B C10_VERSION=<target> bats tests/c10-close.bats` → all green except CLOSE-tag (pre-tag) and CLOSE-harness-run (still pending the Windows niagaraTest session — qa/c9-harness-runsheet.md). Fill the TODO(freeze) VERSION/tag/SC-13.
