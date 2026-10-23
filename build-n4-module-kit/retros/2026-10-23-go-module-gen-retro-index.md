<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-23 · kit · go-module-gen-retro-index

**Session**: niagara-research quality round 8 work unit T8j-1, branch `r8/t8j1-go-module` from origin/main 23a34b6
**Delta count**: 2

## What happened
The user set a standing rule on 2026-10-22: improvements and new tools are written in Go when possible, and
niagara-tools gets its own Go module. T8j-1 created that module (`go/`, stdlib only) and ported the first tool,
`toolbelt/gen-retro-index.sh`, to `go/internal/retroindex`. The `.sh` file stays the entry point as a thin
wrapper that builds `go/bin/gen-retro-index` on demand and execs it, so every caller (`new-retro.sh`,
`sweep-build-state.sh`, CI, the pre-push gate) keeps working unchanged. Two bats cases (H11/H12 in
`tests/build-retro-sync.bats`) copy the generator into a fake repository toplevel without `go/`, so the wrapper
there exits 3; the fixture must mirror the repository layout, and this unit applies that fix (Δ2 landed).

## Evidence
- Byte-identity harness: the origin/main bash version and the wrapper run at the same path over 21 shapes
  (no args, `--check` fresh/stale/missing, extra trailing newlines, explicit absolute/relative/trailing-slash
  root, empty argument, every header violation, empty and missing `retros/`, `--help`, unknown flag, two
  positionals, `--`); stdout, stderr, exit code and the written INDEX.md match in all 21 `[ev: T8j-1 harness]`
- `go/internal/retroindex` golden test renders the real `retros/` and equals the committed INDEX.md byte for byte
  `[ev: go/internal/retroindex/retroindex_test.go TestGoldenRealTree]`
- `tests/gen-retro-index.bats` passes unchanged against the wrapper; new GRI-go pins exit 3 for a missing Go
  module and for a PATH without `go` `[ev: tests/gen-retro-index.bats GRI-go]`
- H11/H12 fail with `gen-retro-index: no Go module at <tmp>/top/go`: `_fixture_top` copies
  `gen-retro-index.sh` out of the repository `[ev: tests/build-retro-sync.bats _fixture_top]`
- `ci_steps.py` treats a step-level `working-directory:` as UNCOVERED, so the CI Go steps use `cd go && …`
  inside `run:` `[ev: scripts/lib/ci_steps.py STEP_KEYS]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A ported toolbelt tool keeps its `.sh` name as a thin wrapper over `go/bin/<tool>`.** The wrapper builds the binary when it is missing or older than a Go source, execs it with the same arguments, and exits 3 when `go` is missing; the port is proven byte-identical against the bash version before the swap. Why: callers, hooks and CI keep one stable path while the logic moves to Go. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-module-gen-retro-index]` |
| Δ2 | **A test fixture that copies a toolbelt script out of the repository also mirrors `go/`** (a symlink to the repository `go/` next to the copied kit). Why: a wrapped tool resolves `go/` relative to its own path, and H11/H12 broke on exactly that. | `tests/build-retro-sync.bats` `_fixture_top` | `[ev: retro go-module-gen-retro-index]` |

Δ2 landed in this unit: `_fixture_top` symlinks the repository `go/` into its fake toplevel, cited
`[ev: retro go-module-gen-retro-index Δ2]`, and H11/H12 pass again. Δ1 is documented in `CONTRIBUTING.md` § 2.2;
the retro stays pending until a fold review decides Δ1.

## Lessons
- Port behind the old entry point: keep the script path, swap only its body, and compare old vs new at the same
  path so absolute paths in messages match.
- A bash `$(...)` comparison ignores trailing newlines on both sides; a port must keep that, or `--check` turns
  stricter than the tool it replaces.
- Search the tests for `cp …toolbelt/` before wrapping a script: a copied wrapper loses its repository context.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 `CONTRIBUTING.md:100-103` § 2.2, Δ2 `CONTRIBUTING.md:106-107` + `tests/build-retro-sync.bats:293` `_fixture_top`; tokens in `ledger.d/2026-10-23-r8-u0-retro-flips.md`. INDEX row generated from the header: `| 2026-10-23-go-module-gen-retro-index.md | kit | 2026-10-23 | folded | 2 |`
