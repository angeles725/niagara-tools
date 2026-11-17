#!/usr/bin/env bats
# scaffold-module: what the launcher skill and BUILD-LOOP § 1.a tell a newcomer matches what the kit has and
# what the scaffold emits. [ev: retro build-kit-bench-session Δ17] [ev: retro build-kit-bench-session Δ18]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SKILL="$KIT/skill/SKILL.md"
}

# section <file> <heading-prefix>: the lines from the heading that starts with the prefix to the next heading
# of the same or a higher level.
section() {
  awk -v h="$2" '
    index($0, h) == 1 { on = 1; split($0, m, " "); depth = length(m[1]); print; next }
    on && /^#+ / { split($0, m, " "); if (length(m[1]) <= depth) exit }
    on { print }' "$1"
}

@test "TC-DOC1: every \$KIT/<path> skill/SKILL.md names exists in the kit" {
  # Mutation: TC-DOC1 -- naming $KIT/scripts/install-skill.sh again (the kit has no scripts/ directory).
  local p paths missing=()
  # shellcheck disable=SC2016  # the literal text $KIT/ of SKILL.md, not the variable
  paths="$(grep -oE '\$KIT/[A-Za-z0-9_./-]+' "$SKILL" | sed -E 's|^\$KIT/||; s|[.]$||' | grep -v '[<*]' | sort -u)"
  [ -n "$paths" ]
  while IFS= read -r p; do
    [ -e "$KIT/$p" ] || missing+=("$p")
  done <<<"$paths"
  [ "${#missing[@]}" -eq 0 ] || { printf 'missing: %s\n' "${missing[@]}"; false; }
}

@test "TC-DOC2: skill/SKILL.md holds no per-user absolute path" {
  # Mutation: TC-DOC2 -- a /home/<user>/ path in the resolver line.
  run grep -nE '/home/[a-z]|/Users/[A-Za-z]|C:\\Users' "$SKILL"
  [ "$status" -eq 1 ] || { echo "$output"; false; }
}

@test "TC-DOC3: the decision table maps every row to a scaffold --type or 'no scaffold', and offers each type" {
  # Mutation: TC-DOC3 -- a decision-table row without its Scaffold cell.
  local table rows bad
  table="$(section "$SKILL" "## Decision Gates" | grep -E '^\| ')"
  [[ "$(head -1 <<<"$table")" == "| Building | Read | Scaffold \`--type\` |" ]] || { head -1 <<<"$table"; false; }
  rows="$(tail -n +3 <<<"$table")"
  [ "$(wc -l <<<"$rows")" -ge 10 ]
  bad="$(awk -F'|' '{ c = $4; gsub(/^ +| +$/, "", c); if (c !~ /^(`(logic|dashboard|driver)`( \+ `(logic|dashboard|driver)`)*|no scaffold)( \(.*\))?$/) print }' <<<"$rows")"
  [ -z "$bad" ] || { echo "$bad"; false; }
  for t in logic dashboard driver; do
    grep -qF "\`$t\`" <<<"$rows" || { echo "no row maps to $t"; false; }
  done
}

@test "TC-DOC4: skill/SKILL.md describes scaffold-module.sh once in the Toolbelt, by its own job" {
  # Mutation: TC-DOC4 -- the old "run in step 5 before lint-timers" description borrowed from lint-delays.
  run grep -c 'toolbelt/scaffold-module.sh` (' "$SKILL"
  [ "$output" = "1" ]
  run grep -oE 'toolbelt/scaffold-module\.sh` \([^)]*\)' "$SKILL"
  [[ "$output" == *"--vendor"* && "$output" != *"lint-timers"* ]] || { echo "$output"; false; }
}

@test "TC-DOC5: BUILD-LOOP § 1.a matches the scaffold output: no chmod, --vendor, signing, the gradle root, commit after the first build" {
  # Mutation: TC-DOC5 -- the old step 1 "chmod +x gradlew" (the emitted gradlew is already 0755).
  local s
  s="$(section "$KIT/BUILD-LOOP.md" "### 1.a")"
  [ -n "$s" ]
  [[ "$s" != *"chmod"* ]] || { echo "$s"; false; }
  for want in '--vendor' 'niagaraSigningAlias' 'niagaraSigningProfile' '<out-dir>/<ModuleName>' 'settingsPluginVersion' 'commit'; do
    [[ "$s" == *"$want"* ]] || { echo "§ 1.a does not say: $want"; false; }
  done
}

@test "TC-DOC6: BUILD-LOOP § 4 names a generic gradle root, not a client path" {
  # Mutation: TC-DOC6 -- the example root Cliente/<site>/<client> back in the build.sh argument line.
  run bash -c "awk '/^## 4\\. /{on=1; next} on && /^## /{exit} on' '$KIT/BUILD-LOOP.md' | grep -nE 'Cliente/[A-Z][A-Za-z-]+/'"
  [ "$status" -eq 1 ] || { echo "$output"; false; }
}

@test "TC-DOC7: the emitted gradlew is executable, as BUILD-LOOP § 1.a now assumes" {
  # Mutation: TC-DOC7 -- a scaffold that drops the execute bit would need the old chmod step back.
  local kit="$KIT" out="$BATS_TEST_TMPDIR/out"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime
  . "$kit/../go/go-tool.sh"
  go_tool_bin "$kit/../go" scaffold-module >/dev/null
  run "$kit/toolbelt/scaffold-module.sh" ModuleA "$out" --vendor Example
  [ "$status" -eq 0 ]
  [ -x "$out/ModuleA/gradlew" ]
}
