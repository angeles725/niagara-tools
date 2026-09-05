# Exploration — build-n4-module-campaign6

**Date**: 2026-09-05 | **Phase**: explore | **Status**: complete
**Engram**: `sdd/build-n4-module-campaign6/explore` (observation 8165)
**OpenSpec target**: `openspec/changes/build-n4-module-campaign6/explore.md`
**Orchestrator gate notes** (investigador, 2026-09-05): contract conformant; the `run-pure-test.bats` skip claim was re-verified at `tests/run-pure-test.bats:12`; the ng-deploy.bats count below was corrected from 26 to the actual 37 `@test`; the freeze-stat "delta 1 + 4 already folded" claim is treated as a grep-before-fold obligation for the tasks phase, not as established.

---

## 1. Current-State Map — Enforce vs Declare

### Docs (what they actually do)

| File | Lines | State | Enforces | Declares only |
|---|---|---|---|---|
| METHODOLOGY.md | 51 | mature | nothing (read gate) | rt/schema/editing/build/debugging/maintenance checklists |
| BUILD-LOOP.md | 64 | mature | §7 pre-push hook via install-hooks.sh | orient/preflight/design/deploy steps |
| build-verify.md | 121 | mature | via verify-module.sh, run-pure-test.sh (called from build.sh) | deploy recipes, signing paths, "Known gap: -ux slotomatic" |
| corpus-index.md | 68 | mature | nothing | P0/P1/P2 pointer map to B729-B760 |
| BUILD-STATE.md | ~165 | operative | GATED: retro_required/retro_pending (sweep + hook) | DECLARED: all module/jar fields (separate repos, unverifiable) |
| SOURCES.md | 30 | thin | nothing | corpus-nav/preview/exemplar pointers; one stale Engram topic |
| types/logic.md | 78 | growing | nothing | rt control rules, HOA, pure-class extraction |
| types/dashboard.md | 83 | mature | nothing | servlet-SPA, RBAC, HMI, plano overlay, extending |
| types/wb-widgets.md | 19 | **seed** | nothing | ladder + 3 recipes; TODO: PxEditor flow |

### Toolbelt (what each script enforces)

| Script | Enforces | Test coverage | Bite confirmed |
|---|---|---|---|
| verify-module.sh | bytecode 52, signed, types resolve, typecount (--src), baja stamp, stored, empty-palette WARN | V1-V17 (verify-module.bats) | mutation-checked |
| build.sh | Java 8, clean, slotomatic all profiles with sources, then verify gate | B1-B7 (build-sh.bats) | mutation-checked |
| mirror-niagara-home.sh | source/mirror identity guards (exit 20) | M1-M5 | yes |
| run-pure-test.sh | compiles+runs zero-Baja JUnit; exits 3 on empty cache (set-e fixed PR14) | P1-P6 (run-pure-test.bats) | **SKIPS in CI** (no gradle pre-fetch) |
| stored-repack.sh | STORED repack; never overwrites | S1-S3 | yes |
| sweep-build-state.sh | envelope structure (module, retro_required, retro_pending booleans); INDEX rows ↔ real files; review-status ∈ {pending,folded} | H1-H10 (build-retro-sync.bats) | mutation-checked |
| install-hooks.sh | opt-in, refuses to clobber existing hooksPath without --force | I1-I4 (install-hooks.bats) | yes |

**sweep does NOT enforce**: (a) that a retro file's `<!-- review-status: X -->` marker agrees with its INDEX row (marker-blind: it parses the INDEX row only and tests the file with `[ -f ]`); (b) that a `folded [ev: retro X]` citation appears in a core kit file.

### Tests (9 files, 104 @test) — bite summary

| Suite | Count | Weakness |
|---|---|---|
| verify-module.bats | 17 | none — strong, mutation-checked |
| ng-deploy.bats | 37 | none — includes slotomatic paths |
| build-retro-sync.bats | 19 | no case where marker and INDEX disagree; no case with a missing marker |
| run-pure-test.bats | 6 | **skips entirely when junit not in ~/.gradle — CI never runs P1-P6** |
| kit-links.bats | 3 | L1 correct (external pointer skipped, bites under HOME=/nonexistent) |
| build-sh.bats | 10 | adequate |
| mirror-niagara-home.bats | 5 | adequate |
| stored-repack.bats | 3 | adequate |
| install-hooks.bats | 4 | adequate |

Baseline (QA, 2026-09-05): 104/104 pass, 8.6 s wall, flake risk low (fakebin git, no network).

---

## 2. Diagnosis

### Consistency gaps

