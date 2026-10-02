#!/usr/bin/env bash
# lint-manual-labels.sh — every quoted UI label in an operator manual exists in the UI source (advisory).
#
# A manual is code that nobody compiles: when a UI label is renamed and the manual still quotes
# the old one, nothing fails. This lint takes every quoted label of the manual — "…", “…” and
# «…» (HTML entities &quot; &ldquo; &rdquo; &laquo; &raquo; decoded; tags and their attribute
# values dropped, so href="x.html" is never a label) — and looks for it in the UI source:
# *.html *.htm *.js *.mjs *.cjs *.json *.lexicon *.properties *.java *.xml under each <ui-dir>
# (dot-dirs and node_modules pruned; \uNNNN escapes and HTML entities decoded). The label must
# EQUAL one whole UI text unit — an HTML text node, a string literal, or a lexicon/properties
# value — or be its prefix ending at a word boundary ("Rearranque" matches "Rearranque 1",
# "Guardado" matches "Guardado (" + n + ")"), ignoring case, accents, collapsed whitespace and a
# trailing ':'. A mid-unit substring is not a match: a manual quoting "Corte" after the UI
# renamed it "Temp. de corte" is the drift to catch.
# A label not found is a WARN — labels may be paraphrased on purpose, so a human reads each row.
#
# Usage:  lint-manual-labels.sh [--strict] <manual.html|manual.md> <ui-dir> [<ui-dir>...]
#   <ui-dir> is an rc/ tree, a -ux/-rt profile src dir or a lexicon dir (several allowed).
# Exits:  0 clean or WARN-only · 1 any WARN under --strict · 3 usage/env (no python3, bad manual or ui-dir)
#
# Row format:
#   WARN  lint-manual-labels  <manual>:<line>  label "<label>" not found in the UI source
#   SUMMARY  lint-manual-labels  checked=<n> missing=<m>
#   (ML10 pins these prefixes against the emitted rows.)
# VCS-free by design; version control is never invoked (kit-links L2).
# [ev: retro operator-manual-lockstep Δ3]
# Mutation: ML2 -- dropping the not-found check stops the WARN on a renamed label the manual still quotes
# Mutation: ML4 -- dropping the \uNNNN decode makes an accented lexicon label look missing
# Mutation: ML8 -- dropping the word-boundary test lets "Guard" match the unit "Guardar"
set -u

usage() {
  printf 'usage: lint-manual-labels.sh [--strict] <manual.html|manual.md> <ui-dir> [<ui-dir>...]\n' >&2
  exit 3
}

STRICT=0
case "${1:-}" in --strict) STRICT=1; shift ;; esac
[ $# -ge 2 ] || usage
MANUAL="$1"; shift
[ -f "$MANUAL" ] || { printf 'lint-manual-labels: manual not found: %s\n' "$MANUAL" >&2; exit 3; }
for d in "$@"; do
  [ -d "$d" ] || { printf 'lint-manual-labels: not a directory: %s\n' "$d" >&2; exit 3; }
done
command -v python3 >/dev/null 2>&1 || { printf 'lint-manual-labels: missing tool: python3\n' >&2; exit 3; }

MANUAL="$MANUAL" python3 - "$@" <<'PYEOF'
import html, os, re, sys, unicodedata

manual = os.environ["MANUAL"]
dirs = sys.argv[1:]
EXTS = (".html", ".htm", ".js", ".mjs", ".cjs", ".json", ".lexicon", ".properties", ".java", ".xml")
UESC = re.compile(r"\\u([0-9a-fA-F]{4})")

UNITS = (
    re.compile(r">([^<>]+)<"),                                   # HTML text node
    re.compile(r'"((?:[^"\\\n]|\\.)*)"|\'((?:[^\'\\\n]|\\.)*)\'|`([^`]*)`'),  # string literal
    re.compile(r"(?m)^\s*[^#!=\s][^=\n]*=(.*)$"),                  # lexicon / properties value
)

def norm(s):
    s = "".join(c for c in unicodedata.normalize("NFKD", s) if not unicodedata.combining(c))
    return re.sub(r"\s+", " ", s).strip().rstrip(":").strip().casefold()

def found(label, units):
    if label in units:
        return True
    n = len(label)
    return any(u.startswith(label) and not u[n].isalnum() for u in units if len(u) > n)

def decode(s):
    return html.unescape(UESC.sub(lambda m: chr(int(m.group(1), 16)), s))

hay = set()
for root in dirs:
    for cur, subdirs, files in os.walk(root):
        subdirs[:] = sorted(d for d in subdirs if not d.startswith(".") and d != "node_modules")
        for f in sorted(files):
            if f.lower().endswith(EXTS):
                with open(os.path.join(cur, f), encoding="utf-8", errors="replace") as fh:
                    body = decode(fh.read())
                for rx in UNITS:
                    for m in rx.finditer(body):
                        hay.update(norm(g) for g in m.groups() if g)

with open(manual, encoding="utf-8", errors="replace") as fh:
    text = fh.read()
keep_lines = lambda m: "\n" * m.group(0).count("\n")
if manual.lower().endswith((".html", ".htm")):
    text = re.sub(r"(?is)<(script|style)\b.*?</\1\s*>", keep_lines, text)
    text = re.sub(r"(?s)<!--.*?-->", keep_lines, text)
    text = re.sub(r"(?s)<[^>]*>", keep_lines, text)
    text = html.unescape(text)

LABEL = re.compile(r'"([^"\n]{2,60})"|“([^”\n]{2,60})”|«([^»\n]{2,60})»')
checked = missing = 0
for lineno, line in enumerate(text.split("\n"), 1):
    for m in LABEL.finditer(line):
        label = next(g for g in m.groups() if g is not None).strip()
        if not re.search(r"[^\W\d_]", label) or "://" in label:
            continue
        checked += 1
        if not found(norm(label), hay):
            missing += 1
            print('WARN  lint-manual-labels  %s:%d  label "%s" not found in the UI source' % (manual, lineno, label))
print("SUMMARY  lint-manual-labels  checked=%d missing=%d" % (checked, missing))
# Exit 10 is a private sentinel for "rows printed, some label missing": it lets the shell
# wrapper tell a WARN result (0, or 1 under --strict) from a crashed helper (any other code -> 3).
sys.exit(10 if missing else 0)
PYEOF
rc=$?
case "$rc" in
  0) exit 0 ;;
  10) [ "$STRICT" -eq 1 ] && exit 1; exit 0 ;;
  *) printf 'lint-manual-labels: python3 helper failed (exit %d)\n' "$rc" >&2; exit 3 ;;
esac
