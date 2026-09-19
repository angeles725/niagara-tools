# BUILD-LOOP — the operational cycle for one N4 module

The contract the launcher runs. Follow it in order; the gates are not optional.

## 0. Orient (before touching anything)
- `corpus-nav find "<topic>"` — read the matching blocks. 90% is already documented; don't re-derive.
- Read the EXEMPLAR source for the type (SOURCES.md), verbatim — not from memory.
- Pick the module type (SKILL.md decision table) → load `types/<type>.md`.

### 0.a Orient from BUILD-STATE (before touching anything)
- Run **`toolbelt/orient-guard.sh <module-root>`** — asserts that the module root is under the sole legal build prefix `/home/cristian/modulos_niagara_n4/Cliente/`; exit 1 = wrong location, abort; exit 0 (PASS or WARN) = proceed. Set `BUILD_N4_CLIENTE_OVERRIDE=1` to warn-and-pass on an exceptional path. `[ev: retro module-worktree-location-retro]`
- Read `BUILD-STATE.md` for the module you are about to build. From its `build-state.v1` envelope + prose, tell the operator in ONE line:
  `<module> · built <last_build>/gate <verify_gate>/deployed <deployed> · next: <last_session tail> · open_issues=N · retro_pending=Y/N`.
- If `retro_pending: true`, the previous session left an OWED retro — writing it is the FIRST task unless the operator redirects.
- If the module has no section, this is a first build — say so, and add its section at close.
- `toolbelt/sweep-build-state.sh --age --today <YYYY-MM-DD> <retros-dir> INDEX.md` — retro-debt aging; call at orient and again at close.
- **Meta-work exemption:** auditing the kit / tooling / a retro, or a single one-off question, does NOT gate on orient — say so and proceed (mirrors the research-sdd PASO 0 exemption).

### 0.b Preflight (before the first build)
- Run **`toolbelt/preflight.sh <niagara_home> <gradle-root>`** — automates win-path, jdk8, plugin-pin, jar-lock checks; exit 0 = all clear.
- **JDK 8** present (`ls /usr/lib/jvm`).
- **niagara_home chosen** = the LOWEST target version you must support, AND its pinned gradle plugin version (from `settings.gradle.kts`) is present in `<niagara_home>/etc/m2` — each install ships only one (build-verify.md).
- **Station live?** A running station LOCKS its `modules/<mod>.jar`. Free the lock FIRST (close Workbench, or stop a non-production station) and build directly; use a mirror (`toolbelt/mirror-niagara-home.sh`) ONLY for a live production supervisor you must not stop — see build-verify.md §Building against a running station. [ev: retro coldroompan-dashboardpan-freeze-stat-leds · B2]
- **This module's build target may differ from the last one's — read THIS module's `gradle.properties` + `settings.gradle.kts`:** a sibling in the same client repo can target a different `niagara_home` + plugin (e.g. ColdRoomPan → PowerB 4.15.3 / 7.6.22 vs CompPan → Honeywell 4.14 / 7.6.17); never carry over the previous module's target. [ev: retro module-palette-and-build-target · B6]

## 1. Design
- State the module's job, its profiles (rt/ux/wb), and the slot/endpoint contract in one paragraph.
- For a dashboard: the facade slots (display link-in + writable config), the servlet routes, the JSON `{v,st}` contract, the HMI resolution.
- **New module skeleton:** run `toolbelt/scaffold-module.sh <ModuleName> <out-dir>` to emit a pre-slotomatic tree from `fixtures/MinimalPan`; exits 0 ok / 2 usage / 3 env (skeleton missing). [ev: retro tool-integration]

### 1.a New-module bring-up (before the first build)

After `scaffold-module.sh` generates the skeleton, three steps MUST happen before running
`toolbelt/build.sh` for the first time:

1. **`chmod +x gradlew`** — the scaffold copies the Gradle wrapper but does not set the execute
   bit; `./gradlew` is non-functional without it.
2. **Align `gradle.properties` + `settings.gradle.kts` to a SIBLING module of the same SDK family:**
   copy `niagara_home`, `niagara_user_home`, `nodeHome`, and
   `org.gradle.java.installations.paths` from an existing sibling that deploys to the same
   station. Do NOT leave the scaffold's commented-out (auto-detect) JDK/node block: a commented
   block builds in WSL (where `build.sh` overrides the path via `-P`) but fails reproducibility
   on a Windows `gradlew` build. Also align `gradlePluginVersion` / `settingsPluginVersion` in
   `settings.gradle.kts` to the sibling's SDK family (each Niagara install ships exactly one
   plugin version — see `build-verify.md §Build target & plugin version`). For a logic-only
   module with no `-ux` profile the node/JDK lines may stay commented.
