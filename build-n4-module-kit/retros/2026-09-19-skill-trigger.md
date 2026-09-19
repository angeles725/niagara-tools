<!-- review-status: folded -->
# 2026-09-19 · kit · skill-trigger

**Session**: 2026-09-19 kit-improvement campaign (branch odd/issue-116-skill) — expand the skill trigger + sync the installed copy (issue #116).
**Delta count**: 1

## What happened
The installed ~/.claude skill was stale vs the kit's skill/SKILL.md, and the trigger only named
rt-logic/ux-dashboard/wb-widget. The diff-and-warn tooling the audit asked for ALREADY existed in
scripts/install-skill.sh (sha256 compare, exit 1 on divergence, --force to sync, --dry-run to
detect). So this reduced to: expand the trigger to cover the new module classes, then sync.

## Evidence
- install-skill.sh diverged → --force synced → --dry-run "already current"; install-skill.bats 5/5; full suite 576/0; shellcheck 0.10.0 exit 0. Harness re-loaded the expanded trigger. `[ev: scripts/install-skill.sh]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | expand skill description/trigger to name driver/network, station service, utility/library, theme/branding, cloud connector (beyond rt/ux/wb) | `skill/SKILL.md` frontmatter | `[ev: retro driver-authoring]` |

## Lessons
- The install-skill.sh sha256 diff-warn (--dry-run detects, --force syncs) is the sync gate — no new tooling needed; the drift was just an un-run install.
- Editing skill/SKILL.md diverges the installed copy; run install-skill.sh --force to re-sync (the harness then reloads the trigger).

---
**Status**: FOLDED — trigger expanded + installed copy synced (2026-09-19), issue #116.
