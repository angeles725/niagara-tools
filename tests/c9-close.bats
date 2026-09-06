#!/usr/bin/env bats
# CAMPAIGN-9 CLOSE GATE — the exact command + expected output that proves each success criterion
# (SC-1..SC-13, openspec/changes/build-n4-module-campaign9/spec.md) on main.
#
# HOW TO RUN: this is a CLOSE gate, not a normal-suite pin. Every test skips unless C9_CLOSE=1, so the
# regular `bats tests/*.bats` stays green mid-campaign. At close, run:
#     C9_CLOSE=1 bats tests/c9-close.bats
# and require ALL green before tagging v0.20.0. Local-smoke criteria (SC1/SC2/SC3/SC16 — real client
# trees / station consoles / bogs) additionally need their env var (C8_CLIENT_REPO, *_CONSOLE_DIR,
# C8_PANCCADIA_BOG, C8_MX60_BOG); they SKIP without it, so the gate is honest about what it could not run.
#
# CLOSE-STATE at authoring (C9 open on main after c0447c2 / v0.19.0): VERSION 0.19.0, no v0.20.0 tag, INDEX
# pending grows as C9 retros land. So CLOSE-version / CLOSE-changelog / CLOSE-tag / CLOSE-index-pending are RED
# until the C9 close PR + tag; that is the gate doing its job. Run post-hoc with C9_CLOSE_COMMIT=<close sha>.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$REPO/build-n4-module-kit"
  BASE="c0447c2"            # campaign-9 base = the campaign-8 close commit; no-trailer sweep range base
}
_close() { [ -n "${C9_CLOSE:-}" ] || skip "campaign-9 close gate — run with C9_CLOSE=1 at close"; }

# ===========================================================================
# META close invariants (SC5, SC6, SC7, SC10-SC13 close aspects)
# ===========================================================================

@test "CLOSE-sweeps (SC-12): sweep-fold-audit --strict exit 0 AND sweep-build-state exit 0" {
  _close
  run "$KIT/toolbelt/sweep-fold-audit.sh" --strict "$KIT/retros/INDEX.md" "$KIT"
  [ "$status" -eq 0 ]
  run bash "$KIT/toolbelt/sweep-build-state.sh" "$KIT/BUILD-STATE.md" "$KIT/retros" "$KIT/retros/INDEX.md"
  [ "$status" -eq 0 ]
}

@test "CLOSE-index-pending (SC-12): retros/INDEX.md has ZERO '| pending |' rows after the fold PR" {
  _close
  [ "$(grep -c '| pending |' "$KIT/retros/INDEX.md")" -eq 0 ]
}

@test "CLOSE-version (SC-12): VERSION == 0.20.0" {
  _close
  [ "$(cat "$REPO/VERSION" | tr -d '[:space:]')" = "0.20.0" ]
}

@test "CLOSE-changelog (SC-12): CHANGELOG.md has a [v0.20.0] section" {
  _close
  grep -qE '^\#\# \[v?0\.20\.0\]' "$REPO/CHANGELOG.md"
}

@test "CLOSE-no-trailers (SC-12): no attribution trailer across the campaign range c0447c2..HEAD" {
  _close
  [ "$(git -C "$REPO" log ${BASE}..HEAD --format=%B | grep -ciE 'co-authored|generated with|claude-session|noreply@anthropic')" -eq 0 ]
}

@test "CLOSE-kit-links (SC-11): kit-links.bats all green (L1-L8, routing + orchestration + post-deploy)" {
  _close
  run bats "$REPO/tests/kit-links.bats"
  [ "$status" -eq 0 ]
}

@test "CLOSE-install-skill (SC-11): the installed SKILL.md matches the tracked source (install-skill --dry-run exit 0)" {
  _close
  [ -f "$HOME/.claude/skills/build-n4-module/SKILL.md" ] || skip "no installed skill copy on this machine"
  run bash "$REPO/scripts/install-skill.sh" --dry-run
  [ "$status" -eq 0 ]
}

@test "CLOSE-shellcheck (SC-12): shellcheck exit 0 over scripts + toolbelt + tests helpers" {
  _close
  command -v shellcheck >/dev/null 2>&1 || skip "shellcheck not installed"
  run shellcheck "$REPO"/scripts/*.sh "$KIT"/toolbelt/*.sh
  [ "$status" -eq 0 ]
}

@test "CLOSE-tag (SC-12): tag v0.20.0 exists and points at the final campaign commit" {
  _close
  git -C "$REPO" rev-parse -q --verify 'refs/tags/v0.20.0' >/dev/null
  # the tag must sit on the close commit: C9_CLOSE_COMMIT when given (post-hoc, main has moved on), else HEAD
  [ "$(git -C "$REPO" rev-parse 'v0.20.0^{commit}')" = "$(git -C "$REPO" rev-parse "${C9_CLOSE_COMMIT:-HEAD}")" ]
}

# ===========================================================================
# Per-tool SC pins — each proven GREEN by its campaign bats file on main.
# (SC1/SC2/SC3/SC16 also carry a REAL smoke, env-guarded below.)
# ===========================================================================

@test "CLOSE-tool-pins (SC-1..SC-10): every campaign pin file passes on main" {
  _close
  # The FINAL pin set: the campaign-8 set plus the C9 kit lints (S19 ext-writable-shape, S18
  # lint-silent-protection, S7 demand-in-scope — SC-8). (C8 note: all present once wave 3 — PR18 lint-structure, PR19 lint-write-path,
  # PR20 station-logic checks CHECK13-19 — has merged). PR20's SL13-19 + SL-smoke pins live in
  # bog-audit.bats (station-logic.bats was the standalone RED file, removed once folded). An absent
  # pin at close is itself a failure.
  for f in lint-delays triage-console lint-timers report-module schema-risk facets-lint \
           slot-coverage rc-scan station-snapshot bog-audit lint-wb-threading verify-module \
           lint-servlet retro-loop lint-structure lint-write-path \
           ext-writable-shape lint-silent-protection demand-in-scope; do
    [ -f "$REPO/tests/$f.bats" ] || { echo "ABSENT pin (campaign not complete): $f.bats"; false; }
    run bats "$REPO/tests/$f.bats"
    [ "$status" -eq 0 ] || { echo "FAILED pin: $f.bats"; false; }
  done
}

# ---- Real local smokes (env-guarded; SKIP is honest "could not run", never a silent pass) ----

@test "SC-13 (client vendorVersion, schema-risk-cleared): CompPan-rt 2.2.0, ColdRoomPan-rt 2.1.0, DashboardPan-ux 2.2.0" {
  _close
  R="${C9_CLIENT_REPO:-/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-a109249}"
  [ -d "$R" ] || skip "client repo not on this machine (set C9_CLIENT_REPO)"
  # the version lives in each GROUP's build.gradle.kts as defaultModuleVersion("X.Y.Z") (not in the module .kts)
  v() { grep -ohE 'defaultModuleVersion\("[0-9.]+"\)' "$1" 2>/dev/null | head -1 | grep -oE '[0-9.]+'; }
  [ "$(v "$R/Compresores/build.gradle.kts")" = "2.2.0" ]   # CompPan: 2.0.3 -> 2.1.0 (R1) -> 2.2.0 (R9)
  [ "$(v "$R/Paccadia/build.gradle.kts")"    = "2.1.0" ]   # ColdRoomPan (R8)
  [ "$(v "$R/Dashboard/build.gradle.kts")"   = "2.2.0" ]   # DashboardPan (R6)
}
