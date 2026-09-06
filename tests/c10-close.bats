#!/usr/bin/env bats
# Campaign-10 CLOSE gate — SKELETON (mirror of tests/c9-close.bats). Env-guarded on C10_CLOSE so it is
# inert in the normal suite; the close worker fills the TODOs when the C10 scope is frozen.
#
# C10 scope (FROZEN): kit lint-precision — S21 lint-timers companion-flag keys on a FIELD not a
# method-local; S22 ext-writable per-slot @NiagaraAction exemption; S23 silent-protection Pattern-B
# (BIAlarmSource + newOffnormalAlarm) adapter surface; S24 run-pure-test.sh cd "$rt" before java
# (structural cwd); S25 lint-write-path STALE advisory. Plus client S26 (gitignore build cache +
# [concept] rows). No new tool FILES — the kit fixes edit existing toolbelt scripts, so the tool-pins
# loop is the C9 set. VERSION: MINOR 0.21.0 (client work S26 landed, so not a kit-only PATCH). SC-13
# client versions carry over 2.2.0/2.1.0/2.2.0 (S26 bumped no module version — verified).
#
# Run post-hoc:  C10_CLOSE=1 C10_CLOSE_COMMIT=<close sha> C9_CLIENT_REPO=<post-C10 client main worktree> \
#                bats tests/c10-close.bats

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  BASE="1fb63d6"                       # C10 base = the C9 close commit (tag v0.20.0); no-trailer sweep range base
  VERSION_TARGET="${C10_VERSION:-0.21.0}"   # frozen: MINOR 0.21.0 — client work S26 landed, not a kit-only PATCH
  TAG="v${VERSION_TARGET}"
}
_close() { [ -n "${C10_CLOSE:-}" ] || skip "C10 close gate — set C10_CLOSE=1 to run"; }

@test "CLOSE-sweeps: sweep-fold-audit --strict exit 0 AND sweep-build-state exit 0" {
  _close
  run "$REPO/build-n4-module-kit/toolbelt/sweep-fold-audit.sh" --strict "$REPO/build-n4-module-kit/retros/INDEX.md" "$REPO/build-n4-module-kit"
  [ "$status" -eq 0 ]
  run bash "$REPO/build-n4-module-kit/toolbelt/sweep-build-state.sh" "$REPO/build-n4-module-kit/BUILD-STATE.md" "$REPO/build-n4-module-kit/retros" "$REPO/build-n4-module-kit/retros/INDEX.md"
  [ "$status" -eq 0 ]
}

@test "CLOSE-index-pending: retros/INDEX.md has ZERO '| pending |' rows after the fold PR" {
  _close
  [ "$(grep -cE '\| *pending *\|' "$REPO/build-n4-module-kit/retros/INDEX.md")" -eq 0 ]
}

@test "CLOSE-version: VERSION == the C10 target" {
  _close
  [ "$(cat "$REPO/VERSION")" = "$VERSION_TARGET" ]
}

