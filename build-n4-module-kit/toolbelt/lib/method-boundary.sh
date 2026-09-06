#!/usr/bin/env bash
# method-boundary.sh — shared awk method-boundary parser library (C11 T1).
# ONE shell variable: MB_AWK — awk function definitions only (no BEGIN/END/pattern rules).
# Consumed by lint-timers.sh, lint-silent-protection.sh, lint-ext-writable-shape.sh.
#
# Consumption forms (D1b):
#   inline:   awk "$MB_AWK"' { lines[++n]=$0 } END { mb_strip(lines,n,slines); ... }' "$f"
#   heredoc:  printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"
#             awk ... -f "$_TMP/method-boundary.awk" -f "$_TMP/main.awk" "$f"
#
# Interface (D1d):
#   mb_strip(src, n, dst)
#     Blank // and /* */ content; block-comment state carried across lines;
#     line numbers preserved. dst[1..n] out.
#
#   mb_parse(src, n, m_start, m_end, m_name) -> cnt
#     INPUT   src[1..n]  ALREADY comment-stripped lines (caller runs mb_strip first); n = line count
#     OUTPUT  m_start[0..cnt-1]  first line of the method (the line carrying the opening brace)
#             m_end  [0..cnt-1]  line on which the method's brace closes (== m_start for a one-liner)
#             m_name [0..cnt-1]  resolved identifier
#     RETURN  cnt — number of methods found
#     All awk locals declared as trailing params. No globals read or written. No I/O.
#
# Invariants (D1h):
#   I1  PEAK open: max_d > old_d && max_d >= 2 (class-body guard; depth-1 annotaton/FIELD never fires)
#   I2  Case-B backward scan stops at @ (annotation line)
#   I3  Keyword exclusion: if|for|while|switch|catch|try|else|do|new not named as methods
#   I4  One-liner accessor skip: (one-liner) AND mname ~ /^(get|set|is)[A-Z_]/ -> not entered
#   I5  /* */ strip in mb_strip alongside // (B832-G2; no biting fixture; pinned by 3x3 baselines)
#
# Location: sourced via BASH_SOURCE, not $KIT (D1c — avoids writable-target seam).
# shellcheck shell=bash

# shellcheck disable=SC2034  # MB_AWK is exported to consumers via `. method-boundary.sh`; not unused
# shellcheck disable=SC2016  # awk program text in single quotes — $ belongs to awk, not shell
MB_AWK='
function mb_strip(src, n, dst,    i, ln, out, j, in_bc, c2) {
    in_bc = 0
    for (i = 1; i <= n; i++) {
        ln = src[i]; out = ""; j = 1
        while (j <= length(ln)) {
            if (in_bc) {
                if (substr(ln, j, 2) == "*/") { in_bc = 0; j += 2 }
                else j++
            } else {
                c2 = substr(ln, j, 2)
                if (c2 == "/*") { in_bc = 1; j += 2 }
                else if (c2 == "//") break
                else { out = out substr(ln, j, 1); j++ }
            }
        }
        dst[i] = out
    }
}

function mb_parse(src, n, m_start, m_end, m_name,    cnt, i, ln, old_d, brace_depth, max_d, in_m, m_dep, m_st, cur_name, mname, ci, c, seg, bonly, k, lk, lk_t, seg2, cn) {
    cnt = 0; brace_depth = 0; in_m = 0; m_dep = 0; m_st = 0; cur_name = ""
    for (i = 1; i <= n; i++) {
        ln = src[i]; old_d = brace_depth; max_d = brace_depth
        for (ci = 1; ci <= length(ln); ci++) {
            c = substr(ln, ci, 1)
            if (c == "{") { brace_depth++; if (brace_depth > max_d) max_d = brace_depth }
            else if (c == "}") brace_depth--
        }
        # I1: PEAK open — max_d increased AND max_d >= 2 (class-body guard)
        # A one-liner void arm(){ ... } has max_d=old_d+1 >= 2 -> fires; brace_depth=old_d -> closes.
        if (!in_m && max_d > old_d && max_d >= 2) {
            mname = ""
            # Case A: single-line signature — identifier(...) [throws ...] {
            if (match(ln, /[A-Za-z_][A-Za-z0-9_<>\[\]]*[[:space:]]*\([^)]*\)[[:space:]]*(throws[^{]*)?\{/)) {
                seg = substr(ln, RSTART)
                match(seg, /^[A-Za-z_][A-Za-z0-9_<>\[\]]*/); mname = substr(seg, 1, RLENGTH)
                # I3: keyword exclusion
                if (mname ~ /^(if|for|while|switch|catch|try|else|do|new)$/) mname = ""
            }
            # Case B: { alone on line — backward scan for identifier(
            # I2: stop at annotation (@), prior statement (;$), or prior block open ({)
            # Exclude class/interface/enum — never name the class body as a method
            if (mname == "") {
                bonly = ln; gsub(/[[:space:]]/, "", bonly)
                if (bonly == "{") {
                    for (k = i-1; k >= 1 && k >= i-20; k--) {
                        lk = src[k]; lk_t = lk; gsub(/^[[:space:]]*/, "", lk_t)
                        if (substr(lk_t, 1, 1) == "@") break
                        if (match(lk, /;[[:space:]]*$/) || index(lk, "{") > 0) break
                        if (match(lk, /[A-Za-z_][A-Za-z0-9_<>\[\]]*[[:space:]]*\(/)) {
                            seg2 = substr(lk, RSTART)
                            match(seg2, /^[A-Za-z_][A-Za-z0-9_<>\[\]]*/); cn = substr(seg2, 1, RLENGTH)
                            if (cn !~ /^(if|for|while|switch|catch|try|else|do|new|class|interface|enum)$/) {
                                mname = cn; break
                            }
                        }
                    }
                }
            }
            # I4: one-liner accessor skip — (one-liner) AND mname ~ /^(get|set|is)[A-Z_]/
            # A multi-line isDirty() that schedules is untouched; only the one-liner form is skipped.
            if (mname != "" && brace_depth <= old_d && mname ~ /^(get|set|is)[A-Z_]/) mname = ""
            if (mname != "") { in_m = 1; m_st = i; m_dep = max_d; cur_name = mname }
        }
        # Close: brace_depth < m_dep, tested in SAME iteration as open (D1g).
        # Prevents runaway span: a one-liner closes on the same line it opens;
        # the class-FIELD guard (>= 2) prevents the class body from ever being entered.
        if (in_m && brace_depth < m_dep) {
            m_start[cnt] = m_st; m_end[cnt] = i; m_name[cnt] = cur_name; cnt++
            in_m = 0; cur_name = ""
        }
    }
    return cnt
}
'