3. **Run `toolbelt/preflight.sh` + first build:** `preflight.sh <niagara_home> <gradle-root>`
   validates the environment (§0.b); then `toolbelt/build.sh <group-dir> <MOD>` runs the first
   full clean+slotomatic+jar+verify cycle.

`[ev: corpus B1016]`

## 2. Build the layers
- Follow `types/<type>.md` + `METHODOLOGY.md`. Keep a facade pure; keep control logic in rt; keep UI in ux/wb. For framework-extension authoring (custom service, ORD scheme, point extension, analytics node, job, watchdog): see `types/logic-authoring.md` (companion to `types/logic.md`).
- **What to READ for this layer, in priority order: `corpus-index.md`** — the curated map of the niagara-research authoring corpus (B729–B760). `corpus-nav FIRST` for a term; `corpus-index.md` for what to read by layer/priority (P0 before building).
- Apply the slot rules as you write each `@NiagaraProperty` (flags, facets/units, the annotation+generated+imports rule).

## 3. Preview (UI types) — BEFORE compiling
- `python3 /home/cristian/niagara-research/tools/dashboard-preview.py --rc <mod>/src/rc --prefix /<mount>`
- Open `http://localhost:<port>/hmi` (1280×800 frame). Iterate design here — refresh, no build.
- **Operator preview + explicit OK is a REQUIRED gate before building any `-ux` change (not optional):** seed the mock with the state that triggers the new behavior (e.g. `Cuarto3/evapNValveState` to exercise the output LEDs), so the operator actually sees it. The mock does NOT run `-rt` logic, so the preview approves the DASHBOARD's behavior, not the physical rt effect — say that to the operator. An `-rt`/`-wb`-only change has no preview (approved by design + pure tests). [ev: retro self-retro-preview-gate · T5]

## 4. Build — the ONLY valid build
- **`build.sh` argument order:** `toolbelt/build.sh <module-root=GROUP dir> <MOD> [niagara_home]`. The first argument is the GROUP directory (the gradle root, e.g. `Cliente/Leon-Guanjuato/Paccadia`), NOT the module sub-directory. `niagara_home` (arg 3 or the env var `NIAGARA_HOME`) is REQUIRED — the script exits 10 (env) when it cannot resolve it. A client multi-project layout places `./gradlew` at the GROUP ancestor, so always pass the GROUP dir, not a profile dir. `[ev: retro tree-selection-and-schema-risk-baseline Δ3]`
- Three roles (build-verify.md §Doctrine): `toolbelt/build.sh` is the recommended WSL build (clean + slotomatic for every profile with sources + jar, then it calls the gate); `scripts/ng-deploy.sh --strict-slotomatic` is the station deploy wrapper (backup→build→copy→verify; strict aborts if annotations changed without slotomatic, and its slotomatic guard is rt-only); `toolbelt/verify-module.sh` is THE gate, run on the built jars.
- Confirm: bytecode major **52**, jars **signed**, no raw-double facet. (build-verify.md)
- A `gradle :jar` with the default JDK is NOT a build.

### 4.a Gradle task matrix (`niagara-module` plugin tasks) — when to run each

| Task | Touches | When required |
|------|---------|---------------|
| `clean` | deletes `build/` outputs | always before a deploy build; exits 31 if station holds `modules/<jar>` lock — see station-lock recipe below `[ev: corpus B807]` |
| `slotomatic` | generates slot registry, dispatch, and `_BProxy` sources from `@NiagaraProperty`/`@NiagaraAction`/`@NiagaraType` | any annotation change; `build.sh` always runs it before `jar`; skipping compiles against a stale proxy `[ev: corpus B807]` |
| `compileJava` | compiles `.java` sources (incl. slotomatic output) | implicit; depends on `slotomatic` `[ev: corpus B807]` |
| `jar` | packages compiled classes into the profile jar | always (kit default; `build.sh` uses this, not `build`) `[ev: corpus B807]` |
| `build` | `jar` + `test` (JUnit) | only when JUnit tests must run alongside compile; heavier than `jar` alone `[ev: corpus B807]` |
| `moduleTestJar` | packages test classes for the `niagaraTest` framework | combined with `niagaraTest` only `[ev: corpus B807]` |
| `niagaraTest` | runs tests inside a live Niagara container | only when Niagara-framework tests exist; requires a running station; rare in WSL `[ev: corpus B807]` |
| `bajadoc` | Baja API docs | release/on-demand only; not part of the normal build loop `[ev: corpus B807]` |

