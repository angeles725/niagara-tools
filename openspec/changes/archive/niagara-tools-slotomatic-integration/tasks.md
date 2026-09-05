# Tareas: niagara-tools-slotomatic-integration

**Estado**: tasks | **Versión objetivo**: v0.3.0 | **Topic**: sdd/niagara-tools-slotomatic-integration/tasks
**Derivado de**: spec (engram #1927) + design (engram #1928) | **Fecha**: 2026-05-18

---

## Pronóstico de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas cambiadas estimadas | ~350 |
| Riesgo de presupuesto 400 líneas | Low |
| PRs encadenados recomendados | No |
| Split sugerido | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Unidades de trabajo sugeridas

| Unidad | Objetivo | PR | Notas |
|--------|----------|----|-------|
| 1 | Todo el cambio v0.3.0 | PR único | <400 líneas, single PR viable |

---

## Fase P — Preflight

- [ ] P.1 — Crear rama `git checkout -b feat/slotomatic-integration` desde `main` (ce8c089). **Archivos**: git. **REQ**: todos (prerequisito). **Agent-runnable**: sí. **Verificación**: `git branch --show-current` → `feat/slotomatic-integration`.
- [ ] P.2 — Ejecutar baseline de tests: `bats tests/ng-deploy.bats`. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-15. **Agent-runnable**: sí. **Verificación**: 17 tests verdes, 0 fallos.

---

## Fase R — RED (tests fallando primero)

- [ ] R.1 — Agregar fakebin `git` en `setup()` de `tests/ng-deploy.bats`: dispatch por `$1` → `diff` usa `FAKE_GIT_DIFF_OUTPUT`, `rev-parse --git-dir` exit 0, `rev-parse HEAD` usa `FAKE_GIT_REV_PARSE_OUTPUT`, `cat-file` usa `FAKE_GIT_CAT_FILE_EXIT`. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-4, REQ-7, REQ-9. **Agent-runnable**: sí. **Verificación**: variable `FAKE_GIT_DIFF_OUTPUT` presente en setup(); fakebin git creado en `$BATS_TMPDIR/bin/`.
- [ ] R.2 — Refactorizar stub `gradlew` para que append cada invocación en `gradlew.calls.log` (nueva) Y siga escribiendo `gradlew.args` (backward-compat con T1–T17). **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-13, REQ-15. **Agent-runnable**: sí. **Verificación**: T1–T17 siguen verdes; `gradlew.calls.log` acumula líneas por llamada.
- [ ] R.3 — T18: `--with-slotomatic` antepone tarea slotomatic ANTES de buildJar en `gradlew.calls.log`. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-1, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.4 — T19: fallo de gradle slotomatic → exit 15, "slotomatic" en stderr, sin build. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-3, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.5 — T20: detection WARN default cuando `FAKE_GIT_DIFF_OUTPUT` tiene match `@NiagaraType` y `--with-slotomatic` ausente → WARN en stderr, exit 0. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-4, REQ-5, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.6 — T21: `--strict-slotomatic` con detección positiva → exit 15 sin build. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-5, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.7 — T22: `--mode B --with-slotomatic` → WARN en stderr, slotomatic no invocado, ux build ejecutado, exit 0. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-2, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.8 — T23: mode B con annotation diff → sin detección, sin WARN, exit 0. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-6, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.9 — T24: deploy exitoso → `.last-deploy-sha` contiene HEAD SHA post-verify. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-7, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.10 — T25: `.last-deploy-sha` NO escrito si verify falla (helper `no-git-fakebin`). **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-7, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.11 — T26: `SLOTOMATIC_DETECTION=off` → sin detección, sin WARN, exit 0. **Archivos**: `tests/ng-deploy.bats`. **REQ**: REQ-5, REQ-13. **Agent-runnable**: sí. **Verificación**: test existe y falla (RED).
- [ ] R.12 — Ejecutar `bats tests/ng-deploy.bats` → confirmar 17 verdes + 9 rojos (T18–T26 RED). **Archivos**: ninguno. **REQ**: REQ-15. **Agent-runnable**: sí. **Verificación**: salida bats muestra exactamente 9 fallos nuevos y 17 pasos.

---

## Fase G — GREEN (implementación)

- [ ] G.1 — Agregar globals `WITH_SLOTOMATIC=0` y `STRICT_SLOTOMATIC=0` en sección de globals de `scripts/ng-deploy.sh`. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-1, REQ-5. **Agent-runnable**: sí. **Verificación**: vars presentes antes de `parse_args()`.
- [ ] G.2 — Extender `parse_args()` con cases `--with-slotomatic` (SET WITH_SLOTOMATIC=1) y `--strict-slotomatic` (SET STRICT_SLOTOMATIC=1). **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-1, REQ-5. **Agent-runnable**: sí. **Verificación**: T18 pasa; `parse_args --with-slotomatic` no error.
- [ ] G.3 — Implementar `read_baseline_sha()`: lee `$(pwd)/.last-deploy-sha`, valida con `git cat-file -e ${sha}^{commit}`, fallback `HEAD~1`. Always return 0. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-4, REQ-8. **Agent-runnable**: sí. **Verificación**: función presente; T24 avanza.
- [ ] G.4 — Implementar `write_last_deploy_sha()`: `git rev-parse HEAD > $(pwd)/.last-deploy-sha`, silent-fail end-to-end. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-7. **Agent-runnable**: sí. **Verificación**: T24 pasa; T25 pasa.
- [ ] G.5 — Implementar `warn_slotomatic_recommended()`: heredoc multi-línea a stderr indicando `--with-slotomatic`. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-5. **Agent-runnable**: sí. **Verificación**: T20 pasa.
- [ ] G.6 — Implementar `detect_annotation_changes()`: valida `SLOTOMATIC_DETECTION` != `off`, valida `command -v git` (exit 1 + notice si falta), valida repo (`rev-parse --git-dir`), llama `read_baseline_sha()`, ejecuta `git diff <baseline>..HEAD -- '*/src/com/**/*.java' | grep -E '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)'`. Return 0 si hay matches, 1 si no. NUNCA llama `die`. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-4, REQ-8, REQ-9. **Agent-runnable**: sí. **Verificación**: T20, T21, T23, T26 pasan.
- [ ] G.7 — Implementar `run_slotomatic()`: invoca `./gradlew :${MODULE_NAME}-rt:slotomatic ${GRADLE_OVERRIDES_3P}` (mismos -P que build_jars); `die 15` en exit no-zero. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-1, REQ-3. **Agent-runnable**: sí. **Verificación**: T18, T19 pasan.
- [ ] G.8 — Actualizar `print_usage()`: añadir `--with-slotomatic`, `--strict-slotomatic`, env var `SLOTOMATIC_DETECTION`, exit code 15 al bloque de ayuda. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-14. **Agent-runnable**: sí. **Verificación**: `ng-deploy.sh --help` muestra los 4 nuevos items.
- [ ] G.9 — Modificar `main()`: insertar bloque detection+slotomatic entre `backup` y `build_jars` según el snippet de diseño §3. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6. **Agent-runnable**: sí. **Verificación**: T18, T19, T20, T21, T22, T23 pasan.
- [ ] G.10 — Modificar `main()`: insertar `write_last_deploy_sha` después de `verify_cachebuster` y antes de `print_restart_reminder`. **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-7. **Agent-runnable**: sí. **Verificación**: T24, T25 pasan.
- [ ] G.11 — Ejecutar `bats tests/ng-deploy.bats` → confirmar los 26 tests verdes. **Archivos**: ninguno. **REQ**: REQ-13, REQ-15. **Agent-runnable**: sí. **Verificación**: 26 passed, 0 failed.

---

## Fase D — Docs y meta

- [ ] D.1 — CREAR `.env.local.example` en root del repo: ~80 LOC con 6 vars requeridos, 2 por-modo, `BUILD_ID`, `SLOTOMATIC_DETECTION=warn`, sección IMPORTANT con aviso `.gitignore` para `.last-deploy-sha`. **Archivos**: `.env.local.example` (nuevo). **REQ**: REQ-10. **Agent-runnable**: sí. **Verificación**: archivo existe; contiene `SLOTOMATIC_DETECTION` y aviso `.gitignore`.
- [ ] D.2 — Actualizar `CLAUDE.md §1` tabla de modos: fila Slotomatic/Property → `A --with-slotomatic (or :slotomatic separately first)`. **Archivos**: `CLAUDE.md`. **REQ**: REQ-11. **Agent-runnable**: sí. **Verificación**: `rg 'with-slotomatic' CLAUDE.md` retorna al menos 1 match.
- [ ] D.3 — Actualizar `docs/knowledge-base/slotomatic.md`: añadir Card 4 con tabla de escenarios, mecánica del heurístico, requisito `.gitignore`, edge case false-positive de comentario. **Archivos**: `docs/knowledge-base/slotomatic.md`. **REQ**: REQ-11. **Agent-runnable**: sí. **Verificación**: sección "Card 4" o "Integration" visible en el archivo.
- [ ] D.4 — Actualizar `docs/GOTCHAS.md`: añadir fila anti-patrón "Deploy with stale slotomatic". **Archivos**: `docs/GOTCHAS.md`. **REQ**: REQ-11. **Agent-runnable**: sí. **Verificación**: `rg 'stale slotomatic' docs/GOTCHAS.md` retorna match.
- [ ] D.5 — Actualizar `tests/smoke-checklist.md`: paso opcional `--with-slotomatic` en Mode A; nota no-op en Mode B. **Archivos**: `tests/smoke-checklist.md`. **REQ**: REQ-11. **Agent-runnable**: sí. **Verificación**: `rg 'with-slotomatic' tests/smoke-checklist.md` retorna match.
- [ ] D.6 — Bump `VERSION` de `0.2.0` a `0.3.0`. **Archivos**: `VERSION`. **REQ**: REQ-12. **Agent-runnable**: sí. **Verificación**: `cat VERSION` → `0.3.0`.
- [ ] D.7 — Añadir sección `## [0.3.0]` en `CHANGELOG.md` en formato Keep a Changelog: Added (flags, exit code, env var, .env.local.example), Changed (main() flow), References. **Archivos**: `CHANGELOG.md`. **REQ**: REQ-12. **Agent-runnable**: sí. **Verificación**: `rg '\[0.3.0\]' CHANGELOG.md` retorna match.

---

## Fase V — Validación

- [ ] V.1 — `bats tests/ng-deploy.bats` final → 26+ tests verdes. **Archivos**: ninguno. **REQ**: REQ-13, REQ-15. **Agent-runnable**: sí. **Verificación**: output "26 passed, 0 failed".
- [ ] V.2 — `shellcheck scripts/ng-deploy.sh` → exit 0 (o disable comments documentados). **Archivos**: `scripts/ng-deploy.sh`. **REQ**: REQ-15. **Agent-runnable**: sí. **Verificación**: shellcheck exit 0.
- [ ] V.3 — Cross-check REQ-1 a REQ-15: cada REQ tiene al menos 1 test y 1 función/change en código. **Archivos**: ninguno. **REQ**: todos. **Agent-runnable**: sí. **Verificación**: tabla de trazabilidad REQ → task sin huecos.
- [ ] V.4 — Cross-check escenarios S1–S11: cada escenario cubierto por al menos 1 test bats. **Archivos**: ninguno. **REQ**: todos. **Agent-runnable**: sí. **Verificación**: tabla S1–S11 → test ID sin huecos.

---

## Fase C — Preparación del commit (SIN commit por sdd-apply)

- [ ] C.1 — `git diff --stat` y capturar resumen de archivos/líneas modificados. **Archivos**: ninguno. **Agent-runnable**: sí. **Verificación**: output visible con 8–9 archivos.
- [ ] C.2 — `git add` de los archivos de implementación (excluir `openspec/` planning — se commitea por separado). **Archivos**: `scripts/ng-deploy.sh`, `tests/ng-deploy.bats`, `.env.local.example`, `CLAUDE.md`, `docs/knowledge-base/slotomatic.md`, `docs/GOTCHAS.md`, `tests/smoke-checklist.md`, `CHANGELOG.md`, `VERSION`. **Agent-runnable**: sí. **Verificación**: `git status` muestra solo los archivos de implementación staged.
- [ ] C.3 — Redactar borrador de mensaje de commit en estilo conventional commits (feat(ng-deploy): ...). **Archivos**: ninguno. **Agent-runnable**: sí. **Verificación**: borrador contiene scope `ng-deploy`, tipo `feat`, y referencias a REQ/tests.
- [ ] C.4 — STOP: presentar diff al operador para revisión antes de ejecutar `git commit`. **Agent-runnable**: no. **Verificación**: operador aprueba explícitamente.

---

## Trazabilidad REQ → Fase

| REQ | Fase(s) |
|-----|---------|
| REQ-1 | R.3, G.1, G.2, G.7, G.9 |
| REQ-2 | R.7, G.9 |
| REQ-3 | R.4, G.7 |
| REQ-4 | R.1, R.5, G.3, G.6 |
| REQ-5 | R.5, R.6, R.11, G.5, G.6, G.9 |
| REQ-6 | R.8, G.9 |
| REQ-7 | R.9, R.10, G.4, G.10 |
| REQ-8 | G.3, G.6 |
| REQ-9 | R.1, G.6 |
| REQ-10 | D.1 |
| REQ-11 | D.2, D.3, D.4, D.5 |
| REQ-12 | D.6, D.7 |
| REQ-13 | R.3–R.11, G.11, V.1 |
| REQ-14 | G.8 |
| REQ-15 | P.2, R.2, G.11, V.1, V.2 |
