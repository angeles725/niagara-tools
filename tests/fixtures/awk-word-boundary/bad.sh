#!/usr/bin/env bash
# Every awk regex below uses \b, which gawk reads as a BACKSPACE (0x08), never a word boundary.
awk '{ if (ln !~ /\bclass\b/) print }' "$1"
awk '$0 ~ /x\b/ { n++ }' "$1"
awk '{ gsub(/\bfoo/, "bar"); print }' "$1"
awk '{ if (match($0, /\breturn[^;]*[<>]/)) print }' "$1"
awk '
  /^\s*\bstatic/ { s++ }
' "$1"
