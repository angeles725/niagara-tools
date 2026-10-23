#!/usr/bin/env bats
# scan-files.bats — toolbelt/lib/scan-files.sh, the shared fail-closed file lister (deferred-lints D1/D1b).
# [ev: issue #226 R3-find-error-still-fail-open]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  # shellcheck disable=SC1091
  . "$KIT/toolbelt/lib/scan-files.sh"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SF1: a dot-named root is walked, only dot SUB-directories are pruned (D1b, R4-dot-root-prune)" {
  # Mutation: SF1 -- applying the dot-dir prune at depth 0 prunes the whole tree and lists nothing.
  mkdir -p "$TMPDIR_T/.mod/src/.git" "$TMPDIR_T/.mod/src/com"
  printf 'x\n' > "$TMPDIR_T/.mod/src/com/A.java"
  printf 'x\n' > "$TMPDIR_T/.mod/src/.git/B.java"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/.mod" -name '*.java'
  [ "$(cat "$TMPDIR_T/files")" = "$TMPDIR_T/.mod/src/com/A.java" ]
}

@test "SF2: '.' as the root is walked (D1b)" {
  mkdir -p "$TMPDIR_T/m/com"
  printf 'x\n' > "$TMPDIR_T/m/com/A.java"
  cd "$TMPDIR_T/m"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" . -name '*.java'
  [ "$(cat "$TMPDIR_T/files")" = "./com/A.java" ]
}

@test "SF3: a sort failure returns non-zero with its reason in the err file (D1b, R2/R3 sort-err)" {
  mkdir -p "$TMPDIR_T/m" "$TMPDIR_T/out-is-a-dir"
  printf 'x\n' > "$TMPDIR_T/m/A.java"
  run scan_files "$TMPDIR_T/out-is-a-dir" "$TMPDIR_T/err" "$TMPDIR_T/m" -name '*.java'
  [ "$status" -ne 0 ]
  [ -s "$TMPDIR_T/err" ]
}

@test "SF4: a root that is not a directory returns non-zero with a reason, never an empty clean list (D1c)" {
  # Mutation: SF4 -- dropping the directory check lists nothing for a file root and returns 0.
  printf 'x\n' > "$TMPDIR_T/A.java"
  run scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/A.java" -name '*.java'
  [ "$status" -ne 0 ]
  grep -q 'not a directory' "$TMPDIR_T/err"
}

@test "SF5: --prune <name> prunes every sub-directory of that name, at any depth (audit A0)" {
  # Mutation: SF5 -- ignoring the --prune option lists the files under build/ again.
  mkdir -p "$TMPDIR_T/m/lib" "$TMPDIR_T/m/build/libs" "$TMPDIR_T/m/sub/build"
  printf 'x\n' > "$TMPDIR_T/m/lib/a.jar"
  printf 'x\n' > "$TMPDIR_T/m/build/libs/b.jar"
  printf 'x\n' > "$TMPDIR_T/m/sub/build/c.jar"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/m" --prune build -name '*.jar'
  [ "$(cat "$TMPDIR_T/files")" = "$TMPDIR_T/m/lib/a.jar" ]
}

@test "SF6: an unreadable sub-directory under a --prune walk still returns non-zero (audit A0)" {
  mkdir -p "$TMPDIR_T/m/locked" "$TMPDIR_T/m/build"
  printf 'x\n' > "$TMPDIR_T/m/locked/a.jar"
  chmod 000 "$TMPDIR_T/m/locked"
  if [ -r "$TMPDIR_T/m/locked" ]; then chmod 755 "$TMPDIR_T/m/locked"; skip "running as root"; fi
  run scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/m" --prune build -name '*.jar'
  chmod 755 "$TMPDIR_T/m/locked"
  [ "$status" -ne 0 ]
  grep -q 'locked' "$TMPDIR_T/err"
}

@test "SF7: scan_files_or_exit lists a complete walk, and exits with the caller's prefix and code on an incomplete one" {
  # Mutation: SF7 -- returning instead of exiting lets the caller scan a short list and pass clean.
  mkdir -p "$TMPDIR_T/m/com" "$TMPDIR_T/m/locked"
  printf 'x\n' > "$TMPDIR_T/m/com/A.java"
  run scan_files_or_exit demo 3 "$TMPDIR_T/files" "$TMPDIR_T/m/com" -name '*.java'
  [ "$status" -eq 0 ]
  [ "$(cat "$TMPDIR_T/files")" = "$TMPDIR_T/m/com/A.java" ]
  chmod 000 "$TMPDIR_T/m/locked"
  if [ -r "$TMPDIR_T/m/locked" ]; then chmod 755 "$TMPDIR_T/m/locked"; skip "running as root"; fi
  run scan_files_or_exit demo 4 "$TMPDIR_T/files" "$TMPDIR_T/m" -name '*.java'
  chmod 755 "$TMPDIR_T/m/locked"
  [ "$status" -eq 4 ]
  [[ "$output" == "demo: cannot list every file under $TMPDIR_T/m: "*locked* ]]
}

