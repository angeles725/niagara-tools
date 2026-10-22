#!/usr/bin/env bash
# lib/awk-lint.sh — shared skeleton of the per-file awk lints (lint-<name>.sh that write one
# main.awk and run it once per source file). Sourced only (never invoked directly). Fragment
# rule: edit THIS shared fragment, never re-implement the skeleton in a consumer lint — mirrors
# lib/scan-files.sh. VCS-free by design (kit-links L2). Sourcing it also sources scan-files.sh
# and exports LC_ALL=C.
#
# A consumer lint is:
#   . ".../lib/awk-lint.sh"
#   awk_lint_parse <name> <arity> '<usage line>' "$@"
#   cat > "$AL_TMP/main.awk" << 'AWKEOF' ... AWKEOF
#   awk_lint_run <find-test...>
#
# A line-loop lint (a shell loop over the files, no main.awk, no --strict) is:
#   . ".../lib/awk-lint.sh"
#   awk_lint_parse <name> raw '<usage line>' "$@"
#   awk_lint_list <find-test...>
#   while IFS= read -r f; do ...; done < "$AL_TMP/files"
#   awk_lint_exit <code>
#
# awk_lint_parse <name> <arity> <usage> <args...>
#   Parses `[--strict] [--] <root>...`. <name> prefixes every message ("<name>: not a directory: ..."),
#   <usage> is printed verbatim. <arity> says how the roots are taken:
#     one    exactly one root; any other count prints <usage> (exit 3)
#     first  at least one root; only the first is checked and scanned, the rest are ignored
#     each   at least one root; every root is checked before any scan, then each is scanned in order
#     raw    like first, but no flag is parsed: `-h`, `--` or `--strict` is taken as the root
#   Sets AL_NAME, AL_USAGE, AL_STRICT (0/1), AL_ROOTS (array) and AL_TMP (a temp dir removed on EXIT).
#   A flag other than --strict/-- goes to the function named by AL_FLAG_HOOK (set it after sourcing),
#   called with every remaining argument; it sets AL_SHIFT to the count it consumed, or leaves 0 for
#   "unknown flag". Usage, unknown flag and a non-directory root exit 3.
#
# awk_lint_run <find-test...>
#   For each root, lists the files matching the find tests (scan_files), runs
#   `awk -v FILE=<file> <AL_AWK_ARGS...> -f "$AL_TMP/main.awk" <file>` on each, and prints its
#   rows. Exits 3 when a walk was incomplete or an awk run failed, 1 when a row was printed under
#   --strict, 0 otherwise. Never returns.
#
# awk_lint_list <find-test...>
#   Lists the files under the first root that match the find tests into "$AL_TMP/files" (scan_files).
#   An incomplete walk prints "<name>: cannot list every file under <root>: <reason>" and sets
#   AL_ERR=1; the files it did list are still written, so the caller scans them before exiting.
#
# A comment-stripped per-file lint (a grep check per file, no main.awk) is:
#   . ".../lib/awk-lint.sh"
#   awk_lint_parse <name> first '<usage line>' "$@"
#   <check-fn>() { ...reads "$AL_TEXT", prints the line number to report or nothing...; }
#   awk_lint_strip_run <check-fn> <find-test...>
#
# awk_lint_strip_run <check-fn> <find-test...>
#   Lists the files like awk_lint_list, sets AL_TEXT to each file with its // line comments stripped
#   (`sed 's://.*::'`, same line numbers) and calls <check-fn> <file>. A printed line number becomes
#   "WARN  <name>  <file>:<line>  <that source line, trimmed>". An unreadable file is
#   "<name>: cannot scan <file>: <reason>" and the scan goes on. Exits like awk_lint_run (3 / 1 under
#   --strict / 0). Never returns.
#
# awk_lint_exit <code>
#   Exits 3 when AL_ERR is set (an incomplete walk or a failed scan), <code> otherwise.
#
# Why: ten lints carried the same ~50-line copy of this skeleton; a fail-closed fix (issue #226)
# had to land ten times. [ev: retro 2026-10-22-awk-lint-skeleton]

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "${BASH_SOURCE[0]%/*}/scan-files.sh"
LC_ALL=C
export LC_ALL

AL_AWK_ARGS=()
AL_FLAG_HOOK=
AL_SHIFT=0
AL_ERR=0

