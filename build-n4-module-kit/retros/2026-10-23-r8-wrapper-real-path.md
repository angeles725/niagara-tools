<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · r8-wrapper-real-path

**Session**: niagara-research quality round 8, branch `fix/r8-wrapper-real-path` from origin/main b62e40c (rebased onto 4449766)
**Delta count**: 1

## What happened
The six Go-backed toolbelt wrappers (check-facets-card, gen-build-state, gen-lint-index, gen-retro-index,
lint-guard-pins, sweep-fold-audit) found `<repo>/go` as `$here/../../go`, with `here` taken from the unresolved
`${BASH_SOURCE[0]}`. report-module's tests run their members from a toolbelt copy built by
`tests/helpers/stub-toolbelt.bash`, which symlinks every toolbelt FILE one by one into a temp dir. Through that
copy `here` is the temp dir, `../../go` does not exist, and the wrapper exits 3 with "go-tool.sh missing". RM40,
RM41, RM-floor-schema and RM-wp-nomatrix broke that way when a lint became a Go wrapper. Every Go wrapper now
derives its dir from its real path, with this exact line:

    here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"

Why this idiom: the copy's directory is real and only the file is a symlink, so `pwd -P` or `cd -P` resolves
nothing. Only a file-level resolution reaches the real kit. `readlink -f` needs GNU coreutils, which the r8
script-dir retro used as a reason to reject it for every script. Here it is limited to the Go wrappers, which
need `go` on PATH anyway, and CI runs on ubuntu-latest. Non-Go scripts keep the dirname form, because
report-module itself must resolve TOOLBELT to the copy so that its stubs are used. go/go-tool.sh needs no change:
`go_tool_bin` takes the already-absolute GO_DIR. The lookup happens before go-tool.sh is sourced, so the fix
belongs in the wrapper.

## Evidence
- RED on b62e40c: `bats tests/go-wrapper-symlink.bats`. GWS1 (check-facets-card through the symlinked copy from an
  unrelated cwd) failed with exit 3. GWS2 named all six wrappers. GWS3 failed `[ev: tests/go-wrapper-symlink.bats GWS1]`
- GREEN after one sed on the six `here=` lines: 4/4 `[ev: tests/go-wrapper-symlink.bats]`
- Mutation: reverting lint-guard-pins.sh:31 to the unresolved form fails GWS2 (it names `lint-guard-pins.sh`) and
  GWS3. GWS1 stays green, because it pins check-facets-card only `[ev: tests/go-wrapper-symlink.bats GWS3]`
- Host crash during the first mutation run (WSL pids limit). A static read found no recursion. The wrappers exec
  only their Go binary, the binaries exec only `diff`, and every bad flag is a fast usage exit 3. The re-run under
  `ulimit -u 2000` plus `timeout 120` finished in seconds. GWS2 now wraps each wrapper in `timeout 60` and names a
  hang `(timeout)` `[ev: tests/go-wrapper-symlink.bats GWS2]`
- The first full ci-local failed `tests/gen-retro-index.bats` GRI-go and GRI-nogo-stale. Their fake PATH linked
  only `bash dirname` (and `find head`), so the wrapper printed `readlink: command not found` and "no Go module"
  instead of "go is not on PATH". Both lists now link `readlink`. No other fake-PATH bats runs a Go wrapper:
  go-tool.bats sources go-tool.sh directly, and retro-loop RL4 runs kit-ticket.sh `[ev: tests/gen-retro-index.bats GRI-go]`
- gen-lint-index.sh is tracked 100644, so a direct exec through the copy fails with exit 126. GWS2 runs every
  wrapper as `bash <path>`. A separate commit sets the mode to 100755.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A toolbelt script that sources go/go-tool.sh derives `here=` with `readlink -f "${BASH_SOURCE[0]}"`; GWS3 ratchets it | `tests/go-wrapper-symlink.bats` § GWS3 | `[ev: tests/go-wrapper-symlink.bats GWS3]` |

## Lessons
- A symlinked copy made file by file defeats `pwd -P`. Resolve the file (`readlink -f`), not the directory.
- A script that locates siblings must say which root it wants. The Go module is in the real repo, but
  report-module's stubs are in the invoked copy, so the two cannot share one idiom.
- A sweep over every member must invoke each one through `bash <path>` and under `timeout`. That way a file-mode
  slip or a hang is named instead of hiding the result or stalling the host.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-r8-wrapper-real-path.md | kit | 2026-10-23 | pending | 1 |`
