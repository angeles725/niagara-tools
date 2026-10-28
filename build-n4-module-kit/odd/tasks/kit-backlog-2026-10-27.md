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
- [x] B2 · A5 skill/doc drift. Route: inline (two doc lines).
- [x] B3 · A6 pre-push every ref + CI npm job + skip allowlist. Route: inline (writer is this session).
- [x] B4 · A4 sweep-build-state remainder. Route: inline (one Go package).
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

- 2026-10-27 B2 (inline). `scripts/ng-deploy.sh` read on this branch: annotation guard + opt-in slotomatic in modes A/C
  only (`:642-657`; -ux slotomatic in mode A when -ux is annotated, `:528-535`), version-bump precheck in every mode and
  schema-risk in A/C before the copy (`:668-678`; exits 51/52/54, 12/53), verify-module gate after the copy in A/C
  (`:704-710`, exit 50, `--no-gate`). `skill/SKILL.md:37` and `BUILD-LOOP.md` §4 (:129) said "backup → build → copy →
  verify; slotomatic guard rt-only"; §5 (:216) listed exits 51/52 without 54. All three now name the gate and prechecks.
  Checks: the 10 suites that read BUILD-LOOP.md or skill/SKILL.md, 107/107 ok; `kit-cite-resolve --strict` on both
  cards LIVE 0 / HISTORY 0. `scripts/check-skill-drift.sh` FAILs (installed launcher differs) — it already failed on
  d982a704; the reinstall (`scripts/install-skill.sh --force --skill build-n4-module`) is owed to the parent session.
  Gaps outside the surface: `build-verify.md:11` still says "its slotomatic guard is **rt-only**; run verify-module.sh
  afterwards", and `build-verify.md:276` says modes A and C regenerate -ux slots (mode C is -rt only, `ng-deploy.sh:532`).
  Finding: the gate runs AFTER the copy, so a gate FAIL leaves the new jar in the station's modules dir (exit 50 but
  the copy happened) — a candidate for a later ng-deploy fix, not in this brief.

- 2026-10-27 B3 (inline). Re-verified: `.githooks/pre-push:49-51` read only the first ref line, and a branch
  deletion on that line exited 0 for the whole push. Design: the per-ref body is `gate_ref() ( … )` (a subshell, so
  its `exit`s end one ref's gate), looped over every stdin line; the first failing ref's status stops the push.
  - RED (real scratch repos): H30 (second ref ungated kit work) and H31 (deletion first, bad ref second) exit 0 on the
    base; H32 (two passing refs) stays green. GREEN: build-retro-sync + pre-push-* + changelog-gate + install-hooks +
    fail-open-idioms 94/94 ok. A first H30 draft changed README.md, a live doctrine card, which tripped the live-cite
    working-tree check; the fixture uses `notes.txt`.
  - CI: the bats job sets up Node.js 24.19.0 (`actions/setup-node@v5`, npm cache keyed on
    `toolbelt/eslint/package.json`) and installs eslint 10.11.0 + acorn 8.18.0 (the exact pins already in that
    package.json), asserting each installed version. The same step through `scripts/ci-local.sh --only` installed and
    asserted both locally. `ci-local.sh` gains a SKIP rule for the Node.js setup step (node on PATH).
  - Global SKIP allowlist: new `scripts/check-bats-skips.sh` + `tests/bats-skip-allowlist.txt` (whole-reason EREs,
    client-tree reasons written as generic shapes), called at the end of the CI bats step; routed in BUILD-LOOP §7
    (kit-links L12). The 77 skip lines of the last main CI run (job 121205121598) replayed through it: 70 allowlisted,
    the 7 unlisted are exactly the acorn/eslint skips the npm install removes. RED: SK1-SK7 not ok (no script);
    GREEN 7/7. CL17's "full run" fixture now skips for an allowlisted reason; new CL18 (unlisted skip fails the step).
  - Observed mutations (restored, `cmp`): check call dropped from ci.yml → CL18; `-x` dropped → SK3 (after SK3 got an
    unanchored allowlist line — the first version did not bite because its pattern carried `^…$`); unlisted branch
    disabled → SK2; ERE validation disabled → SK6; plan check disabled → SK5.
  - With the npm tools installed: report-module + lint-vendor-floor + scaffold-module + ci-local + check-bats-skips
    141 tests, 140 ok before the CL17 fixture fix, then all ok; their 4 local skips all allowlisted.
  - `actionlint`: not installed on this host (ci.yml parsed by `ci-steps` through `ci-local.sh --list`, 0 UNCOVERED).
  - Gap: `build-n4-module-kit/toolbelt/eslint/.gitignore` (outside the surface) still says CI never installs these
    tools; the lockfile + `npm ci` switch it names is recorded as an open_issue in
    `ledger.d/2026-10-27-kit-backlog-a6-ci.md`. Pinned shellcheck 0.10.0 also flags SC2015 at `tests/module-map.bats:25,32`
    (modularity campaign, not this unit) — fixed in B4b.

- 2026-10-27 B4 (inline). Re-verified on this branch: markers balanced + freshness already in
  `go/internal/buildstatefields` (`CheckEnvelopes`, `freshness`); no ledger tag check, no module-field WARN, `--strict`
  documented as "not implemented" (`toolbelt/sweep-build-state.sh:14`) and refused as an unknown flag. Intent of the
  audit's "module_root/source_of_truth WARN": the two DECLARED fields every module envelope must carry
  (`BUILD-STATE.src.md` § How to read, "required for every module"); `--age --strict` contract from the wrapper header
  (E5): exit 1 on escalation — clear, so implemented rather than deferred.
  - Design: the tag check reuses `citeparse.Card` (the scanner kit-cite-resolve and the fold audit use), so a
    backticked `` `[ev:` `` stays code (one such literal is in `ledger.d/2026-10-24-r10-kcr-false-positives.md`); a tag is
    unbalanced when citeparse reports it unterminated or its text holds another `[ev:` / `[deferred:` opener. It runs
    after the INDEX checks, before freshness, only in a ledger.d/ kit.
  - RED: `TestMainLedgerEvidenceTags` (3 FAIL cases), `TestMainSourceOfTruthWarn` (2 WARN cases),
    `TestMainAgeStrict` not ok on the base; fixtures `writeState`, S8, S9 gained `module_root` / `source_of_truth`, and
    the "unknown flag" case of `TestMainAgeArgs` now uses `--bogus`. GREEN: `memcap go test ./internal/buildstatefields/`
    ok; `go vet` clean; bats D10 (`tests/retro-debt.bats`) + build-retro-sync + sweep-fold-audit + kit-state +
    kit-links 91/91 ok before D10, D10 ok.
  - Observed mutations (restored, `cmp`): LT1 unterminated case off, LT2 nested case off, the `ledgerTags` call off,
    SOT1 field check off, AGE-strict off — each turns its Go test red.
  - Real kit (fresh kit-state output): content check exit 0 with no WARN; `--age --strict --today 2026-10-27` 0 pending.
  - Decision: CI's `sweep-retro-debt` step stays non-strict; turning retro debt into a CI failure is the owner's call.

## Next step
B4b.
