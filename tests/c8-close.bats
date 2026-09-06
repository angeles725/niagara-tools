#!/usr/bin/env bats
# CAMPAIGN-8 CLOSE GATE — the exact command + expected output that proves each success criterion
# (SC1-SC16, openspec/changes/build-n4-module-campaign8/{spec.md,wave3.md}) on main.
#
# HOW TO RUN: this is a CLOSE gate, not a normal-suite pin. Every test skips unless C8_CLOSE=1, so the
# regular `bats tests/*.bats` stays green mid-campaign. At close, run:
#     C8_CLOSE=1 bats tests/c8-close.bats
# and require ALL green before tagging v0.19.0. Local-smoke criteria (SC1/SC2/SC3/SC16 — real client
# trees / station consoles / bogs) additionally need their env var (C8_CLIENT_REPO, *_CONSOLE_DIR,
# C8_PANCCADIA_BOG, C8_MX60_BOG); they SKIP without it, so the gate is honest about what it could not run.
#
# CLOSE-STATE at authoring (633f2bb, mid-close): VERSION 0.19.0, CHANGELOG [v0.19.0] present, no-trailer
# sweep 0, sweeps 0/0, kit-links 8/8 green — BUT INDEX pending = 17 (fold PR not yet landed) and tag
# v0.19.0 not yet cut. So CLOSE-index-pending and CLOSE-tag are RED until the close PR + tag; that is the
# gate doing its job.

load lib/client-root   # C11 T2: one blessed client read root; env override wins [ev: design.md D3b]

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$REPO/build-n4-module-kit"
  BASE="48fb210"            # campaign-8 base commit (pre-PR1); the no-trailer sweep range base
}
_close() { [ -n "${C8_CLOSE:-}" ] || skip "campaign-8 close gate — run with C8_CLOSE=1 at close"; }

# ===========================================================================
# META close invariants (SC5, SC6, SC7, SC10-SC13 close aspects)
# ===========================================================================

@test "CLOSE-sweeps (SC6): sweep-fold-audit --strict exit 0 AND sweep-build-state exit 0" {
  _close
  run "$KIT/toolbelt/sweep-fold-audit.sh" --strict "$KIT/retros/INDEX.md" "$KIT"
  [ "$status" -eq 0 ]
  run bash "$KIT/toolbelt/sweep-build-state.sh" "$KIT/BUILD-STATE.md" "$KIT/retros" "$KIT/retros/INDEX.md"
  [ "$status" -eq 0 ]
}

@test "CLOSE-index-pending (SC6): retros/INDEX.md has ZERO '| pending |' rows after the fold PR" {
  _close
  [ "$(grep -c '| pending |' "$KIT/retros/INDEX.md")" -eq 0 ]
}

@test "CLOSE-version (SC7): VERSION == 0.19.0" {
  _close
  [ "$(cat "$REPO/VERSION" | tr -d '[:space:]')" = "0.19.0" ]
}

@test "CLOSE-changelog (SC7): CHANGELOG.md has a [v0.19.0] section" {
  _close
  grep -qE '^\#\# \[v?0\.19\.0\]' "$REPO/CHANGELOG.md"
}

@test "CLOSE-no-trailers (SC7): no attribution trailer across the campaign range 48fb210..HEAD" {
  _close
  [ "$(git -C "$REPO" log ${BASE}..HEAD --format=%B | grep -ciE 'co-authored|generated with|claude-session|noreply@anthropic')" -eq 0 ]
}

@test "CLOSE-kit-links (SC5): kit-links.bats all green (L1-L8, routing + orchestration + post-deploy)" {
  _close
  run bats "$REPO/tests/kit-links.bats"
  [ "$status" -eq 0 ]
}

@test "CLOSE-install-skill (SC5): the installed SKILL.md matches the tracked source (install-skill --dry-run exit 0)" {
  _close
  [ -f "$HOME/.claude/skills/build-n4-module/SKILL.md" ] || skip "no installed skill copy on this machine"
  run bash "$REPO/scripts/install-skill.sh" --dry-run
  [ "$status" -eq 0 ]
}

