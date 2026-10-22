#!/usr/bin/env bash
# lint-driver-scaffold.sh — flags NDriver wizard stubs left in shipped driver sources.
#
# The devkit NDriver wizard generates `public void doPing() { // TODO - add ping implementation  pingOk(); }`
# (devkit-wb.jar!gradle/ndriver/BNfooDevice.java.vm:102-111): left as is, every device reports UP
# forever, so a dead device never goes DOWN (silent success). That is a FAIL. The wizard's other
# placeholders compile and run but do nothing useful, so they WARN. See types/driver-authoring.md § 5.
# [ev: retro n4-course-handson-ndriver Δ2] [ev: retro n4-course-handson-ndriver Δ6]
#
# Usage:  lint-driver-scaffold.sh <src-root>     (scans *.java under it)
#   Row:  FAIL  lint-driver-scaffold  <file>:<line>  doPing() only calls pingOk() | doPing() still holds the wizard TODO
#   Row:  WARN  lint-driver-scaffold  <file>:<line>  wizard stub: <marker> | <method>() only returns null
#                                                    | doSubmitDiscoveryJob() submits with a null Context
#                                                    | method boundaries not trusted (<reason>)
#   Exit: 0  no FAIL (WARN-only is 0) · 1  any FAIL · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# A method body is the text between a member header's `{` and its matching `}`, at any class depth (a nested
# device class counts). The header is the code since the last `;`, `{` or `}`, annotations dropped, so
# `@SuppressWarnings({"x"}) public void doPing()` and a `throws` line still name doPing. The text is read
# after lib/method-boundary.sh `mb_strip` (comments out, literal text kept) with literal contents blanked,
# so a "tcp://" or "/*" string, a '{' char or a text block holding braces cannot move a boundary. A file
# whose braces do not balance, or that ends inside a block comment, a text block or an unterminated
# "..."/'...' literal, gets one WARN row: its boundaries are not trusted, so a stub in it may be missed.
# The path reaches awk through ENVIRON, so a backslash in it is printed as is. VCS-free by design.
# [ev: retro r8-276-driver-scaffold Δ2]
# Mutation: LDS2 -- drop the pingOk-only body match and the TODO match, and the wizard doPing stops failing
# Mutation: LDS3 -- drop the TODO-in-doPing match and a half-edited wizard doPing passes
# Mutation: LDS4 -- drop the submit(null) match and the wizard discovery job stops warning
# Mutation: LDS6 -- stop blanking literal contents and the '{' char and the text block move the boundaries
# Mutation: LDS7 -- drop the END balance check and the truncated file reports clean
# Mutation: LDS9 -- name the method by the leftmost identifier( without dropping annotations and the annotated stubs stop failing
# Mutation: LDS10 -- pass the path with awk -v and its \n becomes a newline in the row
# Mutation: LDS-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/method-boundary.sh"
awk_lint_parse lint-driver-scaffold raw 'usage: lint-driver-scaffold.sh <src-root>' "$@"
awk_lint_list -name '*.java'

