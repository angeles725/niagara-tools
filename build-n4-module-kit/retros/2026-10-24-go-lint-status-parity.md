<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-status-parity

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/status-parity-go` from origin/main 2285da0
**Delta count**: 2

## What happened
`toolbelt/lint-status-parity.sh` (bash + one gawk process per file) is now the thin `n4lint lint-status-parity`
wrapper over `go/internal/lint/rules/statusparity`, header, usage, row and `# Mutation:` pins kept. No new engine
field: `First` + `BareUsage` + `StrictFails` (WARN prints FAIL under `--strict`) + `BarePathRow`, and the rule sets
`Finding.Path` to the class (basename without `.java`), which the runner keeps, so the row stays
`WARN  lint-status-parity  <Class>  N=…` and `commissioning-verify.sh` relays it unchanged. The K20 row text comes
from `Spec.EmptyError.Msg`. lint-config-sanity's private `@NiagaraProperty` accumulator moved to
`javasrc.NiagaraProperties` (+ `ParenDepth`), unit-tested, and both rules call it (config-sanity goldens unchanged).

Declared differences of the port (byte identity otherwise): (a) an unreadable file is `cannot scan` on stderr,
exit 3 (gawk printed its fatal and the lint exited 0); (b) a file name holding a blank keeps its whole class name
(the bash `while read cls n m` split it into the N and M columns).

Truth fixes after byte identity, each RED first (Go golden + bats pin): T1, a one-line `@NiagaraProperty` closes
on its own line (the awk always appended the next line, so every other one of a run of one-line annotations went
uncounted; LSP-oneline); T2, annotations are read in the comment-blanked Strip view, so a commented-out slot
neither makes a false WARN nor hides a real one (LSP-comment).

## Evidence
- Goldens: 28 cases recorded from the bash lint (4 runs of `tests/lint-status-parity.bats`, LSP-locked pinned by `TestLocked`, 24 edge trees: argument shapes, K20, names, annotation shapes, comments, CRLF / no final newline / tabs, walk order) RED against a stub Check, then green `[ev: go/internal/lint/rules/statusparity/statusparity_test.go TestGolden]`
- Byte identity old vs new (stdout, stderr, exit), default and `--strict`: Cliente (237 *.java, 5 rows, exit 0 / 1) and sources/decompiled (0 rows) identical; 4.51 s bash vs 0.03 s Go on Cliente, 10.15 s vs 0.12 s on decompiled
- T1: Cliente 5 → 6 rows (Leon-Guanjuato BRoomPanel N=5/M=4 appears; panccadia-leon BRoomPanel N=11/M=4 → N=14/M=7) `[ev: go/internal/lint/rules/statusparity/testdata/truth-one-line-1.txt]`
- T2: Cliente 6 → 6 rows (no commented slots there) `[ev: go/internal/lint/rules/statusparity/testdata/truth-comment-1.txt]`
- Shared accumulator: `TestNiagaraProperties` + config-sanity's 47 goldens unchanged `[ev: go/internal/javasrc/annotation_test.go TestNiagaraProperties]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The row subject is the file basename and every class of the file is counted together (a nested class's slots go to the outer one; two `BCompressorControl` rows on Cliente cannot be told apart); decide a per-class count and a `<path>` subject, then update the commissioning-verify / report-module relays (open_issue T5) | `build-n4-module-kit/toolbelt/lint-status-parity.sh` | `[ev: go/internal/lint/rules/statusparity/testdata/edge-shapes-1.txt]` |
| Δ2 | A port can often avoid a new engine field: a per-file rule may set `Finding.Path` to any subject (the runner fills only an empty one); say so in § Adding a lint rule | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/run.go findings]` |

## Lessons
- Recording goldens through a sed-rewritten copy of the bats suite (lint variable pointed at a recorder, case name from the test description) needs no edit of the real suite.
- A quirk-for-quirk accumulator kept only for the byte-identical commit, then replaced by the shared one in the truth fix, keeps each step provable without a shared option for the bug.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — Δ1 not folded, needs code (open_issue T5, a product decision), owed in `METHODOLOGY.md` § Kit maintenance (round-10 lint follow-ups) and `BUILD-STATE.md` open_issues; Δ2 cross-cutting Go-port lesson, cited by the lints-a fold. INDEX row generated from the header: `| 2026-10-24-go-lint-status-parity.md | kit | 2026-10-24 | folded | 2 |`
