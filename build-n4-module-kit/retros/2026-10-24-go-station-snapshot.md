<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-station-snapshot

**Session**: niagara-research quality round 10, branch `feat/go-station-snapshot` from origin/main 1a086dd1
**Delta count**: 3

## What happened
`toolbelt/station-snapshot.sh` (180 lines of bash) is now the thin `go/go-tool.sh` wrapper over
`go/cmd/station-snapshot` (`go/internal/stationsnapshot`). The port keeps every observable of the bash: the
`cp -p` + `chmod 0644` copy of config.bog and console*.txt (mtimes kept, outputs never +x), the PASS/FAIL rows,
history/** as size pointers walked as `find -P` did (dot-dirs pruned unread, symlinks neither followed nor
listed, C byte sort), the unescaped relpaths of manifest.json, the out-dir-inside-station guard on the logical
(`cd -L`) paths, including its blind spot for a relative out dir whose parent does not exist yet, and the
messages the shell, cp, mkdir and find printed on a failure (`STATION_SNAPSHOT_SCRIPT` carries the wrapper's
path for the `<script>: line N:` prefix). The UTC stamp comes from `Host.Now`. No bats runs the script under a
restricted PATH, so no bats file changed.

## Evidence
- 58 goldens replayed by the shared `golden.Runner`: 57 recorded from the bash at 4d602a9b by
  `golden/record/record.sh` with a fake `date` in the case tree, 1 truth-* case
  `[ev: go/internal/stationsnapshot/golden_test.go TestGolden]`
- The truth case: the bash with an unwritable TMPDIR lost every manifest entry and still exited 0; the port keeps
  no temp files `[ev: go/internal/stationsnapshot/testdata/truth-tmpdir-unused.txt]`
- RED (stub Command) then GREEN 58/58 byte-identical; dropping the dot-dir prune fails edge-history-nested and
  edge-history-unreadable-dotdir `[ev: go/internal/stationsnapshot/testdata/edge-history-nested.txt]`
- station-snapshot.bats SN1-SN5 and kit-links.bats 18/18 ok through the wrapper `[ev: tests/station-snapshot.bats SN5]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A tool that stamps the clock is recorded with a fake `date` in the case tree printing the stamp the Go test injects** (`Host.Now`), so the manifest is a plain recorded golden, not a truth case | `go/internal/lint/rules/golden/record/README.md` § Case set | `[ev: go/internal/stationsnapshot/testdata/setup.sh]` |
| Δ2 | **A file name holding a tab cannot be a recorded tree entry**: record.sh's snapshot index is tab-separated and loses the entry; keep such names out of cases or fix the index first | `go/internal/lint/rules/golden/record/README.md` § Recording | `[ev: go/internal/lint/rules/golden/record/record.sh snap]` |
| Δ3 | **A truth-* case is the place for a bash defect the port drops** (here a silent empty manifest on a bad TMPDIR): state the bash behaviour in its @kind so the fix is visible, not just the new output | `BUILD-LOOP.md` § 5 What to port | `[ev: go/internal/stationsnapshot/testdata/truth-tmpdir-unused.txt]` |

## Follow-ups (not done here)
- GNU cp quotes a path holding a quote or a control character in shell-escape style; the port prints the locale
  style. No station path holds one.
- Not reproduced (no golden): a directory at `<out-dir>/config.bog` (cp copied into it), a dangling or
  same-file symlink at the destination, a file name holding a newline (the bash read loop split it), the order
  of several find errors (find's readdir order), a `CDPATH` exported into the script, and a `sha256sum` on PATH
  that is not executable.

## Lessons
- Record from a copy of the script inside the case dir: the shell's `<script>: line N:` messages then carry a
  path the recorder rewrites to @DIR@ instead of a random temp path.
- Walking every failure branch of a small bash tool turns up behaviours nobody meant (exit 0 with an empty
  manifest); a truth case is cheaper than arguing about them later.

## Fold
- The port is code already shipped in the PR that wrote this retro (`go/internal/stationsnapshot`); not re-implemented.
- Δ1 and Δ2 → `BUILD-LOOP.md` § 5 golden-case bullet (fake `date` printing the `Host.Now` stamp; a tab in a file name is not recordable: `record.sh` `snap` keys its index and sort on tabs, checked).
- Δ3 → `BUILD-LOOP.md` § 5 "Port byte-identical first" bullet (a bash defect the port cannot reproduce is a truth case whose `@kind` states it).
- The follow-ups (GNU cp quoting, the unreproduced shapes) are recorded limits with no real station path; no issue.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail3`; log in `ledger.d/2026-10-25-r10-fold-tail3.md`) — Δ1, Δ2 in the `BUILD-LOOP.md` § 5 golden-case bullet; Δ3 in the § 5 port bullet.
