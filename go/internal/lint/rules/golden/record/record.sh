#!/usr/bin/env bash
# record.sh <script> <case-dir> <out-dir> [case-glob] — records the golden cases of <case-dir> from the bash
# <script> a Go port replaces, for golden.Runner to replay (README.md beside this file has the case format).
#
# <case-dir>/record.env pins the source: SOURCE (its repository path), COMMIT, SHA256 (of <script>), EXTRA
# (space-separated "<path>:<sha256>" files beside <script> it sources, e.g. lib/fs-type.sh), CHDIR=1 (run in the
# case dir). An optional <case-dir>/setup.sh may define setup_once (run once, with the host environment: build
# a Go tool the script's children exec) and setup_case (per case, after its tree is written: may set RUN, the
# script path to run, SKIP, entries no tree shows ("p/" = p and all below), and SETUP_ENV, env overrides).
#
# Every case not named truth-* is read from <case-dir> for its inputs (args, @env, the tree before the run),
# replayed, and written to <out-dir> with the outputs recorded: exit, stdout, stderr (exactly, "(noeol)" when
# the last line has no newline), @kind (source, commit, sha256 and the bash/awk/find versions), the tree
# before and the @after/@gone delta the run made; record.env and setup.sh are copied beside them, so
# <out-dir> is a whole case set. Compare with `diff -r <case-dir> <out-dir>`.
# Refuses: root (mode 000 would not block reads), a script or EXTRA whose sha256 differs from the pin, a
# script that runs a Go tool (go_tool_bin / go-tool.sh: the recording would compare Go with itself), an
# <out-dir> or work dir inside a git worktree. Each run is `env -i` with only the case's environment (PATH,
# HOME and LC_ALL default to /usr/bin:/bin, the case dir and C, as golden.Runner does).
set -euo pipefail
HOST_PATH="$PATH"; export PATH=/usr/bin:/bin # the recorder's own tools are the system ones; setup_once gets HOST_PATH
die() { printf 'record.sh: %s\n' "$*" >&2; exit 2; }
[ $# -ge 3 ] && [ $# -le 4 ] || die "usage: record.sh <script> <case-dir> <out-dir> [case-glob]"
[ "$(id -u)" != 0 ] || die "refusing to run as root: mode 000 entries would not block reads"
SCRIPT="$(realpath -e "$1")"; CASES="$(realpath -e "$2")"; FILTER="${4:-*}"
REPO="$(cd "$(dirname "$(realpath -e "$0")")/../../../../../.." && pwd)"
[ -f "$CASES/record.env" ] || die "no record.env in $CASES"
SOURCE="" COMMIT="" SHA256="" EXTRA="" CHDIR=""
# shellcheck disable=SC1091  # the case set's own pins
. "$CASES/record.env"
[ -n "$SOURCE" ] && [ -n "$COMMIT" ] && [ -n "$SHA256" ] || die "record.env needs SOURCE, COMMIT and SHA256"
pin() { local sum; sum="$(sha256sum "$1")" || die "cannot read $1"; [ "${sum%% *}" = "$2" ] || die "$1: sha256 ${sum%% *}, pinned $2"; }
pin "$SCRIPT" "$SHA256"
FILES=("$(basename "$SCRIPT")")
for x in $EXTRA; do pin "$(dirname "$SCRIPT")/${x%%:*}" "${x##*:}"; FILES+=("${x%%:*}"); done
for f in "${FILES[@]}"; do
  ! grep -Eq 'go_tool_bin|go-tool\.sh' "$(dirname "$SCRIPT")/$f" || die "$f runs a Go tool: recording it would compare the port with itself"
done
intree() { PATH="$HOST_PATH" git -C "$1" rev-parse --is-inside-work-tree >/dev/null 2>&1; }
! intree "$(dirname "$3")" || die "$3 is inside a git worktree"
mkdir -p "$3"; OUT="$(realpath -e "$3")"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/golden-rec.XXXXXX")"
trap 'chmod -R u+rwx "$WORK" 2>/dev/null || true; rm -rf "$WORK"' EXIT
! intree "$WORK" || die "$WORK is inside a git worktree"
umask 022
mkdir -p "$WORK/bin"
for f in "${FILES[@]}"; do mkdir -p "$WORK/bin/$(dirname "$f")"; cp "$(dirname "$SCRIPT")/$f" "$WORK/bin/$f"; done
BASH_V="$(bash --version | head -n 1)"; AWK_V="$(awk --version 2>/dev/null | head -n 1 || awk -W version 2>&1 | head -n 1)"
# shellcheck disable=SC2185  # GNU find needs no path for --version
FIND_V="$(find --version | head -n 1)"
setup_once() { :; }
setup_case() { :; }
if [ -f "$CASES/setup.sh" ]; then
  # shellcheck disable=SC1091  # the case set's own hooks
  . "$CASES/setup.sh"
fi
PATH="$HOST_PATH" setup_once

# split <case-file> <dir>: one file per section (<dir>/<n>) and <dir>/names (one section name per line).
split() {
  rm -rf "$2"; mkdir -p "$2"
  awk -v d="$2" '
    /^-- .* --$/ { if (f) close(f); n++; f = d "/" n; printf "" > f; print substr($0, 4, length($0) - 6) > (d "/names"); next }
    n { print > f }' "$1"
  [ -f "$2/names" ] || : > "$2/names"
}
subst() { sed -e "s#@DIR@#$D#g" -e "s#@REPO@#$REPO#g" "$1"; }
# materialize <sections-dir> <root>: writes the tree sections under <root>, @DIR@ standing for $D.
materialize() {
  local n=0 name body p modes=()
  while IFS= read -r name; do
    n=$((n + 1)); body="$1/$n"
    case "$name" in args|want|stderr|"want (noeol)"|"stderr (noeol)"|@kind|@env|@after\ *|@gone\ *) continue ;; @*) die "unknown section $name" ;; esac
    local kind=file mode="" noeol=0 m
    while [[ "$name" =~ ^(.*)\ \(([^\)]*)\)$ ]]; do
      m="${BASH_REMATCH[2]}"
      case "$m" in
        noeol) noeol=1 ;; dir|symlink) kind="$m" ;; mode\ [0-7][0-7][0-7]) mode="${m#mode }" ;;
        *) die "section marker ($m) is not supported by record.sh" ;;
      esac
      name="${BASH_REMATCH[1]}"
    done
    p="$2/$name"; mkdir -p "$(dirname "$p")"
    case "$kind" in
      dir) mkdir -p "$p" ;;
      symlink) ln -s "$(subst "$body" | head -n 1)" "$p" ;;
      *) subst "$body" > "$p"; [ "$noeol" = 0 ] || truncate -s -1 "$p" ;;
    esac
    [ -z "$mode" ] || modes+=("$mode $name")
  done < "$1/names"
  local i; for ((i = ${#modes[@]} - 1; i >= 0; i--)); do chmod "${modes[$i]%% *}" "$2/${modes[$i]#* }"; done
}
skipped() { local s; for s in "${SKIP[@]+"${SKIP[@]}"}"; do [ "$1" = "$s" ] || [ "$1/" = "$s" ] || [[ "$s" == */ && "$1" == "$s"* ]] && return 0; done; return 1; }
# snap <root> <dir>: <dir>/<path-index> holds the section of each entry under <root> (SKIP excluded), <dir>/index
# lists "<path>\t<n>" in path order. Unlocks <root> as it reads it.
snap() {
  local root="$1" out="$2" list="$2.list" rel mode type n=0 p
  rm -rf "$out"; mkdir -p "$out"; : > "$list"
  local queue=("")
  while [ ${#queue[@]} -gt 0 ]; do # one find per locked directory: its own mode is listed before it is opened
    rel="${queue[0]}"; queue=("${queue[@]:1}")
    chmod u+rwx "$root/$rel"
    find "$root/$rel" -mindepth 1 -maxdepth 1 -printf "%m\t%y\t${rel:+$rel/}%P\n" >> "$list"
    while IFS=$'\t' read -r mode type p; do queue+=("$p"); done < <(find "$root/$rel" -mindepth 1 -maxdepth 1 -type d -printf "%m\t%y\t${rel:+$rel/}%P\n")
  done
  LC_ALL=C sort -t $'\t' -k 3 "$list" | while IFS=$'\t' read -r mode type rel; do
    skipped "$rel" && continue
    n=$((n + 1)); p="$root/$rel"
    case "$type" in
      l) printf -- '-- %s (symlink) --\n%s\n' "$rel" "$(readlink "$p" | sed -e "s#$D#@DIR@#g" -e "s#$REPO#@REPO@#g")" ;;
      d) if [ -z "$(ls -A "$p")" ]; then printf -- '-- %s (dir)%s --\n' "$rel" "$([ "$mode" = 755 ] || printf ' (mode %03o)' "0$mode")"
         elif [ "$mode" != 755 ]; then printf -- '-- %s (dir) (mode %03o) --\n' "$rel" "0$mode"
         else continue; fi ;;
      *) local m=""; [ "$mode" = 644 ] || m="$(printf ' (mode %03o)' "0$mode")"; chmod u+r "$p"
         sed -e "s#$D#@DIR@#g" -e "s#$REPO#@REPO@#g" "$p" > "$out/body"
         if [ -s "$out/body" ] && [ "$(tail -c 1 "$out/body" | od -An -c | tr -d ' ')" != '\n' ]; then
           printf -- '-- %s%s (noeol) --\n' "$rel" "$m"; cat "$out/body"; printf '\n'
         else printf -- '-- %s%s --\n' "$rel" "$m"; cat "$out/body"; fi ;;
    esac > "$out/$n"
    printf '%s\t%s\n' "$rel" "$n" >> "$out/index"
  done
  rm -f "$out/body"; touch "$out/index"
}
# output <name> <file>: one recorded stream as a section ("(noeol)" when its last line has no newline).
output() {
  sed -e "s#$D#@DIR@#g" -e "s#$REPO#@REPO@#g" "$2" > "$2.sub"
  if [ -s "$2.sub" ] && [ "$(tail -c 1 "$2.sub" | od -An -c | tr -d ' ')" != '\n' ]; then printf -- '-- %s (noeol) --\n' "$1"; cat "$2.sub"; printf '\n'
  else printf -- '-- %s --\n' "$1"; cat "$2.sub"; fi
}

record() { # record <case-file>
  local name; name="$(basename "$1" .txt)"
  split "$1" "$WORK/sec"
  D="$WORK/case"; local B="$WORK/before"
  chmod -R u+rwx "$D" "$B" 2>/dev/null || true; rm -rf "$D" "$B"; mkdir -p "$D" "$B"
  materialize "$WORK/sec" "$D"; materialize "$WORK/sec" "$B"
  local args=() env=() n=0 sec envsec="" k
  while IFS= read -r sec; do
    n=$((n + 1))
    case "$sec" in args) mapfile -t args < <(subst "$WORK/sec/$n") ;; @env) envsec="$WORK/sec/$n"; mapfile -t env < <(subst "$envsec") ;; esac
  done < "$WORK/sec/names"
  local kv found
  for k in PATH=/usr/bin:/bin "HOME=$D" LC_ALL=C; do
    found=0; for kv in "${env[@]+"${env[@]}"}"; do [ "${kv%%=*}" != "${k%%=*}" ] || found=1; done
    [ "$found" = 1 ] || env+=("$k")
  done
  RUN="$WORK/bin/$(basename "$SCRIPT")"; SKIP=(); SETUP_ENV=()
  setup_case
  local rc=0
  ( if [ "$CHDIR" = 1 ]; then cd "$D"; fi
    env -i "${env[@]}" "${SETUP_ENV[@]+"${SETUP_ENV[@]}"}" bash "$RUN" "${args[@]+"${args[@]}"}" > "$WORK/out" 2> "$WORK/err" ) || rc=$?
  snap "$B" "$WORK/sb"; snap "$D" "$WORK/sa"
  {
    n=0; while IFS= read -r sec; do n=$((n + 1)); [ "$sec" = args ] && { printf -- '-- args --\n'; cat "$WORK/sec/$n"; }; done < "$WORK/sec/names"
    printf 'exit %s\n' "$rc" | cat - "$WORK/out" > "$WORK/want"; output want "$WORK/want"; output stderr "$WORK/err"
    printf -- '-- @kind --\nrecorded from %s at %s (sha256 %s)\ntools: %s; %s; %s\n' "$SOURCE" "$COMMIT" "$SHA256" "$BASH_V" "$AWK_V" "$FIND_V"
    [ -z "$envsec" ] || { printf -- '-- @env --\n'; cat "$envsec"; }
    cut -f 2 "$WORK/sb/index" | while IFS= read -r n; do cat "$WORK/sb/$n"; done
    LC_ALL=C join -t $'\t' -a 1 -a 2 -e - -o 0,1.2,2.2 <(LC_ALL=C sort "$WORK/sb/index") <(LC_ALL=C sort "$WORK/sa/index") | LC_ALL=C sort |
    while IFS=$'\t' read -r p b a; do
      if [ "$a" = - ]; then printf -- '-- @gone %s --\n' "$p"
      elif [ "$b" = - ] || ! cmp -s "$WORK/sb/$b" "$WORK/sa/$a"; then sed '1s/^-- /-- @after /' "$WORK/sa/$a"; fi
    done
  } > "$OUT/$name.txt"
}

shopt -s nullglob
count=0
cp "$CASES/record.env" "$OUT/"; [ ! -f "$CASES/setup.sh" ] || cp "$CASES/setup.sh" "$OUT/"
for f in "$CASES"/*.txt; do
  name="$(basename "$f" .txt)"
  # shellcheck disable=SC2053  # FILTER is a glob on purpose
  [[ "$name" == truth-* || "$name" != $FILTER ]] && continue
  record "$f"; count=$((count + 1))
done
printf 'record.sh: recorded %d cases into %s\n' "$count" "$OUT" >&2