Safe combinations: `clean slotomatic jar` — the only correct kit build (`build.sh` default, per-profile). `clean slotomatic build` if JUnit tests must run alongside. Avoid `jar` without `clean` (stale class outputs). `[ev: corpus B807]`

**Station-lock recipe (exit 31):** `:clean` fails with "Unable to delete `<niagara_home>/modules/<jar>`" when a running station holds the lock (`toolbelt/build.sh:15,:82-88`). Fix: `toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>` creates a writable mirror so the plugin can copy freely without touching the live install (see §0.b). `[ev: corpus B807]`

### 4.b Test layer — BTestNg station-lifecycle tests `[ev: corpus B815]`

A `moduleTest` / `BTestNg` station-lifecycle test exercises the compiled rt code inside a real Niagara container. This is the DOCUMENTATION layer — it proves that the compiled classes load and behave correctly in a station context, but it does NOT run from WSL (see `build-verify.md §Unit tests in WSL`).

**When to write one:** any rt component whose correctness depends on the Niagara lifecycle (started/stopped/atSteadyState, Clock.Ticket arming, slot-change propagation) and whose pure-JUnit test cannot reach those hooks.

**Recipe (grounded in `BEvaporatorUnit` / ColdRoomPan c66e412):**
1. Create `src<Test>/com/<vendor>/<Module>/BMyComponentTest.java` extending `BTestNg`.
2. Open a station: `try (BTestStation station = createTestStation()) { ... }` (try-with-resources — the station tears down automatically).
3. Add the component under test to the station tree and call `start()` or wait for `atSteadyState()`.
4. Drive slots via `component.set(slotName, value)` and read results via `component.get(slotName)`.
5. To assert a `Clock.Ticket` was armed: read the field via reflection (`Field f = ... f.setAccessible(true); Clock.Ticket t = (Clock.Ticket)f.get(component)`) — NEVER wait on wall-clock timers (a `Thread.sleep` is a flaky gate).
6. Assert the invariants the pure test cannot reach (e.g. defrost PRESERVES the powerOnTicket; `stopped()` cancels all tickets AND clears companion flags).

**Build gate:** in WSL, add a `moduleTestJar` compile-gate step (`build.sh` variant or `./gradlew :MOD-rt:moduleTestJar`). This compiles the test classes inside the Niagara container's compile classpath — a compile failure surfaces annotation bugs early. `niagaraTest` runs native/JACE only.

**Scaffold gradle:** the `-rt` profile gradle must declare `moduleTestImplementation("Tridium:test-wb")`
so `BTestNg` resolves.  The correct dep key is `moduleTestImplementation` (NOT `testImplementation`) —
using the plain `testImplementation` key makes the dep visible to the standard JUnit runner, not to
`niagaraTest`, so the Niagara framework cannot resolve `BTestNg` at test-container startup.  Lint L11
(`lint-structure.sh`) flags a module that mixes pure-JUnit + Baja test deps without both declarations.

**`moduleTest-include.xml` — separate descriptor for test types:**  test classes annotated with
`@NiagaraType` must be registered in a `<part>/moduleTest-include.xml`, NOT in the production
`module-include.xml`.  The plugin builds a separate `moduleTestJar` from this file; production types and
test types never mix.  A WSL gate step to compile the test jar: `./gradlew :<MOD>-rt:moduleTestJar`
(fails on annotation bugs without needing a running station).  See `types/moduleTest.md` for the full
file schema and gradle dep block. `[ev: corpus B958 §958.3–958.4]`

**WSL limitation (honest):** `niagaraTest` discovers 0 tests from WSL (plugin 7.6.17 bug — needs native
`bin/test` + dev license; cannot run inside a WSL JUnit context).  WSL can COMPILE the `moduleTestJar`
as an early gate; only a native/JACE Workbench run (Tools → Run Tests) produces an actual test result.
Do not count the compile-only pass as test execution.  `[ev: corpus B961 §961.3]`

