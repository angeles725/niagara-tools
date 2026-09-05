<!-- review-status: pending -->
<!-- Marker lifecycle: maintainer flips 'pending' → 'applied <date> · kit <sha>' (or 'dismissed') once folded; sweep-retros.sh reads this (METHODOLOGY §18). -->
# Retro — niagara-research · companero session · 2026-09-05 · B792 (palette-lexicon-agents tool + census) + B793 (§19 build/PoC) — kit deltas

> §18 self-retro for the `companero` session, the INCREMENTAL work AFTER the focus-close retro
> (`2026-09-05-research-sdd-module-authoring-exemplars-FOCUS-CLOSE-retro.md`, which already carries B772's
> point-extension delta). Scope here: the research-tooling block B792, the build/PoC block B793, and the
> research-sdd kit observations. READ-ONLY on both kits — PROPOSES only, never applies.

## A. → `build-n4-module-kit`

### A1. `toolbelt/` — the dup-lexicon-key FAIL check (PR5b/PR7 rationale) — from **B792**
- A lint that FAILS on duplicate keys in a module's `.lexicon` (the B759 hazard) now has cross-corpus evidence
  it catches real defects: **57 / 662** modules corpus-wide carry duplicate bare keys (**10 / 37** within the
  module-authoring exemplar set), worst `schedule-rt` `summary` **×3** (two silent overrides). investigador1's
  B788 audit of the operator's 3 modules found ZERO — so the check is clean on our code AND bites on Tridium's.
- Extractor to reuse: the tracked, stdlib-only `tools/palette-lexicon-agents.py` (`find_duplicate_keys` is the
  exact rule; `--all` runs the census). Committed this session (166513261).

### A2. `types/logic.md` "minimal module" + `toolbelt/scaffold-module.sh` (PR9 candidate) — from **B793**
- B790's minimal skeleton was proven to BUILD gate-green (verify-module.sh exit 0: bytecode 52, signed, types,
  baja 4.14, palette). But the real build forced **6 genuine B790 spec corrections** — the scaffolder must emit
  the CORRECTED shapes, not B790's INFER text:
  1. source lexicon file = `module.lexicon` (plugin renames to `<MOD>-<profile>.lexicon` in the jar).
  2. source type manifest = `module-include.xml` (plugin generates `META-INF/module.xml`; do NOT hand-author it).
  3. class body has NO hand AUTO region — slotomatic generates `//region /*+…+*/ … //endregion`.
  4. dev MUST hand-write `do<Action>()` (Baja calls `doTickExpired()`, not the generated `tickExpired()` wrapper).
  5. profile gradle file = `<MOD>-<profile>.gradle.kts` (findProjects convention), not `build.gradle.kts`.
  6. `preferredSymbol` is auto-assigned by the plugin (profile-dir name); a custom symbol needs a manifest prop
     the minimal `moduleManifest{}` block lacks — so a palette `m="<sym>=…"` must match the auto symbol.
- RED→GREEN fixture: the built `~/modulos_niagara_n4/_scratch/MinimalPan` (KEPT, do not delete). Test contract:
  `scaffold-module.sh <MOD> <vendor> <symbol>` → `build.sh` exit 0 + `verify-module.sh` ALL PASS + B788 biting
  checks pass (palette non-empty, no dup lexicon keys, Clock.Ticket has a `stopped()`-cancel); a mutation that
  empties the palette / dups a key / drops the `stopped()`-cancel must FAIL.

## B. → research-sdd kit (for explorador's team)

- **B1. `verify-state.sh` shared-global false FAIL / STALE bleed.** In a multi-focus corpus, `research-sdd-status.sh
  --next --focus <f>` returns STALE because `verify-state.sh` still scans ALL `RESEARCH-STATE*.md` and one other
  focus's `covered_blocks != <total-on-disk>` FAIL propagates to the scoped query. `--focus` scopes the backlog
  parse but not the consistency scan. Confirmed independently by explorador (6 focuses); upstream fix approved.
  Workaround used this session: treat the cross-focus FAIL as noise, and hand-edit envelope counters (never
  `--sync-state`, which would write the corpus total into a per-focus `covered_blocks`).
- **B2. tool-registry.md gap.** `toolbelt/tool-registry.md` lists no `module_nav`/module-navigator commands, so the
  new `palette-lexicon-agents` command/script was NOT registered there. Decision deferred to the kit team: start
  listing module_nav-family commands, or leave module-navigator self-documented in its own README.

## C. → build kit `toolbelt/preflight.sh` (tooling gap found during B793)
- `preflight.sh` reports `FAIL jdk8` for the WSL `openjdk-8` at `/usr/lib/jvm/java-8-openjdk-amd64` because it
  keys on a `release` file the Debian/WSL package omits. `build.sh` uses `[ -d $J8 ]` and builds fine, so the
  FAIL is a false negative — but it is misleading. Proposal: preflight should also accept a `bin/java` whose
  `-version` reports 1.8 when the `release` file is absent.

## D. Process notes (multi-session coordination — for the team, not a kit)
- **Worktree-per-lane** eliminated the shared-checkout index collisions: every block this session (B792, B793,
  the tool port) was authored in a fresh `~/niagara-research-worktrees/<lane>` off `origin/main`, so the ~10
  uncommitted kit-team `review-status` flips in the main checkout never interfered, and `pull --rebase` stayed clean.
- **Last-pusher recomputes envelope counters by hand** worked: when investigador1's B778 and my B772 both landed,
  the naive git auto-merge collapsed both `covered_blocks 0→1` to `1`; I recomputed the true aggregate (2) at push.
- **Verify-before-accept paid off twice**: (1) it caught a wrong second-reader refutation on B772 (the peer cited
  `:342`, which is `getExtensions()`, not the exec loop) — I re-read source and kept the correct cite; (2) it
  caught that B790's "gate-green by construction" needed 6 real corrections before it was truly [CERT].

## Delta index (block → destination)
- B792 → build-kit toolbelt (dup-key check rationale) + tools/palette-lexicon-agents.py (tracked extractor).
- B793 → build-kit `types/logic.md` "minimal module" + `toolbelt/scaffold-module.sh` (PR9) with the 6 corrections.
- research-sdd kit → B1 (STALE bleed), B2 (tool-registry), C (preflight jdk8 false-negative).
