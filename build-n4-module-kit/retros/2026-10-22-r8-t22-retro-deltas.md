<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 9 -->
# 2026-10-22 · kit · r8-t22-retro-deltas

**Session**: niagara-research quality round 8 work unit T22 (cross-session retro), branch `kit/r8-t22-retro-deltas` from origin/main e3713cf
**Delta count**: 9

## What happened
Quality round 8 ran several sessions in parallel on the kit (PRs #270-#296). The same lessons came back across
them: new checks written in bash and later ported, lints that forked a process per source line, Java scanners
that read inside string literals or across nested scopes, a local shellcheck that missed a CI finding, hand
edits to generated ledgers, review candidates over the lens budget, and a loaded shared host. This retro files
the seven lessons once and folds them into `skill/SKILL.md` (short rules) and `BUILD-LOOP.md` (procedure), with
one structural guard for the per-line fork (Δ2). The coordinator added two more during the unit: when a
tool is worth porting to Go (Δ8) and how the Go code is layered so a new lint reuses one scanner, walker and
formatter (Δ9, a user requirement). `CONTRIBUTING.md` § 2.2 and § 6 already carried the Go-module
layout and the `ci-local.sh -j 3` replay, and `BUILD-LOOP.md` § 7 already said the retro index and the kit
ledger are generated, so Δ1, Δ4 and Δ5 refine and point instead of restating.

## Evidence
- Go layout today: `go/internal/{retroindex,cisteps,buildstate,facetscard,cli}` each behind a thin `go/cmd/<tool>/main.go`
  (PRs #288, #296); `internal/javasrc` landed in #297 (96f8bc2) and `internal/lint` is in flight (T8k-2a), so Δ9 is the layout new lint code
  starts from
- Go tools: `go/cmd/{gen-retro-index,ci-steps,gen-build-state,check-facets-card}` + `go/internal/*` + `go/go-tool.sh`
  wrappers (PRs #288, #290, #296); owed Go lints specified as ledger items G1-G3 (archive fragment) and G4-G7
  (`ledger.d/2026-10-22-r8-t9g-nav.md`) `[ev: retro go-module-gen-retro-index]` `[ev: retro go-ci-steps]`
- Per-line forks: retro line-loop-lint-forks timings (bql-string-concat 55.95 s -> 0.20 s, arbitrary-ord
  49.19 s -> 0.19 s; byte-identical stdout) `[ev: retro line-loop-lint-forks Δ1]`
- Guard: `tests/lint-fork-per-line.bats` RED with an empty scanner (FPL1, FPL3, FPL4 not ok), GREEN 4/4 after;
  mutations bite (raw-line match -> FPL2 not ok; no balance check -> FPL3; no read-variable stack -> FPL1+FPL4;
  a `printf | sed` planted on `$line` in lint-timers.sh -> FPL4). The real tree still holds 8 per-line forks in 5
  lints (lint-guard-pins 2, lint-lexicon-ascii 1, lint-silent-protection 1, lint-timers 3, lint-write-path 1),
  frozen in `tests/fixtures/lint-fork-per-line/baseline.txt` and owed in the ledger fragment
  `[ev: tests/lint-fork-per-line.bats]`
- Fail-open Java scanning: lint-driver-scaffold (#277, #283), lint-service-types (#279, #285), the shared
  `mb_strip` (#278) `[ev: retro mb-strip-string-literals]` `[ev: retro r8-276-driver-scaffold]`
- shellcheck 0.11 passed what CI's pinned 0.10.0 flagged as SC2015 (#273) `[ev: retro ci-local-replay]`
- Generated ledgers: `gen-retro-index.sh` (#280), `ledger.d/` + `gen-build-state.sh` (#292)
  `[ev: retro generated-retro-index Δ1]` `[ev: retro ledger-fragments]`
- Review sizing: native review stops a candidate over its lens budget (`lens_context_budget_exceeded`, observed
  at about 500 changed lines); PR #270 landed as a chain of commits reviewed one by one against the parent SHA
- Shared host: at load 28, `verify-module.bats` flaked under a wide `bats -j`; GNU `parallel` children inherit
  an open `flock` fd and kept the lock after the parent exited

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **New tools and checks are Go** in `go/` (`go/cmd/<tool>` + `go/internal/<pkg>` + a thin `.sh` wrapper over `go/go-tool.sh`); bash gets small fixes only; a check not yet ported is specified as an owed `Go lint` item in a `ledger.d/` fragment (shape: G1-G7). Why: user rule 2026-10-22; one tested implementation instead of bash then a port. | `skill/SKILL.md` § Hard Rules; `BUILD-LOOP.md` § 5 lint doctrine | `[ev: retro go-module-gen-retro-index]` |
| Δ2 | **A lint never forks a process per input line**: one awk per file (`lib/awk-lint.sh` `awk_lint_each`); `tests/lint-fork-per-line.bats` fails a new `printf`/`echo` of a `while read` variable piped into grep/sed/awk (two-way baseline). Why: 35-97 s per tree against 0.13-1.25 s. | `BUILD-LOOP.md` § 5 lint doctrine; `skill/SKILL.md` § Hard Rules | `[ev: retro line-loop-lint-forks Δ1]` |
| Δ3 | **Java-text scanning is literal-aware and scope-aware**: strings, chars and text blocks never feed a match, nested lambdas and anonymous classes never close the enclosing method; use `lib/method-boundary.sh` `mb_strip`/`mb_parse` (or the Go Java scanner of T8k once landed), never a private stripper. Why: four fail-open lint bugs in one round. | `BUILD-LOOP.md` § 5 lint doctrine; `skill/SKILL.md` § Hard Rules | `[ev: retro mb-strip-string-literals]` |
| Δ4 | **Before every PR run `scripts/ci-local.sh -j 3`** (exact CI replay, pinned shellcheck 0.10.0). Why: a local 0.11 missed SC2015. | `BUILD-LOOP.md` § 7; `skill/SKILL.md` step 7 | `[ev: retro ci-local-replay]` |
| Δ5 | **Generated ledgers are never hand-edited** — `retros/INDEX.md` (`gen-retro-index.sh`) and the kit block of `BUILD-STATE.md` (`ledger.d/` + `gen-build-state.sh`). Already in `BUILD-LOOP.md` § 7; the launcher step 7 named only the ledger, so it gains the index. | `skill/SKILL.md` step 7 | `[ev: retro generated-retro-index Δ1]` |
| Δ6 | **Size a review before it starts**: a candidate over ~500 changed lines stops with `lens_context_budget_exceeded`; land large doc/retro work as a chain of commits of at most ~400 lines, each reviewed with `--base-ref <exact parent SHA>` (never `origin/main`), then one PR; after review, merge origin/main into the branch instead of rebasing so the reviewed commits stay byte-identical. | `BUILD-LOOP.md` § 0.c (refines the size-at-planning bullet) | `[ev: retro comppan-pressure-staging Δ1]` |
| Δ7 | **Shared-host test discipline**: cap test parallelism at `-j 3` when several sessions share the host; GNU `parallel` children inherit an open `flock` fd, so start a locked parallel run under `setsid` and release it with `kill -- -<PGID>`. | `BUILD-LOOP.md` § 7; `skill/SKILL.md` step 7 | `[ev: retro r8-t22-retro-deltas]` |
| Δ8 | **Go-port criterion**: port a kit tool to Go when it PARSES structured input (Java, XML, jars, ledgers) or has had parsing bugs; keep pure orchestration glue (`build.sh`, `preflight.sh`, wrappers that only sequence gradle/copy steps) in bash. Every port proves byte-identical stdout, stderr and exit code against the bash version on the real `Cliente/` trees, records before/after timing, and keeps the bats suite as the black-box test. Why: the parsing tools are where the bugs and the time went; glue gains nothing from a port. | `BUILD-LOOP.md` § 5 kit tooling doctrine; `skill/SKILL.md` § Hard Rules | `[ev: retro go-module-gen-retro-index]` |
| Δ9 | **Modularity rule (user requirement)**: Go code in `go/` is layered — `internal/javasrc` (scanner only) → `internal/lint` (the `Rule` interface, the registry, one shared `Finding` type and one output formatter, one walker that parses each file once) → `internal/lint/rules/<lint>` (one rule and its tests) → thin `cmd/` binaries. Adding a lint is one rule file plus its registration, never a copied walker, formatter or stripper. Bash likewise reuses `toolbelt/lib/*` instead of private copies. Why: the four fail-open bugs of this round were private strippers; PRs #288/#296 already show the thin-`cmd/` over `internal/` shape. | `BUILD-LOOP.md` § 5 kit tooling doctrine; `skill/SKILL.md` § Hard Rules | `[ev: retro mb-strip-string-literals]` |

Owed (not in this unit's edit surface): `METHODOLOGY.md` § Kit maintenance should link Δ1-Δ3, Δ6, Δ8 and Δ9 (another
session is editing it in T9i), and the 8 baselined per-line forks are ported to `awk_lint_each` or Go; both are
`open_issue` lines in `ledger.d/2026-10-22-r8-t22-retro-deltas.md`.

## Lessons
- A rule that only lives in a retro or in CONTRIBUTING.md is not followed by an agent that reads the launcher; put
  the one-line rule in `skill/SKILL.md` and the procedure in `BUILD-LOOP.md`.
- A structural guard over existing code ships with a frozen two-way baseline, so it lands green without hiding the
  debt and cannot drift in either direction.
- A text scanner reports when it loses the structure (unbalanced loops or quotes); silence from a confused scanner
  is not a clean result.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-r8-t22-retro-deltas.md | kit | 2026-10-22 | pending | 9 |`
