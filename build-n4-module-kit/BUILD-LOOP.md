# BUILD-LOOP — the operational cycle for one N4 module

The contract the launcher runs. Follow it in order; the gates are not optional.

## 0. Orient (before touching anything)
- **Run a current kit:** `scripts/kit-resolve.sh` (niagara-tools repo root; the launcher runs it first) must print `KIT=<path> SHA=<sha> (origin/main, clean)`. A clean kit-main worktree strictly behind origin/main is fast-forwarded automatically (`kit-resolve: refreshed <old>→<new>`). Exit 2 = the chosen kit is dirty, unfetchable, ahead of or diverged from origin/main (unmerged commits), or a stale `$BUILD_N4_KIT`/default checkout (never touched): fix as printed before any build or gate. It lives in `scripts/` because toolbelt scripts never call version control (kit-links L2). `[ev: retro r10-stale-kit-resolution Δ1]`
- `corpus-nav find "<topic>"` — read the matching blocks. 90% is already documented; don't re-derive.
- Read the EXEMPLAR source for the type (SOURCES.md), verbatim — not from memory.
- **Reference first:** when the user supplies a reference implementation (a course lab, a vendor sample, an earlier
  module of theirs) or their own code for the same exercise, mirror it and the user's style (names, layout,
  permission file, palette) first, and state each deviation with its reason. For course code, apply the
  "what not to copy" fixes (niagara-research corpus B1242 §1242.4) only as named, opt-in changes. The N4 Developer
  course is a primary exemplar source: blocks B1230-B1258, synthesis B1242. `[ev: retro n4-course-as-exemplar-source Δ2]`
- Copying a reference file whole (diff first) and work on a laptop running Workbench: `METHODOLOGY.md` § Hands-on work.
- Pick the module type (SKILL.md decision table) → load `types/<type>.md`.
- **Read the client's version ledger before the first write of a new session on that client:** `odd/VERSION-LEDGER.md` in the client repository (scaffold: `templates/VERSION-LEDGER.md`). Check its `Class` column against the classes the new work is likely to hit — the template's "Root-cause classes" table maps typical new work to classes and is the only copy of that map — and read those rows' prevention deltas first. Verify a candidate against the CLIENT's own history, not just this session's. No ledger yet → create it from the template at this session's close. `[ev: retro panccadia-version-defect-ledger Δ2]`

### 0.a Orient from BUILD-STATE (before touching anything)
- Run **`toolbelt/orient-guard.sh <module-root>`** — asserts that the module root is under the sole legal build prefix `/home/cristian/modulos_niagara_n4/Cliente/`; exit 1 = wrong location, abort; exit 0 (PASS or WARN) = proceed. Set `BUILD_N4_CLIENTE_OVERRIDE=1` to warn-and-pass on an exceptional path. `[ev: retro module-worktree-location-retro]`
- Run **`toolbelt/kit-state.sh build-state`** and read the module you are about to build in its output. `BUILD-STATE.md` is GENERATED from the committed `BUILD-STATE.src.md` + `ledger.d/` and NOT committed, so never trust a local copy you did not just regenerate (`--write` materializes it; the `.githooks` do so on every pull). `[ev: retro ledger-generated-not-committed]` From its `build-state.v1` envelope + prose, tell the operator in ONE line:
  `<module> · built <last_build>/gate <verify_gate>/deployed <deployed> · next: <last_session tail> · open_issues=N · retro_pending=Y/N`.
- If `retro_pending: true`, the previous session left an OWED retro — writing it is the FIRST task unless the operator redirects.
- If the module has no section, this is a first build — say so, and add its section at close.
- `toolbelt/sweep-build-state.sh --age --today <YYYY-MM-DD> <retros-dir> <INDEX>` with `<INDEX>` = a file holding `toolbelt/kit-state.sh retro-index` output — retro-debt aging; call at orient and again at close.
- Resuming: list the newest files of the working tree before trusting a session summary (`METHODOLOGY.md` § Hands-on work).
- **Meta-work exemption:** auditing the kit / tooling / a retro, or a single one-off question, does NOT gate on orient — say so and proceed (mirrors the research-sdd PASO 0 exemption).

### 0.b Preflight (before the first build)
- Run **`toolbelt/preflight.sh <niagara_home> <gradle-root>`** — automates win-path, jdk8, plugin-pin, jar-lock checks; exit 0 = all clear. Pass **`--plugin-version <v>`** (or export `NIAGARA_PLUGIN_VERSION`) when the build overrides the plugin: plugin-pin then checks the version gradle will actually use instead of re-deriving a `settings.gradle.kts` default that may have drifted from this `niagara_home` (`build.sh` forwards its own `--plugin-version` automatically). `[ev: retro continuous-fan-post-defrost-delay Δ1]` For a project you did not generate (course zip, vendor sample) add **`--sample`**: the `sample-*` rows check the retarget (`niagara_home` present + same version, no `<…>` user-home placeholder, root signing block, no shipped `build/`). `[ev: retro n4-course-signed-jars-gate-fix Δ6]`
- Run **`scripts/check-client-source.sh --build-state BUILD-STATE.src.md --module <MOD> <module-root>`** — is this checkout the module's declared `source_of_truth` (`<path>@<branch>@<commit>`)? WARN when the module root is not in a git work tree or the field is undeclared/`unknown`; FAIL on a detached HEAD, the wrong checkout, the wrong branch, or a HEAD behind the declared commit (stale). Local refs only, never fetches. It lives in `scripts/` because toolbelt scripts never call version control (kit-links L2); a FAIL means you are about to build from the wrong tree — stop and switch. `[ev: retro client-source-of-truth Δ1]`
- **A fix branch must start from the installed line:** before building a client fix branch, run **`scripts/check-installed-line.sh <client-repo> <branch> <installed-ref>`** (`<installed-ref>` = the commit whose jars are on the station). Exit 1 = a GROUP `build.gradle.kts` would ship a `defaultModuleVersion` lower than installed (`DOWNGRADE`), drop an installed group (`MISSING`) or carry a suffix that cannot be compared (`INCOMPARABLE`) — rebase the fix onto the installed line, never build it; exit 2 = unknown ref or nothing to compare. Local refs only, never fetches. `[ev: retro r10-installed-line-guard Δ1]`
- **JDK 8** present (`ls /usr/lib/jvm`).
- **A second local test station** needs its own ports: in the new station's services set Fox 4911 → 4912 and Web
  443 → 444, then **save** the station, or nothing is written and the two stations collide. `[ev: corpus B1416 §1416.5 a]` `[ev: retro course-recordings Δ4]`
- **niagara_home chosen** = the LOWEST target version you must support, AND its pinned gradle plugin version (from `settings.gradle.kts`) is present in `<niagara_home>/etc/m2` — each install ships only one (build-verify.md).
- **Station live?** A running station LOCKS its `modules/<mod>.jar`. Free the lock FIRST (close Workbench, or stop a non-production station) and build directly; use a mirror (`toolbelt/mirror-niagara-home.sh`) ONLY for a live production supervisor you must not stop — see build-verify.md §Building against a running station. [ev: retro coldroompan-dashboardpan-freeze-stat-leds · B2]
- **This module's build target may differ from the last one's — read THIS module's `gradle.properties` + `settings.gradle.kts`:** a sibling in the same client repo can target a different `niagara_home` + plugin (e.g. ColdRoomPan → PowerB 4.15.3 / 7.6.22 vs CompPan → Honeywell 4.14 / 7.6.17); never carry over the previous module's target. [ev: retro module-palette-and-build-target · B6]

### 0.c Classify the change by blast radius — BEFORE the first write
Pick the tier from what the change can MOVE in the plant, not from its file count or effort. Record the tier and
its evidence (which slots control logic reads — staging, commands, modes — and whether the new ones are among them)
as the FIRST line of the feature doc; a P0 change has no feature doc, so it records the tier and evidence as the
first line of its commit message body instead. The tier sets the ceremony and a wall-clock budget; the
non-skippable check floor (§5) applies to every tier.

