# Design: niagara-tools-slotomatic-integration

**Estado**: design | **Versión**: v0.2.0 → v0.3.0 (MINOR) | **Topic**: `sdd/niagara-tools-slotomatic-integration/design`
**Proposal**: engram #1925 + `openspec/changes/niagara-tools-slotomatic-integration/proposal.md`
**Exploration**: engram #1924 + `openspec/changes/niagara-tools-slotomatic-integration/explore.md`

> Diseño del **HOW** a nivel arquitectónico. La descomposición concreta en steps de implementación vive en `tasks.md` (siguiente fase).

---

## 0. Resumen ejecutivo

Se extiende `scripts/ng-deploy.sh` (368 LOC, v0.2.0) con un **Step 2.5 opt-in** (`run_slotomatic`) y un **Step 2.4 heurístico** (`detect_annotation_changes`) que se insertan entre `backup` y `build_jars` del `main()` actual. Toda la nueva lógica es **aditiva**: ninguna ruta existente cambia de comportamiento por defecto, y los 17 tests existentes deben permanecer verdes sin tocar.

El diseño descansa sobre cuatro decisiones que ya están congeladas en la proposal:

1. **Approach (a)+(b)**: flag explícito `--with-slotomatic` + heurística pasiva `detect_annotation_changes` con dos sub-modos (`warn` default / `strict` opt-in).
2. **Baseline state**: archivo `.last-deploy-sha` en `$(pwd)` escrito **después** de `verify`, con fallback `HEAD~1` si falta.
3. **Mode B guard**: WARN+skip en runtime (no reject en `parse_args`) — operadores con aliases no se rompen.
4. **Backward compat**: globals nuevos default 0 / `SLOTOMATIC_DETECTION` default `warn`. Tests 1–17 intocables.

LOC estimado: **~80 LOC bash + ~150 LOC bats + ~50 LOC .env.local.example + ~70 LOC docs ≈ 350 LOC totales** → cabe en un PR único (<400 line budget).

---

## 1. Descomposición de funciones nuevas en `ng-deploy.sh`

Cada función nueva tiene: **firma**, **params**, **return**, **side effects**, **error conditions** (cuándo `die`, cuándo no).

### 1.1 `run_slotomatic()`

```bash
# run_slotomatic — invoca :MODULE_NAME-rt:slotomatic reusando los 3 -P overrides
# Params: ninguno (lee globales MODULE_NAME, GRADLEW_PATH, NIAGARA_HOME, NIAGARA_USER_HOME, JAVA_HOME)
# Returns: 0 al success
# Side effects: invoca gradlew; puede modificar archivos AUTO GENERATED CODE en *-rt/src/...
# Errors: die 15 si gradlew exit != 0
run_slotomatic() {
    printf '[ng-deploy] slotomatic: regenerating AUTO GENERATED CODE for %s-rt\n' "$MODULE_NAME"
    "${GRADLEW_PATH}" \
        -Pniagara_home="${NIAGARA_HOME}" \
        "-Pniagara_user_home=${NIAGARA_USER_HOME}" \
        -Porg.gradle.java.installations.paths="${JAVA_HOME}" \
        ":${MODULE_NAME}-rt:slotomatic" \
        || die 15 "slotomatic failed (gradlew exited non-zero)"
    printf '[ng-deploy] slotomatic ok\n'
}
```

**Decisión**: NO calcular `module_root` con `dirname "$GRADLEW_PATH"` (como sugería la exploración). El gradlew wrapper ya sabe resolverse vía la task qualified `:MODULE-rt:slotomatic`. Mantenemos paralelismo exacto con `build_jars()`.

### 1.2 `detect_annotation_changes()`

```bash
# detect_annotation_changes — heurística: ¿cambió alguna @NiagaraXxx annotation desde el último deploy?
# Params: ninguno (lee SLOTOMATIC_DETECTION, llama read_baseline_sha)
# Returns: 0 = cambios detectados (caller debe warn/abort), 1 = no cambios o no aplicable
# Side effects: puede imprimir a stderr en modo verbose
# Errors: NUNCA llama die directamente — devuelve 1 (no-cambios) en cualquier modo fallback
detect_annotation_changes() {
    local detection_mode="${SLOTOMATIC_DETECTION:-warn}"

    # Modo explícitamente apagado: skip silencioso
    case "$detection_mode" in
        off) return 1 ;;
        warn|strict) ;;
        *)
            printf '[ng-deploy] WARN invalid SLOTOMATIC_DETECTION="%s" (valid: warn|strict|off); treating as off\n' \
                "$detection_mode" >&2
            return 1
            ;;
    esac

    # git ausente → skip silencioso con nota (estaciones bare)
    if ! command -v git >/dev/null 2>&1; then
        printf '[ng-deploy] note: git not available, skipping annotation change detection\n' >&2
        return 1
    fi

    # Si no estamos en un repo git válido, también skip
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        printf '[ng-deploy] note: not inside a git repository, skipping annotation change detection\n' >&2
        return 1
    fi

    local baseline
    baseline="$(read_baseline_sha)"

    # diff filtrado por production source paths + annotations Niagara
    # why: solo *-rt y *-ux production source (no srcTest, no build/, no docs/)
    # why: grep -E POSIX-friendly; rg deliberadamente prohibido para estaciones bare
    if git diff "${baseline}..HEAD" -- '*/src/com/**/*.java' 2>/dev/null \
        | grep -E '^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)' \
        > /dev/null 2>&1; then
        return 0
    fi
    return 1
}
```

**Decisiones de refinamiento sobre la exploración**:

- **Edge case: baseline SHA orphaned** — manejado en `read_baseline_sha` (validación `git cat-file -e`).
- **Edge case: detached HEAD** — `HEAD~1` sigue siendo resoluble en detached HEAD si hay ≥2 commits. Si no hay, `git diff "HEAD~1..HEAD"` falla silenciosamente con `2>/dev/null` → return 1 (no-cambios). Aceptable: tree con un solo commit ya es un caso degenerado.
- **Edge case: diff command falla** — `2>/dev/null` swallow + el pipe a grep recibe vacío → grep retorna 1 → función retorna 1. No die. Operador puede pasar `--with-slotomatic` manualmente.
- **Refinamiento grep**: el patrón ahora exige prefijo `^[+-]` (linea diff add/remove) y permite whitespace. Esto elimina los falsos positivos de `^---`/`^+++` (headers de diff) que tenía la exploración original sin re-filtrar después. Más simple y más correcto.
- **Validación de `SLOTOMATIC_DETECTION`**: valores no reconocidos NO abortan el deploy — emiten WARN y degradan a `off`. Filosofía: detección es un assist, no un gate (excepto strict explícito).

