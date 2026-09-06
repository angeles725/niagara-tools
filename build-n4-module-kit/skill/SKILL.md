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
| New module scaffold / source layout audit | `$KIT/types/structure.md` |

**wb builders**: `types/wb-widgets.md` was a seed and is now exemplar-backed for the -wb model seam, but the guide is still thin — start from corpus B751–B753 and B762/B780 before the type guide. The Predicate-injection seam (DWB1) is the critical entry point.

`$KIT/METHODOLOGY.md` (common checklist) applies to ALL.

## Execution Steps

1. Orient from `$KIT/BUILD-STATE.md` (BUILD-LOOP §0.a) — state the module's one-line leave-off; then read `METHODOLOGY.md` + `BUILD-LOOP.md` + the type guide; skim `$KIT/corpus-index.md` for the P0 blocks of the layer you are building.
1b. Explore shard: when the change touches > 3 files, open a `sdd-explore` shard (sonnet) to map the affected symbol set before writing; audit-first prevents fold errors on renamed or moved files. `[ev: METHODOLOGY.md K6]`
1c. Design shard: when the change introduces a schema annotation, a new slot, or a new façade type, open a `sdd-design` shard (opus) before writing; these are high blast-radius changes that need architectural scoping (`gentle-sdd-ff` path). `[ev: ORCHESTRATION.md §Which model runs each phase; CLAUDE.md Model Assignments]`
2. corpus-nav the topic; read the exemplar source.
3. Build the layers; preview the UI.
4. Build with `toolbelt/build.sh` (Java 8 + slotomatic); verify major-52 + signed.
5. Pre-gate checks before `verify-module.sh`: run `toolbelt/lint-delays.sh <src>` (Clock.schedule delay-floor lint; exit 1 = any FAIL — a non-positive floor silently kills the timer at runtime) [ev: retro campaign8-lint-delays]; run `toolbelt/lint-timers.sh <src>` (timer-ticket/discarded-ticket; companion-flag requires a class-scope FIELD + same-method Clock.schedule — method-locals excluded, brace_depth >= 2 guard prevents class body misidentification; exit 0 clean / 1 any FAIL / 3 usage) [ev: retro campaign10-lint-timers-scope]; run `toolbelt/lint-wb-threading.sh <wb-src-dir>` on any -wb profile with Java sources (Swing-thread traversal + agent-breadth WARN; exit 0 WARN-only / 1 under --strict) [ev: retro campaign8-wb-audit]; run `toolbelt/lint-demand-scope.sh [--strict] <src>` (demand-in-scope: a control/staging method that reads a process variable with NO demand-shaped input in scope → WARN "pressure without demand"; exit 0 WARN-only / 1 under --strict / 3 usage) [ev: retro campaign9-demand-scope]; run `toolbelt/slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>` (type-set lexicon coverage; empty or missing lexicon with declared types → exit 1) and `toolbelt/schema-risk.sh <before-dir> <after-dir>` before deploy (two-snapshot slot diff; verdict SAFE/LOSSY/OUTAGE — OUTAGE means a deployed change breaks saved data) [ev: retro tool-integration]; then run the verify gate (`toolbelt/verify-module.sh`) with `--plano <index.html|jar>` when available [ev: retro campaign7-plano]; finally run `toolbelt/report-module.sh <module-root> [--target-version x.y]` for one aggregated punch-list — exit 1 = FAILs block hand-off [ev: retro campaign7-report-module], `$KIT/toolbelt/triage-console.sh` (own-module exception triage over station console logs; three attribution channels C1/C2/C3; exit 0 clean / 1 rows / 3 usage; run after reload — see step 6) [ev: retro campaign8-triage-console].`toolbelt/lint-structure.sh <module-root>` (module source-tree structure lint: package naming L1, @NiagaraType count L2, pure-model advisory L3, lexicon/palette non-empty L4/L5, module-include.xml vs META-INF/module.xml L6, 3-part version floors L7, empty-skeleton -wb/-ux L9, absolute-host-path in gradle.properties L10, mixed-srcTest decls L11; exit 0 clean / 1 any FAIL / 3 usage) [ev: retro campaign8-structure]; run`toolbelt/lint-write-path.sh <module-root>` (OPERATOR-slot write-path matrix; row in docs/write-path-matrix.md required per OPERATOR slot; exit 0 all covered / 1 any uncovered) [ev: retro campaign8-write-path].`toolbelt/lint-silent-protection.sh [--strict] <src>` on any -rt profile with Java sources (silent-protection-trip lint; WARN rows for protection trips with no alarm/reason surface; exit 0 WARN-only / 1 under --strict / 3 usage) [ev: retro campaign9-silent-protection]; run `toolbelt/lint-ext-writable-shape.sh [--strict] <src>` (ext-writable-shape: an OPERATOR complex property (BStatusNumeric/BStatusBoolean/BStatusEnum) with no writing action → WARN — write it via the oBIX child leaf …/value (bare <real>, B826) or add an OPERATOR action (B822); exit 0 WARN-only / 1 under --strict / 3 usage) [ev: retro campaign9-ext-writable-shape]; run
6. After a reload (rt or full station restart), triage the console: `toolbelt/triage-console.sh --package com.vendor <station-dir>/console*.txt` surfaces own-module exceptions and load-time failures invisible to C1/C2 alone (three channels: own frame, own logger tag, [sys]/[sys.xml] load-fail shape); exit 1 = rows found — investigate before calling the deploy clean. [ev: retro campaign8-triage-console]
7. Retro + close (HARD gate): run `toolbelt/new-retro.sh <module|kit> <slug>` and fill the stub (What happened / Evidence / Proposed kit deltas / Lessons); a kit-check/doctrine defect also opens `toolbelt/kit-ticket.sh "<one line>"`. Update `$KIT/BUILD-STATE.md` (envelope + `retro_pending`) — or declare `Retro: none (trivial: <reason>)`. `sweep-build-state.sh --age` at orient shows outstanding retro debt. [ev: retro campaign8-retro-loop]

