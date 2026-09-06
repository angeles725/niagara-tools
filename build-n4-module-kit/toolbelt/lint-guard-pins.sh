#!/usr/bin/env bash
# lint-guard-pins.sh — Guard-pin meta-check for the kit's own lint scripts (C11 T4, B833).
# Scope: toolbelt/lint-*.sh only (D4b — non-lint toolbelt scripts are out of T4's scan).
#
# Every lint-*.sh must declare at least one header line (lines 1-60):
#   # Mutation: <fixture-id> -- <what it flips>
# Each declared id must resolve to an @test "<id>:" or "@test "<id> " entry in tests/*.bats.
# A lint with no declaration is a latent unpinned guard — the defect C11 kept catching by hand
# (S23-and, EW-s22-nondo, vacuous WP-stale decoys). [ev: design build-n4-module-campaign11 §D4]
#
# Mutation: T4-nomutation -- removing this line makes lint-guard-pins.sh itself a vacuously-clean guard
#
# Row format:
#   MATCH  lint-guard-pins  <script>:<line>  mutation <id> -> tests/<batsfile>.bats:<line>
#   WARN   lint-guard-pins  <script>:<line>  mutation <id> -> no fixture
#   WARN   lint-guard-pins  <script>  no # Mutation: line (K24(7): add '# Mutation: <fixture-id> -- <what it flips>')
#
# Usage:  lint-guard-pins.sh [--strict] <kit-root>
# Exits:  0 clean or WARN-only (no --strict) · 1 any WARN under --strict · 3 usage/env (K20)
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2). LC_ALL=C.
# [ev: retro campaign11-lint-guard-pins]
set -u
LC_ALL=C
export LC_ALL

STRICT=0
case "${1:-}" in --strict) STRICT=1; shift ;; esac

usage_exit() {
  printf 'usage: lint-guard-pins.sh [--strict] <kit-root>\n' >&2
  exit 3
}

[ $# -eq 1 ] || usage_exit
ROOT="$1"

# Resolve toolbelt directory (D4b scope: lint-*.sh only)
if [ -d "$ROOT/toolbelt" ]; then
  TOOLBELT="$ROOT/toolbelt"
elif [ -d "$ROOT/build-n4-module-kit/toolbelt" ]; then
  TOOLBELT="$ROOT/build-n4-module-kit/toolbelt"
else
  printf 'lint-guard-pins: no toolbelt/ found under %s\n' "$ROOT" >&2
  exit 3
fi

# Resolve tests directory
TESTS_DIR="$ROOT/tests"
if [ ! -d "$TESTS_DIR" ]; then
  printf 'lint-guard-pins: no tests/ found under %s\n' "$ROOT" >&2
  exit 3
fi

WARNED=0

# Scan lint-*.sh scripts only; prune dot-dirs (D9b); sorted for determinism
while IFS= read -r script; do
  has_mutation=0
  lineno=0

  # Read first 60 lines only (header region, D4c)
  while IFS= read -r hline; do
    lineno=$((lineno + 1))
    # Declaration ERE: ^# Mutation: [A-Za-z][A-Za-z0-9_-]* -- .+$  (ASCII --, no em dash)
    if printf '%s\n' "$hline" | grep -qE '^# Mutation: [A-Za-z][A-Za-z0-9_-]* -- .+$'; then
      has_mutation=1
      mid=$(printf '%s\n' "$hline" | sed -E 's/^# Mutation: ([A-Za-z][A-Za-z0-9_-]*) -- .*/\1/')
      # Locate fixture: @test "<id>:" or "@test "<id> " in tests/*.bats (-H forces filename even for 1 file)
      fixture_hit=$(grep -HnoE "^@test \"${mid}[: ]" "$TESTS_DIR"/*.bats 2>/dev/null | head -1)
      if [ -n "$fixture_hit" ]; then
        fixture_path=$(printf '%s\n' "$fixture_hit" | cut -d: -f1)
        fixture_line=$(printf '%s\n' "$fixture_hit" | cut -d: -f2)
        fixture_rel="${fixture_path#"${ROOT}"/}"
        printf 'MATCH  lint-guard-pins  %s:%d  mutation %s -> %s:%s\n' \
          "$script" "$lineno" "$mid" "$fixture_rel" "$fixture_line"
      else
        printf 'WARN  lint-guard-pins  %s:%d  mutation %s -> no fixture\n' \
          "$script" "$lineno" "$mid"
        WARNED=1
      fi
    fi
  done < <(head -60 "$script")

  if [ "$has_mutation" -eq 0 ]; then
    printf "WARN  lint-guard-pins  %s  no # Mutation: line (K24(7): add '# Mutation: <fixture-id> -- <what it flips>')\n" \
      "$script"
    WARNED=1
  fi
done < <(find "$TOOLBELT" -maxdepth 1 -name 'lint-*.sh' ! -name '.*' | LC_ALL=C sort)

[ "$STRICT" -eq 1 ] && [ "$WARNED" -eq 1 ] && exit 1
exit 0
