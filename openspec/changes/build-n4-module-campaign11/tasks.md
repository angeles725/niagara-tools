# Tasks: build-n4-module-campaign11

**Source**: kit v0.21.0 (tag `dab0807`, C10 archive `154803c`) → **Target**: v0.22.0
**Chain**: stacked-to-main | 5 PRs | PR1 → PR2/PR3 (parallel after PR1) → PR4 → PR5

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1413 total (PR1 ~663, PR2 ~120, PR3 ~150, PR4 ~300, PR5 ~180) |
| 400-line budget risk | High (PR1 `size:exception` granted — ceiling 700; PR2–PR5 each under 400) |
| Chained PRs recommended | Yes |
| Suggested split | PR1 → PR2/PR3 (parallel after PR1) → PR4 → PR5 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | PR | Goal | Est. lines | Focused test | Runtime harness | Rollback |
|------|----|------|-----------|-------------|-----------------|---------|
| 1 | PR1 | Shared method-boundary parser — PEAK depth, 5 invariants (T1) | ~663 (`size:exception` ≤700) | `bats tests/parser-oneliner.bats && bats tests/golden-parser.bats` | 3×3 baselines byte-identical at main-ff1b659 | `git revert`; restores three in-script parsers + RED cherry-picks in same commit |
| 2 | PR2 | Concept-row-drift advisory in lint-write-path.sh (T3) | ~120 | `bats tests/lint-write-path.bats` | 0 DRIFT at client 00e7118; `--strict` exit 1 | `git revert`; STALE+FAIL byte-identical |
| 3 | PR3 | Centralise client-tree defaults — tests/lib/client-root.bash (T2) | ~150 | `bats tests/` with all four vars unset | LD5 clean exit 0; RC8 still 1 FAIL at ff1b659 | `git revert`; restores 10 literals + LD5 stale assertion |
| 4 | PR4 | Guard-pins meta-check lint-guard-pins.sh + header retrofit (T4) | ~300 | `bats tests/lint-guard-pins.bats` | 0 WARN over kit at dab0807 + PR1..PR3 | `git revert`; removes new script + routing rows |
| 5 | PR5 | C11 close — VERSION 0.22.0, CHANGELOG, 5 retros, sweeps | ~180 | `C11_CLOSE=1 bats tests/c11-close.bats` | N/A — doc/version only | `git revert`; retros return to `pending` |

---

## PR1 — feat/c11-shared-method-boundary (~663; `size:exception` ceiling 700)

