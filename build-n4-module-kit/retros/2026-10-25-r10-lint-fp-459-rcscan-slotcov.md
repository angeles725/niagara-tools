<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-lint-fp-459-rcscan-slotcov

**Session**: niagara-research quality round 10 (Ayudante3), branch `fix/lint-fp-459-rcscan-slotcov` from origin/main 6d8e99bf
**Delta count**: 1

## What happened
The last open lint shapes of kit issue #459 outside report-module, each checked against origin/main first
(8 of the 9 issue bullets were already fixed by #464 / #474 / #483; lint-servlet cache-nofinger `?v=` is PR #528):

- slot-coverage per-slot, OPERATOR in a comment: a FALSE POSITIVE, fixed. The OPERATOR test read the raw joined
  `@NiagaraProperty` text, so a READONLY slot whose annotation carries a comment such as "Read-only (never
  OPERATOR-writable)" was required to have a lexicon key and reported MISSING (client station A
  `condenser1AutoLocked`, flags `SUMMARY | READONLY`). The test now reads the annotation with comments blanked
  (`javasrc.Strip`, mirrored line for line beside the unchanged `PropertyScan`); the name scan and the known-key
  set (STALE) are unchanged, so only the OPERATOR set moves. The recorded `edge-ps-java-shapes` golden had captured
  the bash behaviour for a trailing `// not OPERATOR` comment (`commentOp` MISSING); its want now drops it, a
  declared truth difference from the recording.
- rc-scan host-literal, `base: "http://<host>/..."` in the config of an unselected mode: a TRUE POSITIVE, no
  change. Already decided on main (e991a8d7, recorded in the rc-scan.sh header and retro go-rc-scan (c)): a
  one-word mode edit makes the fixed plain-http host live; the fix is a relative base.
- rc-scan browser-floor under `--profile unknown`: AS DESIGNED, no change. It is WARN (exit 0) under `unknown`
  and `lan`, FAIL only under `--strict` or `--profile hmi|both`; types/frontend-standard.md § Profiles defines
  `unknown` as "rc-scan.sh browser-floor stays WARN", and report-module passes only `--profile <ui_profile>`
  (never `--strict`). The issue's 202 rows were vendored three.js (cleared by RC33) plus the module's own CSS.

## Evidence
- RED before the fix: `TestGolden/truth-operator-comment` reported `MISSING blockNote` and `MISSING lockedFlag`
  (a `//` comment and an inline `/* */` comment naming OPERATOR on READONLY slots) `[ev: go/internal/slotcoverage/testdata/truth-operator-comment.txt]`
- True-positive neighbours that must still fire, in the same golden: `knob` (OPERATOR beside a plain comment) and
  `endpointUrl` (OPERATOR after a `"http://..."` string literal, whose `//` is not a comment) stay MISSING
- Mutation: testing OPERATOR on the raw text again fails `truth-operator-comment`, `edge-ps-java-shapes` and the bats
  pin SP-operator-comment `[ev: tests/slot-coverage.bats SP-operator-comment]`
- Real tree (client station A rack-control rt module, lexicon with the three `*AutoLocked` keys removed so the slot needs
  one): origin/main `pct=98.0`, `MISSING condenser1AutoLocked`; branch `pct=100.0`, no row (only condenser1 carries
  the comment)
- rc-scan re-check on main with a synthetic tree (`mode: "json"` beside `base: "http://example.invalid/..."`, a
  flex `gap` and an `inset`): host-literal FAIL under every profile; browser-floor WARN under unknown and lan,
  FAIL under hmi

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A flag test over joined `@NiagaraProperty` text reads comment-stripped code: lint-config-sanity CS2 (`setpointPass` over `f.Raw`) still tests `OPERATOR` on raw text, so a zero-default setpoint whose annotation comment says "not OPERATOR" would FAIL; feed it `f.Stripped()` as statusparity and facets-req already do | `go/internal/lint/rules/configsanity/configsanity.go` setpointPass | `[ev: go/internal/slotcoverage/perslot.go slots]` |

## Lessons
- A golden recorded from the bash for byte identity can pin a false positive; a truth fix updates that want and
  names the difference, rather than treating the recording as the contract.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-25-r10-lint-fp-459-rcscan-slotcov.md | kit | 2026-10-25 | pending | 1 |`
