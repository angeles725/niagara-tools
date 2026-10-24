#!/usr/bin/env bash
# lib/slotomatic-skip.sh — shared detector for a source file Slot-o-Matic skipped (METHODOLOGY K15).
# Sourced only (never invoked directly) by toolbelt/build.sh (both gradle branches) and scripts/ng-deploy.sh
# (run_slotomatic). Fragment rule: edit THIS shared fragment, never re-implement the skip scan in a consumer —
# mirrors lib/method-boundary.sh's fragment rule.
#
# Why: Slot-o-Matic throws "Found multiple metadata blocks" for a generated region whose /*@ ... @*/ stamp count
# is not exactly one (devkit-wb vineflower JavaUnit.java:265-272). Compiler.process (Compiler.java:240-246) logs
# "Could not process <path>.java: <msg>", counts the failure and returns; Slotomatic.printResults
# (Slotomatic.java:148-158) prints "The following files failed to compile: " and one "  <abs path>: <msg>" line
# per file, all at WARNING level — and the gradle task still ends BUILD SUCCESSFUL. A jar built after a skip
# carries stale (or no) generated code, so a skip is a FAIL whether or not compileJava then passes.
#
# slotomatic_skip_scan <gradle-log>  — prints one row per skipped file, in log order, each file once:
#   FAIL slotomatic-skip <path> (<reason>)
# A file named by both forms is one row. A file under srcTest/ is ignored: Slot-o-Matic walks test sources
# too (a plain B*Test.java has no metadata block), but they never reach the module jar. Returns 0 when no file was skipped, 1 when any was, 2 (with a
# FAIL row naming the log) when the log cannot be read — fail closed, never a silent pass.
# One awk over the whole log, no per-line process fork (tests/lint-fork-per-line.bats doctrine).
# [ev: retro r10-slotomatic-skip-fail-open Δ1]
slotomatic_skip_scan() {
  local log="$1"
  if [ ! -r "$log" ] || [ -d "$log" ]; then
    printf 'FAIL slotomatic-skip %s (gradle log unreadable)\n' "$log"
    return 2
  fi
  awk '
    # split_entry "<path>.java[: <msg>]" -> sets P (path) and R (reason); 0 when s names no .java file.
    function split_entry(s, dflt,   j) {
      j = index(s, ".java: ")
      if (j > 0) { P = substr(s, 1, j + 4); R = substr(s, j + 7); return 1 }
      if (s ~ /\.java$/) { P = s; R = dflt; return 1 }
      return 0
    }
    function add(p, r) {
      if (p in seen || p ~ /[\/\\]srcTest[\/\\]/) return   # srcTest/ is never packaged: not a stale-jar FAIL
      seen[p] = 1; n++
      printf "FAIL slotomatic-skip %s (%s)\n", p, (r == "" ? "skipped by Slot-o-Matic" : r)
    }
    { sub(/\r$/, "") }
    (i = index($0, "Could not process ")) > 0 {
      if (split_entry(substr($0, i + 18), "Could not process")) add(P, R)
      next
    }
    index($0, "The following files failed to compile:") { inblk = 1; next }
    inblk && /^> Task |^BUILD (SUCCESSFUL|FAILED)/ { inblk = 0; next }
    # A block entry is "  <path>.java[: <msg>]" (two-space indent, after any logger prefix). A javac
    # "X.java:12: error" line has no ".java: " and does not end in ".java", so it never matches.
    inblk && match($0, /[[:space:]][[:space:]]+[^[:space:]]/) {
      s = substr($0, RSTART + RLENGTH - 1)
      sub(/[[:space:]]+$/, "", s)
      if (split_entry(s, "listed under: The following files failed to compile")) add(P, R)
    }
    END { exit (n > 0) }
  ' "$log"
}