**RED**: `qa/c11-parser-oneliner` tip **`d88af78`** + `qa/c11-golden-parser` tip **`ed2088f`** (7 cases; base `dab0807`) — re-read both tips at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c11-parser` (branch off kit main `dab0807`)
**D-ids**: D1, D1a-k · **Gate**: SC-1, SC-2, SC-3, SC-4, SC-4b, SC-9, SC-10, SC-11, SC-12

- [ ] 1.1 Re-read `git show origin/qa/c11-parser-oneliner:tests/parser-oneliner.bats` (tip `d88af78`) and `git show origin/qa/c11-golden-parser:tests/golden-parser.bats` (tip `ed2088f`): confirm C11-tl-oneliner exits 0 on `dab0807` (RED); C11-sp-oneliner exits 0 no WARN (RED); C11-g1-setter exits 0 (green guard); 7 golden case expected verdicts match the spec table. `[ev: K13; QA d88af78 / ed2088f; R-T1.9; R-T1.13]`
- [ ] 1.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c11-parser -b feat/c11-shared-method-boundary dab0807`. `[ev: D worktree map]`
- [ ] 1.3 Cherry-pick RED commits as commit 1 (ff-only kit: cherry-pick, never merge — C10 lesson): `git cherry-pick d88af78 ed2088f`; run `bats tests/parser-oneliner.bats` RED; record RED-for-the-right-reason (C11-tl-oneliner exit 0 = RED; C11-sp-oneliner 0 no WARN = RED). `[ev: K13; proposal §4; C10 lesson cherry-pick]`
- [ ] 1.4 Capture 3×3 real-tree baselines **BEFORE** cut: run `lint-timers.sh`, `lint-silent-protection.sh`, `lint-ext-writable-shape.sh` against CompPan-rt, ColdRoomPan-rt, DashboardPan-rt at `Leon-Guanjuato-worktrees/main-ff1b659` (`ff1b659`); record 9 verdicts verbatim (row text + exit code); these MUST be byte-identical after the cut. `[ev: D1j; SC-4; K21]`
- [ ] 1.5 Create `build-n4-module-kit/toolbelt/lib/method-boundary.sh`: define shell variable `$MB_AWK` as awk function-only text (legal in both inline and multi-`-f` mechanisms — D1b); implement `mb_strip(src,n,dst)` (`//` and `/* */` strip, block-state across lines, line numbers preserved) and `mb_parse(src,n,m_start,m_end,m_name)→cnt` (PEAK depth open: `max_d > old_d && max_d >= 2`; close: `brace_depth < m_dep` same iteration; Case-B backward scan stops at `@`; keyword exclusion `if|for|while|switch|catch|try|else|do|new`; one-line accessor skip: `(max_d > old_d && brace_depth <= old_d) && mname ~ /^(get|set|is)[A-Z_]/`; `/* */` strip in Case-B). `[ev: D1b; D1d; D1g; B832 593019540; R-T1.2..R-T1.7]`
- [ ] 1.6 Add source line to `build-n4-module-kit/toolbelt/lint-timers.sh` immediately after `set -u` (D1c): `. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"`; delete `:188-235` (48 lines); replace call site with `"$MB_AWK"'…'` adjacent-quote concat form; **verify exit map `0 / 1 / 2 usage / 3 env` at `:44`/`:58`/`:63` UNCHANGED** — lint-timers uses exit **2** for usage, unlike the other two lints. `[ev: D1a; D1e; R-T1.8; CERT-read lint-timers.sh:44,:58,:63]`
- [ ] 1.7 Add source line to `lint-silent-protection.sh`; delete `:302-373` (72 lines) **including the `:303-307` NET-depth rationale comment**; write `printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"` and add second `-f "$_TMP/method-boundary.awk"` before `main.awk` at `:538`; Case-B scan inherits `/* */` strip via `mb_strip` (B832-G2 pinned by 3×3 baselines, not by a vacuous fixture). `[ev: D1a; D1c; R-T1.11; CERT-read lint-silent-protection.sh:205,:524,:533-538]`
- [ ] 1.8 Replace `lint-silent-protection.sh` `:303-307` NET-depth rationale comment with the PEAK-depth rationale (D1g three bullets: single-line methods ARE entered as method bodies; runaway spans prevented by `brace_depth >= 2` guard + close-test in same iteration; `lint-ext-writable-shape` has shipped this ordering since C9). Verify no file in the kit still documents NET depth as correct after PR1. `[ev: D1g; R-T1.12; SC-4b]`
- [ ] 1.9 Source the fragment in `lint-ext-writable-shape.sh`; extract canonical `:132-176` into the fragment (reference implementation, no behavior change); replace `:137-178` (42 lines) with D1f 7-line array-iteration shape: `_n = mb_parse(slines, NR, ms, me, mn)` + `for (k=0;k<_n;k++) { if (!(mn[k] in do_methods)) continue; body=""; for (bi=ms[k];bi<=me[k];bi++) body = body " " slines[bi]; _scan_writes(body) }`; gains accessor skip from the fragment. `[ev: D1f; R-T1.15; R-T1.16; CERT-read lint-ext-writable-shape.sh:136-186]`
- [ ] 1.10 OBSERVED mutations — THREE real flips, each verbatim RED-then-GREEN naming the fixture (K24(7)): depth guard (drop `&& max_d >= 2`) → **S21-misparse** (`lint-timers.bats:436`) FALSE-FAILs; NET instead of PEAK → **C11-tl-oneliner, C11-sp-oneliner, G-oneliner-timers, G-oneliner-silent** go RED; drop the accessor skip → **G-accessor + C11-g1-setter** FALSE-WARN. I2 (Case-B `@`-stop), I3 (keyword exclusion) and I5 (`/* */` strip) are DEFENSIVE invariants proven redundant for lint output (investigador1 ad2121b69: the PEAK depth guard rejects the depth-1 class body before Case B runs; nested keywords are dead under `!in_m`); they stay in the fragment, documented, pinned by the aggregate golden set + 3×3 baselines — NOT carried as OBSERVED mutations. `[ev: SC-7; K24(7); ad2121b69]`
- [ ] 1.11 Run `bats tests/golden-parser.bats` GREEN: all 7 cases green with each `@test` asserting the exact triple verdict against all three lints simultaneously; G-oneliner-timers FAIL exit 1 (was RED); G-oneliner-silent WARN (was RED); G-accessor 0 WARN on all three. `[ev: D1i; SC-1; R-T1.9; R-T1.13]`
- [ ] 1.12 Capture 3×3 real-tree baselines **AFTER** cut; `diff` each against the pre-cut captures: **must be byte-identical** — 0 one-liner methods with schedule/alarm/trip write in the 42 `.java` at `ff1b659`; any delta is a defect and blocks the merge. `[ev: D1j; SC-4; R-T1.2]`
- [ ] 1.13 K19 routing: fragment-merge `BUILD-LOOP.md` + `skill/SKILL.md` to add `lib/method-boundary.sh` entry (mark as `library — sourced, not invoked`); run `bats tests/kit-links.bats` green; `shellcheck 0.10.0` exit 0 on all three lints + the new fragment; full `bats tests/` green with no C9/C10 pin shifts. `[ev: K19; D4h; SC-11]`
- [ ] 1.14 Retro: `toolbelt/new-retro.sh kit campaign11-shared-method-boundary`; fragment-merge always-conflict files (BUILD-LOOP.md, skill/SKILL.md, retros/INDEX.md, BUILD-STATE.md — append, keep both rows, dedupe by script name, never overwrite); commit `fix(kit): T1 shared method-boundary parser — PEAK depth, I1-I5 invariants` (0 attribution trailers — K11); push; rebase onto kit main before QA ping; verify `git log -1` tip == blessed. `[ev: K11; SC-12]`
- [ ] **[lead]** All 7 golden cases green on all three lints in one `@test` each · C11-tl-oneliner FAIL (was exit 0) OBSERVED · C11-sp-oneliner WARN (was 0) OBSERVED · C11-g1-setter 0 WARN; accessor-skip removal → FALSE-WARN OBSERVED · I1-I5 all OBSERVED naming S21-misparse / G-samemethod / C11-tl-oneliner+C11-sp-oneliner / C11-g1-setter · 3×3 baselines byte-identical before/after · `:303-307` comment replaced · full `bats tests/` green · `shellcheck` 0 · 0 trailers · `git diff --stat` measured ≤700 · ledger settle.

