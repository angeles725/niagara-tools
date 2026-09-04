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
    # resolve: kit root, kit types/, kit retros/ (a bare retro name is a first-class
    # kit citation — e.g. types/logic.md:14 cites a self-firing-timer retro), the
    # niagara-tools repo root (scripts/ng-deploy.sh), or the launcher dir (SKILL.md
    # lives outside the repo, in ~/.claude/skills/build-n4-module/).
    # NOTE: this stays biting — a renamed or deleted retro resolves NOWHERE and still fails.
    if [ -e "$ref" ] || [ -e "types/$ref" ] || [ -e "retros/$ref" ] || [ -e "../$ref" ] \
       || [ -e "$HOME/.claude/skills/build-n4-module/$ref" ]; then continue; fi
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
