<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · mb-strip-string-literals

**Session**: niagara-research quality round 8, work unit T8h, branch `r8/t8h-mb-strip-strings` from origin/main fd77e16
**Delta count**: 2

## What happened
The shared comment stripper `mb_strip` in `toolbelt/lib/method-boundary.sh` had no string or char literal state.
A `//` inside `"tcp://host"` dropped the rest of the line, and a `"/*"` string blanked every line up to the next
`*/`. `mb_parse` runs on the stripped lines, so method boundaries moved and code disappeared from every consumer
lint (fail-open). `mb_parse` also counted a `{` or `}` inside a literal as a brace. The T9e review found the same
blind spot in `lint-driver-scaffold.sh` and fixed it locally (9c41fdf). This unit fixed the shared lib, tests first.
`mb_strip` now copies `"..."`, `'...'` (with `\` escapes) and `"""..."""` text blocks (state carried across lines)
unchanged, contents included, because consumers match string text (ORDs, BFormat patterns). `mb_parse` skips braces
inside the same literals. `mb_strip` returns the open state at end of file (0, 1 block comment, 2 text block), so a
caller can mark boundaries as not trusted, as 9c41fdf does.

## Evidence
- RED on the base lib: `bats tests/method-boundary.bats` MB1, MB2, MB4, MB5, MB8, MB9 and MB-e2e fail
  (MB9: method `a` ended at line 3 on a `"}"` string, method `b` was missed); GREEN 10/10 after, under awk, mawk
  and gawk --posix `[ev: tests/method-boundary.bats]`
- MB-e2e: a class with `String g = "/*";` and 801 fields was reported clean by `lint-size.sh` (the lines after the
  string were blanked as a comment); it now WARNs with 805 hand-written lines `[ev: tests/method-boundary.bats]`
- Consumer suites unchanged: `bats -j 3` over lint-timers, lint-silent-protection, ext-writable-shape,
  lint-bformat-call, lint-license-isoperational-gate, lint-set-null-ord, lint-size, lint-wb-external-ord-value,
  lint-ws-placement, golden-parser, parser-oneliner, c11-close: 180 ok before and after `[ev: toolbelt/lib/method-boundary.sh]`
- Client corpus (9 consumer lints over 5 client trees, base lib vs fixed lib): only `mb_parse` depth changed output,
  all from brace literals (`'{'`, `'}'`, `"{"`) that the old counter read as braces. Removed: 3 false lint-size WARNs
  on panccadia `methodBody()` test helpers (a `'{'` kept the 20-line method open). Added: 3 lint-size WARNs on
  chihuahua `importLinks()`, `fromBAlarmRecord()`, `_parseNotesJson()` and one lint-ws-placement WARN on
  `BChiUp.ensureSchedule()` (earlier unbalanced brace literals shifted the depth, so these methods were never
  entered); `ChiAuditHelper._parseEntry()` span 106 -> 108 `[ev: toolbelt/lib/method-boundary.sh]`
- Private copies of the old stripper remain: `lint-silent-protection.sh` § D6 and its Pattern-B pre-pass, and the
  pass-1 strip of `lint-ext-writable-shape.sh` carry the same no-literal loop `[ev: toolbelt/lint-silent-protection.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Lints with a private comment stripper move onto `mb_strip`.** `lint-silent-protection.sh` (§ D6 and the Pattern-B pre-pass), `lint-ext-writable-shape.sh` (pass 1) and `lint-driver-scaffold.sh` call `mb_strip` instead of their own loop, and a non-zero return becomes a "method boundaries not trusted" WARN. Why: each private copy still drops code after a `//` inside a string, and a fix in the lib does not reach them. Reviewer check: a lint with its own `"*/"` / `"//"` scan loop is a finding. | `BUILD-LOOP.md` § Lint doctrine (shared method-boundary parser rule); the three lints | `[ev: toolbelt/lint-silent-protection.sh]` |
| Δ2 | **Literal blanking for token matching is one shared helper.** A lint that must not match inside strings (lint-timers `blank_str`, 9c41fdf's blanking) uses one `mb_blank_lit` in the lib, applied after `mb_strip`; `mb_strip` itself keeps literal text because consumers read it. Why: two blanking copies already differ (text blocks, escapes). Reviewer check: a second `blank_str`-shaped function is a finding. | `toolbelt/lib/method-boundary.sh`; `toolbelt/lint-timers.sh` | `[ev: toolbelt/lib/method-boundary.sh]` |

## Lessons
- A comment stripper without literal state fails open: it hides code, so every lint built on it misses findings
  instead of reporting false ones, and no consumer test catches it.
- Strip comments but keep literal text in a shared lib; blanking literals is a per-consumer choice, and the brace
  counter must skip literals on its own.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 doc half `BUILD-LOOP.md:234-238` (a private stripper is a finding); the two bash lints still on a private stripper (lint-silent-protection, lint-ext-writable-shape) are an owed Go lint `[deferred: retro mb-strip-string-literals Δ1]` in `ledger.d/2026-10-23-r8-u0-retro-flips.md` (lint-driver-scaffold is already on the Go engine). Δ2 OBSOLETE: its second `blank_str` consumer, lint-timers, moved to Go (`go/internal/lint/rules/timers/awk.go:84`, 2d7b31a), so a bash `mb_blank_lit` has no second user; one shared Go literal blanker belongs in `go/internal/javasrc`. INDEX row generated from the header: `| 2026-10-22-mb-strip-string-literals.md | kit | 2026-10-22 | folded | 2 |`