### 1.3 `read_baseline_sha()`

```bash
# read_baseline_sha — resuelve el SHA baseline para diff de annotations
# Params: ninguno
# Returns: imprime a stdout el SHA válido o "HEAD~1" como fallback. Siempre return 0.
# Side effects: lee $(pwd)/.last-deploy-sha si existe
# Errors: nunca die — siempre produce un baseline usable
read_baseline_sha() {
    local sha_file="$(pwd)/.last-deploy-sha"
    local baseline=""

    if [[ -f "$sha_file" ]]; then
        # tr para sanear newlines/whitespace accidentales
        baseline="$(tr -d '[:space:]' < "$sha_file" 2>/dev/null || true)"
    fi

    # Validar que el SHA exista en la historia git
    if [[ -n "$baseline" ]] && git cat-file -e "${baseline}^{commit}" 2>/dev/null; then
        printf '%s' "$baseline"
        return 0
    fi

    # Fallback: HEAD~1
    printf 'HEAD~1'
    return 0
}
```

**Refinamiento**: usar `${baseline}^{commit}` en `git cat-file -e` para validar que el SHA NO solo existe sino que es un commit (defensa contra `.last-deploy-sha` corrompido con un tree/blob SHA).

### 1.4 `write_last_deploy_sha()`

```bash
# write_last_deploy_sha — graba el SHA actual como baseline para el próximo deploy
# Params: ninguno
# Returns: 0 siempre (silent failure)
# Side effects: escribe $(pwd)/.last-deploy-sha
# Errors: NUNCA die — fallar al escribir baseline no debe tumbar un deploy exitoso
write_last_deploy_sha() {
    if ! command -v git >/dev/null 2>&1; then return 0; fi
    local sha
    sha="$(git rev-parse HEAD 2>/dev/null)" || return 0
    [[ -z "$sha" ]] && return 0
    printf '%s\n' "$sha" > "$(pwd)/.last-deploy-sha" 2>/dev/null || true
    return 0
}
```

**Decisión clave**: cero exit codes derivados de esta función. Si falla, el deploy ya fue exitoso (estamos POST-verify). No queremos hacer "rollback" sobre un deploy bueno por un baseline file que no se pudo escribir.

### 1.5 `warn_slotomatic_recommended()`

```bash
# warn_slotomatic_recommended — emite el WARN canónico cuando detection detecta cambios sin --with-slotomatic
# Params: ninguno (lee MODE para mensaje contextual)
# Returns: 0 siempre
# Side effects: imprime a stderr
# Errors: ninguno
warn_slotomatic_recommended() {
    cat >&2 << 'WARN'
[ng-deploy] WARN: @Niagara* annotation changes detected since last deploy,
                 but --with-slotomatic was NOT passed. The build will proceed,
                 but the generated AUTO GENERATED CODE may be stale, causing
                 'Type X not found' errors at station boot.
                 Re-run with --with-slotomatic to regenerate, or pass
                 --strict-slotomatic / SLOTOMATIC_DETECTION=strict to make
                 this fatal.
WARN
}
```

**Decisión**: mensaje multi-línea con heredoc — provee acción concreta al operador. NO usamos `printf` con `\n` (más legible heredoc, idéntico al patrón `print_usage`).

### 1.6 Helper compartido — NO se extrae

`validate_git_available()` se mencionaba en la proposal, pero se materializa como `command -v git` inline (2 sitios: `detect_annotation_changes` y `write_last_deploy_sha`). **Extraerlo en una función reduciría legibilidad** sin ahorrar LOC. Skip.

---

## 2. Globals + arg parsing

### 2.1 Globals nuevos

Insertados **después** del bloque de globals existentes (después de `ENV_FILE=".env.local"`, antes de `print_usage`):

```bash
# ---------------------------------------------------------------------------
# Slotomatic integration (v0.3.0+) — opt-in flags, default off
# ---------------------------------------------------------------------------
WITH_SLOTOMATIC=0        # --with-slotomatic → 1: run slotomatic before build
STRICT_SLOTOMATIC=0      # --strict-slotomatic → 1: detection findings are fatal
# Env var SLOTOMATIC_DETECTION ∈ {warn|strict|off} consumed by detect_annotation_changes
```

### 2.2 `parse_args()` — nuevos cases

Añadir entre `--no-deploy` y el `*)` catch-all:

```bash
            --with-slotomatic)
                WITH_SLOTOMATIC=1
                shift ;;
            --strict-slotomatic)
                STRICT_SLOTOMATIC=1
                shift ;;
```

**Decisión**: `--strict-slotomatic` NO implica `--with-slotomatic`. Razonamiento: un operador disciplinado puede querer "detección estricta SIN auto-correr slotomatic" (CI gate, o equipo donde slotomatic se corre manualmente con review). Si quisiera ambos, los pasa juntos. Cero magia.

### 2.3 Env var `SLOTOMATIC_DETECTION`

**Decisión**: NO la valido en `validate_required()` (no es requerida) NI creo un `validate_optional()` (over-engineering para una sola var). La validación de valor (`warn|strict|off`) vive **dentro de `detect_annotation_changes()`** — donde se consume, donde el WARN tiene sentido contextual.

**Tradeoff aceptado**: si el operador escribe `SLOTOMATIC_DETECTION=worn` (typo), no se entera hasta que detect corre por primera vez. Aceptable: detect corre en cada deploy mode A/C, así que el WARN aparece en el primer deploy post-typo.

---

## 3. `main()` — flow modificado

### 3.1 Flujo nuevo vs actual

