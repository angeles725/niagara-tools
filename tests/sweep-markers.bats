#!/usr/bin/env bats
# C12 S3 RED — toolbelt/sweep-markers.sh, a reusable git conflict-marker sweep. C11 added
# CLOSE-no-conflict-markers to c11-close.bats + a per-PR verify grep after real leftover
# markers were found committed in a C8 archive doc (latent since campaign 8). That is a
# bats-level point check; C12 promotes it to a routed toolbelt lint (BUILD-LOOP + SKILL),
# a sibling to lint-guard-pins.sh. Flags a start `<<<<<<< ` / end `>>>>>>> ` line (marker +
# space — no markdown uses that) and a bare `=======` middle only in a file that also carries
# a start marker (so a markdown setext underline is never a false positive). Rows MARKER;
# exit 0 clean / 1 any marker / 3 usage; .git excluded.
#
# RED on 66123a2 (v0.22.0): toolbelt/sweep-markers.sh does not exist (command-not-found).
# GREEN once C12 adds the tool. [ev: C12 S3; probe 850791f12; PR2 C8-archive incident]

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  SM="$REPO/build-n4-module-kit/toolbelt/sweep-markers.sh"
}

@test "S3-usage: no root argument -> exit 3" {
  run "$SM"
  [ "$status" -eq 3 ]
}

@test "S3-clean: a tree with no markers -> exit 0, no MARKER rows" {
  d="$BATS_TEST_TMPDIR/clean"; mkdir -p "$d"; printf 'ok\n' > "$d/a.txt"
  run "$SM" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" != *"MARKER"* ]]
}

@test "S3-full-conflict: a file with <<<<<<< / ======= / >>>>>>> -> exit 1, start+middle+end rows" {
  d="$BATS_TEST_TMPDIR/conf"; mkdir -p "$d"
  printf 'a\n<<<<<<< HEAD\nx\n=======\ny\n>>>>>>> branch\nb\n' > "$d/c.txt"
  run "$SM" "$d"
  [ "$status" -eq 1 ]
  [[ "$output" == *"MARKER"* && "$output" == *"start"* && "$output" == *"middle"* && "$output" == *"end"* ]]
}

@test "S3-start-only: a stray start marker alone -> exit 1" {
  d="$BATS_TEST_TMPDIR/st"; mkdir -p "$d"; printf 'x\n<<<<<<< HEAD\ny\n' > "$d/s.txt"
  run "$SM" "$d"
  [ "$status" -eq 1 ]
  [[ "$output" == *"start"* ]]
}

@test "S3-markdown-safe: a markdown setext ======= underline with NO conflict markers is NOT flagged -> exit 0" {
  d="$BATS_TEST_TMPDIR/md"; mkdir -p "$d"
  printf '# Title\n=======\nbody text\n' > "$d/doc.md"
  run "$SM" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" != *"MARKER"* ]]
}

@test "S3-smoke: the real repo is marker-free -> exit 0 (the C8-archive incident is resolved)" {
  run "$SM" "$REPO"
  [ "$status" -eq 0 ]
  [[ "$output" != *"MARKER"* ]]
}
