<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-servlet-try-scope

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/servlet-try-scope` from origin/main e6581fd, with r10/delays-method-scope merged in first (git merge) for javasrc.Scopes
**Delta count**: 2

## What happened
retro go-lint-delays-method-scope Δ1: lint-servlet's input-400 check (servlet.go:174) accepted any `try {` on the
parse line or the 5 lines above it, with no method boundary and no check that the try was still open. It now asks
whether a try block of the same method or class-level lambda body (javasrc.Scopes / Enclosing) opens on or above the
parse and encloses it; a try of another handler or one already closed decides nothing, and the 5-line limit is gone.

Two scanner defects surfaced on the way and were fixed in javasrc first, each its own green commit with RED observed:
(1) Methods took the deepest depth of a body's first line as its body depth, so `void f(String s) { try { p(s); }`
ended the method on that line and the rest of it was in no scope; the body depth is now the depth of the brace the
signature opens (a second declared difference from mb_parse); (2) Scopes now keeps a body still open at the end of
the file (an unbalanced file) as a scope running to the last line, so a statement there keeps its scope (Methods
still drops it, the mb_parse contract).

## Evidence
- RED: truth-try-scope-1 (port: only the false FAIL at line 20; want FAIL at 7 and 11) and bats LSV2e; TestMethods body-depth case got `[{x 2 4} {w 7 7}]`; TestScopes unbalanced case got no scope `[ev: go/internal/lint/rules/servlet/testdata/truth-try-scope-1.txt]`
- Declared golden changes: servlet edge-catch-1 (K.java:36, a parse after a closed try, now FAILs input-400) and edge-input-1 (I.java:9, 6 lines into its own try, no longer FAILs)
- Follow-ups closed on the same branch: input-400 reads its enclosing try from javasrc.TryBlocks (RED truth-try-resources-1: a parse in a try-with-resources body or resource specification was a false FAIL; bats LSV2g), Cliente 15 → 15 and decompiled 22 → 22; block()'s unused needOpen path removed with every golden and the real output byte-identical `[ev: go/internal/lint/rules/servlet/testdata/truth-try-resources-1.txt]`
- Real trees: lint-servlet Cliente 15 → 15 rows (input-400 0 → 0), decompiled 24 → 24 (input-400 3 → 3); the 7 rules on Methods/Scopes give the same rows after both javasrc fixes (Cliente 38, decompiled 454); no new row `[ev: go/internal/lint/rules/servlet/servlet.go inTry]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | lint-servlet's catch-no-400 search still reads up to 4 lines after a try's end with no scope bound; a catch belongs to its try by brace structure, so read it from the try's closing line structurally. DONE 2026-10-24: javasrc.TryBlocks + the deciding-catch rule (first clause catching NumberFormatException or a supertype, innermost try outward, past rethrowing clauses); RED truth-catch-structure-1 / LSV2f; decompiled 24 → 22 (FileServlet's rethrowing :702/:728 dropped, :748 kept), Cliente unchanged | `go/internal/lint/rules/servlet/servlet.go` | `[ev: go/internal/lint/rules/servlet/servlet.go catch-no-400]` |
| Δ2 | javasrc.Methods now differs from mb_parse in two declared shapes (wrapped signatures, body depth); the remaining bash lints on mb_parse keep the old reading until ported | `build-n4-module-kit/toolbelt/lib/method-boundary.sh` | `[ev: go/internal/javasrc/javasrc_test.go goOnlyMethodsCases]` |

## Lessons
- Bounding a walk by scope only helps when the scope itself is right: the first scope-bounded rule exposed that a method opening a nested block on its first line had a one-line body.
- "Inside a try" is containment, not proximity: the deciding try is the one whose block still holds the parse.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — Δ1 folded as code (`javasrc.TryBlocks` catch-no-400), noted in `METHODOLOGY.md` § Kit maintenance; Δ2 cross-cutting Go-port lesson, cited by the lints-a fold. INDEX row generated from the header: `| 2026-10-24-go-lint-servlet-try-scope.md | kit | 2026-10-24 | folded | 2 |`