```
ACTUAL (v0.2.0):
parse_args → load_env_file → validate_required → guard_no_backup
→ backup
→ build_jars → [--no-deploy early exit]
→ copy_jars → verify_jar → verify_cachebuster
→ print_restart_reminder → exit 0

NUEVO (v0.3.0):
parse_args → load_env_file → validate_required → guard_no_backup
→ backup
→ [MODE != B] {                                ← NUEVO bloque slotomatic
    detect_annotation_changes && {
        [STRICT_SLOTOMATIC=1 OR SLOTOMATIC_DETECTION=strict]
            ? die 15 "annotation changes detected; --with-slotomatic required (strict mode)"
            : [WITH_SLOTOMATIC=0]
                ? warn_slotomatic_recommended
                : true  # ya van a correrlo, no warn
    }
    [WITH_SLOTOMATIC=1] && run_slotomatic
  }
→ [MODE == B AND WITH_SLOTOMATIC=1] {          ← NUEVO guard
    printf '[ng-deploy] WARN --with-slotomatic ignored for mode B (ux-only)\n' >&2
  }
→ build_jars → [--no-deploy early exit]
→ copy_jars → verify_jar → verify_cachebuster
→ write_last_deploy_sha                         ← NUEVO (solo si NO --no-deploy)
→ print_restart_reminder → exit 0
```

### 3.2 Pseudocódigo del bloque insertado en `main()`

```bash
    # Step 2.4 (mode A/C only): detection heuristic + strict gate
    if [[ "$MODE" != "B" ]]; then
        if detect_annotation_changes; then
            # cambios detectados
            if [[ "$STRICT_SLOTOMATIC" -eq 1 || "${SLOTOMATIC_DETECTION:-warn}" == "strict" ]]; then
                die 15 "annotation changes detected since last deploy; --with-slotomatic required (strict mode)"
            elif [[ "$WITH_SLOTOMATIC" -eq 0 ]]; then
                warn_slotomatic_recommended
            fi
            # si WITH_SLOTOMATIC=1, no warn — ya lo van a correr
        fi

        # Step 2.5: opt-in slotomatic run
        if [[ "$WITH_SLOTOMATIC" -eq 1 ]]; then
            run_slotomatic
        fi
    elif [[ "$WITH_SLOTOMATIC" -eq 1 ]]; then
        # Mode B + --with-slotomatic = no-op informativo
        printf '[ng-deploy] WARN --with-slotomatic ignored for mode B (ux-only; slotomatic applies to -rt only)\n' >&2
    fi
```

### 3.3 Rationale de ordenamiento — 3 decisiones explícitas

**(a) ¿Por qué detection ANTES de la decisión de strict abort?**
Para que `--strict-slotomatic` aborte si hay cambios pero el operador olvidó `--with-slotomatic`. Si invirtiéramos (run primero, detect después), el deploy ya estaría a medio camino — strict perdería sentido.

**(b) ¿Por qué detection sigue corriendo aunque el operador pase `--with-slotomatic`?**
Es **idempotente y barato** (git diff + grep). Lo importante: cuando `WITH_SLOTOMATIC=1` y detection devuelve true, **NO emitimos WARN** (ya están corrigiendo el problema). Decision-table:

| detection | WITH_SLOTOMATIC | STRICT | Acción |
|---|---|---|---|
| no cambios | 0 | 0 | nada |
| no cambios | 1 | 0 | run_slotomatic (idempotente, expected) |
| no cambios | * | 1 | nada (strict no aplica sin findings) |
| cambios | 0 | 0 | WARN + continúa |
| cambios | 1 | 0 | run_slotomatic (sin warn) |
| cambios | 0 | 1 | **die 15** |
| cambios | 1 | 1 | run_slotomatic (sin warn; cumple la condición) |

**(c) ¿Por qué `write_last_deploy_sha` va DESPUÉS de verify?**
Si `verify_jar` o `verify_cachebuster` fallan con `die 50`, el SHA NO debe avanzar — el próximo deploy debe seguir viendo los cambios como "pendientes". Esto es el patrón estándar de "fallar atómico hacia adelante": el archivo de estado sólo refleja deploys exitosos.

**(d) Interacción con `--no-deploy`**
El early-exit de `--no-deploy` ocurre **después de `build_jars`** (línea 329 del actual). Decisión:

- `--no-deploy` corta el flujo **antes** de copy/verify/write_last_deploy_sha. ✓
- `--no-deploy` **NO inhibe** `run_slotomatic` ni `detect_annotation_changes` (esos corren antes de build_jars en el nuevo flujo). Razonamiento: `--no-deploy` es "build only para inspección" — un operador que pase `--with-slotomatic --no-deploy` quiere regenerar AUTO regions y compilar localmente, sin tocar la estación. Esto es coherente.
- `--no-deploy` **NO escribe `.last-deploy-sha`** (cortamos antes). Coherente: el SHA marca "última versión deployada", no "última versión buildeada".

---

## 4. Algoritmo de detección — pseudo-código final con edge cases

Ver §1.2. Edge cases cubiertos:

| Edge case | Comportamiento |
|---|---|
| `SLOTOMATIC_DETECTION=off` | return 1 silencioso |
| `SLOTOMATIC_DETECTION=garbage` | WARN stderr, treat as off, return 1 |
| `git` no en PATH | nota stderr, return 1 |
| CWD no es repo git | nota stderr, return 1 |
| `.last-deploy-sha` ausente | fallback `HEAD~1` |
| `.last-deploy-sha` con SHA orphaned/inexistente | fallback `HEAD~1` |
| `.last-deploy-sha` con tree/blob SHA (no commit) | fallback `HEAD~1` (gracias a `^{commit}`) |
| Sólo 1 commit en repo | `HEAD~1` falla silenciosamente → grep recibe vacío → return 1 (no cambios) |
| `git diff` exit != 0 por cualquier razón | `2>/dev/null` swallow → return 1 |
| Cambios en `*/srcTest/**` o `*/build/**` | filtrados por path glob `*/src/com/**/*.java` |
| Cambio solo en javadoc de `@NiagaraProperty` (no en la annotation) | NO matchea (grep exige `^[+-]\s*@Niagara`); falso negativo aceptable (R2 carry-forward) |
| Annotation movida de un archivo a otro | matchea ambas líneas (+ en nuevo, - en viejo) → return 0 (correcto: slotomatic necesita regenerar ambos) |

---

## 5. `read_baseline_sha()` — algoritmo final

Ver §1.3. Cambios respecto al draft de la proposal:

