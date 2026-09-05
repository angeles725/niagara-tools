# Proposal: niagara-tools-slotomatic-integration

**Estado**: propuesta
**Versión origen**: niagara-tools `v0.2.0`
**Versión objetivo**: niagara-tools `v0.3.0` (MINOR — flags nuevos, env var nuevo opcional, exit code nuevo, archivo nuevo `.env.local.example`)
**Phase**: propose (post-explore)
**Topic key**: `sdd/niagara-tools-slotomatic-integration/proposal`
**Exploración previa**: engram `#1924` + `openspec/changes/niagara-tools-slotomatic-integration/explore.md`

---

## 1. Por qué (Why)

### El dolor real, documentado

El día que se desplegó chihuahua sin haber corrido `:chihuahua-rt:slotomatic` después de agregar un nuevo `@NiagaraProperty`, la estación N4 quedó en **boot-loop** con el error `Type "chihuahua:ChiDashboardService" not found` (engram `#1074`). El jar compilado tenía la clase pero **sin** la región `AUTO GENERATED CODE` regenerada por slotomatic; el classloader de Niagara no pudo registrar el `BComponent`, la `ChiDashboardService` cayó al arrancar, y todo el host quedó inservible hasta que un humano hizo rollback del jar a mano desde `_backups/`.

Esa es **una sola** ocurrencia documentada. La gotcha hermana — "slotomatic exits `SUCCESS` pero no edita el archivo si la build previa tenía errores de compilación" (engram `#942`) — significa que **incluso operadores disciplinados** que "se acuerdan de correr slotomatic" pueden creer falsamente que lo corrieron y deployar a una estación rota.

### El upstream debe arreglarlo, no cada consumer

`chihuahua/build-and-deploy.ps1` tiene **el mismo gap** (engram `#1923`). El operador podría parcharlo ahí localmente, pero:

1. Cada nuevo consumer de `niagara-tools` (futuros proyectos: oficinas, sucursales, otros clientes) heredaría el mismo bug.
2. La política del repo (`CONTRIBUTING.md` §1) dice claramente que `ng-deploy.sh` es **la** superficie pública de deploy. Reparar el gap en chihuahua y dejar `niagara-tools` roto duplica el conocimiento y garantiza divergencia.
3. La mitigación actual — **disciplina del operador** ("acordate de correr slotomatic primero") — ya falló al menos una vez en producción. No es una mitigación; es un futuro postmortem esperando ocurrir.

### Costo de NO hacerlo

- Próximo cambio de anotación en chihuahua, oficinas o cualquier futuro módulo → riesgo recurrente de boot-loop.
- Cada operador tiene que aprender la lección dolorosamente (rollback nocturno, restart de estación, explicación al cliente).
- La regla de oro "el script público debe ser seguro por defecto" se viola en silencio.

---

## 2. Qué cambia (What)

Cambios concretos, todos contenidos en `niagara-tools`:

### 2.1 `scripts/ng-deploy.sh` (núcleo del cambio)

- **Flags nuevos** (aditivos, default off):
  - `--with-slotomatic` → corre `:${MODULE_NAME}-rt:slotomatic` como paso explícito antes de `build_jars`.
  - `--strict-slotomatic` → eleva la heurística de detección de `WARN` a `ABORT exit 15`.
- **Función nueva** `run_slotomatic()` — invoca `${GRADLEW_PATH}` con los mismos 3 overrides `-P` que `build_jars()` (consistencia validada por engram `#1402`).
- **Función nueva** `detect_annotation_changes()` — heurística con `git diff` sobre `*/src/com/**/*.java` filtrando `@Niagara(Type|Property|Action|Topic|Singleton)` con `grep` POSIX (no `rg`: el script corre en estaciones bare per `CLAUDE.md`).
- **Insertion point en `main()`**: entre `backup` y `build_jars` (Step 2.5). Slotomatic debe correr ANTES de build porque modifica fuentes Java que se compilan en el jar; corre DESPUÉS de backup porque backup snapshotea el jar desplegado (no las fuentes), así que el orden es indiferente para backup.
- **Exit code nuevo** `15` — slotomatic failed o (en modo strict) cambios de anotación detectados sin `--with-slotomatic`. Hueco `15` está libre entre los códigos existentes `10/20/30/40/50`.
- **Archivo nuevo** `.last-deploy-sha` — escrito a `$(pwd)/.last-deploy-sha` **después** de que `verify` (exit 50) pasa, NUNCA antes (ver OQ1 abajo). Si no existe, fallback baseline = `HEAD~1` (first-deploy safe).
- **Mode B guard**: si `--with-slotomatic` se pasa con `--mode B`, emite WARN y skip (slotomatic solo aplica a `-rt`); NO aborta (ver OQ3 abajo).
- **Help text actualizado** — `print_usage()` documenta los flags nuevos, el exit `15`, y la dependencia opcional de git.