1. **Marker vs INDEX drift**: the sweep never reads the in-file marker. Real drift today: `2026-09-02-comppan-fase1-staging.md` and `2026-09-02-dashboardpan-detail-render-doors.md` carry `review-status: fresh` while INDEX says `folded`; `2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md` is INDEX=`pending` with no marker. Of the freeze-stat retro's 4 deltas, delta-1 (free-lock remedy) looks folded in `build-verify.md`; delta-4 (junit gradle-cache path) may be folded elsewhere (grep before folding); deltas 2 (annotation-only with build.sh) and 3 (BDouble import) are missing from `types/logic.md §Regenerating slots` and `METHODOLOGY §Slot rules`.

2. **~25 folded retros without marker**: INDEX says `folded`, files have no machine-readable marker. Tolerable if INDEX is the source of truth, but invisible to file-only tooling. Decision (orchestrator + QA): option 2-lite — when a marker is present it must equal the INDEX row; a missing marker is tolerated.

3. **Content-fold-audit gap** (BUILD-STATE open_issue #3): sweep validates structural anchors but NOT that a `folded [ev: retro X]` citation actually appears in a core kit file. Manual-only today.

4. **`DashboardPan.verify_gate = unknown`**: BUILD-STATE stale — deployed but gate never run post-deploy. Separate repo; no mechanical fix possible from this kit.

5. **`SOURCES.md`**: references stale `Engram topic research/niagara/coldroom-ux/progress`. Corpus-nav and dashboard-preview paths are machine-specific absolute paths with no `$BUILD_N4_KIT`-style indirection.

6. **`build-verify.md` "Known gap — ng-deploy.sh runs slotomatic for -rt only"** is stale for modes A/C (`scripts/ng-deploy.sh:493-500` adds `:MODULE-ux:slotomatic` when -ux is annotated) but still true for mode B (`:552-553` ignore `--with-slotomatic`). Doc-vs-code contradiction; retitle to the mode-B gap.

### Rot risks

- Machine-specific absolute paths in SOURCES.md and BUILD-LOOP.md §3. Break silently on a different machine.
- `run-pure-test.bats` P1-P6: always SKIP in CI. Green CI ≠ tested for run-pure-test.sh.
- SOURCES.md stale Engram topic.

### What five campaigns left half-done

| Item | Source | Status |
|---|---|---|
| freeze-stat deltas 2+3 | `2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md` | Not folded (INDEX=pending) |
| Gate-exit-taxonomy rule | `2026-09-04-gate-exit-taxonomy-promotion.md` | Rule not in METHODOLOGY yet |
| C3 meta-lessons (3 rules: gate signal/noise, WARN>silence, marker>prose) | `2026-09-04-campaign3-close-process-meta-lessons.md` | Not folded |
| C4 meta-lessons (2 rules: worktree-off-origin, grep-before-fold) | `2026-09-04-campaign4-close-process-meta-lessons.md` | Not folded |
| C5 note (feature PR uses exit a) | `2026-09-04-campaign5-gate-activation.md` | Not folded |
| CI rules (pin linter, env-coupling check) | `2026-09-04-ci-server-side-enforcement.md` | Not folded |
| Content-fold-audit machine check | BUILD-STATE open_issue #3 | Never started |
| run-pure-test.bats silent in CI | ci.yml + run-pure-test.bats | Not fixed |
| Slotomatic openspec change | `openspec/changes/niagara-tools-slotomatic-integration/` (May 2026, 40 tasks, 0 tracked) | Verified 100% superseded by campaigns 3-5 (flags, functions, docs, 37 ng-deploy tests all exist; targeted v0.3.0 vs current v0.15.1) → archive with supersession note, never apply |
| `openspec/` directory untracked | trabajador close report | Commit under campaign 6 (hybrid store decision) |

---

## 3. Backlog of Candidate Improvements

The full harvest table grouped by destination file (investigador1) lives at `/home/cristian/modulos_niagara_n4/niagara-tools-worktrees/campaign6-harvest.md` (uncommitted working document); its condensed content is merged here.

### (a) Doctrine deltas from pending retros and harvest

| # | Destination | Source | Delta | Value | Size | Biting test? |
|---|---|---|---|---|---|---|
| A1 | `types/logic.md §Regenerating slots` | freeze-stat delta-2 / harvest L1 | Annotation-only is sufficient with build.sh; AUTO stub needed only when compiling without slotomatic | med | +3L | No |
| A2 | `METHODOLOGY.md §Slot rules` | freeze-stat delta-3 | "MIN/MAX facet needs `BDouble.make`; confirm `import javax.baja.sys.BDouble`" | med | +1L | No |
| A3 | `METHODOLOGY.md §Kit-maintenance` | gate-exit-taxonomy | Gate exit set must cover every legitimate change class; typed exit + proof-of-work guard | med | +2L | No |
| A4 | `METHODOLOGY.md §Kit-maintenance` | C3-meta-1 | Gate check: HIGH-signal/low-FP — scope to the reported defect; mutation-prove the guard | high | +3L | No |
| A5 | `METHODOLOGY.md §Kit-maintenance` | C3-meta-2 | WARN over silent removal of a safety guard | low | +2L | No |
| A6 | `METHODOLOGY.md §Kit-maintenance` | C3-meta-3 | Marker is the promotable unit, not operator-session narrative prose | low | +2L | No |
| A7 | `METHODOLOGY.md §Kit-maintenance` | C4-meta-1 | Coverage checks in a worktree off `origin/main`, not local stale checkout | high | +2L | No |
| A8 | `METHODOLOGY.md §Kit-maintenance` | C4-meta-3 | Grep every kit file before folding (not only the proposed target file) | high | +2L | No |
| A9 | `METHODOLOGY.md §Kit-maintenance` | C5/gate-exit-taxonomy | Feature PR uses exit (a) of the close gate | low | +1L | No |
| A10 | `CONTRIBUTING.md` | CI-meta-2 | Pin any linter/tool a CI gate depends on | high | +2L | CI catches drift |
| A11 | `METHODOLOGY.md §Kit-maintenance` | CI-meta-3 | Env-coupling check: resolve against repo, prove under `HOME=/nonexistent` | high | +2L | No |
| A12 | `METHODOLOGY.md` (new rules) | harvest M1, M2 (proven live) | Multi-session git-status guard; live-verify write/secret safety | high | ~18L | No |
| A13 | `types/logic.md` | harvest L2 (proven live) | Control ORD is integrator configuration, not source-derivable | med | +3L | No |
| A14 | `types/logic.md` (from closed kitControl corpus B536-B557, no new blocks) | harvest LC1-LC5 (proven by code) | Writable priority array (B536/B544); BLoopPoint/PID (B539); multi-input null contract (B537); fail-safe checklist (B543); alarm+history point-extension chain (B552) | high | ~40L | No |
| A15 | `types/dashboard.md` | harvest D1 (proven by code) | Off-station consumer must recompute derived keys | med | +3L | No |
| A16 | `build-verify.md` | harvest BV1 (proven by code) | Retitle the slotomatic known gap to the mode-B gap (diagnosis #6) | med | ~4L | No |
| A17 | `build-verify.md` | harvest V1-V4 | grep-consumer delta proof; live-overrides doc; freshness-before-"live"; CORS is not a bug | med | ~10L | No |
| A18 | `SOURCES.md` / `corpus-index.md` | harvest S1, S2 | Tool-zero is not absence; no [CERT] off a mangled decompile | low | +4L | No |

### (b) Toolbelt / mechanics improvements

| # | Destination | Source | Delta | Value | Size | Biting test? |
|---|---|---|---|---|---|---|
| B1 | `toolbelt/sweep-build-state.sh` | diagnosis #1, QA RED fe6b88d | Read each retro's marker (first word after `review-status:`, tolerate ` · DATE` suffix and absence); fail on out-of-domain word or marker ≠ INDEX row; re-stamp the two `fresh` files to `folded` | high | ~15L | YES — RED already staged: M1 disagree, M4 `fresh` shape, M5 real tree |
| B2 | new `toolbelt/sweep-fold-audit.sh` | BUILD-STATE open_issue #3 | For each `folded` slug, grep core kit files for `[ev: retro <slug>`; WARN when absent | high | ~30L | YES: fold in INDEX without `[ev:]` citation → WARN |
| B3 | `tests/run-pure-test.bats` + `ci.yml` | CI gap | Pre-fetch junit in CI so P1-P6 actually run | high | +5L ci.yml | YES: P2 (failing pure test exits non-zero) is the mutation proof |
| B4 | `toolbelt/verify-module.sh` | harvest T1 | `--plano` aspect-ratio check | med | ~20L | YES (fixture image with wrong ratio → FAIL) |
| B5 | new `toolbelt/preflight.sh` | harvest P1 | Automate BUILD-LOOP §0.b preflight | high | ~60L | YES (missing env / locked jar → typed exit) |

### (c) SKILL.md improvements

| # | Source | Delta | Value |
|---|---|---|---|
| C1 | skill-improver | Remove `state` column from decision table (growing/mature/seed not actionable for user) | low |
| C2 | skill-improver | Explicit warning that wb-widgets.md is SEED — direct wb builders to corpus B751-B753 | med |
| C3 | BUILD-LOOP §0 | Align SKILL §Execution step 1 wording with BUILD-LOOP's orient/corpus-nav order | low |

### (d) Derivations (new lanes)

| # | Lane | Description | Value | Size | Biting test? |
|---|---|---|---|---|---|
| D1 | toolbelt | `scaffold-module.sh <name> <type>`: gradle layout + module-include.xml + BUILD-STATE section + empty lexicon | high | ~80L | YES: scaffold → verify-module.sh PASS on empty module |
| D2 | toolbelt | `slot-coverage.sh`: compare module.lexicon keys vs declared types+slots; WARN on empty lexicon | high | ~40L | YES: empty-lexicon module → WARN (CompPan T8 real defect) |
| D3 | toolbelt | `dashboard-screenshot.sh`: Playwright headless PDF wrapper (types/dashboard.md U9) | med | ~25L | No (external Playwright dep) |
| D4 | openspec/archive | Archive slotomatic openspec with supersession note; commit `openspec/` | med | ~20L | No |
| D5 | research lane (investigador1, /research-sdd on niagara-research) | New focuses `module-testing` (what to test where) and `module-servlet-security` residue; deltas return as pending retros | high | blocks B762+ | n/a |

### (e) Quantitative models

| # | Model | How it changes a decision |
|---|---|---|
| E1 / MM1 | **Gate-coverage %** (which legitimate change classes the close gate covers; QA RED first) | Below 100% → a typed exit is missing; drives A3 |
| E2 / MM2 | **Set-coverage** palette ∩ lexicon ∩ declared types | Any type outside the intersection → WARN before build (D2) |
| E3 / MM3 | **Schema-survival risk classifier** (slot type change vs add vs rename) | High-risk change → station-no-boot warning at preflight (B5) |
| E4 | **Build-readiness score** over BUILD-STATE fields | Score < threshold at orient → address issues first |
| E5 | **Retro-debt aging** (days pending; WARN > 30) | High debt → promotion PR before a new module build |
| E6 | **Coverage metric** (% of corpus-index blocks cited with `[ev: corpus B...]`) | Drives next corpus expansion priority |
| E7 | **Mutation score per test file** | Identifies low-bite test files |

---

## 4. Campaign Structure: Three Approaches

| Approach | Description | Pros | Cons | PR budget |
|---|---|---|---|---|
| **A: Single big doctrine PR** | All doctrine deltas in one PR | Simple, one CI run | All-or-nothing rollback; hard fidelity verification | ~150L, Low risk |
| **B: Destination-file-grouped chained PRs** | PR-tools (B1) → PR-ci (B3) → PR-meta (METHODOLOGY/CONTRIBUTING) → PR-types (logic/dashboard/build-verify) → PR-skill → PR-deriv | Coherent review units; per-file fidelity; independent rollback | more PRs | ~25-60L each, all Low |
| **C: Lane-per-session** | investigador: SDD chain + doctrine; investigador1: research lane + harvest + model specs; QA: RED-first tests + verify | Parallel execution; matches team | Merge serialization; file-ownership must be explicit | Same as B within each lane |

**Recommendation: Option C with file ownership, executed as Option B inside the kit repo.** Implementation writers are `sdd-apply` workers launched by the orchestrator (one writer per PR, disjoint file sets); QA owns `tests/*.bats` RED commits and `ci.yml`; investigador1 owns niagara-research commits (research lane) and the uncommitted harvest/model spec documents; METHODOLOGY.md is touched by exactly one PR. Merge order: B1 (RED already staged) → B3 (CI) → doctrine → types → tools/derivations → skill.

---

## 5. Risks and Open Questions

| Risk | Severity | Recommendation |
|---|---|---|
| Double-fold of freeze-stat deltas 1+4 or of any harvest delta already in the core | HIGH | Grep every kit file for the rule BEFORE writing (A8 applied to ourselves); if found, flip INDEX only |
| run-pure-test.bats P1-P6 never run in CI | HIGH | B3 early (second PR) |
| Slotomatic openspec D tasks (docs) state | LOW | Verified superseded (docs/knowledge-base/slotomatic.md, GOTCHAS.md, .env.local.example exist); archive |
| Content-fold-audit false positives on slugs in cross-links or INDEX rows | MED | Scope grep to `[ev: <slug>` pattern; test against known-folded and known-not-folded retros |
| Two writers touching METHODOLOGY.md | MED | One doctrine PR owns it; other lanes submit deltas as retro proposals |
| wb-widgets.md at 19 lines under-serves a wb builder | LOW | C2 short-term; enrichment waits for the research lane or a real wb build |
| Test padding vs speed | MED | Only B1-B5, D1, D2 get tests, each with a stated mutation; doctrine deltas get none |
