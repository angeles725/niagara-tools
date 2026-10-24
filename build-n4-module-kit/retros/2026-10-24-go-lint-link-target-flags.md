<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-link-target-flags

**Session**: niagara-research quality round 10, branch `r10/link-target-flags-go` from origin/main 8a1c26f
**Delta count**: 2

## What happened
`toolbelt/lint-link-target-flags.sh` (a `while read` loop forking one awk per file plus an awk harvest of the
`--wiring-map` Table 2) is now the per-file rule `go/internal/lint/rules/linktargetflags` and a thin wrapper that
execs `n4lint lint-link-target-flags`. Its argument contract (`[--wiring-map <file>] <java-src-dir>` in any order)
matched no engine mode, so the engine gained `Mode Valued` (`Spec.Option`) and `lint.Configurer` (the rule reads
its option value after the root check and before the walk) in their own commit with tests. The port went
byte-identical first, then one truth fix with RED first. The map contract is unchanged: the
`toolbelt/generate-wiring-map.sh` layout (a `## ` section naming Table 2, the section's first table row as header,
the Source column from that header, backticked first-column slots, whole-cell `_(fill)_`/self/n/a/none/dash
sentinels after backticks are stripped), and the LTF1 reason text `(wiring-map Table 2)` that report-module.sh and
commissioning-verify.sh match.

## Evidence
- Goldens recorded from the original bash through a recording wrapper that copies each run's root and map into a
  case directory (the suite reads repo fixtures and absolute paths): 25 bats runs and 29 edge trees (argument
  shapes and orders, a missing, repeated or directory map, CRLF, no final newline, walk order and pruning, the
  annotation and comment-block state machine, LTF2 names, Table 2 section, header, column and sentinel shapes).
  `TestLocked` and `TestEmptyDir` pin what the format cannot hold. RED: 25/54 cases failed against a stub
  `Check`/`Configure` `[ev: go/internal/lint/rules/linktargetflags/linktargetflags_test.go]`
- Byte identity old vs new (stdout, stderr, exit): Cliente (21 rows, exit 1) with and without the ford
  `docs/wiring-map.md`, ford -rt and -ux src with the map in either order, panccadia-leon, Leon-Guanjuato,
  `_salvage`, sources/decompiled (0 rows, ± the map) and the kit fixtures. Timing: Cliente 630 → 64 ms,
  decompiled 7456 → 371 ms `[ev: toolbelt/lint-link-target-flags.sh]`
- Why Cliente exits 1: all 21 rows are FAIL and true by the rule, none in live source. 15 are in the stale
  Leon-Guanjuato checkout (4f5f1c7, 53 commits behind origin): DashboardPan `BRoomPanel` and ColdRoomPan
  `BEvaporatorUnit` HOA/mode slots TRANSIENT by a documented "reverts to auto on restart (safety)" choice that the
  later persist-HOA doctrine (retro panccadia-persistent-config-hoa Δ1) superseded; 6 are in
  `_salvage/panccadia-leon-2026-09-21/untracked` (BCompressorPanel `hours1-3` READONLY link-in targets, `comp1-3Mode`
  TRANSIENT). The live panccadia-leon tree has 0 rows
- Fork-per-line ratchet: the lint was not on tests/fixtures/lint-fork-per-line/baseline.txt (its loop forks one
  awk per file, not per line), so no baseline row changed; the port removes the per-file forks anyway
- Declared port difference: an existing but unreadable `--wiring-map` is exit 3 (bash: the awk printed its error
  and the lint went on with no Table 2 target, exit 0 on a mapped READONLY slot); bats LTF-mapread was RED on the bash
- Truth fix (RED: four `truth-*` cases and bats LTF-code): the annotation is read on code (comments blanked,
  literal contents removed) for its marker, parentheses and flag tokens, and on the stripped lines for its name;
  the comment block stays on raw lines. A commented-out annotation, a flag word in a defaultValue string or in a
  comment inside the annotation, and a parenthesis in a string no longer count. Real trees unchanged (Cliente
  21 → 21, ± the ford map; decompiled 0 → 0) `[ev: go/internal/lint/rules/linktargetflags/linktargetflags.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Lint roots under stale checkouts and salvage copies report old defects as live FAILs.** Every Cliente row of this lint (21, exit 1) comes from the stale Leon-Guanjuato checkout or `_salvage/`; the live source is clean. A client-tree sweep should name its live module roots (or prune `_salvage` and checkouts behind origin) before reading a FAIL count as current. | `BUILD-LOOP.md` § 5 (kit sweeps) | `[ev: toolbelt/lint-link-target-flags.sh]` |
| Δ2 | **A recorder that copies roots keeps goldens free of machine paths; one that does not leaks them.** Arguments that are not copied (a missing root, a file root, a map under the scratch tree) were recorded with absolute scratch paths until the recorder copied file roots, kept trailing slashes and the edges used stable `/nonexistent/...` paths. Reviewer check: `grep -l "/tmp/" testdata/*.txt` is empty before a golden commit. | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/lint/rules/linktargetflags/testdata/edge-fileroot-1.txt]` |

## Lessons
- An argument contract the engine lacks becomes one Spec field and one interface, tested on their own, never a
  private argument parser in the rule.
- The comment block is the one place this lint must read raw text: the link-in convention lives in comments, while
  the annotation it heads must be read as code.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-link-target-flags.md | kit | 2026-10-24 | pending | 2 |`