### 2.2 `.env.local.example` (CREAR, hoy no existe)

`CLAUDE.md` y `README.md` ya lo referencian pero el archivo NO existe en el repo. Esta SDD lo crea con:
- Plantilla completa de los vars requeridos (`MODULE_NAME`, `GRADLEW_PATH`, `NIAGARA_HOME`, etc.).
- Var opcional nueva `SLOTOMATIC_DETECTION` (`warn` | `strict` | `off`) con doc inline.
- Aviso prominente: "**Agregar `.last-deploy-sha` a `.gitignore` de tu repo consumer.**"

### 2.3 `tests/ng-deploy.bats`

Mínimo **5 tests nuevos** (numerados 18+) sobre el archivo existente (que mantiene sus 17 tests intactos). Cubren:

- T18: `--with-slotomatic` prepende `:MODULE-rt:slotomatic` al pipeline de gradle (verificado con fakebin de `gradlew` capturando args).
- T19: slotomatic falla → exit `15`, mensaje a stderr, no se ejecuta build.
- T20: heurística detecta `@NiagaraProperty` agregado vía git fakebin → WARN a stderr, deploy continúa, exit `0`.
- T21: `--strict-slotomatic` + heurística positiva → exit `15`, deploy aborta antes de build.
- T22: `--mode B --with-slotomatic` → WARN "slotomatic only applies to -rt", build de ux sigue normal, exit `0`.

Patrón: reusar el fakebin pattern de `setup()` (engram `#1924` confirma compatibilidad). `git` se mockea como fakebin para tests unitarios; opcionalmente fixture de git real para `tests/smoke-checklist.md`.

### 2.4 `CLAUDE.md` §1 tabla de modos

Actualizar la fila "Slot/Property/Type/Action add or modify" para recomendar `--with-slotomatic` en lugar de "run `:slotomatic` separately first".

### 2.5 `CHANGELOG.md` + `VERSION`

- `VERSION`: `0.2.0` → `0.3.0` (MINOR per `CONTRIBUTING.md` §4: flag nuevo opt-in + env var nuevo opcional + exit code nuevo).
- `CHANGELOG.md`: entrada nueva con `### Added` listando flags + env var + exit `15` + archivo `.env.local.example`; `### Changed` para tabla de modos en CLAUDE.md.

### 2.6 Docs

- `docs/knowledge-base/slotomatic.md` — agregar tarjeta "Integración con `--with-slotomatic`" con cita a engram `#1074` y `#942`.
- `docs/GOTCHAS.md` — fila nueva en anti-patrones: "Deploy con slotomatic stale" con link bidireccional al KB topic.

---

## 3. Criterios de éxito (Success criteria)

1. **Regresión cero**: los 17 tests existentes en `tests/ng-deploy.bats` quedan green sin modificaciones.
2. **Tests nuevos green**: T18–T22 (mínimo) cubren los cinco caminos críticos.
3. **`shellcheck scripts/ng-deploy.sh tests/ng-deploy.bats` exit 0** (lint gate del repo, `CONTRIBUTING.md` §1).
4. **Smoke manual en chihuahua** (documentado en `tests/smoke-checklist.md`): operador ejecuta `ng-deploy.sh --mode A --with-slotomatic` desde `chihuahua/`, observa que slotomatic corre antes del build, deploy termina exit `0`, types verify pasa, estación arranca sin `Type not found`.
5. **`VERSION` = `0.3.0`** y `CHANGELOG.md` entry escrita siguiendo el formato del repo.
6. **`.env.local.example` existe en root** con la plantilla completa (no solo el var nuevo).
7. **`docs/knowledge-base/slotomatic.md` + `docs/GOTCHAS.md`** actualizados con links bidireccionales (`CONTRIBUTING.md` §3).

