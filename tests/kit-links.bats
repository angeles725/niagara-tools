#!/usr/bin/env bats
# Guards the kit's internal references and the toolbelt's threat-matrix rule.
# L1 regression: types/logic.md and types/wb-widgets.md cited checklist-common.md and type-dashboard.md
#    after those files were renamed (dangling for a whole release).
# L2 regression: design §5 — no toolbelt script may invoke git (they run inside worktrees and on stations).

setup() { KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"; }

# every kit/repo-local reference in the given files, one per line:
#   types/x.md toolbelt/x.sh retros/x.md scripts/x.sh, and bare X.md doc names (backticked or not).
# A name preceded by "/" (docs/x.md, /abs/path/x.md) is an external pointer and is ignored.
# Bare *.sh names are not collected: they refer to per-module scripts (chihuahua deploy.sh).
kit_refs() {
  grep -ohE '(types|toolbelt|retros|scripts)/[A-Za-z0-9_.-]+\.(md|sh)|(^|[^/A-Za-z0-9_.-])[A-Za-z0-9_-]+\.md' "$@" \
    | sed -E 's/^[^A-Za-z0-9_]+//' | sort -u
}

@test "L1: every kit-local file reference in kit *.md resolves (dangling checklist-common.md / type-dashboard.md)" {
  cd "$KIT"
  missing=()
  while IFS= read -r ref; do
    [ -n "$ref" ] || continue
    # SKILL.md is the launcher — it lives OUTSIDE the repo (~/.claude/skills/build-n4-module/) by
    # design, so it is a known external pointer, always OK. Skip it unconditionally; do NOT probe a
    # machine-specific $HOME path (that made L1 pass only where the launcher happened to be installed,
    # and fail in CI / any clean checkout — an environment coupling CI surfaced).
    if [ "$ref" = "SKILL.md" ]; then continue; fi
    # resolve: kit root, kit types/, kit retros/ (a bare retro name is a first-class kit citation —
    # e.g. types/logic.md:14 cites a self-firing-timer retro), or the niagara-tools repo root
    # (scripts/ng-deploy.sh). NOTE: this stays biting — a renamed or deleted retro resolves NOWHERE.
    if [ -e "$ref" ] || [ -e "types/$ref" ] || [ -e "retros/$ref" ] || [ -e "../$ref" ]; then continue; fi
    missing+=("$ref")
  done < <(kit_refs ./*.md types/*.md)
  if [ ${#missing[@]} -gt 0 ]; then printf 'dangling reference: %s\n' "${missing[@]}" >&2; return 1; fi
}

@test "L2: no toolbelt script invokes git (threat matrix: scripts run in worktrees and on stations)" {
  run grep -nE '(^|[^A-Za-z0-9_./-])git( |$)' "$KIT"/toolbelt/*.sh
  [ "$status" -eq 1 ]     # grep: no match
}

@test "L3: the launcher's default kit path exists and holds METHODOLOGY.md, BUILD-LOOP.md, types/, toolbelt/" {
  skill="$HOME/.claude/skills/build-n4-module/SKILL.md"
  [ -f "$skill" ] || skip "launcher not installed on this machine"
  def=$(grep -oE '/home/[A-Za-z0-9_./-]*build-n4-module-kit' "$skill" | head -1)
  [ -n "$def" ]
  [ -f "$def/METHODOLOGY.md" ] && [ -f "$def/BUILD-LOOP.md" ] && [ -d "$def/types" ] && [ -d "$def/toolbelt" ]
}

@test "L4: every types/*.md named in skill/SKILL.md decision table exists on disk" {
  # Regression: a new types/*.md added to the routing table must also exist on disk
  # Named mutation: add a row to the table pointing at a non-existent file -> L4 fails
  skill="$KIT/skill/SKILL.md"
  [ -f "$skill" ] || skip "skill/SKILL.md not found in kit"
  cd "$KIT"
  missing=()
  # SC2016: single-quoted $KIT is intentional — we match the literal string "$KIT/types/" in SKILL.md
  # shellcheck disable=SC2016
  while IFS= read -r ref; do
    [ -n "$ref" ] || continue
    [ -e "types/$ref" ] || missing+=("types/$ref")
  done < <(grep -oE '\$KIT/types/[A-Za-z0-9_.-]+\.md' "$skill" | sed 's|\$KIT/types/||')
  if [ ${#missing[@]} -gt 0 ]; then printf 'missing types doc: %s\n' "${missing[@]}" >&2; return 1; fi
}

@test "L5: every toolbelt/*.sh is named in BUILD-LOOP.md or skill/SKILL.md (D10 routing guard)" {
  # Regression: a new toolbelt script must appear in at least one routing doc
  # Named mutation: delete a script name from both BUILD-LOOP.md and skill/SKILL.md -> L5 fails
  [ -f "$KIT/skill/SKILL.md" ] || skip "skill/SKILL.md not found in kit (launcher path)"
  cd "$KIT"
  missing=()
  for sh in toolbelt/*.sh; do
    name=$(basename "$sh")
    if ! grep -qF "$name" BUILD-LOOP.md && ! grep -qF "$name" skill/SKILL.md; then
      missing+=("$name")
    fi
  done
  if [ ${#missing[@]} -gt 0 ]; then printf 'toolbelt script not in BUILD-LOOP.md or skill/SKILL.md: %s\n' "${missing[@]}" >&2; return 1; fi
}

@test "L7: every §6.a post-deploy step script is named in BUILD-LOOP.md (PR13 pin)" {
  # Regression guard for BUILD-LOOP.md §6.a — every post-deploy step script must stay named.
  # Named mutation: remove any script name from BUILD-LOOP.md -> L7 fails.
  cd "$KIT"
  ok=1
  for script in station-snapshot.sh triage-console.sh bog-audit.sh report-module.sh schema-risk.sh; do
    if ! grep -qF "$script" BUILD-LOOP.md; then
      echo "§6.a step script missing from BUILD-LOOP.md: $script" >&2
      ok=0
    fi
  done
  [ "$ok" -eq 1 ]
}

@test "L6: types/logic.md and types/logic-authoring.md exist and cite each other" {
  # Regression: the split creates two companion files; both must exist and point at each other
  # Named mutation: remove the cross-reference from either file -> L6 fails
  cd "$KIT"
  [ -f "types/logic.md" ] || { echo "types/logic.md missing" >&2; return 1; }
  [ -f "types/logic-authoring.md" ] || { echo "types/logic-authoring.md missing" >&2; return 1; }
  grep -q 'logic-authoring' types/logic.md \
    || { echo "types/logic.md does not cite logic-authoring.md" >&2; return 1; }
  grep -q 'logic\.md' types/logic-authoring.md \
    || { echo "types/logic-authoring.md does not cite logic.md" >&2; return 1; }
}
