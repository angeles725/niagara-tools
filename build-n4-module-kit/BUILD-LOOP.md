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

### 0.c Classify the change by blast radius — BEFORE the first write
Pick the tier from what the change can MOVE in the plant, not from its file count or effort. Record the tier and
its evidence (which slots control logic reads — staging, commands, modes — and whether the new ones are among them)
as the FIRST line of the feature doc. The tier sets the ceremony and a wall-clock budget; the non-skippable check
floor (§5) applies to every tier.

| Tier | What it covers | Budget | Ceremony |
|---|---|---|---|
| **P0 Cosmetic** | HMI copy, CSS, a default value, a comment; no slot/schema change | ≤ 10 min | inline, no feature doc, no delegated writer; build the one touched group; structural readback only |
| **P1 Additive indicator** | a new READONLY/alarm-only slot + its UI mirror, never read by staging/commands/modes | ≤ 25 min | feature doc, one writer given the recipe map (`skill/SKILL.md` § Recipe), one focused pure test of the new latch run with the feature ENABLED, touched groups built |
| **P2 Control** | changes a state machine, staging, rotation, protection, defrost | ~45-60 min | the sequence restated in field terms and confirmed first (`types/logic.md` § RT control logic), the Behavior decisions gate (§1), the design-shard checklist (`ORCHESTRATION.md` §3), full TDD, native review |
| **P3 Structural / deploy risk** | a facet/unit, persistence, schema rename, boot path, link-target flags | no cap | P2 + the structural checks before deploy (unit exists on the target, no `READONLY` on a link-in target, consumer-impact notice) + a boot smoke on the target distribution and version |

- Classify BEFORE the first write; reclassify (and say so in the feature doc) when exploration shows the change
  reaches a higher tier. A P0/P1 change that turns out to touch a slot control logic reads is P2.
- A run that exceeds its tier budget records the overrun (planned vs actual, and which phase ate the time) as retro
  input — an overrun is evidence to tune the budgets, not a failure to absorb silently.
- Speed comes from the tier and from reused maps, never from dropping the floor. `[ev: retro panccadia-commissioning-lessons Δ14]` `[ev: retro change-tier-time-budgets Δ1]`
- **Size review candidates at planning time:** when the feature forecasts more than ~1000 authored lines (adapter +
  generated slotomatic region count), plan one native review per work-unit commit from the start — review each commit
  from a detached worktree (`git worktree add --detach <wt> <commit>`) against `--base-ref <parent>` — instead of
  discovering a lens context-budget stop at the end on the accumulated diff. `[ev: retro comppan-pressure-staging Δ1]`

## 1. Design
- State the module's job, its profiles (rt/ux/wb), and the slot/endpoint contract in one paragraph.
- For a dashboard: the facade slots (display link-in + writable config), the servlet routes, the JSON `{v,st}` contract, the HMI resolution.
- **New module skeleton:** run `toolbelt/scaffold-module.sh <ModuleName> <out-dir>` to emit a pre-slotomatic tree from `fixtures/MinimalPan`; exits 0 ok / 2 usage / 3 env (skeleton missing). [ev: retro tool-integration]
- **Behavior decisions — HARD gate before the first source write of any rt control, protection, automatic action
  or operator-facing UI change:** list every behavior the code must decide that the user has not stated explicitly
  (generate the list from the question catalog in `METHODOLOGY.md` § Domain correctness, not from memory). Each one
  becomes a question with 2-4 concrete options — the recommended one first and marked, each with its field
  consequence in plain terms — plus a free "your own option" answer (format: `skill/SKILL.md` Execution Steps). No
  behavior on the list is coded until the user answers. Record the answers in a `## Behavior decisions` table in
  the feature doc (question, options shown, answer, who, date). This extends the field-terms confirmation of an
  ambiguous phrase (`types/logic.md` § RT control logic) to EVERY unstated behavior. `[ev: retro behavior-decisions-ask-dont-assume Δ1]`

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
- **A path mapped twice belongs in a recipe:** a read-only indicator carried from rt to the dashboard follows the
  cached hop list in `skill/SKILL.md` § Recipe — add a read-only indicator (rt → dashboard); start from the sibling
  slot you are mirroring instead of re-tracing the path. `[ev: retro change-tier-time-budgets Δ2]`

