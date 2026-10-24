#!/usr/bin/env bats
# Tests for scripts/kit-resolve.sh — fail-closed resolution of a current, clean build-n4-module kit.
# [ev: retro r10-stale-kit-resolution Δ1] — a default checkout 821 commits behind origin/main ran a client
# build. Hermetic: every repo is a scratch clone of a scratch bare origin under $TMPDIR_T; HOME points
# there too, so the real checkouts and the real kit-main worktree are never read or touched.

bats_require_minimum_version 1.5.0

setup() {
  TMPDIR_T="$(mktemp -d)"
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KR="$REPO/scripts/kit-resolve.sh"
  H="$TMPDIR_T/home"
  ORIGIN="$TMPDIR_T/origin.git"
  KM="$H/modulos_niagara_n4/niagara-tools-worktrees/kit-main"
  DEF="$H/modulos_niagara_n4/niagara-tools"
  mkdir -p "$H"
  export GIT_CONFIG_NOSYSTEM=1 GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t
  unset BUILD_N4_KIT BUILD_N4_KIT_MAIN
  git init -q --bare -b main "$ORIGIN"
  seed="$TMPDIR_T/seed"
  git init -q -b main "$seed"
  mkdir -p "$seed/build-n4-module-kit"
  printf 'methodology\n' > "$seed/build-n4-module-kit/METHODOLOGY.md"
  git -C "$seed" add -A && git -C "$seed" commit -q -m c1
  git -C "$seed" remote add origin "$ORIGIN" && git -C "$seed" push -q origin main
}
teardown() { rm -rf "$TMPDIR_T"; }

_clone() { git clone -q "$ORIGIN" "$1"; }                      # a clean checkout at origin/main
_advance() {                                                   # origin/main gains one commit
  git -C "$TMPDIR_T/seed" commit -q --allow-empty -m "c-$RANDOM"
  git -C "$TMPDIR_T/seed" push -q origin main
}
_resolve() { HOME="$H" run "$KR" "$@"; }

@test "KR1: \$BUILD_N4_KIT current and clean -> first line KIT=<abs> SHA=<sha> (origin/main, clean), exit 0" {
  _clone "$TMPDIR_T/dev"
  _clone "$KM"                                               # a kit-main also exists: $BUILD_N4_KIT wins
  BUILD_N4_KIT="$TMPDIR_T/dev/build-n4-module-kit" HOME="$H" run "$KR"
  [ "$status" -eq 0 ]
  sha=$(git -C "$TMPDIR_T/dev" rev-parse --short HEAD)
  [ "${lines[0]}" = "KIT=$TMPDIR_T/dev/build-n4-module-kit SHA=$sha (origin/main, clean)" ]
}

@test "KR2: kit-main current -> OK line naming kit-main, preferred over the default checkout" {
  _clone "$KM"
  _clone "$DEF"
  _resolve
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "KIT=$KM/build-n4-module-kit SHA="*" (origin/main, clean)" ]]
}

@test "KR3: clean kit-main strictly behind origin/main -> auto fast-forward, refreshed <old>→<new> on stderr, exit 0" {
  # changed by r10-kit-resolve-autoff: behind + clean kit-main used to be exit 2 REFUSED stale
  _clone "$KM"
  old=$(git -C "$KM" rev-parse --short=7 HEAD)
  _advance
  new=$(git -C "$ORIGIN" rev-parse --short=7 main)
  HOME="$H" run --separate-stderr "$KR"
  [ "$status" -eq 0 ]
  # shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
  [[ "$stderr" == *"kit-resolve: refreshed $old→$new"* ]]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$(git -C "$ORIGIN" rev-parse main)" ]
  [[ "${lines[0]}" == "KIT=$KM/build-n4-module-kit SHA="*" (origin/main, clean)" ]]
}

@test "KR4: dirty kit tree -> exit 2 REFUSED dirty, even when current" {
  _clone "$KM"
  printf 'edit\n' >> "$KM/build-n4-module-kit/METHODOLOGY.md"
  _resolve
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit-resolve: REFUSED dirty"* ]]
  [[ "$output" != *"KIT="* ]]
}

