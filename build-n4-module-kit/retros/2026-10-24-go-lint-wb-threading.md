<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-wb-threading

**Session**: niagara-research quality round 10, branch `r10/wb-threading-go` from origin/main 1c2e425
**Delta count**: 3

## What happened
`toolbelt/lint-wb-threading.sh` (two awk programs per file in a read loop) is now the per-file rule
`go/internal/lint/rules/wbthreading` and a thin wrapper that execs `n4lint lint-wb-threading`. Its contract needed two
engine features, in their own commit with tests: `Mode Anywhere` (`--strict` before or after the root, the bash's
`unknown option` / `unexpected argument` errors) and `Spec.CheckFirst` (rows `<check>  WARN  <path>:<line>  <detail>`,
which report-module relays as a swap member). The awk visited the callee set with `for (k in a)`, so the call chain a
row names follows gawk's hash order: `rules/awkorder` reproduces it (an edge case pins `doInvoke -> gamma() -> walk()`
over five candidate helpers). The bash never used `mb_parse` (its own `brace_body` counts braces), so the pending
wrapped-signature change to `javasrc.Methods` does not touch this port. Byte-identical first, then two truth fixes,
RED first; the comment-text view the agent check needed is a shared `javasrc.StripComments` (one scan for code and
comments, `Strip` delegates to it), not a second scanner.

## Evidence
- Goldens recorded from the original bash on copies of the roots: 9 bats runs and 25 edge trees (argument shapes and
  orders, hash-order chains, BqlQuery detail, raw comments and strings, brace and indent shapes, depth and cycles,
  unbalanced braces, agent comment shapes, walk order and pruning, CRLF, no final newline). `TestLocked` pins
  LW-locked with the bash text. RED: 19/34 cases failed against a stub `Check` `[ev: go/internal/lint/rules/wbthreading/wbthreading_test.go]`
- Byte identity old vs new (stdout, stderr, exit): Cliente (2 rows, exit 0; exit 1 under `--strict`), Apillm-wb, and
  sources/decompiled (1 row, ± `--strict`). Timing: Cliente 859 → 105 ms, decompiled 18.1 s → 0.94 s
  `[ev: toolbelt/lint-wb-threading.sh]`
- Rows: Cliente's 2 are true and in live source: chihuahua-wb `BBatchLinkEditor.java:303` (`doInvoke -> searchNavTree()
  -> collectNavMatches() -> getNavChildren`, no guard anywhere in the file: the DFS on the UI thread the bats header
  cites) and `:67` (`@AgentOn(types = "baja:Component")` with no justification). The decompiled row was false:
  program-wb `BatchCommands.java:233` opens a `BBqlQueryBuilder` dialog, matched as `BqlQuery` by substring
- Truth fix 1 (RED: four `truth-*` cases, bats WBT-code): both checks read code. A nav call or guard named only in a
  comment or a string, a brace in a string, a commented-out `@AgentOn`, and a `whyNot` variable as "justification" no
  longer count. Three recorded edges change as declared; real trees unchanged (Cliente 2 → 2, decompiled 1 → 1)
- Truth fix 2 (RED: `truth-whole-words`, bats WBT-words): the nav and guard words match whole words and the row names
  the nav call found (the awk always printed `getNavChildren`). edge-bql now names `BqlQuery`; decompiled 1 → 0 (the
  false BBqlQueryBuilder row), Cliente 2 → 2 `[ev: go/internal/lint/rules/wbthreading/wbthreading.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **The traversal check skips one-character callees.** The awk's `get_ids` keeps identifiers of two or more characters, so a helper named `r()` is never expanded and a nav call behind it is invisible (edge-depth class I). Kept for byte identity; a truth fix would accept one-character lowercase names, measured on real trees first. | `go/internal/lint/rules/wbthreading` | `[ev: go/internal/lint/rules/wbthreading/testdata/edge-depth-1.txt]` |
| Δ2 | **Phase 1 only collects helpers declared on an indented line.** A `private` method at column 0 (`^[[:space:]]+(private|protected)` in the awk) is not a helper, and a declaration whose name is on the next line (`private\n void x()`) neither (edge-shapes). Real Workbench sources indent members, so no row changes today; the next truth pass could read helpers from `javasrc.Decls` (G41 branch) instead of a line regex. | `go/internal/lint/rules/wbthreading` | `[ev: go/internal/lint/rules/wbthreading/testdata/edge-shapes-1.txt]` |
| Δ3 | **A mutation spot check must use a case that reaches the code under test.** The class-I guard case in edge-depth never exercised the guard (its callee was one character long, Δ1), so the whole-word guard fix needed its own discriminating case. | `go/README.md` § Adding a lint rule (step 6) | `[ev: go/internal/lint/rules/wbthreading/testdata/truth-whole-words.txt]` |

## Lessons
- When a bash lint's output depends on gawk's `for (k in a)` order, port the order (rules/awkorder), not a sort: the
  call chain a row names is part of its bytes.
- A check that needs comment text is served by one scan that returns both views; a second comment scanner would be
  the duplicate G41 flags.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — Δ1 and Δ2 not folded, needs code (truth fixes measured on real trees), owed in `METHODOLOGY.md` § Kit maintenance (round-10 lint follow-ups) and `BUILD-STATE.md` open_issues; Δ3 cross-cutting Go-port lesson, cited by the lints-a fold. INDEX row generated from the header: `| 2026-10-24-go-lint-wb-threading.md | kit | 2026-10-24 | folded | 3 |`
