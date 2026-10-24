<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-slot-coverage

**Session**: niagara-research quality round 10 (Ayudante1), branch `r10/slot-coverage-go` from origin/main e45a7ba9
**Delta count**: 3

## What happened
`toolbelt/slot-coverage.sh` (501 lines: grep/sed/comm pipelines and two `xargs awk` passes) was ported to Go as
`go/internal/slotcoverage` + `go/cmd/slot-coverage`, byte for byte first, then three false results were fixed
RED-first, one commit each. It is not a lint row emitter (three modes, each with its own lines and exit contract),
so it is a standalone package over the shared pieces instead of a `lint.Spec`: `lint.Walk`, a new
`javasrc.PropertyScan` (the `NiagaraProperties` accumulator fed line by line, `NeedParen` off for the awk rule)
and a new `golden.RunMain` (`golden.Run` for a `cli.Main`).

- **Port.** 64 goldens recorded from the bash on origin/main: every `slot-coverage.bats` run plus edge trees
  built from the kit fixtures. `TestUnreadable` holds the chmod cases, recorded by hand. The bash quirks were kept
  in the identity commit: grep errors then a verdict, gawk's fatal open ending the slot scan, the xargs blank
  split, per-line leftmost-longest `<type>` grep, an annotation carried across files.
- **T1 path with a blank.** xargs split the file list, so a source under `my dir/` was never read.
- **T2 unreadable input.** An unreadable xml or lexicon still got a verdict; an unreadable source ended the awk
  scan, so with nothing scanned the verdict was `pct=N/A`, exit 0. Now each is exit 3 `cannot read`.
- **T3 types via xmlsrc.** A `<type>` written over two lines was not required, a commented-out one was, and a
  malformed module-include.xml still got a verdict (R8-D12).
- **Mid-run correction.** The repository is public. The first golden set included goldens built from two client
  trees; they were dropped before any commit. Client-tree parity runs locally only and is reported as counts.

## Evidence
- Identity: 64 goldens + 5 `TestUnreadable` cases green on the identity commit `[ev: go/internal/slotcoverage/testdata]`
- Client-tree parity (local, not committed): 2 client trees, 7 modules × (parse, `--strict`, `--facade`, per-slot) = 28 of 28 runs identical in stdout, stderr and exit, before and after T1-T3 `[ev: retro go-slot-coverage]`
- Timing: per-slot on the largest module 0.11 s → 0.03 s `[ev: retro go-slot-coverage]`
- Truth fixes: T1 `edge-ps-path-space` + `SP-path-space`; T2 `TestUnreadable` + `SP-unreadable`; T3 `edge-parse-lex-shapes*`, `truth-xml-malformed` + `SC-xml-multiline` `[ev: tests/slot-coverage.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A toolbelt tool that is not a lint row emitter is ported as `go/internal/<pkg>` + `go/cmd/<tool>` over the shared pieces (`lint.Walk`, `javasrc`, `xmlsrc`, `strerror`) and replayed with `golden.RunMain`; list `slotcoverage`, `javasrc.PropertyScan` and `golden.RunMain` in the layout table | `go/README.md` § Layout; `CONTRIBUTING.md` § 2.2 | `[ev: go/internal/lint/rules/golden/golden.go]` |
| Δ2 | The repository is public: goldens, fixtures and test data come from kit fixtures and hand-built edge trees only. Byte parity on client trees runs locally, and the PR reports it as run counts that name no client | `build-n4-module-kit/BUILD-LOOP.md` § 5 Kit tooling doctrine (Port byte-identical first) | `[ev: retro go-slot-coverage]` |
| Δ3 | Before porting a `... \| xargs awk` pass, record one edge golden for each of its hidden fail-opens: a path with a blank (split by xargs), an unreadable file in the middle (gawk's fatal open drops every later file; with `2>/dev/null` the verdict is silently clean), and an open construct at a file end (carried into the next file) | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/slotcoverage/testdata/edge-ps-java-shapes.txt]` |

## Lessons
- `xargs awk ... 2>/dev/null` over a file list can drop files without any error: one unreadable file can turn a tree into a clean `N/A` verdict.
- A grep-based XML reader is wrong both ways: it misses tags written over several lines and counts commented-out ones.
- Generalize the shared helper (`PropertyScan`, `RunMain`) when a port needs a variant. Never copy it.
