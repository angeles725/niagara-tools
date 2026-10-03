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

@test "SH1-multiline: a replacement that continues on the next line is still checked (D1b)" {
  # Mutation: SH1-multiline -- scanning one line at a time treats the split expansion as clean.
  cat > "$TMPDIR_T/split.sh" <<'SH'
out=${out//"(map)"/(map:
$MAP)}
SH
  run python3 "$CHECK" "$TMPDIR_T/split.sh"
  [ "$status" -eq 1 ]
  [[ "$output" == *"split.sh:1: unquoted replacement"* ]]
}

@test "SH1-unterminated: an expansion that never closes is reported, never passed as clean (D1b)" {
  # Mutation: SH1-unterminated -- treating an unclosed expansion as clean passes it silently.
  cat > "$TMPDIR_T/open.sh" <<'SH'
out=${out//a/"b"
SH
  run python3 "$CHECK" "$TMPDIR_T/open.sh"
  [ "$status" -eq 1 ]
  [[ "$output" == *"open.sh:1:"*"unterminated"* ]]
}

@test "SH1-squote: a backslash inside single quotes is literal, so the unquoted replacement after it is flagged (D1b)" {
  # Mutation: SH1-squote -- treating the backslash as an escape inside '...' swallows the closing quote and hides the hit.
  cat > "$TMPDIR_T/sq.sh" <<'SH'
x=${x//'\'/$y}
SH
  run python3 "$CHECK" "$TMPDIR_T/sq.sh"
  [ "$status" -eq 1 ]
  [[ "$output" == *"sq.sh:1: unquoted replacement"* ]]
}
