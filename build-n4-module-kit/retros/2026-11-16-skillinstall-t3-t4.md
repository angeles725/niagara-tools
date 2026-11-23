<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-11-16 · kit · skillinstall-t3-t4

**Session**: rows T3 and T4 of the 2026-11-15 coordination-stretch retro: the seven native-review findings and
the four independent-verifier points on niagara-tools #692, re-checked against origin/main.
**Delta count**: 1

## What happened
Every finding already had a fix or a pinning test from #694 (2d0f29fe) and T10 (ad8e00fe, 5224ef7c); the package
tests pass on origin/main. One part of T4 (3), "the install is not atomic across SKILL.md and prompt", still
reproduced: the pre-write plan refused diverged copies and symbolic links, but a `prompts` path that is a regular
file (or a target that is a directory) failed only at its write, after the harness's SKILL.md was installed.

## Evidence
- Reproduced on origin/main in a scratch home: `install-skill: installed .../skills/mcp-n4/SKILL.md` then
  `cannot create target directory: .../prompts`, exit 3, SKILL.md left behind.
- RED before GREEN: `TestUnplaceableTargetRefusesBeforeAnyWrite` (three shapes, with and without `--force`, under
  `--harness all`) and bats HS8 fail on the base code. `[ev: go/internal/skillinstall/placeable_test.go]`
- Differential of 48 install-skill runs (both kit skills, every harness, fresh and agent homes; dry-run, real,
  diverged, dry-run diverged and `--force` for mcp-n4) between origin/main and this branch: identical.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. T4 (3) residual: the pre-write plan also refuses a file that cannot be placed (`unplaceable`: a non-directory on its path, a target that is not a regular file), exit 3, nothing written | Go `internal/skillinstall` `target.go`, `plan.go`; `scripts/install-skill.sh` header | `[ev: go/internal/skillinstall/target.go]` |

## Lessons
- A "nothing written" plan must check every condition a write can fail on that is visible beforehand, not only
  the policy refusals; otherwise the first I/O error still leaves half an install.