---

## 4. Fuera de alcance (Out of scope) — explícito

| Item | Razón |
|------|-------|
| `scripts/ng-deploy.ps1` | No existe hoy; SDD separada lo crea heredando esta lógica. |
| Bump del pin de chihuahua a `v0.3.0` | Ritual manual del operador post-merge — no es responsabilidad del repo upstream. |
| Internals del task gradle `:slotomatic` | Propiedad de Tridium plugins; no se tocan. |
| Ejecución real de slotomatic en mode B | Guard + skip solamente; mode B es ux-only y slotomatic solo aplica a `-rt`. |
| Actualizar `.gitignore` de los consumers | No se puede enforce desde `niagara-tools`; se documenta en `.env.local.example` y GOTCHAS. |
| File locking sobre `.last-deploy-sha` para deploys concurrentes | Operador solo; race aceptada (ver R6 carry-forward). |
| Auto-detección de cuándo correr slotomatic sin flag (approach (c)) | RECHAZADO por operador en pre-decisión: convierte el opt-in en magic-behavior. |

---

## 5. Open questions resueltas

### OQ1: ¿`.last-deploy-sha` se escribe antes o después de `verify` (exit 50)?

**Respuesta**: **DESPUÉS** del último `verify_jar` / `verify_cachebuster` exitoso, como último paso de `main()` antes del `print_restart_reminder`.

**Rationale**: el archivo representa "último deploy EXITOSO". Si verify falla (exit 50), el baseline NO debe avanzar — el próximo run sigue viendo el diff de anotaciones y puede WARN/strict correctamente. Escribirlo antes de verify rompería esa garantía.

### OQ2: ¿Mantener el valor `SLOTOMATIC_DETECTION=off`?

**Respuesta**: **MANTENER**.

**Rationale**: algunos consumers futuros pueden tener módulos no-Niagara en el mismo monorepo (servicios auxiliares, herramientas) donde la heurística generaría falsos positivos constantes. `off` es la válvula de escape sin tener que romper la build. Costo: 1 branch en el `case` de `detect_annotation_changes()` que retorna `0` (no detección). Beneficio: cero fricción para consumers con estructura no-canónica.

### OQ3: ¿Mode B + `--with-slotomatic` → WARN+skip o REJECT en `parse_args`?

**Respuesta**: **WARN + skip en runtime**, después de `parse_args` (en `main()` justo antes del call a `run_slotomatic`).

**Rationale**: rechazar en `parse_args` sorprende a operadores que usan aliases de shell o scripts wrapper que pasan `--with-slotomatic` incondicionalmente. El WARN explica el comportamiento ("slotomatic only applies to -rt, mode B is ux-only — skipping") y continúa. El operador aprende sin que su pipeline rompa.

### OQ4: Alcance del heurístico — `*/src/com/**/*.java` o también `**/srcTest/**`?

**Respuesta**: **Solo production source** (`*/src/com/**`).

**Rationale**: las anotaciones `@Niagara*` en test sources rara vez generan deploys reales (los tests no se compilan al jar de producción ni se registran en `module.xml`). Incluir test sources infla la tasa de falsos positivos sin beneficio. Si en el futuro algún consumer usa `@NiagaraType` en test fixtures, puede setear `SLOTOMATIC_DETECTION=off`.

---

## 6. Effort estimate

**Total: 1–2 días** incluyendo TDD (red → green → refactor) para cada test nuevo.

| Component | LOC aprox |
|-----------|-----------|
| `scripts/ng-deploy.sh` additions (run_slotomatic, detect_annotation_changes, parse_args extension, main() insertion, print_usage update) | ~80 |
| `tests/ng-deploy.bats` (T18–T22 + git fakebin helper) | ~150 |
| `.env.local.example` (template completo, no solo el var nuevo) | ~50 |
| `CLAUDE.md` §1 tabla update + §7 search hint | ~10 |
| `CHANGELOG.md` entry | ~15 |
| `docs/knowledge-base/slotomatic.md` integration card | ~25 |
| `docs/GOTCHAS.md` anti-pattern row | ~5 |
| `tests/smoke-checklist.md` slotomatic step | ~15 |
| **Total** | **~350** |

