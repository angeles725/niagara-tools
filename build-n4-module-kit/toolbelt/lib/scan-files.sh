#!/usr/bin/env bash
# lib/scan-files.sh — shared fail-closed source-file lister for the kit lints.
# Sourced only (never invoked directly). Fragment rule: edit THIS shared fragment, never
# re-implement the file walk separately in a consumer lint — mirrors lib/method-boundary.sh.
# VCS-free by design (kit-links L2).
#
# scan_files <out-file> <err-file> <root> [--prune <dir-name>]... <find-test...>
#   Writes every regular file under <root> that matches the find tests (for example
#   `-name '*.java'`, or `-name '*.html' -o -name '*.js'`) to <out-file>, one per line,
#   sorted with LC_ALL=C. Dot SUB-directories are pruned, and so is every sub-directory named by a
#   `--prune <dir-name>` (e.g. `--prune build` for gradle outputs); the root itself never is (`-mindepth 1`),
#   so `.`, `..` or a dot-named root is walked, not pruned whole. Returns 0 when the walk was
#   complete, and non-zero when <root> is not a directory, find could not enter a sub-directory or
#   the sorted list could not be written; <err-file> then holds the reason.
#
# Why: a lint that reads `find ... | sort` through a process substitution never sees find's
# exit status, so a sub-directory find cannot enter is skipped and the lint reports the
# module clean. A caller treats a non-zero return as an env error (exit 3), never as a
# clean scan. [ev: issue #226 R3-find-error-still-fail-open]
scan_files() {
  local out="$1" err="$2" root="$3" rc=0
  local -a prune=( -name '.*' )
  shift 3
  while [ "${1:-}" = "--prune" ] && [ $# -ge 2 ]; do
    prune+=( -o -name "$2" )
    shift 2
  done
  # -mindepth 1 never tests the root itself, so a FILE root would list nothing and look clean.
  if [ ! -d "$root" ]; then
    printf 'scan_files: not a directory: %s\n' "$root" > "$err"
    : > "$out"
    return 2
  fi
  find "$root" -mindepth 1 \( -type d \( "${prune[@]}" \) -prune \) -o -type f \( "$@" \) -print > "$out.unsorted" 2> "$err" || rc=$?
  { LC_ALL=C sort "$out.unsorted" > "$out"; } 2>> "$err" || rc=1
  rm -f "$out.unsorted"
  return "$rc"
}