- Sustituí `cat ... | tr` por `tr -d ... < file` (un proceso menos, shellcheck-friendly).
- Añadí `${baseline}^{commit}` a `git cat-file -e` para rechazar SHAs no-commit.
- Sin `return 1` — siempre devuelve algo usable (HEAD~1). Esto simplifica el caller (no necesita branchear según return code).

---

## 6. `write_last_deploy_sha()` — algoritmo final

Ver §1.4. Diseño minimalista: silent-fail end-to-end. Si en el futuro queremos telemetría ("¿cuántas veces falla?"), añadiremos un counter — fuera de scope.

---

## 7. Arquitectura de tests bats

### 7.1 Tabla de tests nuevos

| # | Nombre | Mecanismo | Aserción |
|---|---|---|---|
| 18 | `--with-slotomatic prepends slotomatic task before build` | mode A + fake gradlew que appendea cada invocación a `gradlew.calls.log` | log contiene `:test-rt:slotomatic` ANTES de `:test-rt:clean` |
| 19 | `--with-slotomatic exits 15 when slotomatic gradlew fails` | mode A + `FAKE_GRADLEW_SLOTOMATIC_EXIT=1` (nuevo env var en stub) | `status -eq 15` y `stderr` contiene `slotomatic failed` |
| 20 | `detection emits WARN when annotations changed and no --with-slotomatic` | mode A + fake git stub con `FAKE_GIT_DIFF_OUTPUT="+@NiagaraProperty"` | `status -eq 0` (continúa) y `stderr` contiene `WARN` + `annotation changes detected` |
| 21 | `--strict-slotomatic aborts 15 when detection finds changes` | mode A + fake git + `--strict-slotomatic` sin `--with-slotomatic` | `status -eq 15` y `stderr` contiene `strict` |
| 22 | `mode B + --with-slotomatic emits skip WARN and does NOT run slotomatic` | mode B + `--with-slotomatic` + log gradlew invocations | `status -eq 0`, `stderr` contiene `ignored for mode B`, log NO contiene `:slotomatic` |
| 23 | `detection skip when git absent` | mode A + PATH sin git + `FAKE_GIT_DIFF_OUTPUT="+@NiagaraProperty"` (irrelevante) | `status -eq 0`, `stderr` contiene `git not available` |
| 24 | `.last-deploy-sha written after successful verify` | mode A full + fake git con `FAKE_GIT_REV_PARSE_OUTPUT=abc123` | `[[ -f "$TMPDIR_T/.last-deploy-sha" ]]` y contiene `abc123` |
| 25 | `.last-deploy-sha NOT written when verify fails` | mode A + `FAKE_UNZIP_TYPES=99` (mismatch → die 50) | `status -eq 50` y `[[ ! -f "$TMPDIR_T/.last-deploy-sha" ]]` |
| 26 | `SLOTOMATIC_DETECTION=off disables detection completely` | mode A + `SLOTOMATIC_DETECTION=off` + fake git con cambios | `status -eq 0`, `stderr` NO contiene `WARN` ni `annotation changes` |

**Nota**: la proposal compromete "T18-T22 ≥5 tests". Subimos a **9 tests nuevos** (T18-T26) porque cubren todos los edge cases definidos en §4 sin inflar el script. Mantiene el budget (~150 LOC bats vs 80 LOC bash) y respeta el patrón "una aserción ≈ un test".

### 7.2 Diseño del git fakebin

Añadir al `setup()` del bats (después del `tar` stub):

```bash
    # --- fake git ---
    # Dispatch by first arg:
    #   git diff <range> -- <pathspec>   → FAKE_GIT_DIFF_OUTPUT (default empty)
    #   git rev-parse HEAD               → FAKE_GIT_REV_PARSE_OUTPUT (default abc123)
    #   git rev-parse --git-dir          → exits 0 (we ARE in a fake repo)
    #   git cat-file -e <sha>{^{commit}} → FAKE_GIT_CAT_FILE_EXIT (default 0 = sha exists)
    # All other invocations: exit 0 silently (idempotent no-op)
    cat > "$TMPDIR_T/fakebin/git" << 'STUB'
#!/usr/bin/env bash
case "$1" in
    diff)
        printf '%s\n' "${FAKE_GIT_DIFF_OUTPUT:-}"
        exit 0
        ;;
    rev-parse)
        case "${2:-}" in
            --git-dir) exit 0 ;;
            HEAD)      printf '%s\n' "${FAKE_GIT_REV_PARSE_OUTPUT:-abc123}"; exit 0 ;;
            *)         exit 0 ;;
        esac
        ;;
    cat-file)
        exit "${FAKE_GIT_CAT_FILE_EXIT:-0}"
        ;;
    *)
        exit 0
        ;;
esac
STUB
    chmod +x "$TMPDIR_T/fakebin/git"
```

**Decisiones**:

- **No fallback a `command -v git` real**: el patrón ya es "PATH-injected fakebin que controla 100% del comportamiento". Mezclar con git real introduciría flakiness (depende del CWD bats real, qué commit existe, etc.). Coherente con `gradlew`/`unzip`/`tar`.
- **No requerimos `git init` real**: `rev-parse --git-dir` devuelve 0 incondicionalmente en el stub → `detect_annotation_changes` cree que está en un repo. Perfecto para unit tests.
- **Para el test "git ausente" (T23)**: el test override `PATH` con `export PATH="$TMPDIR_T/no-git-fakebin:$BASE_NON_GIT_PATH"` donde `no-git-fakebin` solo tiene gradlew/unzip/tar pero NO git. Helper sugerido en `setup()`:

```bash
    # alt PATH for tests that need git ABSENT (T23)
    mkdir -p "$TMPDIR_T/no-git-fakebin"
    cp "$TMPDIR_T/fakebin/gradlew" "$TMPDIR_T/no-git-fakebin/"
    cp "$TMPDIR_T/fakebin/unzip"   "$TMPDIR_T/no-git-fakebin/"
    cp "$TMPDIR_T/fakebin/tar"     "$TMPDIR_T/no-git-fakebin/"
    # NOTE: no git copy → git absent for tests that override PATH to this dir
```

### 7.3 Refactor del fake gradlew

El stub actual `printf '%s\n' "$@" > "${TMPDIR_T}/gradlew.args"` **sobrescribe** en cada invocación. Para T18 necesitamos ver **dos invocaciones** (slotomatic + build). Cambio:

```bash
    # --- fake gradlew ---
    cat > "$TMPDIR_T/fakebin/gradlew" << 'STUB'
#!/usr/bin/env bash
# Append every invocation to gradlew.calls.log (one line per call, args joined)
printf '%s\n' "$*" >> "${TMPDIR_T}/gradlew.calls.log"
# Last invocation also written to gradlew.args for backward-compat with tests 1-17
printf '%s\n' "$@" > "${TMPDIR_T}/gradlew.args"
# Per-task exit code override for slotomatic
for arg in "$@"; do
    case "$arg" in
        *:slotomatic) exit "${FAKE_GRADLEW_SLOTOMATIC_EXIT:-0}" ;;
    esac
done
exit "${FAKE_GRADLEW_EXIT:-0}"
STUB
```

**Compat**: tests 1–17 leen `gradlew.args` (última invocación). Como mode A/B/C sin `--with-slotomatic` sólo invocan gradlew una vez, `gradlew.args` sigue igual. Tests nuevos leen `gradlew.calls.log`. Cero regresión.

### 7.4 Asserción "order" (T18)

```bash
@test "T18: --with-slotomatic prepends slotomatic task before build" {
    run bash "$SCRIPT" \
        --env-file "$TMPDIR_T/dot_env_local" \
        --no-backup \
        --i-know-what-im-doing \
        --no-deploy \
        --with-slotomatic \
        --mode A
    [ "$status" -eq 0 ]
    # Línea 1 del log debe ser slotomatic, línea 2 debe ser el build
    run head -1 "$TMPDIR_T/gradlew.calls.log"
    [[ "$output" == *":test-rt:slotomatic"* ]]
    run sed -n '2p' "$TMPDIR_T/gradlew.calls.log"
    [[ "$output" == *":test-rt:clean"* ]]
}
```

Restricción: NO usamos `grep` / `cat` directamente sobre el log en la asserción del test — preferimos `run head` + `[[ ]]` para mantenerse en el patrón bats existente. (Anti-regla "no cat/grep" del CLAUDE.md global aplica a invocaciones del usuario en zsh, no a stubs internos; pero por consistencia usamos `head`/`sed` que es POSIX y existe en estaciones bare.)

### 7.5 Total de tests post-cambio

17 existentes + 9 nuevos = **26 tests**. Excede el comprimiso "22+" de la proposal por margen positivo.

---

## 8. Layout de actualizaciones de documentación

### 8.1 `print_usage()` — diff

Insertar nuevas Options entre `--no-deploy` y `--no-backup`:

```diff
   --env-file <PATH>        Source env from PATH instead of .env.local
   --no-deploy              Build only; do not copy or verify (jars stay in build/libs/)
+  --with-slotomatic        Run :MODULE-rt:slotomatic BEFORE clean+build (regenerates
+                             AUTO GENERATED CODE for @Niagara* annotation changes).
+                             No-op for --mode B (ux-only). Requires --mode A or C.
+  --strict-slotomatic      Make annotation-change detection FATAL: exit 15 if changes
+                             detected since last deploy and --with-slotomatic not passed.
   --no-backup              WARNING: skip backup step (dangerous — live station has no rollback)
```

En la sección "Optional", añadir env var:

```diff
 Optional:
   BUILD_ID  — when set, verifies index.html in ux jar contains ?v=$BUILD_ID
+  SLOTOMATIC_DETECTION  ∈ {warn|strict|off} (default: warn) — annotation-change
+    detection mode. 'warn' prints stderr advice; 'strict' aborts with exit 15;
+    'off' disables detection (useful for non-Niagara modules to avoid false positives).
```

En "Exit codes", insertar 15 entre 10 y 20:

```diff
 Exit codes:
   0   success (or --help)
   10  missing required env var or invalid path
+  15  slotomatic failed, or strict detection found annotation changes without --with-slotomatic
   20  backup failed, or --no-backup without --i-know-what-im-doing
```

### 8.2 `CLAUDE.md` §1 tabla — diff

**Antes** (línea 13):
```
| Slot / Property / Type / Action add or modify | A + run `:slotomatic` separately first | Yes |
```

**Después**:
```
| Slot / Property / Type / Action add or modify | A `--with-slotomatic` (or `:slotomatic` separately first) | Yes |
```

(Una línea, sin sub-bullets. La explicación full vive en `docs/knowledge-base/slotomatic.md` Card 4 — ver §8.4.)

### 8.3 `CHANGELOG.md` — entrada `[v0.3.0]` exacta

Insertar **ABOVE** `## [v0.2.0]` (per CONTRIBUTING.md §5 paso 2):

```markdown
## [v0.3.0] - 2026-05-18

### Added — `niagara-tools-slotomatic-integration`

- `scripts/ng-deploy.sh`: `--with-slotomatic` flag — runs `:MODULE-rt:slotomatic`
  BEFORE clean+build (Step 2.5 in the deploy flow). Reuses the 3 mandatory `-P`
  overrides validated in `wsl-build-gotchas.md`. No-op for `--mode B`.
- `scripts/ng-deploy.sh`: `--strict-slotomatic` flag — makes annotation-change
  detection fatal (exit 15 if detection finds `@Niagara*` changes and
  `--with-slotomatic` was not passed).
- `scripts/ng-deploy.sh`: passive heuristic `detect_annotation_changes()` —
  `git diff` since last successful deploy (or `HEAD~1` fallback) filtered by
  `*/src/com/**/*.java` and grepped for `@Niagara(Type|Property|Action|Topic|Singleton)`.
  Default mode: WARN to stderr; strict mode: abort.
- `scripts/ng-deploy.sh`: new exit code **15** — slotomatic failure OR strict
  detection findings.
- `scripts/ng-deploy.sh`: `.last-deploy-sha` state file written to `$(pwd)`
  AFTER successful verify (NOT on `--no-deploy`, NOT on verify failure). Used as
  baseline for the next `detect_annotation_changes()` run.
- Optional env var `SLOTOMATIC_DETECTION` ∈ `{warn|strict|off}` (default: `warn`).
- `.env.local.example` (CREATED — file was missing despite being referenced in
  README/CLAUDE.md): full config schema for 8 required vars + `BUILD_ID` +
  `SLOTOMATIC_DETECTION`, with `.gitignore` advice for `.last-deploy-sha`.
- `tests/ng-deploy.bats`: 9 new tests (T18–T26, 26 total). New `git` fakebin
  stub following the existing PATH-injected pattern. Refactored `gradlew` stub
  to append every invocation to `gradlew.calls.log` (backward-compat with the
  17 existing tests that read `gradlew.args`).
- `docs/knowledge-base/slotomatic.md`: new Card 4 — `--with-slotomatic` integration,
  citing chihuahua hotfix #1074 and slotomatic-success-without-edit gotcha #942.
- `docs/GOTCHAS.md`: new anti-pattern row — "Deploy with stale slotomatic" →
  fix: `--with-slotomatic` flag.

### Changed — `niagara-tools-slotomatic-integration`

- `CLAUDE.md` §1 decision table: the "Slot/Property/Type/Action" row now
  mentions `--with-slotomatic` as the preferred path (manual `:slotomatic`
  remains supported).

### References

- SDD slug: `niagara-tools-slotomatic-integration`
- Engram: explore #1924, proposal #1925, design (this commit topic_key
  `sdd/niagara-tools-slotomatic-integration/design`).
- Cross-project source: chihuahua hotfix #1074 (stale AUTO GENERATED CODE),
  chihuahua gotcha #942 (slotomatic SUCCESS without file edit).
- Tag: `v0.3.0` (this commit).
```