---

## PR2 — feat/c11-concept-row-drift (~120)

**RED**: `qa/c11-concept-drift` tip **`77352a7`** (WP-drift-neg / WP-drift-strict RED on `dab0807`; WP-drift-true-concept + WP-drift-decoy guards; base `dab0807`) — PR does not open until RED is blessed; re-read tip at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c11-drift` (branch off kit main `dab0807`)
**D-ids**: D2, D2a-f · **Gate**: SC-5, SC-9, SC-10, SC-11, SC-12
**Note**: may run in parallel with PR3 after PR1 merges; touches `toolbelt/lint-write-path.sh` only, no `tests/lib` overlap.

- [ ] 2.1 Re-read `git show origin/qa/c11-concept-drift:tests/lint-write-path.bats` (tip `77352a7`): confirm WP-drift-neg exits 0 + 1 DRIFT row; WP-drift-strict exits 1; WP-drift-true-concept exits 0 / 0 DRIFT; WP-drift-decoy 0 DRIFT (HTML-comment strip). `[ev: K13; D2b; R-T3.1..R-T3.10]`
- [ ] 2.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c11-drift -b feat/c11-concept-row-drift dab0807`. `[ev: D worktree map]`
- [ ] 2.3 Cherry-pick RED as commit 1; run `bats tests/lint-write-path.bats` RED; record RED-for-the-right-reason (WP-drift-neg DRIFT row absent on `dab0807`). `[ev: K13]`
- [ ] 2.4 Modify `lint-write-path.sh` at seam `:441` (D2a): replace unconditional `case "$_row" in *'[concept]'*) continue ;; esac` with `_is_concept=0; case "$_row" in *'[concept]'*) _is_concept=1 ;; esac`; at covered-set branch `:450` add DRIFT branch for `_is_concept=1` (emit DRIFT row, `DRIFT=1`); add `[ "$_is_concept" -eq 1 ] && continue` for true-concept miss (slot absent → silent). DRIFT row grammar STATUS-first (R-T3.5): `DRIFT  lint-write-path  <matrix-path>:<line>  slot <name>: concept marker but a source slot exists`. `[ev: D2a; D2b; CERT-read lint-write-path.sh:432-455]`
- [ ] 2.5 Update exit expression `:462-464` (D2e): `[ "$FAILED" -eq 1 ] && exit 1`; `[ "$STRICT" -eq 1 ] && { [ "$STALE" -eq 1 ] || [ "$DRIFT" -eq 1 ]; } && exit 1`; `exit 0`. Verify exit 3 (usage) emitted upstream at `:137` — range stays `{0,1} ∪ {3}` (K20). `[ev: D2e; K20; R-T3.6; R-T3.7; R-T3.8; R-T3.9]`
- [ ] 2.6 Verify STALE rows and uncovered FAIL exit-1 byte-identical to v0.21.0 (D2b truth table — plain+uncovered unchanged); verify covered set reused from `_covered_flat` at `:432`, no second harvest (D2d). `[ev: D2b; D2d; R-T3.8]`
- [ ] 2.7 OBSERVED mutations (attributed — second read 7e9669480): (a) drop the DRIFT emit → WP-drift-neg + WP-drift-strict RED; (b) delete the covered-branch `_is_concept` guard OR skip the `:439` comment strip → WP-drift-decoy RED (a commented `[concept]` marks the row → covered slot false-DRIFTs); (c) delete the true-concept skip (`[ "$_is_concept" -eq 1 ] && continue`) → WP-drift-true-concept RED (the uncovered concept row falls through to STALE). Record verbatim RED-then-GREEN for each. `[ev: SC-7; K24(7); 7e9669480]`
- [ ] 2.8 Real-tree pin at client `00e7118`: run `lint-write-path.sh` on the worktree `Leon-Guanjuato-worktrees/main-ff1b659` updated to `00e7118`; must report **0 DRIFT rows**, exit 0 (five PR6 `[concept]` rows — `hoaMode`, `inhibit`, `freezeEnabled`, `setpoint`, `coolOnSensorFault` — have no source slot at `00e7118`). `[ev: R-T3.11; K21; SC-5]`
- [ ] 2.9 Run `bats tests/lint-write-path.bats` GREEN: WP-drift-neg exits 0 + 1 DRIFT row; WP-drift-strict exits 1; WP-drift-true-concept 0 DRIFT; WP-drift-decoy 0 DRIFT; STALE rows unchanged; uncovered FAIL exit 1 byte-identical; exit 3 preserved. `[ev: SC-5; R-T3.1..R-T3.10]`
- [ ] 2.10 `shellcheck 0.10.0` exit 0; full `bats tests/` green; fragment-merge always-conflict files; retro `toolbelt/new-retro.sh kit campaign11-concept-row-drift`; commit `fix(kit): T3 concept-row-drift advisory in lint-write-path (77352a7)` (0 trailers — K11); push; rebase before QA ping. `[ev: SC-12; K11]`
- [ ] **[lead]** WP-drift-neg exits 0 + 1 DRIFT row · WP-drift-strict exits 1 · WP-drift-true-concept 0 DRIFT · WP-drift-decoy 0 DRIFT (HTML-comment strip) · STALE rows unchanged · uncovered FAIL exit 1 byte-identical with and without `--strict` · exit 3 preserved · real tree `00e7118` → 0 DRIFT · OBSERVED flip (D2f) · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR3 — feat/c11-client-root-lib (~150)