@test "CLOSE-shellcheck (SC7): shellcheck exit 0 over scripts + toolbelt + tests helpers" {
  _close
  command -v shellcheck >/dev/null 2>&1 || skip "shellcheck not installed"
  run shellcheck "$REPO"/scripts/*.sh "$KIT"/toolbelt/*.sh
  [ "$status" -eq 0 ]
}

@test "CLOSE-tag (SC7): tag v0.19.0 exists and points at the final campaign commit" {
  _close
  git -C "$REPO" rev-parse -q --verify 'refs/tags/v0.19.0' >/dev/null
  # the tag must sit on main's tip (the close commit)
  [ "$(git -C "$REPO" rev-parse 'v0.19.0^{commit}')" = "$(git -C "$REPO" rev-parse HEAD)" ]
}

# ===========================================================================
# Per-tool SC pins — each proven GREEN by its campaign bats file on main.
# (SC1/SC2/SC3/SC16 also carry a REAL smoke, env-guarded below.)
# ===========================================================================

@test "CLOSE-tool-pins (SC1,3,4,8,9,11-16): every campaign pin file passes on main" {
  _close
  # The FINAL pin set (all present once wave 3 — PR18 lint-structure, PR19 lint-write-path,
  # PR20 station-logic checks CHECK13-19 — has merged). PR20's SL13-19 + SL-smoke pins live in
  # bog-audit.bats (station-logic.bats was the standalone RED file, removed once folded). An absent
  # pin at close is itself a failure.
  for f in lint-delays triage-console lint-timers report-module schema-risk facets-lint \
           slot-coverage rc-scan station-snapshot bog-audit lint-wb-threading verify-module \
           lint-servlet retro-loop lint-structure lint-write-path; do
    [ -f "$REPO/tests/$f.bats" ] || { echo "ABSENT pin (campaign not complete): $f.bats"; false; }
    run bats "$REPO/tests/$f.bats"
    [ "$status" -eq 0 ] || { echo "FAILED pin: $f.bats"; false; }
  done
}

# ---- Real local smokes (env-guarded; SKIP is honest "could not run", never a silent pass) ----

@test "SC1-smoke: lint-delays is CLEAN on the blessed tree (main-ff1b659 ColdRoomPan-rt: exit 0, 0 FAIL rows; defrost time<=0 bug fixed post-C9)" {
  _close
  R="$C8_CLIENT_REPO"   # via client-root.bash default=main-ff1b659; env override wins [ev: design.md D3c site 8]
  CRP="$R/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"
  [ -d "$CRP" ] || skip "client ColdRoomPan tree not on this machine (set C8_CLIENT_REPO)"
  run "$KIT/toolbelt/lint-delays.sh" "$CRP"
  # R-T2.10: pin the blessed-tree verdict, not "either tree". The delay-floor RULE stays
  # pinned by the synthetic LD1/LD3/LD6 (non-positive floor -> FAIL); this smoke asserts the
  # real blessed tree is clean, so it can no longer rot green against a stale pre-fix checkout.
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL')" -eq 0 ]
}

@test "SC16-smoke: bog-audit CHECK13-19 exact rows on PANCCADIA + CHECK13-19 clean on MX60" {
  _close
  RB="${C8_PANCCADIA_BOG:-}"; [ -n "$RB" ] && [ -f "$RB" ] || skip "set C8_PANCCADIA_BOG"
  T="$BATS_TEST_TMPDIR"; case "$RB" in *.xml) ( cd "$(dirname "$RB")" && cp "$(basename "$RB")" "$T/f.xml" ) && ( cd "$T" && zip -q p.bog f.xml ); RB="$T/p.bog";; *) cp "$RB" "$T/p.bog"; RB="$T/p.bog";; esac
  run "$KIT/toolbelt/bog-audit.sh" "$RB" --module ColdRoomPan --module CompPan --module DashboardPan
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK11  FAIL')" -eq 17 ]
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK14  WARN')" -eq 1 ]
  [[ "$output" == *"CHECK14  WARN"*"EvaporatorUnit2"*"evapOut"* ]]
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK18  FAIL')" -eq 2 ]
  [[ "$output" == *"CHECK18  FAIL"*"EvaporatorUnit_1"* ]] && [[ "$output" == *"CHECK18  FAIL"*"EvaporatorUnit_3"* ]]
  for c in CHECK13 CHECK15 CHECK16 CHECK17 CHECK19; do [ "$(printf '%s\n' "$output" | grep -c "$c  FAIL")" -eq 0 ]; done
  MB="${C8_MX60_BOG:-}"; [ -n "$MB" ] && [ -f "$MB" ] || skip "PANCCADIA rows green; set C8_MX60_BOG for the MX60 half"
  case "$MB" in *.xml) ( cd "$(dirname "$MB")" && cp "$(basename "$MB")" "$T/m.xml" ) && ( cd "$T" && zip -q m.bog m.xml ); MB="$T/m.bog";; *) cp "$MB" "$T/m.bog"; MB="$T/m.bog";; esac
  run "$KIT/toolbelt/bog-audit.sh" "$MB" --module chihuahua
  for c in CHECK13 CHECK14 CHECK15 CHECK16 CHECK17 CHECK18 CHECK19; do
    [ "$(printf '%s\n' "$output" | grep -c "$c  FAIL")" -eq 0 ]; [ "$(printf '%s\n' "$output" | grep -c "$c  WARN")" -eq 0 ]
  done
}