### 8.4 `docs/knowledge-base/slotomatic.md` — nueva Card 4

Insertar **antes** del footer `← Back to [GOTCHAS index]`:

```markdown
---

## Card 4: `--with-slotomatic` integration in `ng-deploy.sh` (v0.3.0+)

**Why this card exists**: Slotomatic-out-of-sync is a recurring class of bug.
Two confirmed incidents documented:
- **chihuahua hotfix #1074** (search `mem_search project:honeywell-mx60-chihuahua`):
  deploy after adding `@NiagaraProperty` without slotomatic → boot-loop
  `Type "chihuahua:ChiDashboardService" not found`. Required nocturnal rollback
  from `_backups/`.
- **chihuahua gotcha #942**: slotomatic gradle task exits SUCCESS without editing
  the file if the previous build had errors — disciplined operators still deployed
  stale AUTO regions.

`ng-deploy.sh` v0.3.0 closes both gaps with two new flags and a passive heuristic.

### When to use which flag

| Scenario | Recommended invocation |
|---|---|
| You know you changed `@Niagara*` annotations | `ng-deploy.sh --mode A --with-slotomatic` |
| You aren't sure if anything changed (most cases) | `ng-deploy.sh --mode A` and read stderr WARN |
| You want CI/strict gate (fail fast on stale) | `ng-deploy.sh --mode A --strict-slotomatic` (without `--with-slotomatic`, fails if detection finds changes) |
| Module is non-Niagara (no `@Niagara*` ever) | set `SLOTOMATIC_DETECTION=off` in `.env.local` |
| Mode B (ux-only changes) | flag is a no-op (slotomatic only applies to `-rt`); WARN emitted |

### How the heuristic works

`ng-deploy.sh` runs `git diff <last-deploy-sha>..HEAD -- '*/src/com/**/*.java'`
and greps for `@Niagara(Type|Property|Action|Topic|Singleton)` in the diff
output. When found AND `--with-slotomatic` was NOT passed, it emits a WARN
to stderr (or aborts in strict mode).

The baseline `<last-deploy-sha>` is read from `$(pwd)/.last-deploy-sha`, a file
ng-deploy.sh writes after a successful deploy. If the file is missing
(first-deploy), the baseline falls back to `HEAD~1`.

### Required gitignore entry (per consumer)

Each consumer module repo MUST add `.last-deploy-sha` to its `.gitignore`:
```
# ng-deploy.sh state — DO NOT commit
.last-deploy-sha
```
ng-deploy.sh cannot enforce this from upstream — it lives in the consumer's
working tree. The `.env.local.example` shipped with v0.3.0 includes a reminder
comment.

### Idempotency note

`run_slotomatic()` is idempotent when annotations are stable (see Card 3).
Running it twice in a row produces byte-identical output. So `--with-slotomatic`
on every deploy is safe (modulo build time cost). If you prefer to opt-in only
when needed, rely on the default WARN heuristic.

### Edge case: false positive on annotation-comment-only change

The grep pattern matches lines starting with `[+-]\s*@Niagara*`, so a pure javadoc
edit on a `@NiagaraProperty` field (no annotation line touched) will NOT trigger
detection. If you only edited comments above the annotation, the heuristic is
silent (correct: slotomatic doesn't need to rerun for comment-only changes,
though it would be a no-op anyway).

---

← Back to [GOTCHAS index](../GOTCHAS.md)
```

### 8.5 `docs/GOTCHAS.md` — nueva fila

Insertar tras la fila "Hand-editing AUTO GENERATED CODE region" (mantiene agrupación temática):

```markdown
| Deploy with stale slotomatic (annotation changed, AUTO region not regenerated) | Station boot-loop with `Type "..." not found` after a deploy that added/modified `@Niagara*` | `ng-deploy.sh --with-slotomatic` (or `--strict-slotomatic` for CI); detection heuristic warns by default since v0.3.0 | [slotomatic.md Card 4](knowledge-base/slotomatic.md) |
```

### 8.6 `.env.local.example` — contenido canónico (CREAR)

