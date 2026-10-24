#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
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
#          retros/, retros/INDEX.md; PLUS scripts/** and go/** (except go/bin/).
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

@test "BSF-pending: an envelope with no retro_pending line names that missing field (exit 1, empty stdout)" {
  # Mutation: BSF-pending -- dropping the retro_pending presence check falls through to the boolean check
  # ("retro_pending not boolean: ''"): S4 still exits 1, this exact row flips.
  write_state true OMIT
  write_index
  run --separate-stderr "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
  [ "$output" = "" ]
  # shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
  [ "$stderr" = "sweep: envelope (DemoPan) missing required field: retro_pending" ]
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

# --- third §7 exit: promotion (a fold-only PR fits neither "new retro" nor "trivial") ---
# A promotion trailer is a valid close ONLY when it moves the registry with a STRUCTURAL ANCHOR in
# range — either a retros/INDEX.md diff (a FULL promotion flips folded/pending marks) OR a
# BUILD-STATE.md diff (a PARTIAL promotion folds content + narrows the owed open_issue list without
# flipping a row, because the source retro stays pending). The trailer with NEITHER anchor must NOT
# be a blanket escape. Both anchored paths still run sweep for ledger coherence.
@test "H8: a kit change + 'Retro: promotion (folds …)' trailer AND an INDEX.md diff PASSES" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/types/logic.md\nbuild-n4-module-kit/retros/INDEX.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'docs: fold logic lessons\n\nRetro: promotion (folds L3,L4 from existing retros)\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H9: a promotion trailer with NEITHER an INDEX diff NOR a BUILD-STATE diff FAILS (no blanket escape)" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/types/logic.md\n' > "$BATS_TEST_TMPDIR/changed"   # kit file only: no anchor
  printf 'docs: fold logic lessons\n\nRetro: promotion (folds L3 from existing retros)\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -ne 0 ]
}

