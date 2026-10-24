<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-bformat-call

**Session**: niagara-research quality round 10, branch `r10/bformat-call-go` from origin/main 45ddb3b5, with
`r10/agent-on-shape-go` merged in first (unit A moves its goldens)
**Delta count**: 2

## What happened
Two units. (A) The engine prints an `*xmlsrc.Error` returned by a rule once (`<lint>: path:line:col: msg`) instead of
behind `cannot scan <path>: <rule>:`, which named the file twice (retro go-lint-agent-on-shape Δ1, one change for every
XML consumer); lint-agent-on-shape's three malformed-file goldens and header moved as declared. (B)
`toolbelt/lint-bformat-call.sh` (awk_lint_run over *.java, *.lexicon, *.bog, *.palette) is now the rule
`go/internal/lint/rules/bformatcall` and a thin wrapper that execs `n4lint lint-bformat-call`: byte-identical first,
then two truth fixes RED first. Java needed none: the bash already read `mb_strip` (comments blanked, string literals
and text blocks kept), which is exactly where BFormat patterns live; edge-java pins that decision.

## Evidence
- (A) RED: `TestXMLErrorOnce` printed the doubled line before the run.go change `[ev: go/internal/lint/lint_test.go]`
- (B) Goldens recorded from the original bash on copies of the root: 6 bats runs and 17 edge trees (Java strings,
  comments and text blocks, lexicon comments and continuations, bog attributes, comments and entities, palette,
  extensions, CRLF, no final newline, walk order). RED: 14/23 cases failed against a stub `[ev: go/internal/lint/rules/bformatcall/bformatcall_test.go]`
- Byte identity old vs new (stdout, stderr, exit): Cliente (0 rows, ± `--strict`; 2,943 Java files, 31 lexicons and
  30 palettes across Cliente and decompiled), sources/decompiled (0 rows), the PANCCADIA station `config.bog` (a
  zipped bog, read-only copy) and the kit repository. Timing: Cliente 2501 → 62 ms, decompiled 29.2 s → 0.45 s
- Truth fix XML (RED: `truth-xml`, `TestZippedBog`, bats BF-xml): .bog and .palette through `internal/xmlsrc`: attribute
  values and text, entity-decoded, at the element's line; an XML comment is not read; `&#37;user&#37;` is found; a zipped
  bog is read through its file.xml (the awk read compressed bytes); a malformed file is the positioned error, printed
  once. The zip case is built so the pattern never appears as plain bytes (a short payload is stored, not deflated, and
  passed on the awk read). edge-bog and edge-palette change as declared; real trees unchanged, the station bog
  well-formed (111 ms)
- Truth fix lexicon (RED: `truth-lexicon`, bats BF-lexicon): .lexicon through `internal/props`: a `#` comment is not read,
  a continued value is reported at its entry, `%` is a percent. edge-lexicon changes as declared; real trees
  unchanged `[ev: go/internal/lint/rules/bformatcall/bformatcall.go]`
- Rows: 0 on every real tree, before and after; no BFormat misuse in the client sources, lexicons, palettes or station

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A zip test must prove the zip compresses.** A short payload is stored, not deflated, so the pattern sits in the file as plain bytes and the old byte-level read passes the test too; assert the pattern is absent from the zip bytes before relying on it. | `go/internal/xmlsrc` tests, any zipped-bog consumer | `[ev: go/internal/lint/rules/bformatcall/bformatcall_test.go]` |
| Δ2 | **Station bogs are zipped; a text read of one is silently empty.** Every lint that greps a `.bog` as text (outside xmlsrc) sees compressed bytes on a real station's config.bog and reports it clean. Inventory the remaining text readers of `*.bog` and move them onto `xmlsrc.Open`. | `toolbelt/` bog readers | `[ev: toolbelt/lint-bformat-call.sh]` |

## Lessons
- Reuse is decided by what the file is: a lexicon is a properties file and a bog is XML, so the shared readers give
  comments, continuations, entities and zips for free where the awk had to approximate them.
- A RED that passes for the wrong reason (the stored zip) is caught only by asserting the test's own precondition.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-bformat-call.md | kit | 2026-10-24 | pending | 2 |`
