#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# The client-identifier step of .githooks/pre-push: before every other check the hook hands its
# arguments and the WHOLE pre-push stdin to niagara-go's client-guard and refuses on its non-zero
# exit. Without client-guard on PATH it refuses when a client denylist exists, else warns and goes on.
# Hermetic: client-guard and git are fakebins; the denylist is a fake path in the test tmpdir.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  HOOK="$REPO/.githooks/pre-push"
  FAKEBIN="$BATS_TEST_TMPDIR/bin"; mkdir -p "$FAKEBIN"
  cat > "$FAKEBIN/git" <<'GIT'
#!/usr/bin/env bash
case "$1" in rev-parse) echo "$BATS_TEST_TMPDIR" ;; *) exit 0 ;; esac
GIT
  GUARD_LOG="$BATS_TEST_TMPDIR/guard"; export GUARD_LOG
  cat > "$BATS_TEST_TMPDIR/client-guard" <<'GUARD'
#!/usr/bin/env bash
printf 'args: %s\n' "$*" > "$GUARD_LOG"; cat >> "$GUARD_LOG"
exit "${GUARD_RC:-0}"
GUARD
  chmod +x "$FAKEBIN/git" "$BATS_TEST_TMPDIR/client-guard"
  DENYLIST="$BATS_TEST_TMPDIR/denylist.txt"
  # two deletes: once past the guard step, the rest of the hook exits 0 at the delete check
  STDIN=$'refs/heads/a 0000000000000000000000000000000000000000 refs/heads/a aaaa\nrefs/heads/b 0000000000000000000000000000000000000000 refs/heads/b bbbb'
}

hook() { # hook: run the hook with only FAKEBIN (plus the shell basics) on PATH
  run env PATH="$FAKEBIN:/usr/bin:/bin" NIAGARA_CLIENT_DENYLIST="$DENYLIST" \
    bash "$HOOK" origin https://example/repo.git <<<"$STDIN"
}

@test "CG1: client-guard gets the hook arguments and every stdin line; exit 0 lets the push go on" {
  cp "$BATS_TEST_TMPDIR/client-guard" "$FAKEBIN/"
  GUARD_RC=0 hook
  [ "$status" -eq 0 ]
  run cat "$GUARD_LOG"
  [[ "$output" == *"args: origin https://example/repo.git"* ]]
  [[ "$output" == *"refs/heads/a 0000"* ]]
  [[ "$output" == *"refs/heads/b 0000"* ]]
}

@test "CG2: a client-guard refusal (exit 1) refuses the push" {
  cp "$BATS_TEST_TMPDIR/client-guard" "$FAKEBIN/"
  GUARD_RC=1 hook
  [ "$status" -ne 0 ]
}

@test "CG3: a broken guard (exit 2) refuses the push" {
  cp "$BATS_TEST_TMPDIR/client-guard" "$FAKEBIN/"
  GUARD_RC=2 hook
  [ "$status" -ne 0 ]
}

@test "CG4: no client-guard on PATH and a denylist present refuses with an install hint" {
  : > "$DENYLIST"
  hook
  [ "$status" -eq 1 ]
  [[ "$output" == *"client-guard is not on PATH"* ]]
  [[ "$output" == *"go install ./cmd/client-guard"* ]]
}

@test "CG5: no client-guard and no denylist warns once and lets the push go on" {
  hook
  [ "$status" -eq 0 ]
  [[ "$output" == *"client-identifier check skipped"* ]]
}
