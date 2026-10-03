# ODD feature — Deferred lint candidates and the polish close-retro fold

**Status**: COMPLETE (2026-10-03) · **Created**: 2026-10-02 · **Owner**: ODD orchestrator · **Repo**: angeles725/niagara-tools (main, PUBLIC)

## Objective
Fold the 5 deltas of the polish close retro (`retros/2026-10-02-polish-2026-10-02-close.md`), close the one real gap
on #226 (a `find` error silently skips files in six lints), and implement the lint candidates that earlier folds
recorded as DEFERRED in `BUILD-STATE.md` (`lint-size`, `lint-license-isoperational-gate`, `lint-set-null-ord`) plus the
project-level `split-package-check` of #142. Close with release v0.31.0.

## Problem / why
- The polish close retro is `pending`: its lessons (anti-cascade policy, structural parser rules, two shell traps, the
  `candidate_context_unavailable` recovery) live only in the retro until they reach doctrine.
- Six lints read `find … | sort` through a process substitution, so a sub-directory that `find` cannot enter is
  skipped and the lint reports the module clean. That is a fail-open, already true before the polish feature.
- Three lint candidates were evaluated and recorded but never built, each with a corpus-backed failure mode
  (an oversized class, a licensed component acting while unlicensed, a null ORD written into a slot). #142 has the
  same status for the split-package shadowing gotcha.

## Authorization & delivery
- Operator pre-authorized on 2026-10-02: ODD + RDD (consent granted), commit + push + PR + merge + issue
  comments/close + tags, chained and automatic.
- Delivery: one PR per task, branched from the current `origin/main`, merged after CI green (+ RDD when due) before
  the next starts. The ~400-line budget is advisory.
- Repository is PUBLIC: new text uses generic names (no customer, site, host, IP or user names).
- ANTI-CASCADE POLICY (carried over from `odd/tasks/polish-2026-10-02.md`, now doctrine in `ORCHESTRATION.md` §7):
  only a review finding that shows a fail-open (a check that can report PASS on a real defect) or a regression
  introduced by this feature becomes a sub-task. Every other advisory is posted on #226 and listed under
  "Parked advisories" below; no new sub-task is created for it.

## Scope
In: the toolbelt scripts, lib helpers, bats, fixtures, docs, ledgers and the release named by the tasks below.
Out: wiring `split-package-check` into a client repository's CI (that repository is outside this kit); doctrine not
traceable to the close retro, #226 or #142.

## Tasks
- [x] D1 · fold the 5 deltas of `retros/2026-10-02-polish-2026-10-02-close.md` — Δ1 anti-cascade policy up front
      (`ORCHESTRATION.md` §7), Δ2 structural rules for heuristic parsers (`METHODOLOGY.md` § Conformance rules), Δ3
      bash 5.2 `patsub_replacement` (`CONTRIBUTING.md` §2 + `ORCHESTRATION.md` §3.c, mechanical guard
      `tests/shell-hygiene.bats` SH1), Δ4 bats `! cmd` (`CONTRIBUTING.md` §2 + `ORCHESTRATION.md` §3.c; gated by
      shellcheck SC2314 in CI), Δ5 `candidate_context_unavailable` recovery (`ORCHESTRATION.md` §4) — and the #226 lead
      item: the six T3/T4/T5a lints fail closed (exit 3) when `find` cannot enter a sub-directory, through a shared
      `toolbelt/lib/scan-files.sh`. Retro flipped `folded`. Route: inline (the parent is the bounded writer for this
      feature; one lib helper + six mechanical edits + docs).