@test "KR5: no candidate holds METHODOLOGY.md -> exit 3, names every path tried" {
  _resolve
  [ "$status" -eq 3 ]
  [[ "$output" == *"kit-resolve: no kit found"* ]]
  [[ "$output" == *"$KM/build-n4-module-kit"* ]]
  [[ "$output" == *"$DEF/build-n4-module-kit"* ]]
}

@test "KR6: \$BUILD_N4_KIT set to a dir without METHODOLOGY.md -> exit 3, never falls through to kit-main" {
  _clone "$KM"
  mkdir -p "$TMPDIR_T/empty"
  BUILD_N4_KIT="$TMPDIR_T/empty" HOME="$H" run "$KR"
  [ "$status" -eq 3 ]
  [[ "$output" == *"BUILD_N4_KIT"*"no METHODOLOGY.md"* ]]
  [[ "$output" != *"KIT="* ]]
}

@test "KR7: origin unreachable -> named WARN offline and exit 2 refused; --offline-ok continues with the WARN" {
  _clone "$KM"
  git -C "$KM" remote set-url origin "$TMPDIR_T/missing.git"
  _resolve
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit-resolve: WARN offline: could not fetch origin; freshness unchecked"* ]]
  [[ "$output" == *"--offline-ok"* ]]
  [[ "$output" != *"KIT="* ]]
  HOME="$H" run --separate-stderr "$KR" --offline-ok
  [ "$status" -eq 0 ]
  # shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
  [[ "$stderr" == *"kit-resolve: WARN offline: could not fetch origin; freshness unchecked"* ]]
  [[ "${lines[0]}" == "KIT=$KM/build-n4-module-kit SHA="*" (origin/main unfetched, clean)" ]]
}

@test "KR8: --refresh on a clean kit-main behind origin/main -> refreshed to origin/main, then OK" {
  _clone "$KM"
  _advance
  _resolve --refresh
  [ "$status" -eq 0 ]
  want=$(git -C "$ORIGIN" rev-parse --short main)
  [ "$(git -C "$KM" rev-parse --short HEAD)" = "$want" ]
  [[ "$output" == *"KIT=$KM/build-n4-module-kit SHA=$want (origin/main, clean)"* ]]
}

@test "KR9: --refresh never touches the default checkout (HEAD unchanged, still refused)" {
  _clone "$DEF"
  _advance
  before=$(git -C "$DEF" rev-parse HEAD)
  _resolve --refresh
  [ "$status" -eq 2 ]
  [ "$(git -C "$DEF" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"--refresh applies only to the kit-main worktree"* ]]
  [[ "$output" == *"kit-resolve: REFUSED stale"* ]]
}

@test "KR10: --refresh never touches \$BUILD_N4_KIT (HEAD unchanged, still refused)" {
  _clone "$TMPDIR_T/dev"
  _advance
  before=$(git -C "$TMPDIR_T/dev" rev-parse HEAD)
  BUILD_N4_KIT="$TMPDIR_T/dev/build-n4-module-kit" HOME="$H" run "$KR" --refresh
  [ "$status" -eq 2 ]
  [ "$(git -C "$TMPDIR_T/dev" rev-parse HEAD)" = "$before" ]
}

@test "KR11: --refresh on a dirty kit-main refuses and does not check out" {
  _clone "$KM"
  _advance
  before=$(git -C "$KM" rev-parse HEAD)
  printf 'edit\n' >> "$KM/build-n4-module-kit/METHODOLOGY.md"
  _resolve --refresh
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED dirty"* ]]
}

@test "KR12: \$BUILD_N4_KIT_MAIN overrides the kit-main location" {
  _clone "$TMPDIR_T/elsewhere"
  BUILD_N4_KIT_MAIN="$TMPDIR_T/elsewhere" HOME="$H" run "$KR"
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "KIT=$TMPDIR_T/elsewhere/build-n4-module-kit SHA="* ]]
}

@test "KR13: unknown flag -> usage, exit 3" {
  _resolve --bogus
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: kit-resolve.sh"* ]]
}

