# ODD feature — Kit backlog 2026-10-27: bookkeeping, audit A4/A5/A6 remainder, review fix-forward

**Status**: B1–B4 + B4b DONE; run 2 (B5.0, B5.1, B6, B7) in progress (2026-10-27, unpushed on `chore/kit-backlog`); B5, B8 later · **Created**: 2026-10-27 · **Branch**: `chore/kit-backlog`
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
| B5.0 | Safety: `scripts/ng-deploy.sh` runs every precheck, the type-count verify and the verify-module gate on `build/libs` BEFORE the copy (a failing check copies nothing); docs: skill/SKILL.md:37, BUILD-LOOP §4/§5, build-verify.md:11/:21/:276 | run 2 |
| B5.1 | Recorded gaps: `toolbelt/eslint` lockfile + `npm ci` in CI + `.gitignore` claim; MM7 duplicate-concern rule in `toolbelt/lint-module-map.sh` header and `types/structure.md`; close both ledger open_issues | run 2 |
| B5 | New lints G21 `permissions-xml`, G34 `structural-tests`, G40 `logger-name`, G38 `absolute-claims` (+ G39 `count-retro-deltas`) | later |
| B6 | Issue #459 tail (rc-scan innerhtml computed-literal rows, go-rc-scan Δ3; lint-vacuous-tests report membership) + audit A7-D2 (rc-scan `status-default-ok`) | run 2 |
| B7 | Audit A9: `ledger-check.sh` + scaffold VERSION-LEDGER, T1 test backfill, slotomatic last-comment-line WARN | run 2 |
| B8 | Audit A7-D1: adapter `@NiagaraProperty defaultValue` vs pure-model Cfg default lint | later |

## Tasks
- [x] B1 · bookkeeping (docs). Route: inline (docs only, evidence already mapped).
- [x] B2 · A5 skill/doc drift. Route: inline (two doc lines).
- [x] B3 · A6 pre-push every ref + CI npm job + skip allowlist. Route: inline (writer is this session).
- [x] B4 · A4 sweep-build-state remainder. Route: inline (one Go package).
- [x] B4b · review fix-forward. Route: inline (scope added by the parent mid-run; extra surfaces authorized).
- [x] B5.0 · ng-deploy gate order (safety). Route: inline (one script + its bats + three doc lines; writer is this session).
- [x] B5.1 · recorded gaps (eslint lockfile/`npm ci`; MM7 duplicate-concern docs). Route: inline.
- [ ] B5 · G21 / G34 / G40 / G38 (+G39) new lints — later (next run).
- [x] B6 · #459 tail + A7-D2 + lint-vacuous-tests report membership. Route: inline (Go packages rcscan, reportmodule).
- [ ] B7 · A9 ledger-check + scaffold VERSION-LEDGER + T1 + slotomatic last-comment-line. Route: inline.
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
    (modularity campaign, not this unit); B3 left them, the later B4b unit fixed them (its entry below).

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

