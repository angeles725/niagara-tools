#!/usr/bin/env bash
# Fixture: a heredoc whose delimiter never appears; the loop below it is unreachable for the scanner.
cat <<EOF
header
while read -r line; do
  printf "%s\n" "$line" | grep -q x
done < "$1"
