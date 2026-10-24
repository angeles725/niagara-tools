#!/usr/bin/env bash
# record.sh <checkout> <sha> <out-dir> [case-glob] — records the go/internal/preflight golden cases from the bash
# toolbelt/preflight.sh of <checkout> (a tree holding build-n4-module-kit/toolbelt and tests/fixtures/preflight
# as of <sha>; the bash left the toolbelt in the Go switch, so extract it first:
#   git archive <sha> build-n4-module-kit/toolbelt tests/fixtures/preflight | tar -x -C <checkout>
# The goldens were recorded at 5a4668f0). Cases are in cases.sh beside this file. Each case runs under
# `env -i` with PATH=<case>/fakebin:<case>/.tools (.tools links the host tools in TOOLS; it is not written to
# the case) and N4_FSTYPE_MOUNTS_FILE=<case>/mounts, plus the case's ENVV; the case file holds args, want
# (exit + stdout), stderr, @kind (the source and sha), @env, one @expect section per file in EXP as the run
# left it, then the tree as it was before the run. A truth-* case is hand-edited from a recorded one and
# says so in its @kind; this script never writes one.
set -u
REPO="$(cd "$1" && pwd)"; SHA="$2"; OUT="$3"; FILTER="${4:-}"
PF="$REPO/build-n4-module-kit/toolbelt/preflight.sh"
FIX="$REPO/tests/fixtures/preflight"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/pfrec.XXXXXX")"
mkdir -p "$OUT"
TOOLS=(bash sh env grep sed awk cut find paste sort uniq wc tr comm basename dirname head tail xargs df sleep cat printf mkdir ls readlink timeout rm seq)

new() {
  D="$(mktemp -d "$WORK/c.XXXXXX")"
  mkdir -p "$D/fakebin" "$D/.tools"
  local t; for t in "${TOOLS[@]}"; do ln -s "/usr/bin/$t" "$D/.tools/$t"; done
  printf 'ext4 / ext4 rw 0 0\n' > "$D/mounts"
  ENVV=(); EXP=()
}
fx() { # copy the bats fixtures in
  cp -r "$FIX/jvm" "$D/jvm"; cp -r "$FIX/jvm-no-release" "$D/jvmnr"
  cp -r "$FIX/niagara-home" "$D/nh"; cp -r "$FIX/niagara-home-no-pin" "$D/nhnp"; cp -r "$FIX/gradle-root" "$D/gr"
}
lsof_fake() { printf '#!/usr/bin/env bash\n%s\n' "$1" > "$D/fakebin/lsof"; chmod +x "$D/fakebin/lsof"; }
nhome() { mkdir -p "$1/etc/m2/repository/com/tridium/niagara-module/com.tridium.niagara-module.gradle.plugin/$2" "$1/bin" "$1/modules"; }

sub() { local s="${1//"$D"/@DIR@}"; printf '%s' "$s"; }
serialize() {
  local p rel
  ( cd "$D" && LC_ALL=C find . -mindepth 1 -path ./.tools -prune -o -print | LC_ALL=C sort ) | while IFS= read -r p; do
    rel="${p#./}"; p="$D/$rel"
    if [ -L "$p" ]; then
      printf -- '-- %s (symlink) (noeol) --\n%s\n' "$rel" "$(sub "$(readlink "$p")")"
    elif [ -d "$p" ]; then
      if [ ! -r "$p" ]; then printf -- '-- %s/ (locked) --\n' "$rel"
      elif [ -z "$(ls -A "$p")" ]; then printf -- '-- %s/ --\n' "$rel"; fi
    else
      local x="" body; [ -x "$p" ] && x=" (exec)"
      body="$(cat "$p"; printf x)"; body="${body%x}"; body="${body//"$D"/@DIR@}"
      if [ -n "$body" ] && [ "${body: -1}" != $'\n' ]; then printf -- '-- %s%s (noeol) --\n%s\n' "$rel" "$x" "$body"
      else printf -- '-- %s%s --\n%s' "$rel" "$x" "$body"; fi
    fi
  done
}
rec() {
  local name="$1"; shift
  # shellcheck disable=SC2053  # FILTER is a glob on purpose
  [ -n "$FILTER" ] && [[ "$name" != $FILTER ]] && return
  local tree; tree="$(serialize; printf x)"; tree="${tree%x}"
  local rc=0
  ( cd "$D" && env -i PATH="$D/fakebin:$D/.tools" N4_FSTYPE_MOUNTS_FILE="$D/mounts" "${ENVV[@]}" \
      "$D/.tools/bash" "$PF" "$@" > "$WORK/out" 2> "$WORK/err" ) || rc=$?
  find "$D" -type d ! -perm -u+r -exec chmod 755 {} + 2>/dev/null
  {
    printf -- '-- args --\n'; local a; for a in "$@"; do printf '%s\n' "$(sub "$a")"; done
    printf -- '-- want --\nexit %s\n' "$rc"; sed "s#$D#@DIR@#g" "$WORK/out"
    printf -- '-- stderr --\n'; sed "s#$D#@DIR@#g" "$WORK/err"
    printf -- '-- @kind --\nrecorded from build-n4-module-kit/toolbelt/preflight.sh at %s\n' "$SHA"
    printf -- '-- @env --\nPATH=@DIR@/fakebin:@DIR@/.tools\nN4_FSTYPE_MOUNTS_FILE=@DIR@/mounts\n'
    for a in "${ENVV[@]}"; do printf '%s\n' "$(sub "$a")"; done
    for a in "${EXP[@]}"; do printf -- '-- @expect %s --\n' "$a"; sed "s#$D#@DIR@#g" "$D/$a"; done
    printf '%s' "$tree"
  } > "$OUT/$name.txt"
}

# shellcheck disable=SC1091  # cases.sh beside this script, resolved at runtime
. "$(dirname "$0")/cases.sh"
rm -rf "$WORK"
