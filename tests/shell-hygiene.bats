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
  run python3 "$CHECK" "$ROOT"/scripts/*.sh "$ROOT"/scripts/lib/*.sh "$ROOT"/build-n4-module-kit/toolbelt/*.sh \
    "$ROOT"/tests/helpers/*.bash
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

# SH2: a script that takes `--home <dir>` reads or writes the Claude skills directory of a home directory.
# A bats case that runs one without `--home` or a `HOME=` override on the same command works on the real
# home of whoever runs the suite. [ev: retro go-install-tools-port Δ3]
#
# _home_scripts <root>: the basenames of the scripts/*.sh that take --home (the option is the marker).
_home_scripts() { grep -l -e '--home' "$1"/scripts/*.sh | sed 's|.*/||' | tr '\n' ' '; }

# _home_lint <names> <bats-file>...: one "<file>:<line>: ..." row per command that runs a named script with
# neither --home nor HOME=; exit 1 when there is any. The rules and their accepted limits: tests/lib/home-lint.awk.
_home_lint() {
  local names="$1"; shift
  awk -v names="$names" -f "$BATS_TEST_DIRNAME/lib/home-lint.awk" "$@"
}

@test "SH2: no bats case runs a --home script without --home or an overridden HOME" {
  names="$(_home_scripts "$ROOT")"
  [[ " $names" == *" install-skill.sh "* ]]      # non-vacuous: the marker still finds the installer
  run _home_lint "$names" "$ROOT"/tests/*.bats
  [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; return 1; }
  [ -z "$output" ]
}

@test "SH2-neg: the lint flags a path, a variable and a continued command without --home (file:line named)" {
  # Mutation: SH2-neg -- dropping the variable lookup in script_of reports the "$SCRIPT" runs as clean.
  # The fixture spells its cases %test: a line starting with the real keyword is read by bats as a case of THIS file.
  sed 's/^%test /@test /' > "$TMPDIR_T/bad.bats" <<'SH'
SCRIPT="$REPO/scripts/install-skill.sh"
%test "a" { _close; run "$REPO/scripts/install-skill.sh" --dry-run; [ "$status" -eq 0 ]; }
%test "b" {
  run bash "$SCRIPT" --dry-run
  run env FOO=1 /bin/bash "${SCRIPT}" \
    --force
  out="$("$REPO/scripts/check-skill-drift.sh" 2>&1)"
}
SH
  run _home_lint "install-skill.sh check-skill-drift.sh" "$TMPDIR_T/bad.bats"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 4 ]
  [[ "${lines[0]}" == *"bad.bats:2: install-skill.sh runs without --home or HOME="* ]]
  [[ "${lines[1]}" == *"bad.bats:4: install-skill.sh "* ]]
  [[ "${lines[2]}" == *"bad.bats:5: install-skill.sh "* ]]
  [[ "${lines[3]}" == *"bad.bats:7: check-skill-drift.sh "* ]]
}

@test "SH2-pos: --home, a HOME= override, a continued --home and a script that is only read stay clean" {
  sed 's/^%test /@test /' > "$TMPDIR_T/good.bats" <<'SH'
SCRIPT="$REPO/scripts/install-skill.sh"
# run "$SCRIPT" --dry-run   (a comment is not a command)
_install() { "$REPO/scripts/install-skill.sh" --home "$H" "$@" >/dev/null; }
%test "a" {
  run bash "$SCRIPT" --home "$BATS_TEST_TMPDIR/home" --dry-run
  run env HOME=/nonexistent bash "$SCRIPT" --dry-run
  run "$SCRIPT" --dry-run \
    --home "$H"
  cp "$SCRIPT" "$FAKE/scripts/"
  grep -q SKILLS "$REPO/scripts/install-skill.sh"
  [ -f "$SCRIPT" ]
  run "$REPO/scripts/install-hooks.sh"
}
SH
  run _home_lint "install-skill.sh check-skill-drift.sh" "$TMPDIR_T/good.bats"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SH2-forms: every form of tests/lib/home-lint-forms.tsv lints as its row says (run, clean or limit)" {
  # Mutation: SH2-forms -- dropping a launcher (env, exec, time, timeout, if, elif, while, until), the ; ( $( rule,
  # the $( ) rule, the comment strip (or its quote handling) or the assignment rule fails a row.
  local expect form i=0 fails=""
  while IFS=$'\t' read -r -u 3 expect form; do
    [[ -z "$expect" || "$expect" == \#* ]] && continue
    form="${form//\\t/$'\t'}"; form="${form//\\n/$'\n'}"; i=$((i + 1))
    printf '%s\n%s\n' "SCRIPT=\"\$REPO/scripts/install-skill.sh\"" "$form" > "$TMPDIR_T/f$i.bats"
    run _home_lint "install-skill.sh check-skill-drift.sh" "$TMPDIR_T/f$i.bats"
    if [ "$expect" = run ]; then
      [ "$status" -eq 1 ] && [[ "$output" == *"f$i.bats:2: install-skill.sh runs without --home"* ]] && continue
    elif [ "$status" -eq 0 ] && [ -z "$output" ]; then continue; fi
    fails+="row $i ($expect): $form -> status $status: $output"$'\n'
  done 3< "$BATS_TEST_DIRNAME/lib/home-lint-forms.tsv"
  [ "$i" -ge 44 ]                                          # non-vacuous: the verifier's forms are all read
  [ -z "$fails" ] || { printf '%s' "$fails" >&2; return 1; }
}

@test "SH2-marker: the --home marker finds every script that takes --home, not only the installer" {
  # Mutation: SH2-marker -- a marker that finds install-skill.sh alone leaves check-skill-drift.sh runs unlinted.
  names="$(_home_scripts "$ROOT")"
  [[ " $names" == *" install-skill.sh "* ]]
  [[ " $names" == *" check-skill-drift.sh "* ]]
}
