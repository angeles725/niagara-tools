<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-agent-on-shape

**Session**: niagara-research quality round 10, branch `r10/agent-on-shape-go` from origin/main ecfa381 (origin/main
c0fdb46 merged in for `Spec.ModuleRoot`)
**Delta count**: 2

## What happened
`toolbelt/lint-agent-on-shape.sh` (awk_lint_each over every module-include.xml) is now the rule
`go/internal/lint/rules/agentonshape` and a thin wrapper that execs `n4lint lint-agent-on-shape`. It is the first
`internal/xmlsrc` consumer (R8-D12; go/README § XML rules now names it, replacing split-package-check, which reads no
XML). Byte-identical first: the awk's backward line scan reproduced, `Spec.ShortScanError` (new, own commit and test)
for awk_lint_each's `cannot scan <file>: <reason>`, ContinueOnWalkError for awk_lint_list, and `Spec.ModuleRoot` to
keep it out of `n4lint --all` (module-include.xml sits beside src/, never under a source root). Then the switch to
xmlsrc, RED first, with every difference declared.

## Evidence
- Goldens recorded from the original bash: 3 bats runs and 23 edge trees (argument shapes, attribute order, single
  quotes, entities, two `<on>` on a line, comments, CDATA, CRLF, an empty type, lookalike element names, `>` in a value,
  a malformed file, walk order and pruning). `TestUnreadable` pins the walk-error and unreadable-file texts, checked
  against the bash on the same tree. RED: 15/26 cases failed against a stub `[ev: go/internal/lint/rules/agentonshape/agentonshape_test.go]`
- Byte identity old vs new (stdout, stderr, exit): Cliente (17 module-include.xml, 0 rows), panccadia-leon,
  sources/decompiled (no module-include.xml, only jar module.xml: K20, exit 3) and the kit repository (55 files, 1
  row: the deliberate report-module `agent-on` fixture, `FooService`). Timing: Cliente 65 → 20 ms
- xmlsrc switch (RED: `truth-xml-elements`, `truth-xml-malformed`, bats AOS-xml). Declared differences: a second `<on>`
  on a line and a single-quoted value are read; an `<on>` inside a comment or CDATA is not; entities are decoded
  (`m:T&#36;X` is `m:T$X`, clean); an empty `type=""` is malformed; the row line is the element's; a malformed file is
  `cannot scan <file>: lint-agent-on-shape: <file>:<line>:<col>: <reason>` (exit 3, the other files still scanned).
  Eight recorded edges change as declared (one, two-per-line, had two root elements, so it is now correctly malformed:
  `junk after document element`). Real trees unchanged: every real module-include.xml is well-formed
  `[ev: go/internal/lint/rules/agentonshape/agentonshape.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A rule's own error is printed after the engine's `cannot scan <path>: <rule>:` prefix.** A malformed XML file reads `cannot scan F: lint-agent-on-shape: F:3:1: ...`, naming the file twice. The engine could print an `*xmlsrc.Error` bare after `cannot scan` (it already names its path) in one shared change, for every XML consumer to come. | `go/internal/lint` run.go (rule error text) | `[ev: go/internal/lint/rules/agentonshape/testdata/truth-xml-malformed.txt]` |
| Δ2 | **Regenerating golden want sections by rebuilding the tree outside the runner loses a final newline.** A position in an error message (here 3:1 vs 2:11) depends on it; regenerate by running `golden.Run`'s own replay, or edit and re-run, never by a second tree builder. | `go/internal/lint/rules/golden` | `[ev: go/internal/lint/rules/agentonshape/testdata/edge-malformed-1.txt]` |

## Lessons
- An XML lint ported byte-identical first shows every line-based blind spot as a recorded edge, so the XML switch
  is a list of declared differences, each visible in a golden diff.
- A fixture with two root elements is not "two tags on a line": XML says it is malformed, and the edge recorded the
  awk reading it anyway.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-a`) — Δ1 already landed (the engine prints an `*xmlsrc.Error` once, `go/README.md` § XML rules), cited in `BUILD-LOOP.md` § 5 kit tooling doctrine (XML lints); Δ2 `BUILD-LOOP.md` § 5 kit tooling doctrine (golden cases from the replay).
