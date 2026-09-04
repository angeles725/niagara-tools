#!/usr/bin/env bats
# RED-FIRST enforcement test for the continuity + retro gate (campaign PR2).
# Frozen contract (Mejoras, 2026-09-04), split in two because kit-links.bats L2 forbids
# a toolbelt/*.sh from invoking git:
#
#   1) build-n4-module-kit/toolbelt/sweep-build-state.sh  — CONTENT ONLY, NO git.
#        usage: sweep-build-state.sh <BUILD-STATE.md> <retros-dir> <INDEX.md>
#        exit 0 clean · 1 named integrity violation · 3 usage/env
#        checks: each build-state.v1 envelope well-formed (markers balanced + required
#          fields module/retro_required/retro_pending); retro_pending is boolean;
#          retro_required:true ⇒ retro_pending present; INDEX integrity (every retro file
#          has a row, every row points to a real retro, review-status ∈ {pending,folded}).
#
#   2) .githooks/pre-push  — git ALLOWED (a hook is not toolbelt/*.sh).
#        invoked: pre-push <remote> <url>, ref lines on stdin "<lref> <lsha> <rref> <rsha>".
#        build-relevant IN this repo = build-n4-module-kit/** EXCLUDING BUILD-STATE.md,
#          retros/, retros/INDEX.md; PLUS scripts/**.
#        if a build-relevant path changed in range AND NOT (BUILD-STATE.md + a pending
#          retro + its INDEX row all in range) AND NOT a `Retro: none (trivial: …)` trailer
#          in range ⇒ FAIL non-zero. Otherwise 0. Calls sweep for the content half.
#        Must NOT fire on a module-repo path or a non-build-relevant path (no false fail).
#
# These bites are real: flip the guarded condition and the outcome flips. No real push,
# no real build — git is a fakebin (ng-deploy.bats pattern), fixtures are tiny (sub-second).

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$REPO/build-n4-module-kit"
  SWEEP="$KIT/toolbelt/sweep-build-state.sh"
  HOOK="$REPO/.githooks/pre-push"

  # ---- content fixtures (for sweep) ----
  FIX="$BATS_TEST_TMPDIR/kit"
  mkdir -p "$FIX/retros"
  STATE="$FIX/BUILD-STATE.md"
  RETRODIR="$FIX/retros"
  INDEX="$FIX/retros/INDEX.md"

  # ---- fakebin git (for the hook) ----
  FAKEBIN="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$FAKEBIN"
  cat > "$FAKEBIN/git" <<'GIT'
#!/usr/bin/env bash
# Scripted git for hook tests: reads FAKE_TOPLEVEL / FAKE_CHANGED_FILE / FAKE_LOG_FILE.
case "$1" in
  rev-parse) echo "${FAKE_TOPLEVEL:-$PWD}" ;;
  diff)      [ -f "$FAKE_CHANGED_FILE" ] && cat "$FAKE_CHANGED_FILE" ;;
  log)       [ -f "$FAKE_LOG_FILE" ] && cat "$FAKE_LOG_FILE" ;;
  *)         exit 0 ;;
esac
GIT
  chmod +x "$FAKEBIN/git"
}

# --- content helpers ---------------------------------------------------------
write_state() {  # write_state <retro_required> <retro_pending|OMIT>
  { echo "# BUILD-STATE"; echo
    echo "<!-- build-state.v1 -->"
    echo "schema: build-state.v1"
    echo "module: DemoPan"
    echo "module_repo: Cliente/Demo"
    echo "module_root: Demo/DemoPan"
    echo "retro_required: $1"
    [ "$2" != OMIT ] && echo "retro_pending: $2"
    echo "last_commit: abc1234"
    echo "last_session: 2026-09-04"
    echo "<!-- /build-state.v1 -->"
  } > "$STATE"
}
write_retro() { # write_retro <basename> <marker-line>
  printf '%s\n\n# demo retro\n' "$2" > "$RETRODIR/$1"
}
write_index() { # write_index <rows...> each "file|status"
  { echo "| Retro file | review-status |"; echo "|---|---|"
    for r in "$@"; do echo "| ${r%%|*} | ${r##*|} |"; done
  } > "$INDEX"
}

# --- hook helper -------------------------------------------------------------
run_hook() { # run_hook  (uses FAKE_CHANGED_FILE / FAKE_LOG_FILE already set)
  FAKE_TOPLEVEL="$REPO" \
  run env PATH="$FAKEBIN:$PATH" bash "$HOOK" origin https://example/repo.git <<<'refs/heads/main aaaa refs/heads/main bbbb'
}

# ============================ CONTENT (sweep) =================================

