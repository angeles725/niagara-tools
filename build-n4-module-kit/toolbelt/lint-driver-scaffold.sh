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
#   Exit: 0  no FAIL (WARN-only is 0) · 1  any FAIL · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# A method body is the text between a class-body method header's `{` and its matching `}`, with
# comments stripped (string literals are not parsed). VCS-free by design.
# Mutation: LDS2 -- drop the pingOk-only body match and the TODO match, and the wizard doPing stops failing
# Mutation: LDS3 -- drop the TODO-in-doPing match and a half-edited wizard doPing passes
# Mutation: LDS4 -- drop the submit(null) match and the wizard discovery job stops warning
# Mutation: LDS-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-driver-scaffold raw 'usage: lint-driver-scaffold.sh <src-root>' "$@"
awk_lint_list -name '*.java'

cat > "$AL_TMP/main.awk" << 'AWKEOF'
function row(sev, line, msg) { printf "%s  lint-driver-scaffold  %s:%d  %s\n", sev, FILE, line, msg }
# code of one line with /* */ and // comments removed; `inblk` carries an open block comment
function strip(s,   out, i, j) {
  out = ""
  while (s != "") {
    if (inblk) { i = index(s, "*/"); if (!i) return out; s = substr(s, i + 2); inblk = 0; continue }
    i = index(s, "/*"); j = index(s, "//")
    if (j && (!i || j < i)) return out substr(s, 1, j - 1)
    if (!i) return out s
    out = out substr(s, 1, i - 1); s = substr(s, i + 2); inblk = 1
  }
  return out
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
{
  raw = $0; code = strip(raw)
  if (raw ~ /TODO add poll support|TODO - convert linkMessage|TODO add code to detect beginning and end of message/) {
    marker = raw; sub(/^[[:space:]]+/, "", marker)
    row("WARN", FNR, "wizard stub: " marker)
  }
  if (depth == 1 && match(code, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
    pending = substr(code, RSTART, RLENGTH); sub(/[[:space:]]*\($/, "", pending); pline = FNR
  }
  if (depth >= 2) rawbody = rawbody "\n" raw
  n = length(code)
  for (k = 1; k <= n; k++) {
    c = substr(code, k, 1)
    if (c == "{") {
      depth++
      if (depth == 2) { name = pending; hline = pline; body = ""; rawbody = raw; continue }
    } else if (c == "}") {
      depth--
      if (depth == 1) { finish(); continue }
    }
    if (depth >= 2) body = body c
  }
  if (depth >= 2) body = body " "
}
AWKEOF

fail=0
while IFS= read -r f; do
    # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
    if ! out=$(awk -v FILE="$f" -f "$AL_TMP/main.awk" "$f" 2>"$AL_TMP/awk.err"); then
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