**RED**: `qa/c11-client-root` tip **`54078f6`** (C11-T2-lib-exists + C11-T2-no-hardcode: 10 → 0; LD5 flips to clean on `ff1b659`; RC8 unchanged; base `dab0807`) — re-read tip at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c11-clientroot` (branch off kit main `dab0807`)
**D-ids**: D3, D3a-f · **Gate**: SC-6, SC-7, SC-9, SC-10, SC-11, SC-12
**Note**: may run in parallel with PR2 after PR1 merges; touches `tests/` only, no toolbelt overlap with PR2.

- [ ] 3.1 Re-read `git show origin/qa/c11-client-root:tests/lib/client-root.bash` and `:tests/lint-delays.bats` (tip `54078f6`): confirm lib-exists RED (file absent on `dab0807`); confirm 10 `Leon-Guanjuato` literals in `tests/*.bats` outside lib (no-hardcode RED); confirm LD5 expected clean exit 0 + `BDefrostController` absent; confirm RC8 unchanged 1 FAIL `host`. `[ev: K13; R-T2.1..R-T2.9]`
- [ ] 3.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c11-clientroot -b feat/c11-client-root-lib dab0807`. `[ev: D worktree map]`
- [ ] 3.3 Cherry-pick RED as commit 1; run `bats tests/` RED (lib-exists + no-hardcode RED — 10 literals found); record RED-for-the-right-reason. `[ev: K13]`
- [ ] 3.4 Create `tests/lib/client-root.bash` (D3a): `: "${CLIENT_READ_ROOT:=…/Leon-Guanjuato-worktrees/main-ff1b659}"` + `: "${C9_CLIENT_ROOT:=$CLIENT_READ_ROOT}"` + `: "${C9_CLIENT_REPO:=$CLIENT_READ_ROOT}"` + `: "${C8_CLIENT_REPO:=$CLIENT_READ_ROOT}"` + `export CLIENT_READ_ROOT C9_CLIENT_ROOT C9_CLIENT_REPO C8_CLIENT_REPO`. `:=` gives env-override-wins with no branch (R-T2.3). `[ev: D3a; R-T2.1; R-T2.2; apply-package 0ad09c658]`
- [ ] 3.5 Add `load lib/client-root` at file scope (not inside `setup()`) to each of the 10 affected bats files per the D3b kit idiom: `tests/{ext-writable-shape,demand-in-scope,lint-silent-protection,lint-timers,lint-write-path,c9-close,c10-close,c8-close,lint-delays,rc-scan}.bats`. `[ev: D3b; R-T2.4; CERT-read preflight.bats:24]`
- [ ] 3.6 Convert all 10 literal sites per D3c table: sites 1-5 (`C9_CLIENT_ROOT`) → `ROOT="$C9_CLIENT_ROOT"`; sites 6-7 (`C9_CLIENT_REPO`) → `R="$C9_CLIENT_REPO"`; site 8 (`C8_CLIENT_REPO`) → `R="$C8_CLIENT_REPO"`; site 9 (`lint-delays:53` bare `$HOME`) → `CRP="$C9_CLIENT_ROOT/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"`; site 10 (`rc-scan:75` bare `$HOME`) → `UX="$C9_CLIENT_ROOT/Dashboard/DashboardPan/DashboardPan-ux"`. `[ev: D3c; R-T2.4; R-T2.5]`
- [ ] 3.7 Retarget LD5 in `tests/lint-delays.bats` (D3d): change `:56-57` to `[ "$status" -eq 0 ]` + `[[ "$output" != *"FAIL"* ]]`; re-cut test title to name the clean state of the blessed tree. Commit body MUST carry verbatim: (i) `lint-delays.sh` output + exit on `4f5f1c7` (exit 1, FAIL `BDefrostController`), (ii) same on `main-ff1b659` (exit 0, no FAIL), (iii) the sentence that the rule is pinned by synthetic fixtures `LD1`/`LD3`/`LD6`/`LD11-misguard`. `[ev: D3d; R-T2.6; R-T2.7; SC-7]`
- [ ] 3.8 Verify `c8-close.bats` SC1-smoke: root-only change (site 8), assertion `status 1 || 0` unchanged. Verify `rc-scan.bats` RC8 assertion unchanged (1 FAIL `host` — the `:701` host literal IS present at `ff1b659`; the FAIL is the rc-scan rule firing correctly on real code, not a transient defect). RK5 audit: classify every remaining real-tree smoke — a smoke asserting FAIL/WARN is correct when it reflects the tree's current CORRECT verdict (the lint rule should fire on what is there); it is wrong only when it pins a previously-present defect that has since been fixed. LD5 old assertion (exit 1 + FAIL BDefrostController) was wrong-class: the defect was fixed and the lint rule is separately pinned by LD1/LD3/LD6. Record audited list in PR body (D3f). `[ev: D3e; D3f; R-T2.8; R-T2.9; R-T2.10; coordinator smoke-assertion-class amendment 2026-09-07]`
- [ ] 3.9 Verify full suite green with `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, `C8_CLIENT_REPO` all **unset** (default resolves); one override pin per variable proving env wins; LD5 exits 0 clean (`BDefrostController` absent from output); LD1/LD3/LD6/LD11-misguard still FAIL on synthetic inputs; RC8 still 1 FAIL `host`. `[ev: SC-6; SC-7; R-T2.3; K22]`
- [ ] 3.10 Verify NO toolbelt script was modified (K12); `shellcheck 0.10.0` exit 0; fragment-merge always-conflict files; retro `toolbelt/new-retro.sh kit campaign11-client-root`; commit `fix(kit): T2 centralise client-tree defaults — tests/lib/client-root.bash (54078f6)` (0 trailers — K11); push; rebase before QA ping. `[ev: K11; K12; SC-12]`
- [x] **[lead]** 10 → 0 absolute `Leon-Guanjuato` literals outside the lib (OBSERVED) · full suite green with all four vars unset · one override pin per variable · LD5 exit 0 + `BDefrostController` absent (retargeted from stale `4f5f1c7`) · LD1/LD3/LD6/LD11-misguard green · c8-close SC1-smoke root-only (assertion unchanged) · RC8 1 FAIL `host` unchanged · RK5 audit list in PR body · no toolbelt script in diff · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR4 — feat/c11-lint-guard-pins (~300)

**RED**: `qa/c11-guard-pins` tip **`ebc15e8`** (tests: T4-usage, T4-match, T4-nofixture, T4-nomutation, T4-scope, T4-strict, T4-smoke; base `dab0807` — real-kit smoke is RED with **9 WARN** on `dab0807`; after PR4 expects ≥10 MATCH over 10 distinct scripts, 0 WARN) — re-read tip at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c11-guardpins` (branch off kit main AFTER PR1-PR3 merge)
**D-ids**: D4, D4a-i · **Gate**: SC-8, SC-9, SC-10, SC-11, SC-12
**Apply support**: companero apply-support map `niagara-research 08149bcd0`; close package `e12ce2815`
**Depends on**: PR1-PR3 merged (real-kit self-verify smoke at step 4.6 requires all prior changes)

- [ ] 4.1 Re-read `git show origin/qa/c11-guard-pins:tests/lint-guard-pins.bats` (tip `ebc15e8`; K13): confirm T4-usage (exit 3); T4-match (declared mutation maps to fixture → MATCH row, 0 WARN); T4-nofixture (declared mutation with no fixture → 1 WARN, exit 0); T4-nomutation (no mutation line in header → 1 WARN); T4-scope (non-lint toolbelt script not scanned); T4-strict (1 WARN + `--strict` → exit 1); T4-smoke (real-kit on `dab0807` is **RED: 9 WARN** — 9 lint-*.sh have no mutation header yet; after D4e retrofit → 0 WARN). `[ev: K13; D4g; R-T4.1..R-T4.9; ebc15e8]`
- [ ] 4.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c11-guardpins -b feat/c11-lint-guard-pins <main-after-PR1-PR3>`. `[ev: D worktree map]`
- [ ] 4.3 Cherry-pick RED as commit 1; run `bats tests/lint-guard-pins.bats` RED; record RED-for-the-right-reason (tool absent on `dab0807` — GP-pos 1 WARN missing). `[ev: K13]`
- [ ] 4.4 Create `build-n4-module-kit/toolbelt/lint-guard-pins.sh` (D4a-D4d): scan `toolbelt/lint-*.sh` **only** (D4b scope rule stated in header; non-lint scripts OUT); lines 1-60 per file; declaration ERE: `^# Mutation: [A-Za-z][A-Za-z0-9_-]* -- .+$`; fixture-set extraction: `grep -hoE '^@test "[^":]+:' tests/*.bats | sed -E 's/^@test "([^":]+):.*/\1/' | sort -u`; emit MATCH/WARN rows (D4d): a declared id missing from the fixture set → WARN; zero mutation lines → WARN (R-T4.2); D9b dot-dir prune; `--strict` → exit 1; usage → exit 3 (K20). `[ev: D4c; D4d; R-T4.1..R-T4.8]`
- [ ] 4.5 Retrofit `# Mutation:` header lines to the 9 existing lint scripts (D4e) — 13 fixture ids (11 verified by investigador1 `eae61fb27`, plus LS7@lint-structure.bats:44 and WBT1c@lint-wb-threading.bats:32 verified at c620af37d) at their exact lines: `lint-timers.sh` → `# Mutation: S21-misparse -- …` (`:436`) + `# Mutation: S21-neg -- …` (`:367`); `lint-silent-protection.sh` → `S23-pos` (`:188`) + `S23-and` (`:270`); `lint-ext-writable-shape.sh` → `EW-s22-neg2` (`:227`) + `EW-s22-nondo` (`:245`); `lint-write-path.sh` → `WP-stale-perrow` (`:250`) + `WP-stale-concept-decoy` (`:237`); `lint-demand-scope.sh` → `DS2` (`demand-in-scope.bats:51`); `lint-delays.sh` → `LD1` (`lint-delays.bats:28`); `lint-servlet.sh` → `LSV1` (`lint-servlet.bats:25`); `lint-structure.sh` → `LS7` (`lint-structure.bats:44`); `lint-wb-threading.sh` → `WBT1c` (`lint-wb-threading.bats:32`). Add `# Mutation: GP-pos -- …` to `lint-guard-pins.sh` own header. Total: **14 `# Mutation:` lines across 10 lints**. `[ev: D4e; investigador1 eae61fb27 — all 11 fixture ids PASS; CERT-read]`
- [ ] 4.6 Self-verify (D4f / T4-smoke): run `lint-guard-pins.sh` over the kit in this worktree (= `dab0807` + PR1..PR3 + D4e retrofit); confirm T4-smoke flips from RED 9 WARN on `dab0807` to: (i) lint-`*.sh` count = **10**; (ii) distinct scripts in MATCH rows = **10**; (iii) one exact MATCH row `MATCH  lint-guard-pins  …/lint-timers.sh:<n>  mutation S21-misparse -> tests/lint-timers.bats:436`; (iv) WARN count = **0**. Any remaining WARN rows are the finding and MUST be fixed in this PR. `[ev: D4f; R-T4.9; SC-8; ebc15e8 T4-smoke]`
- [ ] 4.7 OBSERVED mutation: delete a bats fixture that is the sole carrier for a named mutation → `lint-guard-pins.sh` exits 0 with 1 WARN for the unmatched mutation; record verbatim RED-then-GREEN naming the fixture; restore → 0 WARN. `[ev: K24(7); R-T4.9]`
- [ ] 4.8 Run `bats tests/lint-guard-pins.bats` GREEN: GP-pos 1 WARN exit 0; GP-nomut 1 WARN; GP-neg 0 WARN; GP-strict exit 1; GP-usage exit 3; GP-prune 0 WARN; GP-decoy 0 WARN; GP-real MATCH = 10 + exact row + WARN = 0. `[ev: D4g; SC-8; R-T4.1..R-T4.9]`
- [ ] 4.9 K19 routing: fragment-merge `BUILD-LOOP.md` + `skill/SKILL.md` to add `lint-guard-pins.sh` routing rows AND `lib/method-boundary.sh` as `(library — sourced, not invoked)` if not already present from PR1; run `bats tests/kit-links.bats` green; K20 disjoint exits verified **per lint's own documented contract** — `lint-guard-pins.sh` uses 0/1/3; `lint-timers.sh` uses **0/1/2 usage/3 env** (`:58`/`:63` — do NOT assert usage=3 for lint-timers in a generic check); D9b prune pinned by GP-prune fixture. `[ev: D4h; K19; K20; D9b; investigador1 eae61fb27 — lint-timers exit 2 usage nuance]`
- [ ] 4.10 Fold K24(7) enforcement as doctrine in `build-n4-module-kit/METHODOLOGY.md` (the `# Mutation:` grammar becomes doctrine); `shellcheck 0.10.0` exit 0; full `bats tests/` green; fragment-merge always-conflict files; retro `toolbelt/new-retro.sh kit campaign11-guard-pins`; commit `fix(kit): T4 lint-guard-pins meta-check + 14 Mutation header lines` (0 trailers — K11); push; rebase before QA ping. `[ev: D4i; K11; SC-12]`
- [ ] **[lead]** GP-pos 1 WARN · GP-nomut 1 WARN · GP-neg 0 WARN · GP-strict exit 1 · GP-usage exit 3 · GP-prune 0 WARN · GP-decoy 0 WARN · GP-real MATCH = 10 + exact row + WARN = 0 · D4e retrofit complete (14 `# Mutation:` lines, all fixture ids verified at base) · OBSERVED flip (fixture deletion → 1 WARN) naming fixture · K19 routing in both files · scope rule D4b in header · `shellcheck` 0 · 0 trailers · ledger settle.

