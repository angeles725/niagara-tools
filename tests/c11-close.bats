#!/usr/bin/env bats
# Campaign-11 CLOSE gate — SKELETON (mirror of tests/c10-close.bats). Env-guarded on
# C11_CLOSE so it is inert in the normal suite; the close worker fills the TODOs when the
# C11 scope is frozen.
#
# C11 scope (parser unification + kit self-guards, kit-only so far):
#   T1 — one shared section-D method-boundary parser (PEAK/max_d depth + get/set/is
#        accessor skip) across lint-timers / lint-silent-protection / lint-ext-writable;
#        golden set tests/golden-parser.bats + tests/parser-oneliner.bats.
#   T2 — single-source the blessed client read tree via tests/lib/client-root.bash
#        (one default = main-ff1b659); pin tests/client-root-single-source.bats.
#   T3 — lint-write-path DRIFT advisory (a [concept] row whose slot IS covered);
#        pin tests/write-path-drift.bats.
#   T4 — NEW tool toolbelt/lint-guard-pins.sh (a lint header naming a mutation with no
#        matching bats fixture -> WARN) + the # Mutation: retrofit across the lints;
#        pin tests/guard-pins.bats.
# NEW tool FILE this campaign (lint-guard-pins.sh) -> the tool-pins loop gains guard-pins
# plus the four new C11 pin bats. VERSION MINOR 0.22.0 (new tool + new DRIFT advisory,
# not a kit-only PATCH). SC-13 client versions carry over (C11 is kit-only -> no bumps).
#
# Run post-hoc:  C11_CLOSE=1 C11_CLOSE_COMMIT=<close sha> C9_CLIENT_REPO=<post-C11 client main worktree> \
#                bats tests/c11-close.bats

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  BASE="dab0807"                       # C11 base = the C10 close commit (tag v0.21.0); no-trailer sweep range base
  VERSION_TARGET="${C11_VERSION:-0.22.0}"   # frozen: MINOR 0.22.0 — new tool lint-guard-pins + DRIFT advisory
  TAG="v${VERSION_TARGET}"
}
_close() { [ -n "${C11_CLOSE:-}" ] || skip "C11 close gate — set C11_CLOSE=1 to run"; }

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

@test "CLOSE-version: VERSION == the C11 target" {
  _close
  [ "$(cat "$REPO/VERSION")" = "$VERSION_TARGET" ]
}

@test "CLOSE-changelog: CHANGELOG.md has a [v<target>] section naming the four C11 work units (T1-T4)" {
  _close
  sec="$(awk -v t="## [v${VERSION_TARGET}]" 'index($0,t)==1{f=1;next} /^## \[/{f=0} f' "$REPO/CHANGELOG.md")"
  [ -n "$sec" ] || { echo "no [v${VERSION_TARGET}] section"; false; }
  # Frozen: the four C11 units, each named in the [v0.22.0] section (greps below).
  printf '%s\n' "$sec" | grep -qiE 'shared parser|PEAK|one-liner|accessor'    || { echo "CHANGELOG missing the T1 shared-parser entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'client-root|single-source|client tree'     || { echo "CHANGELOG missing the T2 client-root entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'concept-drift|DRIFT'                        || { echo "CHANGELOG missing the T3 concept-drift entry"; false; }
  printf '%s\n' "$sec" | grep -qiE 'lint-guard-pins|guard-pins|# Mutation'      || { echo "CHANGELOG missing the T4 lint-guard-pins entry"; false; }
}

@test "CLOSE-no-trailers: no attribution trailer across the campaign range (BASE..HEAD)" {
  _close
  [ "$(git -C "$REPO" log --format=%B "${BASE}..HEAD" | grep -ciE 'co-authored|generated with|claude-session|noreply@anthropic')" -eq 0 ]
}

@test "CLOSE-no-conflict-markers: no leftover git conflict markers in tracked .md/.sh/.bats/.bash" {
  _close
  # A rebase fragment-merge (C11 PR2) can leave <<<<<<< / >>>>>>> markers; real ones were
  # found committed in a C8 archive doc. Line-anchored, marker + space, .git excluded.
  offenders=$(grep -rlE '^(<<<<<<< |>>>>>>> )' --include='*.md' --include='*.sh' --include='*.bats' --include='*.bash' --exclude-dir='.git' "$REPO" || true)
  n=$(printf '%s' "$offenders" | grep -c . || true)
  [ "$n" -eq 0 ] || { echo "conflict markers in $n file(s):"; printf '%s\n' "$offenders"; false; }
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
  [ "$(git -C "$REPO" rev-parse "${TAG}^{commit}")" = "$(git -C "$REPO" rev-parse "${C11_CLOSE_COMMIT:-HEAD}")" ]
}

@test "CLOSE-tool-pins: every campaign pin file passes on main (C10 set + the NEW C11 tool lint-guard-pins and the four C11 pin bats)" {
  _close
  # C11 adds a NEW toolbelt/*.sh: lint-guard-pins.sh -> its pin (guard-pins) joins the loop,
  # alongside the T1/T2/T3 pin bats (golden-parser, parser-oneliner, client-root-single-source,
  # write-path-drift). Frozen: the C10 set (20) + the new tool's pin guard-pins + the four
  # C11 pin bats = the full C11 pin set on main.
  for f in lint-delays triage-console lint-timers report-module schema-risk facets-lint \
           slot-coverage rc-scan station-snapshot bog-audit lint-wb-threading verify-module \
           lint-servlet retro-loop lint-structure lint-write-path \
           ext-writable-shape lint-silent-protection demand-in-scope run-pure-test \
           guard-pins golden-parser parser-oneliner client-root-single-source write-path-drift; do
    [ -f "$REPO/tests/$f.bats" ] || { echo "ABSENT pin: $f.bats"; false; }
    run bats "$REPO/tests/$f.bats"
    [ "$status" -eq 0 ] || { echo "FAILED pin: $f.bats"; false; }
  done
}

@test "SC-13 (client group defaultModuleVersion): C11 carry-over 2.2.0/2.1.0/2.2.0 (kit-only campaign — no client bump)" {
  _close
  R="${C9_CLIENT_REPO:-/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659}"
  [ -d "$R" ] || skip "client repo not on this machine (set C9_CLIENT_REPO)"
  v() { grep -ohE 'defaultModuleVersion\("[0-9.]+"\)' "$1" 2>/dev/null | head -1 | grep -oE '[0-9.]+'; }
  # C11 is kit-only (no client module bumped a version) -> versions carry over from C10 (verified).
  [ "$(v "$R/Compresores/build.gradle.kts")" = "2.2.0" ]
  [ "$(v "$R/Paccadia/build.gradle.kts")"    = "2.1.0" ]
  [ "$(v "$R/Dashboard/build.gradle.kts")"   = "2.2.0" ]
}

@test "CLOSE-harness-run (CARRIED OVER from C9/C10 — still pending): qa/c9-harness-run.md records three 'Failures: 0, Skips: 0' runs" {
  _close
  f="$REPO/qa/c9-harness-run.md"
  [ -f "$f" ] || { echo "no harness run record — the C9 Windows niagaraTest session (CRA1/2/3-live, CPB5, R14 lockout+AuditEvent) is still owed; qa/c9-harness-procedure.md + the run-sheet"; false; }
  [ "$(grep -cE '^Total tests run: [1-9][0-9]*, Failures: 0, Skips: 0' "$f")" -eq 3 ]
  [ "$(grep -cE 'Skips: [1-9]' "$f")" -eq 0 ]
}
