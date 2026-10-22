#!/usr/bin/env bash
# lib/sha256.sh — shared sha256 helper for install-skill.sh and check-skill-drift.sh.
# Sourced only (never invoked directly). Linux has sha256sum, macOS has shasum -a 256.
# VCS-free by design, like the scripts that source it.
#
# sha256_require <prefix> — returns 0 when sha256sum or shasum is on PATH; otherwise prints
#   "<prefix>: no sha256 tool found (sha256sum or shasum)" on stderr and returns 3.
#   Call it BEFORE any `$(sha256_of ...)`: an exit inside $( ) cannot stop the caller.
# sha256_of <file> — prints the hex digest of <file>; prints nothing and returns 3 with no tool.

sha256_require() {
  if command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1; then return 0; fi
  printf '%s: no sha256 tool found (sha256sum or shasum)\n' "$1" >&2
  return 3
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    return 3
  fi
}
