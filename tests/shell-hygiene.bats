#!/usr/bin/env bats
# shell-hygiene.bats — the kit's own shell hygiene (deferred-lints D1).
# SH1: bash 5.2 `patsub_replacement` expands an unquoted `&` in a pattern-substitution replacement
# (or in the value of an unquoted `$rep`) to the matched text, so `${v//pat/$rep}` garbles a value
# that contains `&`. Every replacement that carries an expansion or `&` must be quoted.
# [ev: retro polish-2026-10-02-close Δ3]
# The bats `! cmd` pitfall (Δ4) is gated by shellcheck SC2314, which CI runs at every severity.

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  CHECK="$ROOT/tests/helpers/patsub-check.py"
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SH1: no kit script has an unquoted \$ or & in a pattern-substitution replacement" {
  run python3 "$CHECK" "$ROOT"/scripts/*.sh "$ROOT"/build-n4-module-kit/toolbelt/*.sh \
    "$ROOT"/build-n4-module-kit/toolbelt/lib/*.sh "$ROOT"/tests/helpers/*.bash
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SH1-neg: the checker flags an unquoted \$rep or & replacement (file:line named)" {
  # Mutation: SH1-neg -- skipping the replacement part reports the unquoted replacements as clean.
  cat > "$TMPDIR_T/bad.sh" <<'SH'
out=${out//"(map)"/"(map: x)"}
out=${out//"(map)"/(map: $MAP)}
out=${out/x/&}
SH
  run python3 "$CHECK" "$TMPDIR_T/bad.sh"
  [ "$status" -eq 1 ]
  [[ "$output" == *"bad.sh:2:"* ]]
  [[ "$output" == *"bad.sh:3:"* ]]
  if [[ "$output" == *"bad.sh:1:"* ]]; then return 1; fi
}

@test "SH1-pos: quoted replacements, a quoted \$'…' pattern and a literal replacement stay clean" {
  cat > "$TMPDIR_T/good.sh" <<'SH'
v="${v//"$bs"/"$bs$bs"}"
v="${v//$'\n'/"${bs}n"}"
p=${p//.//}
x="${SKILL_NAMES//|/ or }"
# a comment ${v//a/$b} is not code
SH
  run python3 "$CHECK" "$TMPDIR_T/good.sh"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
