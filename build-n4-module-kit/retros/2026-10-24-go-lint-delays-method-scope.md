<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-delays-method-scope

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/delays-method-scope` from origin/main d051831
**Delta count**: 3

## What happened
An architect review found two fail-opens in the lint-delays Go port on main: the variable-binding walk
(`scan.go`, 50 lines back) and the positivity-guard walk (80 lines back) had no method boundary, and the binding
reader skipped a plain re-assignment (`long ms = p;`), so a call borrowed an earlier method's `Math.max(x, 1L)` or
`ms > 0` and an unfloored delay passed. Fixed in five green commits, each RED observed first:
1. `javasrc.Scopes` (+ `Enclosing`, `File.Scopes`): Methods plus the lambda bodies opened outside a method, from
   the same scanner with lambdas on (Methods unchanged); a lambda inside a method stays in the method's body, as the
   locals it captures do.
2. The binding walk stops at the enclosing scope's start and the nearest assignment of the variable decides
   (Math.max floor, BRelTime literal, helper; anything else FAIL unfloored), never falling through (LD25).
3. The guard walk stops at the same start (LD26).
4. `javasrc.Methods` names a signature wrapped before its brace (`void m(int a,` / `int b) {`, or a throws line),
   declared as its one difference from mb_parse; every rule's goldens re-run.
5. The helper scan (D2b) reads bodies from javasrc.Methods over comment-free code instead of a raw brace counter,
   which a `"{"` in a string stretched over the next method's floor (LD27).

Other "N lines back" walks checked: no other Go rule on main walks back for a binding or a guard; servlet.go:174
looks 5 lines back for a `try {` with no method boundary (out of this unit's surface, Δ1); the bash
lint-persist-hot-write already bounds its 5-line walk by the scope start; lint-wb-threading's 3 lines above an
annotation is comment proximity, not scope. The unlanded r10/recovery-path-go already uses javasrc.Methods.

## Evidence
- RED: truth-binding-scope-1 (port: only ms2 FAILs; want ms, ms2, ms3), truth-guard-scope-1 (port: two PASS rows guarded from other methods), truth-helper-body-1 (port: open() passes, later() FAILs), bats LD25/LD26/LD27; TestMethods wrapped cases got `[{z 11 12}]` and `[]` against the old scanner `[ev: go/internal/lint/rules/delays/testdata/truth-binding-scope-1.txt]`
- lint-delays on real trees, every commit: Cliente FAIL 4 → 4 (all four in the stale Leon-Guanjuato/Paccadia BDefrostController checkout; live roots panccadia-leon and the rest 0 → 0; no _salvage row), decompiled FAIL 28 → 28; no NEW FAIL, no row changed `[ev: go/internal/lint/rules/delays/scan.go scopeStart]`
- Methods fix, the 7 rules on Methods/Scopes: Cliente 38 → 38 rows, decompiled 453 → 454 (one new lint-session-store-lazy-evict WARN, opcUaClient-rt OpcUaMonitoredDataItemUtil.java:50, inside a wrapped signature); only the setnullord golden edge-method-shapes-1 changed (declared) `[ev: go/internal/javasrc/javasrc_test.go goOnlyMethodsCases]`
- Declared golden change: delays edge-window-1 (y2's binding 79 lines up in the same method now reads zero-floor k=0)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | lint-servlet's input-400 check accepts a `try {` up to 5 lines back with no method boundary (servlet.go:174): bound it to the call's enclosing try block or scope through javasrc | `go/internal/lint/rules/servlet/servlet.go` | `[ev: go/internal/lint/rules/servlet/servlet.go input-400]` |
| Δ2 | The bash lints still on mb_parse (lint-bformat-call, lint-license-isoperational-gate, and lint-size / lint-ws-placement until their ports land) keep missing wrapped signatures; port them or give mb_parse the same back-scan | `build-n4-module-kit/toolbelt/lib/method-boundary.sh` | `[ev: go/internal/javasrc/javasrc.go wrappedEnd]` |
| Δ3 | The unlanded r10/lint-size-go and r10/ws-placement-go goldens hold wrapped signatures (lint-size edge-methods, ws-placement edge-guard `wrapped()`): when they rebase on this, re-run and declare those rows | `go/internal/lint/rules/{size,wsplacement}/testdata` | `[ev: go/internal/javasrc/javasrc_test.go TestMethods]` |

## Lessons
- A backward walk with a line budget but no scope boundary is a fail-open whenever an earlier method holds the
  good-looking line; the scope must come from the shared scanner, never from a private brace counter.
- "Nearest binding decides" needs the reader to recognise every plain assignment as a binding; a reader that only
  recognises good bindings turns every bad one into a fall-through.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-lint-delays-method-scope.md | kit | 2026-10-24 | pending | 3 |`
