#!/usr/bin/env bash
# lint-se-display.sh — a -se part must not use display-dependent AWT/Swing classes.
#
# The -se profile loads on the headless station daemon (JACE launches -rp:rt,se). A
# display class (JFrame/JDialog/JWindow, java.awt.Frame/Window) instantiates a GUI and
# fails at runtime on a headless controller — with NO build-time signal. Headless-safe
# AWT (java.awt.print, java.awt.Font, java.awt.image) is fine. See types/structure.md §-se.
# [ev: retro 2026-09-19-se-profile]
#
# Usage:  lint-se-display.sh <se-src-root>     (a -se profile source tree)
#   Row:  FAIL  lint-se-display  <file>:<line>  display class: <source>
#   Exit: 0  clean · 1  any FAIL · 3  usage/env
# VCS-free by design.
# Mutation: SED2 -- drop the JFrame import case and SED2 stops flagging a display import
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-se-display.sh <se-src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-se-display: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

fail=0
while IFS= read -r f; do
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=${line%%//*}
        case "$code" in
            *import\ javax.swing.JFrame*|*import\ javax.swing.JDialog*|*import\ javax.swing.JWindow*|\
            *import\ javax.swing.JApplet*|*import\ javax.swing.JOptionPane*|\
            *import\ java.awt.Frame*|*import\ java.awt.Window*|*import\ java.awt.Dialog*|\
            *new\ JFrame*|*new\ JDialog*|*new\ JWindow*)
                trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
                printf 'FAIL  lint-se-display  %s:%s  display class: %s\n' "$f" "$ln" "$trimmed"
                fail=1
                ;;
        esac
    done < "$f"
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print 2>/dev/null)

exit "$fail"