---

## PR5 — chore/c11-close (~180)

**RED**: `qa/c11-close-checklist` tip **`97624cb`** (skeleton; QA freezes `TODO(freeze)` pins at close) — re-read tip at apply (K13)
**Repo**: `niagara-tools` · **Worktree**: `niagara-tools-worktrees/c11-close` (branch off kit main AFTER PR1-PR4 merge)
**D-ids**: D5 · **Gate**: SC-13, SC-9, SC-10, SC-11, SC-12
**Depends on**: PR1-PR4 merged; `retros/INDEX.md` pending = 0 before PR5 opens

- [ ] 5.1 Re-read `git show origin/qa/c11-close-checklist:tests/c11-close.bats` at skeleton tip (K13): confirm `C11_CLOSE=1` guard; BASE pin `dab0807`; `TODO(freeze)` on VERSION/tag/SC-13; tool-pins include `lib/method-boundary.sh` + `lint-guard-pins.sh`; mirrors `c10-close.bats:16-45` shape. `[ev: K13; D5; R-C11.4]`
- [ ] 5.2 Create worktree: `git worktree add ../niagara-tools-worktrees/c11-close -b chore/c11-close <main-after-PR1-PR4>`. `[ev: D worktree map]`
- [ ] 5.3 Cherry-pick RED skeleton as commit 1; run `C11_CLOSE=1 bats tests/c11-close.bats` — all pins inert/skip (RED-for-the-right-reason before `TODO(freeze)` pins filled). `[ev: K13; D5]`
- [ ] 5.4 Create five retro stubs with `toolbelt/new-retro.sh kit <slug>`: `campaign11-shared-method-boundary`, `campaign11-concept-row-drift`, `campaign11-client-root`, `campaign11-guard-pins`, `campaign11-close-process-meta-lessons`. Each stub produces an INDEX row. Fold campaign lessons (best-effort): (a) real-tree smoke must pin current clean state, not a known bug (LD5 lesson — R-T2.6/R-T2.10); (b) N copies of a logic fragment silently diverge (three-parser lesson — R-T1.1). Record C12 seeds: 3 legacy NAMED MUTATION prose lines in non-lint scripts (D4b), RK5 smoke-class rule (D3f), B832-G2 missing biting fixture. `[ev: D5; R-C11.3; R-C11.9]`
- [ ] 5.5 Flip all five retros to `folded` in `retros/INDEX.md`; run `toolbelt/sweep-fold-audit.sh --strict` exit 0 (0 uncited). `[ev: R-C11.3; R-C11.8]`
- [ ] 5.6 Update `CHANGELOG.md`: rename `## [Unreleased]` → `## [v0.22.0] - 2026-09-<dd>`; add `### Changed — Campaign 11: shared parser (T1), DRIFT advisory (T3), client-root lib (T2), guard-pins meta-check (T4)` with one bullet per PR citing `[ev: retro campaign11-<slug>]`. `[ev: R-C11.2; CONTRIBUTING §4-5]`
- [ ] 5.7 Update `VERSION` at repo root (not in kit — `c10-close.bats:39` precedent): `0.21.0` → **`0.22.0`** in the same commit as the CHANGELOG (CONTRIBUTING §5). `[ev: D5; R-C11.1; CERT-read VERSION c10-close.bats:39]`
- [ ] 5.8 Coordinate with QA to freeze `TODO(freeze)` pins in `tests/c11-close.bats` against the actual merged tip: VERSION `0.22.0`, tag `v0.22.0`; tool-pins include `toolbelt/lib/method-boundary.sh` + `toolbelt/lint-guard-pins.sh`; client versions carry over 2.2.0/2.1.0/2.2.0 (C11 ships no jar, no `vendorVersion` bump). `[ev: D5; R-C11.4; SC-13]`
- [ ] 5.9 Run all close gates: `C11_CLOSE=1 bats tests/c11-close.bats` green; `toolbelt/sweep-build-state.sh` exit 0; `toolbelt/sweep-fold-audit.sh --strict` exit 0; full `bats tests/` green; **grep entire PR1-PR5 commit range for `Co-Authored-By` / AI attribution trailers — count must be 0** (K11). `[ev: R-C11.6; R-C11.8; K11]`
- [ ] 5.10 `BUILD-STATE.md` envelope: `retro_pending: false`; append `last_session: 2026-09-<dd> · Campaign 11 CLOSE v0.22.0 — T1 shared parser (PEAK), T3 DRIFT, T2 client-root lib, T4 guard-pins; 5 retros folded; client versions carry over 2.2.0/2.1.0/2.2.0 (C11 no jar).` Fragment-merge always-conflict files; `shellcheck` 0; commit `chore(c11-close): v0.22.0 — CHANGELOG+VERSION, 5 retros folded, BUILD-STATE flip` (0 trailers — K11); push; open PR; verify `git log -1` tip before settle. `[ev: K11; SC-12]`
- [ ] **[lead]** `c11-close.bats` green under `C11_CLOSE=1` · BASE `dab0807` · VERSION `0.22.0` + CHANGELOG · both sweeps exit 0 · `retros/INDEX.md` pending = 0 · tool-pins include `lib/method-boundary.sh` + `lint-guard-pins.sh` · SC-13 carry-over (no jar: 2.2.0/2.1.0/2.2.0) · 0 attribution trailers in whole PR1-PR5 range (K11) · post-merge: `git tag v0.22.0 <sha> && git push origin v0.22.0` · `scripts/install-skill.sh` · sdd-archive · ledger settle.
