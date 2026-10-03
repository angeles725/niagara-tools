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
