# Spec: niagara-tools-slotomatic-integration

**Estado**: spec  
**Versión objetivo**: `v0.3.0` (MINOR)  
**Derivado de**: proposal.md (engram #1925)  
**Topic key**: `sdd/niagara-tools-slotomatic-integration/spec`  
**Fecha**: 2026-05-18

---

## 0. Alcance del delta

Este spec describe **únicamente el comportamiento nuevo o modificado** respecto a `v0.2.0`. Todo lo que no aparezca aquí queda sin cambios.

Los 17 tests de bats existentes deben continuar pasando sin modificación. El contrato de flags, exit codes y env vars de v0.2.0 es inmutable.

---

## 1. Requisitos (MUST)

### REQ-1 — Flag `--with-slotomatic`: invocación explícita antes de build

Cuando `--with-slotomatic` es pasado, `ng-deploy.sh` MUST invocar `${GRADLEW_PATH}` con la tarea `:${MODULE_NAME}-rt:slotomatic` ANTES de las tareas `clean` y `jar` del step de build (`build_jars`).

La invocación de slotomatic MUST usar los mismos 3 overrides `-P` que ya usa `build_jars()`:
- `-Pniagara_home="${NIAGARA_HOME}"`
- `"-Pniagara_user_home=${NIAGARA_USER_HOME}"`
- `-Porg.gradle.java.installations.paths="${JAVA_HOME}"`

Sin `--with-slotomatic`, el comportamiento de build MUST ser idéntico al de v0.2.0.

### REQ-2 — Guard `--mode B` + `--with-slotomatic`

Cuando se pasan simultáneamente `--with-slotomatic` y `--mode B`, el script MUST:
1. Emitir un mensaje WARN a stderr explicando que slotomatic solo aplica a `-rt` y que mode B es ux-only.
2. Omitir la invocación de slotomatic (el step de slotomatic queda como no-op).
3. Continuar con el build ux normal.
4. Salir con código 0 si todo lo demás es exitoso.

El script MUST NOT abortar por esta combinación de flags.

### REQ-3 — Exit code 15 para falla de slotomatic

Si la invocación de gradle para slotomatic retorna un código de salida distinto de 0, `ng-deploy.sh` MUST:
1. Invocar `die 15` con un mensaje que identifique a slotomatic como el punto de falla.
2. No continuar con build ni deploy.
3. No restaurar automáticamente el backup ya tomado.

El exit code 15 MUST NOT colisionar con los existentes (10/20/30/40/50) y MUST aparecer documentado en `print_usage()`.

### REQ-4 — Función `detect_annotation_changes()`

El script MUST incluir una función `detect_annotation_changes()` que:

1. Determina el baseline:
   - Si `$(pwd)/.last-deploy-sha` existe Y no está vacío Y el SHA en él existe en el historial git → usa ese SHA como baseline.
   - En cualquier otro caso (archivo inexistente, vacío, SHA inválido) → usa `HEAD~1` como baseline.
2. Ejecuta `git diff <baseline>..HEAD -- '*/src/com/**/*.java'`.
3. Filtra la salida con `grep -E '@Niagara(Type|Property|Action|Topic|Singleton)'` (POSIX, no `rg`).
4. Retorna 0 si hay matches (= cambio de anotación detectado).
5. Retorna 1 si no hay matches (= sin cambio de anotación).

La función MUST usar únicamente bash y herramientas POSIX. MUST NOT usar `rg` ni herramientas que requieran instalación extra.

### REQ-5 — Matriz de comportamiento según detección

Cuando `detect_annotation_changes()` retorna 0 (cambio detectado) Y `--with-slotomatic` NO fue pasado, el comportamiento MUST seguir esta tabla:

| Condición | Comportamiento MUST |
|-----------|-------------------|
| `SLOTOMATIC_DETECTION` no seteado O `=warn` (default) | Emitir WARN a stderr. Continuar deploy. Exit 0. |
| Flag `--strict-slotomatic` pasado O `SLOTOMATIC_DETECTION=strict` | Invocar `die 15` con mensaje claro. NO build. NO deploy. |
| `SLOTOMATIC_DETECTION=off` | No invocar `detect_annotation_changes()` en absoluto. Sin WARN. |

Valores de `SLOTOMATIC_DETECTION` distintos a `warn`, `strict`, `off` MUST ser tratados como `warn` (default seguro).

### REQ-6 — Mode B omite detección de anotaciones

Cuando `--mode B` es seleccionado, `detect_annotation_changes()` MUST NOT ser invocada bajo ninguna circunstancia (con o sin `SLOTOMATIC_DETECTION` seteado, con o sin `--strict-slotomatic`). Mode B es ux-only; la detección de anotaciones aplica únicamente a fuentes `-rt`.

### REQ-7 — Escritura de `.last-deploy-sha` post-deploy exitoso

Tras completar exitosamente los pasos de verify (`verify_jar` + `verify_cachebuster` cuando aplica), y antes de llamar a `print_restart_reminder`, el script MUST:
1. Ejecutar `git rev-parse HEAD`.
2. Escribir el SHA resultante en `$(pwd)/.last-deploy-sha`.
3. Si `git rev-parse` falla o el directorio de trabajo no es un repo git, silenciar el error y omitir la escritura (no emitir nada a stderr ni stdout).

La escritura MUST ocurrir únicamente tras verify exitoso — nunca antes, nunca si verify falla.

### REQ-8 — Tolerancia en la lectura de `.last-deploy-sha`

`detect_annotation_changes()` MUST manejar los siguientes casos sin abortar ni emitir error al operador:

| Caso | Comportamiento MUST |
|------|-------------------|
| Archivo no existe | Usar `HEAD~1` como baseline |
| Archivo existe pero está vacío | Usar `HEAD~1` como baseline |
| SHA en el archivo no existe en historial git | Usar `HEAD~1` como baseline, sin error |

### REQ-9 — Guard cuando `git` no está disponible

`detect_annotation_changes()` MUST verificar `command -v git` al inicio. Si `git` no está en `$PATH`:
1. Emitir una sola línea a stderr: `[ng-deploy] git not available, skipping annotation change detection`.
2. Retornar 1 (sin detección).
3. El script MUST continuar normalmente (build + deploy).

`--with-slotomatic` explícito no depende de `git` y MUST funcionar independientemente de si `git` está disponible.

### REQ-10 — `.env.local.example` en root del repo

El repo MUST contener el archivo `.env.local.example` en la raíz con una plantilla completa que incluya:

- Todos los vars requeridos por `ng-deploy.sh`: `NIAGARA_TOOLS_HOME`, `MODULE_NAME`, `MODULE_NICKNAME`, `STATION_MODULES_DIR`, `NIAGARA_HOME`, `NIAGARA_USER_HOME`, `JAVA_HOME`, `EXPECTED_RT_TYPES`, `EXPECTED_UX_TYPES`, `BUILD_ID_PATH`.
- El nuevo var opcional `SLOTOMATIC_DETECTION=warn` con comentario explicando los 3 valores (`warn` | `strict` | `off`).
- Un comentario prominente indicando que el consumer DEBE agregar `.last-deploy-sha` a su `.gitignore`.

Cada var MUST estar comentado en el template (prefijado con `#`) para que el consumer descomente solo lo necesario.

### REQ-11 — Actualización de tabla de modos en `CLAUDE.md` §1

La fila "Slot / Property / Type / Action add or modify" en la tabla de `CLAUDE.md §1` MUST pasar de:

```
A + run `:slotomatic` separately first
```

a indicar `--with-slotomatic` como el mecanismo recomendado:

```
A --with-slotomatic
```

con la nota de que station restart sigue siendo requerido.

### REQ-12 — VERSION bump + entrada en CHANGELOG.md

- El archivo `VERSION` MUST contener exactamente `0.3.0`.
- `CHANGELOG.md` MUST tener una entrada `## [0.3.0] - YYYY-MM-DD` en formato Keep a Changelog, incluyendo:
  - `### Added`: `--with-slotomatic`, `--strict-slotomatic`, exit code 15, `SLOTOMATIC_DETECTION` env var, `.env.local.example`.
  - `### Changed`: tabla de modos en `CLAUDE.md §1`.

### REQ-13 — Suite de tests bats expandida (mínimo 5 tests nuevos)

`tests/ng-deploy.bats` MUST agregar al menos los siguientes tests (T18–T22), usando el mismo patrón fakebin de `setup()`:

| Test | Qué verifica |
|------|-------------|
| T18 | `--with-slotomatic` + `--mode A` → gradle invocado con `:test-rt:slotomatic` ANTES de `:test-rt:clean` |
| T19 | Slotomatic falla (gradlew exit != 0) → exit code 15, mensaje "slotomatic" en stderr, no se invoca build |
| T20 | Heurística detecta `@NiagaraProperty` (git fakebin retorna diff positivo) + default SLOTOMATIC_DETECTION → WARN en stderr, exit 0, deploy continúa |
| T21 | Heurística positiva + `--strict-slotomatic` → exit 15, NO build, NO deploy |
| T22 | `--mode B` + `--with-slotomatic` → WARN en stderr sobre slotomatic siendo rt-only, slotomatic NO invocado, build ux normal, exit 0 |

`git` MUST ser mockeado como fakebin en `setup()` para los tests que involucren `detect_annotation_changes()`.

Los 17 tests existentes (T1–T17) MUST pasar sin modificación.

### REQ-14 — Actualización de `print_usage()`

`print_usage()` MUST documentar, en el mismo estilo que los flags existentes:
- `--with-slotomatic`: descripción y efecto.
- `--strict-slotomatic`: descripción y efecto.
- Exit code `15`: en la tabla de exit codes.
- `SLOTOMATIC_DETECTION`: en la sección de vars opcionales.

### REQ-15 — Compatibilidad hacia atrás (backward compatibility)

Todos los modos existentes (`--mode A|B|C`) invocados sin los flags nuevos MUST producir un comportamiento IDÉNTICO al de v0.2.0. Los 17 tests bats existentes MUST seguir verdes sin ninguna modificación.

---

## 2. Escenarios de aceptación (Given / When / Then)

### S1 — Happy path: `--with-slotomatic --mode A` exitoso

**Given** un repo válido con `.env.local` completo, `--mode A`, `--with-slotomatic`.  
**When** el script ejecuta.  
**Then**:
- `gradlew` es invocado primero con `:MODULE-rt:slotomatic` (con los 3 overrides `-P`).
- Luego `gradlew` es invocado con `:MODULE-rt:clean :MODULE-rt:jar :MODULE-ux:clean :MODULE-ux:jar`.
- Backup, copy y verify se completan.
- `.last-deploy-sha` es escrito con el SHA de `HEAD` tras verify exitoso.
- Exit code: 0.

### S2 — Slotomatic falla

**Given** `--with-slotomatic --mode A` y gradlew retorna exit 1 al invocar `:MODULE-rt:slotomatic`.  
**When** el script ejecuta.  
**Then**:
- Stderr contiene "[ng-deploy] ERROR" y la palabra "slotomatic".
- Exit code: 15.
- Backup ya tomado NO es restaurado automáticamente.
- `build_jars()` NO es invocado (ninguna tarea `clean`/`jar` en `gradlew.args`).
- `.last-deploy-sha` NO es escrito.

### S3 — Heurística WARN con default SLOTOMATIC_DETECTION

**Given** `--mode A` sin `--with-slotomatic`, `SLOTOMATIC_DETECTION` no seteado (o `=warn`), y `git diff HEAD~1..HEAD` retorna al menos una línea con `@NiagaraProperty`.  
**When** el script ejecuta.  
**Then**:
- Stderr contiene una línea de WARN mencionando la detección de cambios de anotación y la sugerencia de usar `--with-slotomatic`.
- Build, copy y verify continúan normalmente.
- Exit code: 0.

### S4 — Detección strict aborta antes de build

**Given** `--mode A`, `--strict-slotomatic` pasado, y `git diff` retorna `@NiagaraType` en fuentes java.  
**When** el script ejecuta.  
**Then**:
- Exit code: 15.
- Stderr contiene mensaje claro sobre cambio de anotación detectado y ausencia de `--with-slotomatic`.
- `build_jars()` NO es invocado.
- `copy_jars()` NO es invocado.

### S5 — Mode B con `--with-slotomatic`: WARN + skip

**Given** `--mode B --with-slotomatic`.  
**When** el script ejecuta.  
**Then**:
- Stderr contiene WARN explicando que slotomatic solo aplica a `-rt` y que mode B es ux-only.
- `:MODULE-rt:slotomatic` NO aparece en los args de gradle.
- Build ux (`clean`/`jar` para ux) se ejecuta normalmente.
- Exit code: 0 (si build + verify son exitosos).

### S6 — Mode B omite detección de anotaciones

**Given** `--mode B`, fuentes Java con `@NiagaraProperty` modificadas entre baseline y HEAD.  
**When** el script ejecuta.  
**Then**:
- `detect_annotation_changes()` NO es invocada (ningún `git diff` ejecutado).
- Ningún WARN sobre anotaciones en stderr.
- Build ux se ejecuta normalmente.
- Exit code: 0.

### S7 — Escritura de `.last-deploy-sha` tras verify exitoso

**Given** deploy completo con `--mode A`, verify de types ok, verify de cachebuster ok (si BUILD_ID seteado).  
**When** todos los verifies pasan.  
**Then**:
- `$(pwd)/.last-deploy-sha` contiene el SHA de `git rev-parse HEAD`.
- La invocación siguiente de `detect_annotation_changes()` usa ese SHA como baseline.
- `print_restart_reminder` se llama después de escribir el archivo.

### S8 — Primer deploy (sin `.last-deploy-sha`)

**Given** `--mode A` sin `--with-slotomatic`, sin `$(pwd)/.last-deploy-sha`, y hay cambio de anotación entre `HEAD~1` y `HEAD`.  
**When** el script ejecuta.  
**Then**:
- `detect_annotation_changes()` usa `HEAD~1` como baseline.
- WARN emitido a stderr sobre cambio de anotación detectado.
- Deploy continúa normalmente.
- Exit code: 0.

### S9 — `git` ausente del PATH

**Given** `git` no está disponible en `$PATH` (ni `--with-slotomatic` ni `--strict-slotomatic` pasados), `--mode A`.  
**When** el script ejecuta.  
**Then**:
- Stderr contiene exactamente una línea: `[ng-deploy] git not available, skipping annotation change detection`.
- Build, copy y verify proceden normalmente.
- Exit code: 0 (si todo lo demás es exitoso).

### S10 — `SLOTOMATIC_DETECTION=off` desactiva detección

**Given** `SLOTOMATIC_DETECTION=off` en el entorno, cambio de anotación presente, `--mode A` sin `--with-slotomatic`.  
**When** el script ejecuta.  
**Then**:
- Ningún WARN sobre anotaciones en stderr.
- `detect_annotation_changes()` no es invocada.
- Deploy procede normalmente.
- Exit code: 0.

### S11 — Regresión: todos los tests existentes siguen green

**Given** la suite de bats con los 17 tests existentes (T1–T17) más los nuevos (T18–T22+).  
**When** se ejecuta `bats tests/ng-deploy.bats`.  
**Then**:
- Los 17 tests originales pasan sin ninguna modificación en su código.
- Los tests nuevos T18–T22 (mínimo) también pasan.
- `shellcheck scripts/ng-deploy.sh tests/ng-deploy.bats` retorna exit 0.

---

## 3. Requisitos no funcionales

### Rendimiento

El overhead de slotomatic (cuando `--with-slotomatic` es usado) es de 3–5 segundos adicionales según experiencia documentada de chihuahua. Este overhead es aceptable y solo ocurre cuando el operador opt-in explícitamente.

### Compatibilidad de shell

El script MUST continuar siendo `bash` con herramientas POSIX únicamente. MUST NOT introducir dependencias de `rg`, `fd`, `bat` ni ninguna herramienta que no esté disponible en estaciones Niagara bare. `git` es opcional y guarded (REQ-9).

### Sin nuevas dependencias de runtime

Las únicas dependencias del script son `bash`, `git` (opcional + guarded), `gradle` (ya requerido). No se agrega ninguna otra.

---

## 4. Fuera de alcance (mirror de proposal §4)

| Item | Razón |
|------|-------|
| `scripts/ng-deploy.ps1` | No existe hoy; SDD separada lo crea heredando esta lógica |
| Bump del pin de chihuahua a `v0.3.0` | Ritual manual del operador post-merge |
| Internals del task gradle `:slotomatic` | Propiedad de Tridium plugins; no se tocan |
| Ejecución real de slotomatic en mode B | Guard + skip solamente |
| Actualizar `.gitignore` de los consumers | Documentado en `.env.local.example` y GOTCHAS |
| File locking sobre `.last-deploy-sha` | Operador solo; race aceptada |
| Auto-detección sin flag (approach c) | RECHAZADO; convierte opt-in en magic-behavior |

---

## 5. Superficie pública de v0.3.0 (contrato formal)

### Flags nuevos

| Flag | Tipo | Default | Efecto |
|------|------|---------|--------|
| `--with-slotomatic` | boolean | off | Corre `:${MODULE_NAME}-rt:slotomatic` antes de build |
| `--strict-slotomatic` | boolean | off | Eleva detección de WARN a die 15 |

### Env var nuevo

| Variable | Valores | Default (si no seteada) | Efecto |
|----------|---------|------------------------|--------|
| `SLOTOMATIC_DETECTION` | `warn` \| `strict` \| `off` | `warn` | Controla comportamiento de detección heurística |

### Exit codes v0.3.0 (completo)

| Código | Significado |
|--------|-------------|
| 0 | Éxito (o `--help`) |
| 10 | Env var requerida ausente o path inválido |
| 20 | Backup falló, o `--no-backup` sin `--i-know-what-im-doing` |
| 30 | Build (gradlew) retornó non-zero |
| 40 | Copy del jar a `STATION_MODULES_DIR` falló |
| 50 | Verify falló: mismatch de tipos o `BUILD_ID` no encontrado en `index.html` |
| **15** | **Slotomatic gradle task falló, O (modo strict) cambio de anotación detectado sin `--with-slotomatic`** |

### Archivos nuevos en el repo

| Archivo | Descripción |
|---------|-------------|
| `.env.local.example` | Template completo de configuración con todos los vars + `SLOTOMATIC_DETECTION` + aviso de `.gitignore` |
| (por deploy) `.last-deploy-sha` | SHA del último deploy exitoso; NO rastreado por git del repo consumer |

### Funciones nuevas en `scripts/ng-deploy.sh`

| Función | Responsabilidad |
|---------|----------------|
| `run_slotomatic()` | Invoca `${GRADLEW_PATH}` con `:${MODULE_NAME}-rt:slotomatic` + los 3 overrides `-P`; die 15 si falla |
| `detect_annotation_changes()` | Heurística git diff; retorna 0 si hay `@Niagara*` en diff de fuentes; guarded para git ausente |

### Orden de invocación en `main()` (delta)

```
parse_args
load_env_file
validate_required
guard_no_backup

[NEW] if mode != B && SLOTOMATIC_DETECTION != off:
          detect_annotation_changes → WARN or die 15 per REQ-5

backup (si !NO_BACKUP)

[NEW] if --with-slotomatic && mode != B:
          run_slotomatic → die 15 si falla
      elif --with-slotomatic && mode == B:
          WARN a stderr, skip slotomatic

build_jars
copy_jars     (si !NO_DEPLOY)
verify_*      (si !NO_DEPLOY)

[NEW] write .last-deploy-sha (silencioso si git falla)

print_restart_reminder
exit 0
```

---

## 6. Riesgos que forzaron asunciones en el spec

| Riesgo | Asunción tomada |
|--------|----------------|
| R1 (carry): fallback `HEAD~1` produce falsos negativos en ciclos multi-commit | Aceptado; `.last-deploy-sha` mitiga para deploys subsiguientes; WARN incluye baseline usado |
| R2 (carry): falsos positivos por comentarios sobre líneas con `@Niagara*` | Aceptado; WARN no aborta; trade-off explícito |
| R3 (carry): `.last-deploy-sha` debe agregarse al `.gitignore` del consumer | No enforceable upstream; mitigado con doc en `.env.local.example` + GOTCHAS |
| R9 (nuevo): `git` puede no estar en PATH en estaciones bare | REQ-9 guarda con `command -v git`; retorna 1 (no detección), continúa normalmente |
| Riesgo de spec: posición exacta de `detect_annotation_changes()` antes vs después de backup | Asunción: ANTES del backup (fail fast antes de tomar el snapshot) — no debatido explícitamente en la proposal; no es crítico porque backup no modifica fuentes, pero el orden conservador es pre-backup |