awk_lint_parse() {
  AL_NAME="$1"
  AL_USAGE="$3"
  local arity="$2" root
  shift 3
  case "$arity" in
    one|first|each|raw) ;;
    *) printf 'awk_lint_parse: unknown arity: %s\n' "$arity" >&2; exit 3 ;;
  esac
  AL_STRICT=0
  while [ "$arity" != raw ] && [ $# -gt 0 ]; do
    case "$1" in
      --strict) AL_STRICT=1; shift ;;
      --) shift; break ;;
      -*) AL_SHIFT=0
          if [ -n "$AL_FLAG_HOOK" ]; then "$AL_FLAG_HOOK" "$@"; fi
          if [ "$AL_SHIFT" -eq 0 ]; then
            printf '%s: unknown flag: %s\n%s\n' "$AL_NAME" "$1" "$AL_USAGE" >&2
            exit 3
          fi
          shift "$AL_SHIFT" ;;
      *) break ;;
    esac
  done
  if [ $# -lt 1 ] || { [ "$arity" = one ] && [ $# -ne 1 ]; }; then
    printf '%s\n' "$AL_USAGE" >&2
    exit 3
  fi
  if [ "$arity" = each ]; then AL_ROOTS=("$@"); else AL_ROOTS=("$1"); fi
  for root in "${AL_ROOTS[@]}"; do
    if [ ! -d "$root" ]; then
      printf '%s: not a directory: %s\n' "$AL_NAME" "$root" >&2
      exit 3
    fi
  done
  AL_TMP=$(mktemp -d)
  trap 'rm -rf "$AL_TMP"' EXIT
}

# _al_list <root> <find-test...>: the fail-closed walk shared by awk_lint_list and awk_lint_run.
_al_list() {
  local root="$1"
  shift
  # A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
  if ! scan_files "$AL_TMP/files" "$AL_TMP/find.err" "$root" "$@"; then
    printf '%s: cannot list every file under %s: %s\n' "$AL_NAME" "$root" "$(head -n 1 "$AL_TMP/find.err")" >&2
    AL_ERR=1
  fi
}

awk_lint_list() {
  _al_list "${AL_ROOTS[0]}" "$@"
}

awk_lint_exit() {
  [ "$AL_ERR" -eq 0 ] || exit 3
  exit "$1"
}

awk_lint_run() {
  local root f out had_warn=0
  for root in "${AL_ROOTS[@]}"; do
    _al_list "$root" "$@"
    while IFS= read -r f; do
      # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
      # The +-guarded expansion keeps an empty AL_AWK_ARGS safe under set -u on bash < 4.4.
      if ! out=$(awk -v FILE="$f" ${AL_AWK_ARGS[@]+"${AL_AWK_ARGS[@]}"} -f "$AL_TMP/main.awk" "$f" 2>"$AL_TMP/awk.err"); then
        printf '%s: cannot scan %s: %s\n' "$AL_NAME" "$f" "$(head -n 1 "$AL_TMP/awk.err")" >&2
        AL_ERR=1
        continue
      fi
      if [ -n "$out" ]; then
        printf '%s\n' "$out"
        had_warn=1
      fi
    done < "$AL_TMP/files"
  done
  if [ "$had_warn" -eq 1 ] && [ "$AL_STRICT" -eq 1 ]; then
    awk_lint_exit 1
  fi
  awk_lint_exit 0
}

awk_lint_strip_run() {
  local check="$1" f line had_warn=0
  shift
  awk_lint_list "$@"
  while IFS= read -r f; do
    # An unreadable file is an env error, never an empty text that passes clean (fail closed).
    # shellcheck disable=SC2034  # AL_TEXT is read by the caller's <check-fn>
    if ! AL_TEXT=$(sed 's://.*::' "$f" 2>"$AL_TMP/sed.err"); then
      printf '%s: cannot scan %s: %s\n' "$AL_NAME" "$f" "$(head -n 1 "$AL_TMP/sed.err")" >&2
      AL_ERR=1
      continue
    fi
    line=$("$check" "$f")
    [ -n "$line" ] || continue
    printf 'WARN  %s  %s:%s  %s\n' "$AL_NAME" "$f" "$line" \
      "$(sed -n "${line}p" "$f" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')"
    had_warn=1
  done < "$AL_TMP/files"
  if [ "$had_warn" -eq 1 ] && [ "$AL_STRICT" -eq 1 ]; then
    awk_lint_exit 1
  fi
  awk_lint_exit 0
}
