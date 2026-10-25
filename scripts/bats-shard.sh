#!/usr/bin/env bash
# bats-shard.sh — print one CI shard's share of tests/*.bats, balanced by measured duration.
#
# Files in the weight table (built in below, or --weights FILE) are placed longest-first onto the
# currently lightest shard (ties: lower shard index, then file name). Files missing from the table go
# round-robin by sorted name starting at shard 1, so a new test file always runs somewhere. Table rows
# naming a file that no longer exists are ignored. The split is deterministic (C-locale sort). A shard
# lists its weighted files heaviest first, then the rest by name: `bats -j` starts files in argument
# order, so the longest file starts at once instead of late in the alphabet (where it set the old wall).
#
# Usage:
#   bats-shard.sh [--dir DIR] [--weights FILE] <index> <count>   print shard <index> of <count>, one path per line
#   bats-shard.sh [--dir DIR] [--weights FILE] --check <count>   verify the shards partition DIR/*.bats exactly
#     --dir DIR       tests directory (default: tests, relative to the repo root; output paths keep that prefix)
#     --weights FILE  weight table "<seconds> <file.bats>" per line, # comments (default: the built-in table)
#
# Exit: 0 ok · 1 --check found a file in no shard or in two shards · 2 usage · 3 environment (no .bats files)
set -u
export LC_ALL=C

usage() { sed -n '/^# Usage:/,/^#$/p' "$0" | sed 's/^# \{0,1\}//' >&2; exit 2; }

# The slowest files, in seconds: timed one file at a time locally (bats 1.14, 2026-10-25), with the three
# files whose local time is unrepresentative (report-module, ci-local, preflight) set from the CI TAP
# stream of run 40173517342. Only the rough ordering of the heavy files matters; refresh when one shard
# drifts well past the other.
builtin_weights() {
  cat <<'EOF'
# seconds file
70 report-module.bats
25 ci-local.bats
15 build-sh.bats
14 preflight.bats
9.2 scaffold-module.bats
9.1 ng-deploy.bats
9.1 kit-state-hooks.bats
7.3 lint-timers.bats
6.8 verify-module.bats
6.6 run-pure-test.bats
6.6 build-retro-sync.bats
6.5 commissioning-verify.bats
6.1 lint-empty-root.bats
5.2 kit-links.bats
4.7 go-tool.bats
4.6 lint-size.bats
4.3 permissions-root.bats
4.2 rc-scan.bats
4.2 lint-inert-coordination.bats
4.1 kit-resolve.bats
4.1 go-wrapper-symlink.bats
3.9 demand-in-scope.bats
3.9 bog-audit.bats
3.6 triage-console.bats
3.5 lint-write-path.bats
3.2 gen-retro-index.bats
EOF
}

dir="" weights="" check=0
while [ $# -gt 0 ]; do
  case "$1" in
    --dir) [ $# -ge 2 ] || usage; dir="$2"; shift 2 ;;
    --weights) [ $# -ge 2 ] || usage; weights="$2"; shift 2 ;;
    --check) check=1; shift ;;
    -h|--help) usage ;;
    --*) usage ;;
    *) break ;;
  esac
done

isnum() { case "$1" in ''|*[!0-9]*) return 1 ;; esac; [ "$1" -gt 0 ]; }
if [ "$check" -eq 1 ]; then
  { [ $# -eq 1 ] && isnum "$1"; } || usage
  index=0 count="$1"
else
  { [ $# -eq 2 ] && isnum "$1" && isnum "$2" && [ "$1" -le "$2" ]; } || usage
  index="$1" count="$2"
fi

if [ -z "$dir" ]; then
  cd "$(dirname "$0")/.." || exit 3
  dir=tests
fi
if [ -n "$weights" ]; then
  [ -r "$weights" ] || { printf 'bats-shard: cannot read weight table %s\n' "$weights" >&2; exit 3; }
  table="$(cat "$weights")"
else
  table="$(builtin_weights)"
fi

files="$(find "$dir" -maxdepth 1 -type f -name '*.bats' -printf '%f\n' | sort)"
[ -n "$files" ] || { printf 'bats-shard: no .bats files in %s\n' "$dir" >&2; exit 3; }

# assign — print "<shard> <seconds> <file>" for every file: weighted heaviest first, then the rest by name.
assign() {
  awk -v n="$count" '
    FNR == NR { if ($0 !~ /^[[:space:]]*(#|$)/ && $1 ~ /^[0-9.]+$/) w[$2] = $1 + 0; next }
    { name[++nf] = $0 }
    END {
      for (i = 1; i <= nf; i++) if (name[i] in w) heavy[++nh] = name[i]
      # Longest first (ties: file name): insertion sort, the table is small.
      for (i = 2; i <= nh; i++) {
        x = heavy[i]
        for (j = i - 1; j >= 1 && (w[heavy[j]] < w[x] || (w[heavy[j]] == w[x] && heavy[j] > x)); j--) heavy[j + 1] = heavy[j]
        heavy[j + 1] = x
      }
      for (s = 1; s <= n; s++) load[s] = 0
      for (i = 1; i <= nh; i++) {
        best = 1
        for (s = 2; s <= n; s++) if (load[s] < load[best]) best = s
        shard[heavy[i]] = best; load[best] += w[heavy[i]]
      }
      rr = 0
      for (i = 1; i <= nf; i++) if (!(name[i] in shard)) { shard[name[i]] = rr % n + 1; rr++ }
      for (i = 1; i <= nh; i++) printf "%d %s %s\n", shard[heavy[i]], w[heavy[i]], heavy[i]
      for (i = 1; i <= nf; i++) if (!(name[i] in w)) printf "%d 0 %s\n", shard[name[i]], name[i]
    }' <(printf '%s\n' "$table") <(printf '%s\n' "$files")
}

rows="$(assign)"
if [ "$check" -eq 0 ]; then
  awk -v i="$index" -v d="$dir" '$1 == i { print d "/" $3 }' <<<"$rows"
  exit 0
fi

# --check: list every shard through the same printing path and compare their union with DIR/*.bats.
listed="$(for ((s = 1; s <= count; s++)); do awk -v i="$s" '$1 == i { print $3 }' <<<"$rows"; done | sort)"
dups="$(uniq -d <<<"$listed")"
missing="$(comm -23 <(printf '%s\n' "$files") <(uniq <<<"$listed"))"
if [ -n "$dups" ] || [ -n "$missing" ]; then
  [ -z "$dups" ] || printf 'FAIL bats-shard in two shards: %s\n' "$dups"
  [ -z "$missing" ] || printf 'FAIL bats-shard in no shard: %s\n' "$missing"
  exit 1
fi
loads="$(awk -v n="$count" '{ c[$1]++; t[$1] += $2 } END { for (s = 1; s <= n; s++) printf " shard%d=%d files/%.0fs", s, c[s], t[s] }' <<<"$rows")"
printf 'PASS bats-shard %d files in %d shards:%s (weighted seconds)\n' "$(wc -l <<<"$files")" "$count" "$loads"