### 4.c Version-bump checklist (before any slot-touching commit)
- `vendorVersion` (in `module.xml` / `gradle.properties`) MUST be bumped on every schema change — slot add, remove, retype, or rename. On reload the station re-decodes `config.bog` against the new module's type/slot registry; a retype or remove is a schema-risk OUTAGE. `[ev: corpus B807]` `[ev: corpus B795]`
- `bajaVersion` is the Niagara platform API target set in `gradle.properties` / `settings.gradle.kts`; do NOT bump it between normal builds — it follows the `niagara_home` chosen at build time and is managed by the plugin. `[ev: corpus B807]`
- Restart mandatory for any `-rt` or `-wb` jar change (Java classes loaded at boot); a `-ux`-only change needs no restart — browser hard-reload only (§6). `[ev: corpus B807]`
- Run `toolbelt/schema-risk.sh` before any bump that touches slots; see §6 for the MANDATORY deploy gate. `[ev: corpus B795]`

## 5. Verify gate (before "done")
- Run `METHODOLOGY.md` (common) + the `types/<type>.md` checklist against the built module. Every item pass, or fix it.
- **Pre-gate (run before `verify-module.sh`):** `toolbelt/lint-structure.sh <module-root>` (module source-tree structure lint: package naming L1, @NiagaraType-per-file L2, pure-model advisory L3 WARN, lexicon non-empty L4, palette non-empty for rt L5, no hand-authored META-INF/module.xml L6, 3-part version floors L7, no empty-skeleton -wb/-ux L9, no absolute host paths in gradle.properties L10, mixed-srcTest decls L11; exit 0 clean / 1 any FAIL / 3 usage; L8 signed-jar stays in verify-module.sh) [ev: retro campaign8-structure]; `toolbelt/lint-delays.sh <src>` (Clock.schedule delay-floor lint; exit 1 = any FAIL — a non-positive floor silently kills the timer at runtime) [ev: retro campaign8-lint-delays]; `toolbelt/lib/method-boundary.sh` (shared method-boundary awk library — sourced, not invoked; C11 T1 fragment; fragment rule: edit the shared fragment, never overwrite consumer lint parser blocks separately) [ev: retro campaign11-shared-method-boundary]; `toolbelt/lint-timers.sh <src>` (timer-ticket/discarded-ticket; companion-flag requires class-scope FIELD + same-method Clock.schedule; PEAK-depth parser via lib/method-boundary.sh; fragment rule: never overwrite; exit 0 clean / 1 any FAIL / 2 usage / 3 env — K20 per-lint contract, lint-timers.sh:44/:58/:63) [ev: retro campaign10-lint-timers-scope] [ev: retro campaign11-shared-method-boundary]; `toolbelt/lint-wb-threading.sh <wb-src-dir>` (Swing-thread traversal and agent-breadth heuristic over a -wb src tree; WARN rows for human review; exit 0 WARN-only / 1 under --strict; run on any -wb profile with Java sources) [ev: retro campaign8-wb-audit]; `toolbelt/lint-demand-scope.sh [--strict] <src>` (demand-in-scope: a control/staging method that reads a process variable with NO demand-shaped input in scope → WARN "pressure without demand"; exit 0 WARN-only / 1 under --strict / 3 usage) [ev: retro campaign9-demand-scope]; `toolbelt/slot-coverage.sh [--strict] <module-include.xml> <module.lexicon>` (type-set lexicon coverage; empty or missing lexicon with declared types exits 1); `toolbelt/schema-risk.sh <before-dir> <after-dir>` (two-snapshot slot diff before deploy; verdict SAFE/LOSSY/OUTAGE, exits 0/1/2/3/4 — exit 2 means the slot change would break saved data) [ev: retro tool-integration] [ev: retro campaign7-plano]; `toolbelt/verify-module.sh --plano <ux-profile>/src/rc/index.html` (when a -ux profile is present); `toolbelt/rc-scan.sh <ux-artifact-dir>` (browser-resource lint over rc/ assets: hardcoded ORD/host literals, bare .catch(()=>{}), null display branches; exit 1 = FAIL) [ev: retro campaign8-rc-scan].; `toolbelt/lint-servlet.sh <ux-profile>/src` (BWebServlet security lint: auth gate, input-400, unbounded-set, cache-nofinger, log-in-handler, csrf-xrw-only; exit 0 clean or WARN-only / 1 any FAIL / 3 usage; run on any -ux profile that has BWebServlet subclasses) [ev: retro campaign8-lint-servlet]`toolbelt/lint-write-path.sh <module-root> [--bog <config.bog>] [--strict]` (OPERATOR-slot write-path matrix coverage; every @NiagaraProperty with Flags.OPERATOR must have a row in docs/write-path-matrix.md; --bog adds link-traced dashboard slots; STALE advisory: a matrix row naming a slot absent from ALL source @NiagaraProperty/Action names prints STALE (per-row, [concept] exempts the marked row only, exit 0); DRIFT advisory (C11 T3): a [concept]-marked matrix row whose backtick slot name IS in the covered set prints DRIFT — the concept exemption is stale; true concept rows (slot absent from source) stay silent; --strict promotes any STALE or DRIFT to exit 1; uncovered FAIL always exits 1 (unchanged with and without --strict); exit 3 usage) [ev: retro campaign8-write-path] [ev: retro campaign10-write-path-stale] [ev: retro campaign11-concept-row-drift]`toolbelt/lint-silent-protection.sh [--strict] <src>` (silent-protection-trip lint; WARN rows for protection trips with no alarm/reason surface; exit 0 WARN-only / 1 under --strict / 3 usage; run on any -rt profile with Java sources; PEAK-depth parser via lib/method-boundary.sh second -f; fragment rule: never overwrite) [ev: retro campaign9-silent-protection] [ev: retro campaign11-shared-method-boundary]; `toolbelt/lint-ext-writable-shape.sh [--strict] <src>` (ext-writable-shape: an OPERATOR complex property (BStatusNumeric/BStatusBoolean/BStatusEnum) with no writing action → WARN — write it via the oBIX child leaf …/value (bare <real>, B826) or add an OPERATOR action (B822); exit 0 WARN-only / 1 under --strict / 3 usage; PEAK-depth parser via lib/method-boundary.sh adjacent-quote concatenation; fragment rule: never overwrite) [ev: retro campaign9-ext-writable-shape] [ev: retro campaign11-shared-method-boundary]; S22 (C10): exemption is now PER-SLOT — a do<Action>() body must write THAT slot (action x → doX(), brace_depth>=2 scope filter); class-level has_action removed; EW10 CompPan-rt 0→1 faultReset (contract change — false negative, not regression) [ev: retro campaign10-ext-writable-per-slot]; `toolbelt/lint-guard-pins.sh [--strict] <kit-root>` (guard-pin meta-check: scans toolbelt/lint-*.sh headers for # Mutation: <id> -- <desc> declarations and resolves each id against tests/*.bats @test names; MATCH rows confirm declared pins; WARN rows flag missing declarations or unresolvable ids — a lint with no # Mutation: line is a latent unpinned guard; scope = lint-*.sh only (D4b); exit 0 WARN-only / 1 under --strict / 3 usage; run as the last pre-gate step after retrofitting headers) [ev: retro campaign11-lint-guard-pins]; `toolbelt/lint-recovery-path.sh <src>` (protection-output recovery-path gate Wave 3 LR1: detects *Out/*Resistance/*Heat/*Resistencia slots written OFF only inside a mode-gated method with no unconditional reset in started/atSteadyState/enable; grep-level heuristic; exit 0 clean / 1 any FAIL / 3 usage) [ev: retro live-commissioning-verification-gaps]; `toolbelt/lint-status-parity.sh [--strict] <src>` (per-instance config/status slot ratio Wave 3 LR2: WARN when N>1 *Interval/*Duration/*Setpoint config slots but fewer *Status/*Active/*Start/*Since status slots; exit 0 WARN-only / 1 under --strict / 3 usage) [ev: retro live-commissioning-verification-gaps]; `toolbelt/lint-config-sanity.sh <src>` (unsafe @NiagaraProperty default patterns Wave 3 LR3: CS1 FAIL interval<=duration; CS2 FAIL zero-setpoint OPERATOR no min>0 facet; CS3 WARN comment-only flag enforcement; exit 0 clean or WARN-only / 1 any FAIL / 3 usage) [ev: retro live-commissioning-verification-gaps]; `toolbelt/lint-lexicon-ascii.sh <module-root>` (ASCII-purity lint for *.lexicon files; Niagara reads lexicons as Latin-1 so UTF-8 accents arrive on-station as mojibake; exit 0 clean / 1 any FAIL / 3 usage; run as part of pre-gate; wired into report-module.sh step 4b) [ev: retro 2026-09-17-umbrelladashboard-module-creation]
- Every lint that ships in a campaign carries a REAL-TREE smoke on the four client module roots: exact COUNT + SUBJECT of each finding + one ABSENCE pin (a slot that must NOT be flagged) — a bats fixture alone is not acceptance (METHODOLOGY K22). `[ev: retro campaign8-close-process-meta-lessons §lesson 11]`
- The automated half of the gate is `toolbelt/verify-module.sh <jars…>` (bytecode 52, signature, type resolution; `--target-version` / `--stored` / `--src` opt-in). A jar that has not passed it does not go to a station. `--src` sub-checks: `typecount`, `facets` (raw-number MIN/MAX), `facets-req` (OPERATOR numeric without facets key; setpoint/count-like without UNITS/PRECISION — WARN), `ord-literal` (hardcoded station:|slot:/ string — WARN) [ev: retro campaign8-facets-lint].
- **Aggregated punch-list before hand-off:** `toolbelt/report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>]` composes all the above per profile artifact (including lint-delays, schema-risk per D9a, and optionally triage-console) and prints one aggregated report; exit 1 = punch-list has FAILs that block hand-off. [ev: retro campaign7-report-module] [ev: retro campaign8-report-integration]

## 6. Deploy (station) — operator
- **MANDATORY before a live-station deploy — the §5 schema-risk verdict must be SAFE:** a **LOSSY/OUTAGE** verdict means the new jar's slots no longer match the station's saved `.bog` and it will fail to load. Proven LIVE: an OUTAGE-class retype (`BStatusNumeric`↔`BDouble`, `BRelTime`↔`BComplex`) crashed PANCCADIA after a ColdRoomPan reload — `SEVERE [sys] Cannot load station`, a full outage, not a warning. NEVER deploy a LOSSY/OUTAGE change to a station holding saved data without a bog migration. `[ev: corpus B800 §800.8]` `[ev: corpus B795]`
- Sign (auto), stop station, replace jars in `<niagara_home>/modules/`, start. Place components at their fixed ORDs; link points. Open the URL.
- **A `-ux`-only change needs NO station restart — browser hard-reload only; `-rt`/`-wb` needs a restart:** the servlet serves `rc/` from the classloader per request, so a new `-ux` jar is picked up on reload, but Java classes need a restart. Batch rt changes; iterating UI in production is cheap. [ev: retro ux-only-deploy-no-station-restart · D1]
- **The production station may run on a DIFFERENT device (e.g. an ATLAS snap) at a different NRE than the build PC — verify what runs via LIVE slots (oBIX / Slot Sheet), not the PC's `modules/` jar:** identify the station's real `niagara_home` from its boot log first; a PC `modules/` jar is only what Workbench can push. Device-side deploy = push to the device (Software Manager / Provisioning) + restart the device's station. [ev: retro station-corre-en-atlas-snap · D2]
- **After a panel redeploy, power-cycle the HMI (the WebView caches the old page) before suspecting code:** a "blank after redeploy" is usually stale WebView state — disconnect/reconnect the panel first. [ev: retro dashboardpan-detail-render-doors · D3]
- **Before running `ng-deploy.sh`, `cd` to the gradle root** (where `gradlew` / `settings.gradle` live — e.g. `Paccadia/` or `Dashboard/`): it runs gradlew from the CWD, not from `GRADLEW_PATH`'s dir. [ev: retro ng-deploy-type-count-and-cwd · B9]
- **`ng-deploy.sh` backs up LIGHTWEIGHT by default (only this module's own `-rt/-ux/-wb` jars) and auto-purges to keep-N (default 3, `--keep N`):** the old default tarred the WHOLE modules dir (~240 MB/deploy) and never purged (8 deploys → ~1.9 GB). `--no-backup` skips it (plain opt-in; prints a git-rollback WARN, no gate), `--full-backup` / `FULL_BACKUP=1` restores the whole-dir backup. [ev: retro ng-deploy-backup-liviano-y-autopurga · B10] (as of niagara-tools 0.12.0)
- **`ng-deploy.sh`'s `verify_jar` counts `<type ` (WITH a space), so the `<types>` wrapper is NOT counted → `EXPECTED_*_TYPES` = the real type count, same as `verify-module.sh`, no `+1`:** real counts are ColdRoomPan-rt 6, DashboardPan-rt 2, DashboardPan-ux 1. [ev: retro ng-deploy-type-count-and-cwd · B8] (fixed in niagara-tools 0.12.0; before, it counted `<type` and you had to set real-types + 1)

- **Before and after a live-station deploy — snapshot the audit surface first:** `toolbelt/station-snapshot.sh <station-dir> <out-dir>` copies `config.bog` + `console*.txt` into `<out-dir>`, records history/alarm db pointers (paths + sizes, never the db files), and writes `manifest.json` with sha256 per file; source dir is never opened for write; exit 0 ok / 1 copy failure / 3 usage. Keep the pre-deploy snapshot as a baseline for `schema-risk.sh` and `bog-audit.sh` after the deploy. [ev: retro campaign8-station-snapshot]
- **After a reload (rt or full station restart), triage the console before closing the session:** `toolbelt/triage-console.sh --package com.vendor <station-dir>/console*.txt` surfaces own-module exceptions and load-time failures that the framework swallows silently (three attribution channels: own frame, own logger tag, [sys]/[sys.xml] load-fail shape). exit 1 = rows found; investigate before calling the deploy clean. [ev: retro campaign8-triage-console]
- **Audit the station bog for ghost slots, dangling links, orphan handles, proxy-link safety, and station-logic wiring:** `toolbelt/bog-audit.sh <config.bog|file.xml> --module <MOD> [--source-dir <src-dir>] [--strict]` (CHECK1-CHECK19; exit 1 = any FAIL). Runs from the bog alone for CHECK1/8/9/10/11/12/13/14/15/16/17/18/19; add `--source-dir` for the source-coupled checks (CHECK2-7). Proxy-link-safety (CHECK11) fires when an own-module output is linked to a BooleanWritable/NumericWritable with no explicit fallback — the writable holds last state on station stop/reload, masking the fault. Station-logic checks (CHECK13-19): relay-double-source, own-output-unlinked, sensor-crossed-by-name, hasDefrost<->DefrostController sibling, roomN-index-mismatch, tile-number consistency, link-direction. [ev: retro campaign8-bog-audit] [ev: retro campaign8-station-logic]

### 6.a Post-deploy verification (after hot module reload or station restart)

Ordered steps — run within ≤5 min of a hot module reload (Out-of-date: Module changed). `[ev: corpus B811]`

1. `toolbelt/station-snapshot.sh <station-dir> <out-dir>` — snapshot the station before the deploy; keep `<out-dir>` as the deploy baseline (`schema-risk.sh <out-dir> <post-deploy-snapshot>` compares before vs. after). `[ev: corpus B811]`
2. `toolbelt/triage-console.sh --package <com.vendor> <station-dir>/console*.txt` — scan for own-module load failures ("Cannot load station", "Missing frozen property", "ClassCastException", "Missing class for \"<own-prefix>:\""); exit 1 = rows found, investigate before calling the deploy clean. `[ev: corpus B800]`
3. `toolbelt/bog-audit.sh <config.bog|file.xml> --module <MOD>` — ghost slots, dangling links, orphan handles, proxy-link safety (CHECK11); exit 1 = any FAIL. `[ev: corpus B795]`
4. `toolbelt/report-module.sh <module-root> --console-dir <console-dir>` — aggregated punch-list; exit 1 = FAILs block hand-off. `[ev: retro campaign7-report-module]`
- The proxy-link safety row (CHECK11) must be clean before operator hand-off. `[ev: corpus B810]`

## 6.b Commissioning-verify requirement (modules needing station rewiring) `[ev: retro live-commissioning-verification-gaps Δ7]`

The kit verify gate (§5) is **code-level and blind to live commissioning.** It confirms: bytecode 52, signed jars, types resolve, lint rules pass, pure JUnit green. It does NOT verify: facade↔rt link wiring, config values, per-instance control/status parity, or whether the station actually controls the plant correctly.

**A module that requires station rewiring is NOT "done" until a commissioning-verify pass confirms the following:**
1. Every facade display slot is linked from its corresponding control point (`bog-audit.sh CHECK11`, `obix-nav.py` batch link audit).
2. Every facade config/setpoint slot is linked TO the corresponding control slot (not to a display slot or a link-target side).
3. Config values make physical sense: `interval > duration`, setpoint ≠ 0 on a cooling unit, `hasDefrost=true` AND `airDefrost=true` on an air-defrost unit, `duration` above a floor (e.g. ≥ 60 s).
4. Per-instance control/config surfaces have matching per-instance status slots (N configs → N status slots, not 1 scalar).
5. After a hot reload: `triage-console.sh` clean + `bog-audit.sh` CHECK11 clean.

**The kit — not the operator — owns the watchdog role.** If the operator must direct each verification check, the kit has a gap. Add commissioning-verify tooling (step-by-step `obix-nav.py`-style auditor) for any module needing wiring rather than shipping the wiring verification as implicit operator knowledge.

## 7. Retro + close (HARD close gate — not optional)
- **Lead merge/settle order:** merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger; a ledger settle on a reported-but-unverified merge records the wrong evidence revision. For parallel workers: rebase onto the new main tip before the QA ping so the blessed tip is the one that merges, not the pre-rebase base. `[ev: retro campaign8-close-process-meta-lessons]`
- **Every run ends by writing its retro** — run `toolbelt/new-retro.sh <module|kit> <slug>` and fill the stub (§1); a defect in a KIT CHECK or DOCTRINE additionally opens `toolbelt/kit-ticket.sh "<one line>"`. The retro is a PRECONDITION for "done", not an at-STOP afterthought — `toolbelt/sweep-build-state.sh --age` at orient (BUILD-LOOP §0.a) surfaces the accrued retro debt so it cannot be skipped across a continuous chain. [ev: retro campaign8-retro-loop]
- **Update `BUILD-STATE.md`** for the module: refresh the `build-state.v1` envelope (`last_build`, `verify_gate`, `deployed`, `bytecode_major`, `signed`, `last_commit`, `last_session`, `open_issues`), set `retro_required` honestly, and set `retro_pending`.
- **Kit-infrastructure work** (changing the kit itself — toolbelt, type guides, methodology — not building a module) has no module build to record: update the `kit` self-section of `BUILD-STATE.md` instead, under the same close gate.
- A session that changed KIT files is NOT "done" until ONE of:
  - (a) it wrote a retro at `retros/<date>-<module>.md` (line 1 `<!-- review-status: pending -->`, lessons as PROPOSED kit deltas — propose-never-apply), recorded it in the retro index, and set `retro_pending: false` in `BUILD-STATE.md`; OR
  - (b) it declared the change TRIVIAL: `Retro: none (trivial: <reason>)` in the commit trailer AND `retro_required: false` in the envelope; OR
  - (c) it is a PROMOTION of already-filed lessons into the core: `Retro: promotion (folds <ids> from existing retros)` in the commit trailer AND a STRUCTURAL ANCHOR — either an in-range `retros/INDEX.md` change (a FULL promotion flips a folded/pending mark) OR an in-range `BUILD-STATE.md` change (a PARTIAL promotion folds content while its source retro stays `pending` for owed halves, so it flips no row — it stamps the owed `open_issue` in the ledger instead). The trailer ALONE is not a blanket escape; with NEITHER anchor it fails. Either anchored path still runs `sweep` for ledger coherence. A promotion PR folds existing retros, so it owes no NEW retro.
- **Envelope-pairing rule (all exits):** every close-exit — (a) new retro, (b) trivial trailer, (c) promotion — MUST pair its retro/INDEX anchor with the kit `BUILD-STATE.md` self-envelope in the SAME push range; a branch push is not proof — the hook evaluates the whole PR on `main`. [ev: retro doctrine-fold] [ev: retro types-fold] [ev: retro close-process-meta-lessons] [ev: retro campaign7-retro-fold]
- The **Output Contract MUST print** `retro: <path> (N deltas, review-status: pending)` — or `retro: none (trivial: <reason>)` — as an explicit line. A written-but-invisible retro reads as a missing one.
- Do NOT silently rewrite METHODOLOGY — propose; a human folds it in. This is how the kit matures from seed to solid.
- **Sweep at close:** `toolbelt/sweep-build-state.sh <BUILD-STATE.md> <retros-dir> INDEX.md` (envelope content check); `toolbelt/sweep-fold-audit.sh --strict INDEX.md <kit-root>` (fold-citation audit).
- Machine enforcement (opt-in, per-clone, reversible): activate with **`scripts/install-hooks.sh`** (sets `git config core.hooksPath .githooks`; `--uninstall` restores the default; it REFUSES to clobber a pre-existing custom `hooksPath` unless `--force`). Once active, `.githooks/pre-push` blocks a build-relevant push that skips the close gate, delegating the ledger check to `toolbelt/sweep-build-state.sh`.