# H10 drives the AG-PR1 fix: a PARTIAL promotion (folds content, source retro stays pending, so it
# flips NO INDEX row) anchors on an in-range BUILD-STATE.md diff instead — the shape C4-PR2 shipped.
# RED today: the current hook (.githooks/pre-push line 43) requires an INDEX diff (has_index>=1), so
# a BUILD-STATE-only promotion falls through to the FAIL. The fix accepts a BUILD-STATE diff as an
# ALTERNATIVE structural anchor and still runs sweep for ledger coherence.
# MUTATION (run post-green): drop the BUILD-STATE alt-anchor acceptance in the hook → H10 flips back
# to FAIL, proving the acceptance clause (not sweep) is what carries it.
@test "H10: a promotion trailer + an in-range BUILD-STATE.md diff (no INDEX) PASSES (partial promotion)" {
  [ -f "$HOOK" ] || skip "pre-push hook not implemented yet (red-first)"
  printf 'build-n4-module-kit/types/logic.md\nbuild-n4-module-kit/BUILD-STATE.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'docs: fold B737 composition\n\nRetro: promotion (folds B737 into logic.md; source retro stays pending for the owed halves)\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

# --- generated INDEX (r8 T8i): retros/INDEX.md is regenerated from the retro headers by
# toolbelt/gen-retro-index.sh, so the new-retro exit no longer needs a hand-edited INDEX.md diff in
# range: BUILD-STATE.md + a retro is enough WHEN the index is fresh, and sweep (the content half)
# fails a stale one. RED before T8i: the hook required has_index>=1, so H11 fell through to FAIL.
# _fixture_top — a fake toplevel with the real sweep + generator and a one-retro kit.
_fixture_top() {
  TOP="$BATS_TEST_TMPDIR/top"; local k="$TOP/build-n4-module-kit"
  mkdir -p "$k/toolbelt" "$k/retros"
  ln -s "$(cd "$KIT/.." && pwd)/go" "$TOP/go" # the wrapper execs <toplevel>/go/bin [ev: retro go-module-gen-retro-index Δ2]
  cp "$KIT/toolbelt/sweep-build-state.sh" "$KIT/toolbelt/gen-retro-index.sh" "$k/toolbelt/"
  printf '<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 1 -->\n# demo\n' > "$k/retros/2026-09-04-demo.md"
  printf '<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\nretro_pending: true\n<!-- /build-state.v1 -->\n' > "$k/BUILD-STATE.md"
  bash "$k/toolbelt/gen-retro-index.sh" "$k" >/dev/null
  printf 'build-n4-module-kit/types/logic.md\nbuild-n4-module-kit/BUILD-STATE.md\nbuild-n4-module-kit/retros/2026-09-04-demo.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'feat: change a control rule + retro\n' > "$BATS_TEST_TMPDIR/log"
}
run_hook_at() { # run_hook_at <toplevel>
  FAKE_TOPLEVEL="$1" FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" \
  run env PATH="$FAKEBIN:$PATH" bash "$HOOK" origin https://example/repo.git <<<'refs/heads/main aaaa refs/heads/main bbbb'
}

@test "H11: BUILD-STATE.md + a retro with NO INDEX.md diff PASSES when the generated index is fresh" {
  _fixture_top
  run_hook_at "$TOP"
  [ "$status" -eq 0 ]
}

@test "H12: the same range FAILS when the generated index is stale (retro header changed, not regenerated)" {
  _fixture_top
  sed -i 's/deltas: 1/deltas: 2/' "$TOP/build-n4-module-kit/retros/2026-09-04-demo.md"
  run_hook_at "$TOP"
  [ "$status" -ne 0 ]
  [[ "$output" == *"stale"* ]]
}

# --- ledger fragments (r8 T8i slice B): the kit block's last_change, open_issues and session log are
# GENERATED from build-n4-module-kit/ledger.d/*.md by toolbelt/gen-build-state.sh. A new or changed
# fragment in range satisfies the "BUILD-STATE.md in range" half of every exit, and sweep (the content
# half) fails a BUILD-STATE.md that is stale against ledger.d/. RED before slice B: the hook counted
# only a BUILD-STATE.md diff, and ledger.d/ was build-relevant.
# _fixture_ledger_top — _fixture_top plus ledger.d/, the generator and a kit block generated from it.
_fixture_ledger_top() {
  _fixture_top; local k="$TOP/build-n4-module-kit"
  cp "$KIT/toolbelt/gen-build-state.sh" "$k/toolbelt/"
  mkdir -p "$k/ledger.d"
  # BUILD-STATE.md is generated from BUILD-STATE.src.md + ledger.d/ (R10-D1); the source keeps the
  # kit envelope's generated parts empty.
  printf '<!-- build-state.v1 -->\nmodule: kit\nlast_change: generated               # from ledger.d/; run toolbelt/gen-build-state.sh\nopen_issues:\nretro_required: true\nretro_pending: true\nlast_session: below\n<!-- /build-state.v1 -->\n' > "$k/BUILD-STATE.src.md"
  printf '<!-- ledger: kit -->\n<!-- date: 2026-09-04 -->\nlog: 2026-09-04 · archived line\n' > "$k/ledger.d/0000-archive-pre-2026-10-23.md"
  printf '<!-- ledger: kit -->\n<!-- date: 2026-09-05 -->\nlog: 2026-09-05 · demo change\nopen_issue: OWED demo half\n' > "$k/ledger.d/2026-09-05-demo.md"
  bash "$k/toolbelt/gen-build-state.sh" "$k" >/dev/null
  printf 'build-n4-module-kit/types/logic.md\nbuild-n4-module-kit/ledger.d/2026-09-05-demo.md\nbuild-n4-module-kit/retros/2026-09-04-demo.md\n' > "$BATS_TEST_TMPDIR/changed"
}

@test "H13: a kit change + a ledger.d fragment + a retro (no BUILD-STATE.md diff) PASSES when BUILD-STATE.md is fresh" {
  _fixture_ledger_top
  run_hook_at "$TOP"
  [ "$status" -eq 0 ]
}

@test "H14: the same range FAILS when BUILD-STATE.md is stale against ledger.d (fragment added, not regenerated)" {
  # Mutation: GBS-gate -- a hook that counts a fragment without sweep's freshness check passes this.
  _fixture_ledger_top
  printf '<!-- ledger: kit -->\n<!-- date: 2026-09-06 -->\nlog: 2026-09-06 · not regenerated\n' > "$TOP/build-n4-module-kit/ledger.d/2026-09-06-late.md"
  run_hook_at "$TOP"
  [ "$status" -ne 0 ]
  [[ "$output" == *"stale vs ledger.d"* ]]
}

@test "H15: a promotion trailer + an in-range ledger.d fragment (no INDEX, no BUILD-STATE diff) PASSES" {
  _fixture_ledger_top
  printf 'build-n4-module-kit/types/logic.md\nbuild-n4-module-kit/ledger.d/2026-09-05-demo.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'docs: fold demo\n\nRetro: promotion (folds demo Δ1; source retro stays pending)\n' > "$BATS_TEST_TMPDIR/log"
  run_hook_at "$TOP"
  [ "$status" -eq 0 ]
}

@test "H16: a ledger.d-only range is not build-relevant (does NOT fire the gate)" {
  printf 'build-n4-module-kit/ledger.d/2026-09-05-demo.md\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'docs: ledger entry\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

@test "H17: a go/ source change with no state/retro/trailer FAILS (go/** is build-relevant)" {
  # Mutation: H17 -- dropping go/ from the build-relevant set lets a Go port change lint behavior ungated.
  # [ev: retro go-javasrc-scanner Δ1]
  printf 'go/internal/javasrc/javasrc.go\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'feat: port a lint to Go\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -ne 0 ]
}

@test "H18: a go/bin/-only range is not build-relevant (built binaries, not source)" {
  # Mutation: H18 -- matching go/ without the go/bin/ exclusion fires the gate on a rebuilt binary.
  printf 'go/bin/n4lint\n' > "$BATS_TEST_TMPDIR/changed"
  printf 'chore: rebuild\n' > "$BATS_TEST_TMPDIR/log"
  FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" run_hook
  [ "$status" -eq 0 ]
}

# ============ MARKER<->INDEX CONSISTENCY (campaign 6, Option 2-lite) ==========
# Frozen contract (investigador, 2026-09-05): when a retro file carries a
# `<!-- review-status: X -->` marker, X MUST equal that retro's INDEX row status;
# a MISSING marker is tolerated (the legacy folded retros stay untouched), and X
# must be a valid domain word {pending,folded}.
#
# RED-FIRST: sweep-build-state.sh today is MARKER-BLIND — Section 2 reads
# review-status only from the INDEX row (awk over row cells, line ~76) and never
# opens the retro file except for existence (line ~78). So M1 and M4 FAIL now
# (sweep returns 0 where 2-lite wants 1); they turn GREEN when the impl PR
# (campaign task 1) makes the sweep read + reconcile the in-file marker.
# MUTATION (run post-green): revert the marker-read clause in the sweep -> M1
# (disagree) flips back to exit 0, proving the marker-read clause — not the
# pre-existing INDEX-row check — is what carries the new bite.

@test "M1: marker disagrees with INDEX row (marker=folded, row=pending) FAILS (2-lite)" {
  write_state true false
  write_retro "2026-09-04-demo.md" "<!-- review-status: folded -->"
  write_index "2026-09-04-demo.md|pending"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "M2: marker AGREES with INDEX row (both pending) stays clean (exit 0)" {
  write_state true false
  write_retro "2026-09-04-demo.md" "<!-- review-status: pending -->"
  write_index "2026-09-04-demo.md|pending"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 0 ]
}

@test "M3: a MISSING marker is tolerated (INDEX row folded, no in-file marker) exit 0" {
  # 2-lite MUST NOT over-reach: the ~25 legacy folded retros carry no marker and stay clean.
  write_state true false
  write_retro "2026-09-04-demo.md" ""                          # body carries no review-status marker
  run ! grep -q 'review-status' "$RETRODIR/2026-09-04-demo.md"     # fixture guard: really no marker
  write_index "2026-09-04-demo.md|folded"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 0 ]
}