```bash
# ng-deploy.sh — consumer module config
# Copy this file to <module-root>/.env.local and fill in for your station.
# This file is the schema; .env.local is gitignored per .gitignore convention.

# -----------------------------------------------------------------------------
# REQUIRED — all 6 must be set for any mode (A/B/C)
# -----------------------------------------------------------------------------

# Module slug — used to assemble gradle task names :MODULE_NAME-rt:jar etc.
MODULE_NAME=chihuahua

# Absolute path to the consumer module's gradlew wrapper.
# WSL: use /mnt/c/... paths, NOT C:\... backslashes.
GRADLEW_PATH=/mnt/c/Users/you/path/to/chihuahua/gradlew

# Niagara installation root (the iC-Niagara-X.Y.Z directory).
NIAGARA_HOME=/mnt/c/Niagara/iC-Niagara-4.13.2.18

# Niagara user home (the per-user station registry directory).
# Quote-safe: ng-deploy.sh forwards this as "-Pniagara_user_home=$VAR" with
# proper escaping for paths containing spaces.
NIAGARA_USER_HOME=/mnt/c/Users/you/Niagara4.13/iSMA CONTROLLI

# JDK 8 (Niagara N4.13 requires Java 8).
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64

# Where ng-deploy.sh copies the built jars. This MUST be the live station's
# modules dir. Backup is created from THIS directory before each deploy.
STATION_MODULES_DIR=/mnt/c/Users/you/Niagara4.13/iSMA CONTROLLI/stations/myStation/shared/modules

# -----------------------------------------------------------------------------
# REQUIRED per-mode — verify_jar() type count assertion
# -----------------------------------------------------------------------------

# Number of <type entries expected in MODULE-rt.jar/META-INF/module.xml.
# Required for --mode A and --mode C. Bump when adding/removing BComponent types.
EXPECTED_RT_TYPES=9

# Number of <type entries expected in MODULE-ux.jar/META-INF/module.xml.
# Required for --mode A and --mode B.
EXPECTED_UX_TYPES=2

# -----------------------------------------------------------------------------
# OPTIONAL — cache-buster verification (v0.1.0+)
# -----------------------------------------------------------------------------

# If set, ng-deploy.sh asserts MODULE-ux.jar/rc/index.html contains ?v=$BUILD_ID.
# Use a fresh ID per ux deploy to bust browser caches.
# BUILD_ID=20260518a

# -----------------------------------------------------------------------------
# OPTIONAL — slotomatic detection (v0.3.0+)
# -----------------------------------------------------------------------------

# Controls passive heuristic that warns when @Niagara* annotations changed
# since the last successful deploy and --with-slotomatic was NOT passed.
#
# warn   → print WARN to stderr but continue (default; safest)
# strict → abort with exit 15 if changes detected and --with-slotomatic missing
# off    → disable detection entirely (use for non-Niagara modules)
SLOTOMATIC_DETECTION=warn

# -----------------------------------------------------------------------------
# IMPORTANT — add to consumer module's .gitignore
# -----------------------------------------------------------------------------
#
# ng-deploy.sh v0.3.0+ writes a state file `.last-deploy-sha` to $(pwd) after
# each successful deploy. This file is the baseline for the next detection run.
# It is per-working-tree state — do NOT commit it. Add to your .gitignore:
#
#   # ng-deploy.sh state
#   .last-deploy-sha
#
# This cannot be enforced from upstream (niagara-tools doesn't own consumer
# .gitignore files). Each consumer module must add the entry once.
```

---

## 9. Test gates — qué completa la fase apply

La fase `sdd-apply` se considera completa cuando todos los siguientes son verdaderos:

1. **bats**: `bats tests/ng-deploy.bats` exit 0, **26 tests** verdes (17 existentes intocables + 9 nuevos T18–T26).
2. **shellcheck script**: `shellcheck scripts/ng-deploy.sh` exit 0. Cualquier `# shellcheck disable=SCxxxx` nuevo debe traer comentario `# why: ...` adyacente (patrón existente líneas 92–93, 197–198, 213–214, 267–268).
3. **shellcheck bats**: `shellcheck tests/ng-deploy.bats` exit 0. El nuevo git fakebin heredoc se trata como string (no parseado por shellcheck) — no nuevas excepciones esperadas.
4. **Smoke checklist updated**: `tests/smoke-checklist.md` sección Mode A actualizada con un paso opcional `--with-slotomatic`, expected stderr WARN cuando se cumple detection, comportamiento de `.last-deploy-sha` documentado. (Mode B sección añade nota "ignored — slotomatic only applies to -rt".)
5. **VERSION**: `cat VERSION` retorna `0.3.0`.
6. **CHANGELOG**: entry `[v0.3.0]` presente y formato Keep a Changelog (per CONTRIBUTING.md §5).
7. **`.env.local.example`**: existe en repo root con el contenido de §8.6.
8. **KB topic**: `docs/knowledge-base/slotomatic.md` contiene Card 4.
9. **GOTCHAS row**: `docs/GOTCHAS.md` contiene la fila "Deploy with stale slotomatic".
10. **CLAUDE.md §1 row updated**: contiene `--with-slotomatic`.

**Manual gate** (no automatable, ejecutado en `verify` o post-merge):

- Smoke real en chihuahua: `ng-deploy.sh --mode A --with-slotomatic` → slotomatic corre antes de build, exit 0, verify_jar pasa, estación arranca sin `Type ... not found`.

---

## 10. Riesgos re-evaluados con profundidad de diseño

Carry-forward de R1–R10 de proposal, refinados:

