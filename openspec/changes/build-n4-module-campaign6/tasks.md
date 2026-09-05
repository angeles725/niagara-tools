# Tasks: build-n4-module-campaign6

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~615 total across 8 PRs (~25–160 per slice) |
| 400-line budget risk | Low (each slice ≤160 authored lines) |
| Chained PRs recommended | Yes |
| Suggested split | PR1→PR2→PR3→PR4→PR5a→PR5b→PR6→PR7 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Low

### Work-Unit Summary

| PR | Branch | Est. lines | Focused test cmd | Rollback boundary |
|----|--------|-----------|-----------------|-------------------|
| PR1 | feat/c6-marker-index | ~60 | `bats tests/build-retro-sync.bats` | Revert sweep-build-state.sh + 2 markers |
| PR2 | feat/c6-ci-pure-test | ~25 | CI run + `grep "# skip"` in log | Revert ci.yml + run-pure-test.bats |
| PR3 | feat/c6-doctrine | ~85 | `./toolbelt/sweep-build-state.sh` exits 0 | Revert METHODOLOGY.md + CONTRIBUTING.md + 7 markers |
| PR4 | feat/c6-types | ~110 | `bats tests/kit-links.bats` | Revert doc files + freeze-stat marker + INDEX row |
| PR5a | feat/c6-tools-audit | ~150 | `bats tests/sweep-fold-audit.bats tests/verify-coverage.bats` | Revert sweep-fold-audit.sh + verify-module.sh coverage |
| PR5b | feat/c6-tools-env | ~160 | `bats tests/preflight.bats tests/slot-coverage.bats` | Revert preflight.sh + slot-coverage.sh |
| PR6 | feat/c6-close | ~130 | `bats tests/` (≥104 pass) + `./toolbelt/sweep-build-state.sh` | Revert + manual re-edit `~/.claude/skills/build-n4-module/SKILL.md` |
| PR7 | feat/c6-research-fold | TBD | `./toolbelt/sweep-build-state.sh` exits 0 | Revert retro files + INDEX rows |

Runtime harness: N/A for all slices — no station, jar, or operator data is touched.

---

## PR1 — feat/c6-marker-index
**Gate exit (a)**: carry PR1 retro · **QA hook**: `qa/c6-marker-index-drift` tip `cb0dd7d` (25 tests; M1/M4 RED)

- [x] **T1.1** `build-n4-module-kit/toolbelt/sweep-build-state.sh` — Add `marker_of()`: `sed -n '1,5p'` + `grep -m1 '^<!--.*review-status:'` + `awk '{print $1}'`; strip ` · DATE` suffix via first-word awk. Domain check `{pending|folded}` → exit 1 out-of-domain. Compare vs INDEX row status → exit 1 on mismatch with file + both values. Absent marker → tolerated (exit 0 for that file). [ev: QA cb0dd7d; design D1; spec R1-1–R1-5] · **Tests**: M1 (marker=pending, INDEX=folded → exit 1), M4 (`fresh · DATE` vs INDEX=folded → exit 1), M5 (real kit tree → exit 0), M6 (review-status in line-40 table cell, no line-1 marker, INDEX=pending → exit 0). **Named mutation**: delete marker-read block → M1 flips exit 1→0 (biting proof). · shellcheck 0.10.0 · ~50 lines

- [x] **T1.2** `build-n4-module-kit/tests/build-retro-sync.bats` — Merge QA branch `qa/c6-marker-index-drift` tip `cb0dd7d`; re-read tip at apply time. 25 tests pass including M6 prose-mention case. [spec R1-8; QA cb0dd7d] · ~10 lines delta

- [x] **T1.3** `build-n4-module-kit/retros/2026-09-02-comppan-fase1-staging.md`, `build-n4-module-kit/retros/2026-09-02-dashboardpan-detail-render-doors.md` — Re-stamp `<!-- review-status: fresh … -->` → `<!-- review-status: folded -->` in the same commit as T1.1. [spec R1-6] · marker-only edits (0 authored lines)

- [x] **T1.4** PR1 retro — Append to `build-n4-module-kit/retros/`. `sweep-build-state.sh` exits 0 on post-PR1 tree. · ~5 lines

---

## PR2 — feat/c6-ci-pure-test
**Gate exit**: n/a (no kit file touched; no retro required) · **QA hook**: CI log P1–P6 zero-SKIPs; P2 mutation RED

