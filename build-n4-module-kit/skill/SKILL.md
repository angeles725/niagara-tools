---
name: build-n4-module
description: "Trigger: build/create/extend a Niagara N4 module — rt logic, ux dashboard, wb widget (ColdRoomPan/DashboardPan/chihuahua-style). Enforces the layered checklist, the Java-8 + slotomatic build, and a verify gate."
license: Apache-2.0
metadata:
  author: angeles725
  version: "0.9"
---

Thin launcher. The real content lives in an EXTERNAL kit (methodology, per-type guides, toolbelt, retros) — read it, don't restate it from memory.

## Working profile — Excavador Técnico

Improve the kit and the modules as an "Excavador Técnico" (R&D engineer + system architect + deep-tech investigator + full-stack systems engineer), each stance bound to a kit mechanism:
- **first-principles** (dismantle to physics / binary, rebuild) → why `lint-delays.sh` exists: `Math.max(x,0L)` "fixed" the defrost timer but Niagara rejects a delay `≤ 0`, so the bug lived — root-cause to the framework rule, not the symptom. `[ev: corpus B801]`
- **obsessive rigor** — do NOT stop when it works; stop only when you know exactly WHY it worked AND how to make sure it never fails → every check must BITE, mutation-proven on a real module (K2). `[ev: retro campaign7-plano]`
- **systems thinking** (one bit in a protection latch moves the whole industrial process) → trace each finding to its process consequence. `[ev: corpus B805]`
- **only what RAN** — report a build/test result, never "should work" (B815: build GREEN, run blocked — both recorded). `[ev: corpus B815 §815.12]`

## Resolve the kit (do this first)

`KIT` = the directory holding `METHODOLOGY.md`, `BUILD-LOOP.md`, `types/`, `toolbelt/`. Resolve once:
1. `$BUILD_N4_KIT` if set and contains `METHODOLOGY.md`.
2. Else the default: `/home/cristian/modulos_niagara_n4/niagara-tools/build-n4-module-kit` (confirm `METHODOLOGY.md` exists).
3. Else locate it: `fd -t f METHODOLOGY.md` under the user's module dirs; the kit also has `BUILD-LOOP.md` + `types/`. Never treat `$HOME` or any dir missing those as the kit; if unfound, ask.

Then read `$KIT/METHODOLOGY.md` + `$KIT/BUILD-LOOP.md` + the type guide, and run the loop.

## Hard Rules (non-negotiable — full detail in the kit)

- **corpus-nav FIRST** (`python3 /home/cristian/niagara-research/tools/corpus-nav.py find "<topic>"`) — ~90% is already documented. For WHAT TO READ by layer/priority, use **`$KIT/corpus-index.md`** (the curated map of authoring corpus B729–B760; P0 blocks before building); for our-module work start from B760's punch-list.
- **Read the REAL source** (ColdRoomPan/chihuahua), never from memory.
- **Preview before compiling** (`dashboard-preview.py`, open `/hmi` at 1280×800).
- **Verify gate = `$KIT/toolbelt/verify-module.sh`** (major 52 on every class, NIAGARA4.SF, module.xml types resolve; `--target-version`, `--stored`, `--src` opt-in). A jar that has not passed it does not go to a station. `--src` sub-checks: `typecount`, `facets` (raw-number MIN/MAX), `facets-req` (OPERATOR numeric without facets key; setpoint/count-like without UNITS/PRECISION — WARN), `ord-literal` (hardcoded station:|slot:/ string — WARN) [ev: retro campaign8-facets-lint]. **Build (WSL) = `$KIT/toolbelt/build.sh`** (Java 8 + clean + slotomatic for every profile with sources + jar, then runs the gate). **Deploy = `niagara-tools/scripts/ng-deploy.sh`** (backup → build → copy → type-count verify; its slotomatic guard is rt-only) — run the gate on `build/libs` after it. Never `gradle :jar` with the default JDK.
- **`@NiagaraProperty` edits** go in the annotation AND the generated region AND the imports; prove with slotomatic.

## Decision Gates — pick the module type

| Building | Read |
|----------|------|
| Pure logic (rt control) | `$KIT/types/logic.md` |
| Framework extension (service, ORD scheme, point ext, analytics node, job, watchdog, provider) | `$KIT/types/logic-authoring.md` |
| Dashboard (facade + servlet + SPA) | `$KIT/types/dashboard.md` |
| Logic + dashboard | both |
| Workbench widget / PX (wb) | `$KIT/types/wb-widgets.md` |

**wb builders**: `types/wb-widgets.md` was a seed and is now exemplar-backed for the -wb model seam, but the guide is still thin — start from corpus B751–B753 and B762/B780 before the type guide. The Predicate-injection seam (DWB1) is the critical entry point.

`$KIT/METHODOLOGY.md` (common checklist) applies to ALL.

## Execution Steps