| ID | Riesgo | Mitigación de diseño (refinada) |
|---|---|---|
| R1 | HEAD~1 fallback produce false negatives en ciclos multi-commit | `.last-deploy-sha` es la fuente primaria; HEAD~1 sólo en first-deploy o si el archivo se borra. Documentado en `.env.local.example` §"IMPORTANT". |
| R2 | False positives por cambios de comentario en `@NiagaraProperty` | Grep ahora exige `^[+-]\s*@Niagara` — sólo matchea **la línea de annotation**, no su javadoc. Falsos positivos restantes (e.g., commit que añade `@NiagaraProperty` y luego lo revierte en el mismo range) son aceptables: WARN no aborta. |
| R3 | `.last-deploy-sha` debe ir al `.gitignore` del consumer | `.env.local.example` §"IMPORTANT" + `docs/knowledge-base/slotomatic.md` Card 4 + `docs/GOTCHAS.md` row. Tres puntos de visibilidad. No enforce; documentación es el único vector. |
| R4 | Slotomatic con `-P` overrides validado | `run_slotomatic` reusa **literalmente** los 3 `-P` de `build_jars`. Mismo orden, mismo quoting. Riesgo cerrado por construcción. |
| R5 | Mode B + `--with-slotomatic` debe ser WARN+skip | Implementado en §3.2: `elif MODE==B && WITH_SLOTOMATIC==1` → printf WARN, no run. Test T22 cubre. |
| R6 | Concurrent deploys racing en `.last-deploy-sha` | Aceptado. Doc en KB Card 4. File locking fuera de scope. |
| R7 | Consumers sin `SLOTOMATIC_DETECTION` default a `warn` | Default `${SLOTOMATIC_DETECTION:-warn}` en `detect_annotation_changes`. Test T26 verifica `off`. Non-breaking. |
| R8 | `.env.local.example` no existe — debe crearse | §8.6 contenido completo. Task explícita en `tasks.md`. Bloquea archive si falta. |
| R9 | `git` puede no estar en estaciones bare | `command -v git` guard en `detect_annotation_changes` y `write_last_deploy_sha`. Test T23 cubre. Operador con `--with-slotomatic` explícito NO requiere git (slotomatic invoca gradlew directo). |
| R10 | Slotomatic no-idempotente bajo concurrencia | Card 4 §"Idempotency note" cita Card 3 (idempotente cuando annotations stable). Locking fuera de scope. |
| **R11 (NEW)** | Refactor del fake gradlew rompe tests 1-17 si el cambio no es backward-compat | `gradlew.args` se sigue escribiendo (última invocación) para compat. `gradlew.calls.log` es additive. Test 6 (mode A → "rt" + "ux") sigue verde porque mode A sin slotomatic = una sola invocación = `gradlew.args` igual que antes. |
| **R12 (NEW)** | `.last-deploy-sha` queda obsoleto si operador hace `git reset --hard` post-deploy | Aceptado. Próximo deploy: `read_baseline_sha` valida con `cat-file -e ^{commit}` — si el SHA fue rebased away, cae a `HEAD~1`. Degradación graciosa. |
| **R13 (NEW)** | Heredoc del git fakebin en bats puede fallar shellcheck por SC2086 en `case "$1"` | El stub es heredoc `'STUB'` (single-quoted) — shellcheck NO parsea contenido. Cero falsos positivos. |

---

## 11. Open questions resueltas in-line (no estaban en proposal)

### OQ-D1: ¿`detect_annotation_changes` debe correr en `--no-deploy`?

**Sí.** `--no-deploy` es "build sin deploy" — el operador típico la usa para verificar que un cambio compila. Si la heurística detecta cambios, queremos avisar **igual** (warn-then-no-deploy es informativo, no fatal). Coherente con §3.3-(d).

### OQ-D2: ¿`run_slotomatic` debe correr en `--no-deploy --with-slotomatic`?

**Sí.** Si el operador pasa ambos, quiere "regenerar AUTO regions + compilar localmente, sin tocar la estación". Caso válido para pre-commit local checking.

### OQ-D3: ¿Strict mode debe abortar si `SLOTOMATIC_DETECTION=off` ignora detection?

**No** (no aborta). Si `SLOTOMATIC_DETECTION=off`, `detect_annotation_changes` retorna 1 (no-cambios). El check de strict (`elif STRICT_SLOTOMATIC && detect-returned-0`) entonces NO se dispara. Coherente: si el operador apagó detection a propósito, strict respeta esa decisión. Si quiere strict sin off, no setea `SLOTOMATIC_DETECTION=off`.

### OQ-D4: ¿Dónde escribimos `.last-deploy-sha` cuando el operador corre desde un dir != consumer module root?

**`$(pwd)`** (CWD del operador, no `dirname GRADLEW_PATH`). Razonamiento: la heurística de detección también corre git diff desde `$(pwd)` — el baseline y el diff deben estar en el mismo working tree. Si el operador corre `ng-deploy.sh` desde un dir random, el archivo aparece allí y la próxima vez funcionará si vuelve al mismo dir; de lo contrario, fallback a HEAD~1. Aceptable: el patrón documentado en CLAUDE.md §1 es `cd /path/to/chihuahua && /path/to/niagara-tools/scripts/ng-deploy.sh`.

### OQ-D5: ¿`STRICT_SLOTOMATIC=1` debe implicar `SLOTOMATIC_DETECTION≠off`?

**No.** Cero magia. Si el operador setea ambos en oposición (`SLOTOMATIC_DETECTION=off` + `--strict-slotomatic`), strict no se dispara (porque detection retorna 1). Es un caso degenerado del propio operador — no es nuestro job rescatarlo. Documentar el behaviour en `--help` Optional section es suficiente.

### OQ-D6: ¿Tests bats deben tocar git real?

**No.** Toda la línea de tests usa el git fakebin (§7.2). Tests con git real serían integration tests, no unit tests — viven en `tests/smoke-checklist.md` (manual gate post-implement).

### OQ-D7: ¿`write_last_deploy_sha` corre para mode B?

**Sí**, después de un deploy mode B exitoso. Razonamiento: el SHA marca "última versión deployada" — independiente de qué modo se usó. Próximo deploy en mode A/C usará ese SHA como baseline para detection y verá los commits ux-only como "sin cambios de @Niagara*", correctamente devolviendo 1 (no cambios). Consistente.

---

## 12. Resumen de archivos tocados

| Archivo | Acción | Tipo de cambio |
|---|---|---|
| `scripts/ng-deploy.sh` | edit | +6 funciones, +2 globals, +2 cases parse_args, +bloque slotomatic en main, +update print_usage |
| `tests/ng-deploy.bats` | edit | +1 git fakebin, +1 no-git-fakebin helper, refactor gradlew stub, +9 tests (T18–T26) |
| `.env.local.example` | create | nuevo archivo, ~80 LOC |
| `VERSION` | edit | `0.2.0` → `0.3.0` |
| `CHANGELOG.md` | edit | nueva sección `[v0.3.0]` arriba de `[v0.2.0]` |
| `CLAUDE.md` | edit | una fila §1 actualizada |
| `docs/GOTCHAS.md` | edit | una nueva fila en antipatterns table |
| `docs/knowledge-base/slotomatic.md` | edit | nueva Card 4 antes del footer |
| `tests/smoke-checklist.md` | edit | sección Mode A + nota Mode B sobre `--with-slotomatic` |

**Total**: 9 archivos. 1 nuevo, 8 edits. ~350 LOC neto.

---

## 13. Next

`sdd-tasks` — desglose en steps TDD ordenados, work-unit commits, respetando el orden sugerido en proposal §7: T18–T26 red → parse_args+usage → run_slotomatic+main slot → detect_annotation_changes + read_baseline_sha → write_last_deploy_sha → shellcheck pass → docs + CHANGELOG + VERSION → manual smoke.