- [x] **T2.1** `.github/workflows/ci.yml` — Add `actions/setup-java@v4 temurin 8` step; add `curl` pre-fetch: `junit-4.13.2.jar` sha256 `8e495b634469d64fb8acfa3495a065cbacc8a0fff55ce1e31007be4c16dc57d3` + `hamcrest-core-1.3.jar` sha256 `66fdef91e9739348df7a096aa384a5685f4e875584cce89386a7a47251c4d8e9` → `~/.gradle/caches/modules-2/files-2.1/…`; sha256 mismatch = hard stop, never silent re-pin; add `grep "# skip"` log step after bats run. [design D4; spec R2-1] · ~15 lines

- [x] **T2.2** `tests/run-pure-test.bats` — Add `${CI:-}` guard: convert `skip` → `fail` when `$CI` is set so P1–P6 produce zero-SKIPs in CI log. [spec R2-2/R2-3] · **Named mutation**: CI=1 HOME=/nonexistent → 6 not ok (exit 1 vs previous 6 skip). · ~10 lines

---

- [x] **T2.3** `tests/build-retro-sync.bats` — Add `bats_require_minimum_version 1.5.0` at the top so `run !` semantics (introduced by the PR1 SC2314 fix) are guaranteed and the bats "minimum guaranteed version" stderr warning disappears. Test-infra hygiene only; QA follow-up from PR1 verification. · 1 line

---

## PR3 — feat/c6-doctrine
**Gate exit (a)**: carry PR3 retro · **Doc-only: zero new bats tests (RX-2)** · **QA hook**: `sweep-build-state.sh` exits 0 with 7 folded + 1 pending (freeze-stat)
**Grep-before-fold REQUIRED before each delta** (rule K6/RX-4). If key assertion found in any kit file → flip INDEX/marker only; record grep evidence in PR description.

- [x] **T3.1** `build-n4-module-kit/METHODOLOGY.md §Kit-maintenance` — grep: `rg "gate exit|mutation.prove|high.signal|promotable unit|worktree|origin.main|grep.*kit|exit.*(a)|HOME.*nonexistent|set -e" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add K1–K9 per spec §5 table [ev: respective retro slugs: gate-exit-taxonomy-promotion, campaign3-close-process-meta-lessons, campaign4-close-process-meta-lessons, campaign5-gate-activation, ci-server-side-enforcement]. · ~60 lines

- [x] **T3.2** `build-n4-module-kit/METHODOLOGY.md` — grep: `rg "multi.session|dirty tree|git status.*diff|prod.*check|no.token.*401|test.*creds" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add §Multi-session-coordination (harvest §M1) [ev: retro dashboardpan-2d-to-3d-port + module-authoring-mega-campaign]; add §Live-verify-safety (harvest §M2) [ev: retro live-cutover-and-authenticated-control + obix-and-loginless-dashboard-runbooks]. · ~12 lines

- [x] **T3.3** `build-n4-module-kit/METHODOLOGY.md` — grep: `rg "what.*test.*where|test.*matrix|per.*type" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add LC7 what-to-test-where table per module type [ev: corpus B743/B12/BTI]. · ~8 lines

- [x] **T3.4** `build-n4-module-kit/CONTRIBUTING.md` — grep: `rg "tool.*pin|linter.*pin|No CI" build-n4-module-kit/CONTRIBUTING.md`. Add K10/A10: pin any linter/tool a CI gate depends on [ev: retro ci-server-side-enforcement]. Fix §8 "No CI" claim → note ci.yml exists since campaign 5. · ~5 lines

- [x] **T3.5** `build-n4-module-kit/retros/INDEX.md` + 7 retro files — Flip 7 rows `pending`→`folded`; stamp `<!-- review-status: folded -->` in each file in the same commit that flips its INDEX row: kit-continuity-and-retro-gate-campaign, run-pure-test-set-e-empty-cache, gate-exit-taxonomy-promotion, campaign3-close-process-meta-lessons, campaign4-close-process-meta-lessons, campaign5-gate-activation, ci-server-side-enforcement. **NOTE**: A2 (BDouble.make) already lives at METHODOLOGY.md:9+11; do NOT re-fold — INDEX row and its marker handled in PR4 (freeze-stat). [spec R3-7/R3-8] · marker edits only

- [x] **T3.6** Guard + PR3 retro — `./build-n4-module-kit/toolbelt/sweep-build-state.sh` exits 0 (7 folded, 1 pending: freeze-stat). Append retro (exit a). · ~5 lines

---

## PR4 — feat/c6-types
**Gate exit (a)**: carry PR4 retro · **Doc-only: zero new bats tests (RX-2)** · **QA hook**: `bats tests/kit-links.bats` exits 0; `grep pending build-n4-module-kit/retros/INDEX.md` → 0 rows
**Grep-before-fold REQUIRED before each delta** (rule K6/RX-4).

- [x] **T4.1** `build-n4-module-kit/types/logic.md` — grep: `rg "annotation.only|AUTO stub|MOUNT.*ORD|integrator.placed|priority array|16.level|null.*skipped|fail.safe|rampTime|alarm.ext|at\(delayMs\)" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add A1/L1 annotation-only is sufficient in build.sh/slotomatic flow [ev: retro freeze-stat-leds Δ2]; A13/L2 control BComponent station MOUNT/ORD is integrator-placed config [ev: retro dashboardpan-2d-to-3d-port Δ4]; LC1–LC5 [ev: corpus B536/B539/B537/B543/B552]; LC6 scheduler seam DI `Sched{at(delayMs);cancel(t)}` [ev: corpus B743]. Remove "GROWING" label from header (R4-5). · ~60 lines

