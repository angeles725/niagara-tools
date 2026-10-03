#!/usr/bin/env bash
# lint-license-isoperational-gate.sh — flags a LICENSED class (getLicenseFeature() returns a feature) whose
# changed() / action (timer) / servlet-write callback acts without an isOperational() / isFault() gate.
#
# The framework's license check is advisory: BAbstractService.checkLicense() sets a fatal fault and logs SEVERE on an
# unlicensed feature but does not throw, and ServiceManager runs serviceStarted() right after with no fault check, so
# a feature.check() in serviceStarted() only guards STARTUP. Later callbacks keep firing under a fault icon unless each
# one gates on isOperational() (= !isFatalFault() && !isDisabled() && !isFault()), the way the shipped services do by
# convention. See types/security.md §5.1. [ev: retro secure-authoring-isoperational-gate Δ1]
# [ev: corpus B1143] [ev: corpus B1145] [ev: corpus B1146]
# [ev: code javax/baja/sys/BAbstractService.java:87-90,148-150,152-190] [ev: code com/tridium/sys/service/ServiceManager.java:297-298]
#
# Structural rules (comments blanked by lib/method-boundary.sh mb_strip; methods from mb_parse):
#   Licensed  — the file declares `Feature getLicenseFeature()` and its body is not just `return null;` (the
#               BAbstractService default). A file with no declaration is out of scope.
#   Callback  — a method named `changed` (the property callback), `do<Upper>…` (an action: Clock.schedule timers and
#               operator actions both land in one) or a servlet write handler doPost/doPut/doDelete/doPatch; the
#               read handlers doGet/doHead/doOptions/doTrace are excluded.
#   Acts      — the callback body has a statement other than a `super.<name>(…);` call or a bare `return;`.
#   Gated     — the callback body calls isOperational(, isFault( or isFatalFault(. A gate in a helper the callback
#               calls is NOT followed (one row the reviewer clears by moving the gate up — WARN, never FAIL).
#   Overlap   — none with lint-status-parity (config/status slot ratio) or lint-silent-protection (a protection trip
#               with no operator surface): neither reads getLicenseFeature() or the operational gate.
#
# Usage:  lint-license-isoperational-gate.sh [--strict] <src-root>
#   Row:  WARN  lint-license-isoperational-gate  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: LIG1 -- dropping the gate check WARNs a callback that returns early on !isOperational()
# Mutation: LIG2 -- treating `return null;` as a license feature WARNs an unlicensed service
# Mutation: LIG3 -- counting a super.changed() call as acting WARNs a callback that only delegates
# Mutation: LIG-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: LIG-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
LC_ALL=C
export LC_ALL

USAGE='usage: lint-license-isoperational-gate.sh [--strict] <src-root>'
STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-license-isoperational-gate: unknown flag: %s\n%s\n' "$1" "$USAGE" >&2; exit 3 ;;
    *) break ;;
  esac
done
if [ $# -ne 1 ]; then
  printf '%s\n' "$USAGE" >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-license-isoperational-gate: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"

cat > "$_TMP/main.awk" << 'AWKEOF'
# body text of the brace block that opens on line `from` (from the first `{` to its matching `}`)
function block_body(from, n,    i, j, ch, d, started, out) {
  d = 0; started = 0; out = ""
  for (i = from; i <= n; i++) {
    for (j = 1; j <= length(code[i]); j++) {
      ch = substr(code[i], j, 1)
      if (ch == "{") { d++; if (!started) { started = 1; continue } }
      else if (ch == "}") { d--; if (started && d == 0) return out }
      if (started) out = out ch
    }
    if (started) out = out "\n"
  }
  return out
}
{ raw[NR] = $0 }
END {
  n = NR
  mb_strip(raw, n, code)
  lic = 0
  for (i = 1; i <= n; i++) {
    if (code[i] ~ /getLicenseFeature[[:space:]]*\([[:space:]]*\)[[:space:]]*;/) continue  # abstract / interface
    if (code[i] ~ /Feature[[:space:]]+getLicenseFeature[[:space:]]*\([[:space:]]*\)/) {
      b = block_body(i, n); gsub(/[[:space:]]/, "", b)
      if (b != "" && b != "returnnull;") lic = 1
    }
  }
  if (!lic) exit 0
  cnt = mb_parse(code, n, ms, me, mn)
  for (k = 0; k < cnt; k++) {
    m = mn[k]
    if (m != "changed" && m !~ /^do[A-Z]/) continue
    if (m ~ /^do(Get|Head|Options|Trace)$/) continue
    b = block_body(ms[k], n)
    if (b ~ /isOperational[[:space:]]*\(|isFault[[:space:]]*\(|isFatalFault[[:space:]]*\(/) continue
    # acts? drop super.<name>(...); and bare return; then look for any remaining statement text
    t = b
    gsub(/super\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([^;]*\)[[:space:]]*;/, "", t)
    gsub(/return[[:space:]]*;/, "", t)
    gsub(/[[:space:]]/, "", t)
    if (t == "") continue
    printf "WARN  lint-license-isoperational-gate  %s:%d  licensed class: %s() acts with no isOperational()/isFault() gate -- the license fault does not stop callbacks; add `if (!isOperational()) return;` (types/security.md §5.1)\n", FILE, ms[k], m
  }
}
AWKEOF

had_warn=0
had_err=0
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.java'; then
  printf 'lint-license-isoperational-gate: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  had_err=1
fi
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -f "$_TMP/method-boundary.awk" -f "$_TMP/main.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-license-isoperational-gate: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
    had_err=1
    continue
  fi
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    had_warn=1
  fi
done < "$_TMP/files"

[ "$had_err" -eq 0 ] || exit 3
if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