## 3. Preview (UI types) — BEFORE compiling
- `python3 /home/cristian/niagara-research/tools/dashboard-preview.py --rc <mod>/src/rc --prefix /<mount>`
- Open `http://localhost:<port>/hmi` (1280×800 frame). Iterate design here — refresh, no build.
- **Operator preview + explicit OK is a REQUIRED gate before building any `-ux` change (not optional):** seed the mock with the state that triggers the new behavior (e.g. `Cuarto3/evapNValveState` to exercise the output LEDs), so the operator actually sees it. The mock does NOT run `-rt` logic, so the preview approves the DASHBOARD's behavior, not the physical rt effect — say that to the operator. An `-rt`/`-wb`-only change has no preview (approved by design + pure tests). [ev: retro self-retro-preview-gate · T5]

## 4. Build — the ONLY valid build
- **ONE command does everything:** `toolbelt/build.sh <module-root=GROUP dir> <MOD> [niagara_home]` chains all four steps automatically: preflight (§0.b env checks) → gradle clean + slotomatic + jar → verify gate → report-module punch-list. It also runs the build-location WARN (§4.c) and the deployed-baseline drift gate (§4.c) around the gradle step. The agent runs one command, not four. Use `--no-preflight`, `--no-report`, and/or `--no-drift-check` to skip those bookend steps for inner rebuild loops where the environment is known-good and a full punch-list is not needed yet. `toolbelt/preflight.sh`, `toolbelt/verify-module.sh`, and `toolbelt/report-module.sh` remain available as standalone tools.
- **`build.sh` argument order:** the first argument is the GROUP directory (the gradle root, e.g. `Cliente/Leon-Guanjuato/Paccadia`), NOT the module sub-directory. `niagara_home` (arg 3 or the env var `NIAGARA_HOME`) is REQUIRED — the script exits 10 (env) when it cannot resolve it. A client multi-project layout places `./gradlew` at the GROUP ancestor, so always pass the GROUP dir, not a profile dir. `[ev: retro tree-selection-and-schema-risk-baseline Δ3]`
- Three roles (build-verify.md §Doctrine): `toolbelt/build.sh` is the recommended WSL build (clean + slotomatic for every profile with sources + jar, then it calls the gate); `scripts/ng-deploy.sh --strict-slotomatic` is the station deploy wrapper (backup→build→copy→verify; strict aborts if annotations changed without slotomatic, and its slotomatic guard is rt-only); `toolbelt/verify-module.sh` is THE gate, run on the built jars.
- Confirm: bytecode major **52**, jars **signed**, no raw-double facet. (build-verify.md)
- A `gradle :jar` with the default JDK is NOT a build.

### 4.a Gradle task matrix (`niagara-module` plugin tasks) — when to run each

In the inner loop run only what the tier needs (P0: build the touched group; P1: the new latch test + that module's
existing pure suite); the full builds and the non-skippable floor run once at task close — §5. `[ev: retro change-tier-time-budgets Δ3]`

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

### 4.c Version-bump checklist (before any shipped-bytes commit)
- `vendorVersion` (in `module.xml` / `gradle.properties`) MUST be bumped on EVERY change to shipped bytes — a schema change (slot add, remove, retype, rename) OR a behavior-only change with no slot touched (Java logic, `rc/` assets, resources). Software Manager compares versions, not bytes: a same-version rebuild reports "Up to Date" and is silently skipped even when the fix is real. On reload the station re-decodes `config.bog` against the new module's type/slot registry; a retype or remove is a schema-risk OUTAGE. `[ev: corpus B807]` `[ev: corpus B795]` `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]`
- `bajaVersion` is the Niagara platform API target set in `gradle.properties` / `settings.gradle.kts`; do NOT bump it between normal builds — it follows the `niagara_home` chosen at build time and is managed by the plugin. `[ev: corpus B807]`
- Restart mandatory for any `-rt` or `-wb` jar change (Java classes loaded at boot); a `-ux`-only change needs no restart — browser hard-reload only (§6). `[ev: corpus B807]`
- **`toolbelt/build.sh` carries an automatic deployed-baseline drift gate — you do not have to remember this checklist by hand.** Before a rebuild's `:jar` task overwrites `<niagara_home>/modules/<jar>`, `build.sh` snapshots the currently-installed jar's shipped-bytes content hash + `vendorVersion`; after the build, if the new jar's content differs from that snapshot but `vendorVersion` did not change, the build FAILs (exit 51) with the version-bump reminder instead of silently producing a jar Software Manager will call "Up to Date". `--no-drift-check` opts out for a niagara_home that is not the real deploy target. `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]`
- **`toolbelt/build.sh` also WARNs (non-fatal) when the gradle root or `niagara_home` sits on a WSL 9p/drvfs mount (`/mnt/<drive>`)** — Gradle's per-file work is far slower there than on ext4 (a measured file walk was 10x slower on 9p). See `build-verify.md` "Build location — ext4 vs 9p/drvfs". `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ7]`
- Run `toolbelt/schema-risk.sh` before any bump that touches slots; see §6 for the MANDATORY deploy gate. `[ev: corpus B795]`

