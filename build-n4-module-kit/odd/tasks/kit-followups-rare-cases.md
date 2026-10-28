# Kit follow-ups: rare-case tests (2026-10-28)

## Objective
Close the advisory follow-ups listed in `kit-backlog-2026-10-27.md` § Follow-ups, mainly by pinning untested
rare cases with tests; fix a defect only when its new test shows one (RED first).

## Scope
- In: Go rule/tool packages named in § Follow-ups, their testdata/goldens, the backlog § Follow-ups list.
- Out: ledger-check LC4 side-table item (kept as a real defect by decision); new lints; client repos.
- Delivery: work-unit commits on `chore/kit-followups-rare-cases`; strategy `ask-on-risk`; forecast ~300-450 lines.

## Tasks
- [x] F1 rcscan: test `recheck()` on a scalar (non-`.map`) verdict.
- [x] F2 absoluteclaims: golden for claim units ending at a Markdown table and a code fence; resolve R2-001 wording.
- [x] F3 retrodeltas: cases for header-only mismatch and multi-file TOTAL.
- [x] F4 lifecycleguard: LG2 on anonymous-class `owner()` case; fixture comments say what each case pins.
- [x] F5 intrinsiclock: IL2 `<Outer>.this` in anonymous class; IL3 deferred body (lambda / anon class) not a nested lock.
- [x] F6 lexiconapostrophe: closed-quote true branch test.
- [x] F7 charsetdefault: keyword before `(` not read as receiver.
- [x] F8 structure/palette: commented-out `moduleName` skipped; part-over-root precedence golden.
- [x] F9 shared part-output prune helper in `go/internal/lint`; decide permissionsxml non-part `build/`/`test/` prune.
- [x] F10 update backlog § Follow-ups (move closed items, keep open ones with reason).
- [ ] F11 real `toolbelt/build.sh --werror` run on one client module (werror R3 remainder).

## Routing
- F1-F10: delegated direct (one writer; 2+ non-trivial files).
- F11: delegated per-action worker (build run, no kit source change unless it finds a kit defect).

## Progress / evidence
Route F1-F10: delegated direct, one writer (this session). Pinning = the new test passed first; its teeth were shown
by a mutation of the code under test (reverted after). Defect = RED observed first, then GREEN.
| Task | Commit | Evidence |
|---|---|---|
| F1 rcscan scalar recheck | acca0d9b | pinning: `truth-innerhtml-computed-3` passed first; dropping the scalar branch of `recheck()` fails only it |
| F2 absoluteclaims | 40135db5 | pinning: `table-fence` golden (a table/fence glued to the line above shares its unit); R2-001 line reworded into two cases (word boundary, devguide tag); ending a unit at `\|` or a fence, or dropping `\b`, fails |
| F3 retrodeltas | 0ef0f996 | pinning: header-only MISMATCH case + `TestTotalOverSeveralRetros` (4 retros, 2 dirs); mutations CRD-header, CRD-total, per-dir INDEX cache, file count, mismatch count each fail |
| F4 lifecycleguard | dcc9b1dd | DEFECT: RED `station-started` missed `BAcmeAnon.java:15` (two anonymous `BComponent` bodies shared the LG2 key); GREEN keys by the enclosing class-body brace; callbacks.txt comments say what each case pins |
| F5 intrinsiclock | dcb947af | DEFECT x2: RED `locks` missed IL2 `BAcmeDeferred.java:6` (`<Outer>.this` in an anonymous body) and reported IL3 at :10/:11 (lambda / anonymous method under a lock); GREEN; mutations IL2-outer-this-anon, IL3-deferred, strict region start, lambda match fail |
| F6 lexiconapostrophe | 08ff5aec | pinning: `TestLoneApostrophe` (8 cases); LX-closed-quote mutation and dropping the blank check fail |
| F7 charsetdefault | 6a3c4d5e | DEFECT: RED `receiver` missed `B.java:3-5` (`return/assert (s + "!").getBytes()`); GREEN via `parenKeyword`; `label(s + "!")` stays unjudged |
| F8 structure/palette | 0f0d0066 | DEFECT: RED `l17-module-name-3` (commented-out moduleName won); GREEN reads gradle with comments blanked. Pinning `l17-module-name-4` (part over root): swapping the lookup order fails |
| F9 shared prune | 9680c52e | `lint.PartOutputs()` replaces three copies; `TestPartOutputs` + permissions-xml `group-level-build`; pruning `*/build` or dropping part `build/` fails |
| F10 backlog | (this commit) | § Follow-ups: closed items dropped with commits; ledger-check LC4, werror remainder, loggername raw moduleName, intrinsic-lock wrapper header kept open |

F9 decision (B9 R4-001/R3-001): a `build/` or `test/` directory not directly under a part (a group-level `build/`)
stays read. Conservative: no generated source is known to land there, pruning it by name would hide source, and for
lint-permissions-xml Java outside `-rt/src/` is never judged anyway; a group-level `module-permissions.xml` is judged.

Verification (2026-10-28, after 9680c52e): `cd go && memcap go test ./internal/...`: 133 ok, 0 FAIL; `go vet
./internal/lint/... ./internal/rcscan/... ./internal/retrodeltas/...`: clean; `gofmt -l go/internal`: empty; `bats -j 4
--no-parallelize-within-files` over the 11 affected suites (lifecycle-guard, intrinsic-lock, charset-default,
structure, permissions-xml, adapter-cfg-default, lexicon-apostrophe, absolute-claims, count-retro-deltas, logger-name,
rc-scan): 128/128 ok; `lint-guard-pins.sh --strict .`: no WARN.

## Next step
F1-F10 done (not pushed). Build worker runs F11. Open from this run: loggername raw `moduleName` read and the
`toolbelt/lint-intrinsic-lock.sh` header wording (both in the backlog § Follow-ups).
