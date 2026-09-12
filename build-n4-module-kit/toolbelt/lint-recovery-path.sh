#!/usr/bin/env bash
# lint-recovery-path.sh — Protection-output recovery-path gate (Wave 3, LR1, B860).
#
# Detects the "stranded-heater" shape: a protection or heat output slot
# (*Out, *Resistance, *Heat, *Resistencia) is written OFF/safe ONLY inside a
# mode-gated method and has NO unconditional reset on the started()/atSteadyState()/
# enable()/start() path.
#
# Real commissioning defect (PANCCADIA 2026-09): resistanceOut was turned off inside
# execute() guarded by `if (inDefrost) return;` but started() never reset it
# unconditionally — when the mode cleared, the heater stayed off silently.
#
# Usage:  lint-recovery-path.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir>.
#   Emits FAIL rows; exits 1 on any FAIL, 0 clean, 3 usage/env.
#
# Row format:  FAIL  lint-recovery-path  <file>:<line>  LRP: <slot> guarded-only in <method> -- no reset in started/atSteadyState/enable
# Exits:       0 no FAIL · 1 any FAIL · 3 usage/env (K20)
#
# Dot-directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Documented limitation (grep-level heuristic):
#   False negatives are possible when the unconditional reset occurs in a helper method
#   that is called from started() but not itself named started/atSteadyState/enable/start.
#   Cross-method call chains cannot be followed at this level of analysis. Default posture
#   is FAIL (hard-safety): an undetected reset produces a false FAIL; an undetected missing
#   reset is a silent production defect. Accept the false FAIL and add the unconditional
#   reset inline in started() or suppress with a documented comment.
#
# Source of classification:
#   UNCONDITIONAL: getX().setValue(false) or setX(false) inside a method named
#     started | atSteadyState | changed | enable | start
#   GUARDED:       same write inside any other method
#   FAIL when:     slot has GUARDED write AND no UNCONDITIONAL write in this file.
#
# Mutation: LRP-noguard -- removes the guarded-only classification so mode-gated-only safe-offs pass instead of FAIL
set -u
LC_ALL=C
export LC_ALL

[ $# -ge 1 ] || { printf 'usage: lint-recovery-path.sh <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-recovery-path: not a directory: %s\n' "$SRC" >&2; exit 3; }

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

FAILED=0

while IFS= read -r f; do
  result=$(awk -v FILE="$f" '
  function count_char(s, c,   n, i, ch) {
    n = 0
    for (i = 1; i <= length(s); i++) {
      ch = substr(s, i, 1)
      if (ch == c) n++
    }
    return n
  }

  { lines[NR] = $0 }

  END {
    depth = 0; cur_meth = ""

    for (i = 1; i <= NR; i++) {
      ln = lines[i]
      # Strip line comments for structural analysis
      s = ln; sub(/\/\/.*$/, "", s)

      d_open  = count_char(s, "{")
      d_close = count_char(s, "}")

      # When transitioning from class body (depth==1) into a method (depth==2),
      # identify the method name by scanning the current and up to 5 preceding lines.
      if (depth == 1 && d_open > 0) {
        cur_meth = ""
        for (j = i; j >= i - 5 && j >= 1; j--) {
          t = lines[j]; sub(/\/\/.*$/, "", t)
          if (match(t, /[a-z][A-Za-z0-9_]+[[:space:]]*\(/)) {
            seg = substr(t, RSTART)
            match(seg, /^[a-z][A-Za-z0-9_]+/)
            nm = substr(seg, 1, RLENGTH)
            # Skip control-flow and call-site keywords
            if (nm !~ /^(if|for|while|switch|try|catch|super|this|return|throw|new|else|assert|synchronized|do)$/) {
              cur_meth = nm
              break
            }
          }
        }
      }

      depth += d_open - d_close
      # Back at class level or above: clear current method context
      if (depth <= 1) cur_meth = ""

      # Check for a safe-off write on a protection output slot.
      # Pattern A: getXxx().setValue(false) where Xxx ends in Out|Resistance|Heat|Resistencia
      # Pattern B: setXxx(false) where Xxx ends in Out|Resistance|Heat|Resistencia
      slot = ""
      if (match(s, /get[A-Za-z][A-Za-z0-9_]*\(\)\.setValue\([[:space:]]*false[[:space:]]*\)/)) {
        seg = substr(s, RSTART)
        sub(/^get/, "", seg); sub(/\(\)\.setValue.*$/, "", seg)
        if (seg ~ /(Out|Resistance|Heat|Resistencia)$/) slot = seg
      }
      if (slot == "" && match(s, /set[A-Za-z][A-Za-z0-9_]*\([[:space:]]*false[[:space:]]*\)/)) {
        seg = substr(s, RSTART)
        sub(/^set/, "", seg); sub(/\(.*$/, "", seg)
        if (seg ~ /(Out|Resistance|Heat|Resistencia)$/) slot = seg
      }

      if (slot != "") {
        m = (cur_meth != "") ? cur_meth : "unknown"
        if (m ~ /^(started|atSteadyState|changed|enable|start)$/) {
          unconditional[slot] = 1
        } else {
          # Record first guarded write per slot
          if (!(slot in guarded_line)) {
            guarded_line[slot] = i
            guarded_meth[slot] = m
          }
        }
      }
    }

    # Emit FAIL for each slot with a guarded-only write and no unconditional reset
    for (slot in guarded_line) {
      if (!(slot in unconditional)) {
        printf "FAIL  lint-recovery-path  %s:%d  LRP: %s has guarded-only safe-off in method \"%s\" -- no unconditional reset in started/atSteadyState/enable\n",
          FILE, guarded_line[slot], slot, guarded_meth[slot]
      }
    }
  }
  ' "$f")

  if [ -n "$result" ]; then
    printf '%s\n' "$result"
    printf '%s\n' "$result" | grep -q '^FAIL' && FAILED=1
  fi
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

[ "$FAILED" -eq 0 ] || exit 1
exit 0