@test "S1: a clean state + matching retro + INDEX row exits 0" {
  write_state true false
  write_retro "2026-09-04-demo.md" "<!-- review-status: pending -->"
  write_index "2026-09-04-demo.md|pending"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 0 ]
}

@test "S2: a malformed envelope (unbalanced marker) exits 1" {
  { echo "<!-- build-state.v1 -->"; echo "module: DemoPan"; } > "$STATE"   # no closing marker
  write_index
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "S3: a retro file with NO INDEX row exits 1 (INDEX integrity)" {
  write_state true false
  write_retro "2026-09-04-demo.md" "<!-- review-status: pending -->"
  write_index    # empty index — the retro has no row
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "S4: retro_required:true with retro_pending MISSING exits 1" {
  write_state true OMIT
  write_index
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "S5: an INDEX row pointing at a NON-existent retro exits 1" {
  write_state false false
  write_index "2026-09-04-ghost.md|pending"   # no such file in retros/
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "S6: a bad review-status (not pending/folded) exits 1" {
  write_state true false
  write_retro "2026-09-04-demo.md" "<!-- review-status: banana -->"
  write_index "2026-09-04-demo.md|banana"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "S7: missing argument exits 3 (usage/env)" {
  run "$SWEEP" "$STATE"
  [ "$status" -eq 3 ]
}

@test "S8: a prose-embedded marker (indented/backticked) is NOT counted as an envelope" {
  # The real BUILD-STATE.md 'How to read' prose quotes the OPEN and CLOSE markers on SEPARATE
  # indented+backticked lines (not one line). Reproduce that exact shape: a col-0-anchored sweep
  # ignores both prose lines and validates only the real envelope → exit 0. A parser that matched
  # the marker as a SUBSTRING would treat the prose open line as an envelope start and the prose
  # close line as its end → no module → fail. Verified: this fixture stays green on the anchored
  # sweep and goes RED under a substring-match mutation (so S8 bites the anchor independently of H3).
  { echo "## How to read"
    echo "- one \`<!-- build-state.v1 -->\`"
    echo "  \`<!-- /build-state.v1 -->\` per module"
    echo
    echo "<!-- build-state.v1 -->"
    echo "module: DemoPan"
    echo "retro_required: false"
    echo "retro_pending: false"
    echo "<!-- /build-state.v1 -->"
  } > "$STATE"
  write_index
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 0 ]
}

@test "S9: a multi-line open_issues list is tolerated (matches the real file shape) exits 0" {
  { echo "<!-- build-state.v1 -->"
    echo "module: DemoPan"
    echo "retro_required: true"
    echo "retro_pending: false"
    echo "open_issues:"
    echo "  - DefrostController.java has zero pure tests (HIGH)."
    echo "  - suction-2 sensor stuck at 130.5342 psi."
    echo "last_commit: abc1234"
    echo "<!-- /build-state.v1 -->"
  } > "$STATE"
  write_retro "2026-09-04-demo.md" "<!-- review-status: pending -->"
  write_index "2026-09-04-demo.md|pending"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 0 ]
}

# ============================ CLASSIFICATION (hook) ==========================

@test "H1: a KIT-file change with NO state/retro update and NO trivial trailer FAILS" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/types/logic.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'feat: change a control rule\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -ne 0 ]
}

@test "H2: the SAME kit change WITH a 'Retro: none (trivial: …)' trailer PASSES" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/types/logic.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'chore: fix a typo\n\nRetro: none (trivial: comment-only wording fix)\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H3: the SAME kit change WITH BUILD-STATE.md + a pending retro + INDEX row PASSES" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/types/logic.md\nbuild-n4-module-kit/BUILD-STATE.md\nbuild-n4-module-kit/retros/2026-09-04-demo.md\nbuild-n4-module-kit/retros/INDEX.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'feat: change a control rule + retro\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H4: a MODULE-REPO path (build-n4-module-kit not involved) does NOT fire the gate" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  # A module lives in another repo; over-reaching to gate it would be a false fail.
  printf 'Compresores/CompPan/CompPan-rt/src/com/angeles/CompPan/CompressorControl.java\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'fix: compressor staging\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H5: a NON-build-relevant change (top-level README) does NOT fire the gate" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'README.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'docs: readme tweak\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H6: an EXCLUDED kit path (retros/ only) does NOT fire the gate (a retro edit isn't build-relevant)" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/retros/2026-09-04-demo.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'retro: notes\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H7: a scripts/ change with no state/retro/trailer FAILS (scripts/ is build-relevant)" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'scripts/ng-deploy.sh\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'feat: ng-deploy flag\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -ne 0 ]
}