@test "SF8: --prune-path <pattern> prunes every sub-directory whose path matches, at any depth (rc-scan rc/ext)" {
  # Mutation: SF8 -- ignoring --prune-path lists the vendored files under rc/ext again.
  mkdir -p "$TMPDIR_T/m/rc/ext" "$TMPDIR_T/m/a/rc/ext/x" "$TMPDIR_T/m/ext"
  printf 'x\n' > "$TMPDIR_T/m/rc/a.js"
  printf 'x\n' > "$TMPDIR_T/m/rc/ext/b.js"
  printf 'x\n' > "$TMPDIR_T/m/a/rc/ext/x/c.js"
  printf 'x\n' > "$TMPDIR_T/m/ext/d.js"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/m" --prune-path '*/rc/ext' -name '*.js'
  [ "$(cat "$TMPDIR_T/files")" = "$(printf '%s\n' "$TMPDIR_T/m/ext/d.js" "$TMPDIR_T/m/rc/a.js")" ]
}

@test "SF9: sibling *-worktrees checkouts are pruned at any depth, so a finding is not reported once per checkout" {
  # Mutation: SF9 -- dropping the '*-worktrees' prune lists the copy under chihuahua-worktrees/ again.
  mkdir -p "$TMPDIR_T/m/chihuahua/src" "$TMPDIR_T/m/chihuahua-worktrees/wt/src" "$TMPDIR_T/m/a/x-worktrees"
  printf 'x\n' > "$TMPDIR_T/m/chihuahua/src/A.java"
  printf 'x\n' > "$TMPDIR_T/m/chihuahua-worktrees/wt/src/A.java"
  printf 'x\n' > "$TMPDIR_T/m/a/x-worktrees/B.java"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/m" -name '*.java'
  [ "$(cat "$TMPDIR_T/files")" = "$TMPDIR_T/m/chihuahua/src/A.java" ]
}

@test "SF10: a symlinked root is followed like find -H, paths keep the root as given (issue #329)" {
  # Mutation: SF10 -- dropping -H from the find in scan_files lists nothing under a symlinked root and returns 0.
  mkdir -p "$TMPDIR_T/m/com"
  printf 'x\n' > "$TMPDIR_T/m/com/A.java"
  ln -s "$TMPDIR_T/m" "$TMPDIR_T/link"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/link" -name '*.java'
  [ "$(cat "$TMPDIR_T/files")" = "$TMPDIR_T/link/com/A.java" ]
}

@test "SF11: below a symlinked root, symlinks are still neither followed nor listed (issue #329)" {
  # Mutation: SF11 -- -L instead of -H follows the inner links and lists other/B.java and L.java.
  mkdir -p "$TMPDIR_T/m/com" "$TMPDIR_T/other"
  printf 'x\n' > "$TMPDIR_T/m/com/A.java"
  printf 'x\n' > "$TMPDIR_T/other/B.java"
  ln -s "$TMPDIR_T/other" "$TMPDIR_T/m/inner"
  ln -s "$TMPDIR_T/m/com/A.java" "$TMPDIR_T/m/L.java"
  ln -s "$TMPDIR_T/m" "$TMPDIR_T/link"
  scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/link" -name '*.java'
  [ "$(cat "$TMPDIR_T/files")" = "$TMPDIR_T/link/com/A.java" ]
}

@test "SF12: a dangling root symlink or a root symlink to a file is not a directory, never a clean pass (issue #329)" {
  ln -s "$TMPDIR_T/gone" "$TMPDIR_T/dangling"
  printf 'x\n' > "$TMPDIR_T/A.java"
  ln -s "$TMPDIR_T/A.java" "$TMPDIR_T/filelink"
  for r in dangling filelink; do
    run scan_files "$TMPDIR_T/files" "$TMPDIR_T/err" "$TMPDIR_T/$r" -name '*.java'
    [ "$status" -eq 2 ]
    grep -q "not a directory: $TMPDIR_T/$r" "$TMPDIR_T/err"
  done
}
