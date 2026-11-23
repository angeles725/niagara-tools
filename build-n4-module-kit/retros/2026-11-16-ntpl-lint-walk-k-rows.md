<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-11-16 · kit · ntpl-lint-walk-k-rows

**Session**: kit rows K6, K7, K3, K8, K1 and K2 of the 2026-11-15 coordination-stretch retro: the template lint's walk stays inside the module, and the lint runs in report-module.
**Delta count**: 6

## What happened
A recheck of the template lint (lint G92) left the K rows: `lint.WalkFollow` followed a symlinked directory out of
the module root and pruned a link by its own name; an empty `--niagara-home` read as "no installation"; a jar
counted as shipped by its name alone; a `.ntpl` FIFO printed "nothing to check"; nothing ran the lint; and
verify-module kept a private copy of the gradle dependency reader. Containment came first, then the wiring.
K3 (stock modules a station template lists) needed no rule change: the rule already skipped modules the
installation ships whenever it had the installation path. It is closed by report-module passing that path.

## Evidence
- RED before GREEN: the prune, FIFO, empty-option, jar and link-row tests fail on the base walk and rule; the
  link-to-`/` case is not run against the base walk, which would walk the whole file system.
  `[ev: go/internal/lint/walkfollow_test.go]`
- Differential, base `eb9d5622` against the branch: `lint-ntpl-manifest-deps` on a module with an in-module link is
  byte-identical; `report-module --lint-only` differs only by the new member row and its count.
  `[ev: go/internal/reportmodule/ntpl_member_test.go]`
- The 116 re-recorded report-module goldens change no verdict and no exit code (verdict words counted pairwise
  over the diff). `[ev: go/internal/reportmodule/testdata/README.md]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. K6: `lint.WalkFollow` follows a link only when its real target is inside the root, prunes a followed link by its target too, reads each file once and returns the links it did not follow; the lint prints one `WARN ntpl-link` row per such link and never "nothing to check" over one | `toolbelt/lint-ntpl-manifest-deps.sh` header · Go `internal/lint` `walkfollow.go`, `internal/lint/rules/ntplmanifest` `links.go` | `[ev: go/internal/lint/walkfollow.go]` |
| Δ2 | APPLIED 2026-11-16. K7: `--niagara-home ""` is a usage error with the lint's usage exit (3, its documented contract); a module jar ships only when it is a regular file the lint can open; a `.ntpl` that is not a regular file is exit 3 | `toolbelt/lint-ntpl-manifest-deps.sh` header · Go `internal/lint/rules/ntplmanifest` `ntplmanifest.go`, `installed.go` | `[ev: go/internal/lint/rules/ntplmanifest/installed.go]` |
| Δ3 | APPLIED 2026-11-16. K1: verify-module's phantom-dep check calls `lint.DeclaredDeps`; its copy is gone (no behaviour change) | Go `internal/verifymodule` `phantomdep.go` | `[ev: go/internal/verifymodule/phantomdep.go]` |
| Δ4 | APPLIED 2026-11-16. K8, K2: `lint-ntpl-manifest-deps` is a once-per-module report-module member, relayed whole (row mode `whole`), with `report-module --niagara-home <dir>` (else the inherited `NIAGARA_HOME`); `build.sh` passes its `niagara_home`; every member golden re-recorded in its own commit | `BUILD-LOOP.md` § 5 · `toolbelt/report-module.sh` header · Go `internal/reportmodule` `members.go`, `rows.go`, `internal/buildmodule` `gate.go` | `[ev: go/internal/reportmodule/members.go]` |
| Δ5 | APPLIED 2026-11-16. K3 (user decision 2026-11-15, modules shipped by the installation are ignored): closed by Δ4, as the rule already ignored them when given the installation path | Go `internal/reportmodule` `ntpl_member_test.go` | `[ev: go/internal/reportmodule/ntpl_member_test.go]` |
| Δ6 | OPEN. `golden.Runner` has no re-record mode: a member added to report-module changes over a hundred golden cases, and re-recording them needed a throw-away hook (a case with `CHDIR=1` replays in its tree, so the hook must write to an absolute testdata path). Add an opt-in rewrite of the `want` and `stderr` sections | Go `internal/lint/rules/golden` `runner.go` · `record/README.md` | `[ev: go/internal/lint/rules/golden/runner.go]` |