- [x] **T4.2** `build-n4-module-kit/types/dashboard.md` — grep: `rg "ElapsedMs|RemainingMs|RouteAction|purity.gradient|write.surface|canWrite|SPA JS|module.exports" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add A15/D1 off-station consumer derived keys [ev: retro dashboardpan-2d-to-3d-port Δ2]; DUX1 route()→RouteAction -ux servlet seam (HIGH) [ev: corpus B762]; DUX2 purity gradient / inject Baja as Function (MED) [ev: corpus B762]; DWS1 5-gate write-surface checklist (HIGH) [ev: corpus B763]; DWS2 canWrite(boolean) pure-RBAC test seam (HIGH) [ev: corpus B763]; DJS1 inline SPA JS + module.exports shim + node harness (MED) [ev: corpus B762]. · ~30 lines

- [x] **T4.3** `build-n4-module-kit/types/wb-widgets.md` — grep: `rg "Predicate.*inject|wb.*model.*seam|exemplar" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add DWB1 wb/model Predicate-injection seam — turns seed into exemplar-backed (HIGH) [ev: corpus B762/B763]. · ~8 lines

- [x] **T4.4** `build-n4-module-kit/build-verify.md` — grep: `rg "Known gap|rt only|consumer.absence|live.vs.doc|headless.QA|CORS" build-n4-module-kit/build-verify.md`. Retitle "Known gap" → "mode B ignores `--with-slotomatic`"; add A17/V1–V4 [ev: respective retros]. Exact string "Known gap — ng-deploy.sh runs slotomatic for -rt only" must not survive in any kit file. [spec R4-7] · ~12 lines

- [x] **T4.5** `build-n4-module-kit/SOURCES.md`, `build-n4-module-kit/corpus-index.md` — grep: `rg "tool.zero|absence|CERT.*decompile" build-n4-module-kit/ --include="*.md" -l` (exclude `retros/`). Add A18/S1–S2 [ev: respective retros]. · ~6 lines

- [x] **T4.6** `build-n4-module-kit/retros/INDEX.md` + `build-n4-module-kit/retros/2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md` — Flip freeze-stat row `pending`→`folded`; stamp `<!-- review-status: folded -->` in retro file in same commit. Pending count = 0 after PR4 merges. [spec R4-10] · marker edits only

- [x] **T4.7** Guard + PR4 retro — `bats build-n4-module-kit/tests/kit-links.bats` exits 0; pending = 0. Append retro (exit a). · ~5 lines

---

## PR5a — feat/c6-tools-audit
**Gate exit (a)**: carry PR5a retro · **QA hook**: `qa/c6-coverage-pct-red` tip `d7e52a8` (8 pins MM1–MM8, output only — not the summary line; RED)

