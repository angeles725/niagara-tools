#!/usr/bin/env bash
# A loop the scanner never sees close: the self-check must report it (FPL3).
while IFS= read -r line; do
  printf '%s\n' "$line"