printf '%s\n' "$MB_AWK" > "$AL_TMP/method-boundary.awk"
cat > "$AL_TMP/main.awk" << 'AWKEOF'
function row(sev, line, msg) { printf "%s  lint-driver-scaffold  %s:%d  %s\n", sev, FILE, line, msg }
# dst = src with the contents of "...", '...' and """...""" literals blanked (quotes kept, \ escapes
# honoured, text-block state carried across lines), so a { } ; ( or "pingOk();" inside one is never code.
# mb_strip keeps literal text for its other consumers; this lint matches code only. Sets `unterm` for a
# "..." or '...' still open at its line end (mb_strip ends it there).
function blank(src, n, dst,   i, ln, out, j, c, q, tb) {
  for (i = 1; i <= n; i++) {
    ln = src[i]; out = ""; q = ""
    for (j = 1; j <= length(ln); j++) {
      c = substr(ln, j, 1)
      if (tb) { if (c == "\\") j++; else if (substr(ln, j, 3) == "\"\"\"") { out = out "\"\"\""; tb = 0; j += 2 } }
      else if (q != "") { if (c == "\\") j++; else if (c == q) { out = out q; q = "" } }
      else if (substr(ln, j, 3) == "\"\"\"") { out = out "\"\"\""; tb = 1; j += 2 }
      else { if (c == "\"" || c == "\047") q = c; out = out c }
    }
    if (q != "") unterm = 1
    dst[i] = out
  }
}
# method name of a member header, or "" for a class, an initializer, a control block or a `new X() {` body
function head(h,   nm) {
  if (!match(h, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) return ""
  nm = substr(h, RSTART, RLENGTH); sub(/[[:space:]]*\($/, "", nm)
  if (nm ~ /^(if|for|while|switch|catch|try|else|do|synchronized|return|new)$/) return ""
  if (substr(h, 1, RSTART - 1) ~ /(^|[^A-Za-z0-9_])new[[:space:]]+$/) return ""
  return nm
}
function finish(   norm) {
  norm = body; gsub(/[[:space:]]/, "", norm)
  if (name == "doPing") {
    if (norm == "pingOk();" || norm == "this.pingOk();")
      row("FAIL", hline, "doPing() only calls pingOk(): every device reports UP (wizard stub, silent success)")
    else if (rawbody ~ /TODO - add ping implementation/)
      row("FAIL", hline, "doPing() still holds the wizard TODO: replace it with a real reachability check")
  }
  if (norm == "returnnull;" && FILE ~ /MessageFactory[^\/]*\.java$/)
    row("WARN", hline, name "() only returns null (wizard message factory)")
  if (name == "doSubmitDiscoveryJob" && norm ~ /\.submit\(null\)/)
    row("WARN", hline, "doSubmitDiscoveryJob() submits with a null Context: use job.submit(cx)")
}
{ raw[NR] = $0 }
END {
  FILE = ENVIRON["LDS_FILE"]
  open = mb_strip(raw, NR, code); blank(code, NR, code)
  # depth: braces open; mdep: depth of the method being read (0 outside one); hdr: header text since the
  # last ; { or } outside a method, with pline the line of its first ( — the row line of the method
  for (i = 1; i <= NR; i++) {
    if (raw[i] ~ /TODO add poll support|TODO - convert linkMessage|TODO add code to detect beginning and end of message/) {
      marker = raw[i]; sub(/^[[:space:]]+/, "", marker)
      row("WARN", i, "wizard stub: " marker)
    }
    if (mdep) rawbody = rawbody "\n" raw[i]
    ln = code[i]
    if (!mdep) gsub(/@[A-Za-z_][A-Za-z0-9_.]*[[:space:]]*(\([^)]*\))?/, " ", ln)   # annotations name nothing
    n = length(ln)
    for (k = 1; k <= n; k++) {
      c = substr(ln, k, 1)
      if (mdep) {
        if (c == "{") depth++
        else if (c == "}" && --depth < mdep) { finish(); mdep = 0; hdr = ""; pline = 0; continue }
        if (mdep) body = body c
      } else if (c == "{") {
        depth++; name = head(hdr)
        if (name != "" && depth >= 2) { mdep = depth; hline = pline; body = ""; rawbody = raw[i] }
        hdr = ""; pline = 0
      } else if (c == "}" || c == ";") {
        if (c == "}" && --depth < 0) { depth = 0; negative = 1 }
        hdr = ""; pline = 0
      } else { hdr = hdr c; if (c == "(" && !pline) pline = i }
    }
    if (mdep) body = body " "; else hdr = hdr " "
  }
  # an open comment or literal is named first: it is why the braces after it look unbalanced
  why = ""
  if (open == 1) why = "ends inside a block comment"
  else if (open == 2) why = "ends inside a text block"
  else if (unterm) why = "an unterminated string or char literal"
  else if (depth != 0 || negative) why = "braces do not balance"
  if (why != "") row("WARN", NR, "method boundaries not trusted (" why "): a stub in this file may be missed")
}
AWKEOF

fail=0
while IFS= read -r f; do
    # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
    if ! out=$(LDS_FILE="$f" awk -f "$AL_TMP/method-boundary.awk" -f "$AL_TMP/main.awk" "$f" 2>"$AL_TMP/awk.err"); then
        printf '%s: cannot scan %s: %s\n' "$AL_NAME" "$f" "$(head -n 1 "$AL_TMP/awk.err")" >&2
        # shellcheck disable=SC2034  # read by awk_lint_exit (lib/awk-lint.sh)
        AL_ERR=1
        continue
    fi
    [ -n "$out" ] || continue
    printf '%s\n' "$out"
    if grep -q '^FAIL ' <<< "$out"; then fail=1; fi
done < "$AL_TMP/files"

awk_lint_exit "$fail"