- [x] **T5a.1** `build-n4-module-kit/toolbelt/sweep-fold-audit.sh` — Usage: `[--strict] <INDEX.md> <kit-root>`; exits 0 (WARN-only or clean) / 1 (`--strict` + uncited) / 3 (usage/env). Stem = filename minus leading `^[0-9]{4}-[0-9]{2}-[0-9]{2}-` minus `.md`. Harvest abbreviated tokens: `grep -ohE '\[ev: retro [A-Za-z0-9][A-Za-z0-9._-]*'` across corpus `*.md + types/*.md` (exclude `retros/` and `INDEX.md`). Hyphen-aligned match: `case "-$stem-" in *"-$T-"*`; 6-char token floor; WARN on uncited: `fold-audit: WARN <f> folded with no [ev: retro …] citation`; `NOTE ambiguous` on multi-stem credit. CI runs `--non-strict` (visible WARN, not blocking). [design D3; spec R5-1] · prose says "version control" not "git" · no `$HOME` coupling · shellcheck 0.10.0 · ~80 lines

- [x] **T5a.2** `build-n4-module-kit/tests/sweep-fold-audit.bats` — F1 (folded+uncited → WARN exit 0), F2 (`--strict` → exit 1). **Named mutations**: F3 (self-cite inside `retros/` → still WARN; drop `retros/` exclusion → F3 loses WARN); F4 (token `5rooms` credits `dashboardpan-5rooms` stem; exact-stem match → F4 gains spurious WARN); F6 (token `ender-doors` NOT credits `detail-render-doors`; plain substring → F6 loses WARN). · ~40 lines

- [x] **T5a.3** `build-n4-module-kit/toolbelt/verify-module.sh` — Add `coverage` subcommand dispatched **before** `while [ $# -gt 0 ]` loop. Args `<npass> <nfail> <nwarn> <nskip>`. `A = P+F+W`; bash integer-tenths: `t=$(( (1000*P + A/2) / A ))`; print `$((t/10)).$((t%10))`. Print `N-A` (literal) when `P+F+W+S == 0`. Exit 0 always; exit 2 + `usage: verify-module.sh coverage …` on stderr for bad argc or non-integer. [design D2; spec R5-7] · shellcheck 0.10.0 · ~25 lines

- [x] **T5a.4** `build-n4-module-kit/tests/verify-coverage.bats` — Merge QA branch `qa/c6-coverage-pct-red` tip `d7e52a8`; re-read tip at apply time. 8 pins MM1–MM8 pass (output lines only; not the summary line per design D2). **Named mutation**: return `100` when `P+F+W+S == 0` → N-A assertion flips. · ~10 lines delta

- [x] **T5a.5** PR5a retro + shellcheck sweep — Append retro (exit a). Run `shellcheck 0.10.0` on all `build-n4-module-kit/toolbelt/*.sh`. Full bats suite ≥104 pass ≤15 s.

---

## PR5b — feat/c6-tools-env
**Gate exit (a)**: carry PR5b retro · **QA hook**: `qa/c6-slot-coverage-red` tip `5a7d90a` (CSV input; 3 prefixed lines `pct=`/`missing=`/`extra=`; 6 pins SC1–SC6; RED)