- 2026-10-27 B4b (inline; scope added mid-run by the parent with go/internal/{modulemap,lint/rules/modulemap,rcscan,
  reportmodule,lint/rules/servlet}, toolbelt/lint-servlet.sh, toolbelt/module-map.sh).
  - rc-scan (64cda653, R4/R3/R2): anchors and `vendored` now read `rcRel(rel)` (the path below the artifact dir, after
    its first `rc` segment). RED `TestMapAnchorsIgnoreAncestors`: under `srv/rc/js/services` the page fetch was exempt,
    under `checkout/rc/vendor` and `x/rc/ext` the whole module reported nothing; `TestThirdParty` covers the rc/ext and
    *.min.js branches and an rc-dir argument. The vendored skip had the same ancestor flaw (found while fixing).
    `mapRows` doc comment rewritten (+ `mapRowsUnder`).
  - add-concern / tests (e9aeb6e8): RED `TestAddConcernRollback` (dirs left, retry refused), `TestAddConcernTmpLeftover`
    (`/dev/full` symlink as the temp file), `TestTestsCellContainment` (4 entry shapes), `TestTestsStationImportLine`,
    `TestAddConcernDefaultPackage`; `TestTests` now sets STUB_EXIT with `t.Setenv` (the child reads the process env;
    `cli.Stdio` only reads single variables, so production keeps inheriting it — recorded instead of a `cmd.Env`
    that could not enumerate the injected env) and pins exit 3 for a runner exit 3 and a runner that cannot start.
    `baseMap` comment fixed; the test path is computed once. R2-row-mirror: kept — `mmrule.Row` is the exported
    read model of the tool and `row` the lint's scan record; one conversion line in `ReadMap`, no behaviour risk.
  - MM7 (a2708ddf): message counts folder names (`concern folders of 2 names (cooling, schedules: …)`); a second row
    of one concern name is an MM7 row (decision: reject explicitly rather than re-key `byRow` per row — the tool
    already treats names as unique, `findRow`); goldens `mm7-plumbing` (passes already: a coverage pin) and
    `mm7-dup-concern` (RED); TestGolden doc lists every case (20); MMB4 asserts the folder-level row.
  - lint-servlet header (8b616e55): handler-size severity `WARN/FAIL`; TC-MAP6 exists (`tests/scaffold-module.bats:577`,
    no skip) and both mutations bite (handler-size check off; strict read as legacy) — pin text names both.
  - report-module (180112c8): `moduleTree` doc name, `lerr` un-shadowed in `modularity.go`, e2e asserts ` · 1 FAIL · `.
  - Found on the way (caused by B3): the reportmodule goldens symlinked `toolbelt/eslint` whole, so an installed
    `node_modules/` (the B3 ci-local npm step) flipped 8 esl-path goldens; `toolbeltCopy` now links it without
    `node_modules/` (hermetic). Shellcheck SC2015 at `tests/module-map.bats:25,32` fixed.
  - Observed mutations (restored, `cmp`): MMA-undo-dirs, MMA-tmp, MMT-import, MMT-contain, default-package guard,
    TC-MAP6 ×2 — each turns its test red. RC-ancestor is the observed RED itself.
  - Gap: MM7's duplicate-concern rule is not yet in `toolbelt/lint-module-map.sh` (MM7 header line) nor
    `types/structure.md:298` (outside the surface; open_issue in `ledger.d/2026-10-27-kit-backlog-review-fix-forward.md`).

- 2026-10-27 B5.0 (run 2, inline; commit 5512a99f). Re-read on b2e0e430: `main` ran `copy_jars` (Step 4) before
  `verify_jar`/`verify_cachebuster` (Step 5) and `run_gate` (Step 5.5); the version/schema prechecks were already before
  the copy. Fix: verify + cache-buster + gate run on `build/libs` before `copy_jars`; usage exit 50 says nothing is
  copied. RED (bats, stubbed gradlew/unzip/gate, no station): NG-order-gate (mode A+C gate exit 1 left
  `modules/test-rt.jar`), NG-order-verify (type-count 99 and a missing `?v=` left jars), NG-order-pass (`gate ok` printed
  after `copy ok`) — 3/3 not ok on the base. GREEN: `tests/ng-deploy.bats` 51/51; 15 doc-pin suites (kit-links,
  check-skill-drift, install-skill, kit-p1-*, pre-push-live-cite, …) 224/224 ok. Docs: skill/SKILL.md:37,
  BUILD-LOOP.md:129 (§4) and :216 (§5), build-verify.md:11, :21, :276 (mode C no longer claimed to regenerate -ux).
- 2026-10-27 B5.1a (run 2, inline). Lockfile: the `package-lock.json` the B3 local install wrote (lockfileVersion 3,
  77 packages, all on registry.npmjs.org, 0 `hasInstallScript`) committed; `npm ci --offline` in a scratch copy
  installed eslint 10.11.0 + acorn 8.18.0 (no network). CI step: `npm ci --ignore-scripts` (parent fix-forward item 5,
  78abca8e R1-001), cache keyed on the lockfile; `.gitignore` and package.json description rewritten. RED CL19
  (`tests/ci-local.bats`: lockfile tracked, `npm ci … --ignore-scripts`, cache key, not ignored, lock pins = package.json
  pins) not ok on the base; GREEN ci-local 19/19; `npm_config_offline=true scripts/ci-local.sh --only 'Install the
  pinned npm tools'` PASS. Ledger open_issue in `2026-10-27-kit-backlog-a6-ci.md` closed DONE.

