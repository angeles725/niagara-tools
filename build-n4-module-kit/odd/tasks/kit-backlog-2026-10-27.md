# ODD feature — Kit backlog 2026-10-27: bookkeeping, audit A4/A5/A6 remainder, review fix-forward

**Status**: IN PROGRESS (B1–B4 + B4b this run; B5–B8 later) · **Created**: 2026-10-27 · **Branch**: `chore/kit-backlog`
(from d982a704, the closed modularity campaign; its own PR) · **Repo**: angeles725/niagara-tools (PUBLIC — generic names only)

## Objective
Bring the kit's backlog documents in line with what has already landed, and close the remaining small audit items of
`odd/tasks/audit-2026-10-03.md` (A4 ledger integrity remainder, A5 skill/doc drift, A6 CI hardening), plus the advisory
findings of the approved modularity-campaign reviews (B4b), as Conventional-Commit work units, test-first.

## Problem / why
- The audit doc still shows A3 and A10 open although both landed; the 2026-10-15 register shows C40, C59, C81, C86 and
  C87 unfolded or partial although their code is on main; several ledger.d `open_issue:` lines name owed work that landed.
- A4: `sweep-build-state` does not catch an unterminated `[ev:` tag in a ledger fragment, does not WARN on a module
  envelope without `module_root` / `source_of_truth`, and documents `--age --strict` as "not implemented".
- A5: the skill launcher and BUILD-LOOP §4 describe `ng-deploy.sh` as "backup → build → copy → verify; slotomatic guard
  rt-only", which no longer matches its gate and prechecks.
- A6: `.githooks/pre-push` gates only the FIRST ref line of a multi-ref push; CI never installs the pinned npm tools, so
  the ESLint/acorn suites always skip; no global assertion keeps a suite from skipping for a new, unreviewed reason.

## Scope
In: the allowed edit surfaces of the brief (audit doc, register, ledger.d, skill/SKILL.md, BUILD-LOOP.md,
BUILD-STATE.src.md, sweep-build-state / kit-state wrappers and their Go packages, retros/, `.githooks/pre-push`, ci.yml,
scripts/, tests/, changelog.d/) and, for B4b, go/internal/modulemap, go/internal/lint/rules/modulemap, go/internal/rcscan,
go/internal/reportmodule, go/internal/lint/rules/servlet, toolbelt/lint-servlet.sh, toolbelt/module-map.sh.
Out: push, PR, GitHub writes (issue #459 stays open), skill reinstall (owed to the parent session).

## Plan (8 blocks)
| Block | Content | This run |
|---|---|---|
| B1 | Bookkeeping: audit A3/A10 ticked, register fold status (C40, C59, C81, C86, C87; done halves of C41, C47, C51), stale ledger open_issues closed | yes |
| B2 | Audit A5: skill/SKILL.md + BUILD-LOOP §4 name ng-deploy's gate and prechecks | yes |
| B3 | Audit A6: pre-push gates every ref; CI installs the pinned npm tools; global bats SKIP allowlist | yes |
| B4 | Audit A4 remainder: unbalanced `[ev:` in ledger.d (FAIL), module_root/source_of_truth WARN, `--age --strict` | yes |
| B4b | Review fix-forward of the approved modularity reviews (rc-scan, add-concern, MM7, lint-servlet header, report-module) | yes |
| B5 | New lints G21 `permissions-xml`, G34 `structural-tests`, G40 `logger-name`, G38 `absolute-claims` (+ G39 `count-retro-deltas`) | later |
| B6 | Issue #459 tail (rc-scan innerhtml computed-literal rows, go-rc-scan Δ3; lint-vacuous-tests report membership) + audit A7-D2 (rc-scan `status-default-ok`) | later |
| B7 | Audit A9: `ledger-check.sh` + scaffold VERSION-LEDGER, T1 test backfill, slotomatic last-comment-line WARN | later |
| B8 | Audit A7-D1: adapter `@NiagaraProperty defaultValue` vs pure-model Cfg default lint | later |

## Tasks
- [x] B1 · bookkeeping (docs). Route: inline (docs only, evidence already mapped).
- [ ] B2 · A5 skill/doc drift. Route: inline (two doc lines).
- [ ] B3 · A6 pre-push every ref + CI npm job + skip allowlist. Route: inline (writer is this session).
- [ ] B4 · A4 sweep-build-state remainder. Route: inline (one Go package).
- [ ] B4b · review fix-forward. Route: inline.
- [ ] B5 · G21 / G34 / G40 / G38 (+G39) new lints — later.
- [ ] B6 · #459 tail + A7-D2 — later.
- [ ] B7 · A9 — later.
- [ ] B8 · A7-D1 — later.

## Acceptance
- Each block lands as one or two work-unit commits with RED → GREEN evidence where a deterministic test exists.
- `memcap go test ./...` green once at the end; touched-package tests, bats suites, shellcheck, gen-*-index `--check`,
  strict fold audit, `scripts/changelog-gate.sh --range d982a704..HEAD` and `.githooks/pre-push` over the branch range pass.

## Checks
`memcap go test ./<pkg>/...` per package · `go vet` on touched packages · `bats -j 4` on touched suites ·
`shellcheck` 0.10.0 on touched shell · `actionlint` on ci.yml (if available) · `gen-lint-index.sh --check` ·
`gen-retro-index.sh --check` · `sweep-fold-audit.sh --strict --exact-stems --deltas-since 2026-09-24` ·
`scripts/changelog-gate.sh --range d982a704..HEAD`.

## Progress / evidence
(per block, appended as each work unit lands)
- 2026-10-27 B1 (inline). Re-verified on d982a704 before writing: A3 = six members at
  `go/internal/reportmodule/members.go:73-76,82-83`; A10 = tag v0.32.0 → 643ed84c (PR #543), v0.32.1, v0.33.0, #246
  closed; C40 = `go/cmd/run-niagara-test` (16296e05, goldens 49c75e62); C59 = `lint-private-api-import` member at
  `members.go:115` (the brief's `:110` had shifted), a5639149; C81 = 7edef596, #260 closed; C86 = af5de781,
  `build.sh:362`, `tests/build-sh.bats:527`; C87 = 0b71fc54, now `go/internal/preflight/sample.go:45-53`. Done halves:
  C41 scaffold emits `moduleTest-include.xml` (f9bc8194, fixtures 8812c1de — G97); C47 fixture roots `<permissions/>`
  (5de000e9 — G98); C51 cex Δ1 `SOURCES.md` § Exemplar source + `corpus-index.md` rows (aaf61de2 — P1). Their owed
  lints (G21, G34, G38/G39, G40) stay for B5. Ledger open_issues closed as `DONE <date> (<evidence>)`:
  r10-run-niagara-test-go (G33 phase 2), r10-run-niagara-test-run (first live run), r8-u5-c59-c62 (C59 doc half).
  Checks: register no-loss proof PASS (missing=0); `gen-build-state.sh --check` valid.

## Next step
B1.