---

## 7. Delivery strategy hint

**Single PR**. ~350 LOC totales, por debajo del threshold de 400 líneas del Review Workload Guard del orquestador. No se requieren chained PRs ni `size:exception`.

Sub-batches sugeridos para `sdd-apply` (orden TDD-friendly, NO PRs separados):

1. Red: T18–T22 bats tests (fallan contra el script actual).
2. Green: `parse_args` + `print_usage` + globals (T18 + T22 pasan parcialmente, otros aún rojo).
3. Green: `run_slotomatic()` + insertion en `main()` (T18, T19, T22 green).
4. Green: `detect_annotation_changes()` + escritura de `.last-deploy-sha` post-verify (T20, T21 green).
5. Refactor + lint gate (`shellcheck` exit 0).
6. Docs + `.env.local.example` + `CHANGELOG.md` + `VERSION` bump.

---

## 8. Risks (carry-forward desde exploration + nuevos)

- **R1** (carry): fallback `HEAD~1` produce falsos negativos en ciclos multi-commit. **Mitigación**: `.last-deploy-sha` cuando existe; `HEAD~1` solo first-deploy safety; WARN explica claramente el baseline usado.
- **R2** (carry): falsos positivos por cambios en comentarios sobre líneas con `@Niagara*`. **Mitigación aceptada**: WARN no aborta; trade-off explícito de bajo signal-to-noise.
- **R3** (carry): `.last-deploy-sha` debe agregarse al `.gitignore` del consumer. No se puede enforce desde el repo upstream. **Mitigación**: doc prominente en `.env.local.example` + fila en `GOTCHAS.md`.
- **R4** (carry, resuelto): overrides `-P` para slotomatic validados (engram `#1402`). Sin riesgo.
- **R5** (carry, resuelto en OQ3): Mode B + flag = WARN + skip.
- **R6** (carry, aceptado): race en `.last-deploy-sha` para deploys concurrentes; no se implementa file-locking (operador solo).
- **R7** (carry, resuelto): default `warn` cuando `SLOTOMATIC_DETECTION` no está seteado → no-breaking.
- **R8** (carry, resuelto): `.env.local.example` se crea en esta SDD.
- **R9** (nuevo): `git` puede no estar disponible en algunas estaciones bare. **Mitigación**: `detect_annotation_changes()` debe envolver el call a `git` en `command -v git >/dev/null 2>&1 || { ...skip detection with WARN... }`. Solo afecta a la heurística; `--with-slotomatic` explícito no necesita git.
- **R10** (nuevo): operadores que descubren `--with-slotomatic` y lo agregan a un script CI/CD podrían encontrar que slotomatic no es idempotente bajo concurrencia (dos invocaciones simultáneas chocan en el mismo source tree). **Mitigación**: doc en KB topic dice "no correr en paralelo"; fuera del alcance de este SDD implementar locking.

---

## 9. Decisiones congeladas (NO re-debatir)

- Approach **(a) + (b)** confirmado por operador pre-decisión + exploration. Approach (c) auto-run RECHAZADO.
- Scope: SOLO `ng-deploy.sh` bash. `ng-deploy.ps1` queda en SDD futura separada.
- Heurística baseline: `.last-deploy-sha` con fallback a `HEAD~1`.
- Default: WARN-only; ABORT solo con `--strict-slotomatic` o `SLOTOMATIC_DETECTION=strict`.
- Backward compatibility: los 17 bats tests existentes quedan green sin tocar.
- Version bump: `0.2.0` → `0.3.0` (MINOR).
- `.env.local.example` se CREA en esta SDD.

---

## 10. Next phases

- `sdd-spec` — formaliza el contrato de flags, exit codes, env vars (puede correr en paralelo con `sdd-design`).
- `sdd-design` — diseña la estructura interna del script: orden de funciones, parámetros internos de `run_slotomatic` / `detect_annotation_changes`, formato del `.env.local.example`, scaffolding del fakebin de `git` para los bats.
- (después de ambos) `sdd-tasks` → `sdd-apply` (TDD strict) → `sdd-verify` → `sdd-archive`.