| Tier | What it covers | Budget | Ceremony |
|---|---|---|---|
| **P0 Cosmetic** | HMI copy, CSS, a default value, a comment; no slot/schema change | ≤ 10 min | inline, no feature doc, no delegated writer; build the one touched group; structural readback replaces review and design ceremony — the §5 floor still runs (a pure copy/CSS/comment change records its RED/GREEN as `n/a (no behavior change)`; a default-value change gets its focused RED/GREEN) |
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
- **Review sizing for every candidate, kit docs and retros included:** native review stops a candidate over about
  500 changed lines with `lens_context_budget_exceeded`. Plan large work as a chain of commits of at most ~400
  changed lines; review each one with `--base-ref <exact parent SHA>` (a SHA, never `origin/main`, which moves
  under the review), then open one PR for the chain (PR #270). After review, bring main in with
  `git merge origin/main` into the branch, never a rebase: a rebase rewrites the reviewed commits, so their review
  no longer covers the bytes that merge. `[ev: retro r8-t22-retro-deltas Δ6]`

## 1. Design
- State the module's job, its profiles (rt/ux/wb), and the slot/endpoint contract in one paragraph.
- For a dashboard: the facade slots (display link-in + writable config), the servlet routes, the JSON `{v,st}` contract, the HMI resolution.
- **New module skeleton:** run `toolbelt/scaffold-module.sh <ModuleName> <out-dir> [--type logic|dashboard|driver]` to emit a pre-slotomatic tree from `fixtures/MinimalPan` (logic, the default), `fixtures/MinimalDash` or `fixtures/MinimalDriver`; exits 0 ok / 2 usage / 3 env (skeleton missing). [ev: retro campaign7-tool-integration] The dashboard `-ux` servlet overrides `doGet`/`doPost(WebOp)`, never `doService(WebOp)`, which is not a `BWebServlet` method and failed `:<Mod>-ux:compileJava` on 4.14 and 4.15 (`types/dashboard.md` override contract; `tests/scaffold-module.bats` TC-DASH3). `[ev: retro r10-scaffold-station-test Δ6]`
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
`toolbelt/build.sh` for the first time. (Workbench's New Module wizard instead defaults to the Workbench user home;
point it at the client tree that `orient-guard.sh` accepts, never the user home. `[ev: corpus B1417 §1417.3 p]`)

1. **`chmod +x gradlew`** — the scaffold copies the Gradle wrapper but does not set the execute
   bit; `./gradlew` is non-functional without it.
2. **Set the host build-location keys once per machine, in the user-level `gradle.properties`:**
   `niagara_home`, `niagara_user_home`, `nodeHome`, and `org.gradle.java.installations.paths` go in
   `~/.gradle/gradle.properties` in WSL/Linux, `%USERPROFILE%\.gradle\gradle.properties` on Windows (`$GRADLE_USER_HOME/gradle.properties` when `GRADLE_USER_HOME` is set),
   never in the repository; the repository keeps only the machine-independent
   `org.gradle.java.installations.auto-detect=false` (lint-structure L12). Gradle reads that file before the build root's `gradle.properties` and a command-line `-P` beats both, so `build.sh -P…` keeps working (`[ev: https://docs.gradle.org/current/userguide/build_environment.html, read 2026-10-24]`).
   lint-structure L10 WARNs any of the four keys left in a client `gradle.properties`. Also align
   `gradlePluginVersion` / `settingsPluginVersion` in `settings.gradle.kts` to a sibling of the same
   SDK family (each Niagara install ships exactly one plugin version — see
   `build-verify.md §Build target & plugin version`). `[ev: retro r10-user-level-gradle-keys Δ1]`
   > **Superseded (2026-10-24, R10-D8):** "copy `niagara_home`, `niagara_user_home`, `nodeHome`, and
   > `org.gradle.java.installations.paths` from an existing sibling … Do NOT leave the scaffold's
   > commented-out (auto-detect) JDK/node block: a commented block … fails reproducibility on a Windows
   > `gradlew` build." The user-level file gives the Windows build its paths without committing them.
3. **Run `toolbelt/preflight.sh` + first build:** `preflight.sh <niagara_home> <gradle-root>`
   validates the environment (§0.b); then `toolbelt/build.sh <group-dir> <MOD>` runs the first
   full clean+slotomatic+jar+verify cycle.

`[ev: corpus B1016]`

## 2. Build the layers
- Follow `types/<type>.md` + `METHODOLOGY.md`. Keep a facade pure; keep control logic in rt; keep UI in ux/wb. For framework-extension authoring (custom service, ORD scheme, point extension, analytics node, job, watchdog): see `types/logic-authoring.md` (companion to `types/logic.md`).
- **What to READ for this layer, in priority order: `corpus-index.md`** — the curated map of the niagara-research corpus for module builders (B29–B1027). `corpus-nav FIRST` for a term; `corpus-index.md` for what to read by layer/priority (P0 before building).
- Apply the slot rules as you write each `@NiagaraProperty` (flags, facets/units, the annotation+generated+imports rule).
- **After editing a part's `*.gradle.kts`, Gradle-sync the IDE before you trust its imports:** a new `api(":x-rt")`
  (a module from another Gradle build or the install) or an `uberjar(...)` dependency does not resolve in IntelliJ
  until the sync runs. Keep the dependency block grouped NRE (Niagara Runtime Environment) / module / test. Forms: `types/module-wiring.md` §1.
  `[ev: corpus B1418 §1418.5 c]` `[ev: corpus B1419 §1419.4 a]` `[ev: retro course-recordings Δ5]`
- **A path mapped twice belongs in a recipe:** a read-only indicator carried from rt to the dashboard follows the
  cached hop list in `skill/SKILL.md` § Recipe — add a read-only indicator (rt → dashboard); start from the sibling
  slot you are mirroring instead of re-tracing the path. `[ev: retro change-tier-time-budgets Δ2]`

## 3. Preview (UI types) — BEFORE compiling
- `~/.local/lib/niagara-go/bin/dashboard-preview --rc <mod>/src/rc --prefix /<mount>`
- Open `http://localhost:<port>/hmi` (1280×800 frame). Iterate design here — refresh, no build.
- **Operator preview + explicit OK is a REQUIRED gate before building any `-ux` change (not optional):** seed the mock with the state that triggers the new behavior (e.g. `Cuarto3/evapNValveState` to exercise the output LEDs), so the operator actually sees it. The mock does NOT run `-rt` logic, so the preview approves the DASHBOARD's behavior, not the physical rt effect — say that to the operator. An `-rt`/`-wb`-only change has no preview (approved by design + pure tests). [ev: retro self-retro-preview-gate-wsl-tests-live-verify · T5]

## 4. Build — the ONLY valid build
- **ONE command does everything:** `toolbelt/build.sh <module-root=GROUP dir> <MOD> [niagara_home]` chains all four steps automatically: preflight (§0.b env checks) → gradle clean + slotomatic + jar → verify gate → report-module punch-list. It also runs the build-location WARN (§4.c) and the deployed-baseline drift gate (§4.c) around the gradle step. The agent runs one command, not four. Use `--no-preflight`, `--no-report`, and/or `--no-drift-check` to skip those bookend steps for inner rebuild loops where the environment is known-good and a full punch-list is not needed yet. `toolbelt/preflight.sh`, `toolbelt/verify-module.sh`, and `toolbelt/report-module.sh` remain available as standalone tools.
- **`build.sh` argument order:** the first argument is the GROUP directory (the gradle root, e.g. `Cliente/Leon-Guanjuato/Paccadia`), NOT the module sub-directory. `niagara_home` (arg 3 or the env var `NIAGARA_HOME`) is REQUIRED — the script exits 10 (env) when it cannot resolve it. A client multi-project layout places `./gradlew` at the GROUP ancestor, so always pass the GROUP dir, not a profile dir. `[ev: retro tree-selection-and-schema-risk-baseline Δ3]`
- Three roles (build-verify.md §Doctrine): `toolbelt/build.sh` is the recommended WSL build (clean + slotomatic for every profile with sources + jar, then it calls the gate); `scripts/ng-deploy.sh --strict-slotomatic` is the station deploy wrapper (backup→build→prechecks→copy→type-count verify→verify-module gate; strict aborts if annotations changed without slotomatic; modes A/C only run slotomatic, -rt plus -ux in mode A when -ux is annotated; the version-bump precheck runs in every mode, the schema-risk precheck and the gate in modes A/C — exits in §5 below); `toolbelt/verify-module.sh` is THE gate, run on the built jars.
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

- **Passing the REPO root instead of the GROUP dir** (a client repo nesting several gradle groups as siblings): `build.sh` finds no `./gradlew` in the argument or any ancestor, exits 10, and lists every gradle root up to 3 levels BELOW the argument — re-run with the one that holds `<MOD>/`. `[ev: retro comppan-fase2-amps-alarms Δ2]`
- **Dashboard profile pass-through:** `build.sh --ui-profile <hmi|lan|both|unknown> [--legacy]` forwards the module's BUILD-STATE `ui_profile` to `report-module.sh --profile` (and `--legacy`), so the automatic punch-list applies the same browser floor a standalone `report-module.sh --profile` run would. Without it, rc-scan runs with no profile.
- **Independent gradle groups build in parallel:** when a change touches modules in DIFFERENT gradle roots (e.g. a compressor group and a dashboard group), launch one `build.sh` per group concurrently (each in the background, then wait for both) instead of serially — each run has its own gradle root, its own jars and its own drift snapshot. Never run two builds on the SAME gradle root at once (gradle's project lock and `:clean` collide). Give a delegated writer the recipe hops (the files to touch and the sibling slot to copy) in its prompt rather than leaving it to rediscover them (the orchestration half is owed to `ORCHESTRATION.md`). `[ev: retro change-tier-time-budgets Δ5]`

**Station-lock recipe (exit 31):** `:clean` fails with "Unable to delete `<niagara_home>/modules/<jar>`" when a running station holds the lock (`toolbelt/build.sh:15,:82-88`). Fix: `toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>` creates a writable mirror so the plugin can copy freely without touching the live install (see §0.b). `[ev: corpus B807]`

**Post-jar copy lock (exit 32):** the plugin's LAST step copies the new jar into `<niagara_home>/modules/`; a Windows-side process (Workbench, a station) holding the old jar open on a `/mnt/c` mount makes that copy fail with `FileAlreadyExistsException` — invisible to `preflight.sh`'s jar-lock scan, which SKIPs 9p/drvfs mounts by design. `build.sh` recognizes the signature, exits 32 (not an opaque 30), and names each `build/libs/<jar>` rebuilt THIS run as the artifact of record (a profile gradle never reached is flagged "not rebuilt"). The verify gate did NOT run: either close the holder and re-run, or run `verify-module.sh` on the `build/libs` jar before any deploy. `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ3]` The classifier is route-independent: the Windows `gradlew.bat` route's backslash paths and Kotlin message (`Tried to overwrite the destination, but failed to delete it`) also exit 32. Always run QUALIFIED tasks (`:<MOD>-<part>:jar`) — a bare `gradlew jar` builds every subproject and fails on an unrelated jar the station holds; on Windows identify the holder with `tasklist` (`station.exe`, `wb.exe`): leftover `java.exe` processes are Gradle daemons (`gradlew --stop`). `[ev: retro n4-course-signed-jars-gate-fix Δ5]`

**Slot-o-Matic skip (exit 34):** gradle ended BUILD SUCCESSFUL but Slot-o-Matic logged `Could not process <file>.java` for a generated region whose stamp count is not exactly one, so the jar carries stale generated code. `build.sh` prints one `FAIL slotomatic-skip <path> (<reason>)` row per file (`toolbelt/lib/slotomatic-skip.sh`), restores the pre-build `modules/` jar and skips the gate. Fix: delete that file's generated region and re-run (METHODOLOGY K15). `[ev: retro r10-slotomatic-skip-fail-open Δ1]`

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

**From WSL:** a WSL-side Gradle `niagaraTest` (plugin 7.6.17) discovers 0 tests; run `gradlew.bat … :<part>:clean
:<part>:niagaraTest` through `cmd.exe` instead: in place when the gradle root is under `/mnt/<drive>/`, on a Windows
copy only when it lives on ext4 (`build-verify.md` § Raw gradle, `types/moduleTest.md` § Running from WSL), and
count the tests: `BUILD SUCCESSFUL` with 0 tests run is a failure. Workbench Tools → Run Tests is the fallback.
Do not count the compile-only `moduleTestJar` pass as test execution.  `[ev: corpus B961 §961.3]`
`[ev: retro testing-parity-btestng Δ1]`
`[ev: retro n4-course-handson-workflow Δ3]`

### 4.c Version-bump checklist (before any shipped-bytes commit)
- `vendorVersion` (in `module.xml` / `gradle.properties`) MUST be bumped on EVERY change to shipped bytes — a schema change (slot add, remove, retype, rename) OR a behavior-only change with no slot touched (Java logic, `rc/` assets, resources). Software Manager compares versions, not bytes: a same-version rebuild reports "Up to Date" and is silently skipped even when the fix is real. On reload the station re-decodes `config.bog` against the new module's type/slot registry; a retype or remove is a schema-risk OUTAGE. `[ev: corpus B807]` `[ev: corpus B795]` `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]`
- `bajaVersion` is the Niagara platform API target set in `gradle.properties` / `settings.gradle.kts`; do NOT bump it between normal builds — it follows the `niagara_home` chosen at build time and is managed by the plugin. `[ev: corpus B807]`
- Restart mandatory for any `-rt` or `-wb` jar change (Java classes loaded at boot); a `-ux`-only change needs no restart — browser hard-reload only (§6). `[ev: corpus B807]`
- **`toolbelt/build.sh` carries an automatic deployed-baseline drift gate — you do not have to remember this checklist by hand.** Before a rebuild's `:jar` task overwrites `<niagara_home>/modules/<jar>`, `build.sh` snapshots the currently-installed jar's shipped-bytes content hash + `vendorVersion`; after the build, if the new jar's content differs from that snapshot but `vendorVersion` did not change, the build FAILs (exit 51) with the version-bump reminder instead of silently producing a jar Software Manager will call "Up to Date". A jar the gate cannot list (no `unzip`, an unreadable jar) or a changed jar whose own `vendorVersion` cannot be read FAILs closed (exit 52, "cannot verify"), never a silent pass. `--no-drift-check` opts out for a niagara_home that is not the real deploy target. `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ2]` The gate reads an X.Y or X.Y.Z module vendorVersion and compares the two versions numerically per part, a missing part = 0 (`1.3` == `1.3.0`, `1.03` == `1.3`), so a two-part version such as chihuahua-rt `1.3` is read, not refused with exit 52. `[ev: code toolbelt/build.sh:282 _module_own_version, :295 _same_version]`
- **`toolbelt/build.sh` also WARNs (non-fatal) when the gradle root or `niagara_home` sits on a WSL 9p/drvfs mount (`/mnt/<drive>`)** — Gradle's per-file work is far slower there than on ext4 (a measured file walk was 10x slower on 9p). See `build-verify.md` "Build location — ext4 vs 9p/drvfs". `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ7]`
- Run `toolbelt/schema-risk.sh` before any bump that touches slots; see §6 for the MANDATORY deploy gate. `[ev: corpus B795]`
- **Before a production deploy of version N, pre-build a REVERT BUILD:** build and sign the last-known-good source under a version ABOVE N (e.g. before loading `x.6.0`, build the `x.5.0` source as `x.6.1`), keep it next to the N package, and record its jar sha256 in the feature doc. If N crashes or degrades the station, the operator loads the revert build at once (§6 roll-forward) instead of starting a fix-and-build cycle while the plant is down; the real fix then ships as the next version up. A revert is new bytes with a new number, never the old bytes. `[ev: retro roll-forward-recovery Δ2]`

## 5. Verify gate (before "done")
- Run `METHODOLOGY.md` (common) + the `types/<type>.md` checklist against the built module. Every item pass, or fix it.
- **Non-skippable floor — survives an explicit request for maximum speed:** the build (`toolbelt/build.sh`),
  `toolbelt/verify-module.sh`, `toolbelt/schema-risk.sh`, one focused RED/GREEN on the changed behavior (a P0 change
  with no behavior — copy, CSS, a comment — records `RED/GREEN: n/a (no behavior change)` instead), and — before
  a production deploy — the cold-boot smoke on a station of the target distribution and version. Every tier runs it
  (§0.c). Run it ONCE at task close, not per edit: speed comes from not re-running unaffected suites mid-loop, never
  from skipping the floor. A session under speed pressure records in the feature doc, under `## Checks skipped`,
  exactly which OTHER checks it skipped and why — never left implicit in the commit history. `[ev: retro change-tier-time-budgets Δ3]` `[ev: retro panccadia-commissioning-lessons Δ12]`
  The floor is enforced, not only stated: `report-module.sh` reads a SKIPped floor check (verify-module with no built jar, schema-risk with no `<artifact>/.deploy-baseline`) as verdict INCOMPLETE, exit 4, never CLEAN, and a run that also has a FAIL reads `ISSUES (+INCOMPLETE)` (exit 1), so clearing the FAILs never reveals the skipped floor late `[ev: retro r10-client-gate-retro Δ4]` — `build.sh` maps it to exit 53; `--first-deploy` declares a never-deployed module (schema-risk N/A), `--lint-only` a punch-list run that is not the floor. `scripts/ng-deploy.sh` (which calls gradlew directly) blocks a deploy with no verify-module gate (exit 50), runs the schema-risk SAFE precheck on `SCHEMA_BASELINE_DIR` (exit 12 without it unless `--first-deploy`; OUTAGE/LOSSY exit 53, `--accept-lossy` for LOSSY) and the version-bump precheck against the deployed jar (exit 51 same version, 52 unverifiable, 54 downgrade), mirroring the build.sh drift gate; the gate itself runs after the copy, in modes A/C only. `[ev: retro change-tier-time-budgets Δ3]`
- **A refactor keeps observable output, and a new shared helper is tested first:** before callers move to a shared
  helper, its unit test is written and seen RED, then GREEN; the refactor is proven by an unchanged golden or
  structural test of the callers' output (servlet JSON, rendered HTML, slot values, lint rows). The PR shows both the
  helper's RED→GREEN and an output check that did not change; a dedupe that changes behavior is a regression, not a
  cleanup. Test placement: `types/moduleTest.md` § What to test. `[ev: retro modular-line-efficient-quality Δ4]`
  A PR or retro that claims fewer lines measures them on named commits: `wc -l` of each touched file at the base
  SHA and at the tip it merges from (`git show <sha>:<path> | wc -l`), both SHAs stated, next to that output proof.
  An estimate is not a measurement: one `report-module.sh` refactor was reported as "~900", 882 and 885 lines
  (1610 → 885 at merge 9213b39). Review check: the counts reproduce from the stated SHAs.
  `[ev: retro r10-dedupe-pins-measured-cuts Δ3]` A port's test code is budgeted the same way, against the code it
  replaces, reusing `golden.Runner` before any test-only package (the rule: `go/README.md` § Adding a lint rule,
  step 3). `[ev: retro r10-stored-repack-go Δ3]`
- **Pre-gate (run before `verify-module.sh`) — the lint list is GENERATED, not enumerated here:** `toolbelt/kit-state.sh lint-index` prints the generated lint index, which lists every `toolbelt/lint-*.sh` with what it checks, its usage, its exit contract, its evidence and whether `report-module.sh` already runs it (**Auto** column). `toolbelt/build.sh` → `report-module.sh` runs the Auto=`yes` lints per profile; run each Auto=`no` lint by hand on every profile its usage names (e.g. a `<wb-src-root>` lint on each -wb profile with Java sources, an `<ux-src-root>` lint on each -ux profile with an rc/ SPA). A new or changed lint updates its own header (line 2 `# <name> — <what it checks>`, a `# Usage:` line, an `# Exit:`/`# Exits:` line, its evidence tags), and commit the lint alone: the index is generated and not committed (`toolbelt/kit-state.sh --write` materializes it), so two PRs that each add a lint never conflict on it, and a new `report-module.sh` member (it flips the lint's Auto cell) needs no index commit either `[ev: retro issue-459-truth Δ2]`; `toolbelt/gen-lint-index.sh --check` (CI, pre-push) exits 1 on a broken header. `[ev: retro ledger-generated-not-committed]` Never enumerate or count lints in this file — that inline list was a merge-conflict hotspot that drifted. `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`
- **Non-lint pre-gate tools (not in the lint index):** `toolbelt/split-package-check.sh [--strict] <project-root>` (PROJECT-level, run on ONE project checkout before a release, never on a folder of several checkouts, which reports every package as split (`split-package-check.sh:16`) `[ev: retro go-split-package-check Δ2]`, not per module: a Java package declared under the `src/` of two or more modules is a WARN row — N4 class loading is first-dependency-wins, so one copy is silently shadowed; `--strict` exit 1 is the client-repository gate, issue #142: install it per client clone with `scripts/install-client-hooks.sh [--force] <client-repo>` — copies `templates/client-pre-push` to `<client-repo>/.githooks/pre-push` (commit it), sets that clone's `core.hooksPath` and its local `n4kit.toolbelt`, and refuses (exit 3, nothing changed) when that would silently stop other hooks — an active hook in `.git/hooks`, or a global/system hooksPath — until they are moved into `.githooks/` and it is re-run with `--force`; the hook checks every pushed COMMIT and blocks on a split package (exit 1) and on any check it cannot run (exit 3, no toolbelt) — never a silent pass; a client repo with GitHub CI copies `templates/split-package-check.yml` instead, which pins the kit to a release tag) [ev: retro module-hardening-failure-modes-deltas Δ12]; `toolbelt/slot-coverage.sh [--strict] [--facade] <module-include.xml> <module.lexicon>` (type-set lexicon coverage; empty or missing lexicon with declared types exits 1; a missing `*Panel`/`*Facade` type — every missing type under `--facade` — FAILs by default) [ev: retro panccadia-commissioning-lessons Δ9]; `toolbelt/schema-risk.sh <before-dir> <after-dir>` (two-snapshot slot diff before deploy; verdict SAFE/LOSSY/OUTAGE, exits 0/1/2/3/4 — exit 2 means the slot change would break saved data) [ev: retro campaign7-tool-integration] [ev: retro campaign7-plano]; `toolbelt/verify-module.sh --plano <ux-profile>/src/rc/index.html` (when a -ux profile is present); `toolbelt/rc-scan.sh <ux-artifact-dir> [--strict] [--profile <ui_profile>] [--legacy]` (browser-resource lint over rc/ assets: hardcoded ORD/host literals, bare .catch(()=>{}), null display branches, plus the frontend-standard checks browser-floor (FAIL under --profile hmi|both), disabled-gate, fetch-no-signal, setinterval-async, innerhtml-server, datauri-budget, orphan-page, inline-block-size — check ids and severities in its header; exit 1 = FAIL) [ev: retro campaign8-rc-scan] [ev: retro dashboard-frontend-reliability-rules Δ5]; `toolbelt/generate-wiring-map.sh <facade-src-dir> [--lexicon <module.lexicon>]` (scaffolds `docs/wiring-map.md` from the facade @NiagaraProperty OPERATOR-config + SUMMARY-display slots, each row with the Workbench display name from the lexicon and a full-ord column, for §6.b commissioning verification) [ev: retro live-commissioning-verification-gaps]; `toolbelt/lib/method-boundary.sh` (shared method-boundary awk library — sourced, not invoked; fragment rule: edit the shared fragment, never overwrite a consumer lint's parser block separately) [ev: retro campaign11-shared-method-boundary]; `toolbelt/eslint.config.mjs` (the kit ESLint flat config for `-ux` rc js — ecmaVersion 2020 = the panel floor, `max-lines-per-function` 60, `no-unused-vars`, `eqeqeq`, `no-console` except error; run by report-module.sh; pinned install `npm install --prefix toolbelt/eslint`, which also provides acorn for `lint-vendor-floor.sh`) [ev: retro dashboard-frontend-standard Δ10]; `node toolbelt/hmi-sweep.js --url <preview-url> [--scenario name=query] [--subtab <sel>] [--target <sel>]` (HMI no-scroll + target-visibility sweep over every nav view at 1280×800; puppeteer-core + Chrome, exit 4 when unavailable) [ev: retro comppan-fase2-amps-alarms Δ3].
- **Lint doctrine the index rows do not carry (rules, not an index — do not append new lints here):** run `toolbelt/lint-guard-pins.sh --strict` as the LAST pre-gate step; every lint declares a `# Mutation: <fixture-id> -- <what it flips>` guard-pin resolved against a bats `@test` [ev: retro campaign11-lint-guard-pins]. `lint-timers.sh` splits usage (exit 2) from env (exit 3) — the K20 per-lint contract, `lint-timers.sh:49/:72/:77` [ev: retro campaign10-lint-timers-scope]. `lint-timers.sh`, `lint-silent-protection.sh` and `lint-ext-writable-shape.sh` share the PEAK-depth parser in `lib/method-boundary.sh` [ev: retro campaign11-shared-method-boundary]. `lint-ext-writable-shape.sh` exempts PER SLOT: a `do<Action>()` body must write THAT slot (S22 contract change; a newly reported slot is a fixed false negative, not a regression) [ev: retro campaign10-ext-writable-per-slot]. `lint-write-path.sh` semantics: uncovered OPERATOR slot FAIL always exits 1; STALE (matrix row naming a slot absent from all source names, per-row, `[concept]` exempts only the marked row) and DRIFT (a `[concept]` row whose slot IS in the covered set) are advisory, promoted to exit 1 by `--strict`; `--bog` adds link-traced dashboard slots [ev: retro campaign8-write-path] [ev: retro campaign10-write-path-stale] [ev: retro campaign11-concept-row-drift].
- **Kit tooling doctrine (rules for whoever writes or changes a kit check):**
  - **Go first.** A new tool or check is written in Go in the niagara-tools `go/` module
    (`github.com/angeles725/niagara-tools/go`): `go/cmd/<tool>/` over `go/internal/<pkg>/`, with the `.sh` name kept
    as a thin wrapper that gets its binary from `go/go-tool.sh` (layout and check commands: `CONTRIBUTING.md`
    § 2.2). A bash script gets small fixes only. A check that is not ported yet is written down as an owed
    `Go lint` item in a `ledger.d/` fragment `open_issue:` line — id, scanned files, rule, row format, exit
    contract and the cases that must WARN and stay clean (G1-G7 are the model). `[ev: retro r8-t22-retro-deltas Δ1]`
    `go_tool_bin`'s staleness walk watches what `go build` reads: every file under `go/` except `bin/`, `testdata/`
    and `*_test.go`, `//go:embed` assets included; a `*.go`/`go.mod` filter served a stale embedded asset
    (`tests/go-tool.bats` GT10-GT14). `[ev: retro r10-go-tool-walk-scope Δ1]` Before a cache or stamp replaces such a
    walk, count its stat calls: a check that is fail-safe against later edits stats the same files (a prebuilt stamp
    measured 12.5-16 ms against a 16-25 ms walk), so narrow the walked set instead. `[ev: retro r10-go-tool-walk-scope Δ2]`
  - **What to port.** Port a tool to Go when it PARSES structured input (Java, XML, jars, ledgers) or has had a
    parsing bug; pure orchestration glue (`toolbelt/build.sh`, wrappers that only sequence gradle or copy steps)
    stays bash. Every port proves stdout, stderr and exit code byte-identical to the
    bash version on the real `Cliente/` trees, records before/after timing, and keeps the existing bats suite as
    its black-box test. `[ev: retro r8-t22-retro-deltas Δ8]` `toolbelt/preflight.sh` was counted as glue until its
    checks proved to parse (settings scripts, the signing block, an `lsof` listing) and two truth bugs surfaced; it
    is `go/cmd/preflight` now. `[ev: retro r10-preflight-go Δ1]` A bats test that runs a Go-wrapped script under a
    restricted PATH builds the binary in `setup_file` with the full PATH and links `find` and `readlink` into the
    restricted one: the wrapper's staleness check is fail-closed and stays so (`tests/retro-loop.bats` RL4).
    `[ev: retro go-kit-ticket Δ1]` Retiring a recorder or a private loop greps every header that names it
    (`toolbelt/new-retro.sh` still named the deleted newretro recorder). `[ev: retro r10-wiringmap-runner Δ2]`
  - **Port byte-identical first, then fix** (steps: `go/README.md` § Adding a lint rule). Record the goldens from
    the bash lint on the current origin/main and re-record at the merge (main moved lint-servlet's walk under its
    port). Keep the bash quirks until old and new agree byte for byte on fixtures and real trees; a difference the
    shared code forces is declared in the port commit and pinned by its own case, never emulated: a raw-text
    emulation of what `xmlsrc` reads is the second tokenizer R8-D12 forbids (bog-audit's malformed bog,
    obix-link-audit's entity-spelled links). `[ev: retro go-bog-audit Δ8]` `[ev: retro go-obix-link-audit Δ1]`
    A bash defect the port cannot reproduce (station-snapshot lost every manifest entry on an unwritable TMPDIR and
    still exited 0; the port keeps no temp files) is a `truth-*` case whose `@kind` states the bash behaviour, so
    the fix is visible, not just the new output. `[ev: retro go-station-snapshot Δ3]`
    Byte identity proves the port faithful, not right: where a sibling parser reads the same input, diff the
    port's parsed graph against it once (bog-audit's env-gated dump against niagara-go `bog-nav` found three
    blind spots the python shared). `[ev: retro go-bog-audit Δ9]` Then fix each truth bug in its own RED-first commit, a known
    false negative included (never kept as a "limitation"); a changed golden puts `declared difference: <why>`
    above `-- args --`. Only a cosmetic quirk no real tree hits may stay (split-package-check prints a root
    module's full path). Use the engine, never a private copy: gawk `for (k in a)` order from `rules/awkorder`, a
    layout lint as a `lint.RootRule`, awk_lint_each's unreadable-file text via `Spec.ShortScanError`. An awk `\b`
    stays the backspace byte in the identity port, then is decided per check with real-tree row counts.
    `[ev: retro go-lint-silent-protection Δ1]` `[ev: retro go-lint-servlet Δ1]` `[ev: retro go-lint-config-sanity Δ3]`
    `[ev: retro go-lint-session-store Δ1]` `[ev: retro go-split-package-check Δ3]` `[ev: retro go-lint-config-sanity Δ1]`
    `[ev: retro go-lint-structure Δ1]` `[ev: retro go-e2-tiny-lints Δ1]` `[ev: retro go-lint-demand-scope Δ1]`
    A truth fix that tightens an operand checks the format's own grammar first: verify-module's first cut made a
    decimal SVG viewBox (`1920.5`, legal SVG) a usage error, which would have left real dashboards uncheckable
    `[ev: retro r10-verify-module-truth Δ1]`. A bash comparison that carried no tolerance (exact integer
    cross-products) gets a tolerance chosen and named when it moves to floats (`verifymodule` `ratioTolerance`,
    float rounding only) and pinned by a golden, never inherited `[ev: retro r10-verify-module-truth Δ2]`.
  - **An orchestrator port keeps its tests' stubs and its structural pins.** A Go port of a script that calls toolbelt
    members receives the invoked (unresolved) toolbelt directory from its wrapper (`COMMISSIONING_VERIFY_TOOLBELT`),
    because resolving it like `here=` silently runs the real lints instead of a test's stubbed toolbelt copy. At the
    switch, re-point every structural pin that greps the ported script: a bash-idiom grep over a file that is now a
    wrapper passes vacuously (RM45 now covers `report-module.sh` only, mutation RM-setu observed).
    `[ev: retro go-commissioning-verify]`
  - **A test that needs files outside the repository never runs in CI** (`XMLSRC_EXPAT_SUITE`, `XMLSRC_SWEEP_ROOTS`,
    `JAVASRC_DIFF_ROOTS`): the landing PR states that it ran them, with the counts. `[ev: retro go-xmlsrc Δ3]`
    Byte parity on client trees is such a test: the repository is public, so goldens, fixtures and test data come
    from kit fixtures and hand-built edge trees only, and the PR reports the parity as run counts that name no
    client. `[ev: retro go-slot-coverage Δ2]` A port of a tool whose defaults name clients keeps those default
    lines unchanged in its wrapper and passes them in the environment (`TRIAGE_CONSOLE_DEFAULT_*`); the Go test
    reads them from the wrapper, so a drift fails and the Go code gets no client string.
    `[ev: retro go-triage-console Δ4]` A kit default that names a place on disk is derived from `$HOME` (or an env
    override), never spelled with one user's home directory, and an empty `HOME` fails closed (exit 3) instead of
    collapsing the path to `/` (`toolbelt/orient-guard.sh`, `tests/orient-guard.bats` OG10).
    `[ev: retro r10-home-paths Δ1]`
  - **Record the shell's silent input drops as edge goldens before porting** (each is a fail-open the identity
    port keeps and a truth fix removes): `xargs` splits a path on a blank; one unreadable file makes gawk's fatal
    open skip every later file (clean under `2>/dev/null`); an open construct at a file end carries into the next
    file `[ev: retro go-slot-coverage Δ3]`; awk `NR == FNR` is true for the second file when the first is empty, so
    an empty "before" snapshot read every "after" slot as removed `[ev: retro go-schema-risk Δ1]`; grep under
    C.UTF-8 counts a line with an encoding error (Latin-1) but never prints it, a file with a NUL byte prints
    nothing and `$(...)` drops NUL bytes `[ev: retro go-verify-module Δ2]`; a `$(( ))` operand bash cannot evaluate
    (`08`, `2.5`, a bare name) drops the whole top-level command and the script goes on
    `[ev: retro go-verify-module Δ3]`; a `"<dir>"/*/` glob sorts with the trailing slash (`x-worktrees/` before
    `x/`) `[ev: retro r10-preflight-go Δ2]`; `find -path "*V*"` matches V anywhere in the path, the start point
    included, so a version is matched as an exact directory name `[ev: retro r10-preflight-go Δ3]`; the first hit
    of a `find` is readdir order (host-dependent on ext4), so a Go walk reads in bytewise order
    `[ev: retro r10-preflight-go Δ4]`; a `grep` stage outside the awk's `LC_ALL=C` runs in the caller's locale, so
    under UTF-8 a row holding a Latin-1 byte prints `binary file matches` instead of the row, exit 0
    `[ev: retro go-triage-console Δ2]`.
  - **A network tool's bats run against a local `127.0.0.1` stand-in** (plain http and a throwaway self-signed
    https), never a stub of the client binary: curl stubs pinned argv/stdin plumbing the port no longer has, while
    the stand-in cases passed on the bash script and the port alike (`tests/obix-link-audit.bats` `_station`).
    `[ev: retro go-obix-link-audit Δ2]`
  - **XML and `.bog` lints port onto `go/internal/xmlsrc` only** (R8-D12, `go/README.md` § XML rules). Its first
    consumer is lint-agent-on-shape; split-package-check reads no XML. Change `Open`'s allocation (about 23 bytes
    per input byte) only after a measured slow sweep. `[ev: retro go-xmlsrc Δ1]` `[ev: retro go-split-package-check Δ1]`
    `[ev: retro go-xmlsrc Δ2]` schema-risk and slot-coverage take the module type from the first `<type>` element
    through it (a grep reader missed a tag over two lines, counted a commented-out one and named a `<dependency>`),
    and a slot annotation that never balances is an UNKNOWN row, never dropped. `[ev: retro go-schema-risk Δ3]`
  - **A lint header claims only what a test pins.** A promised suppression marker or fallback with no code behind it
    is deleted, not kept as advice (lint-config-sanity LCS-doc). `[ev: retro go-lint-config-sanity Δ2]`
  - **Never splice source text into a regexp, never pass a path with `awk -v`.** A dynamic awk regexp built from
    source text (`ln ~ expr "..."`) that is not a valid ERE kills gawk on that file while the lint keeps exit 0/1:
    quote it or use `index()` (none left in the three bash lints, 2026-10-24). `awk -v FILE="$f"` reads the path's
    backslash escapes and mangles the row: use `ENVIRON` or `FILENAME` (left: `verify-module.sh:336`).
    `[ev: retro go-lint-delays Δ1]` `[ev: retro go-lint-ext-writable-shape Δ1]`
  - **One layered Go lint stack (user requirement).** `internal/javasrc` is the Java scanner only;
    `internal/lint` holds the `Rule` interface, the registry, the shared `Finding` type, the one output formatter
    and the one walker that parses each file once; each lint is `internal/lint/rules/<lint>/` (the rule and its
    tests); `cmd/<tool>/` stays a thin `main.go`, as in `go/cmd/gen-retro-index` and `go/cmd/check-facets-card`
    (PRs #288, #296). Adding a lint is one rule file plus its registration — never a copied walker, formatter or
    stripper. Bash follows the same rule: reuse `toolbelt/lib/*` (`scan-files.sh`, `awk-lint.sh`,
    `method-boundary.sh`) instead of a private copy. `[ev: retro r8-t22-retro-deltas Δ9]` A scanner helper only one
    rule needs stays in that rule; the second rule that needs it moves it into `javasrc` with a test.
    `[ev: retro go-lint-session-store Δ2]` `internal/lintheader`
    `Lines` is the one reader of a lint header (its leading run of `#` and blank lines, no line cap: the former
    fixed lines 1-60 window cut off header lines, `[ev:]` tags included) and `internal/gen` `File.Run` is the one
    check-or-write flow of a generator; a second copy of either is a finding. `[ev: retro r9-lintindex-header Δ1]`
    `[ev: retro r9-guardpins-header Δ1]` `[ev: retro truth-ported-rules Δ3]` `[ev: retro r9-m2-gen Δ1]`
    `internal/ktssrc` `Tokens` is the one Kotlin-script reader. `[ev: retro go-lint-uberjar-api Δ2]` lint-structure
    L15 (signing block) and L14 (plugin pin) match their patterns on its `Code` line view (comments dropped, literal
    contents removed), never on raw `*.gradle.kts` text, and a script it cannot read is a visible WARN or exit 3;
    split-package-check reads no gradle content (file-name markers only). `[ev: retro r10-ktssrc-gradle-lints Δ1]`
    `[ev: retro r10-ktssrc-gradle-lints Δ2]`
  - **Pin every shared helper with a sync test.** A PR that moves copies into one `toolbelt/lib/*.sh` file or one
    shared function also adds a bats test that fails when a caller stops using it or a private copy comes back.
    Models: `tests/lint-structure.bats` LS15-sync (`lib/signing-block.sh`, sourced by `build.sh`, equal entry by
    entry to `go/internal/signblock`, read by lint-structure L15 and the preflight port) and `tests/report-module.bats` RM-row-parser (one row parser in
    `report-module.sh`). Without the pin, "never a private copy" is prose and the next quick fix pastes the copy
    back. Review check: the PR names the pinning test and shows an OBSERVED mutation (re-inline one copy, the test
    goes red). `[ev: retro r10-dedupe-pins-measured-cuts Δ1]` `[ev: retro r9-m3-report Δ1]`
  - **A table-driven pin gives each row a positive control.** Each row carries a sample its pattern must match,
    checked before the absence, so a mis-escaped regex cannot pass silently (`tests/kit-packaging-pins.bats` PINS,
    #463). `[ev: retro r10-close-process-meta-lessons Δ6]`
  - **A dedupe stops at a call site that a generator or lint reads.** Before repeated call sites become a table,
    a loop or a generic runner, grep the kit for a generator, lint or test that matches them literally. Example:
    `toolbelt/gen-lint-index.sh` marks a lint Auto only when a non-comment line of `report-module.sh` invokes
    `"$TOOLBELT/<lint>.sh"`, so a generic `"$TOOLBELT/$tool.sh"` runner would turn the whole Auto column to "no"
    (PR #352 kept a two-line call site per member: the literal invocation, then the shared `lint_rows`). When such a
    reader exists, keep the literal call and share everything after it. The site-data table rule
    (`METHODOLOGY.md` § Domain correctness) does not override this. Review check: a refactor that turns literal
    invocations into data shows that grep and an unchanged `gen-*.sh --check`. `[ev: retro r10-dedupe-pins-measured-cuts Δ2]`
  - **No fork per input line.** A shell `while read` loop whose body pipes `printf`/`echo` of the record into
    `grep`/`sed`/`awk` (or reads it with `<<<`) costs a clone and an exec per line: 35-97 s on a client tree
    against 0.13-1.25 s for one awk per file with byte-identical output. Use `toolbelt/lib/awk-lint.sh`
    `awk_lint_each` (or Go). `tests/lint-fork-per-line.bats` scans every `toolbelt/lint-*.sh` and fails on a fork
    not in its frozen baseline (`tests/fixtures/lint-fork-per-line/baseline.txt`); a fixed fork deletes its
    baseline row, a new one is never added there. `[ev: retro r8-t22-retro-deltas Δ2]` `[ev: retro line-loop-lint-forks Δ1]`
    The same class covers a fork per ITEM over a listing captured once: `for jar in …; do printf "$LSOF_OUT" |
    grep -qxF "n$jar"` grows with items times listing size and hung `preflight.sh` for more than 10 minutes on
    410 jars (#432). Load the item set and the listing into one awk (set lookup, one pass). The guard above
    covers only `while read` loops in `lint-*.sh`, so any other script pins this with a bounded-time bats case
    at a realistic item count (hundreds), as `tests/preflight.bats` PF-jarlock-bulk-timeout does.
    `[ev: retro r10-fork-per-line-again Δ1]`
  - **Literal-aware and scope-aware Java scanning.** A lint that reads Java text never matches inside a string,
    a char or a text block, and never lets a nested lambda or anonymous class end the enclosing method. Use the
    shared `toolbelt/lib/method-boundary.sh` (`mb_strip` for comments with literals kept, `mb_parse` for method
    bounds) or the Go Java scanner `go/internal/javasrc` (T8k-1, #297); a private stripper is a finding. Four fail-open bugs in
    one round came from private strippers (lint-driver-scaffold #277/#283, lint-service-types #279/#285,
    `mb_strip` itself #278). `[ev: retro r8-t22-retro-deltas Δ3]` `[ev: retro truth-ported-rules Δ1]` The lint-write-path and lint-arbitrary-ord
    ports read `linescan.Code`. `[ev: retro go-lint-delays Δ2]` `[ev: retro go-lint-arbitrary-ord]`
    Match on blanked source (comments and literal contents removed: `go/internal/javasrc` `Strip` then
    `BlankLiterals`) and on the whole statement, not the raw line: raw lines flag a call inside a comment or a
    string (lint-delays, lint-timers, lint-no-system-out) and miss a call split over lines
    (lint-null-context-write). A finding row carries `file:line` with a real line number, never an empty line
    or a rule ID in that position (lint-timers file-level rows, lint-structure L10/L14); `Line` 0 is only for a
    finding about the whole file (`go/README.md` § Adding a lint rule). `[ev: retro truth-ported-rules Δ2]` A flag
    test over joined `@NiagaraProperty` text reads comment-stripped code too: lint-config-sanity CS2 and the
    slot-coverage OPERATOR set read raw text, so a READONLY slot whose annotation comment says "not OPERATOR" was
    flagged; both read stripped text now, as statusparity and facets-req already did. A golden recorded from the bash
    can pin such a false positive: the truth fix updates that want and names the difference.
    `[ev: retro r10-lint-fp-459-rcscan-slotcov Δ1]` Owed engine item G51. `[ev: retro go-lint-structure Δ3]`
    `[ev: retro truth-audit Δ3]` An awk regex never uses `\b`: in gawk and POSIX awk it is a backspace (0x08), the
    word boundary is `\y`, so `/\bclass\b/` never matches and the lint can never fire (lint-demand-scope, fixed by
    its Go port). Prefer the Go engine. `[ev: retro truth-audit Δ8]` `[ev: tests/awk-word-boundary.bats AWB1]`
    A backward walk (the binding or guard "N lines up") stops at the enclosing method or class-level lambda body
    (`javasrc.Scopes`, `Enclosing`), never at a line budget alone: lint-delays took an earlier method's floor and
    lint-servlet an earlier method's `try {`. A change to the shared scanner re-runs every rule's goldens and
    declares each changed row (#405 wrapped signatures: lint-size, lint-ws-placement). `javasrc.Methods` now reads two
    shapes (wrapped signatures, body depth) unlike `mb_parse`; a bash lint on `mb_parse` keeps the old reading until ported.
    `[ev: retro go-lint-delays-method-scope Δ1]` `[ev: retro go-lint-delays-method-scope Δ3]` `[ev: retro go-lint-servlet-try-scope Δ2]`
  - **Fix a lint false positive by modeling the framework, never by suppressing it.** Encode the framework behavior
    that makes the code correct, with a RED truth fixture (Baja's equal-value skip in `hotwrite.Hot`, #445); an
    allow-list or a suppression is a finding. `[ev: retro r10-close-process-meta-lessons Δ4]` lint-servlet
    cache-nofinger models the request contract: a `max-age` that only a `?v=`-fingerprinted request receives (the
    then-branch of a fingerprint-test ternary, or a private helper's boolean parameter that every caller passes such a
    test) is no row, while a build-time token the servlet cannot see stays one
    (`go/internal/lint/rules/servlet/fingerprint.go`). Before the RED test for an issue checklist, run
    `git log origin/main --grep '#<issue>'`: eight of the nine #459 bullets were already fixed.
    `[ev: retro r10-lint-fp-459 Δ1]`
  - **A truth case is RED on the old code for the reason it names** (METHODOLOGY.md § Kit maintenance, observed
    flips): a case that only exercises the bug can pass before the fix (a brace in a literal where the body already
    acts; a short zip payload stored, not deflated, so the old byte read found it). Pick the input that flips the
    outcome and assert the test's own precondition. A mutation spot check likewise uses a case that reaches the
    guarded code. `[ev: retro go-lint-license-gate Δ1]` `[ev: retro go-lint-bformat-call Δ1]` `[ev: retro go-lint-wb-threading Δ3]`
  - **Golden cases come from the recorder or `golden.Run`'s replay, never a second tree builder:** a rebuilt tree
    lost a final newline and moved an error position (3:1 vs 2:11). A shape the format cannot hold (an empty
    directory replays as a missing root) gets a direct Go test (`rules/licensegate` `TestEmptyDir`). The recorder
    copies every root argument (file roots too) so no machine path enters a golden: `grep -l /tmp/ testdata/*.txt` is
    empty before a golden commit. `[ev: retro go-lint-agent-on-shape Δ2]` `[ev: retro go-lint-license-gate Δ2]`
    `[ev: retro go-lint-link-target-flags Δ2]` Record the trees under a path with no segment the tool matches on:
    rc-scan's `-path '*/rc/*'` matched the recorder's own scratch directory. `[ev: retro go-rc-scan Δ2]`
    A command that reads jars records them as `(jar)` sections (entries in archive order with their method, stored
    or deflated), fixture class headers as `(class N)`, any other binary as `(base64)` and an empty directory as
    `(dir)`; record only the paths an argument names, so a jar's source tree does not double the golden.
    `[ev: retro go-verify-module Δ1]`
    A write-side tool's case pins every file the run leaves (the whole tree after the run, never a chosen file
    list), so a stub left behind or a write outside the target directory fails it; new-retro's out-of-dir write
    showed up only as an extra `@expect` file. Inject the clock into the command (`Command(today)`) and fake `date`
    in the recorder, never read the real clock in a case `[ev: retro go-new-retro Δ3]`. The golden harness has no
    update flag: a `declared difference` golden is copied from the `go test` "got" block, so the commit that
    declares it is reviewed line by line `[ev: retro r10-verify-module-truth Δ3]`.
    A Go port's goldens replay through `golden.Runner` and record through `golden/record/record.sh`
    (`go/internal/lint/rules/golden/record/README.md`): hermetic environment, the whole post-run tree (`@after`,
    `@gone`), modes, symlinks, directories, the exact case count and `@kind` pinned to `record.env`; no package keeps
    a private loop or recorder. `[ev: retro r10-golden-runner Δ1]` `record.env` `EXTRA` pins every file the script
    sources, not only the script (a preflight recording ran without `lib/signing-block.sh`).
    `[ev: retro r10-golden-runner Δ3]` A case's environment is only what it declares (`PATH`, `HOME` and `LC_ALL`
    defaulted), and the wrappers' Go binaries are built once before the cases, never under the case PATH.
    `[ev: retro r10-golden-runner Δ4]` A clock or a pid the bash exposes is faked in the case tree and injected in Go:
    a fake `date` prints the stamp `Host.Now` returns, so the manifest stays a plain recorded golden
    `[ev: retro go-station-snapshot Δ1]`; a recording that leaks `$$` becomes a `truth-*` case with the pid replaced
    by the injected `Host.Pid`, stated in its `@kind`, never dropped `[ev: retro go-kit-ticket Δ3]`. An external
    command is pinned with a fake in the case tree that echoes its argv and the files it is handed, never the host
    binary on the case PATH (the host gh sat in `/usr/bin`). `[ev: retro go-kit-ticket Δ2]` A golden that mirrors a
    bats run reads the bats fixture in place (`@REPO@/tests/fixtures/...`), never a copy that can drift or carry a
    client string. `[ev: retro go-triage-console Δ5]` A file name holding a tab cannot be a recorded entry: the
    recorder's snapshot index is tab-separated and loses it. `[ev: retro go-station-snapshot Δ2]` A new Go tool with
    no bash source still replays through `golden.Runner`: hand-written cases, no `record.env`, each `@kind` a
    `truth:` line, and a `testdata/README.md` naming where the fixtures' shapes come from
    (`go/internal/niagaratest`). `[ev: retro r10-run-niagara-test-go Δ1]` A golden-format conversion is proved by
    re-recording from the pinned bash and `diff -r`, not by re-reading: a converter bug that changes an input changes
    the bash output too `[ev: retro r10-golden-runner Δ2]`; the converter reads bytes, not text (Python text mode
    turned CRLF into LF, and a UTF-8 decode failed on a non-UTF-8 case): binary mode or latin-1, split on `\n`
    only. `[ev: retro r10-wiringmap-runner Δ1]`
  - **A write-side tool validates every operand before its first write and lands all or nothing**
    (`go/internal/newretro`, the Go port of `toolbelt/new-retro.sh`). The whole slug is checked, not a prefix
    (the bash glob read 6 characters, so `abcdef/../../x` reached `cp`) `[ev: retro go-new-retro Δ1]`; `--date`
    must be a calendar date and `--deltas` a count, both before the stub is written (exit 3), because `--date ../x`
    wrote outside retros/ yet flipped the envelope `[ev: retro go-new-retro Δ2]`; a section that is missing or has
    no `retro_pending` exits 3 before anything is written, and an already-true envelope is reported as such, never as
    a flip `[ev: retro go-new-retro Δ5]`. A shared file is rewritten through a temporary file and a rename, and a
    later failure removes the stub and restores the index `[ev: retro go-new-retro Δ4]`. Never size a loop, slice or
    builder from an input whose parse error was dropped: `strconv.Atoi` on an overflow returns the clamped maximum
    with its error, and one such golden (`--deltas 99999999999999999999`) took the host to 37 GB RSS; bound the value
    (0-99) and check the error in the parser `[ev: retro go-new-retro Δ6]`. A tool that read-modify-writes a
    committed file holds one exclusive `flock` on a directory that is never replaced, from its first read through
    its last rename (generators included): new-retro locks `retros/` itself, because a lock on BUILD-STATE.src.md
    or INDEX.md, which it replaces by rename, can land on a stale inode, and the kernel drops the lock when the
    process dies. Its race test forces the interleaving with a barrier after the read, never a sleep
    (`go/internal/newretro/concurrent_test.go`). `[ev: retro r10-new-retro-flip-lock Δ1]`
  - **Reuse the engine before adding to it:** a per-file rule may set `Finding.Path` to any subject (the runner fills
    only an empty one); a rule that needs literal text and code positions together reads `javasrc.MaskLiterals` (same
    line lengths) instead of mapping `BlankLiterals` columns back. An engine option a port does need (`Spec.IntFlags`)
    lands as its own green commit, then the port with its goldens. `[ev: retro go-lint-status-parity Δ2]`
    `[ev: retro go-lint-ws-placement Δ3]` `[ev: retro go-lint-size Δ3]` A tool whose contract is not a lint (its
    operand is not a source root, or its rows and arguments fit no engine mode) is its own `go/cmd/<tool>` over the
    shared packages, reusing the engine's walk, `Finding.Row` and `golden.Read`; its retro records the misfit
    (rc-scan, bog-audit, obix-link-audit, schema-risk with its verdict exits, slot-coverage, verify-module,
    preflight). Its goldens replay through `golden.RunMain` (`golden.Run` for a `cli.Main`), extended rather than
    copied when a port needs a section it lacks. `[ev: retro go-rc-scan Δ1]` `[ev: retro go-slot-coverage Δ1]`
  - **An XML or `.bog` lint reads through `internal/xmlsrc`** (`go/README.md` § XML rules): a station `config.bog`
    is zipped, so a text read sees compressed bytes and passes clean; a malformed file is one positioned error,
    printed once. `[ev: retro go-lint-bformat-call Δ2]` `[ev: retro go-lint-agent-on-shape Δ1]`
    A well-formedness rule the kit enforces on a `.bog` is checked against Niagara's reader first: where XML 1.0
    and `javax.baja.xml.XParser` disagree, the station's reader is the truth. `XWriter.safe` saves a NUL in a String
    slot as `&#x0;` and `XParser` reads it back, so `xmlsrc` reads U+0000 in a `bajaObjectGraph` with one named
    WARN (`Doc.Warnings`, `XML  WARN  <bog>:<line>:<col>  U+0000 in N places`) and keeps refusing it in any other
    document; never exit 3 for the whole file, never a silent pass. `[ev: retro r10-xmlsrc-nul-per-niagara Δ1]`
  - **A bash `while IFS= read -r x` loop drops an unterminated last line** (7 of the 13 Tridium lexicons end without
    one): a byte-identical port keeps the drop and its truth fix removes it; a bash lint that stays reads
    `while IFS= read -r x || [ -n "$x" ]`. `[ev: retro go-lint-lexicon-ascii Δ1]`
  - Owed code from the round-10 lint ports (client findings, rule gaps, engine fields) stays an `open_issue:` in
    the port's `ledger.d/2026-10-24-*` fragment (or `ledger.d/2026-10-24-r10-fold-late.md`), not doctrine:
    `[deferred: retro go-bog-audit Δ1]` `[deferred: retro go-bog-audit Δ2]` `[deferred: retro go-bog-audit Δ3]`
    `[deferred: retro go-bog-audit Δ4]` `[deferred: retro go-bog-audit Δ5]` `[deferred: retro go-bog-audit Δ7]`
    `[deferred: retro go-lint-arbitrary-ord Δ1]` `[deferred: retro go-lint-arbitrary-ord Δ2]`
    `[deferred: retro go-lint-arbitrary-ord Δ3]` `[deferred: retro go-lint-uberjar-api Δ1]`
    `[deferred: retro go-rc-scan Δ3]`
    `[deferred: retro go-lint-delays-method-scope Δ2]` `[deferred: retro go-lint-duplicate-helper Δ2]`
    `[deferred: retro go-lint-duplicate-helper Δ3]` `[deferred: retro go-lint-hot-write Δ1]`
    `[deferred: retro go-lint-hot-write Δ2]` `[deferred: retro go-lint-hot-write Δ3]`
    `[deferred: retro go-lint-inert-coordination Δ1]` `[deferred: retro go-lint-inert-coordination Δ2]`
    `[deferred: retro go-lint-inert-coordination Δ3]` `[deferred: retro go-lint-inline-site-literal Δ3]`
    `[deferred: retro go-lint-js-pair Δ1]` `[deferred: retro go-lint-js-pair Δ2]` `[deferred: retro go-lint-js-pair Δ3]`
    `[deferred: retro go-lint-lexicon-ascii Δ2]`. Moving the Python `str` helpers (`internal/htmltext/unicode.go`) to
    their own package waits for a second consumer (one today): `[deferred: retro go-lint-manual-labels Δ4]`
  - **Fail-closed input listing.** A toolbelt script never feeds a loop from a raw `find`; it lists its files once
    with `scan_files_or_exit` (or `awk_lint_list` / `awk_lint_run` / `awk_lint_strip_run`), because `find` read
    through `< <(...)` or a pipe hides its exit status and an unreadable sub-directory becomes a clean pass. Review
    check: `grep -n 'find "' toolbelt/*.sh` outside `lib/`. A new bash lint never pastes the parse/walk/exit
    skeleton: a per-file awk lint writes only its header, `awk_lint_parse`, its `main.awk` and `awk_lint_run`; a
    flagless line-loop lint opens with `awk_lint_parse <name> raw` and `awk_lint_list`, and ends with
    `awk_lint_exit` (`toolbelt/lib/awk-lint.sh` header); its own `scan_files` call or `had_err` tail is a finding.
    `[ev: retro awk-lint-skeleton Δ1]` `[ev: retro awk-lint-line-loops Δ1]` `scan_files` never prunes its start point
    (`-mindepth 1`); the Go lint-structure still does (`dotName`, kept for byte identity, so `lint-structure.sh .`
    skips L10), owed. Its list is newline-separated, so a path holding a newline becomes two missing paths; NUL
    separation is owed (needs code: `lib/scan-files.sh` and its `while read` callers). `[ev: retro go-lint-structure Δ2]`
    `[deferred: retro go-e2-tiny-lints Δ2]` A root with no input files is an ERROR (exit 3), never a
    silent pass (35 of 40 lints exited 0 on an empty root, 2026-10-22; Go engine switch `lint.Spec.EmptyError`,
    owed item G50; the four bash lints with a private empty-root row are Go rules with `EmptyError` now, and
    lint-vendor-floor exits 0 on an rc dir with no `vendor/` by its header). `[ev: retro r9-m1-walkers Δ1]` A walk
    follows a symlinked root like `find -H` (Go: `os.Stat` on the root), never a symlink below it and never `-L`
    (`tests/scan-files.bats` SF10, SF11). `[ev: retro symlink-root-walk Δ1]` A per-file read fails closed too:
    `grep … "$f" 2>/dev/null || true` reads an unreadable file as no hit and `xargs awk` splits a path on
    whitespace: the slot-coverage and schema-risk ports exit on an unreadable input now `[ev: retro go-schema-risk Δ2]`;
    verify-module's `--src` reads stay owed (phase B). `[deferred: retro r9-m1-walkers Δ2]` Go tools: `CONTRIBUTING.md` § 2.2 "Fail closed on I/O"; shell `exit` inside `$( )`:
    `CONTRIBUTING.md` § 6. `[ev: retro fail-closed-walks Δ1]` `[ev: retro truth-audit Δ2]` A tool that judges a log
    fails closed on a cut read: a line past `bufio.Scanner`'s cap (`bufio.ErrTooLong`) is a FAIL reason at its line
    number, never a silent end of input, because a failure after it would go unseen (`go/internal/niagaratest`).
    `[ev: retro r10-run-niagara-test-go Δ2]` A results-file verdict fails on skips and on any FAIL test-method, config
    methods included: TestNG's root counts leave `is-config` methods out, so a failed `@BeforeClass` reads failed=0
    with skips (`go/internal/niagaratest/testdata/testng-config-skips.txt`). `[ev: retro r10-run-niagara-test-testng Δ2]`
  - **A path filter in a gate skips only paths no check applies to.** A gate that picks its checks from the
    changed or staged paths (a hook, a CI step, a wrapper) states, for each class of path it can see (code, tool,
    doc, data, generated), which check runs or why none applies. An early exit is proven by a test that changes
    ONLY a skipped class, with a defect the gate's checks would catch, and asserts the outcome; a skip test that
    changes a clean file proves nothing. A data class that has a check gets one that fails closed: data paths
    configured with no check is an error, not a pass (niagara-research PR #46, `GATE_DATA_PATHS` +
    `GATE_DATA_CHECK`). Kit instance: `.githooks/pre-push:44-50` skips `sweep-build-state.sh` for a
    `retros/`/`ledger.d/`/`BUILD-STATE.md`-only range, the files sweep checks; CI runs sweep unfiltered, and the
    hook fix is an owed `open_issue`. Review check: the PR names the data-only test and shows it red on the old
    filter. `[ev: retro r10-path-keyed-gates-skip-data Δ1]`
  - **A hook that computes a push range handles both all-zero shas.** A new remote branch (all-zero remote sha) is
    gated from its merge-base with `origin/main`, never by diffing the lone local sha against the working tree; a
    delete (all-zero local sha) has nothing to gate; every git call refuses on failure (`.githooks/pre-push`,
    `tests/build-retro-sync.bats` H19-H21). `[ev: retro r9-m1-walkers Δ4]`
  - **An identity pin between shared copies does not skip where drift happens.** A test that keeps copies in two
    repositories byte-identical compares with the upstream's committed `origin/main`
    (`git -C <upstream> show origin/main:<path>`), not a sibling working tree, and skips only when the upstream
    repository is really absent, naming what it looked for. The niagara-research hook pins skipped in every
    worktree, and both downstream copies drifted unseen (module-navigator #11, niagara-help #10). Review check:
    the pin fails in a scratch layout with an old copy, and the message names the sync command.
    `[ev: retro r10-path-keyed-gates-skip-data Δ2]`
  - **Citation pre-filter.** `toolbelt/kit-cite-resolve.sh [--strict] <card>...` (`go/cmd/kit-cite-resolve`)
    resolves every evidence pointer of the given cards and prints one row per pointer with its status
    (RESOLVED, MISSING_FILE, MISSING_SECTION, MISSING_DELTA, LINE_OUT_OF_RANGE, DRIFT, BARE_ID, UNPARSED, AMBIGUOUS, TOOL_ERROR)
    and an excerpt of the cited line; a bare or partial path is searched in the repo, organized/, the versioned source
    and the client checkouts (`--clients`), every copy checked, and AMBIGUOUS lists the candidates, never a guess. RESOLVED means the cited place exists, never that it says the claim: a reviewer
    still reads the excerpt. Not findings: HISTORICAL_DRIFT (a moved target on a history card: retros/, ledger.d/,
    BUILD-STATE, CHANGELOG), UNVERIFIABLE_KIND (engram, live, VCS, command-run or decision evidence) and
    SKIPPED_TEMPLATE (`<stem>` placeholders); the summary counts LIVE and HISTORY apart. Exit 0 / 1 on a finding
    under `--strict` / 3 usage or an unreadable root. `[ev: retro go-kit-cite-resolve]` `[ev: ledger.d/2026-10-24-r10-kcr-status-split.md]`
    `[ev: ledger.d/2026-10-24-r10-kcr-paths-kinds.md]`
    A `guides-clean` line pointer carries a quoted anchor right after its lines (`` `G/<page>.txt:12` "a phrase of line 12" ``
    or the same after a tag's `guide <page>.txt:12`): the rendered guide copies shift on every re-render, so a bare line number
    drifts while still resolving. The phrase must sit on a cited line, give or take 2 lines; otherwise the pointer is
    DRIFT, naming the lines that hold it now (HISTORICAL_DRIFT on a history card). An anchor outside guides-clean is
    shown, not checked. `[ev: retro r10-kcr-guide-anchor-drift Δ1]` An ignored, regenerated tree (`versions/*/`) keeps
    no history, so a re-render erases what an old pointer read; write the anchor at citation time (the #511 sweep had to
    rebuild the old rendering with the renderer from git to recover it). `[ev: retro r10-kcr-guide-anchor-drift]`
    A pointer grammar matches every character a real file name uses: a pattern that rejects one (the dot inside
    `BUILD-STATE.src.md`) does not fail, its leftmost match silently reads a shorter, wrong name. Each such grammar
    pins a case per legal name shape (`go/internal/citeparse` `TestCardSources`), and a pointer that resolves only
    after a rewording is a parser bug, not a doctrine edit. `[ev: retro r10-citeparse-dotted-section-path Δ1]`
    The parser (`go/internal/citeparse`) reads `x.jar!entry` as a `jar` pointer (the jar resolved, the entry noted
    unchecked), `§ 11.1-11.3` as its two ends, every number of `§ 3, § 5.2` and `§§ 3, 5`, a backticked path with
    spaces (the course tree is a path base), and an unclosed `[ev:` inside a code span that closes on its line as
    literal code. Measure a parser fix with a kit-wide before/after diff of row keys, not only on the card that
    exposed it, and give each shape a wider pattern admits a negative test for the prose it must still reject
    (the first GREEN run added 4 MISSING_FILE rows and read backticked `code x-rt …` labels as paths).
    `[ev: retro r10-kcr-false-positives Δ1]` Round 2 reads elided placeholders (`…Δk`, `retro YYYY-MM-DD-…`) as
    SKIPPED_TEMPLATE, the `javap x.jar[!entry]` tag clause like the prose `javap of <jar>` form, a niagara-help
    relative path from `versions/<ver>/` first, and `x.zip!entry` after `code` or `course` as an `archive` pointer
    (the zip resolved, the entry unchecked). A triage class is a hypothesis until a kit card shows it: grep the kit
    and the pinned tests before writing code (two of six classes had no instance, or a test pinning the behaviour as
    intended), and diff the excerpt column with the status, because the version-first read changed excerpts but no
    status. `[ev: retro r10-kcr-false-positives-round2 Δ1]`
- Provenance of the former inline lint enumeration (frozen; new lints cite in their own header, which the index carries): [ev: retro umbrelladashboard-module-creation] [ev: retro dashboard-status-state-modeling Δ3] [ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8] [ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ3] [ev: retro campaign8-lint-delays] [ev: retro campaign8-lint-servlet] [ev: retro campaign8-structure] [ev: retro campaign8-wb-audit] [ev: retro campaign9-demand-scope] [ev: retro campaign9-ext-writable-shape] [ev: retro campaign9-silent-protection] [ev: retro live-diagnosis-hardening-deltas Δ1] [ev: retro live-diagnosis-hardening-deltas Δ3] [ev: retro live-diagnosis-hardening-deltas Δ5] [ev: retro module-hardening-failure-modes-deltas Δ11 BLD2] [ev: retro module-hardening-failure-modes-deltas] [ev: retro new-lints] [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1] [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ5] [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas].
- Every lint that ships in a campaign carries a REAL-TREE smoke on the four client module roots: exact COUNT + SUBJECT of each finding + one ABSENCE pin (a slot that must NOT be flagged) — a bats fixture alone is not acceptance (METHODOLOGY K22). `[ev: retro campaign8-close-process-meta-lessons §lesson 11]` Any new checker runs once on the real tree before its first push, and each false-finding class it shows becomes a pinned case before release (kit-cite-resolve: green fixtures, then 2046 non-RESOLVED rows on its first real run, 9 classes false). `[ev: retro go-kit-cite-resolve Δ1]`
- **Before releasing a new ADVISORY lint, smoke it read-only on one real client checkout and on decompiled vendor
  code,** and record the row count and one sample verdict (genuine / false) in the feature doc — the cheapest
  evidence of its noise level, and the place a scope rule (one checkout, not a directory of checkouts) shows up.
  Report counts only in this public repo. `[ev: retro deferred-lints-2026-10-03-close Δ5]`
- **A lint row is a lead until it is read:** before reporting a row as a defect, read the code at the installed
  commit and what the framework does there, and report what you read; two round-10 sweep rows were already guarded
  or named the wrong failure. `[ev: retro r10-close-process-meta-lessons Δ2]` A false-positive report names the
  client line and the rule clause that fires; the fixer re-runs the rule at the reported commit before choosing the
  fix (#459 named a silent-protection clause that was not the one firing). `[ev: retro issue-459-truth Δ1]`
- **A client-tree FAIL count is current only over live roots:** stale checkouts and `_salvage/` copies keep old
  defects (all 21 Cliente lint-link-target-flags rows came from them; the live tree had 0), so name the live module
  roots, or prune those, before reading a count as current. `[ev: retro go-lint-link-target-flags Δ1]`
- The automated half of the gate is `toolbelt/verify-module.sh <jars…>` (bytecode 52, signature, type resolution; `--target-version` / `--stored` / `--src` opt-in). A jar that has not passed it does not go to a station. `--src` sub-checks: `typecount`, `facets` (raw-number MIN/MAX), `facets-req` (OPERATOR numeric without facets key; setpoint/count-like without UNITS/PRECISION — WARN), `ord-literal` (hardcoded station:|slot:/ string — WARN) [ev: retro campaign8-facets-lint].
- **No test type ships in a production jar:** a moduleTest compile can append the part's `srcTest/` types to the
  production `module-include.xml` (`types/moduleTest.md` § `moduleTest-include.xml`, plugin-version trap). The guard
  is two FAIL rows, and either one blocks "done": `lint-agent-on-include-sync` sub-check `test-type-in-prod` (run
  by `report-module.sh`) and the `moduletest` row of `verify-module.sh --src`, which FAILs once per production
  `<type>` whose class is declared under the part's `srcTest/` or listed in its `moduleTest-include.xml` (a
  malformed XML file is exit 3, never a PASS). `[ev: retro r10-lint-test-types-in-prod Δ2]`
- **Aggregated punch-list before hand-off:** run automatically by `toolbelt/build.sh` at the end of every successful build (pass `--no-report` to skip for inner loops). To run standalone: `toolbelt/report-module.sh <module-root> [--target-version x.y] [--console-dir <dir>] [--profile <ui_profile>] [--legacy] [--wiring-map <file>]` — composes the non-lint checks above per profile artifact (schema-risk per D9a, rc-scan per -ux artifact with the module's `ui_profile` from its BUILD-STATE envelope as `--profile` and `--legacy` for a deployed module not yet restructured, ESLint on -ux rc js, optionally triage-console) plus every lint marked Auto=`yes` in the generated lint index (`toolbelt/kit-state.sh lint-index`) (per-artifact or once per module-root, as its usage names) — the index, not this bullet, is the list of what it runs `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`; the link-target-flags lint gets the wiring map (`--wiring-map`, else `docs/wiring-map.md` in or beside the module root), a slot-coverage facade/empty-lexicon FAIL is a FAIL row, and ADVISORY rows (console-only trips) are relayed with their own count, never as PASS or FAIL `[ev: retro panccadia-commissioning-lessons Δ9]` `[ev: retro alarm-console-design Δ3]`; exit 1 = punch-list has FAILs that block hand-off. [ev: retro campaign7-report-module] [ev: retro campaign8-report-integration] [ev: retro report-module-orphan-wiring] [ev: retro build-sh-auto-chain]

## 6. Deploy (station) — operator
- **MANDATORY before a live-station deploy — the §5 schema-risk verdict must be SAFE:** a **LOSSY/OUTAGE** verdict means the new jar's slots no longer match the station's saved `.bog` and it will fail to load. Proven LIVE: an OUTAGE-class retype (`BStatusNumeric`↔`BDouble`, `BRelTime`↔`BComplex`) crashed PANCCADIA after a ColdRoomPan reload — `SEVERE [sys] Cannot load station`, a full outage, not a warning. NEVER deploy a LOSSY/OUTAGE change to a station holding saved data without a bog migration. `[ev: corpus B800 §800.8]` `[ev: corpus B795]`
- Sign (auto), stop station, replace jars in `<niagara_home>/modules/`, start. Place components at their fixed ORDs; link points. Open the URL.
- Which profile jars a host needs and which Software Manager install/upgrade steps stop the station or reboot the host: `types/platform.md` § 1-2.
- **Pre-deploy gates for a PRODUCTION station — all MANDATORY, before the first jar is loaded:**
  1. **Boot smoke on the TARGET distribution and version.** The jars boot cleanly on a station of the same vendor distribution (OEM overlay or stock) and the same Niagara release as the production station — a matching test station, or a container/snap image of that release. Build, verify and schema-risk gates run against the build `niagara_home`; they cannot see a vendor/version mismatch (any static-init exception, not only a bad unit id), so a smoke on a different overlay or release does not satisfy this gate. `[ev: retro panccadia-commissioning-lessons Δ10]`
  2. **Full station backup before any `-rt` install.** Take a platform/station backup (a Backup `.dist`: station `.bog` + histories + alarms + module refs + platform config — `types/distribution.md` § 12), not only the module jars that `ng-deploy.sh` copies; write where it is stored and the restore path into the deploy notes. A module-jar backup protects code, not the operator's persisted state. `[ev: retro station-backup-before-deploy Δ1]`
  3. **Revert build ready** (§4.c) and **every shipped jar archived under a VERSIONED file name outside `modules/`** (e.g. `deploy/<Module>-rt-<version>.jar`) with its sha256. The station never needs an old file name back, but the team must know exactly which bytes each version was, to rebuild a revert reproducibly and to match a crash to the bytes that caused it — the build emits the same jar file name for every version (the version lives only in `META-INF/module.xml`). `[ev: retro roll-forward-recovery Δ3]`
  4. **The deploy checklist names, explicitly:** (a) for every opt-in (default-off) feature in the release, the exact enabling slot and its CURRENT live value — otherwise a feature ships inert and nobody notices `[ev: retro comppan-auto-lock-indicator Δ5]`; (b) every `BComponent`-only flag lifecycle the change touches (restart gates, lockout and re-entrancy flags on adapter classes), BY NAME, as a station-smoke item — a `BComponent` cannot be instantiated in WSL JUnit, so these flags are proven only on a live station after the restart (§6.a), never left as "should be tested" in a feature doc's prose `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ10]`.
  5. **Observed build→deploy handoff when the build and the deploy side are different checkouts** (e.g. a Linux-filesystem build clone and a Windows-side working tree the deploy machine reads): push the branch from the build clone to the deploy-side origin, switch the deploy-side tree to it (clean apart from known untracked archives), then re-verify `defaultModuleVersion` in each touched group and one new UI string there. What the deploy machine receives is then an observed fact, not an assumption. `[ev: retro comppan-auto-lock-indicator Δ7]`
- **Recovery from a bad deploy is ROLL-FORWARD, never a downgrade.** A version that crashes or degrades the station is replaced by a HIGHER `vendorVersion` — a fix, or the last-known-good source rebuilt under a new version (the §4.c revert build) — never by reinstalling the older version. Software Manager can only "Downgrade to <version>" when that older version is still in the local software database, and same-named jars erase it when the new jar is copied over the old one; an up/downgrade also forces a station stop + host reboot, so it saves nothing over a roll-forward. This supersedes the older "rollback = redeploy the previous jar + restart" wording of the 2026-09-03 slot-type-change retro (pointer recorded on its `retros/INDEX.md` row). `[ev: retro roll-forward-recovery Δ1]` `[ev: retro roll-forward-recovery Δ4]`
- **Remote access for a deploy or commissioning session has preconditions — check all before connecting:** explicit user authorization for the destination, the operation and the credential/session to use (availability of a tunnel or a session is not permission); the local port map of the tunnel written down (which local port reaches which remote service); the LOCAL station stopped when its ports collide with the tunnel's; no credentials, keys or session files copied into any repository. `[ev: retro kit-meta-hygiene-2026-10-01 Δ5]`
- **A `-ux`-only change needs NO station restart — browser hard-reload only; `-rt`/`-wb` needs a restart:** the servlet serves `rc/` from the classloader per request, so a new `-ux` jar is picked up on reload, but Java classes need a restart. Batch rt changes; iterating UI in production is cheap. [ev: retro ux-only-deploy-no-station-restart · D1]
- **The production station may run on a DIFFERENT device (e.g. an ATLAS snap) at a different NRE than the build PC — verify what runs via LIVE slots (oBIX / Slot Sheet), not the PC's `modules/` jar:** identify the station's real `niagara_home` from its boot log first; a PC `modules/` jar is only what Workbench can push. Device-side deploy = push to the device (Software Manager / Provisioning) + restart the device's station. [ev: retro station-corre-en-atlas-snap-no-en-el-pc-de-deploy · D2]
- **After a panel redeploy, power-cycle the HMI (the WebView caches the old page) before suspecting code:** a "blank after redeploy" is usually stale WebView state — disconnect/reconnect the panel first. [ev: retro dashboardpan-detail-render-doors · D3]
- **Before running `niagara-tools/scripts/ng-deploy.sh`, `cd` to the gradle root** (where `gradlew` / `settings.gradle` live — e.g. a `<Group>/` directory such as `Dashboard/`): it runs gradlew from the CWD, not from `GRADLEW_PATH`'s dir. Its sibling `niagara-tools/scripts/install-skill.sh` installs the kit skill; both live in the `niagara-tools` repository's `scripts/`, not in this kit's `toolbelt/`. [ev: retro ng-deploy-type-count-and-cwd · B9] `[ev: retro kit-meta-hygiene-2026-10-01 Δ4]`
- **`ng-deploy.sh` backs up LIGHTWEIGHT by default (only this module's own `-rt/-ux/-wb` jars) and auto-purges to keep-N (default 3, `--keep N`):** the old default tarred the WHOLE modules dir (~240 MB/deploy) and never purged (8 deploys → ~1.9 GB). `--no-backup` skips it (prints a git-rollback WARN; the script does not gate it — on a live station use it only when the pre-deploy full station backup above exists `[ev: retro station-backup-before-deploy Δ1]`), `--full-backup` / `FULL_BACKUP=1` restores the whole-dir backup. [ev: retro ng-deploy-backup-liviano-y-autopurga · B10] (as of niagara-tools 0.12.0)
- **`ng-deploy.sh`'s `verify_jar` counts `<type ` (WITH a space), so the `<types>` wrapper is NOT counted → `EXPECTED_*_TYPES` = the real type count, same as `verify-module.sh`, no `+1`:** real counts are ColdRoomPan-rt 6, DashboardPan-rt 2, DashboardPan-ux 1. [ev: retro ng-deploy-type-count-and-cwd · B8] (fixed in niagara-tools 0.12.0; before, it counted `<type` and you had to set real-types + 1)

- **Before and after a live-station deploy — snapshot the audit surface first:** `toolbelt/station-snapshot.sh <station-dir> <out-dir>` copies `config.bog` + `console*.txt` into `<out-dir>`, records history/alarm db pointers (paths + sizes, never the db files), and writes `manifest.json` with sha256 per file; source dir is never opened for write; exit 0 ok / 1 copy failure / 3 usage. Keep the pre-deploy snapshot as a baseline for `schema-risk.sh` and `bog-audit.sh` after the deploy. [ev: retro campaign8-station-snapshot]
- **After a reload (rt or full station restart), triage the console before closing the session:** `toolbelt/triage-console.sh --package com.vendor <station-dir>/console*.txt` surfaces own-module exceptions and load-time failures that the framework swallows silently (three attribution channels: own frame, own logger tag, [sys]/[sys.xml] load-fail shape). exit 1 = rows found; investigate before calling the deploy clean. [ev: retro campaign8-triage-console]
- **Audit the station bog for ghost slots, dangling links, orphan handles, proxy-link safety, and station-logic wiring:** `toolbelt/bog-audit.sh <config.bog|file.xml> --module <MOD> [--source-dir <src-dir>] [--strict]` (CHECK1-CHECK20; exit 1 = any FAIL). Runs from the bog alone for CHECK1/8/9/10/11/12/13/14/15/16/17/18/19/20; add `--source-dir` for the source-coupled checks (CHECK2-7), pointed at ONE checkout's module sources: over a client folder every `B<Class>.java` copy (live, stale, salvage) merges into one class, last write wins `[ev: retro go-bog-audit Δ6]`. Proxy-link-safety (CHECK11) fires when an own-module output is linked to a BooleanWritable/NumericWritable with no explicit fallback — the writable holds last state on station stop/reload, masking the fault. Station-logic checks (CHECK13-19): relay-double-source, own-output-unlinked, sensor-crossed-by-name, hasDefrost<->DefrostController sibling, roomN-index-mismatch, tile-number consistency, link-direction. Numeric-to-boolean-direct (CHECK20): a multi-state/enum slot (name contains Mode/Command/Cmd) linked directly into a boolean gate (BOrLogic/BAndLogic/BNotLogic) with no Equal(x,N) intermediate — the `!=0`-as-true coercion silently merges HAND and OFF states; add an `Equal(slot, value)` block between the numeric source and the gate. [ev: retro campaign8-bog-audit] [ev: retro campaign8-station-logic] [ev: retro control-model-limits-live-commissioning Δ4]

### 6.a Post-deploy verification (after hot module reload or station restart)

Ordered steps — run within ≤5 min of a hot module reload (Out-of-date: Module changed). `[ev: corpus B811]`

1. `toolbelt/station-snapshot.sh <station-dir> <out-dir>` — snapshot the station before the deploy; keep `<out-dir>` as the deploy baseline (`schema-risk.sh <out-dir> <post-deploy-snapshot>` compares before vs. after). `[ev: corpus B811]`
2. `toolbelt/triage-console.sh --package <com.vendor> <station-dir>/console*.txt` — scan for own-module load failures ("Cannot load station", "Missing frozen property", "ClassCastException", "Missing class for \"<own-prefix>:\""); exit 1 = rows found, investigate before calling the deploy clean. `[ev: corpus B800]`
3. `toolbelt/bog-audit.sh <config.bog|file.xml> --module <MOD>` — ghost slots, dangling links, orphan handles, proxy-link safety (CHECK11); exit 1 = any FAIL. `[ev: corpus B795]`
4. `toolbelt/report-module.sh <module-root> --console-dir <console-dir>` — aggregated punch-list; exit 1 = FAILs block hand-off. `[ev: retro campaign7-report-module]`
- The proxy-link safety row (CHECK11) must be clean before operator hand-off. `[ev: corpus B810]`
- **Attribute before fixing:** for any RECURRING console or field fault, record a verdict — own module / site device / platform / unknown — with its evidence (console rows, oBIX reads, device addresses) BEFORE changing code. An unknown cause stays `unknown` until evidence moves it; a site-device or platform fault is never "fixed" with a module change. `triage-console.sh --site` separates the repeating NON-own shapes (device offline, duplicate device id, comm timeout, history flood) into a `SITE` list with counts and first/last seen, so a known site fault is recorded once instead of being re-diagnosed every session; `SITE` rows never change the exit code. `[ev: retro site-fault-triage-and-incident-journal Δ2]` `[ev: retro site-fault-triage-and-incident-journal Δ1]`
- **Scenario-based live check per behavior decision:** for each row of the feature doc's behavior-decisions table, name the live scenario that exercises it (e.g. "nightly defrost window: zones with every valve closed do not call the plant") and read it read-only over oBIX WHEN that scenario occurs — an arbitrary-time read after the deploy does not exercise a window-specific behavior. Record the scenario, the read time and the observed values in the feature doc. `[ev: retro behavior-decisions-ask-dont-assume Δ5]`

## 6.b Commissioning-verify requirement (modules needing station rewiring) `[ev: retro live-commissioning-verification-gaps Δ7]`

The kit verify gate (§5) is **code-level and blind to live commissioning.** It confirms: bytecode 52, signed jars, types resolve, lint rules pass, pure JUnit green. It does NOT verify: facade↔rt link wiring, config values, per-instance control/status parity, or whether the station actually controls the plant correctly.

**A module that requires station rewiring is NOT "done" until a commissioning-verify pass confirms the following:**
1. Every facade display slot is linked from its DECLARED control point — not merely from some valid source: `toolbelt/obix-link-audit.sh --map docs/wiring-map.md --obix <base> --component <facade-path>` (read-only GET; credentials from `OBIX_USER`/`OBIX_PASS`; `--xml <saved-response>` offline) diffs the wiring map's source column (slot and, when the map names one, the source component instance) against the component's live links → one `MATCH`/`MISMATCH`/`MISSING`/`SKIP` row per slot, exit 1 on any MISMATCH/MISSING. A link from a wrong-but-valid source (a status slot fed by a raw relay command instead of the proven running flag) is structurally fine, so `bog-audit.sh` CHECK7/9/11 cannot see it. `--table 1` audits the facade → control rows against the control component. `[ev: retro panccadia-commissioning-lessons Δ4]`
2. Every facade config/setpoint slot is linked TO the corresponding control slot (not to a display slot or a link-target side).
3. Config values make physical sense: `interval > duration`, setpoint ≠ 0 on a cooling unit, `hasDefrost=true` AND `airDefrost=true` on an air-defrost unit, `duration` above a floor (e.g. ≥ 60 s).
4. Per-instance control/config surfaces have matching per-instance status slots (N configs → N status slots, not 1 scalar).
5. After a hot reload: `triage-console.sh` clean + `bog-audit.sh` CHECK11 clean.
6. **Persisted operator state survives the first restart:** HOA modes, setpoints, run hours and any new backup/persistence slot are SEEDED from the live values (or saved) BEFORE the first restart after the feature ships, and re-read after it. A brand-new backup starts empty on its first boot by construction — the first restart after a new hours backup reset a wear counter to ~0 — so either seed it or record the one-time reset as accepted in the feature doc. Mechanism: `types/logic.md` § Restart & persistence. `[ev: retro panccadia-commissioning-lessons Δ5]` `[ev: retro station-backup-before-deploy Δ3]`
7. **Alarm routing is proven end-to-end:** every alarm source has an alarm class and at least one recipient, and a test alarm per class reaches the console and is acknowledged there. Which faults are console alarms and which are flags: `types/logic.md` § Flag vs console alarm. `[ev: retro alarm-console-design Δ2]`
   Size the store at the same time: the file alarm DB keeps 500 records by default and overwrites the oldest
   (4.15.6.20 guide `Alarms/Alarm-FileAlarmDbConfig-92D9119A.txt:10` "1–250,000 records (defaults to 500)"); every module history ext is enabled, named by
   script and sized (`types/logic-authoring.md` § Logging a point to history). `[ev: retro devguide415-r2-control-alarms-schedules Δ4]`
8. **Consumer impact notified:** every point whose MEANING changed (proof threshold, predicate, mode semantics, default, reset by a first boot — see `METHODOLOGY.md` § Schema / upgrade safety, Consumer impact) was notified to its downstream consumers (web viewers, oBIX/cloud readers, other stations), with the date it takes effect. `schema-risk.sh` SAFE proves decode survival, not semantic stability. `[ev: retro panccadia-commissioning-lessons Δ8]`
9. **Values owed by the field — a table that stays open until filled:** every slot shipped at a "disabled / no value yet" default because the field has not supplied the real value (a nameplate current limit, a low-pressure cutout) gets a row in `docs/values-owed.md` (`| Slot | Owed by | Unit | Safe default | Status |`; Status `owed` until `filled <date>`/`provided`). `commissioning-verify.sh` prints a `MANUAL values-owed` row per still-owed slot, so the debt cannot age out when the feature doc is archived. Where the client keeps a version ledger, its `Open items` appendix aggregates these rows across features (together with pending manual links and open behavior assumptions) — cross-reference the module table, do not restate it. `[ev: retro panccadia-commissioning-lessons Δ11]` `[ev: retro panccadia-version-defect-ledger Δ4]`

**The kit — not the operator — owns the watchdog role.** Run `toolbelt/commissioning-verify.sh <module-root> [--bog <config.bog>] [--module <MOD>] [--values-owed <file>] [--wiring-map <file>] [--strict]` to close this gap: it orchestrates the Wave 3 lints (lint-config-sanity.sh CS1-CS4, lint-status-parity.sh, lint-recovery-path.sh, lint-link-target-flags.sh with the wiring map — default `<module-root>/docs/wiring-map.md` — so its Table 2 link-in targets are checked `[ev: retro panccadia-commissioning-lessons Δ1]`) over every `-rt/src` dir and folds bog-audit.sh CHECK11 (proxy-link-safety) + CHECK13-19 (station-logic) + CHECK20 (numeric-to-bool-direct) into a single §6.b punch-list. Rows: `STATUS  commissioning  <check>  <detail>`; exit 0 clean / 1 any FAIL / 3 usage or environment. Without `--bog`, the station checks emit SKIP rows with a note that a live config.bog is required. The script is the thin wrapper over `go/cmd/commissioning-verify` (bog-audit runs in process) and its verdicts fail closed: a bog-audit exit 3 is a SKIP row that names an unreadable or malformed bog or no component of the module, never a missing interpreter `[ev: retro go-commissioning-verify Δ1]`; the station fold relays a fixed status set, FAIL, WARN and PASS, for every folded check, so the WARN-only CHECK15 sensor-crossed and CHECK19 link-direction rows reach the punch-list (a fold that lists statuses per check drops whatever status a check later starts emitting) `[ev: retro go-commissioning-verify Δ2]`; an orchestrated lint that exits non-zero with no row (126, 127, a crash) is a FAIL, never a PASS `[ev: retro go-commissioning-verify Δ3]`; and a values-owed table that exists but cannot be read is exit 3 before any row `[ev: retro go-commissioning-verify Δ4]`. A `MANUAL` footer lists the steps this tool cannot statically verify (hot-reload console, plant control, per-instance runtime values, persisted state across the first restart, alarm routing, consumer impact, the `obix-link-audit.sh` link-source audit); the values-owed table (default `<module-root>/docs/values-owed.md`) adds one `MANUAL` row per owed slot, or one asking for the table when none exists. Run `toolbelt/generate-wiring-map.sh <facade-src-dir>` first to scaffold `docs/wiring-map.md`. `[ev: retro commissioning-verify Δ1]` Usage and exit contracts of the lints it orchestrates are rows of the generated lint index (`toolbelt/kit-state.sh lint-index`); a lint added to `commissioning-verify.sh` is documented by its own header, not by extending this paragraph. `[ev: retro kit-meta-hygiene-2026-10-01 Δ3]`

## 7. Retro + close (HARD close gate — not optional)
- **Lead merge/settle order:** merge ff-only → verify `git log -1` equals the blessed tip → THEN settle the ledger; a ledger settle on a reported-but-unverified merge records the wrong evidence revision. For parallel workers: rebase onto the new main tip before the QA ping so the blessed tip is the one that merges, not the pre-rebase base. `[ev: retro campaign8-close-process-meta-lessons]`
- **Every run ends by writing its retro** — run `toolbelt/new-retro.sh <module|kit> <slug>` and fill the stub (§1); a defect in a KIT CHECK or DOCTRINE additionally opens `toolbelt/kit-ticket.sh "<one line>"`. The retro is a PRECONDITION for "done", not an at-STOP afterthought — `toolbelt/sweep-build-state.sh --age` at orient (BUILD-LOOP §0.a) surfaces the accrued retro debt so it cannot be skipped across a continuous chain. [ev: retro campaign8-retro-loop]
- **Assumption register at close:** every behavior still decided without an explicit user answer (deferred by the
  user, or discovered late) is listed in the feature doc under `## Assumptions still open` and repeated in the close
  message as "Assumptions I made — confirm or change". An empty register is stated explicitly ("none"), never
  omitted. `[ev: retro behavior-decisions-ask-dont-assume Δ4]`
- **Version ledger (client):** a feature doc does not close until it has appended its version row(s) — module, version, date, status, what changed, and any defect with its root cause, class and prevention — to the client's `odd/VERSION-LEDGER.md` (scaffold `templates/VERSION-LEDGER.md`), written at the build/deploy hand-off together with the module's deployed-baseline fields in `BUILD-STATE.src.md`, never reconstructed later from `git log`. `[ev: retro panccadia-version-defect-ledger Δ1]`
- **Repeated root-cause class = escalation:** when the ledger already holds a live hit of the class this session hit again on the same client, and that class's prevention delta has not shipped, the delta is PRIORITIZED — apply it in the current work or open it as the next work unit, ahead of its normal promotion queue; re-citing it as evidence is not enough. `[ev: retro panccadia-version-defect-ledger Δ3]`
- **Operator manual in lockstep:** any operator-visible label, default, unit or behavior change updates the operator manual — its source and its rendered PDF — in the SAME commit, and the commit trailer or the retro states `manual: updated` or `manual: n/a (no operator-visible change)`. Before closing, run `toolbelt/lint-manual-labels.sh <manual.html|md> <ui-dir>...` (advisory: every quoted UI label of the manual must exist in the UI source; WARN-only, `--strict` promotes) and read each WARN. `[ev: retro operator-manual-lockstep Δ1]` `[ev: retro operator-manual-lockstep Δ3]`
- **Edit `BUILD-STATE.src.md`** (the committed source of the generated `BUILD-STATE.md`) for the module: refresh the `build-state.v1` envelope (`last_build`, `verify_gate`, `deployed`, `bytecode_major`, `signed`, `last_commit`, `last_session`, `open_issues`), set `retro_required` honestly, and set `retro_pending`.
- **Kit-infrastructure work** (changing the kit itself — toolbelt, type guides, methodology — not building a module) has no module build to record: update the `kit` self-section instead (a `ledger.d/` fragment, plus `BUILD-STATE.src.md` for a hand-kept kit field), under the same close gate.
  - **The kit ledger is GENERATED from fragments** — never hand-edit the kit block's `last_change:` line, its `open_issues:` items or its session log (the lines after `last_session:`). Add ONE file `ledger.d/<YYYY-MM-DD>-<slug>.md` per change (`<!-- ledger: kit -->`, `<!-- date: YYYY-MM-DD -->`, optional `<!-- last-change-note: … -->`, exactly one `log: YYYY-MM-DD · <text>` line, zero or more `open_issue: <text>` lines) and commit the fragment ALONE: `BUILD-STATE.md` is generated (`toolbelt/kit-state.sh build-state` prints it) and `.gitignore`d, so two PRs that each add a fragment never conflict. `toolbelt/gen-build-state.sh --check` (CI, pre-push) fails closed on a malformed fragment or on a generated line pasted into `BUILD-STATE.src.md`. The fragment IS the ledger update of every exit below. The pre-fragment ledger is `ledger.d/0000-archive-pre-2026-10-23.md`; edit an old entry there. Module blocks and the other kit fields (`version`, `retro_*`, `last_commit`) stay hand-kept in `BUILD-STATE.src.md`. Regions are found by structure, so a non-log line after `last_session:` is an error. `[ev: retro ledger-fragments Δ1]` `[ev: retro ledger-fragments Δ2]` `[ev: retro ledger-generated-not-committed]` A kit doc may name the generated files: `tests/kit-links.bats` L1 resolves each one through its committed source, so a fresh clone with no hooks and nothing materialized passes. `[ev: retro ledger-generated-not-committed Δ1]` Module envelopes stay hand-kept, so two PRs that edit the SAME module envelope in `BUILD-STATE.src.md` still conflict (real content); when module sessions start colliding there, give module envelopes per-module fragments as `ledger.d/` did for the kit block. `[deferred: retro ledger-generated-not-committed Δ2]` The kit envelope's `last_session:` line is a placeholder for the log below it (the last log line is the last session); dropping the field waits for a decision and a `go/internal/buildstate` change. `[deferred: retro ledger-generated-not-committed Δ4]`
  - **A kit behaviour change carries its CHANGELOG entry in the same range:** a `changelog.d/<YYYY-MM-DD>-<slug>.md`
    fragment (preferred), a non-blank line inside `## [Unreleased]` of `CHANGELOG.md`, or a `Changelog: none
    (<reason>)` trailer. `scripts/changelog-gate.sh` refuses a range that changes `go/`, `build-n4-module-kit/`,
    `tools/` or `scripts/` (minus `go/bin/`, `BUILD-STATE(.src).md`, `ledger.d/` and `retros/`) without one, in
    `.githooks/pre-push` and in CI, so release notes are never rebuilt from the merge log (`CONTRIBUTING.md` § 6.2). `[ev: retro r10-changelog-gate Δ1]`
  - **Before every kit PR run `scripts/ci-local.sh -j 3`** from the niagara-tools root: it replays
    `.github/workflows/ci.yml` step by step with the pinned shellcheck 0.10.0; a local shellcheck 0.11 passed the
    SC2015 that CI failed in PR #273 (`CONTRIBUTING.md` § 6). `[ev: retro r8-t22-retro-deltas Δ4]`
    Its shellcheck step covers `.githooks/*` with `scripts/`, `toolbelt/` and `tests/`, so a hook is never checked by
    hand. `[ev: retro ledger-generated-not-committed Δ3]`
    Its bats step installs the pinned npm tools first (`toolbelt/eslint/package.json`) and ends with
    **`scripts/check-bats-skips.sh <bats.tap> tests/bats-skip-allowlist.txt`**: a test that skips for a reason no
    allowlist line matches fails the run (exit 1), so a lost tool install or a guard that now always skips cannot
    read green. A new legitimate CI skip gets its own commented allowlist line in the same PR (audit A6).
  - **Shared host:** when other sessions run on the same machine, cap test parallelism at `-j 3` (at load 28 a
    wider run made `verify-module.bats` flake). GNU `parallel` children inherit an open `flock` fd and hold the
    lock after the parent exits: start a locked parallel run with `setsid` and stop it with `kill -- -<PGID>`.
    `[ev: retro r8-t22-retro-deltas Δ7]` bats 1.14 has no `--keep-order` (a GNU parallel flag): `bats -j 4
    --keep-order` printed zero oks, so count what ran (`ORCHESTRATION.md` § Count what ran). `[ev: retro r9-m1-walkers Δ3]`
    - **Host pids budget:** on WSL every session shares the `init.scope` cgroup (`pids.max` = systemd
      TasksMax, 15% of `threads-max`; 48,129 here). When it fills, every fork fails and all sessions abort
      together. Before a heavy suite (`bats`, `make check`, a mutation run) read
      `/sys/fs/cgroup/init.scope/pids.current` and wait while it is above 20,000; run under `ulimit -u 4000`.
      `scripts/ci-local.sh` refuses to start (exit 4) while any cgroup is above 50% of its `pids.max`.
      `[ev: retro r8-t41-host-safety Δ1]`
    - **One read:** `toolbelt/host-budget.sh` (wrapper of `go/cmd/host-budget`) reports pids, threads, GOCACHE
      and `/`, `/mnt/c` in one run (exit 1 on a FAIL row); use it for the pids and disk reads here.
      `[ev: retro r8-t47-host-budget Δ1]`
    - **Writer cap:** at most two writers per session run a heavy suite at the same time; the others edit,
      read or wait. The integrator runs the full `ci-local.sh -j 3` once, at the end. `[ev: retro r8-t41-host-safety Δ2]`
    - **Disk at round close:** Go never caps `~/.cache/go-build` (84 GB after five days of multi-worktree
      builds). At round close run `go clean -cache` and `df -h / /mnt/c`; stop heavy work while C: is above
      90%, because WSL then fails to start (`Wsl/Service/E_UNEXPECTED`). Operator hygiene, outside the
      agent's reach: delete orphan `swap.vhdx` files and crash dumps, and compact `ext4.vhdx` (`fstrim -av` in
      WSL, then `wsl --shutdown` and diskpart `compact vdisk`); the vhdx never shrinks on its own.
      `[ev: retro r8-t41-host-safety Δ3]` `[ev: retro r8-t41-host-safety Δ7]`
  - **Landing scripts:** poll `gh pr checks <n>` in a bounded loop instead of `gh pr checks --watch` (it hung
    15 minutes on a PR whose only check had passed). Resolve every path argument to an absolute path before
    any `cd` (a relative `--body-file` failed after `cd` into the worktree). Keep helper scripts in the
    repository or the session scratchpad, never in `/tmp`, which a WSL restart wipes. Toolbelt entry points
    resolve their own real path (#315, `fix/r8-wrapper-real-path`), so call them by path or through a symlink.
    `[ev: retro r8-t41-host-safety Δ4]` `[ev: retro r8-t41-host-safety Δ5]` `[ev: retro r8-t41-host-safety Δ6]`
- A session that changed KIT files is NOT "done" until ONE of:
  - (a) it wrote a retro at `retros/<date>-<module>.md` (line 1 `<!-- review-status: pending -->`, lessons as PROPOSED kit deltas — propose-never-apply), its header is its retro-index row, and set `retro_pending: false` in `BUILD-STATE.src.md`; OR
  - **The retro index is GENERATED** — the generated `retros/INDEX.md` is never hand-edited. Each retro carries its row as header comments (`<!-- review-status: pending|folded -->` on line 1, then `<!-- module: X -->`, `<!-- deltas: N -->`, optional `<!-- index-comment: … -->`); edit the retro header; `INDEX.md` is generated (`toolbelt/kit-state.sh retro-index` prints it) and `.gitignore`d, so it is never in a commit and never conflicts. `toolbelt/gen-retro-index.sh --check` (CI, pre-push) fails closed on a malformed header. `toolbelt/new-retro.sh` writes that header block into its stub and regenerates the local index (it never appends a row). `[ev: retro generated-retro-index Δ1]` `[ev: retro generated-retro-index Δ2]` `[ev: retro ledger-generated-not-committed]`
  - (b) it declared the change TRIVIAL: `Retro: none (trivial: <reason>)` in the commit trailer AND `retro_required: false` in the envelope. A push that only records evidence in a feature doc (`odd/tasks/*.md`) is this case: `.githooks/pre-push` counts every kit file except `BUILD-STATE.src.md`, `ledger.d/**` and `retros/**`, plus `scripts/**` and `go/**` (not `go/bin/`), as build-relevant, so it carries `Retro: none (trivial: feature-doc evidence only)`. `Retro: promotion` does not fit it — with no in-range header flip or ledger anchor the gate rejects that trailer. `[ev: retro retro-fold-campaign-close Δ3]`; OR
  - (c) it is a PROMOTION of already-filed lessons into the core: `Retro: promotion (folds <ids> from existing retros)` in the commit trailer AND a STRUCTURAL ANCHOR — either an in-range retro header flip `review-status: pending` → `folded` (a FULL promotion; its generated INDEX row follows) OR an in-range `BUILD-STATE.src.md` or `ledger.d/` fragment change (a PARTIAL promotion folds content while its source retro stays `pending` for owed halves, so it flips no header — it stamps the owed `open_issue:` line in its fragment instead). The trailer ALONE is not a blanket escape; with NEITHER anchor it fails. Either anchored path still runs `sweep` for ledger coherence. A promotion PR folds existing retros, so it owes no NEW retro.
- **Envelope-pairing rule (all exits):** every close-exit — (a) new retro, (b) trivial trailer, (c) promotion — MUST pair its retro anchor with a kit ledger update (a `ledger.d/` fragment or a `BUILD-STATE.src.md` edit) in the SAME push range; a branch push is not proof — the hook evaluates the whole PR on `main`. [ev: retro campaign6-doctrine-fold] [ev: retro campaign6-types-fold] [ev: retro campaign6-close-process-meta-lessons] [ev: retro campaign7-retro-fold]
- The **Output Contract MUST print** `retro: <path> (N deltas, review-status: pending)` — or `retro: none (trivial: <reason>)` — as an explicit line. A written-but-invisible retro reads as a missing one.
- Do NOT silently rewrite METHODOLOGY — propose; a human folds it in. This is how the kit matures from seed to solid.
- **Sweep at close** over fresh `toolbelt/kit-state.sh build-state` / `retro-index` output written to a temp dir `$T` (the files are not committed; CI does the same): `toolbelt/sweep-build-state.sh "$T/BUILD-STATE.md" <retros-dir> "$T/INDEX.md"` (envelope content check); `toolbelt/sweep-fold-audit.sh --strict --deltas-since 2026-09-24 "$T/INDEX.md" <kit-root>` (fold-citation audit; from the Δ-citation convention on, every Δ1..Δn of a folded retro is cited as `[ev: retro <stem> Δk]` or parked as `[deferred: retro <stem> Δk]` — the committed form of the per-retro reconcile check).
- **Installed-skill drift (close gate):** run **`scripts/check-skill-drift.sh`** (read-only; `--skill <name>`, `--home <dir>`). A FAIL means the launcher an agent actually follows (`~/.claude/skills/<skill>/SKILL.md`) differs from the tracked `skill/SKILL.md` — the session is not closed until `scripts/install-skill.sh --force` restores parity. It is a session-close check, not a CI step (CI has no installed skill). `[ev: retro client-source-of-truth Δ3]`
- **Source of truth moved?** If the authoritative checkout, branch or commit of a module changed this session, update its `source_of_truth` field in `BUILD-STATE.src.md` (its module envelope) in the same close — never only in session memory. `[ev: retro client-source-of-truth Δ2]`
- Machine enforcement (opt-in, per-clone, reversible): activate with **`scripts/install-hooks.sh`** (sets `git config core.hooksPath .githooks`; `--uninstall` restores the default; it REFUSES to clobber a pre-existing custom `hooksPath` unless `--force`). Once active, `.githooks/pre-push` blocks a build-relevant push that skips the close gate, delegating the ledger check to `toolbelt/sweep-build-state.sh`.
  - **A local-only gate skips only on a named, matched absence.** A check CI cannot run (its inputs live only on a
    developer machine) runs in the local hook, before every early exit. Its one skip matches the tool's exact
    message for an absent input and prints a named stderr line; every other non-zero exit refuses, the same exit
    code for another cause included. The pre-push live cite check (`kit-cite-resolve --strict` on
    `scripts/lib/live-doctrine-set.sh`) skips only on `cannot read <research|modules|clients|memory> root …: no such
    file or directory`. `[ev: retro r10-live-cite-prepush-gate Δ1]`