- 2026-10-27 B5.1b (run 2, inline; scope added mid-run by the parent: fix-forward of the APPROVED reviews of
  78abca8e, b9709db0, 39d0a155, b2e0e430; extra surfaces .githooks/pre-push, go/internal/buildstatefields,
  go/internal/citeparse). Items 1, 2, 7, 8 here; items 3, 4, 6 in the next unit; item 5 landed in B5.1a.
  - MM7 rule (decided by the parent): a duplicate is the same (concern, part) pair, case-insensitive; plumbing exempt.
    `byRow`/`byFolder` now key on the row index (owners records row idx), so rt and ux rows of one concern no longer
    merge their folders. RED on the base code (swapped in, restored): mm7-concern-two-parts, mm7-plumbing-per-part
    (false FAILs), mm7-dup-exact / mm7-dup-concern (message). GREEN `memcap go test ./internal/lint/rules/modulemap/`.
    Docs (gap b): `toolbelt/lint-module-map.sh` MM7 header, `types/structure.md` table rule ("one row per concern and
    part") + MM7 row; ledger open_issue of `2026-10-27-kit-backlog-review-fix-forward.md` closed DONE.
  - module-map station tests: `stationTest` = `\bjavax\s*\.\s*baja\s*\.` on `javasrc.Strip` + `MaskLiterals` code.
    RED `TestTestsStationFQN` (fully qualified, spaced qualified ran as pure); comment+string case stays `run`;
    skip reason now `uses javax.baja` (pins in modulemap_test.go and `tests/module-map.bats:87`).
  - module-map small: undo paths are all OS paths under the root; `expand` cleans once; `writeMap` (test) fatals on a
    missing anchor; default-package test compares a full tree dump (mutation: guard off → "the tree changed").
  - rc-scan root: `TestRootTrailingSlash` (root under an ancestor named `rc`; `x/`, `M-ux/`, `./M-ux`) RED for both
    trailing-slash roots (vendor/lib.js reported). Fix classifies on `under` (p minus the Walk prefix) via a new
    `source.rc`; row paths keep the bash port's walked spelling — golden `edge-legacy-3` pins `@DIR@/rc/a.js` for an
    `@DIR@/` argument, so normalising the row path was rejected (decision: parity golden wins). Mutation
    `vendored(rel)` → red.
  - bats: lint-module-map, module-map, rc-scan, scaffold-module, report-module 155/155 ok.

- 2026-10-27 B5.1c (run 2, inline; parent fix-forward items 3, 4, 6).
  - buildstatefields (b9709db0 R2-001/R2-002/R3-nested-deferred-mislabel/R3-nested-false-positive/R2-003/
    R3-ledgertags-coverage/R2-005): `citeparse.Pointer` gains a typed `Unterminated` flag (and the unterminated
    pointer now carries its opener's `Deferred`); `ledgerTags` reads the flag, names the real opener in label and
    text (`unbalancedTag(opener)`), and `nestedOpener` ignores an opener inside a backticked span. RED: deferred
    label (open and nested) and both backticked-opener cases not ok on the base (4 subtests); GREEN. New
    `TestLedgerTagsEnv` (unreadable fragment, unlistable ledger.d → ExitEnv naming the path; coverage, green on the
    base) and `citeparse.TestCardUnterminatedTyped`. The `retrodir/..` kit-root assumption is commented.
  - pre-push (78abca8e R4, R3-h32): the ref loop prints `pre-push: refused <local ref> -> <remote ref> (exit N)`
    before exiting; H30 asserts it names feat and not docs (RED on the base); H32 pushes two passing non-deletion
    refs (docs, docs2). Hook suites (build-retro-sync, pre-push-live-cite, changelog-gate, install-hooks) 86/86.
  - check-bats-skips (R2-last-skip-marker-drops-line, R2-multiline-pattern-implicit): the awk keeps the last
    ` # skip` that is followed by a blank or the line end (SK8, RED on the base: the line was dropped and the run
    passed); allowlist lines are passed as one `-e` each (SK9 pins per-line anchoring). check-bats-skips + ci-local
    28/28. Feature-doc B3 note reworded (R2-odd-note).

- 2026-10-27 B6a (run 2, inline). #459 tail (retro go-rc-scan Δ3): `rcscan/provenance.go` computed collections —
  `.map` over a literal collection with an arrow expression body (object literal → per-field proof, else one
  scalar), reached directly, through a const arrow (`collFnShape`) or a unique function declaration with exactly one
  `return`; `computed` accepts literals, operators, `Math.*`, cleared members, computed consts, the iterator index
  and toFixed/toString-like methods; results cached per call site so `markWrites` dirties them. RED: golden
  `truth-innerhtml-computed-1` wanted 7 rows, got 12 (lines 6-10 flagged). GREEN `memcap go test ./internal/rcscan/`;
  bats rc-scan 40/40. Mutations RC-computed-member/-const/-return/-field each red (the -return one only after the
  fixture's server-data return was moved first — the first version did not bite).
  Real-tree smoke (11 client -ux trees, total rows, old vs new binary): identical on all 11 (105/61/60/66/66/64/60/
  45/60/66/60). The six client D rows the ledger named stay: `historyRows` is a `let` filled by an imperative loop
  from a sampler that reads unit fields the live mode overwrites — not provable from tokens, kept WARN by decision;
  the open_issue in `ledger.d/2026-10-24-go-rc-scan.md` is closed with that decision.

- 2026-10-27 B6b (run 2, inline; commit 8796bb03). Audit A7 D2 intent read from the incident it names (the
  2026-09-24 phase-2 alarms retro, Δ5, and the version-defect ledger 2.6.0 row: `st = p.st || "ok"` showed an unlinked phase
  flag as active). rc-scan `status-default-ok`: `||`/`??`/`||=`/`??=` + string `"ok"` after a status name (st, stat,
  sts, status, `*Status`), token-level (strings and comments never match), WARN in every mode. RED golden
  `truth-status-default-ok-1` (no rows on the base); GREEN; mutations RC-status-name, RC-status-advisory red.
  Real-tree smoke (24 client -ux trees): 21 × 3 rows, 3 × 0; three spot-checked rows are the incident shape.
- 2026-10-27 B6c (run 2, inline; commit 36ed1f85). `lint-vacuous-tests` is a per-module report member (failOnly,
  rows, noSource `no *.java under a srcTest/` — the lint's own exit-3 text, observed on an empty root); removed
  from `notInReport`; `TestScriptsExist` 55 → 56. RED: 104 goldens failed with the member added. Re-record: the
  Go tool's output was dumped through a temporary, uncommitted hook in the golden runner (reverted with
  `git checkout`), and a script compared each old and new `want`: **every golden diff is only the new
  `(module)  PASS|SKIP  lint-vacuous-tests` row (80 PASS under stubs, 24 SKIP from the real lint) plus the summary
  PASS/SKIP counts moved by exactly that row; exit codes, stderr and verdicts are unchanged in all 104 cases.** A
  first write pass lost a `\r` (Python universal newlines; rows-site) and was redone with `newline=''`. New
  `truth-vacuous-tests-member` pins the FAIL relay; mutation RM-vacuous-nosource → 24 goldens red.
  `memcap go test ./internal/reportmodule/` ok; bats report-module + rc-scan + kit-links 128/128.

### Issue #459 closing comment (draft, NOT posted — the owner posts it)

```
All nine shapes and the comments are resolved, each either fixed or decided with evidence:

1. catch-no-400 (re-read fallback) — fixed: aceb724b (#464).
2. silent-protection — fixed: 317a7c74 (#464); the real FP was an unconditional cap below a closed `if`.
3. rc-scan innerhtml-server — fixed: a57ca89b (sink's own value), 66e06eda + d1ff5af6 (members of the page's own
   const literals, #483: 16 -> 6 rows on the reporting page), 0dc0d74a (values computed from those literals by
   .map / a const arrow / a one-return function). The 6 remaining rows on that page are a `let` filled by an
   imperative loop from a sampler that reads fields the live mode overwrites: not provable from tokens, kept WARN
   by decision.
4. rc-scan disabled-gate — fixed: 8f446ac6 (#483), a literal non-gate title on the disabled element.
5. rc-scan host-literal in an unselected mode — decided, stays FAIL (e991a8d7): a one-word mode edit makes it live;
   the page fix is a relative obix base.
6. null-context-write — fixed: 593128de (#464), station-internal own-child seeding.
7. rc-scan null-branch / browser-floor on vendored three.js — fixed: 196211a6 (#483), rc/vendor/**/*.js is left to
   lint-vendor-floor (ancestor-path follow-ups b2e0e430, f04bd0f0); browser-floor stays WARN under
   --profile unknown by decision (frontend-standard § Profiles).
8. bql-string-concat — fixed: 53d4df96 (#464), numeric-only concatenation.
9. arbitrary-ord — fixed: f19a9d1e (guarded, constant-prefixed ord) and 3566d8bf (BAlarmRecord.getSource()), #474.

Also in the issue body: report-module runs the Go-ported lints and the "no failing row" wording — bd1d2a1f (#464);
lint-vacuous-tests joined the report in 36ed1f85.

Comments:
- catch-no-400 with an upstream finite guard — decided a true positive under the per-catch contract (a latent
  silent default in a private helper; one new caller without the guard writes 0); no code change, recorded in
  retro r10-lint-fp-459 (c0e0f42c, #528). Fix in the module: let the helper throw, or pass it the validated double.
- cache-nofinger — fixed for a ?v=-conditioned max-age: 62e43586 (#528); a max-age backed only by a build-time
  token stays WARN (the servlet alone cannot prove the request carries it).
- slot-coverage per-slot "OPERATOR" read from a comment (named in the tick-off comment) — fixed: ad90770c, with
  the same shape in lint-config-sanity CS2: 4171d296 (#530).
```

## Verification (branch close, 2026-10-27)
- `memcap go test ./...`: 123 packages ok, 0 FAIL; `gofmt -l .` empty; `go vet` clean on the six touched packages.
- bats (21 touched/related suites, `-j 4 --no-parallelize-within-files`): 391 planned, 391 ok after regenerating the
  local (ignored) BUILD-STATE.md — its stale copy held a bare map-file name from a first draft of a ledger line
  (kit-links L1); its 4 skips all allowlisted by `scripts/check-bats-skips.sh`.
- Pinned shellcheck 0.10.0 (`ci-local.sh --only shellcheck`): PASS. `actionlint`: not installed on this host.
- `gen-lint-index.sh --check`, `gen-retro-index.sh --check`, `gen-build-state.sh --check`, `gen-changelog.sh --check`:
  valid. Sweep on fresh kit-state output: exit 0. `sweep-fold-audit.sh --strict --exact-stems --deltas-since
  2026-09-24`: 411/411 cited, 789 deltas 0 not cited.
- `scripts/changelog-gate.sh --range d982a704..HEAD`: PASS (4 fragments). `.githooks/pre-push` fed
  `chore/kit-backlog HEAD` over remote `d982a704`: exit 0. `git status --short`: clean.
- Engram mirror: `odd/kit-backlog-2026-10-27/tasks` (saved at close).

## Next (later blocks, exact scope)
- **B5 — new Go lints** (each one rule package under `go/internal/lint/rules/<lint>`, a thin `toolbelt/lint-*.sh`, a
  report-module member or `notInReport`, positive/negative goldens, a guard-pin, INDEX regenerated): G21
  `permissions-xml` (WARN a non-`<permissions>` root of `module-permissions.xml`, `<req-permission>` without
  `<purposeKey>`; C47), G34 `structural-tests` (WARN a srcTest class that reads `.java` as text with no BTestNg class;
  C41), G40 `logger-name` (WARN a logger name other than the exact `moduleName`; C52), G38 `absolute-claims` (report
  absolute "never"/"forbidden" claims that cite only inference; C51) and G39 `count-retro-deltas` (tool: Δ totals by
  script; C51).
- **B6 — issue #459 tail + A7-D2**: rc-scan innerhtml computed-literal rows (retro go-rc-scan Δ3);
  `lint-vacuous-tests` report-module membership; rc-scan `status-default-ok` (audit A7 D2, advisory first, real-tree
  smoke counts only). Close #459 only after both #459 items land (owner's call).
- **B7 — audit A9**: `ledger-check.sh` + scaffold VERSION-LEDGER, T1 test backfill, slotomatic last-comment-line WARN.
- **B8 — audit A7-D1**: lint for an adapter `@NiagaraProperty defaultValue` that differs from the pure-model Cfg
  default (advisory first, real-tree smoke counts only).
- Owed to the parent session: skill reinstall (`scripts/install-skill.sh --force --skill build-n4-module`) and
  `scripts/check-skill-drift.sh` (FAILed already on d982a704; `skill/SKILL.md` changed again in B2).
