#!/usr/bin/env bash
# None of these is an awk regex with \b: gawk word boundary \y, grep -E / sed / Python \b, a printf escape.
awk '{ if (ln !~ /\yclass\y/) print }' "$1"
grep -qE '\bB[0-9]+\b' "$1"
sed 's/\bfoo\b/bar/' "$1"
python3 -c 'import re; print(re.search(r"\bx\b", "a x"))'
printf 'a\bb\n'
awk '{ str = str "\b" }' "$1"
