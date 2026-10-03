#!/usr/bin/env bash
# lint-spa-poll-no-recovery.sh — flags an rc/ SPA poll loop (setInterval/setTimeout) whose catch
# path has no recovery keyed on time since the last SUCCESS: either nothing ever reloads after the
# underlying session dies (a station restart kills the web session; the kiosk polls the dead one
# forever), or the reload is driven by a failure COUNT / a bare reload-on-error, which is blind to a
# hung request that neither fails nor succeeds.
#
# Shape 1 (no recovery, 2.4.2): `catch` only marks data stale and repaints "error"; a fixed HMI
# panel stays on dead values until power-cycled. [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1]
# [ev: corpus DashboardPan-ux index.html poll() 2.4.2 -> 2.4.3]
# Shape 2 (failure-count watchdog, 2.4.3..2.8.0): reload after N consecutive failures; recovered ten
# restarts and froze on the eleventh because a fetch without timeout hung and the counter never
# advanced. Doctrine: types/dashboard.md § Poll loop reliability (watchdog on `lastOkAt`, set only by
# a validated success; probe via fetchT; reload on the app marker; hard ceiling).
# [ev: retro dashboard-frontend-reliability-rules Δ3]
#
# Usage:  lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>
#   Scans *.html and *.js under <ux-src-root> (dot-dirs pruned). `//` comments line-stripped. Per file:
#     1. the FIRST `setInterval(<name>, ...)` / `setTimeout(<name>, ...)` call names the poll fn;
#     2. its declaration body -- `function <name>(`, or `const|let|var <name> = [async] function(`
#        / an arrow function -- is brace-extracted; it must contain `catch (`/
#        `.catch(` — no catch, no shape to flag;
#     3. recovery = `location.reload(` or `location.href =` ANYWHERE in the file;
#     4. success-time key = a `lastOk*` identifier assigned from a clock (`Date.now()`,
#        `performance.now()`, `new Date`) INSIDE the poll fn body (the success path; a timestamp
#        set only at load is not a success time) AND read in a subtraction/comparison
#        (`- lastOkAt`, `lastOkAt <`/`>`) ANYWHERE in the file.
#   Rows (WARN at the setInterval/setTimeout call line):
#     no 3        -> "... no location.reload(/location.href recovery anywhere in the file ..."
#     3 but no 4  -> "... recovery is not keyed on time since the last success ..."
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
#
# Known limitations (advisory heuristic): file-wide co-occurrence (a reload button elsewhere plus a
# lastOk* timestamp suppresses the WARN — false-NEGATIVE bias); only the FIRST interval/timeout call
# per file is checked; a success timestamp not named lastOk* (e.g. `lastSuccessAt`) is a false
# positive — rename it or accept the WARN; a success timestamp set in a helper the poll calls
# (`.then(onOk)`) is not seen either — set it in the poll body; naive `//` stripping can cut a `//`
# inside a string/URL.
# VCS-free by design (kit-links L2). LC_ALL=C.
# Mutation: SPR2 -- drop the lastOk success-time requirement so a failure-count-only watchdog passes clean
# Mutation: SPR8 -- drop the location.reload(/location.href recovery check so a lastOk timestamp that never reloads passes clean
# Mutation: SPR9 -- count a clock assignment anywhere in the file so a lastOk set only at load passes clean
# Mutation: SPR10 -- matching only `function <name>(` loses an arrow-function poll (silent pass)
# Mutation: SPR-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: SPR-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-spa-poll-no-recovery: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-spa-poll-no-recovery: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  for (i = 1; i <= NR; i++) { ln = lines[i]; sub(/\/\/.*$/, "", ln); lines[i] = ln }

  # 1. First setInterval(<name>, ...) or setTimeout(<name>, ...) call.
  fn = ""; call_line = 0
  for (i = 1; i <= NR; i++) {
    if (match(lines[i], /(setInterval|setTimeout)\([[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*,/)) {
      seg = substr(lines[i], RSTART, RLENGTH)
      match(seg, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*,$/)
      fn = substr(seg, RSTART, RLENGTH)
      gsub(/[[:space:]]*,$/, "", fn)
      call_line = i
      break
    }
  }
  if (fn == "") exit 0

  # 2. Locate the function declaration and brace-extract its body forward.
  decl_line = 0
  for (i = 1; i <= NR; i++) {
    # `function poll(` / `async function poll(`, or a binding `const|let|var poll = [async]
    # function(...)` / `(...) =>` / `x =>` (an arrow-function poll is the common SPA form).
    if (match(lines[i], "function[[:space:]]+" fn "[[:space:]]*\\(") ||
        match(lines[i], "(const|let|var)[[:space:]]+" fn "[[:space:]]*=[[:space:]]*(async[[:space:]]*)?(function[[:space:]]*\\(|\\([^)]*\\)[[:space:]]*=>|[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=>)")) {
      decl_line = i; break
    }
  }
  if (decl_line == 0) exit 0

  depth = 0; started = 0; body_end = NR
  for (i = decl_line; i <= NR; i++) {
    ln = lines[i]
    for (ci = 1; ci <= length(ln); ci++) {
      c = substr(ln, ci, 1)
      if (c == "{") { depth++; started = 1 }
      else if (c == "}") depth--
    }
    if (started && depth == 0) { body_end = i; break }
  }

  # 2b. The body must contain a catch clause -- otherwise this is not the flagged shape.
  has_catch = 0
  for (i = decl_line; i <= body_end; i++) {
    if (match(lines[i], /catch[[:space:]]*\(/)) { has_catch = 1; break }
  }
  if (!has_catch) exit 0

  # 3. Whole-file recovery signal.
  has_recovery = 0
  for (i = 1; i <= NR; i++) {
    if (index(lines[i], "location.reload(") > 0 || match(lines[i], /location\.href[[:space:]]*=/)) {
      has_recovery = 1; break
    }
  }
  if (!has_recovery) {
    printf "WARN  lint-spa-poll-no-recovery  %s:%d  setInterval/setTimeout poll loop '%s()' has a" \
      " catch path with no location.reload(/location.href recovery anywhere in the file --" \
      " kiosk/HMI will stay blank after a station restart; add a watchdog on time since the last" \
      " success (lastOkAt) -> fetchT probe of the app marker -> location.reload()\n", FILE, call_line, fn
    exit 0
  }

  # 4. Success-time key: lastOk* assigned from a clock inside the poll body (the success path;
  #    a load-time initializer alone never advances) AND read in a subtraction/comparison anywhere.
  ok_set = 0; ok_read = 0
  for (i = 1; i <= NR; i++) {
    if (i >= decl_line && i <= body_end && match(lines[i], /lastOk[A-Za-z0-9_]*[[:space:]]*=[[:space:]]*(\+[[:space:]]*)?(Date\.now\(|performance\.now\(|new[[:space:]]+Date)/)) ok_set = 1
    if (match(lines[i], /-[[:space:]]*lastOk[A-Za-z0-9_]*/) || match(lines[i], /lastOk[A-Za-z0-9_]*[[:space:]]*[<>]/)) ok_read = 1
  }
  if (ok_set && ok_read) exit 0

  printf "WARN  lint-spa-poll-no-recovery  %s:%d  setInterval/setTimeout poll loop '%s()' reloads," \
    " but the recovery is not keyed on time since the last success (no lastOk* timestamp set from" \
    " a clock in the poll body and compared) -- a failure-count or reload-on-error watchdog is blind to a hung" \
    " request; key it on lastOkAt (types/dashboard.md Poll loop reliability)\n", FILE, call_line, fn
}
AWKEOF

had_warn=0
had_err=0
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.html' -o -name '*.js'; then
  printf 'lint-spa-poll-no-recovery: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  had_err=1
fi
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -f "$_TMP/main.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-spa-poll-no-recovery: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
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