@test "KR14: dirty kit-main behind origin/main -> exit 2 REFUSED dirty, HEAD unchanged (no auto-refresh)" {
  _clone "$KM"
  _advance
  before=$(git -C "$KM" rev-parse HEAD)
  printf 'edit\n' >> "$KM/build-n4-module-kit/METHODOLOGY.md"
  _resolve
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED dirty"* ]]
  [[ "$output" != *"refreshed"* ]]
}

@test "KR15: diverged kit-main (local commit not on origin/main) -> exit 2 REFUSED diverged, HEAD unchanged" {
  _clone "$KM"
  git -C "$KM" checkout -q --detach
  git -C "$KM" commit -q --allow-empty -m local
  _advance
  before=$(git -C "$KM" rev-parse HEAD)
  _resolve
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED diverged"* ]]
  [[ "$output" != *"KIT="* ]]
}

@test "KR16: offline kit-main behind its local origin/main -> exit 2, HEAD unchanged, also with --offline-ok" {
  _clone "$KM"
  _advance
  git -C "$KM" fetch -q origin                                 # local origin/main is ahead of HEAD
  git -C "$KM" remote set-url origin "$TMPDIR_T/missing.git"
  before=$(git -C "$KM" rev-parse HEAD)
  _resolve
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: WARN offline"* ]]
  _resolve --offline-ok
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED stale"* ]]
  [[ "$output" != *"refreshed"* ]]
  _resolve --offline-ok --refresh                              # an explicit --refresh never refreshes unfetched
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" != *"refreshed"* ]]
}

@test "KR17: stale \$BUILD_N4_KIT without flags -> exit 2 REFUSED stale, HEAD untouched" {
  _clone "$TMPDIR_T/dev"
  _advance
  before=$(git -C "$TMPDIR_T/dev" rev-parse HEAD)
  BUILD_N4_KIT="$TMPDIR_T/dev/build-n4-module-kit" HOME="$H" run "$KR"
  [ "$status" -eq 2 ]
  [ "$(git -C "$TMPDIR_T/dev" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED stale"* ]]
  [[ "$output" != *"refreshed"* ]]
}

@test "KR18: stale default checkout without flags -> exit 2 REFUSED stale naming the behind count, HEAD untouched" {
  _clone "$DEF"
  _advance
  before=$(git -C "$DEF" rev-parse HEAD)
  _resolve
  [ "$status" -eq 2 ]
  [ "$(git -C "$DEF" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED stale"*"1 commit(s) behind origin/main"* ]]
  [[ "$output" != *"refreshed"* ]]
}

@test "KR19: kit-main ahead of origin/main (one local commit on top) -> exit 2 REFUSED ahead, HEAD unchanged" {
  _clone "$KM"
  git -C "$KM" checkout -q --detach
  git -C "$KM" commit -q --allow-empty -m local
  before=$(git -C "$KM" rev-parse HEAD)
  _resolve
  [ "$status" -eq 2 ]
  [ "$(git -C "$KM" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED ahead"* ]]
  [[ "$output" != *"KIT="* ]]
}

@test "KR20: \$BUILD_N4_KIT or default checkout ahead of origin/main -> exit 2 REFUSED ahead, HEAD unchanged" {
  _clone "$TMPDIR_T/dev"
  git -C "$TMPDIR_T/dev" commit -q --allow-empty -m local
  before=$(git -C "$TMPDIR_T/dev" rev-parse HEAD)
  BUILD_N4_KIT="$TMPDIR_T/dev/build-n4-module-kit" HOME="$H" run "$KR"
  [ "$status" -eq 2 ]
  [ "$(git -C "$TMPDIR_T/dev" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED ahead"* ]]
  _clone "$DEF"
  git -C "$DEF" commit -q --allow-empty -m local
  before=$(git -C "$DEF" rev-parse HEAD)
  _resolve
  [ "$status" -eq 2 ]
  [ "$(git -C "$DEF" rev-parse HEAD)" = "$before" ]
  [[ "$output" == *"kit-resolve: REFUSED ahead"* ]]
}
