<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · go-javasrc-scanner

**Session**: niagara-research quality round 8, work unit T8k-1, branch `r8/t8k1-javasrc` from origin/main 657ae5b
**Delta count**: 2

## What happened
The Java-parsing lints all build on the awk lib `toolbelt/lib/method-boundary.sh` (`mb_strip` + `mb_parse`).
Before any of them moves to Go, this unit added one shared Go package, `go/internal/javasrc`, with the same
semantics: `Strip` (comments blanked, literal text kept, end state 0/1/2), `Methods` (1-based start/end/name) and
`Lines` (awk records: split on `\n`, `\r` kept). Table tests port every `method-boundary.bats` case plus edge cases,
written RED against a stub first. `TestDifferentialAwk` runs the awk lib under every awk on the host over the same
inputs and requires a byte-identical dump, so equivalence is proven, not claimed. No lint uses the package yet.

## Evidence
- RED: with a stub package, `TestStrip`, `TestStripEndState`, `TestMethods`, `TestLines` and `TestDifferentialAwk`
  failed; GREEN after the port `[ev: go/internal/javasrc/javasrc_test.go]`
- Differential, repo corpus (table inputs + `tests/fixtures/**/*.java` + `build-n4-module-kit/**/*.java`): 176 files,
  2,667 lines, 199 methods, 0 diffs under awk (gawk 5.2.1), mawk and gawk --posix `[ev: go/internal/javasrc/differential_test.go]`
- Differential with `JAVASRC_DIFF_ROOTS=~/modulos_niagara_n4/Cliente` (local only, CI never sets it): 978 files,
  302,813 lines, 7,235 methods, 0 diffs under the same three awks `[ev: go/internal/javasrc/differential_test.go]`
- Teeth: changing the class-body guard to depth 1, dropping char quotes from the brace counter, or shortening the
  20-line backward scan each turns the tests red `[ev: go/internal/javasrc/javasrc_test.go]`
- Speed on the corpus repeated 8 times (19,744 lines, one run, shared machine): Go `Strip` 5-9 ms and `Methods`
  8-10 ms; awk strip about 430 ms (gawk) / 170 ms (mawk), strip+parse about 550 ms / 300 ms
  `[ev: go/internal/javasrc/bench_test.go]`
- Two awk-lib quirks found and kept (pinned as table cases, not fixed): an annotated parameter with parentheses
  (`void f(@Named("x") String s) {`) hides the method, because the signature pattern `\([^)]*\)` stops at the first
  `)`; an anonymous-class field (`Runnable r = new Runnable() {`) is entered as a method named `Runnable`, because
  the `new` exclusion only tests the identifier right before `(` `[ev: go/internal/javasrc/javasrc_test.go]`
- `.githooks/pre-push` treats only kit files (minus BUILD-STATE/retros) and `scripts/**` as build-relevant, so a
  change under `go/**` passes the gate without a retro `[ev: .githooks/pre-push]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **The pre-push gate covers `go/**`.** Add `go/**` to the build-relevant set (except `go/bin/`), so a Go port of a toolbelt script carries a BUILD-STATE line and a retro like the script it replaces. Why: the toolbelt wrappers now exec Go binaries, so a `go/` change changes lint behavior while the gate stays silent. Reviewer check: a `go/` diff without a BUILD-STATE line is a finding. | `.githooks/pre-push` (build-relevant set) | `[ev: .githooks/pre-push]` |
| Δ2 | **Decide the two `mb_parse` quirks in the lib, then in Go, together.** Fix the annotated-parameter miss (allow balanced parentheses inside the parameter list) and decide whether an anonymous-class field counts as a method; change the awk lib and `javasrc` in one unit and update both pinned cases, so `TestDifferentialAwk` stays at 0 diffs. Why: the miss is fail-open (the method's body is never checked); fixing only one side breaks the equivalence proof. Reviewer check: a change to one implementation without the other is a finding. | `toolbelt/lib/method-boundary.sh` § Case A; `go/internal/javasrc` | `[ev: go/internal/javasrc/javasrc_test.go]` |

## Lessons
- A port is proven by running both implementations on the same inputs, including a real client corpus, not by
  porting the test cases alone.
- Byte-wise scanning matches awk's character-wise scanning when every tested character is ASCII; the non-ASCII table
  case and the client corpus confirm it.
- Keep quirks of the original as pinned tests during a port; fix them later in both sides at once.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-go-infra`) — Δ1 already folded in code (61a2f53d, `.githooks/pre-push:88`; its open issue in `ledger.d/2026-10-22-go-lint-engine.md` marked DONE); Δ2 rejected for now, needs code: change the `mb_parse` quirk cases in `toolbelt/lib/method-boundary.sh` and `go/internal/javasrc` together (both still pinned as kept quirks, `javasrc_test.go:170-180`); its open issue in `ledger.d/2026-10-22-go-lint-engine.md` stays.
