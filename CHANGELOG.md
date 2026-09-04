# Changelog

All notable changes to **niagara-tools** (cross-project Niagara N4 tooling) are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning follows [SemVer](https://semver.org/spec/v2.0.0.html) (currently `v0.x` — see CONTRIBUTING.md §Versioning policy for stability promises). Each entry references the SDD change slug and engram observation IDs for traceability.

---

## [v0.8.1] - 2026-09-04

### Changed — promote proven retro lessons into the kit core (PR3)

- **Lesson folds (documentation):** promoted the highest-value [CERT]-grade lessons trapped in `retros/`
  into the living kit files —
  - `METHODOLOGY.md`: never retype a live slot / bog boot crash (S1) + NEW-safety-slot-defaults-SAFE
    carve-out (L15) under a new §Schema/upgrade safety; `0`-means-DISABLED on a protection limit (L2);
    `module.palette` one entry per `@NiagaraType` (B5); the timer `started()`+`atSteadyState()` pointer;
    the 4-layer assurance pointer.
  - `types/logic.md`: `Long.MIN_VALUE` sentinel guard (L1); the slot-default SAFE carve-out (L15); a
    §Pure-class extraction section (extract-test-then-wire, stateful `step()` adapter · T2/L6); cross-refs.
  - `types/dashboard.md`: a generic write endpoint must whitelist / check `Flags.OPERATOR` (U5).
  - `build-verify.md`: free-the-lock-first vs mirror-fallback (B2) + build/libs-still-signed (B1) +
    first-build-no-stop (B3); the `find ~/.gradle` + `run-pure-test.sh` recipe and niagaraTest-is-docs
    (B11); a §How you know it's good 4-layer assurance stack (T1/T3/T4).
- **Contradictions resolved to ONE rule each:** jar-lock (free the lock first; mirror only for a live
  production station) → `build-verify.md`; slot default (preserve current on a deployed path, but a NEW
  safety-gating slot defaults SAFE) → `METHODOLOGY.md` + `types/logic.md`.
- **`kit` self-section** added to `BUILD-STATE.md` (`module: kit`) so kit-infrastructure work has a row to
  update; `BUILD-LOOP.md` §7 notes it. Dogfood: this campaign filed TWO retros (the campaign retro + the
  run-pure-test `set -e` empty-cache lesson) with `retros/INDEX.md` rows, and marked 4 fully-folded retros
  `folded` — the mechanism recording its own evolution.

### References

- SDD change: `build-n4-module-continuity` (PR3 of 3; follows PR1 v0.6.0 ledger, PR2 v0.7.0 gate, v0.8.0
  run-pure-test). Documentation + schema-section only, no new script → PATCH.
- Engram: #8095 (campaign), #8101 (42-lesson mining), #8105 (tasks), #8111 (chain status).

---

## [v0.8.0] - 2026-09-04

### Added — one-command pure-test runner (`run-pure-test.sh`)

- `build-n4-module-kit/toolbelt/run-pure-test.sh` (NEW): runs a ZERO-Baja pure decision class +
  its JUnit test standalone in WSL, resolving junit-4.13.2 + hamcrest-1.3 from `~/.gradle` and
  compiling into a TEMP dir (never the module tree — a parallel session's working tree is
  off-limits). Removes the per-session jar hunt documented in the
  `2026-09-04-junit-standalone-cached-jar-locations` retro. Exit 0 pass / 1 test failed / 2 usage /
  3 env. Proven against the real CompPan `CompressorControlTest`: 31 tests OK in 0.015s.
- `tests/run-pure-test.bats` (NEW): 6 cases, each mutation-checked to bite — P2 (a failing pure
  test must exit non-zero, i.e. the runner does not swallow a JUnit failure) and P6 (an empty
  `~/.gradle` cache exits 3 with an actionable message) both caught real behaviour, the latter a
  `set -e`/pipefail bug in the runner itself.

---

## [v0.7.0] - 2026-09-04

### Added — retro-enforcement gate (sweep + pre-push hook + retro index)

- `build-n4-module-kit/toolbelt/sweep-build-state.sh` (NEW): content-only validator (VCS-free —
  kit-links L2) for BUILD-STATE.md + retros/INDEX.md. Exit 0 clean / 1 named integrity violation /
  3 usage. Checks column-0-anchored `build-state.v1` markers, required boolean fields
  (module/retro_required/retro_pending), a tolerated multi-line `open_issues` list, and INDEX
  integrity (every retro has a row, every row a real file, review-status ∈ {pending,folded}).
- `.githooks/pre-push` (NEW): opt-in gate (enable with `git config core.hooksPath .githooks`).
  Blocks a push that changes build-relevant kit files (`build-n4-module-kit/**` except BUILD-STATE.md
  and `retros/`, plus `scripts/**`) unless the range also carries a BUILD-STATE.md update + a pending
  retro + its INDEX row, or a `Retro: none (trivial: <reason>)` trailer. Delegates the content half
  to `sweep-build-state.sh` (the diff half owns the only VCS calls).
- `build-n4-module-kit/retros/INDEX.md` (NEW): the promotion registry — one row per retro (30
  seeded; 3 folded, 27 pending), with a `deltas` count column.
- `tests/build-retro-sync.bats` (NEW, RED-first): 16 cases (9 sweep + 7 hook), each
  mutation-checked to bite.
- `build-n4-module-kit/BUILD-LOOP.md` §7: parenthetical updated to point at the now-live gate.

### References

- SDD change: `build-n4-module-continuity` (PR2 of 3; follows PR1 v0.6.0 continuity ledger).
- Engram: #8095 (campaign), #8101 (recon), #8105 (tasks), #8107 (PR1 landed).

---

## [v0.6.0] - 2026-09-04

### Added — build-session continuity ledger (BUILD-STATE.md)

- `build-n4-module-kit/BUILD-STATE.md` (NEW): a "where did we leave off" registry for the build
  loop — one `build-state.v1` envelope per module (ColdRoomPan, CompPan, DashboardPan, chihuahua),
  seeded from evidence. The build-loop analog of research-sdd's FOCUSES + RESEARCH-STATE registries,
  collapsed into one file. Fields split GATED (`retro_required`, `retro_pending` — kit-local only)
  vs DECLARED (jar/module/src state, which lives in separate repos and is recorded, not gated).
- `build-n4-module-kit/BUILD-LOOP.md`: new `§0.a Orient from BUILD-STATE` (prints a one-line
  leave-off per module, with the meta-work exemption) and a reworded `§7` HARD close gate — update
  BUILD-STATE and write a retro, or declare `Retro: none (trivial: <reason>)`; print the `retro:`
  line in the Output Contract.
- Launcher `SKILL.md` (in `~/.claude/skills/build-n4-module/`, outside this repo): orient from
  BUILD-STATE at step 1, update it at step 6, print the retro line in the Output Contract.

### References

- SDD change: `build-n4-module-continuity` (PR1 of 3 — continuity registry; PR2 = retro-enforcement
  gate, PR3 = promote proven lessons). Tasks: `openspec/changes/build-n4-module-continuity/tasks.md`.
- Engram: #8095 (campaign kickoff), #8101 (recon: continuity design + 42-lesson mining), #8105 (tasks).

---

## [v0.5.0] - 2026-09-01

### Added — ng-deploy verify-module gate and `-ux` slotomatic

- `scripts/ng-deploy.sh`: runs the `build-n4-module-kit/toolbelt/verify-module.sh` gate on the
  built jars during the verify step, default ON for mode A/C. New `--no-gate` flag to skip it and
  `VERIFY_MODULE_BIN` env var to override the gate binary (used by the tests).
- `scripts/ng-deploy.sh`: runs `:MODULE-ux:slotomatic` (in addition to `:MODULE-rt:slotomatic`)
  under `--with-slotomatic` in mode A when the `-ux` profile source carries a
  `@Niagara(Type|Property|Action|Topic|Singleton)` annotation (presence-based scan).
- `tests/ng-deploy.bats`: 7 new cases — T27–T29 (`-ux` slotomatic) and T30–T33 (verify-module
  gate). Each was mutation-checked to fail when its target behavior is broken. 33 total.

### Changed — ng-deploy verify-module gate and `-ux` slotomatic

- `scripts/ng-deploy.sh`: exit 50 ("verify failed") now also covers a failed `verify-module.sh`
  gate, alongside the existing type-count mismatch and missing-BUILD_ID cases.
- `scripts/ng-deploy.sh`: mode-B `--with-slotomatic` warning enriched to point at
  `--mode A --with-slotomatic` or `toolbelt/build.sh` for regenerating `-ux` slots (behavior
  unchanged — mode B still warns and skips).

### References

- SDD context: build-n4-module-kit v0.2 retro §6 follow-ups P1 (ng-deploy gate) + P2 (`-ux` slotomatic).
- Engram: #7953 (this change).

---

## [v0.4.0] - 2026-09-01

### Added — `build-n4-module-kit-v0.2`

- `build-n4-module-kit`: fold-in of 41 proven lessons from 3 retros (ColdRoomPan rt-hardening,
  DashboardPan 5-rooms, DashboardPan HMI touch UX) and the concurrent session bitácora into
  7 kit files (`types/dashboard.md`, `types/logic.md`, `build-verify.md`, `METHODOLOGY.md`,
  `BUILD-LOOP.md`, `SOURCES.md`, `types/wb-widgets.md`). Evidence markers (`[CERT]`,
  `[CERT-live]`, `[INFER]`) preserved from source retros.
- `toolbelt/verify-module.sh`: THE verify gate — bytecode major 52, `NIAGARA4.SF` presence,
  `module.xml` type-to-class resolution; opt-in `--target-version`, `--stored`, `--src` checks.
  POSIX-only (no JDK). Exit codes 0/1/2/3.
- `toolbelt/build.sh`: rewrite — source-based profile selection (skips stub `-wb`), Java 8 +
  clean + slotomatic + jar, runs `verify-module.sh` on every produced jar. Exit codes 2/10/30/50.
- `toolbelt/mirror-niagara-home.sh`: safe writable mirror of a live Niagara install for builds
  against a running station (refuses real install or any non-mirror dir, exit 20).
- `toolbelt/stored-repack.sh`: STORED repackage for the Workbench re-sign path (B7 recipe);
  manifest-first ordering, `zip -0`, verifiable with `verify-module.sh --stored`.
- `tests/verify-module.bats`, `tests/build-sh.bats`, `tests/mirror-niagara-home.bats`,
  `tests/stored-repack.bats`, `tests/kit-links.bats`: 5 bats suites (27 tests; 53 total with ng-deploy.bats) with generated-jar
  fixtures helper `tests/helpers/n4-fixtures.bash`.

### Changed — `build-n4-module-kit-v0.2`

- Kit doctrine rewritten to three explicit roles: `verify-module.sh` (THE gate),
  `build.sh` (recommended WSL build, runs the gate), `ng-deploy.sh` (station deploy wrapper —
  backup → build → copy → type-count verify; slotomatic guard rt-only). Replaces the ambiguous
  primary/fallback framing in `build-verify.md` and `BUILD-LOOP.md`.
- `toolbelt/build.sh` exit codes: 2 usage error, 10 env/path error, 30 gradle failure,
  50 verify-module gate failure (supersedes previous informal codes).
- `CONTRIBUTING.md`: bats-core install step added to the test-runner prerequisites section.

### Fixed — `build-n4-module-kit-v0.2`

- Dangling kit links (`checklist-common.md` → `METHODOLOGY.md`;
  `type-dashboard.md` → `types/dashboard.md`) removed from kit files.
- `build.sh`: stub-profile selection now skips profiles with a `build.gradle` but no source files
  (was silently including them).
- `build.sh`: all-classes bytecode major check now correctly catches a later `.class` at major 65
  when the first `.class` in the jar is major 52 (false-pass fixed).

### Notes

- Launcher `~/.claude/skills/build-n4-module/SKILL.md` updated to v0.2 (three-role doctrine,
  verify-module.sh in Output Contract). This file lives outside the git repository; it is not
  included in this tag but is documented in the PR3 body and in engram.

### References

- SDD slug: `build-n4-module-kit-v0.2`
- Engram: explore #7937, proposal #7938, spec #7939, design #7940, tasks #7943.
- Tag: `v0.4.0`.

---

## [v0.3.0] - 2026-05-18

### Added — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh`: `--with-slotomatic` flag — invoca `:MODULE-rt:slotomatic` con los
  3 overrides `-P` ANTES de `build_jars` (modos A/C; ignorado en modo B con WARN).
- `scripts/ng-deploy.sh`: `--strict-slotomatic` flag — aborta con exit 15 si se detectan
  cambios de anotación sin `--with-slotomatic`. No implica `--with-slotomatic` (cero magia).
- `scripts/ng-deploy.sh`: `SLOTOMATIC_DETECTION` env var (`warn`|`strict`|`off`, default `warn`) —
  controla la heurística pasiva de detección de cambios en anotaciones `@Niagara*`.
- `scripts/ng-deploy.sh`: `detect_annotation_changes()` — heurística pasiva que corre entre
  backup y build. Lee `.last-deploy-sha` como baseline (fallback `HEAD~1`); filtra diff con
  `grep -E '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)'`.
- `scripts/ng-deploy.sh`: `run_slotomatic()` — invoca gradlew con los mismos 3 `-P` overrides
  que `build_jars`; `die 15` si gradlew retorna distinto de cero.
- `scripts/ng-deploy.sh`: `write_last_deploy_sha()` — escribe `git rev-parse HEAD` en
  `.last-deploy-sha` post-verify exitoso; silencioso si git falla.
- `scripts/ng-deploy.sh`: `read_baseline_sha()` — lee `.last-deploy-sha`, valida con
  `git cat-file -e`, fallback a `HEAD~1` si ausente/vacío/SHA inválido.
- `scripts/ng-deploy.sh`: `warn_slotomatic_recommended()` — heredoc multi-línea a stderr.
- `scripts/ng-deploy.sh`: exit code 15 nuevo — slotomatic falló O cambios de anotación
  detectados en modo strict. Ningún code path existente (0-50) fue modificado.
- `tests/ng-deploy.bats`: 9 tests nuevos (T18–T26), total 26. git fakebin en setup().
  Refactor gradlew stub: `gradlew.calls.log` (log acumulativo) + `gradlew.args` (backward-compat).
- `.env.local.example`: sección `SLOTOMATIC_DETECTION` + nota sobre `.last-deploy-sha` en `.gitignore`.
- `docs/knowledge-base/slotomatic.md`: Card 4 — ng-deploy.sh integration (flags, detection
  mechanics, decision table, false-positive edge case, `.last-deploy-sha` gitignore note).
- `docs/GOTCHAS.md`: fila anti-pattern "Deploy with stale slotomatic".
- `tests/smoke-checklist.md`: paso opcional `--with-slotomatic` en modo A; nota no-op en modo B.

### Changed — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh` `print_usage()`: documenta `--with-slotomatic`, `--strict-slotomatic`,
  `SLOTOMATIC_DETECTION` env var, y exit code 15.
- `CLAUDE.md` §1 tabla: fila "Slot/Property/Type/Action" actualizada →
  `A --with-slotomatic (or :slotomatic separately first)`.
- `scripts/ng-deploy.sh` header comment: añade `--with-slotomatic`, `--strict-slotomatic` a Usage.

### References

- SDD slug: `niagara-tools-slotomatic-integration`
- Engram: spec #1927, design #1928, tasks #1929.
- Tag: `v0.3.0` (pendiente de commit del operador).

---

## [v0.2.0] - 2026-05-18

### Added — `niagara-tools-versioning-and-contributing`

- `VERSION` file at repo root as the single source of truth for the version string.
- `CHANGELOG.md` (this file) in Keep a Changelog format, seeded with retroactive `[v0.1.0]`
  (bootstrap) and the current `[v0.2.0]` entries.
- `CONTRIBUTING.md` (solo+agents scope): release recipe, TDD gate, KB authoring rules,
  conventional commits, pre-commit checklist, versioning policy, content-boundary matrix,
  and known limitations.
- `scripts/ng-deploy.sh`: `--version` flag prints `SCRIPT_VERSION` and exits 0. Value is
  resolved CWD-agnostically via `BASH_SOURCE[0]`-relative read of `VERSION`
  (`cat "${SCRIPT_DIR}/../VERSION"`), with an `"unknown"` fallback when the file is missing.
- `tests/ng-deploy.bats`: 2 new tests (17 total) — `--version` exit/output contract and
  anti-drift regression guard for the path resolution.
- `CLAUDE.md` §9 "Release process": one-liner bump rule + pointer to `CONTRIBUTING.md`
  §Release process. (Decision tree only — no policy duplication; see proposal-decisions #1846.)
- `README.md`: "Versioning" section linking to this changelog.
- Git tags: `v0.1.0` retroactive on bootstrap commit `8d9a396`; `v0.2.0` on this commit.

### Changed — `niagara-tools-versioning-and-contributing`

- `scripts/ng-deploy.sh` header Usage comment: added `--version` next to `--help` in the
  metadata-flags Usage line.

### References

- SDD slug: `niagara-tools-versioning-and-contributing`
- Engram: explore #1843, proposal #1844, proposal-decisions #1846, spec #1848, design #1847.
- Tag: `v0.2.0` (this commit).

---

## [v0.1.0] - 2026-05-17

### Added — `niagara-tools-bootstrap` (retroactive entry)

This release documents the initial bootstrap of the `niagara-tools` repo. The actual commit
(`8d9a396`) shipped before any versioning infrastructure existed; this entry is the retroactive
`v0.1.0` record, tagged after the fact on the bootstrap commit.

- `scripts/ng-deploy.sh`: bash wrapper for the Niagara N4 module deploy cycle
  (backup → build → copy → verify). Flag API: `--mode A|B|C`, `--env-file PATH`,
  `--no-deploy`, `--no-backup --i-know-what-im-doing`, `--help`. Exit codes: 0 (success),
  10 (env/path), 20 (backup), 30 (build), 40 (copy), 50 (verify).
- `.env.local.example`: config schema with all 8 required env vars (`MODULE_NAME`,
  `GRADLEW_PATH`, `NIAGARA_HOME`, `NIAGARA_USER_HOME`, `JAVA_HOME`, `STATION_MODULES_DIR`,
  `EXPECTED_RT_TYPES`, `EXPECTED_UX_TYPES`) and optional `BUILD_ID` cache-buster.
- `tests/ng-deploy.bats`: 15 bats-core unit tests using PATH-injected fakebin stubs for
  `gradlew`, `unzip`, `tar`. No real station or build dependency.
- `tests/smoke-checklist.md`: manual integration checklist for modes A, B, C.
- `docs/GOTCHAS.md`: cross-project anti-patterns index linking the KB topic files.
- `docs/knowledge-base/`: four topic files seeded from chihuahua learnings —
  `bql-gotchas.md` (BQL N4.14 confirmed bugs + persistent-ack pattern),
  `wsl-build-gotchas.md` (WSL build overrides, gradlew path, slotomatic-in-WSL myth),
  `hot-reload-rules.md` (Java = station restart; JS/CSS = browser hard-reload),
  `slotomatic.md` (when to run, slot removal coordinated edit, AUTO region rules).
- `CLAUDE.md`: agent guide with deploy decision table, invariants
  (backup → build → copy → verify), onboarding, gitignore exception note, test runner setup,
  engram `topic_key` conventions, cross-project search hints, KB index.
- `README.md`: human-first quick-start with consumer-module integration pattern.
- `.gitignore`: `_backups/`, `.env.*` with explicit `!.env.local.example` exception.

Bootstrap commit: `8d9a396` (retroactively tagged `v0.1.0`).

### References

- SDD slug: `niagara-tools-bootstrap`
- Engram: init #1806, proposal #1811, delivery decision #1817, apply #1820, verify #1823,
  archive #1824.
