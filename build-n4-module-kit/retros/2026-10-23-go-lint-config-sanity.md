<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-23 · kit · go-lint-config-sanity

**Session**: niagara-research quality round 8 (Ayudante2), branch `r8/config-sanity-go` from origin/main 9c11a26
**Delta count**: 3

## What happened
`toolbelt/lint-config-sanity.sh` ran four gawk passes per file (CS1 interval <= duration, CS2 zero OPERATOR
setpoint, CS3 comment-only flag rule, CS4 permanent floor beside a disabled cutout). They are now one per-file
rule, `go/internal/lint/rules/configsanity`, and the script is the thin `n4lint lint-config-sanity` wrapper with
its header, usage and `# Mutation:` pins kept. The CS1 pairs come out in gawk's `for (k in a)` order, so the
order helper moved from `rules/timers` (private `gawkOrder`) to the shared `rules/awkorder` (`Order`, `Hash`)
first, timers output unchanged. The port keeps the awk's quirks for byte identity: raw text (no comment
stripping), an annotation open at end of file dropped, gawk number conversions.

Declared differences: (a) a listed file the scan cannot read is now `cannot scan` on stderr and exit 3, with the
other files' rows still printed; bash printed four gawk fatals and exited by the other rows (fail-open). (b) the
header promised a `// lint-config-sanity: ok` suppression and a `module-include.xml` defaults fallback, neither
implemented (0 uses of the marker in Cliente); both claims are gone, and the header now says what the code does
(same file not same class, CS3 on any slot type, CS2 reads the whole annotation, `*-worktrees` pruned, only the
first argument read).

(c) the bash CS1/CS2 reader always took the line after the @NiagaraProperty line, so the annotation right after
a one-line annotation was never read by CS1/CS2 (CS4 read it). Policy R8-D14 (a port does not keep a known lie)
made it a fix rather than a documented limitation: a one-line annotation now closes on its own line, as CS4's
does. On real trees 154 such hidden annotations exist (114 in Cliente), none forming a CS1 or CS2 finding, so no
row changes there.

## Evidence
- RED: `awkorder` tests before the package existed (`undefined: Order`) `[ev: go/internal/lint/rules/awkorder/awkorder_test.go TestOrderDifferential]`
- Goldens: 47 cases recorded from the bash lint (every `tests/lint-config-sanity.bats` run but LCS-finderr, plus edge
  trees: one-line swallow, nested annotations, CRLF, 7x5 hash order, 24 BRelTime forms, CS2/CS3/CS4 forms, walk
  order, argument shapes); mutants (identity order, no swallow, `ms >= 0`, `min >= 0`) each fail 1-12 cases
  `[ev: go/internal/lint/rules/configsanity/configsanity_test.go TestGolden]`
- Byte identity old vs new (stdout, stderr, exit): 9 fixtures one by one and together, 8 argument shapes, a locked
  sub-directory, an empty root, Cliente/{Honeywell,panccadia-leon,Leon-Guanjuato}, all of Cliente, niagara-research
  `sources/` and all of `modulos_niagara_n4` (7 rows); all identical except the unreadable file (difference a)
- Time: all of Cliente 2.80 s bash vs 0.07 s Go; niagara-research `sources/` 37.35 s vs 0.40 s
- RED (a): bats LCS-awkfail exited 1 on the bash lint, 3 after the switch `[ev: tests/lint-config-sanity.bats LCS-awkfail]`
- RED (b): bats LCS-doc failed on the old header's suppression claim `[ev: tests/lint-config-sanity.bats LCS-doc]`
- RED (c): bats LCS-oneline exited 0 (missed CS1 and CS2) and Go `TestOneLineAnnotation` got 2 of 4 rows before the
  fix `[ev: go/internal/lint/rules/configsanity/configsanity_test.go TestOneLineAnnotation]`; goldens edge-crlf and
  edge-oneline (was edge-oneline-swallow) gain the rows the bash lint hid (Crlf.java:4 CS1, Swallow.java:2 CS1 vs
  aDuration, Swallow.java:10 CS2)
- Old vs new after (c): modulos_niagara_n4 7 rows identical (4.12 s bash, 0.11 s Go), Cliente and niagara-research
  `sources/` 0 rows both

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A port of a gawk `for (k in a)` loop takes its order from `rules/awkorder`, never a private copy | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/awkorder/awkorder_test.go TestOrderDifferential]` |
| Δ2 | A lint header documents only behaviour a test pins; a claimed escape hatch or fallback with no code is removed, not kept as advice | `build-n4-module-kit/toolbelt/lint-config-sanity.sh` header | `[ev: tests/lint-config-sanity.bats LCS-doc]` |
| Δ3 | A port fixes a known false negative of the bash lint as a declared difference (RED first), never documents it as a limitation | `go/README.md` § Adding a lint rule, step 5 | `[ev: tests/lint-config-sanity.bats LCS-oneline]` |

## Lessons
- A per-file bash lint that runs one gawk per pass per file is slow on large trees (37 s on 483 decompiled files); the
  one-read Go rule took 0.4 s with identical bytes.
- Headers drift from code: two of the header's promises (a suppression marker and a second defaults source) had no
  code behind them; a pinning test is the cheap guard.
- Byte identity first, then fix: a quirk kept for identity and written up as a limitation is still a known lie;
  R8-D14 turns it into its own RED-first commit.
- The golden format has no empty-directory section; an empty-root case needs a non-Java placeholder file.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-go-infra`) — Δ1 and Δ3 `BUILD-LOOP.md` § 5 port rule; Δ2 `BUILD-LOOP.md` § 5 (a lint header claims only what a test pins).
