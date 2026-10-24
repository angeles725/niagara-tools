#!/usr/bin/env bash
# record.sh <checkout> <sha> <out-dir> [case-glob] — records the go/internal/wiringmap golden cases from the bash
# toolbelt/generate-wiring-map.sh of <checkout>: a scratch tree holding build-n4-module-kit/toolbelt and
# tests/fixtures as of <sha>. The Go switch replaced the bash body with the go_tool_bin wrapper, so extract the
# old tree into a fresh scratch directory, never into a live worktree:
#   scratch="$(mktemp -d)"; git archive <sha> build-n4-module-kit/toolbelt tests/fixtures | tar -x -C "$scratch"
#   memcap go/internal/wiringmap/testdata/record/record.sh "$scratch" <sha> <out-dir>
# The goldens were recorded at 15e792d1 (the port's base; the script is unchanged since fa1201e6, but its
# lib/scan-files.sh walk gained *-worktrees pruning and symlinked-root following after that). The script must hash
# to BASH_SHA256 and the sourced lib to LIB_SHA256, both as of 15e792d1, and a wrapper that sources go-tool.sh is
# refused outright, so the Go port is never recorded against itself.
# Cases are in cases.sh beside this file. Each case runs in its own tree $D (the case's cwd) under
# `env -i PATH=/usr/bin:/bin LC_ALL=C`; the case file holds args, want (exit + stdout), stderr, @kind (the source
# and sha), @tools (the tool versions and the script hash; the runner ignores it), one "@expect <file>" section per
# file the run created or changed, then the tree as it was before the run: "<dir> (dir)" an empty directory,
# "<dir> (locked)" one with mode 000, "<file> (unreadable)" a file with mode 000 and "<link> (symlink)" a symlink
# whose body is its target. A run that deletes a file is refused. A truth-* case is hand-written and says so in
# its @kind; this script never writes one.
set -euo pipefail
[ $# -ge 3 ] || { printf 'usage: record.sh <checkout> <sha> <out-dir> [case-glob]\n' >&2; exit 2; }
[ "$(id -u)" -ne 0 ] || { printf 'record.sh: refusing to run as root (mode 000 does not block root)\n' >&2; exit 2; }
REPO="$(cd "$1" && pwd)"; SHA="$2"; OUT="$3"; FILTER="${4:-}"
GWM="$REPO/build-n4-module-kit/toolbelt/generate-wiring-map.sh"
BASH_SHA256=310c9bd76b5f2767b648dca02315b1a8e78b02d4cda02853cef77f2542bd5b48
LIB_SHA256=7cafd5baa19f807165ae4ac74a76b0cc92f669ecf43ae94a3448d3ff58fd9b99
LIB="$REPO/build-n4-module-kit/toolbelt/lib/scan-files.sh"
# shellcheck disable=SC2034  # FIX is read by cases.sh
FIX="$REPO/tests/fixtures"
if grep -q -e 'go-tool\.sh' -e 'go_tool_bin' "$GWM"; then
  printf 'record.sh: %s is the Go wrapper, not the bash body; extract the old tree (see the header)\n' "$GWM" >&2; exit 2
fi
SUM="$(sha256sum < "$GWM" | cut -d' ' -f1)"
[ "$SUM" = "$BASH_SHA256" ] || { printf 'record.sh: %s hashes to %s, want %s\n' "$GWM" "$SUM" "$BASH_SHA256" >&2; exit 2; }
LSUM="$(sha256sum < "$LIB" | cut -d' ' -f1)"
[ "$LSUM" = "$LIB_SHA256" ] || { printf 'record.sh: %s hashes to %s, want %s\n' "$LIB" "$LSUM" "$LIB_SHA256" >&2; exit 2; }
WORK="$(mktemp -d "${TMPDIR:-/tmp}/gwmrec.XXXXXX")"
trap 'chmod -R u+rwx "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT
mkdir -p "$OUT"
# shellcheck disable=SC2185  # find --version takes no path
TOOLS="$(printf '%s\n' "$(/usr/bin/bash --version | head -n 1)" "$(/usr/bin/awk --version | head -n 1)" \
  "$(/usr/bin/find --version | head -n 1)" "$(/usr/bin/sort --version | head -n 1)" "generate-wiring-map.sh sha256 $SUM" "lib/scan-files.sh sha256 $LSUM")"

new() { D="$(mktemp -d "$WORK/c.XXXXXX")"; }
put() { mkdir -p "$(dirname "$D/$1")"; printf '%b' "$2" > "$D/$1"; } # put <rel-path> <printf %b body>

sub() { local s="${1//"$D"/@DIR@}"; printf '%s' "$s"; }
# section <header> <file>: one file section, "(noeol)" when its last line has no newline; @DIR@ for $D.
section() {
  local body; body="$(cat "$2"; printf x)"; body="${body%x}"; body="${body//"$D"/@DIR@}"
  if [ -n "$body" ] && [ "${body: -1}" != $'\n' ]; then printf -- '-- %s (noeol) --\n%s\n' "$1" "$body"
  else printf -- '-- %s --\n%s' "$1" "$body"; fi
}
serialize() {
  local p rel
  ( cd "$D" && { find . -mindepth 1 -print 2>/dev/null || true; } | LC_ALL=C sort ) | while IFS= read -r p; do
    rel="${p#./}"; p="$D/$rel"
    if [ -L "$p" ]; then printf -- '-- %s (symlink) (noeol) --\n%s\n' "$rel" "$(sub "$(readlink "$p")")"
    elif [ -d "$p" ]; then
      if [ ! -r "$p" ]; then printf -- '-- %s (locked) --\n' "$rel"
      elif [ -z "$(ls -A "$p")" ]; then printf -- '-- %s (dir) --\n' "$rel"; fi
    elif [ ! -r "$p" ]; then printf -- '-- %s (unreadable) --\n' "$rel"
    else section "$rel" "$p"; fi
  done
}
# snap: "<sha256>  <path>" of every readable regular file of the case tree, sorted by path.
snap() { ( cd "$D" && { find . -type f -readable -print0 2>/dev/null || true; } | LC_ALL=C sort -z | xargs -0 -r sha256sum ); }
rec() {
  local name="$1"; shift
  # shellcheck disable=SC2053  # FILTER is a glob on purpose
  if [ -n "$FILTER" ] && [[ "$name" != $FILTER ]]; then return 0; fi
  local tree before after rc=0 a
  tree="$(serialize; printf x)"; tree="${tree%x}"
  before="$(snap)"
  ( cd "$D" && env -i PATH=/usr/bin:/bin LC_ALL=C /usr/bin/bash "$GWM" "$@" > "$WORK/out" 2> "$WORK/err" ) || rc=$?
  after="$(snap)"
  if [ -n "$(LC_ALL=C comm -23 <(cut -c67- <<< "$before" | LC_ALL=C sort) <(cut -c67- <<< "$after" | LC_ALL=C sort))" ]; then
    printf 'record.sh: %s: the run deleted a file; the golden format cannot spell that\n' "$name" >&2; exit 2
  fi
  chmod -R u+rwx "$D"
  {
    printf -- '-- args --\n'; for a in "$@"; do printf '%s\n' "$(sub "$a")"; done
    printf -- '-- want --\nexit %s\n' "$rc"; sed "s#$D#@DIR@#g" "$WORK/out"
    printf -- '-- stderr --\n'; sed "s#$D#@DIR@#g" "$WORK/err"
    printf -- '-- @kind --\nrecorded from build-n4-module-kit/toolbelt/generate-wiring-map.sh at %s\n' "$SHA"
    printf -- '-- @tools --\n%s\n' "$TOOLS"
    LC_ALL=C comm -13 <(LC_ALL=C sort <<< "$before") <(LC_ALL=C sort <<< "$after") | cut -c67- | LC_ALL=C sort |
      while IFS= read -r a; do if [ -n "$a" ]; then section "@expect ${a#./}" "$D/$a"; fi; done
    printf '%s' "$tree"
  } > "$OUT/$name.txt"
}

# shellcheck disable=SC1091  # cases.sh beside this script, resolved at runtime
. "$(dirname "$0")/cases.sh"