@test "M4: an out-of-domain marker in the REAL shape ('fresh . DATE') FAILS (2-lite)" {
  # Mirrors the actual drift on the tree: retros/2026-09-02-comppan-fase1-staging.md and
  # 2026-09-02-dashboardpan-detail-render-doors.md carry `review-status: fresh . 2026-09-02`
  # while INDEX says folded. The trailing ` . DATE` suffix forces the impl to parse the FIRST
  # word after the colon, not exact-match the whole marker line — otherwise the real markers
  # (all suffixed) would never be seen and the drift would go undetected.
  write_state true false
  write_retro "2026-09-02-demo.md" "<!-- review-status: fresh . 2026-09-02 -->"
  write_index "2026-09-02-demo.md|folded"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 1 ]
}

@test "M5: the sweep on the REAL kit tree exits 0 (forces the 2 'fresh' files re-stamped in the impl PR)" {
  # GREEN today (marker-blind). When the impl reads markers this goes RED until the two real
  # 'fresh' retros are reconciled to 'folded' (INDEX is right — they were promoted in campaign 2),
  # forcing that reconciliation into the same PR and guarding future strays (ci.yml runs sweep on
  # the real tree).
  run "$SWEEP" "$KIT/BUILD-STATE.md" "$KIT/retros" "$KIT/retros/INDEX.md"
  [ "$status" -eq 0 ]
}

