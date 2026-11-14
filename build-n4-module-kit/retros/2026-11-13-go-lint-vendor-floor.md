<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-13 · kit · go-lint-vendor-floor

**Session**: "all tool logic in Go" pass: `lint-vendor-floor`, the last lint whose logic lived in bash, ported to Go.
**Delta count**: 2

## What happened
`toolbelt/lint-vendor-floor.sh` embedded a node + acorn driver in a heredoc and built its rows in bash. It is now
`go/internal/vendorfloor` behind a thin wrapper: Go owns arguments, tool resolution, the walk, the API table, row
classification, output and exit codes; one embedded `.js` file only parses and tokenizes. node and acorn stay an
external tool, as gradle and javac are, so `go/go.mod` keeps no dependency. The differential kept two paths where
the lint passes without having looked, and the CI Go job turned out to skip every acorn test.

## Evidence
- Differential against the bash script, 53 runs (bats cases, missing node, missing acorn, real `rc/vendor` trees):
  52 identical. The one difference: `KIT_NODE` at a directory was a bash error with exit 126 and is now the SKIP
  row with exit 4. `[ev: 60100c6d]`
- A node that exits without the driver's final record is an env fault (exit 3); bash passed node's exit code
  through, so `KIT_NODE=/bin/true` was a silent exit 0. `[ev: go/internal/vendorfloor/driver.go]`
- A symlinked `vendor/` directory and an unreadable sub-directory both exit 0, as before.
  `[ev: go/internal/vendorfloor/walk.go]`
- The CI Go job installed neither node nor acorn: the seven acorn tests of the package skipped there.
  `[ev: 71cd910c]`
- The bats case `VF-disjoint` edited the bash source with `sed`, so it could not survive the port; the rule is
  pinned by `TestSharedMethodNameIsAnEnvFault` and `TestDefaultTableIsDisjoint`, seen failing with the startup
  check removed. `[ev: a4ea352b]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `lint-vendor-floor` fails closed on its walk: a `vendor/` that is a symlink is followed like the other lint roots (`find -H`), and an unreadable sub-directory is an env fault (exit 3) instead of a pass; RED tests for both | `toolbelt/lint-vendor-floor.sh` (Go `internal/vendorfloor/walk.go`) · `BUILD-LOOP.md` § 5 Fail-closed input listing | `[ev: go/internal/vendorfloor/walk.go]` |
| Δ2 | A test that mutates a script's source text to prove a guard is a finding: the guard is tested through the command's interface with the data injected | `CONTRIBUTING.md` tests · `BUILD-LOOP.md` § 5 Kit tooling doctrine | `[ev: a4ea352b]` |

## Lessons
- A Go test that skips on a missing external tool needs a CI switch that turns the skip into a failure, or the job
  goes green having tested nothing.
- An external parser is a tool, not a dependency: keep only parse and tokenize in its language and every verdict
  in ours.
- A wrapper that passes a child's exit code through inherits the child's silent successes.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-13-go-lint-vendor-floor.md | kit | 2026-11-13 | pending | 2 |`
