<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · skillinstall-t10

**Session**: row T10 of the 2026-11-15 coordination-stretch retro: the four non-blocking findings the native
review of niagara-tools #694 left on install-skill.
**Delta count**: 2

## What happened
One finding was a defect: a diverged prompt rendered from the built-in default template named the kit's
`prompt.md`, a file that does not exist. The other three were untested behaviour: the dry-run digest labels, the
refusal of a whole `--harness all` run before any harness is written, and an empty `--home` whose test only
checked `$HOME`.

## Evidence
- RED before GREEN: the diverged default prompt named `<kit>/skill/prompt.md`.
  `[ev: go/internal/skillinstall/review694_template_test.go]`
- Mutations in a scratch copy, not committed: the dry-run label swapped, the pre-write checks made per harness
  (only the new cross-harness test fails), and a write to the working directory on `--home ""` each turn their
  test red. `[ev: go/internal/skillinstall/review694_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. R3-diverged-template-names-missing-file: a report names the template renderPrompt read, the built-in default when the kit has no prompt.md (`promptLabel`) | Go `internal/skillinstall` `prompt.go`, `target.go` | `[ev: go/internal/skillinstall/prompt.go]` |
| Δ2 | APPLIED 2026-11-16. R3-dry-run-rendered-label-untested, R3-cross-harness-atomicity-untested, R3-empty-home-nothing-written-assertion-weakened: one test each | Go `internal/skillinstall` `review694_test.go`, `followups_paths_test.go` | `[ev: go/internal/skillinstall/review694_test.go]` |
