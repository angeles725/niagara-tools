#!/usr/bin/env bash
# record.sh <checkout> <sha> <out-dir> [case-glob] — records the go/internal/wiringmap golden cases from the bash
# toolbelt/generate-wiring-map.sh of <checkout> (a tree holding build-n4-module-kit/toolbelt and
# tests/fixtures/generate-wiring-map* as of <sha>; the Go switch removed the bash body, so extract it first:
#   git archive <sha> build-n4-module-kit/toolbelt tests/fixtures | tar -x -C <checkout>
# The goldens were recorded at fa1201e6). Cases are in cases.sh beside this file. Each case runs in its own tree
# $D (the case's cwd); the case file holds args, want (exit + stdout), stderr, @kind (the source and sha), one
# "@expect <file>" section per file in EXP as the run left it, then the tree as it was before the run:
# "<dir> (dir)" an empty directory, "<dir> (locked)" one with mode 000, "<file> (unreadable)" a file with mode
# 000 and "<link> (symlink)" a symlink whose body is its target. A truth-* case is hand-written and says so in
# its @kind; this script never writes one.
set -u
REPO="$(cd "$1" && pwd)"; SHA="$2"; OUT="$3"; FILTER="${4:-}"
GWM="$REPO/build-n4-module-kit/toolbelt/generate-wiring-map.sh"
# shellcheck disable=SC2034  # FIX is read by cases.sh
FIX="$REPO/tests/fixtures"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/gwmrec.XXXXXX")"
mkdir -p "$OUT"

new() { D="$(mktemp -d "$WORK/c.XXXXXX")"; EXP=(); }
put() { mkdir -p "$(dirname "$D/$1")"; printf '%b' "$2" > "$D/$1"; } # put <rel-path> <printf %b body>

sub() { local s="${1//"$D"/@DIR@}"; printf '%s' "$s"; }
serialize() {
  local p rel
  ( cd "$D" && LC_ALL=C find . -mindepth 1 -print | LC_ALL=C sort ) | while IFS= read -r p; do
    rel="${p#./}"; p="$D/$rel"
    if [ -L "$p" ]; then printf -- '-- %s (symlink) (noeol) --\n%s\n' "$rel" "$(sub "$(readlink "$p")")"
    elif [ -d "$p" ]; then
      if [ ! -r "$p" ]; then printf -- '-- %s (locked) --\n' "$rel"
      elif [ -z "$(ls -A "$p")" ]; then printf -- '-- %s (dir) --\n' "$rel"; fi
    elif [ ! -r "$p" ]; then printf -- '-- %s (unreadable) --\n' "$rel"
    else
      local body; body="$(cat "$p"; printf x)"; body="${body%x}"; body="${body//"$D"/@DIR@}"
      if [ -n "$body" ] && [ "${body: -1}" != $'\n' ]; then printf -- '-- %s (noeol) --\n%s\n' "$rel" "$body"
      else printf -- '-- %s --\n%s' "$rel" "$body"; fi
    fi
  done
}
rec() {
  local name="$1"; shift
  # shellcheck disable=SC2053  # FILTER is a glob on purpose
  [ -n "$FILTER" ] && [[ "$name" != $FILTER ]] && return
  local tree; tree="$(serialize; printf x)"; tree="${tree%x}"
  local rc=0
  ( cd "$D" && "$GWM" "$@" > "$WORK/out" 2> "$WORK/err" ) || rc=$?
  find "$D" ! -perm -u+r -exec chmod u+r {} + 2>/dev/null
  find "$D" -type d ! -perm -u+x -exec chmod u+x {} + 2>/dev/null
  {
    printf -- '-- args --\n'; local a; for a in "$@"; do printf '%s\n' "$(sub "$a")"; done
    printf -- '-- want --\nexit %s\n' "$rc"; sed "s#$D#@DIR@#g" "$WORK/out"
    printf -- '-- stderr --\n'; sed "s#$D#@DIR@#g" "$WORK/err"
    printf -- '-- @kind --\nrecorded from build-n4-module-kit/toolbelt/generate-wiring-map.sh at %s\n' "$SHA"
    for a in "${EXP[@]}"; do printf -- '-- @expect %s --\n' "$a"; sed "s#$D#@DIR@#g" "$D/$a"; done
    printf '%s' "$tree"
  } > "$OUT/$name.txt"
}

# shellcheck disable=SC1091  # cases.sh beside this script, resolved at runtime
. "$(dirname "$0")/cases.sh"
chmod -R u+rwx "$WORK"; rm -rf "$WORK"
