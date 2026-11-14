<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-14 · kit · go-pre-push-port

**Session**: "all tool logic in Go" pass, wave 2: `.githooks/pre-push` (195 lines of bash) becomes a launcher over `go/cmd/pre-push`.
**Delta count**: 4

## What happened
The push gate was ported to Go with its decision table written down first and a differential of 83 rows against
the original hook (75 direct, 8 through `git push --dry-run`): 80 identical, 3 differing in one diagnostic line
with the same decision. Each gate has a mutation proof. One inherited fail-open path was closed in the port (a
client denylist that cannot be examined now refuses); four gaps stay open.

## Evidence
- The bash hook counted any failed `test -e` on the denylist as "no denylist" and went on unchecked; the port
  refuses unless the error is "no such file" (`TestNoClientGuardWithAnUnexaminableDenylistRefuses`, RED first).
  `[ev: c55c4084]`
- When `git rev-parse --show-toplevel` fails the hook exits 0 before any gate, in bash and in Go
  (`go/internal/prepush/run.go`, "Outside a work tree there is no push to gate"). A push from a bare repository
  or with a broken git is therefore never gated. `[ev: 0b88574f]`
- `git diff` rename detection makes a deleted pending retro plus a similar new folded retro count as the
  same-file header flip the promotion exit asks for, in both implementations. `[ev: 0b88574f]`
- `scripts/lib/live-doctrine-set.sh` has no runtime consumer any more: the two patterns are Go data pinned
  equal to the shell file by a test. `[ev: 337512d4]`
- Comments in `scripts/changelog-gate.sh`, `tests/build-retro-sync.bats`, `tests/changelog-gate.bats` and the
  LC8 title still describe the hook as bash or name `pass_early`. `[ev: 337512d4]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A failing `git rev-parse --show-toplevel` refuses the push with the git error, except in a bare repository, which is named in a one-line notice; both cases are tested | `go/internal/prepush/run.go` · `CONTRIBUTING.md` push gates | `[ev: 0b88574f]` |
| Δ2 | The promotion exit compares the header flip with rename detection off (`--no-renames`), so only a flip inside the same path counts; RED test: delete a pending retro, add a similar folded one | `go/internal/prepush/retro.go` (promotion) | `[ev: 0b88574f]` |
| Δ3 | The live-doctrine set has one home: delete `scripts/lib/live-doctrine-set.sh` and its pin test once no shell caller remains, keeping the Go data | `scripts/lib/live-doctrine-set.sh` · `go/internal/prepush/paths.go` | `[ev: 337512d4]` APPLIED |
| Δ4 | Comments and test titles that describe the hook as bash (`pass_early`, "the bash hook") are reworded to the Go gate | `scripts/changelog-gate.sh` · `tests/build-retro-sync.bats` · `tests/changelog-gate.bats` | `[ev: 337512d4]` |

## Lessons
- Write the decision table before porting a gate; the table is what the differential and the mutation proofs
  are checked against.
- "Could not check" must refuse. Every `test -e`, `stat` and `git` call in a gate needs an answer for "failed
  for another reason".
- `git push --dry-run` runs the pre-push hook, so a hook can be tested through real git without touching a remote.
- A hook that needs a toolchain must refuse when the toolchain is missing, and its tests must run with a PATH
  that lacks it.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-14-go-pre-push-port.md | kit | 2026-11-14 | pending | 4 |`