- [x] **T5b.1** `build-n4-module-kit/toolbelt/preflight.sh` — Usage: `[--jvm-dir <d>] <niagara_home> <gradle-root>`; exits 0/1/2/3. Row format: `PASS|FAIL|WARN|SKIP  <check>  <detail>`. Checks: JDK 8 under `--jvm-dir` (default `/usr/lib/jvm` (read-only)) or `JAVA_HOME` — never `$HOME`; `settings.gradle.kts` plugin pin in `<nh>/etc/m2`; jar lock via `lsof` (SKIP if lsof absent — never false PASS); Windows path `C:` or `\` → FAIL + `/mnt/c/…` (read-only) remedy. [spec R5-3] · no `$HOME` coupling · prose says "version control" not "git" · shellcheck 0.10.0 · ~80 lines

- [x] **T5b.2** `build-n4-module-kit/tests/preflight.bats` — **Named mutations**: PF1 (missing plugin pin → exit 1; make plugin check always PASS → PF1 exits 0); PF2 (identical output under `HOME=/nonexistent`; resolve via `$HOME` → PF2 diverges). Fakebin probes; zero network calls; suite passes under `HOME=/nonexistent`. · ~30 lines

- [x] **T5b.3** `build-n4-module-kit/toolbelt/slot-coverage.sh` — `set-coverage <declared-csv> <required-csv>` (pure MM2): `present = required ∩ declared`; denominator = `|required|` (never `|declared|`); integer-tenths rounding block verbatim-copied from verify-module.sh; `pct=N/A` when `|required|==0` (never 100); `extra` never in numerator. 3 sorted output lines: `pct=`, `missing=`, `extra=`; sets deduped + lex-sorted, comma-joined. Exit 0; exit 2 + usage on argc≠2. Parse subcommand `[--strict] <module-include.xml> <module.lexicon>`: `required` from `<type name=`, `declared` from `key=`; empty lexicon + required≥1 → WARN (CompPan T8 fixture); `--strict` → exit 1 on uncovered. [design D6; spec R5-5] · no `$HOME` coupling · shellcheck 0.10.0 · ~80 lines

- [x] **T5b.4** `build-n4-module-kit/tests/slot-coverage.bats` — Merge QA branch `qa/c6-slot-coverage-red` tip `5a7d90a`; re-read tip at apply time. SC1–SC5 (pure subcommand), SC6 (parse subcommand: empty lexicon + 3 types → `pct=0.0` + WARN). **Named mutations**: denominator `|declared|` → SC2 reads `100.0`; `extra` in numerator → SC3 reads `125.0`; `0/0→100` → SC5 loses `N/A`; `missing = declared−required` → SC2/SC3 swap. · ~20 lines delta

- [x] **T5b.5** *(stretch; first-cut if T5b.1–T5b.4 reaches 400 lines)* `tests/fixtures/plano-sample.html` + `build-n4-module-kit/toolbelt/verify-module.sh --plano` — PL1: two `aspect-ratio` declarations → exit 1. **Named mutation**: count-only → PL1 passes. Record remainder in PR5b retro if deferred.

- [x] **T5b.6** PR5b retro + shellcheck sweep — Append retro (exit a). Run `shellcheck 0.10.0` on all `build-n4-module-kit/toolbelt/*.sh`. Full bats suite ≥104 pass ≤15 s.

---

## PR6 — feat/c6-close
**Gate exit (a)**: carry PR6 retro · **QA hook**: full bats suite ≥104 pass ≤15 s; `sweep-build-state.sh` exits 0; pending = only the Campaign 6 retros (PR1/PR3/PR4/PR5a/PR5b/PR6 — folded by PR7/close, not by PR6; the original 8 pending are 0); shellcheck 0

- [x] **T6.1** `build-n4-module-kit/skill/SKILL.md` — New tracked canonical copy. Remove `state` column from decision table (growing/mature/seed not actionable). Add explicit wb SEED warning directing wb builders to corpus B751–B753. Align §Execution step 1 with BUILD-LOOP orient-then-corpus-nav order. [spec R6-1] · ~25 lines

- [x] **T6.2** `scripts/install-skill.sh` — `[--home <dir>] [--dry-run] [--force]`; exits 0 (installed or current) / 1 (divergence, `--force` absent) / 2 (usage) / 3 (env: target dir not creatable). sha comparison vs tracked `build-n4-module-kit/skill/SKILL.md`; never resolves via `$HOME`; prose says "version control" not "git". Every test passes `--home "$BATS_TEST_TMPDIR/home"` → no test touches real `$HOME`. [design D5; spec R6-5] · shellcheck 0.10.0 · ~35 lines

- [x] **T6.3** `build-n4-module-kit/tests/install-skill.bats` — IS1: `--home "$BATS_TEST_TMPDIR/home"` install is byte-identical (`cmp`) to tracked copy; passes under `HOME=/nonexistent`. **Named mutation**: installer drops last line → `cmp` fails. · ~15 lines

- [x] **T6.4** Deploy — Run `scripts/install-skill.sh [--force]` targeting `~/.claude/skills/build-n4-module/SKILL.md`. Record before/after diff verbatim in PR6 description and save to engram. [spec R6-5] · 0 authored lines in repo (read-only runtime action)

- [x] **T6.5** `openspec/` — Move `openspec/changes/niagara-tools-slotomatic-integration/` → `openspec/changes/archive/niagara-tools-slotomatic-integration/`; add supersession note (100% superseded as of v0.15.1; all 40 tasks live in ng-deploy.sh + tests; 0 tasks applied). `git add openspec/` (hybrid-store durability). [spec R6-2/R6-3]

- [x] **T6.6** *(DONE IN PR4 f589956 — the DashboardPan ledger corrections landed with the types fold; PR6 only re-checks they survived and updates the kit self-envelope)* `build-n4-module-kit/BUILD-STATE.md` — Update ledger: DashboardPan verify_gate pass (rt 7/7, ux 7/7, 2026-09-05, repo HEAD 4f5f1c7); ColdRoomPan rt 7/7; CompPan rt 7/7 (same date). Fix `profiles: rt,ux,wb` → `profiles: rt,ux` + note (DashboardPan-wb is a scaffold: gradle/lexicon/palette/permissions, zero .java, never built). Reword open issue U5: handleSetpointWrite IS gated fail-closed by `DashboardRbacHelper.checkCanWrite` (BDashboardServlet.java:198); residue = lost pure-RBAC test seam, no per-Ord lock/423, optional allowlist (client punch-list, not kit) [ev: corpus B763]. · ~15 lines

- [x] **T6.7** `VERSION`, `CHANGELOG.md` — Set `VERSION=0.16.0`; rename `## [Unreleased]` → `## [v0.16.0] - 2026-09-05` + `### References` block (SDD slug build-n4-module-campaign6 + engram observation IDs); tag `v0.16.0` after merge. [spec R6-4] · ~5 lines

- [x] **T6.8** Guard + PR6 retro — `bats build-n4-module-kit/tests/` ≥104 pass ≤15 s; `sweep-build-state.sh` exits 0; pending = only the Campaign 6 retros (PR1/PR3/PR4/PR5a/PR5b/PR6 — folded by PR7/close, not by PR6; the original 8 pending are 0); `rg "rt only" build-n4-module-kit/build-verify.md` → 0 matches; shellcheck 0.10.0 on all `toolbelt/*.sh scripts/*.sh`. Append PR6 retro (exit a).

---

- [x] **T6.9** `.github/workflows/ci.yml` — Bump `actions/setup-java@v4` → `@v5` (deprecation notice observed in PR2 CI run 33956501710); no behavior change; CI must stay green with P1–P6 `ok`. · 1 line

---

- [x] **T6.10** `build-n4-module-kit/BUILD-STATE.md` — Ledger correction from B788 (own-modules-vs-exemplars OMV4): the kit-envelope open_issue "CompPan-rt/module.lexicon is empty (T8)" is FALSE — CompPan-rt/module.lexicon has 57 populated keys (verified 2026-09-05). Replace with the real findings: ColdRoomPan-rt lexicon partial (32 keys; fanMode/valveMode/freeze* missing → camelCase), DashboardPan-rt ~25% slot coverage, DashboardPan-wb palette is an empty scaffold (`<p t="b:Folder">` only). `[ev: corpus B788]`. · ~4 lines

---

- [x] **T6.11** `build-n4-module-kit/toolbelt/slot-coverage.sh` — Metric-honesty label (QA non-blocking note on PR5b): the parse mode measures TYPE-set coverage (a registered type counts as declared when it has ≥1 lexicon key), not per-slot completeness; add the word `(type-set)` to the parse-mode `pct=` line or usage text so `100.0` cannot be misread as per-slot coverage (B788 measured ~25% per-slot on DashboardPan-rt). Keep the pure `set-coverage` output unchanged (QA pins). Update the parse test assertion only if the pct line changes. · ~3 lines

---

## PR7 — feat/c6-research-fold (slot; research lanes still running)
**Gate exit (a)**: carry PR7 retro · **Doc-only: zero new bats tests** · Reuses PR3/PR4 mechanics verbatim

- [x] **T7.1** Research fold — Source = FOUR niagara-research §18 retros (all `review-status: pending` in ~/niagara-research/retros/): `2026-09-05-research-sdd-module-authoring-exemplars-FOCUS-CLOSE-retro.md` (B772–B785 + extension-idiom META-delta + B775/B777/B784 METHODOLOGY notes), `2026-09-05-research-sdd-own-modules-vs-exemplars-FOCUS-CLOSE-retro.md` (lintable-vs-advisory META-delta + human-review checklist + B790 minimal module), `2026-09-05-research-sdd-module-web-tier-exemplars-audit-retro.md` (DUX-WEB1 pointer table, DUX-WEB2 divergences), and the already-folded ux-testing retro (skip). The paste-ready bullets live in `openspec/changes/build-n4-module-campaign6/pr7-fold-draft.md` (grouped by destination + grep-before-fold term) and the corpus-index rows in `corpus-index-delta.md`. Grep-before-fold per bullet; cite `[ev: corpus B<n>]`. Destinations: types/logic.md (Author-side SPIs idiom + B778/B782/B785 + B773 exception; point extension B772; child-tree containers B779; grouping/relating B781; templates B783; background jobs B774; watchdogs/timers B775; action protection B776; minimal module B790), types/dashboard.md + wb-widgets.md (B780, DUX-WEB1/2), METHODOLOGY.md (B784, B775 note, B777 permissions-inline correction, lintable-vs-advisory rule + human-review checklist), corpus-index.md (26+ rows). · ~150 lines

- [x] **T7.2** Marking — The research retros are NOT rows of the kit's `retros/INDEX.md` (they live in niagara-research); after PR7 merges, the research lane (investigador1) stamps each of the three retros `<!-- review-status: applied … kit <sha> -->` in niagara-research (its sweep convention). In the kit: PR7 retro row + kit self-envelope only. · marker edits in the other repo (NOTE: this is a post-merge action by investigador1, not a kit file edit)

- [ ] **T7.3** *(candidate — OUT of this PR; schema-risk.sh design not finalized)* `build-n4-module-kit/toolbelt/schema-risk.sh` (MM3) — Two-snapshot slot-diff parser from `math-models-mm2-mm3.md §MM3`; pre-deploy guard. No design finalized; scope and budget TBD.

- [x] **T7.4** Guard + PR7 retro — `sweep-build-state.sh` exits 0. Append retro (exit a).

---

- [x] **T7.5** Fold-audit follow-up (from PR5a real-tree run): two retros are INDEX=folded with NO `[ev: retro …]` citation in the kit core — `2026-09-04-junit-standalone-cached-jar-locations-for-wsl-pure-tests.md` and `2026-09-04-kit-continuity-and-retro-gate-campaign.md`. For each: locate the folded content in the kit and add the citation token, or if the fold was structural (shipped code, no prose rule), add a one-line "folded as code: <script>" note carrying the `[ev: retro <slug>]` token in build-verify.md / METHODOLOGY.md so the audit credits it. Re-run `sweep-fold-audit.sh --strict` → exit 0; then switch the CI step to `--strict`. · ~6 lines

---

## PR8 — feat/c6-conformance-lints (slot; from the own-modules-vs-exemplars audit + models)

- [ ] **T8.1** Timer-ticket lint: a Java class owning a `Clock.Ticket` field / calling `Clock.schedule*` without a `stopped()` that cancels it → FAIL (verify-module `--src` check or `toolbelt/lint-timers.sh`). Fixtures from the real shapes (BEvaporatorUnit FAIL; BDefrostController/BCompressorControl PASS; no timers → PASS). QA RED: qa/c6-timer-ticket-lint. Named mutation: drop the stopped()-presence check → FAIL case passes. `[ev: corpus B787]` · ~60 lines
- [ ] **T8.2** Discarded `Clock.schedule` return-value grep (cheap regression guard) inside T8.1. · ~10 lines
- [ ] **T8.3** Empty-palette WARN scoped to modules WITH components (verify-module palette check refinement) — B788 DashboardPan-wb scaffold case; mutation: scope check removed → typeless module warns. `[ev: corpus B788]` · ~15 lines
- [ ] **T8.4** Retro-debt aging: `sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age 30]` (or `sweep-retro-debt.sh`) per contract `openspec/changes/build-n4-module-campaign6/math-models-e5.md` (age from filename date, strictly > max-age → ESCALATED, 0 pending → N/A, injected today). QA RED off the pin vectors (2026-08-06 → 30 not escalated; 2026-08-05 → 31 escalated). Non-strict CI step. · ~50 lines
- [ ] **T8.5** METHODOLOGY "human-review checklist" rules stay ADVISORY (action-without-OPERATOR, order-sensitive container without legality, poll-that-should-subscribe): NO lint; already folded as doctrine in PR7 — verify no hard-fail was added. · 0 lines
- [ ] **T8.6** PR8 retro + kit self-envelope (exit a).

---

## Client Punch-list (out of kit scope — module-repo tasks)

These items are tracked for continuity only. They are NOT kit tasks and must not be applied to niagara-tools.

- **B743-G1**: Concrete `Sched{at(delayMs);cancel(t)}` Java interface + fake-recorder unit test (JUnit) in the module repo that consumes the scheduler seam. Assigned: module author. [ev: corpus B743]