## 5. Verify gate (before "done")
- Run `METHODOLOGY.md` (common) + the `types/<type>.md` checklist against the built module. Every item pass, or fix it.
- **Non-skippable floor — survives an explicit request for maximum speed:** the build (`toolbelt/build.sh`),
  `toolbelt/verify-module.sh`, `toolbelt/schema-risk.sh`, one focused RED/GREEN on the changed behavior, and — before
  a production deploy — the cold-boot smoke on a station of the target distribution and version. Every tier runs it
  (§0.c). Run it ONCE at task close, not per edit: speed comes from not re-running unaffected suites mid-loop, never
  from skipping the floor. A session under speed pressure records in the feature doc, under `## Checks skipped`,
  exactly which OTHER checks it skipped and why — never left implicit in the commit history. `[ev: retro change-tier-time-budgets Δ3]` `[ev: retro panccadia-commissioning-lessons Δ12]`
- **Pre-gate (run before `verify-module.sh`) — the lint list is GENERATED, not enumerated here:** `toolbelt/INDEX.md` lists every `toolbelt/lint-*.sh` with what it checks, its usage, its exit contract, its evidence and whether `report-module.sh` already runs it (**Auto** column). `toolbelt/build.sh` → `report-module.sh` runs the Auto=`yes` lints per profile; run each Auto=`no` lint by hand on every profile its usage names (e.g. a `<wb-src-root>` lint on each -wb profile with Java sources, an `<ux-src-root>` lint on each -ux profile with an rc/ SPA). A new or changed lint updates its own header (line 2 `# <name> — <what it checks>`, a `# Usage:` line, an `# Exit:`/`# Exits:` line, its `[ev: ...]` tags), then `toolbelt/gen-lint-index.sh` regenerates the index; `toolbelt/gen-lint-index.sh --check` (CI) exits 1 on a stale index. Never enumerate or count lints in this file — that inline list was a merge-conflict hotspot that drifted. `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`
- **Non-lint pre-gate tools (not in the lint index):** `toolbelt/slot-coverage.sh [--strict] [--facade] <module-include.xml> <module.lexicon>` (type-set lexicon coverage; empty or missing lexicon with declared types exits 1; a missing `*Panel`/`*Facade` type — every missing type under `--facade` — FAILs by default) [ev: retro panccadia-commissioning-lessons Δ9]; `toolbelt/schema-risk.sh <before-dir> <after-dir>` (two-snapshot slot diff before deploy; verdict SAFE/LOSSY/OUTAGE, exits 0/1/2/3/4 — exit 2 means the slot change would break saved data) [ev: retro tool-integration] [ev: retro campaign7-plano]; `toolbelt/verify-module.sh --plano <ux-profile>/src/rc/index.html` (when a -ux profile is present); `toolbelt/rc-scan.sh <ux-artifact-dir> [--strict] [--profile <ui_profile>] [--legacy]` (browser-resource lint over rc/ assets: hardcoded ORD/host literals, bare .catch(()=>{}), null display branches, plus the frontend-standard checks browser-floor (FAIL under --profile hmi|both), disabled-gate, fetch-no-signal, setinterval-async, innerhtml-server, datauri-budget, orphan-page, inline-block-size — check ids and severities in its header; exit 1 = FAIL) [ev: retro campaign8-rc-scan] [ev: retro dashboard-frontend-reliability-rules Δ5]; `toolbelt/generate-wiring-map.sh <facade-src-dir> [--lexicon <module.lexicon>]` (scaffolds `docs/wiring-map.md` from the facade @NiagaraProperty OPERATOR-config + SUMMARY-display slots, each row with the Workbench display name from the lexicon and a full-ord column, for §6.b commissioning verification) [ev: retro live-commissioning-verification-gaps]; `toolbelt/lib/method-boundary.sh` (shared method-boundary awk library — sourced, not invoked; fragment rule: edit the shared fragment, never overwrite a consumer lint's parser block separately) [ev: retro campaign11-shared-method-boundary]; `toolbelt/eslint.config.mjs` (the kit ESLint flat config for `-ux` rc js — ecmaVersion 2020 = the panel floor, `max-lines-per-function` 60, `no-unused-vars`, `eqeqeq`, `no-console` except error; run by report-module.sh; pinned install `npm install --prefix toolbelt/eslint`, which also provides acorn for `lint-vendor-floor.sh`) [ev: retro dashboard-frontend-standard Δ10]; `node toolbelt/hmi-sweep.js --url <preview-url> [--scenario name=query] [--subtab <sel>] [--target <sel>]` (HMI no-scroll + target-visibility sweep over every nav view at 1280×800; puppeteer-core + Chrome, exit 4 when unavailable) [ev: retro comppan-fase2-amps-alarms Δ3].
- **Lint doctrine the index rows do not carry (rules, not an index — do not append new lints here):** run `toolbelt/lint-guard-pins.sh --strict` as the LAST pre-gate step; every lint declares a `# Mutation: <fixture-id> -- <what it flips>` guard-pin resolved against a bats `@test` [ev: retro campaign11-lint-guard-pins]. `lint-timers.sh` splits usage (exit 2) from env (exit 3) — the K20 per-lint contract, `lint-timers.sh:44/:58/:63` [ev: retro campaign10-lint-timers-scope]. `lint-timers.sh`, `lint-silent-protection.sh` and `lint-ext-writable-shape.sh` share the PEAK-depth parser in `lib/method-boundary.sh` [ev: retro campaign11-shared-method-boundary]. `lint-ext-writable-shape.sh` exempts PER SLOT: a `do<Action>()` body must write THAT slot (S22 contract change; a newly reported slot is a fixed false negative, not a regression) [ev: retro campaign10-ext-writable-per-slot]. `lint-write-path.sh` semantics: uncovered OPERATOR slot FAIL always exits 1; STALE (matrix row naming a slot absent from all source names, per-row, `[concept]` exempts only the marked row) and DRIFT (a `[concept]` row whose slot IS in the covered set) are advisory, promoted to exit 1 by `--strict`; `--bog` adds link-traced dashboard slots [ev: retro campaign8-write-path] [ev: retro campaign10-write-path-stale] [ev: retro campaign11-concept-row-drift].
- Provenance of the former inline lint enumeration (frozen; new lints cite in their own header, which the index carries): [ev: retro 2026-09-17-umbrelladashboard-module-creation] [ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8] [ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ3] [ev: retro campaign8-lint-delays] [ev: retro campaign8-lint-servlet] [ev: retro campaign8-structure] [ev: retro campaign8-wb-audit] [ev: retro campaign9-demand-scope] [ev: retro campaign9-ext-writable-shape] [ev: retro campaign9-silent-protection] [ev: retro live-diagnosis-hardening-deltas Δ1] [ev: retro live-diagnosis-hardening-deltas Δ3] [ev: retro live-diagnosis-hardening-deltas Δ5] [ev: retro module-hardening-failure-modes-deltas Δ11 BLD2] [ev: retro module-hardening-failure-modes-deltas] [ev: retro new-lints] [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1] [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ5] [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas].
- Every lint that ships in a campaign carries a REAL-TREE smoke on the four client module roots: exact COUNT + SUBJECT of each finding + one ABSENCE pin (a slot that must NOT be flagged) — a bats fixture alone is not acceptance (METHODOLOGY K22). `[ev: retro campaign8-close-process-meta-lessons §lesson 11]`
- The automated half of the gate is `toolbelt/verify-module.sh <jars…>` (bytecode 52, signature, type resolution; `--target-version` / `--stored` / `--src` opt-in). A jar that has not passed it does not go to a station. `--src` sub-checks: `typecount`, `facets` (raw-number MIN/MAX), `facets-req` (OPERATOR numeric without facets key; setpoint/count-like without UNITS/PRECISION — WARN), `ord-literal` (hardcoded station:|slot:/ string — WARN) [ev: retro campaign8-facets-lint].
- **Aggregated punch-list before hand-off:** run automatically by `toolbelt/build.sh` at the end of every successful build (pass `--no-report` to skip for inner loops). To run standalone: `toolbelt/report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>] [--profile <ui_profile>] [--legacy]` — composes the non-lint checks above per profile artifact (schema-risk per D9a, rc-scan per -ux artifact with the module's `ui_profile` from its BUILD-STATE envelope as `--profile` and `--legacy` for a deployed module not yet restructured, ESLint on -ux rc js, optionally triage-console) plus every lint marked Auto=`yes` in the generated `toolbelt/INDEX.md` (per-artifact or once per module-root, as its usage names) — the index, not this bullet, is the list of what it runs `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`; exit 1 = punch-list has FAILs that block hand-off. [ev: retro campaign7-report-module] [ev: retro campaign8-report-integration] [ev: retro report-module-orphan-wiring] [ev: retro build-sh-auto-chain]

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
- **Audit the station bog for ghost slots, dangling links, orphan handles, proxy-link safety, and station-logic wiring:** `toolbelt/bog-audit.sh <config.bog|file.xml> --module <MOD> [--source-dir <src-dir>] [--strict]` (CHECK1-CHECK20; exit 1 = any FAIL). Runs from the bog alone for CHECK1/8/9/10/11/12/13/14/15/16/17/18/19/20; add `--source-dir` for the source-coupled checks (CHECK2-7). Proxy-link-safety (CHECK11) fires when an own-module output is linked to a BooleanWritable/NumericWritable with no explicit fallback — the writable holds last state on station stop/reload, masking the fault. Station-logic checks (CHECK13-19): relay-double-source, own-output-unlinked, sensor-crossed-by-name, hasDefrost<->DefrostController sibling, roomN-index-mismatch, tile-number consistency, link-direction. Numeric-to-boolean-direct (CHECK20): a multi-state/enum slot (name contains Mode/Command/Cmd) linked directly into a boolean gate (BOrLogic/BAndLogic/BNotLogic) with no Equal(x,N) intermediate — the `!=0`-as-true coercion silently merges HAND and OFF states; add an `Equal(slot, value)` block between the numeric source and the gate. [ev: retro campaign8-bog-audit] [ev: retro campaign8-station-logic] [ev: retro control-model-limits-live-commissioning Δ4]

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

**The kit — not the operator — owns the watchdog role.** Run `toolbelt/commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>] [--strict]` to close this gap: it orchestrates the Wave 3 lints (lint-config-sanity.sh CS1-CS4, lint-status-parity.sh, lint-recovery-path.sh) over every `-rt/src` dir and folds bog-audit.sh CHECK11 (proxy-link-safety) + CHECK13-19 (station-logic) + CHECK20 (numeric-to-bool-direct) into a single §6.b punch-list. Rows: `STATUS  commissioning  <check>  <detail>`; exit 0 clean / 1 any FAIL / 3 usage. Without `--bog`, the station checks emit SKIP rows with a note that a live config.bog is required. A `MANUAL` footer lists the steps this tool cannot statically verify (hot-reload console, plant control, per-instance runtime values). Run `toolbelt/generate-wiring-map.sh <facade-src-dir>` first to scaffold `docs/wiring-map.md`. `[ev: issue-114-commissioning-verify]` Usage and exit contracts of the lints it orchestrates are rows of the generated `toolbelt/INDEX.md`; a lint added to `commissioning-verify.sh` is documented by its own header + a regenerated index, not by extending this paragraph. `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`

## 7. Retro + close (HARD close gate — not optional)
- **Lead merge/settle order:** merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger; a ledger settle on a reported-but-unverified merge records the wrong evidence revision. For parallel workers: rebase onto the new main tip before the QA ping so the blessed tip is the one that merges, not the pre-rebase base. `[ev: retro campaign8-close-process-meta-lessons]`
- **Every run ends by writing its retro** — run `toolbelt/new-retro.sh <module|kit> <slug>` and fill the stub (§1); a defect in a KIT CHECK or DOCTRINE additionally opens `toolbelt/kit-ticket.sh "<one line>"`. The retro is a PRECONDITION for "done", not an at-STOP afterthought — `toolbelt/sweep-build-state.sh --age` at orient (BUILD-LOOP §0.a) surfaces the accrued retro debt so it cannot be skipped across a continuous chain. [ev: retro campaign8-retro-loop]
- **Assumption register at close:** every behavior still decided without an explicit user answer (deferred by the
  user, or discovered late) is listed in the feature doc under `## Assumptions still open` and repeated in the close
  message as "Assumptions I made — confirm or change". An empty register is stated explicitly ("none"), never
  omitted. `[ev: retro behavior-decisions-ask-dont-assume Δ4]`
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
