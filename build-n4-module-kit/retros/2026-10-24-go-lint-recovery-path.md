<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-recovery-path

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/recovery-path-go` from origin/main 89390e5
**Delta count**: 2

## What happened
`toolbelt/lint-recovery-path.sh` (a bash read loop running one awk per file) is now the thin
`n4lint lint-recovery-path` wrapper over `go/internal/lint/rules/recoverypath`, header, usage, row, classification
and `# Mutation:` pins kept, so commissioning-verify.sh relays the same `FAIL  lint-recovery-path  ` rows. No new
engine field: `Raw` mode (no flag parsed, only the first argument read), FAIL severity, the K20 row text through
`Spec.EmptyError.Msg`, the scan_files_nonempty_or_exit walk contract, and `awkorder.Order` for the awk's
`for (slot in guarded_line)` row order. It was not on the fork-per-line ratchet baseline
(tests/fixtures/lint-fork-per-line/baseline.txt), so that file is unchanged.

Declared difference of the port (byte identity otherwise): an unreadable file is `cannot scan` on stderr, exit 3
(gawk printed its fatal and the lint exited by the other rows, 0 when they were clean: fail-open).

Truth fixes after byte identity, each one green commit with its RED quoted (Go golden + bats pin): T1, braces and
writes are read in code only (linescan.Code), not the raw line cut at its first "//" (LRP-comment); T2, the method of
a write comes from javasrc.Methods instead of a brace depth plus 5-line back-scan for a lower-case call (LRP-scope);
T3, a write starts at an identifier boundary and every write on a line counts (LRP-word).

## Evidence
- Goldens: 19 cases recorded from the bash lint (4 tests/lint-recovery-path.bats runs of the lint, LRP-locked pinned by `TestLocked`, 15 edge trees: argument shapes, K20, prune, slot shapes, methods, gawk for-in order of 17 slots, comments and literals, bytes); RED 9/19 and TestUnreadableFile against a stub Check `[ev: go/internal/lint/rules/recoverypath/recoverypath_test.go TestGolden]`
- Byte identity old vs new (stdout, stderr, exit): Cliente 0 rows, sources/decompiled 0 rows, tests/fixtures 2 rows (exit 1), the wrapper on tests/fixtures; 2.95 s bash vs 0.08 s Go on Cliente, 24.60 s vs 0.72 s on decompiled
- T1, T2, T3: Cliente 0 → 0, decompiled 0 → 0, tests/fixtures 2 → 2 each `[ev: go/internal/lint/rules/recoverypath/testdata/truth-comment-literal-1.txt]` `[ev: go/internal/lint/rules/recoverypath/testdata/truth-method-scope-1.txt]` `[ev: go/internal/lint/rules/recoverypath/testdata/truth-slot-names-1.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The row text says "no unconditional reset in started/atSteadyState/enable" while the classification also accepts changed() and start(); either name all five or drop changed(), whose writes are usually gated on the changed property | `build-n4-module-kit/toolbelt/lint-recovery-path.sh` | `[ev: go/internal/lint/rules/recoverypath/recoverypath.go resetter]` |
| Δ2 | The real trees give no row before or after any fix (Cliente, decompiled), so the truth fixes rest on synthetic trees only; seed a commissioning fixture from the PANCCADIA stranded-heater shape with a nested class and a wrapped signature | `tests/fixtures/lint-recovery-path/` | `[ev: go/internal/lint/rules/recoverypath/testdata/truth-method-scope-1.txt]` |

## Lessons
- A lint with 0 rows on every real tree still needs its edge trees: the awk's method naming, comment cut and identifier matching were wrong on 3 of 5 synthetic shapes and silent on the real code.
- Replacing a private brace-depth scope by the shared method scanner removed the lint's own parser (and its bugs) rather than patching it.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-recovery-path.md | kit | 2026-10-24 | pending | 2 |`
