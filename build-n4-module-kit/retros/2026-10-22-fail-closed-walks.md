<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · fail-closed-walks

**Session**: niagara-research quality round 8, work unit T8c, branch `r8/t8c-fail-closed-walks` from origin/main e1fee85
**Delta count**: 1

## What happened
Fourteen toolbelt scripts still walked their root with a raw `find ... | sort` (or an unsorted `find`) read
through a process substitution or a pipe. find's exit status was never seen, so a sub-directory find could not
enter was skipped and the script reported the module clean (exit 0): the issue #226 fail-open shape that
`lib/scan-files.sh` closed for the other lints. This unit routes all fourteen through the shared libs:

- Three comment-stripped grep lints (no-md5-credential-digest, subscribe-without-unsubscribe, wb-file-chooser) carried
  the same ~40-line `--strict` parse, `sed 's://.*::'` loop, trimmed-source WARN row and exit tail, and walked
  unsorted. `lib/awk-lint.sh` gained `awk_lint_strip_run <check-fn> <find-tests>`: the lib strips comments into
  `AL_TEXT`, walks fail closed, prints the row and the exit; each lint keeps only its grep check.
- Eleven others (delays, demand-scope, ext-writable-shape, recovery-path, status-parity, wb-threading,
  link-target-flags, lexicon-ascii, generate-wiring-map, schema-risk, rc-scan) call the new
  `scan_files_or_exit <prefix> <code>` from `lib/scan-files.sh` once and read that list. rc-scan's local walker is
  gone; its `rc/ext` path prune needed `scan_files --prune-path <pattern>`. schema-risk keeps its own env code (4).

## Evidence
- RED: `bats tests/awk-lint.bats tests/scan-files.bats` AL17-AL20 + SF7 + SF8 failing before the lib change, 28/28 after;
  each of the 14 lints got a `<ID>-locked` case (unreadable sub-directory -> exit 3, schema-risk exit 4), all 14 failing
  on origin/main (exit 0) and passing after `[ev: tests/scan-files.bats]`
- Byte-identity: 13 lints x 82 argument shapes + 10 schema-risk pairs, old (`git archive origin/main`) vs new, 3228 files a
  side. Every diff is one of four intended classes: an unreadable sub-directory (now `<name>: cannot list every file under
  <root>: <reason>`, exit 3/4); an unreadable file in the three strip lints (now `<name>: cannot scan <file>: <reason>`,
  exit 3, was a silent clean pass); a dot-named root such as `.` (was pruned whole and passed clean, or hit the "no Java
  sources" ERROR; now walked, as scan_files already does for every other lint); and the strip lints' row order on a
  multi-directory tree (unsorted find -> LC_ALL=C sorted, 12 shapes, sorted contents identical) `[ev: toolbelt/lib/scan-files.sh]`
- The 16 touched bats files: 205 ok `[ev: tests/awk-lint.bats]`
- Scripts: callers +103/-147, libs +59/-4 `[ev: toolbelt/lib/awk-lint.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A toolbelt script never feeds a loop from a raw `find`; it lists its files once with `scan_files_or_exit` (or `awk_lint_list` / `awk_lint_run` / `awk_lint_strip_run`).** Why: a `find` read through `< <(...)` or a pipe hides find's exit status, so an unreadable sub-directory is a clean pass; fourteen scripts still had that shape after issue #226. Reviewer check: `grep -n 'find "' toolbelt/*.sh` outside lib/ names a walk that must move to the lib (slot-coverage, lint-structure, lint-servlet, lint-timers and verify-module are owned by T9). | `BUILD-LOOP.md` § Lint doctrine (with retros awk-lint-skeleton Δ1 and awk-lint-line-loops Δ1) | `[ev: toolbelt/lib/scan-files.sh]` |

## Lessons
- A dot-named root is a second fail-open in the same raw walk: `find . -type d -name '.*' -prune` prunes the root
  itself, so `lint .` scanned nothing and passed; moving to scan_files fixes it as a side effect, and the snapshot
  must include a dot root to show it.
- A raw walk with no `-type f` also lists symlinks and directories named `*.java`; scan_files lists regular files only,
  so a symlinked source is no longer scanned. No fixture or repo tree holds one, so the snapshot shows no diff.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-22-fail-closed-walks.md | kit | 2026-10-22 | pending | 1 |`