## Output Contract

Report: kit path, type chosen, layers built, preview + build outcome (bytecode 52 + signed), verify-module.sh result (PASS/FAIL per check), proposed kit deltas, and the retro line — `retro: <path> (N deltas, review-status: pending)` or `retro: none (trivial: <reason>)`.

## References

- Orchestration: `$KIT/ORCHESTRATION.md` (model table, delegation triggers, escalation gate, per-run retro/ticket loop). `[ev: retro campaign8-orchestration]`
- `$KIT/METHODOLOGY.md`, `$KIT/BUILD-LOOP.md`, `$KIT/build-verify.md`, `$KIT/types/logic.md`, `$KIT/types/logic-authoring.md`, `$KIT/types/dashboard.md`, `$KIT/types/wb-widgets.md`, `$KIT/SOURCES.md`, `$KIT/scripts/install-skill.sh`.
- Toolbelt: `$KIT/toolbelt/build.sh`, `$KIT/toolbelt/verify-module.sh` (+ coverage, + --plano <ux-profile>/src/rc/index.html), `$KIT/toolbelt/run-pure-test.sh`, `$KIT/toolbelt/mirror-niagara-home.sh`, `$KIT/toolbelt/stored-repack.sh`, `$KIT/toolbelt/preflight.sh`, `$KIT/toolbelt/slot-coverage.sh` (type-set + per-slot modes; per-slot: `per-slot <xml> <lex> <src>`, exit 1 = MISSING) [ev: retro campaign8-slot-per-slot], `$KIT/toolbelt/lint-timers.sh` (timer-ticket/discarded-ticket/companion-flag — class-FIELD + same-method scope; exit 0 / 1 / 3) [ev: retro campaign10-lint-timers-scope], `$KIT/toolbelt/lint-delays.sh` (Clock.schedule delay-floor lint; exit 0 clean / 1 any FAIL / 3 usage; run in step 5 before lint-timers) [ev: retro campaign8-lint-delays], `$KIT/toolbelt/lint-wb-threading.sh` (Swing-thread traversal + agent-breadth WARN over -wb src; exit 0 clean / 1 under --strict / 3 usage) [ev: retro campaign8-wb-audit], `$KIT/toolbelt/lint-demand-scope.sh` (demand-in-scope lint; WARN rows for a control-decision method reading a process variable with no demand-shaped input in scope; exit 0 clean or WARN-only / 1 under --strict / 3 usage or no sources) [ev: retro campaign9-demand-scope], `$KIT/toolbelt/lint-silent-protection.sh` (silent-protection-trip lint; WARN rows for unalarmed trips; exit 0 clean or WARN-only / 1 under --strict / 3 usage; run on any -rt profile with Java sources) [ev: retro campaign9-silent-protection], `$KIT/toolbelt/lint-ext-writable-shape.sh` (ext-writable-shape lint; WARN rows for OPERATOR complex (BStatusNumeric/BStatusBoolean/BStatusEnum) properties with no writing action; exit 0 clean or WARN-only / 1 under --strict / 3 usage) [ev: retro campaign9-ext-writable-shape], `$KIT/toolbelt/sweep-build-state.sh` (+ --age), `$KIT/toolbelt/sweep-fold-audit.sh`, `$KIT/toolbelt/scaffold-module.sh` (skeleton from fixtures/MinimalPan; run in step 5 before lint-timers) [ev: retro campaign8-lint-delays], `$KIT/toolbelt/schema-risk.sh` (two-snapshot slot diff; verdict=SAFE/LOSSY/OUTAGE; exits 0/1/2/3/4) [ev: retro tool-integration], `$KIT/toolbelt/report-module.sh` (aggregated conformance report per module; `--console-dir <dir>` for triage-console integration; exit 0 CLEAN / 1 FAILs / 3 env; run last in step 5 before hand-off) [ev: retro campaign7-report-module] [ev: retro campaign8-report-integration], `$KIT/toolbelt/rc-scan.sh` (browser-resource lint over rc/ assets; exit 0 clean / 1 any FAIL / 3 usage; run before hand-off on any -ux profile with rc/) [ev: retro campaign8-rc-scan], `$KIT/toolbelt/bog-audit.sh` (station config.bog auditor; CHECK1-CHECK12; exit 0 clean / 1 any FAIL / 3 usage/python3-absent; requires python3; add --source-dir for source-coupled checks CHECK2-7; proxy-link-safety CHECK11 is FAIL) [ev: retro campaign8-bog-audit], `$KIT/toolbelt/station-snapshot.sh` (audit-surface snapshot: copies config.bog + console*.txt, writes manifest.json with sha256; exit 0 ok / 1 copy fail / 3 usage; run before/after deploy; source never written) [ev: retro campaign8-station-snapshot], `$KIT/toolbelt/lint-servlet.sh` (BWebServlet security lint: auth gate, input-400, unbounded-set, cache-nofinger, log-in-handler, csrf-xrw-only; exit 0 clean or WARN-only / 1 any FAIL / 3 usage; run on any -ux profile with BWebServlet subclasses) [ev: retro campaign8-lint-servlet]; source never written) [ev: retro campaign8-station-snapshot].`$KIT/toolbelt/lint-structure.sh` (module source-tree structure lint L1–L11; exit 0 clean / 1 FAIL / 3 usage; run at start of step 5 pre-gate) [ev: retro campaign8-structure],`$KIT/toolbelt/lint-write-path.sh` (OPERATOR-slot write-path matrix coverage; every @NiagaraProperty with Flags.OPERATOR must have a row in docs/write-path-matrix.md; --bog adds link-traced dashboard slots; exit 0 all covered / 1 any uncovered / 3 usage) [ev: retro campaign8-write-path]; source never written) [ev: retro campaign8-station-snapshot]
