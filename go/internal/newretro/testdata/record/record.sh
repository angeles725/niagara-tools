#!/usr/bin/env bash
# record.sh <new-retro.sh> <sha> <out-dir> [case-glob] — records the go/internal/newretro golden cases from the
# bash new-retro.sh as of <sha> (the bash left the toolbelt in the Go switch, so extract it first:
#   git show <sha>:build-n4-module-kit/toolbelt/new-retro.sh > /some/dir/new-retro.sh
# The goldens were recorded at 15e792d1). Cases are in cases.sh beside this file. Each case runs in its own
# dir D, the script copied to D/toolbelt/new-retro.sh beside a symlink to this checkout's gen-retro-index.sh
# (so the script's own default kit root is D, the NEW_RETRO_DEFAULT_ROOT the Go test passes), under
# `env -i PATH=D/fakebin:/usr/bin:/bin` plus the case's ENVV; D/fakebin/date prints 2026-01-02, the date the
# Go test injects. The case file holds args, want (exit + stdout), stderr, @kind (the source and sha), @env,
# one @expect section per file the run left (the whole tree after the run, toolbelt/ and fakebin/ excluded),
# then the tree as it was before the run. A truth-* case is hand-written and says so in its @kind; this
# script never writes one.
set -u
NR="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"; SHA="$2"; OUT="$3"; FILTER="${4:-}"
GEN="$(cd "$(dirname "$(readlink -f "$0")")/../../../../.." && pwd)/build-n4-module-kit/toolbelt/gen-retro-index.sh"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/nrrec.XXXXXX")"
mkdir -p "$OUT"
bash "$GEN" --definitely-not-an-option >/dev/null 2>&1   # builds go/bin/gen-retro-index once, before env -i

new() {
  D="$(mktemp -d "$WORK/c.XXXXXX")"
  mkdir -p "$D/toolbelt" "$D/fakebin"
  cp "$NR" "$D/toolbelt/new-retro.sh"; ln -s "$GEN" "$D/toolbelt/gen-retro-index.sh"
  printf '#!/bin/sh\nprintf "2026-01-02\\n"\n' > "$D/fakebin/date"; chmod +x "$D/fakebin/date"
  ENVV=()
}
# kit <dir>: a kit root with the bats fixture (an INDEX header and the minimal `## kit` envelope)
kit() {
  mkdir -p "$D/$1/retros"
  printf '| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n' > "$D/$1/retros/INDEX.md"
  printf '<!-- build-state.v1 -->\n## kit\nretro_required: true\nretro_pending: false\n' > "$D/$1/BUILD-STATE.src.md"
}
put() { mkdir -p "$(dirname "$D/$1")"; printf '%b' "$2" > "$D/$1"; }   # put <path> <printf %b text>

sub() { local s="${1//"$D"/@DIR@}"; printf '%s' "$s"; }
# serialize <prefix>: every file and empty dir under D (toolbelt/, fakebin/ excluded), as golden sections
serialize() {
  local p rel body
  ( cd "$D" && LC_ALL=C find . -mindepth 1 \( -path ./toolbelt -o -path ./fakebin \) -prune -o -print | LC_ALL=C sort ) |
  while IFS= read -r p; do
    rel="${p#./}"; p="$D/$rel"
    if [ -d "$p" ]; then
      [ -n "$1" ] || [ -n "$(ls -A "$p")" ] || printf -- '-- %s/ --\n' "$rel"
      continue
    fi
    body="$(cat "$p"; printf x)"; body="${body%x}"; body="${body//"$D"/@DIR@}"
    if [ -n "$body" ] && [ "${body: -1}" != $'\n' ]; then printf -- '-- %s%s (noeol) --\n%s\n' "$1" "$rel" "$body"
    else printf -- '-- %s%s --\n%s' "$1" "$rel" "$body"; fi
  done
}
rec() {
  local name="$1"; shift
  # shellcheck disable=SC2053  # FILTER is a glob on purpose
  [ -n "$FILTER" ] && [[ "$name" != $FILTER ]] && return
  local tree; tree="$(serialize ""; printf x)"; tree="${tree%x}"
  local rc=0
  ( cd "$D" && env -i PATH="$D/fakebin:/usr/bin:/bin" "${ENVV[@]}" bash "$D/toolbelt/new-retro.sh" "$@" \
      > "$WORK/out" 2> "$WORK/err" ) || rc=$?
  {
    printf -- '-- args --\n'; local a; for a in "$@"; do printf '%s\n' "$(sub "$a")"; done
    printf -- '-- want --\nexit %s\n' "$rc"; sed "s#$D#@DIR@#g" "$WORK/out"
    printf -- '-- stderr --\n'; sed "s#$D#@DIR@#g" "$WORK/err"
    printf -- '-- @kind --\nrecorded from build-n4-module-kit/toolbelt/new-retro.sh at %s\n' "$SHA"
    printf -- '-- @env --\n'; for a in "${ENVV[@]}"; do printf '%s\n' "$(sub "$a")"; done
    serialize "@expect "
    printf '%s' "$tree"
  } > "$OUT/$name.txt"
}

# shellcheck disable=SC1091  # cases.sh beside this script, resolved at runtime
. "$(dirname "$0")/cases.sh"
rm -rf "$WORK"
