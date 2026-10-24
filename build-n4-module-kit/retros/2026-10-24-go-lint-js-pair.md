<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-lint-js-pair

**Session**: niagara-research quality round 10 (Ayudante2 lint-engine owner), branch `r10/js-lints-go` from origin/main 780c7095
**Delta count**: 3

## What happened
The two browser-code lints are ported: `toolbelt/lint-spa-poll-no-recovery.sh` (one awk per .html/.js file) and
`lint-jasmine-ux.sh` (a module-root check) are thin `n4lint` wrappers over `go/internal/lint/rules/spapoll` and
`rules/jasmineux`. One engine field landed first: `Spec.OptionalInAll` (a one-profile rule, here the -ux spa-poll
lint, finds nothing on the -rt roots `n4lint --all` also visits; the EmptyOK opt-out applies under --all only).
jasmineux is a RootRule with `BarePathRow` and `EmptyOK` (the ER5 exit-0 exception, now also asserted by
TestRootRulesOnAnEmptyRoot). The shared tokenizer `go/internal/jssrc` gained `Code` (comments and string/template/
regex contents blanked, line lengths kept) and `HTMLScripts` (only an HTML file's <script> contents), each with its
own test; spapoll reads through them.

Truth fixes on spa-poll, each RED-first in one green commit: T1 code view (a commented or quoted
location.reload() counted as recovery, a commented catch made the shape, a catch after "http://" was hidden, a
template quoting setInterval was the poll call); T2 minified lines (>1000 bytes) are not searched for the poll call
(T1 uncovered bundled three.js / webChart setTimeouts the "//" cut had hidden by luck, and such a line also masked a
page's own poll); T3 recovery shapes (`location.href ==` is not an assignment, `location.reload ( )` is a reload).
jasmine-ux needed none.

## Evidence
- spapoll: 32 goldens (15 bats runs, 17 edge trees) RED 17/32 + TestUnreadable against a stub; jasmineux: 19 goldens (4 bats runs, 15 edge trees) RED 10/19 + TestDirs (empty-directory shapes and JUX-finderr) `[ev: go/internal/lint/rules/spapoll/spapoll_test.go TestGolden]`
- Byte identity, default and `--strict`: spa-poll Cliente 3 rows, decompiled 0, tests/fixtures 0 (1.13 s → 0.24 s on Cliente, 0.57 s → 0.02 s on decompiled); jasmine-ux on the 8 Cliente -ux part roots (5 WARN rows) and decompiled (no src/rc); both wrappers identical
- T1: Cliente 3 → 5, decompiled 0 → 1 (all three new rows minified third-party bundles); T2: Cliente 5 → 3 (the bash lint's rows again, byte for byte), decompiled 1 → 0; T3: no row changed `[ev: go/internal/lint/rules/spapoll/testdata/truth-minified-1.txt]`
- The 3 Cliente rows: Honeywell/MX60/chihuahua chihuahua-ux `js/app/SubscriptionPool.js:567` (`_watchdogTick`) and `js/lib/VersionCheck.js:111` (`_checkOnce`), in the MX60 chihuahua checkout (dormant, last commit 2026-06-21, not a stale copy): the module's location.reload() lives in another file (UpdateBanner.js) and the watchdog resubscribes instead of reloading, so both are the per-file heuristic's limit rather than a confirmed blank-kiosk defect; Leon-Guanjuato DashboardPan-ux `index.html:2294` is the stale Leon-Guanjuato checkout (4f5f1c7)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | spa-poll reads one file: a recovery in another file of the same app (chihuahua's UpdateBanner.js reload) does not count, and only the first poll call of a file is checked; a module-level view (all rc/ JS of a -ux part) would close both | `go/internal/lint/rules/spapoll/spapoll.go` | `[ev: go/internal/lint/rules/spapoll/testdata/edge-shapes-1.txt]` |
| Δ2 | The minified-line cut (1000 bytes) is a heuristic; a pruned vendor/ directory or a `*.min.js` exclusion in Selection would state the intent directly | `go/internal/lint/walk.go` Selection | `[ev: go/internal/lint/rules/spapoll/testdata/truth-minified-1.txt]` |
| Δ3 | Review the 2 chihuahua rows with the client: a watchdog that resubscribes without a reload may still leave the kiosk blank after a station restart | `Cliente/Honeywell/MX60/chihuahua/.../SubscriptionPool.js:567` | `[ev: go/internal/lint/rules/spapoll/spapoll.go]` |

## Lessons
- A comment-cut that is wrong in one direction can hide the noise of another: fixing the code view surfaced minified-library rows the bash had dropped by accident, which needed their own fix.
- A golden tree cannot hold an empty directory; a RootRule over directories needs a Go test for those shapes beside its goldens.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-a`) — Δ1, Δ2 and Δ3 (client review) parked as an owed `open_issue:` in its `ledger.d/` fragment, `[deferred:]` in `BUILD-LOOP.md` § 5 kit tooling doctrine (needs code).
