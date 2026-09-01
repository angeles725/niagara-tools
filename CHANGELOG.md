# Changelog

All notable changes to **niagara-tools** (cross-project Niagara N4 tooling) are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning follows [SemVer](https://semver.org/spec/v2.0.0.html) (currently `v0.x` — see CONTRIBUTING.md §Versioning policy for stability promises). Each entry references the SDD change slug and engram observation IDs for traceability.

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
