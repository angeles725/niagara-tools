#!/usr/bin/env bash
# check-installed-line.sh <repo> <branch> <installed-ref> — refuse a client branch that would silently downgrade a
# module group already installed: every GROUP build.gradle.kts carrying defaultModuleVersion("X") at <installed-ref>
# must exist on <branch> with a version >= X. A fix branch must start from the installed line; run this with the
# installed ref before building (BUILD-LOOP.md). It lives in scripts/ because toolbelt scripts never call version
# control (kit-links L2); the read and the compare are the shared lib/vendor-version.sh (fragment rule).
# Local refs only — never fetches. [ev: retro r10-installed-line-guard Δ1]
#
# Rows: ok|DOWNGRADE|INCOMPARABLE <file> <installed> -> <branch>  ·  MISSING <file> <installed>
# Exit: 0 every group >= installed · 1 any DOWNGRADE, MISSING or INCOMPARABLE
#       2 usage, unknown ref, or no defaultModuleVersion at <installed-ref> (nothing to guard is never a pass)
set -euo pipefail
# shellcheck disable=SC1091  # kit lib, resolved at runtime
. "$(cd "$(dirname "$0")/.." && pwd)/build-n4-module-kit/toolbelt/lib/vendor-version.sh"
[ $# -eq 3 ] || { echo "usage: check-installed-line.sh <repo> <branch> <installed-ref>" >&2; exit 2; }
repo=$1 br=$2 inst=$3
for r in "$br" "$inst"; do
  git -C "$repo" rev-parse -q --verify "$r^{commit}" >/dev/null || { echo "check-installed-line: unknown ref $r"; exit 2; }
done
ver() { { git -C "$repo" show "$1:$2" 2>/dev/null || true; } | vv_source_version; } # ver <ref> <file>; empty = absent
rc=0 n=0
while IFS= read -r f; do
  v=$(ver "$inst" "$f"); [ -n "$v" ] || continue
  n=$((n + 1)); b=$(ver "$br" "$f")
  if [ -z "$b" ]; then echo "MISSING $f $v"; rc=1; continue; fi
  case "$(vv_cmp "$b" "$v")" in
    -1) echo "DOWNGRADE $f $v -> $b"; rc=1 ;;
    0|1) echo "ok $f $v -> $b" ;;
    *) echo "INCOMPARABLE $f $v -> $b"; rc=1 ;;
  esac
done < <(git -C "$repo" grep -l -e 'defaultModuleVersion(' "$inst" -- '*build.gradle.kts' | sed "s|^$inst:||")
[ "$n" -gt 0 ] || { echo "check-installed-line: no defaultModuleVersion in any build.gradle.kts at $inst"; exit 2; }
exit "$rc"