@test "M6: a review-status mention only in prose PAST line 5 is NOT a marker (INDEX folded) exit 0" {
  # design D1: the marker is column-0 anchored within the FIRST 5 LINES. A retro whose only
  # `review-status:` occurrence is a prose/table mention further down (real shape: the kit-continuity
  # retro cites it in a table cell ~line 40) must be treated as NO marker -> 2-lite tolerates it -> 0.
  # This bites a naive `grep -m1 review-status:` impl: it would grab the prose word 'pending', see it
  # != the INDEX 'folded' row, and wrongly FAIL. Green under the first-5-lines column-0 parse.
  write_state true false
  { echo "# Retro: something"; echo; echo "## Context"; echo; echo "Body line."; echo
    echo "Earlier the review-status: pending marker was discussed here in prose."; } \
    > "$RETRODIR/2026-09-05-prose.md"
  write_index "2026-09-05-prose.md|folded"
  run "$SWEEP" "$STATE" "$RETRODIR" "$INDEX"
  [ "$status" -eq 0 ]
}

# --- first push of a new branch (remote sha all zeros), against a REAL scratch repo -------------------
# The range must be <merge-base with origin/main>..<local sha>. Diffing the lone local sha compared it with
# the WORKING TREE (empty after a clean commit), so every new branch passed unchecked.
newbranch_repo() { # newbranch_repo <dir> — main + origin/main at commit A; branch feat adds a kit toolbelt file
  local d="$1"
  git init -q -b main "$d"
  git -C "$d" -c user.name=t -c user.email=t@t commit -q --allow-empty -m base
  git -C "$d" update-ref refs/remotes/origin/main HEAD
  git -C "$d" checkout -q -b feat
  mkdir -p "$d/build-n4-module-kit/toolbelt"
  echo 'echo hi' > "$d/build-n4-module-kit/toolbelt/new.sh"
  git -C "$d" add -A
  git -C "$d" -c user.name=t -c user.email=t@t commit -q -m "feat: new toolbelt script"
}

@test "H19: the first push of a new branch gates the whole branch since main (no BUILD-STATE/retro -> FAIL)" {
  # Mutation: H19-newbranch -- diffing the lone local sha (vs the working tree) passes every new branch unchecked
  local d="$BATS_TEST_TMPDIR/nb"; newbranch_repo "$d"
  local lsha; lsha=$(git -C "$d" rev-parse HEAD)
  run bash -c "cd '$d' && bash '$HOOK' origin x <<<'refs/heads/feat $lsha refs/heads/feat 0000000000000000000000000000000000000000'"
  [ "$status" -ne 0 ]
  [[ "$output" == *"build-relevant kit files changed"* ]]
}

