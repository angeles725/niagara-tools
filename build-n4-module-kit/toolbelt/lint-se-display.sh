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
#   Exit: 0  clean · 1  any FAIL · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: SED2 -- drop the JFrame import case and SED2 stops flagging a display import
# Mutation: SED-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: SED-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-se-display raw 'usage: lint-se-display.sh <se-src-root>' "$@"
awk_lint_list -name '*.java'

# One awk per file (was a shell read loop + a sed fork per hit). Line // comments are stripped first;
# each listed text is a plain substring of the code.
# shellcheck disable=SC2016  # a literal awk program: $0, $n are awk fields
awk_lint_each '
    BEGIN {
        n = split("import javax.swing.JFrame|import javax.swing.JDialog|import javax.swing.JWindow|" \
                  "import javax.swing.JApplet|import javax.swing.JOptionPane|" \
                  "import java.awt.Frame|import java.awt.Window|import java.awt.Dialog|" \
                  "new JFrame|new JDialog|new JWindow", pat, "|")
    }
    { code = $0; i = index(code, "//"); if (i) code = substr(code, 1, i - 1) }
    {
        for (k = 1; k <= n; k++) if (index(code, pat[k])) break
        if (k > n) next
        t = $0; sub(/^[[:space:]]+/, "", t)
        printf "FAIL  lint-se-display  %s:%d  display class: %s\n", ENVIRON["AL_FILE"], FNR, t
    }'

awk_lint_exit "$AL_ROWS"
