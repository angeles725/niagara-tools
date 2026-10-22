#!/usr/bin/env bash
# Planted per-line forks: every marked line must be reported (FPL1).
while IFS= read -r line; do
  printf '%s' "$line" | grep -q foo && echo hit
  echo "$line" | sed 's/a/b/'
done < "$1"
while read -r f; do
  while IFS='|' read -r name rest; do
    out=$(printf '%s\n' "${name}" | awk '{print $1}')
    grep -q x <<< "$rest" && :
    printf '%s' "$f" | grep -q y
  done < "$f"
done < "$2"
printf '%s\n' "$out"