@test "H20: a new branch whose base cannot be found fails closed (never an unchecked pass)" {
  local d="$BATS_TEST_TMPDIR/nb"; newbranch_repo "$d"
  git -C "$d" update-ref -d refs/remotes/origin/main
  local lsha; lsha=$(git -C "$d" rev-parse HEAD)
  run bash -c "cd '$d' && bash '$HOOK' origin x <<<'refs/heads/feat $lsha refs/heads/feat 0000000000000000000000000000000000000000'"
  [ "$status" -ne 0 ]
  [[ "$output" == *"cannot find"* ]]
}

@test "H21: deleting a remote branch (local sha all zeros) is not gated" {
  local d="$BATS_TEST_TMPDIR/nb"; newbranch_repo "$d"
  run bash -c "cd '$d' && bash '$HOOK' origin x <<<'(delete) 0000000000000000000000000000000000000000 refs/heads/feat $(git -C "$d" rev-parse HEAD)'"
  [ "$status" -eq 0 ]
}

# --- a failing git and the origin/HEAD merge-base fallback (#355 R3-git-failure-refusal-untested) --------
@test "H22: a failing git diff refuses the push (never an unchecked pass)" {
  # Mutation: H22-diff-refuse -- dropping '|| refuse' leaves changed empty, so nothing is build-relevant and the push passes
  local d="$BATS_TEST_TMPDIR/nb"; newbranch_repo "$d"
  local lsha; lsha=$(git -C "$d" rev-parse HEAD)
  # the remote sha names an object this repo does not have, so 'git diff <rsha>..<lsha>' fails
  run bash -c "cd '$d' && bash '$HOOK' origin x <<<'refs/heads/feat $lsha refs/heads/feat 1111111111111111111111111111111111111111'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"git diff 1111111111111111111111111111111111111111..$lsha failed; refusing to push unchecked"* ]]
}

@test "H23: a failing git log refuses the push (never read as 'no waiver trailer')" {
  # Mutation: H23-log-refuse -- dropping '|| refuse' falls through to the generic no-BUILD-STATE FAIL, losing the cause
  local d="$BATS_TEST_TMPDIR/nb"; newbranch_repo "$d"
  local lsha; lsha=$(git -C "$d" rev-parse HEAD)
  local real_git; real_git=$(command -v git)
  mkdir -p "$BATS_TEST_TMPDIR/bin"
  # a git that fails only 'log'; every other subcommand is the real git
  # shellcheck disable=SC2016  # $1 and $@ belong to the generated wrapper, not to this test
  printf '#!/bin/sh\n[ "$1" = log ] && exit 128\nexec %s "$@"\n' "$real_git" > "$BATS_TEST_TMPDIR/bin/git"
  chmod +x "$BATS_TEST_TMPDIR/bin/git"
  run bash -c "cd '$d' && PATH='$BATS_TEST_TMPDIR/bin':\"\$PATH\" bash '$HOOK' origin x <<<'refs/heads/feat $lsha refs/heads/feat 0000000000000000000000000000000000000000'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"git log "*"..$lsha failed; refusing to push unchecked"* ]]
}

@test "H24: without origin/main the new-branch base falls back to origin/HEAD" {
  # Mutation: H24-origin-head -- dropping the origin/HEAD fallback refuses with 'cannot find' instead of gating the range
  local d="$BATS_TEST_TMPDIR/nb"; newbranch_repo "$d"
  git -C "$d" update-ref refs/remotes/origin/trunk refs/remotes/origin/main
  git -C "$d" update-ref -d refs/remotes/origin/main
  git -C "$d" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/trunk
  local lsha; lsha=$(git -C "$d" rev-parse HEAD)
  run bash -c "cd '$d' && bash '$HOOK' origin x <<<'refs/heads/feat $lsha refs/heads/feat 0000000000000000000000000000000000000000'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"build-relevant kit files changed"* ]]
  [[ "$output" != *"cannot find"* ]]
}