@test "CLOSE-changelog: CHANGELOG.md has a [v<target>] section naming all five C10 kit lint fixes (S21-S25)" {
  _close
  sec="$(awk -v t="## [v${VERSION_TARGET}]" 'index($0,t)==1{f=1;next} /^## \[/{f=0} f' "$REPO/CHANGELOG.md")"
  [ -n "$sec" ] || { echo "no [v${VERSION_TARGET}] section"; false; }
  printf '%s\n' "$sec" | grep -qiE 'lint-timers|companion-flag' || { echo "CHANGELOG missing the lint-timers S21 entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'ext-writable|per-slot'      || { echo "CHANGELOG missing the ext-writable S22 entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'silent-protection|BIAlarmSource' || { echo "CHANGELOG missing the silent-protection S23 entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'run-pure-test|structural'  || { echo "CHANGELOG missing the run-pure-test S24 entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'lint-write-path|write-path|STALE' || { echo "CHANGELOG missing the lint-write-path S25 entry"; false; }
}

@test "CLOSE-no-trailers: no attribution trailer across the campaign range (BASE..HEAD)" {
  _close
  [ "$(git -C "$REPO" log --format=%B "${BASE}..HEAD" | grep -ciE 'co-authored|generated with|claude-session|noreply@anthropic')" -eq 0 ]
}

@test "CLOSE-kit-links: kit-links.bats all green" { _close; run bats "$REPO/tests/kit-links.bats"; [ "$status" -eq 0 ]; }

@test "CLOSE-install-skill: install-skill --dry-run exit 0" { _close; run "$REPO/scripts/install-skill.sh" --dry-run; [ "$status" -eq 0 ]; }

@test "CLOSE-shellcheck: shellcheck exit 0 over toolbelt + scripts" {
  _close
  run bash -c "shellcheck '$REPO'/build-n4-module-kit/toolbelt/*.sh '$REPO'/scripts/*.sh"
  [ "$status" -eq 0 ]
}

@test "CLOSE-tag: tag v<target> exists and points at the close commit" {
  _close
  git -C "$REPO" rev-parse -q --verify "refs/tags/${TAG}" >/dev/null || { echo "tag ${TAG} absent (pre-tag)"; false; }
  [ "$(git -C "$REPO" rev-parse "${TAG}^{commit}")" = "$(git -C "$REPO" rev-parse "${C10_CLOSE_COMMIT:-HEAD}")" ]
}

@test "CLOSE-tool-pins: every campaign pin file passes on main (C9 set — S21/S22/S23 edit existing lints, no new tool files)" {
  _close
  # C10 added no new toolbelt/*.sh; run-pure-test added below — S24 fixed run-pure-test.sh's cwd
  # and its pin (tests/run-pure-test.bats) was missing from this loop.
  for f in lint-delays triage-console lint-timers report-module schema-risk facets-lint \
           slot-coverage rc-scan station-snapshot bog-audit lint-wb-threading verify-module \
           lint-servlet retro-loop lint-structure lint-write-path \
           ext-writable-shape lint-silent-protection demand-in-scope run-pure-test; do
    [ -f "$REPO/tests/$f.bats" ] || { echo "ABSENT pin: $f.bats"; false; }
    run bats "$REPO/tests/$f.bats"
    [ "$status" -eq 0 ] || { echo "FAILED pin: $f.bats"; false; }
  done
}

@test "SC-13 (client group defaultModuleVersion): C10 carry-over 2.2.0/2.1.0/2.2.0 (S26 bumped nothing — verified)" {
  _close
  R="${C9_CLIENT_REPO:-/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659}"
  [ -d "$R" ] || skip "client repo not on this machine (set C9_CLIENT_REPO)"
  v() { grep -ohE 'defaultModuleVersion\("[0-9.]+"\)' "$1" 2>/dev/null | head -1 | grep -oE '[0-9.]+'; }
  # C10 client work (S26) bumped no module version -> versions carry over from C9 (verified at ff1b659/00e7118).
  [ "$(v "$R/Compresores/build.gradle.kts")" = "2.2.0" ]
  [ "$(v "$R/Paccadia/build.gradle.kts")"    = "2.1.0" ]
  [ "$(v "$R/Dashboard/build.gradle.kts")"   = "2.2.0" ]
}

@test "CLOSE-harness-run (CARRIED OVER from C9 — still pending): qa/c9-harness-run.md records three 'Failures: 0, Skips: 0' runs" {
  _close
  f="$REPO/qa/c9-harness-run.md"
  [ -f "$f" ] || { echo "no harness run record — the C9 Windows niagaraTest session (CRA1/2/3-live, CPB5, R14 lockout+AuditEvent) is still owed; qa/c9-harness-procedure.md + the run-sheet"; false; }
  [ "$(grep -cE '^Total tests run: [1-9][0-9]*, Failures: 0, Skips: 0' "$f")" -eq 3 ]
  [ "$(grep -cE 'Skips: [1-9]' "$f")" -eq 0 ]
}