- [x] D1b · D1 review fail-opens (posted on #226) — R4-dot-root-prune: `lib/scan-files.sh` pruned a `.`, `..` or
      dot-named ROOT whole (pre-existing expression, a fail-open) → `-mindepth 1`; R3-patsub-multiline-skip: the SH1
      checker passed an expansion split over lines (introduced fail-open) → follow it onto the next lines, report an
      unterminated one; R2-patsub-backslash-in-single-quote (could hide a hit) → `'...'` is literal; sort's stderr goes
      to the err file (R2/R3 sort-err, cheap, same lines). Route: inline (one lib, one test helper, two bats).
- [x] D1c · D1b review fail-open (posted on #226) — R3/R4 file-root: with `-mindepth 1` a FILE root lists nothing and
      `scan_files` returned 0 (no current caller reaches it: each rejects a non-directory first) → return 2 with "not a
      directory"; plus the MAX_SPAN comment (R2-maxspan-unexplained, one line). Route: folded into the D2 PR.
- [x] D2 · `lint-size.sh` — advisory WARN-only class/method size smell (BUILD-STATE DEFERRED spec). Route: inline
      (one new lint + its bats + the report-module member; the parent is this feature's writer).
- [x] D2b · D2 review fail-open (posted on #226) — R3-002: `lint-size.sh` merged a BEGIN inside an open region and let a
      stray END close an unterminated one, hiding its lines from the count → `nested-region` / `stray-end` WARN rows.
      With it, the R4/R2 `advisory_member` ERROR row gains the member's first output line (D3 extends that helper).
      Route: folded into the D3 PR.
- [x] D3 · `lint-license-isoperational-gate.sh` — a licensed class acting in `changed()`/timer/servlet-write callbacks
      without an `isOperational()`/`isFault()` gate. Route: inline.
- [x] D3b · D3 review fail-open (posted on #226) — R3-002: a gate call anywhere in the callback body counted, including a
      check after the acting statements and another object's `x.isOperational()` → the gate must be unqualified (or
      `this.`) in the callback's first statement; with it (same helper / same lint): the `advisory_member` ERROR reason
      skips the member's own WARN/FAIL/ADVISORY rows (R4/R2/R3-001), the row names isFatalFault() too (R2), LIG3/4/6
      assert the exit status (R3-003). Route: folded into the D4 PR.
- [x] D4 · `lint-set-null-ord.sh` — a `setXxxOrd(getSlotPathOrd())` / ORD `set(...)` with no null guard. Route: inline.
- [x] D4b · D4 review fail-open (posted on #226) — R2-004: one null check on ANY receiver silenced a later direct pass
      of ANOTHER receiver's ORD → guard per receiver; with it, the §H3 coverage marker that still said "lint DEFERRED"
      (R2-001) and the setter-name comment that claimed the last setter (R2-002/R3-002). Route: folded into the D5 PR.
- [x] D5 · #142 `split-package-check.sh` — project-level check: one Java package declared in two modules. Route:
      inline. Conservative read-only advisory version (see "Open decisions").
- [x] D5b · D5 review fail-opens (posted on #226) — R4-001: the `sort | awk` aggregation of `split-package-check.sh`
      was unchecked (a failed sort → no rows → exit 0, even under `--strict`) → each stage checked, exit 3; R2-001 /
      R3-001: a call-chain or indexed receiver collapsed to the key `.` in `lint-set-null-ord.sh` → only a plain
      identifier chain is recorded as guarded; with it R2-002 (mutation text) and R3-003 (SPC5 status assert). Route:
      inline, own PR (keeps the release PR free of code).
- [x] D6 · close — release v0.31.0 (VERSION, CHANGELOG, tag on the confirmed merge SHA), skill reinstall + drift
      check, feature retro, this doc COMPLETE. Route: inline. CHANGELOG also folds, verbatim, the mcp-n4-kit
      v0.5.1-v0.7.3 entries of PRs #229, #231, #232, #234, #236, #238, #240, #241, #243 (coordinator request).

## Acceptance
- Every task lands as its own merged PR with RED → GREEN evidence where a runnable deterministic test exists, an
  observed mutation per new guard (Mutation IDs equal bats test IDs), and its ledger entry.
- Each new lint grounds its detection rule in cited evidence (corpus block or decompiled source file:line), ships
  positive and negative fixtures, a guard-pin, a regenerated `toolbelt/INDEX.md` and, when it runs per module, its
  `report-module.sh` wiring.
- #226 lead item and #142 closed or commented with a mapping to the PR that resolved them.

## Checks (per task)
`bats tests/*.bats` serially (no GNU parallel) · `python3 -m unittest discover -s mcp-n4-kit/tests` ·
shellcheck per `.github/workflows/ci.yml` · `sweep-build-state.sh` · `sweep-build-state.sh --age --today <date>` ·
`sweep-fold-audit.sh --strict` · `sweep-fold-audit.sh --strict --deltas-since 2026-09-24` ·
`lint-guard-pins.sh --strict .` · `gen-lint-index.sh --check`.

## Progress / evidence
- 2026-10-02 D1 (branch `feat/dl-d1-fold-and-find-fail-closed`, from `origin/main` 255f2d9). Route: inline — the parent
  session is this feature's bounded writer; one lib helper, six mechanical lint edits, docs.
  Evidence:
  - RED: CHW-finderr, PHW-finderr, SSL-finderr, WEO-finderr, ICO-finderr, SPR-finderr not ok before the change (a
    `chmod 000` sub-directory: `find: … Permission denied` on stderr, exit 0, its files skipped).
  - GREEN: the six lint suites all ok (69 tests) after `toolbelt/lib/scan-files.sh` (find status kept, list sorted to a
    file) replaced the process-substitution walk; exit-contract cells now say "or a sub-directory find cannot enter".
  - Observed mutations (restored byte-identical, `cmp`): `scan_files … || true` in each lint → its `<ID>-finderr` not ok
    (CHW, PHW, SSL, WEO, ICO, SPR); SH1-neg (checker skips the replacement part) → SH1-neg not ok.
  - Δ3 guard: `tests/shell-hygiene.bats` SH1/SH1-neg/SH1-pos with `tests/helpers/patsub-check.py` (structural: pattern
    ends at the first unquoted depth-0 `/`, replacement at the matching `}`); the kit scripts are clean today. Δ4 is
    already a mechanical gate: shellcheck SC2314 (style) fires on a bats `! cmd` line and CI runs every severity.
  - `bats tests/*.bats` (serial): 975 ok / 0 not ok (67 env skips), count 975, exit 0. mcp-n4-kit unittest: 484 OK.
  - shellcheck 0.11.0: only the pre-existing SC2329 info in `lint-config-sanity.sh`.
  - `sweep-build-state.sh` exit 0; `--age --today 2026-10-02` exit 0; `sweep-fold-audit.sh --strict` 196/196 cited;
    `--deltas-since 2026-09-24` 145 checked, 0 not cited; `lint-guard-pins.sh --strict .` exit 0;
    `gen-lint-index.sh --check` fresh (regenerated for the six exit cells).
  - Commit 619f61a, PR #230, merge 49185d4. RDD: consent granted (operator pre-authorized), 4 lenses APPROVED and
    acknowledged — lineage review-d02202bd55feff67, authority burned. The first capture round returned
    `invalid_request` (mutation not started): the bound STATUS had been reached from the assess-returned preflight,
    which carries no `--agent`, so the capture tokens had none. Re-running the canonical preflight STATUS with
    `--agent claude-code` returned the same lineage's START; its exact replay (`replayed`) bound the agent and the
    same four slots were reoffered with `--agent`. 5 informational findings posted on #226 → task D1b.
  - Observation (not a sub-task, anti-cascade): 12 other lints still walk with `find … 2>/dev/null`; same class, not
    introduced by this feature, outside the #226 item — recorded on #226.
- 2026-10-02 D1b (branch `feat/dl-d1b-scan-root-and-patsub`, from `origin/main` 49185d4). Route: inline.
  - RED: SH1-multiline, SH1-unterminated, SH1-squote, SF1, SF2, SF3 not ok before the change.
  - GREEN: `bats tests/shell-hygiene.bats tests/scan-files.bats` 9 ok.
  - Observed mutations (restored byte-identical, `cmp`): MAX_SPAN 0 → SH1-multiline not ok; an open expansion counted
    clean → SH1-unterminated not ok; the backslash escape checked before the literal-quote state → SH1-squote not ok;
    `-mindepth 1` dropped → SF1 not ok.
  - `bats tests/*.bats` (serial): 981 ok / 0 not ok (67 env skips), count 981. mcp-n4-kit unittest: 491 OK.
    shellcheck 0.11.0: only the pre-existing SC2329 info. sweep-build-state exit 0; `--age` exit 0; fold-audit
    `--strict` 196/196; `--deltas-since 2026-09-24` 145 checked, 0 not cited; guard-pins `--strict` exit 0;
    gen-lint-index `--check` fresh.
  - Commit d140a64, PR #233, merge ea3b784. RDD: consent granted, 4 lenses APPROVED and acknowledged — lineage
    review-5056c52c4e0fbe6c, authority burned. 5 informational findings posted on #226: R3/R4 file-root fail-open →
    D1c; R2-maxspan-unexplained → one comment in D1c; R3-patsub-span-false-positive-surface and
    R3-patsub-multiline-negative-unpinned → parked.
- 2026-10-03 D2 + D1c (branch `feat/dl-d2-lint-size`, from `origin/main` ea3b784). Route: inline.
  Design (structural, decided up front per `METHODOLOGY.md` § Conformance rules): hand-written lines = non-blank lines
  after `mb_strip` blanks comments, outside slotomatic regions; a region opens on a RAW line containing
  `BEGIN BAJA AUTO GENERATED CODE` and closes on the next `END BAJA AUTO GENERATED CODE` (substring, so the
  `//region /*+ ---- … +*/` form slot-o-matic writes in kit modules and the bare `/*+ … +*/` form both match); an
  unclosed region is a WARN and its lines count; pure class = no `@NiagaraType` and no `extends B<Upper>`; method spans
  from `mb_parse`; thresholds strictly greater than 800 / 80, overridable. Evidence: marker text in the Tridium
  developer doc `docDeveloper … slot-o-matic.html:421,459` and corpus B711 (slot-o-matic writes the region back);
  no corpus block sets a size limit — the thresholds are the retro's review smell, so every row is WARN.
  - RED: LSZ1-LSZ7, LSZ-awkfail, LSZ-finderr not ok (tool absent); SF4 not ok (a file root returned 0); RM49-RM51 not
    ok (no lint-size member).
  - GREEN: `bats tests/lint-size.bats tests/scan-files.bats tests/shell-hygiene.bats` 22 ok; RM49-RM51 ok.
  - Observed mutations (restored byte-identical, `cmp`): region lines counted → LSZ2 not ok; method rule on Baja classes
    → LSZ4 not ok; unclosed-region row dropped → LSZ5 not ok; awk status ignored → LSZ-awkfail not ok; find status
    ignored → LSZ-finderr not ok; directory check dropped → SF4 not ok; advisory_member relays every exit → RM51 not ok.
  - Real-tree smoke (a deployed client checkout, read only, not committed): 10 WARN rows, all genuine — one facade
    BComponent at 1362 hand-written lines and pure `step()`/stage methods of 102-236 lines; no false row seen.
  - `bats tests/*.bats` (serial): 997 ok / 0 not ok (67 env skips), count 997. mcp-n4-kit unittest: 506 OK.
    shellcheck 0.11.0: only the pre-existing SC2329 info. sweep-build-state exit 0; `--age` exit 0; fold-audit
    `--strict` 196/196; `--deltas-since 2026-09-24` 145, 0 not cited; guard-pins `--strict` exit 0 (LSZ2/LSZ4/LSZ5/
    LSZ-awkfail/LSZ-finderr resolve); gen-lint-index `--check` fresh (lint-size row, Auto yes).
  - Commit 9d22fbf, PR #235, merge a3a9b67. RDD: consent granted, 4 lenses APPROVED and acknowledged — lineage
    review-a13efee4322f36c9, authority burned. 4 informational findings posted on #226: R3-002 (nested/stray markers
    hide lines) → D2b; R4/R2 advisory ERROR drops the cause → fixed with D3's helper change; R3-001 (`class ?` for an
    enum/interface/record) → parked.
- 2026-10-03 D3 + D2b (branch `feat/dl-d3-license-gate`, from `origin/main` a3a9b67). Route: inline.
  Design: licensed = the file declares `Feature getLicenseFeature()` with a body that is not just `return null;` (an
  abstract/interface declaration ending in `;` is skipped); callbacks = `changed`, `do<Upper>…` actions (timers fire
  actions) and servlet writes, minus doGet/doHead/doOptions/doTrace; acts = a statement other than `super.<m>(…);` or
  `return;`; gated = `isOperational(`/`isFault(`/`isFatalFault(` in the callback body. Overlap check: no existing lint
  reads `getLicenseFeature`, `isOperational` or `getSlotPathOrd` (grep over toolbelt/). Evidence (decompiled):
  `BAbstractService.java:148-150` (getLicenseFeature returns null by default), `:152-190` (checkLicense sets fatalFault
  and logs, never throws), `:87-90` (isOperational = !fatalFault && !disabled && !fault); `ServiceManager.java:297-298`
  (serviceStarted runs right after the license check with no fault test); corpus B1143, B1145, B1146.
  - RED: LIG1-LIG7, LIG-awkfail, LIG-finderr not ok (tool absent); LSZ8, LSZ9 not ok; RM52, RM53 not ok.
  - GREEN: `bats tests/lint-license-isoperational-gate.bats tests/lint-size.bats` 23 ok; RM49-RM53 ok.
  - Observed mutations (restored byte-identical, `cmp`): gate check dropped → LIG1 not ok; `return null;` treated as
    licensed → LIG2 not ok; super call counted as acting → LIG3 not ok; awk status ignored → LIG-awkfail not ok; find
    status ignored → LIG-finderr not ok; nested BEGIN ignored → LSZ8 not ok; reason dropped from the ERROR row → RM53
    not ok.
  - Smoke on decompiled vendor modules (read only): a licensed driver network and a licensed framework service get rows
    on ungated changed()/actions; the framework service gates inside a helper (`fwServiceStarted()`), the documented
    one-body limit — advisory WARN, cleared by moving the gate into the callback.
  - `bats tests/*.bats` (serial): 1010 ok / 0 not ok (67 env skips), count 1010. mcp-n4-kit unittest: 513 OK.
    shellcheck 0.11.0: only the pre-existing SC2329 info. sweep-build-state exit 0; `--age` exit 0; fold-audit
    `--strict` 196/196; `--deltas-since 2026-09-24` 145, 0 not cited; guard-pins `--strict` exit 0; gen-lint-index
    `--check` fresh (new row, Auto yes).
  - Commit eeb488c, PR #237, merge d551090. RDD: consent granted, 4 lenses APPROVED and acknowledged — lineage
    review-0a160c9b2c00cbe3, authority burned. 6 informational findings posted on #226: R3-002 (gate anywhere /
    qualified) → D3b; R4/R2/R3-001 (ERROR reason may be a WARN row), R2 (row omits isFatalFault), R3-003 (status
    asserts) → folded into D3b.
- 2026-10-03 D4 + D3b (branch `feat/dl-d4-set-null-ord`, from `origin/main` d551090). Route: inline.
  Design (scope = one method from `mb_parse`, comments blanked): SNO1 — `set<Name>(… x.getSlotPathOrd())` where the call
  is itself an argument (followed by `,` or `)`), guarded by a `getSlotPathOrd() == / != null` compare on that line or
  earlier in the method; SNO2 — `v = …getSlotPathOrd();` then `set…(` with `v` as an argument, unless `v == null`,
  `v != null`, `null == v`, `v.isNull()` or `requireNonNull(v` appears from the assignment to the call (a ternary on
  the call line counts). Out of scope (false negatives): values carried through a field, a return or another method.
  Overlap check: `lint-null-context-write` flags a null CONTEXT argument, not a null value. Evidence (decompiled):
  `BComponent.java:321-324` (`getSlotPathOrd()` returns null when `getSlotPath()` is null), `ComponentSlotMap.java:
  171-173` (null when the component has no space), `BComplex.java:386-390` (`set` dereferences `value.getSlotMap()`).
  - RED: SNO1-SNO4, SNO-awkfail, SNO-finderr not ok (tool absent); LIG8, LIG9, LIG10 not ok; RM54, RM55 not ok.
  - Found while proving SNO2-scope: the first draft missed a local passed as the ONLY setter argument
    (`setTargetOrd(ord)`): the argument pattern needed a separator before the name. Pinned by SNO2-single (RED, then
    GREEN), after which the SNO2-scope mutation bites.
  - GREEN: `bats tests/lint-set-null-ord.bats tests/lint-license-isoperational-gate.bats` all ok; RM49-RM55 ok.
  - Observed mutations (restored byte-identical, `cmp`): direct rule off → SNO1 not ok; guard ignored → SNO2-guard not
    ok; locals carried across methods → SNO2-scope not ok; awk status ignored → SNO-awkfail not ok; find status ignored
    → SNO-finderr not ok; gate anywhere in the body → LIG8 not ok; qualified gate accepted → LIG9 not ok; first output
    line as the reason → RM55 not ok.
  - Smoke on decompiled vendor modules (read only, 334 files that call getSlotPathOrd()): 81 SNO1 rows, mostly
    Tridium importers passing a just-resolved point's ORD straight to a setter; advisory, as documented in §H3.
  - `bats tests/*.bats` (serial): 1025 ok / 0 not ok (67 env skips), count 1025. mcp-n4-kit unittest: 522 OK.
    shellcheck 0.11.0: only the pre-existing SC2329 info. sweep-build-state exit 0; `--age` exit 0; fold-audit
    `--strict` 196/196; `--deltas-since 2026-09-24` 145, 0 not cited; guard-pins `--strict` exit 0; gen-lint-index
    `--check` fresh (new row, Auto yes).
  - Commit 391f1ef, PR #239, merge 6cc16f6. RDD: consent granted, 4 lenses APPROVED and acknowledged — lineage
    review-b74d56944839d5e8, authority burned. 9 informational findings posted on #226: R2-004 (receiver-agnostic
    guard) → D4b, with R2-001 (stale §H3 marker) and R2-002/R3-002 (setter-name comment); R3-001, R2-003, R3-003,
    R3-004, R3-005 → parked.
- 2026-10-03 D5 + D4b (branch `feat/dl-d5-split-package-check`, from `origin/main` 6cc16f6). Route: inline.
  Design: module = a directory with `module-include.xml` / `build.gradle[.kts]` next to `src/` (each -rt/-ux/-wb artifact
  is its own N4 module); the module search prunes dot-dirs, `build/`, `node_modules/`, `src/`, `srcTest/`; packages come
  from each `*.java` `package …;` under the module's `src/` (no declaration = `(default)`); a package in two or more
  DISTINCT modules is one WARN row naming each module and one file. Fail closed: the module search, a per-module listing
  or an awk read that fails is exit 3; no module found is exit 3. Not a `lint-*` (project-level, so not in the lint
  index and not run by `report-module.sh`); routed by `BUILD-LOOP.md` §5 non-lint tools (kit-links L5) and §D4c.
  Evidence (decompiled): `ModuleClassLoader.java:186-294` (`nfind`: parent, own jar, then deps — sticky last-hit
  cache, deps that declare the package, every other dep; first hit wins), `NModule.java:333-335` (`containsPackage`
  knows only `<types>` packages); corpus B1125 (first-dependency-wins shadow), B616/B617 (shade bundled libraries).
  - RED: SPC1-SPC7, SPC-finderr not ok (tool absent); SNO1-recv not ok (receiver-agnostic guard).
  - GREEN: `bats tests/split-package-check.bats tests/lint-set-null-ord.bats` all ok (SPC-findtree added with the
    module-search prune, GREEN at once, mutation-proven).
  - Observed mutations (restored byte-identical, `cmp`): files counted instead of modules → SPC2 not ok; scanning the
    module dir (srcTest included) → SPC4 not ok; per-module listing status ignored → SPC-finderr not ok (after the module
    search stopped descending into `src/`, which had masked it); module-search status ignored → SPC-findtree not ok;
    receiver-agnostic guard → SNO1-recv not ok.
  - Smoke (read only): one client project checkout → no row, exit 0; a directory holding 16 checkouts/worktrees of
    the same project → one row per package naming all 16 copies (documented: `<project-root>` is ONE checkout).
  - `bats tests/*.bats` (serial): 1035 ok / 0 not ok (67 env skips), count 1035. mcp-n4-kit unittest: 530 OK.
    shellcheck 0.11.0: only the pre-existing SC2329 info. sweep-build-state exit 0; `--age` exit 0; fold-audit
    `--strict` 196/196; `--deltas-since 2026-09-24` 145, 0 not cited; guard-pins `--strict` exit 0; gen-lint-index
    `--check` fresh (no lint header changed).
  - Commit 4ba3b9f, PR #242, merge ecf93f6. RDD: consent granted, 4 lenses APPROVED and acknowledged — lineage
    review-33645fae1a1ba9d8, authority burned. 6 informational findings posted on #226: R4-001 (unchecked
    aggregation) and R2-001/R3-001 (call-chain receivers) → D5b with R2-002 and R3-003; R3-002 → parked. #142 commented
    with the implementation and the open wiring decision (left open for the operator).
- 2026-10-03 D5b (branch `feat/dl-d5b-aggregation-and-receivers`, from `origin/main` ecf93f6). Route: inline.
  - RED: SPC-aggfail not ok (a PATH-stub sort that fails only for the aggregation's `-k1,1` call: exit 0, no rows);
    SNO1-chain not ok (getA()'s null check silenced getB()'s direct pass).
  - GREEN: `bats tests/split-package-check.bats tests/lint-set-null-ord.bats` 22 ok.
  - Observed mutations (restored byte-identical, `cmp`): sort failure not fatal → SPC-aggfail not ok; any receiver
    recorded as guarded → SNO1-chain not ok.
  - `bats tests/*.bats` (serial): 1037 ok / 0 not ok (67 env skips), count 1037. mcp-n4-kit unittest: 535 OK.
    shellcheck 0.11.0: only the pre-existing SC2329 info. Sweeps, fold-audit (196/196; deltas 145, 0 not cited),
    guard-pins `--strict` and gen-lint-index `--check` clean.
  - Commit 632455c, PR #244, merge 3ebcacb. RDD: consent granted, 4 lenses APPROVED and acknowledged — lineage
    review-7688aedb9a50cce8, authority burned. 3 informational findings (all suggestions: shared receiver regex, null-first
    and indexed receiver pins, awk-branch pin) posted on #226 and parked. The fix-forward chain closed here.
- 2026-10-03 D6 close (branch `chore/dl-d6-release-v0.31.0`, from `origin/main` 3ebcacb). VERSION 0.30.0 → 0.31.0;
  CHANGELOG `[v0.31.0]` (kit D1-D5 + the nine mcp-n4-kit entries verbatim); close retro
  `retros/2026-10-03-deferred-lints-2026-10-03-close.md` (5Δ, pending) + INDEX row; BUILD-STATE kit envelope. The tag,
  the skill reinstall and the drift check run after the merge (recorded in the PR and the final report).

## Parked advisories
Non-blocking review advisories parked under the anti-cascade policy (posted on #226, no sub-task).
- R3-patsub-span-false-positive-surface (D1b, `tests/helpers/patsub-check.py`): following a `${x/` start inside quoted
  text or a heredoc onto later lines can produce a spurious row. It fails closed, and the kit scripts are clean today.
- R3-patsub-multiline-negative-unpinned (D1b, `tests/shell-hygiene.bats`): no negative pin for a quoted multi-line
  replacement, for the MAX_SPAN boundary or for the `$'...'` state. Test depth only.
- R3-002 (D5, `split-package-check.sh`): an annotation (or a block-comment end) on the same line as `package …;`
  (package-info.java) reads as the `(default)` package and can report a spurious split — errs toward WARN / `--strict`
  exit 1, never toward a clean pass.
- R3-001 (D4, `lint-set-null-ord.sh`): any `set…` identifier (`setup(`, `settings(`) counts as a setter and a nested
  call's `)` ends the argument, so a few false WARNs are possible. Advisory, not a fail-open.
- R2-003 (D4): `vguard` means both "guarded" and "already reported". Readability.
- R3-003 (D4): no pins for the `null == v`, `null != v`, `requireNonNull(v` guard branches. Test depth.
- R3-004 (D4, license gate): a gate inside a `try {` / `synchronized (…) {` wrapper that opens the callback is cut off
  by the first-statement head — a false-WARN edge, not a fail-open.
- R3-005 (D4): LIG10 does not assert the exit status. Test depth.
- R3-001 (D2, `lint-size.sh`): an oversized `enum` / `interface` / `record` file is named `class ?`; the count is still
  reported. Readability of the row only.

## Delivery record
| Task | PR | Merge SHA | Review |
|---|---|---|---|
| D1 | #230 | 49185d4 | APPROVED, 4 lenses, review-d02202bd55feff67 |
| D1b | #233 | ea3b784 | APPROVED, 4 lenses, review-5056c52c4e0fbe6c |
| D2 + D1c | #235 | a3a9b67 | APPROVED, 4 lenses, review-a13efee4322f36c9 |
| D3 + D2b | #237 | d551090 | APPROVED, 4 lenses, review-0a160c9b2c00cbe3 |
| D4 + D3b | #239 | 6cc16f6 | APPROVED, 4 lenses, review-b74d56944839d5e8 |
| D5 + D4b | #242 | ecf93f6 | APPROVED, 4 lenses, review-33645fae1a1ba9d8 |
| D5b | #244 | 3ebcacb | APPROVED, 4 lenses, review-7688aedb9a50cce8 |

## Open decisions
- #142 wiring (operator): the issue asks for `split-package-check` to run automatically, in the client repository's CI,
  not as a manual step. That repository is outside this kit, so this feature ships the conservative read-only
  advisory tool (`--strict` ready for a gate) and leaves the choice open: (a) add `split-package-check.sh --strict
  <checkout>` to the client repository's CI (recommended: the issue's own proposal, server-side, cross-module by
  nature); (b) auto-run it from `build.sh` over the GROUP directory it already receives (in-kit, but sees one group,
  not the whole project); (c) keep it manual before a release. Recorded on #142.

## Close (2026-10-03)
- Merged: D1 #230, D1b #233, D2+D1c #235, D3+D2b #237, D4+D3b #239, D5+D4b #242, D5b #244, D6 (this release PR).
- #226: lead item fixed (D1/D1b/D1c); every review's advisories posted there with dispositions; the rest of #226 stays
  open. #142: implemented as the advisory tool; the wiring decision is open (see "Open decisions"), issue left open.
- Close retro: `retros/2026-10-03-deferred-lints-2026-10-03-close.md` (5Δ, pending).
- Owed (not in scope): 12 other lints still walk with `find … 2>/dev/null`.

## Next step
Fold the close retro; the operator's #142 wiring choice; the remaining #226 items.