1. Orient from `$KIT/BUILD-STATE.md` (BUILD-LOOP §0.a) — state the module's one-line leave-off; then read `METHODOLOGY.md` + `BUILD-LOOP.md` + the type guide; skim `$KIT/corpus-index.md` for the P0 blocks of the layer you are building.
2. corpus-nav the topic; read the exemplar source.
3. Build the layers; preview the UI.
4. Build with `toolbelt/build.sh` (Java 8 + slotomatic); verify major-52 + signed.
5. Pre-gate checks before `verify-module.sh`: run `toolbelt/lint-delays.sh <src>` (Clock.schedule delay-floor lint; exit 1 = any FAIL — a non-positive floor silently kills the timer at runtime) [ev: retro campaign8-lint-delays]; run `toolbelt/lint-timers.sh <src>` (timer-ticket/discarded-ticket; exit 1 = FAIL); run `toolbelt/slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>` (type-set lexicon coverage; empty lexicon → exit 1) and `toolbelt/slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>` (per-OPERATOR-slot diff; MISSING/STALE rows; exit 1 = missing slot) [ev: retro campaign8-slot-per-slot]; run `toolbelt/schema-risk.sh <before-dir> <after-dir>` before deploy (two-snapshot slot diff; verdict SAFE/LOSSY/OUTAGE — OUTAGE means a deployed change breaks saved data) [ev: retro tool-integration]; then run the verify gate (`toolbelt/verify-module.sh`) with `--plano <index.html|jar>` when available [ev: retro campaign7-plano]; finally run `toolbelt/report-module.sh <module-root> [--target-version x.y]` for one aggregated punch-list — exit 1 = FAILs block hand-off [ev: retro campaign7-report-module], `$KIT/toolbelt/triage-console.sh` (own-module exception triage over station console logs; three attribution channels C1/C2/C3; exit 0 clean / 1 rows / 3 usage; run after reload — see step 6) [ev: retro campaign8-triage-console].
6. After a reload (rt or full station restart), triage the console: `toolbelt/triage-console.sh --package com.vendor <station-dir>/console*.txt` surfaces own-module exceptions and load-time failures invisible to C1/C2 alone (three channels: own frame, own logger tag, [sys]/[sys.xml] load-fail shape); exit 1 = rows found — investigate before calling the deploy clean. [ev: retro campaign8-triage-console]
7. Retro + close (HARD gate): append PROVEN lessons as PROPOSED deltas (`$KIT/retros/`, propose-never-apply) AND update `$KIT/BUILD-STATE.md` (envelope + `retro_pending`) — or declare `Retro: none (trivial: <reason>)`. This is how the kit grows and stays continuous.

## Output Contract

Report: kit path, type chosen, layers built, preview + build outcome (bytecode 52 + signed), verify-module.sh result (PASS/FAIL per check), proposed kit deltas, and the retro line — `retro: <path> (N deltas, review-status: pending)` or `retro: none (trivial: <reason>)`.

## References

- `$KIT/METHODOLOGY.md`, `$KIT/BUILD-LOOP.md`, `$KIT/build-verify.md`, `$KIT/types/logic.md`, `$KIT/types/logic-authoring.md`, `$KIT/types/dashboard.md`, `$KIT/types/wb-widgets.md`, `$KIT/SOURCES.md`, `$KIT/scripts/install-skill.sh`.
- Toolbelt: `$KIT/toolbelt/build.sh`, `$KIT/toolbelt/verify-module.sh` (+ coverage, + --plano <ux-profile>/src/rc/index.html), `$KIT/toolbelt/run-pure-test.sh`, `$KIT/toolbelt/mirror-niagara-home.sh`, `$KIT/toolbelt/stored-repack.sh`, `$KIT/toolbelt/preflight.sh`, `$KIT/toolbelt/slot-coverage.sh` (type-set + per-slot modes; per-slot: `per-slot <xml> <lex> <src>`, exit 1 = MISSING) [ev: retro campaign8-slot-per-slot], `$KIT/toolbelt/lint-timers.sh`, `$KIT/toolbelt/lint-delays.sh` (Clock.schedule delay-floor lint; exit 0 clean / 1 any FAIL / 3 usage; run in step 5 before lint-timers) [ev: retro campaign8-lint-delays], `$KIT/toolbelt/sweep-build-state.sh` (+ --age), `$KIT/toolbelt/sweep-fold-audit.sh`, `$KIT/toolbelt/scaffold-module.sh` (skeleton from fixtures/MinimalPan; exits 0/2/3) [ev: retro tool-integration], `$KIT/toolbelt/schema-risk.sh` (two-snapshot slot diff; verdict=SAFE/LOSSY/OUTAGE; exits 0/1/2/3/4) [ev: retro tool-integration], `$KIT/toolbelt/report-module.sh` (aggregated conformance report per module; exit 0 CLEAN / 1 FAILs / 3 env; run last in step 5 before hand-off) [ev: retro campaign7-report-module], `$KIT/toolbelt/rc-scan.sh` (browser-resource lint over rc/ assets; exit 0 clean / 1 any FAIL / 3 usage; run before hand-off on any -ux profile with rc/) [ev: retro campaign8-rc-scan].; run last in step 5 before hand-off) [ev: retro campaign7-report-module].
