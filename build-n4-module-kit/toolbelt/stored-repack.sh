#!/usr/bin/env bash
# stored-repack.sh — repackage a signed N4 module jar with every entry STORED (no compression).
#
# WHY: Workbench's JarFileSigner re-deflates each entry while re-signing; when the JDK that built
# the jar (WSL OpenJDK 8) and the Workbench JDK (Windows Zulu 8) deflate to different sizes it aborts
# with "ZipException: invalid entry compressed size (expected N but got M)". A clean rebuild does NOT
# fix it (same deflater, same size) and a local `jarsigner` cannot detect it (same deflater again).
# STORED entries make the mismatch impossible by construction and keep the content bytes identical,
# so the existing signature stays valid. Field-confirmed (operator re-signed STORED jars, 2026-09-01).
#
# Entry order matters for a valid signed jar: META-INF/MANIFEST.MF first, then NIAGARA4.SF/.RSA
# when present, then everything else. Cost: a bigger jar (deflate ratio lost).
#
# Usage: stored-repack.sh <in.jar> <out.jar>      (refuses to overwrite an existing out.jar)
# Check: verify-module.sh --stored <out.jar>      (0 "Defl:" entries)
set -euo pipefail

usage() { sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'; }
[ $# -eq 2 ] || { usage >&2; exit 2; }
IN="$1"; OUT="$2"
[ -f "$IN" ] || { echo "stored-repack: input jar not found: $IN" >&2; exit 1; }
[ ! -e "$OUT" ] || { echo "stored-repack: refusing to overwrite existing $OUT" >&2; exit 1; }
for t in unzip zip; do command -v "$t" >/dev/null || { echo "stored-repack: missing tool: $t" >&2; exit 2; }; done

OUT_ABS="$(cd "$(dirname "$OUT")" && pwd)/$(basename "$OUT")"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
unzip -q "$IN" -d "$TMP/ex" || { echo "stored-repack: cannot unzip $IN" >&2; exit 1; }
cd "$TMP/ex"
[ -f META-INF/MANIFEST.MF ] || { echo "stored-repack: no META-INF/MANIFEST.MF in $IN (not a jar)" >&2; exit 1; }

# -0 store only, -X no extra attrs, -D no directory entries
zip -q -0 -X -D "$OUT_ABS" META-INF/MANIFEST.MF
SIG=()
for f in META-INF/NIAGARA4.SF META-INF/NIAGARA4.RSA; do [ -f "$f" ] && SIG+=("$f"); done
[ ${#SIG[@]} -eq 0 ] || zip -q -0 -X -D "$OUT_ABS" "${SIG[@]}"
zip -q -0 -X -D -r "$OUT_ABS" . -x META-INF/MANIFEST.MF META-INF/NIAGARA4.SF META-INF/NIAGARA4.RSA

d=$(unzip -v "$OUT_ABS" | grep -c 'Defl:' || true)
[ "$d" -eq 0 ] || { echo "stored-repack: $d deflated entries remain in $OUT_ABS" >&2; exit 1; }
n=$(unzip -Z1 "$OUT_ABS" | wc -l)
echo "stored-repack: $OUT_ABS  ($n entries, all STORED, signature entries kept: ${SIG[*]:-none})"
