#!/usr/bin/env bats
# C11 T2 RED — single-source the blessed client read-tree default. Every bats that
# needs the client tree must read it from tests/lib/client-root.bash (one definition,
# RP1), never hardcode an absolute .../Cliente/Leon-Guanjuato... path anywhere in its
# own body — whether as a C9_CLIENT_ROOT/REPO or C8_CLIENT_REPO parameter default, or
# as an override-less absolute path. At dab0807 the literal is copy-pasted across 10
# bats, so a single retarget (as in C10 PR7) has to touch every one instead of one
# file — the drift that made the a109249->ff1b659 fix a multi-commit chase.
#
# The pin keys on the PATH literal, not a variable name, so it catches every shape:
#   5 C9_CLIENT_ROOT   (ext-writable-shape:26, demand-in-scope:27, lint-timers:418,
#                       lint-silent-protection:30, lint-write-path:338)
#   3 C9/C8_CLIENT_REPO (c9-close:108, c10-close:90, c8-close)
#   2 override-less     (lint-delays:53, rc-scan:75)
# T2 lib exports C9_CLIENT_ROOT, C9_CLIENT_REPO and C8_CLIENT_REPO (same default);
# env override still wins. [ev: companero/investigador1 C11 T2; C10 PR7 retarget]

setup() {
  TDIR="$BATS_TEST_DIRNAME"
  SELF="client-root-single-source.bats"
  # Offender = any absolute Leon-Guanjuato client path literal in a bats, in any file
  # but this one (the lib lives at tests/lib/*.bash and is not matched by tests/*.bats).
  PAT='/Cliente/Leon-Guanjuato'
}

@test "C11-T2-lib-exists: tests/lib/client-root.bash is the single definition of the blessed client tree" {
  [ -f "$TDIR/lib/client-root.bash" ]
}

@test "C11-T2-no-hardcode: no bats hardcodes an absolute client-tree path outside tests/lib/client-root.bash" {
  offenders=$(grep -lF "$PAT" "$TDIR"/*.bats 2>/dev/null | grep -v -e "/$SELF" || true)
  n=$(printf '%s' "$offenders" | grep -c . || true)
  [ "$n" -eq 0 ] || { echo "hardcoded client-tree path in $n bats file(s):"; printf '%s\n' "$offenders"; false; }
}
