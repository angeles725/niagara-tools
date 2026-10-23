#!/usr/bin/env bash
# symroot.bash — shared bats helper: a tool run on a symlink to a directory must equal the run on the
# directory itself (issue #329: a plain `find` lists nothing under a symlinked root and the tool passed clean).
# Loaded with `load helpers/symroot` from the bats suite of each toolbelt script that walks its root argument.

# assert_symroot_same <dir> <command> [args...] — run <command> [args...] once as given, then again with every
# occurrence of <dir> in the arguments replaced by a symlink to <dir> (same basename, under $BATS_TEST_TMPDIR).
# Fails unless both runs exit with the same status and the symlink run prints the plain run's output with
# <dir> spelled as the symlink. Leaves $status / $output of the symlink run for further assertions.
# shellcheck disable=SC2154  # status and output are set by bats `run`
assert_symroot_same() {
  local dir="$1" link arg plain_status plain_output
  shift
  link="$BATS_TEST_TMPDIR/symroot/$(basename "$dir")"
  mkdir -p "${link%/*}"
  ln -sfn "$dir" "$link"
  local -a linked=()
  for arg in "$@"; do linked+=("${arg//"$dir"/"$link"}"); done
  run "$@"
  plain_status="$status"
  plain_output="${output//"$dir"/"$link"}"
  run "${linked[@]}"
  if [ "$status" -ne "$plain_status" ] || [ "$output" != "$plain_output" ]; then
    printf 'symlinked root %s: status %s (plain %s)\n--- plain (root respelled)\n%s\n--- symlinked\n%s\n' \
      "$link" "$status" "$plain_status" "$plain_output" "$output" >&2
    return 1
  fi
}
