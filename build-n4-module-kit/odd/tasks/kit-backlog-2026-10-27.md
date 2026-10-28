# ODD feature — Kit backlog 2026-10-27: bookkeeping, audit A4/A5/A6 remainder, review fix-forward

**Status**: B1–B9 DONE (2026-10-27, unpushed on `chore/kit-backlog`; run 3 closed B5.2–B5.8, B5.R and B8; B9 is the run-3 review fix-forward); B10 (course coverage run A) DONE 2026-10-28, B11 (run B) owed · **Created**: 2026-10-27 · **Branch**: `chore/kit-backlog`
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
| B10 | Course coverage run A (retro `course-coverage` Δ1-Δ21): 4 contradictions, doc gaps, corpus-index rows, lints L1 `charset-default`, L2 palette L17, L3 `slot-default-type` | run 4 (done) |
| B11 | Course coverage run B (retro `course-coverage` Δ22-Δ26): `build.sh --werror`, K4 `isRunning` + `stationStarted` WARN, K3 `synchronized`, K6 `javaToolchains` preflight, K11 MessageFormat `'` | later |

## Tasks
- [x] B1 · bookkeeping (docs). Route: inline (docs only, evidence already mapped).
- [x] B2 · A5 skill/doc drift. Route: inline (two doc lines).
- [x] B3 · A6 pre-push every ref + CI npm job + skip allowlist. Route: inline (writer is this session).
- [x] B4 · A4 sweep-build-state remainder. Route: inline (one Go package).
- [x] B4b · review fix-forward. Route: inline (scope added by the parent mid-run; extra surfaces authorized).
- [x] B5.0 · ng-deploy gate order (safety). Route: inline (one script + its bats + three doc lines; writer is this session).
- [x] B5.1 · recorded gaps (eslint lockfile/`npm ci`; MM7 duplicate-concern docs). Route: inline.
- [x] B5 · G21 / G34 / G40 / G38 (+G39) new lints + C51 sweep — run 3, split below (IDs continue after the run-2
  B5.0/B5.1 so no stable ID is reused; the brief's "B5.1–B5.6" maps to B5.2–B5.8).
  - [x] B5.2 · G21 `lint-permissions-xml` (C47). Route: inline (one rule package + shim + bats; writer is this session).
  - [x] B5.3 · G34 `lint-structural-tests` (C41). Route: inline.
  - [x] B5.4 · G40 `lint-logger-name` (C52). Route: inline.
  - [x] B5.R · fix-forward of the APPROVED run-2 native reviews (scope added by the parent mid-run 3; extra surfaces
    rcscan, ledgercheck, citeparse, slotomaticmarkers, check-bats-skips, build-verify.md): R1 ledger-check LC9
    fail-open + rows after a blank line + `TestMain` rename; R2 rc-scan provenance (object() progress guard,
    `declared` shadowing, `mapped` cache order, `literalName` iterator param, names); R3 check-bats-skips last line
    without newline; R4 slotomatic-doc-line blank `//` separator + noise re-check; R5 citeparse unterminated
    `[deferred:` consumers; R6 status-default-ok `||=`/`??=` goldens; R7 report-module truth case kind; R8 CL19 node
    skip; R9 build-verify.md:276 mode-A sentences. Route: inline.
  - [x] B5.5 · report-module wiring of the new module lints (G21, G34, G40, D1) + golden re-record, its own commit.
    Route: inline (one member table edit + script-verified re-record).
  - [x] B5.6 · C51 one-time "never"/"forbidden" sweep of `types/*.md` (docs commit). Route: inline.
  - [x] B5.7 · G38 `lint-absolute-claims` (C51; kit doc report, `notInReport`). Route: inline.
  - [x] B5.8 · G39 `count-retro-deltas` tool (C51; Go cmd + toolbelt wrapper). Route: inline.
- [x] B6 · #459 tail + A7-D2 + lint-vacuous-tests report membership. Route: inline (Go packages rcscan, reportmodule).
- [x] B7 · A9 ledger-check + scaffold VERSION-LEDGER + T1 + slotomatic last-comment-line. Route: inline.
- [x] B8 · A7-D1 `lint-adapter-cfg-default` (adapter `@NiagaraProperty defaultValue` vs pure-model Cfg default;
  WARN, advisory). Route: inline. Ordered BEFORE B5.5 so one golden re-record covers every new module member.
- [x] B9 · fix-forward of the APPROVED run-3 native reviews (scope added by the parent after run 3): report-module
  env-fault verdict, permissions-xml no-root / latin-1 / grant wording, adapter-cfg-default (class, name) keying,
  ledger-check repeated header, KCRD2 teeth, plus logger-name precedence pin and the part-only test/build prune.
  Route: inline (each fix is one package + its goldens/bats; writer is this session). Commits df6aaac7, f8f66cd1.
- [x] B10 · course coverage run A (retro `retros/2026-10-28-course-coverage.md`, Δ1-Δ21; audit: 70 course topics, 46
  covered, 12 partial, 10 missing, 2+2 contradictions). Route: inline (the parent's explorer already mapped every
  target with path:line; each work unit is one doc batch or one rule package + shim + bats; writer is this session).
  - [x] B10.1 · retro + this feature doc.
  - [x] B10.2 · the four contradictions (Δ1-Δ4).
  - [x] B10.3 · doc gaps (Δ5-Δ17), one cited line each.
  - [x] B10.4 · `corpus-index.md` n4-dev-course rows (Δ18).
  - [x] B10.5 · L1 `lint-charset-default` (Δ19).
  - [x] B10.6 · L2 `lint-structure` L17 palette check (Δ20).
  - [x] B10.7 · L3 `lint-slot-default-type` (Δ21).
  - [x] B10.8 · report-module wiring of L1/L3 (+ golden re-record in its own commit when any golden moves).
  - [x] B10.9 · partial fold: the retro records Δ1-Δ21 folded and Δ22-Δ26 owed; it stays `pending`.
- [ ] B11 · course coverage run B (retro `course-coverage` Δ22-Δ26, specs in its "Owed to run B" section). Route: TBD.

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

- 2026-10-27 B6b (run 2, inline; commit 18451068). Audit A7 D2 intent read from the incident it names (the
  2026-09-24 phase-2 alarms retro, Δ5, and the version-defect ledger 2.6.0 row: `st = p.st || "ok"` showed an unlinked phase
  flag as active). rc-scan `status-default-ok`: `||`/`??`/`||=`/`??=` + string `"ok"` after a status name (st, stat,
  sts, status, `*Status`), token-level (strings and comments never match), WARN in every mode. RED golden
  `truth-status-default-ok-1` (no rows on the base); GREEN; mutations RC-status-name, RC-status-advisory red.
  Real-tree smoke (24 client -ux trees): 21 × 3 rows, 3 × 0; three spot-checked rows are the incident shape.
- 2026-10-27 B6c (run 2, inline; commit 4559b1ec). `lint-vacuous-tests` is a per-module report member (failOnly,
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
lint-vacuous-tests joined the report in 4559b1ec.

Comments:
- catch-no-400 with an upstream finite guard — decided a true positive under the per-catch contract (a latent
  silent default in a private helper; one new caller without the guard writes 0); no code change, recorded in
  retro r10-lint-fp-459 (c0e0f42c, #528). Fix in the module: let the helper throw, or pass it the validated double.
- cache-nofinger — fixed for a ?v=-conditioned max-age: 62e43586 (#528); a max-age backed only by a build-time
  token stays WARN (the servlet alone cannot prove the request carries it).
- slot-coverage per-slot "OPERATOR" read from a comment (named in the tick-off comment) — fixed: ad90770c, with
  the same shape in lint-config-sanity CS2: 4171d296 (#530).
```

- 2026-10-27 B7 (run 2, inline). Audit A9 read with its sources (template, the 2026-09-26 version-defect-ledger retro, Δ1).
  - B7a (ea12be06): `ledger-check` (Go `internal/ledgercheck` + `cmd/ledger-check` + `toolbelt/ledger-check.sh`):
    LC1-LC9, contract read from `templates/VERSION-LEDGER.md` (sections, Versions header, the "`Status` is one of"
    vocabulary); `TestTemplateItself` (the template passes with only its sample-row LC9 WARN). RED: package did not
    build (no `Main`); bats LCB3 not ok (no ledger emitted). GREEN Go + bats LCB1-LCB4. Mutations LC1, LC2, LC5,
    LC6 (real-date half), LC7, LC8, LC9 each red. Scaffold: `odd/VERSION-LEDGER.md` at the gradle root for every
    type — the template's own path (one ledger per client repository) — with its sample row dropped.
    Decision: the brief's "add it to the plumbing row" does not apply: the ledger sits at the gradle root, outside
    the module root MODULE-MAP.md maps, and lint-module-map reads only part sources; fresh scaffolds of all three
    types stay clean under lint-module-map, lint-structure, lint-size (module-map.bats, lint-structure.bats) and
    ledger-check (LCB3). The scaffold byte-equality cases (TC3, TC-K8, TC-DASH1, TC-DRV1) exclude `odd/` as they
    already exclude the station-test overlay. BUILD-LOOP §7 names the tool (kit-links L5). Commit size 533 lines
    (Go package + its table test ~400): over the advisory budget, kept as one unit (tool + its only consumer).
  - "T1 test backfill": no definition exists (audit doc since its first version 41c154ff, retros, ledger.d, the
    research repo's ODD docs, git history) — nothing invented; recorded as such in the audit doc.
  - B7b (65137fb6): slotomatic last-comment-line — NOT covered by lint-slotomatic-markers before (it checks stamp
    counts, unclosed regions and TYPE/getType outside the region; none reads `//` comments above
    `@NiagaraProperty`). New WARN sub-check `slotomatic-doc-line` (last line of a ≥2-line `//` block continues the
    line above: previous line ends without . ! ? : ; or last line starts lower-case). RED golden sm-doc-line + bats
    SM9; mutation SM9 red; a first `//`-in-literal guard was unreachable (its mutation did not bite) and was removed.
    Real-tree smoke (67 client -rt parts, counts only): 0 in 4, 1-10 in 51, 27-29 in 12; spot-checks are fragments.
    Fixtures and report-module goldens unchanged.
- 2026-10-27 B5.2 (run 3, inline; commit 510ad024). G21 `lint-permissions-xml` per the spec in
  `ledger.d/2026-10-23-r8-p1-sign.md:5`: TreeRule over each module root (Mode Each, ModuleRoot), selecting
  `module-permissions.xml` + `*.java` (srcTest/test/build pruned; Java only from `-rt/src/`). XML parsed with
  encoding/xml (CharsetReader passthrough for ISO-8859-1/US-ASCII), so the template's commented requests are not read.
  RED: `undefined: Spec`; GREEN 10 goldens; bats PX 7/7; mutations PX1 (accept any root) and PX2 (raw Java lines)
  each red. Decision: a file that does not parse is exit 3 (spec: "3 usage/unreadable"). Real-tree smoke (12 client
  module roots, counts only): 10 clean, 1 FAIL (the pre-#380 `<modulePermissions>` root, a true positive), 1 exit 3
  (a module with no `-rt` part and no permissions file). Temporarily in `notInReport` until B5.5.
- 2026-10-27 B5.3 (run 3, inline; commit d4256bc7). G34 `lint-structural-tests` per `ledger.d/2026-10-23-r8-p1-tests.md:5`:
  TreeRule (Mode Each, ModuleRoot, StrictFails = the spec's "FAIL under --strict") over `*/srcTest/*` Java, grouped
  by part. Decision (spec ambiguity "reads a `.java` path"): a row is a literal ending `.java"` inside the arguments of
  `Files.readAllBytes|readAllLines|readString|lines` or `Paths.get|Path.of` — a `.java` name elsewhere in the file is
  not a read (spec case "a test reading a `.json` fixture → clean"). RED `undefined: Spec`; GREEN 8 goldens; bats ST
  5/5; ST1 (accept only BTestNgStation) and ST2 (any `.java` literal) red. A first ST1 pin (local abstract base) did
  not bite — the base itself extends BTestNg — and was replaced. Smoke (12 roots): 10 exit 0 (3 with rows, 7 WARN
  total; spot-checks: `Files.readAllBytes(new File(src, cls + ".java"))` facade-name readers, true positives), 2 exit 3
  (no srcTest Java).
- 2026-10-27 B5.4 (run 3, inline; commit 0dc0ba69). G40 `lint-logger-name` per `ledger.d/2026-10-23-r8-p1-process.md:5`.
  Decision (spec: "`moduleName` from the group/part gradle"): part gradle `moduleName.set("X")`/`moduleName = "X"`,
  else the gradle file one directory up, else the part directory name minus its suffix. Comparison is exact (case
  matters: the DebugService row is the literal). RED `undefined: Spec`; GREEN 7 goldens; bats LN 5/5; LN1
  (case-insensitive) and LN2 (no child exemption) red. Smoke (12 roots): 11 exit 0, 9 with rows, 26 WARN (lower-case
  module names), 1 exit 3 (no part Java).
- 2026-10-27 B5.R (run 3, inline; commits e9f3d653, 564d51b3). Fix-forward of the APPROVED run-2 reviews:
  - R1 ledger-check: LC9 is now the template's exact sample row only (read from the template, header and separator
    skipped); any other row with `<...>` runs LC4-LC8, a single placeholder cell WARNs and is still checked; rows after a
    blank/comment/prose line are read to the next heading (decision: keep scanning, not FAIL, as prose between rows is
    legal Markdown). `TestMain` → `TestLedgerCheckCases`. RED 3 cases → GREEN; bats ledger-check ok.
  - R2 rc-scan provenance: progress guard in `object()` (RED: the new truth case hung, `go test -timeout 30s` panic);
    `declared` only when `jssrc.Lookup` finds no local binding; `recheck()` at the end of `markWrites` re-takes every
    cached `.map` verdict (fixpoint; verdicts only narrow); `literalName` index param only on a literal-array receiver
    (Map.forEach key → not literal); `numericMethods` → `literalMethods`, `mapped(name…)` → `mapped(mapAt…)`. Truth
    `truth-innerhtml-computed-2` (3 rows); mutations RC-mapped-order, RC-declared-shadow, RC-index-receiver each red.
    Client -ux smoke (34 trees): 69 innerhtml-server rows before and after, 0 timeouts.
  - R3 check-bats-skips: `while read -r p || [ -n "$p" ]`; SK10 RED → GREEN.
  - R4 slotomatic-doc-line: blank `//` separator and an indented list-item last line skipped (SM10 RED via mutation).
    Re-check of intent: Slot-o-Matic keeps the last comment line only, and spot-checks of the hits are real fragments,
    so the rule stays; smoke over 78 client -rt parts before 7×0 8×1 6×2 13×3 11×4 6×5 1×9 14×10 6×27 6×29, after
    7×0 8×1 6×2 13×3 11×4 6×5 15×9 6×27 6×28 (76 unique hits). Decision: still noisy and cosmetic → the row is
    ADVISORY (no exit effect, `--strict` included; report-module's failWarn filter no longer relays it).
  - R5 citeparse consumers: only `buildstatefields` (checks `Unterminated` before anything) and `citeresolve`
    (appends a "deferred" note, never skips) read `Deferred`; neither skips first. New `tests/kit-cite-resolve-deferred.bats`
    KCRD1-2 runs the real resolver: an unterminated `[deferred:` tag is an UNPARSED strict finding (exit 1). A Go test in
    `go/internal/citeresolve/` was outside the surfaces, so the pin is the bats file.
  - R6 status-default-ok: `truth-status-default-ok-2` (`||=` and `??=` flagged; non-status name and non-"ok" default clean).
  - R7 report-module: new `truth-vacuous-tests-envfault` (exit 3 non-noSource text → `(module) ERROR lint-vacuous-tests
    env fault (exit 3)`, exit 3; the summary verdict text reads CLEAN, as in every existing member env-fault golden);
    the member case's @kind narrowed; Cases 121 → 122.
  - R8 CL19: `command -v node || skip "node not installed"`; NOT allowlisted (CI's bats job runs setup-node, so a CI skip
    must fail the gate).
  - R9 build-verify.md:276: rewritten to mode A only (-ux when annotated), mode C -rt only, mode B never (two WARN
    lines with `--with-slotomatic`); "silently" and "Mode B is the exception" removed.
- 2026-10-27 B8 (run 3, inline; commit ccfe6e90). Audit A7-D1 `lint-adapter-cfg-default`. The audit line names only
  "adapter `@NiagaraProperty defaultValue` vs pure-model Cfg default"; decisions recorded for that ambiguity: a Cfg is
  any class named `*Cfg`; the adapter↔field link is an assignment statement `<x>.<field> = get<Slot>();` (optionally
  `.getMillis()`/`.getSeconds()` for a BRelTime slot, converted); the field default is its literal initializer or Java's
  0/false; computed initializers, multi-declarator fields, other right-hand sides and unevaluable defaults are not
  judged; WARN in every mode (advisory first). RED `undefined: Spec`; GREEN 6 goldens; bats ACD 4/4; ACD1 (raw ms, no
  getSeconds conversion) and ACD2 (no-initializer field unjudged) red — a first ACD1 pin (`BRelTime.make(1500)` vs
  1500 ms) could not bite and was replaced. Smoke (12 client module roots): 2 with rows, 24 WARN (fields left at 0/false
  while the slot defaults to a commissioned value), 9 clean, 1 exit 3. Audit A7 ticked in `audit-2026-10-03.md`.
- 2026-10-27 B5.5 (run 3, inline; commit fd3a0080). report-module members `lint-permissions-xml` (failWarn, PASS
  verdict), `lint-structural-tests`, `lint-logger-name`, `lint-adapter-cfg-default` (warnOnly), each with its own
  noSource SKIP; removed from `notInReport`; `Scripts()` 56 → 60. RED: 106 goldens failed. Re-record: a temporary,
  uncommitted `GOLDEN_DUMP_TMP` hook in `golden.Runner` (reverted with `git checkout`) dumped the Go tool's output; a
  script replaced each `want` (read/written with `newline=''`) and checked every case: exit and stderr identical, all
  non-member lines unchanged and in order, summary counts moved by exactly the added rows, verdict unchanged — 106
  changed, 0 problems; added permissions-xml 97 PASS + 9 SKIP, structural-tests 24 SKIP, logger-name 2 SKIP,
  adapter-cfg-default 9 SKIP. Mutation RM-new-members-nosource is pinned by those SKIP goldens. Decision: one re-record
  for all four members (B8 ordered before B5.5) instead of one per lint.
- 2026-10-27 B5.7 (run 3, inline; commit 56c87f56). G38 `lint-absolute-claims` per `ledger.d/2026-10-23-r8-p1-process.md:5`.
  Decision (spec is per line, tags often sit on a continuation line): a line's evidence is its claim unit (bullet or
  paragraph). Strong = a `code`/`course`/`devguide`/`CERT`/`javap` tag or a `.java:<n>` cite; one INFO row per line
  (first word). Spec: exit 0 always (INFO never affects the exit), 3 usage/empty root; ModuleRoot so `--all` skips it;
  `notInReport` (kit doc report). RED `undefined: Spec`; GREEN 5 goldens; bats AC 5/5; AC1 (line-only tags) and AC2
  (corpus as strong) red. Kit `types/`: 423 rows (282 never, 58 throws, 44 silently, 34 always, 5 forbidden).
  Order: G38 landed before the sweep because the sweep consumes its report (brief order sweep → G38 swapped).
- 2026-10-27 B5.6 (run 3; commit 29fc315a; review delegated to one read-only worker, then spot-checked here). The 287
  never/forbidden rows were checked against corpus B1242 §1242.3 (rules 1-73) and §1242.4 (items 1-16): 1 contradicted
  (CoalesceQueue proxy `write()` returned `false`; B1239 §1239.9 quotes the devguide "if the write is enqueued then
  return true" — spot-checked), 2 over-absolute corrected (lexicon keys vs `<Type>.displayName`, B1236 §1236.3; BQL
  `get()` comment, B1237 §1237.7), 1 proposed change NOT applied (`types/moduleTest.md` test types leaking into
  module-include.xml "because of the package layout": B1236 §1236.5 shows the course layout and a shipped
  `moduleTest-include.xml`, not that cause — inferred), 15 supported, ≈268 outside what the course checklist covers
  (dashboard, browser, field behaviour). Not done: adding the 15 corpus pointers to supported lines (they would not
  change G38's verdict, corpus tags are not strong) — optional follow-up.
- 2026-10-27 B5.8 (run 3, inline; commit a23fcef3). G39 `count-retro-deltas` per the spec in the same fragment: Go cmd +
  `internal/retrodeltas` + `toolbelt/count-retro-deltas.sh`, named in BUILD-LOOP §7 and METHODOLOGY K34 (kit-links L5).
  Decisions: the delta section is `## Proposed kit deltas`, `## Deltas` or `## Proposed delta table` (the three headings
  the 410 kit retros use); the INDEX cell is the column headed `Deltas` (some rows carry a note column after it); a value
  not stated prints `-` and is a mismatch; no INDEX.md beside a retro is exit 3. RED `undefined: Main`; GREEN 9 unit
  cases; bats CRD 3/3. Kit retros: TOTAL rows=1331, files=410, 93 mismatches (74 legacy without `**Delta count**`, 19
  real disagreements) → open_issue in `ledger.d/2026-10-27-kit-backlog-b5-count-retro-deltas.md` (retros/ is outside the
  surface). Also: `go/README.md` rows for `internal/ledgercheck/` + `cmd/ledger-check/` (owed open_issue closed in
  place) and the `toolbelt/ledger-check.sh` LC9 header line for the narrowed rule.
- 2026-10-27 run-3 close fixes (final docs commit): the G39 ledger fragment named `<retro.md>`, which the generated
  BUILD-STATE carried into a kit-links L1 dangling reference — reworded `<retro-file>`; `tests/lint-absolute-claims.bats`
  SC2016 (backticks of a Markdown fixture) annotated for the pinned shellcheck.

- 2026-10-27 B9 (run-3 review fix-forward, inline; commits df6aaac7, f8f66cd1).
  - report-module (e9f3d653 R4): a member env fault with no FAIL read verdict CLEAN while exiting 3. Decision: the
    verdict word is the existing `INCOMPLETE` (a member did not run; a FAIL still reads `ISSUES`, exit precedence
    unchanged). RED: `truth-vacuous-tests-envfault` with INCOMPLETE failed. Nine more goldens had the same contradiction
    (adv-exit-reasons, esl-kit-relative, lint-exit3-error, lt-exit3, tc-exit3, truth-sc-no-pct, vm-args-exit3,
    wp-env-empty, wp-env-first-line); a script compared each old/new case line by line: 10 files, the only changed line
    is the summary line and only its last word CLEAN -> INCOMPLETE. `sum-error-not-counted` and `wp-no-matrix` hold an
    ERROR-like text but exit 0 (no env fault) and stay CLEAN. Unit case "env alone" now INCOMPLETE (+ a --lint-only case).
  - ledger-check (e9f3d653 R3/R4): a header row repeated in the Versions section is skipped like the first header;
    separator rows were already skipped anywhere. RED 2 cases (LC5/6/7 on the column names) → GREEN. A second table
    with another header still FAILs LC4 on its rows (kept: a foreign table inside Versions is a real defect).
  - KCRD2 (564d51b3): asserts the resolver file exists, exit 0, the resolved `[deferred]` MISSING_FILE row and its count.
    RED: the old test passed with the resolver path pointed at a missing script (exit 127); the new one fails. Renamed to
    what it checks (the old name claimed "no finding" while the pointer is MISSING_FILE).
  - permissions-xml (510ad024 R3, R4-001, R2): no root element (empty, declaration-only, comment-only) → FAIL at line 1
    (same severity as a wrong root: the file grants nothing), golden `no-root` + bats PX-noroot (guard pin); ISO-8859-1 /
    US-ASCII decoded byte → rune (no golang.org/x), any other declared encoding exit 3 naming it (TestLatin1Decoded RED:
    "invalid UTF-8"; bats PX-latin1); the absolute-path row says "without a FILE_READ or FILE_WRITE grant for it".
  - adapter-cfg-default (ccfe6e90 R2/R3): fields keyed by (Cfg class qualified name, field), slots by (owner class,
    slot); the receiver's type is its `T x` declaration (field, local or parameter) in the link's top-level class,
    matched short, nested-qualified or package-qualified; the slot is the one on a class enclosing the statement, else
    the part's only slot of that name (inherited); anything unresolved or ambiguous is not judged (no row). RED goldens
    `same-field-two-cfgs`, `same-slot-two-components` (row missed by walk order); bats ACD3 (guard pin).
  - logger-name (0dc0ba69 R3): part gradle vs group gradle disagreement pinned (`part-over-group`, part wins — passed
    as is). The `test`/`build` prune by NAME was a real bug: a Java package `com/acme/test` under src/ was skipped
    (logger-name exit 3 on such a part). The three B5 module lints now prune `srcTest` by name and only a part's own
    `<part>-{rt,ux,wb,se}/{test,build}` by path; goldens `package-named-test` (logger-name, permissions-xml) and
    `package-named-build` (adapter-cfg-default) RED under the old prune; bats LN3 (guard pin). The package-qualified
    receiver case surfaced a resolution gap in the new adapter matching, fixed before commit.
  - Smoke (9 client module roots, old binary at d888f5a1 vs new): adapter-cfg-default 24 rows, logger-name 24,
    permissions-xml 1 — byte-identical outputs.

## Verification (run 2 close, 2026-10-27, tip 65137fb6 + docs)
- `memcap go test ./...`: 124 packages ok, 0 FAIL (EXIT 0); `gofmt -l .` empty; `go vet` clean on rcscan,
  reportmodule, ledgercheck, cmd/ledger-check, slotomaticmarkers, lint/rules/modulemap, modulemap,
  buildstatefields, citeparse.
- bats (29 suites incl. ng-deploy, rc-scan, report-module, scaffold-module, lint-module-map, module-map,
  ledger-check, kit-links, lint-slotomatic-markers, ci-local, check-bats-skips, build-retro-sync, doc pins;
  `-j 4 --no-parallelize-within-files`): 479 planned, 479 ok; `scripts/check-bats-skips.sh` on that TAP: 4 skips,
  all allowlisted.
- Pinned shellcheck 0.10.0 (`scripts/ci-local.sh --only shellcheck`): PASS. Guard pins: SM9, LCB1 MATCH.
- `gen-lint-index.sh --check`, `gen-retro-index.sh --check`, `gen-build-state.sh --check`: valid.
  `sweep-fold-audit.sh --strict --exact-stems --deltas-since 2026-09-24`: 411/411 cited, 789 deltas 0 not cited,
  exit 0. `kit-cite-resolve --strict` on skill/SKILL.md, BUILD-LOOP.md, build-verify.md, types/structure.md,
  types/logic-authoring.md: LIVE 0, HISTORY 0 findings.
- `scripts/changelog-gate.sh --range b2e0e430..HEAD`: PASS (9 fragments).
- `.githooks/pre-push` fed `chore/kit-backlog` over b2e0e430: the first pass was refused by `client-guard` (13 new
  lines named two retro stems that carry client names, rules #5 and #25 of the private denylist). The 6 unpushed
  commits that added them were rewritten in place (`git filter-branch --tree-filter`, exact-string rewording to a
  dated retro description; 5512a99f..0dc0d74a unchanged), the backup ref removed, and the SHAs above updated; the
  two remaining matches are pre-existing tags on edited lines, exempted by client-guard as same-hunk.

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
  script; C51). Adding a report member re-records the module-member goldens: reuse the B6c procedure (dump through a
  temporary, uncommitted runner hook; script-verify only the new row and the counts moved; read with `newline=''`).
- **B8 — audit A7-D1**: lint for an adapter `@NiagaraProperty defaultValue` that differs from the pure-model Cfg
  default (advisory first, WARN in every mode; real-tree smoke counts only), then tick A7 in `audit-2026-10-03.md`.
- Issue #459: the closing comment above is ready; posting it and closing the issue is the owner's call.
- Owed outside this branch's surfaces: `go/README.md` rows for `internal/ledgercheck/` + `cmd/ledger-check/`
  (open_issue in `ledger.d/2026-10-27-kit-backlog-b7a-ledger-check.md`); skill reinstall
  (`scripts/install-skill.sh --force --skill build-n4-module`) and `scripts/check-skill-drift.sh` (skill/SKILL.md
  changed again in B5.0).

## Verification (run 3 close, 2026-10-27)
- `memcap go test ./...`: 130 packages ok, 0 FAIL (EXIT 0); `gofmt -l .` empty; `go vet` clean on lint/..., reportmodule,
  rcscan, ledgercheck, retrodeltas, cmd/count-retro-deltas.
- bats (27 touched/related suites, `-j 4 --no-parallelize-within-files`, TAP): 344 planned, 344 ok after the L1 fix;
  `scripts/check-bats-skips.sh` on that TAP: 4 skips, all allowlisted.
- Pinned shellcheck 0.10.0 (`scripts/ci-local.sh --only shellcheck`): PASS (after the SC2016 annotation). Guard pins:
  PX1-2, ST1-2, LN1-2, ACD1-2, AC1-2, SM9-10 MATCH.
- `gen-lint-index.sh --check`, `gen-retro-index.sh --check`, `gen-build-state.sh --check`: valid. Fresh kit-state sweep:
  exit 0. `sweep-fold-audit.sh --strict --exact-stems --deltas-since 2026-09-24`: 411/411 cited, 789 deltas, 0 not cited.
  `kit-cite-resolve --strict` on types/structure.md, dashboard.md, logic-authoring.md, BUILD-LOOP.md, METHODOLOGY.md:
  LIVE 0, HISTORY 0 findings.
- `scripts/changelog-gate.sh --range 81ae3347..HEAD`: PASS (10 fragments). `.githooks/pre-push` fed `chore/kit-backlog`
  over 81ae3347: the first pass was refused by `client-guard` (9 lines of the G40 golden and test used a client module
  name, rule #24 of the private denylist); the 11 run-3 commits were rewritten in place (`git filter-branch
  --tree-filter`, the name replaced by a generic `AcmePan`; 510ad024 and d4256bc7 unchanged), the backup ref removed,
  and the SHAs above updated.

## Backlog close (2026-10-27)
B1-B8 are DONE on `chore/kit-backlog` (unpushed): B1-B4, B4b, B5.0, B5.1, B6, B7 in runs 1-2; B5.2-B5.8, B5.R and B8 in
run 3 (commits 510ad024..HEAD). Audit `audit-2026-10-03.md` A7 is closed; A9 closed in run 2.
What is left, and why:
- 19 kit retros whose delta rows, `**Delta count**` and `deltas` header disagree, and 74 legacy retros without a Delta
  count line (`count-retro-deltas` output): reconciling them edits `retros/`, outside this backlog's surfaces (open_issue).
- C51: the 15 corpus pointers for supported absolute claims (optional) and helper G93 `diff-vs-reference`; the
  `types/moduleTest.md` leak cause stays as written until certified evidence exists.
- C41: whether 4.14+ tooling still discovers the older test layout stays `[INFER]`.
- C52: the client parity ticket items are client follow-ups, not kit work.
- From run 2, unchanged: posting issue #459's closing comment (owner), the skill reinstall + `check-skill-drift.sh`.
- The four new module lints are WARN/advisory (G21 FAILs per its spec); promoting any of them to FAIL is a later decision
  once their real-tree counts settle.

## Verification (B9 close, 2026-10-27, tip f8f66cd1)
- `memcap go test ./...`: 130 packages ok, EXIT 0; `go vet ./...` exit 0; `gofmt -l .` empty.
- RED per GREEN: listed per fix under Progress B9.
- bats `-j 4 --no-parallelize-within-files` report-module, ledger-check, kit-cite-resolve-deferred, lint-permissions-xml,
  lint-adapter-cfg-default, lint-logger-name, guard-pins, gen-lint-index: 128/128 ok; `check-bats-skips.sh` 2 skips, all
  allowlisted. `lint-guard-pins.sh --strict <repo>`: exit 0, 365 MATCH (PX-noroot, LN3, ACD3 included).
- `gen-lint-index.sh --check`, `gen-retro-index.sh --check`, `gen-build-state.sh --check`: exit 0.
  `sweep-fold-audit.sh --strict --exact-stems --deltas-since 2026-09-24 retros/INDEX.md <kit>`: 411/411 cited, 789
  deltas, 0 not cited. `scripts/ci-local.sh --only shellcheck`: PASS.
- `scripts/changelog-gate.sh --range d888f5a1..HEAD`: PASS (2 fragments). `.githooks/pre-push` fed `chore/kit-backlog`
  over d888f5a1: exit 0.

## Follow-ups (advisory, not scheduled)
Remaining run-3 review findings, recorded, not fixed (all advisory):
- rcscan R3-recheck-scalar-untested — `go/internal/rcscan/provenance.go:222-226`: the `recheck()` path for a scalar
  (non-`.map`) verdict has no test.
- absoluteclaims R2-001 — `go/internal/lint/rules/absoluteclaims/testdata/spec-cases.txt:19` (readability finding of
  the run-3 review on that case; wording per the review capture).
- absoluteclaims R3-table-fence-units — `go/internal/lint/rules/absoluteclaims/absoluteclaims.go:87-99`: how claim units
  end at Markdown tables and code fences is not pinned by a golden.
- retrodeltas R3-header-mismatch-unproved — `go/internal/retrodeltas/retrodeltas_test.go`: a mismatch from the deltas
  header alone is not proved by a case.
- retrodeltas R3-multi-file-total-unproved — `go/internal/retrodeltas/retrodeltas_test.go`: the TOTAL line over several
  retro files is not proved by a case.
- ledger-check: a second table with a different header inside `## Versions` FAILs LC4 on each of its rows (kept as a
  real defect; revisit if ledgers legitimately carry side tables there).
- B9 review (f8f66cd1, approved) R4-001/R3-001 — `go/internal/lint/rules/permissionsxml/permissionsxml.go:43`: the
  anchored prune no longer skips a `build/` or `test/` directory that is not directly under a part (for example a
  group-level `build/`), so generated sources there are now scanned; decide whether to prune those by path too.
- B9 review R2-001 — `go/internal/lint/rules/adaptercfgdefault/adaptercfgdefault.go:41-43`: the part-output prune list
  and its comment are copied into three rule packages; move them to one shared helper in `go/internal/lint`.
Fixed in B9 and dropped from this list: permissions-xml R2 FILE_WRITE wording, loggername R3 module-name precedence
(pinned) and R3 prune package segments, KCRD2 naming.

## Progress / evidence — B10 course coverage run A (2026-10-28)
Route: inline for every unit (the parent's read-only explorer had mapped each target with path:line; each unit is one
doc batch or one rule package + shim + bats, writer is this session). Retro `retros/2026-10-28-course-coverage.md`
(pending, 26 deltas; audit counts 70 topics / 46 covered / 12 partial / 10 missing / 2+2 contradictions).
| Unit | Commit | Evidence |
|---|---|---|
| B10.1 retro + feature doc | 9637f758 | `gen-retro-index.sh --check` valid; row `pending 26` |
| B10.2 contradictions (Δ1-Δ4) | c5cd9a08 | `kit-cite-resolve --strict` on the 4 docs: LIVE 0 |
| B10.3 doc gaps (Δ5-Δ17) | 6ae82b23 | `kit-cite-resolve --strict` on the 10 docs: LIVE 0. Targets moved from the brief where the kit had a better home: copy/paste id → `types/logic.md` § Flags; relation IDs / SystemDB → `types/logic-authoring.md` (the kit has no tagging § in structure.md); restart rule → `BUILD-LOOP.md` § 6 (deploy) |
| B10.4 corpus-index (Δ18) | 92b19b05 | B1300, B1302, B1354, B1416-B1420 rows; strict cite LIVE 0 |
| B10.5 L1 `lint-charset-default` (Δ19) | 74031e4c | RED undefined Spec → GREEN 7 goldens; bats CD 6/6; mutations CD1-CD3 red. Smoke: 20 client module src roots 0 WARN, 1 K20 ERROR (empty `-wb/src`); first smoke found a false positive (a name typed `byte[]` in one method and `char[]` in another), fixed before the commit; vendor spot check (baja, web-rt) rows all real |
| B10.6 L2 `lint-structure` L17 (Δ20) | ea65724b | RED l17-1, l17-malformed-1 → GREEN 47 goldens; bats lint-structure 38/38; mutations LS17a, LS17b red. Smoke: 10 client module roots 0 L17 rows; 729 vendor palettes: one real type named `BFormatString` → the foreign-module B-prefix WARN was dropped |
| B10.7 L3 `lint-slot-default-type` (Δ21) | 323184d3 | RED undefined Spec → GREEN 4 goldens; bats SDT 5/5; mutations SDT1, SDT2 red. Smoke: 123 client source roots 0 FAIL; docSource + the whole organized tree 0 FAIL (a first draft FAILed 356 vendor int-on-double rows; ComplexIntrospector widens them, so the rule was corrected) |
| B10.8 report-module members | 3bfa156c | 26 goldens re-recorded and script-checked (only new member rows + summary counts; verdict/exit/stderr identical); bats report-module 75/75 |
| B10.9 partial fold | this commit | retro § Status lists Δ1-Δ21 by commit, Δ22-Δ26 owed; header stays `pending` |

Decisions (recorded, not asked): L1 and L3 report members carry no verdict row (as `lint-service-types`), so a clean
pass adds no golden churn; L17 identical `m=` repeats stay clean (decoder evidence) although the brief listed them.

## Verification (B10 close, 2026-10-28)
- `memcap go test ./...`: 132 packages ok, EXIT 0; `go vet ./...` exit 0; `gofmt -l .` empty.
- bats `-j 4 --no-parallelize-within-files` lint-charset-default, lint-slot-default-type, lint-structure,
  report-module, guard-pins, gen-lint-index, kit-links: 164/164 ok; `check-bats-skips.sh` 2 skips, all allowlisted.
- `lint-guard-pins.sh --strict`: exit 0 (CD1-3, LS17a/b, SDT1-2 MATCH). `scripts/ci-local.sh --only shellcheck`: PASS.
- `gen-lint-index.sh`, `gen-retro-index.sh`, `gen-build-state.sh --check`: exit 0. `sweep-fold-audit.sh --strict
  --exact-stems --deltas-since 2026-09-24`: 411/411 cited, 789 deltas, 0 not cited (the new retro is pending).
- `kit-cite-resolve --strict` on every edited doc: LIVE 0, HISTORY 0.
- `scripts/changelog-gate.sh --range 81db90d8..HEAD`: PASS (7 fragments).

## Next (B11, run B)
Δ22-Δ26 of `retros/2026-10-28-course-coverage.md` § Owed to run B: `build.sh --werror`, K4 `isRunning` guard +
`stationStarted`-only WARN, K3 `synchronized`, K6 `javaToolchains` preflight, K11 MessageFormat `'`. Then flip the
retro to `folded`.
