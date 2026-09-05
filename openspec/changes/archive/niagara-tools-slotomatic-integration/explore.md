# Exploración — niagara-tools-slotomatic-integration

**Fecha**: 2026-05-18
**Fase**: explore
**Estado**: completo
**Engram**: observación #1924 (topic_key `sdd/niagara-tools-slotomatic-integration/explore`)

---

## Estado actual

`scripts/ng-deploy.sh` (v0.2.0, 368 LOC) ejecuta `backup → build → copy → verify` vía `build_jars()` que invoca `${GRADLEW_PATH}` con los 3 `-P` overrides ya en su lugar:

```bash
-Pniagara_home="${NIAGARA_HOME}"
"-Pniagara_user_home=${NIAGARA_USER_HOME}"
-Porg.gradle.java.installations.paths="${JAVA_HOME}"
```

Las tasks se arman por modo (A/B/C) como `:MODULE-rt:clean :MODULE-rt:jar` etc. Slotomatic se insertaría como `:MODULE-rt:slotomatic` corriendo ANTES de clean+jar, en un step dedicado que corre solo para rt (slotomatic solo aplica a BComponent classes en el subproject -rt).

**El script NO tiene lógica de slotomatic hoy.** CLAUDE.md §1 reconoce el gap con la fila "Slot / Property / Type / Action add or modify → A + run `:slotomatic` separately first".

No existe `.env.local.example` en el repo (está referenciado en README y CLAUDE.md pero no fue creado). Gap a cerrar — el nuevo `SLOTOMATIC_DETECTION` se agrega cuando se cree el archivo (no es "agregar una var", es "crear el archivo entero").

## Mapa de exit codes (actual)

- 0 success
- 10 env/arg validation
- 20 backup
- 30 build
- 40 copy
- 50 verify

**Gap entre 10 y 20: exit 15 disponible para slotomatic failure.**

## Áreas afectadas

- `scripts/ng-deploy.sh` — agregar flags `--with-slotomatic`, `--strict-slotomatic`; función `run_slotomatic()`; función `detect_annotation_changes()`; extender `parse_args()` y `main()`
- `tests/ng-deploy.bats` — +5 tests (hoy 17 tests, agregar 18–22+)
- `scripts/ng-deploy.sh` print_usage — documentar nuevos flags + exit 15
- `.env.local.example` (debe CREARSE primero — archivo ausente pese a referencias) — agregar `SLOTOMATIC_DETECTION=warn` opcional
- `CLAUDE.md` §1 tabla — actualizar fila "Slot/Property/Type/Action" mencionando `--with-slotomatic`
- `CHANGELOG.md` + `VERSION` — bump 0.2.0 → 0.3.0 (nuevo flag = MINOR per SemVer)
- `docs/knowledge-base/slotomatic.md` — agregar card `--with-slotomatic`
- `docs/GOTCHAS.md` — agregar anti-pattern row "deploy with stale slotomatic" + fix

## Decisiones arquitectónicas

### 1. Punto de inserción de slotomatic

¿Antes de `backup`? ¿Entre `backup` y `build`?

Slotomatic modifica archivos source (Java AUTO GENERATED CODE regions). DEBE correr ANTES de clean+build para que el source regenerado se compile en el jar. Backup captura el jar deployado actual (de station, no source), entonces backup state es irrelevante a cambios de source de slotomatic.

**Decisión**: Insertar slotomatic DESPUÉS de backup (backup = snapshot de jar deployado, no source), ANTES de build. Es "Step 2.5" en el flow: `backup → slotomatic (opt-in) → build → copy → verify`.

Slot en `main()`: entre la llamada `backup` y la llamada `build_jars`.

### 2. Implementación de `--with-slotomatic`

Nueva función `run_slotomatic()`:

```bash
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

Llamada desde `main()` cuando `WITH_SLOTOMATIC=1`.

### 3. Heurística de detección (opción b)

Tres candidatos evaluados:

| Opción | Pros | Cons | Veredicto |
|---|---|---|---|
| **b1 — comparar contra HEAD~1** | Zero-setup, no extra files | FALLA en ciclos multi-commit (operador hace 3 commits, último deploy 3 commits atrás → b1 solo mira 1 atrás); falsos negativos comunes | TOO UNRELIABLE |
| **b2 — `.last-deploy-sha` file escrito por ng-deploy.sh on success** | Preciso — trackea último deploy exitoso real independiente del count de commits; no requiere commit message discipline | File per-consumer en working-tree (debe `.gitignored`); concurrent deploys (raro) pueden race; introduce state file per-run | **MOST RELIABLE** pero tiene gitignore distribution problem |
| **b3 — git log buscando commit `chore(slotomatic): regen`** | No extra files; leverages git history | Requires commit message discipline; falsos negativos si operador commiteó slotomatic sin convención; extremadamente frágil | FRAGILE, REJECT |

**Recomendación para (b)**: usar **b2** (`.last-deploy-sha`) con documentación clara. Escribir a `$(pwd)/.last-deploy-sha` on successful deploy. Si el archivo no existe, fallback a HEAD~1 para baseline del diff (first-deploy safety). Documentar `.gitignore` requirement prominentemente en `.env.local.example` y smoke-checklist.

Pattern del diff:

```bash
git diff "${baseline}..HEAD" -- '*/src/com/**/*.java' 2>/dev/null \
  | grep -E '@Niagara(Type|Property|Action|Topic|Singleton)' \
  | grep -v '^---' | grep -v '^+++' > /dev/null 2>&1
```

(POSIX `grep` no `rg` — scripts deben correr en bare stations per Compact Rules)

Cuando hay annotation changes detectados Y slotomatic NO pasado:
- Default (`SLOTOMATIC_DETECTION=warn` o no var): emitir WARN a stderr, continuar
- `--strict-slotomatic` o `SLOTOMATIC_DETECTION=strict`: `die 15 "..."`

### 4. Gap de `.env.local.example`

Archivo ausente. Proposal/spec debe incluir crearlo. Nueva var opcional:

```bash
# Optional: slotomatic annotation-change detection
# warn   = print warning but continue (default)
# strict = abort with exit 15 if annotation changes detected and --with-slotomatic not passed
# off    = disable detection entirely
SLOTOMATIC_DETECTION=warn
```

### 5. Backward compatibility

- Todos los paths `--mode A|B|C` existentes: sin cambios. Slotomatic es puramente additive.
- `WITH_SLOTOMATIC` global default 0. `STRICT_SLOTOMATIC` default 0.
- `SLOTOMATIC_DETECTION` env default `warn` cuando no está seteado (safe — warnings no rompen builds).
- Tests 1–17 sin afectar; todos deben quedar verdes.

### 6. Solo `--mode A|C` corre slotomatic (mode B guard)

Slotomatic solo aplica a subprojects `-rt` (BComponent Java). Mode B es ux-only. Si `--with-slotomatic` pasa con `--mode B`, el script debe emitir WARN y skip slotomatic (no abort) — es no-op para cambios ux, no es error. Heurística de detección también skipped para mode B.

### 7. Infraestructura BATS para tests git-state-dependent

`tests/ng-deploy.bats` usa `$TMPDIR_T` (mktemp) con PATH-injected fakebins. Tests git-diff-dependent necesitan fake git repo en `$TMPDIR_T`. Pattern:

```bash
# En setup() o helper:
git init "$TMPDIR_T/fake-repo"
cd "$TMPDIR_T/fake-repo"
git config user.email "test@test.com"
git config user.name "Test"
mkdir -p src/com/test
printf '@NiagaraProperty\n...' > src/com/test/BFakeComponent.java
git add . && git commit -m "initial"
printf '@NiagaraProperty @NiagaraAction\n...' > src/com/test/BFakeComponent.java
```

Alternativa: mock `git` con fakebin stub (como el stub de gradlew) que outputs canned diff content. Evita real git repo setup y es simpler para unit tests — matchea el pattern existente.

**Recomendación**: mock `git` como fakebin para unit tests; usar real git repo fixture solo para smoke-checklist integration tests.

## Impact de version

v0.2.0 → v0.3.0 (MINOR: new flags `--with-slotomatic`, `--strict-slotomatic`; new optional env var `SLOTOMATIC_DETECTION`; new exit code 15; new `.env.local.example` file).

## Riesgos

| ID | Riesgo | Mitigación |
|---|---|---|
| R1 | b1 (HEAD~1) fallback produce falsos negativos para ciclos multi-commit | Baseline b2 (`.last-deploy-sha`) debe ser absent-safe para failar gracefully on first deploy |
| R2 | Falsos positivos de heurística si COMMENTS de annotations cambiaron pero no la annotation en sí (ej. javadoc on `@NiagaraProperty`) | Aceptado low-signal-noise trade-off — WARN no aborta |
| R3 | `.last-deploy-sha` debe agregarse a `.gitignore` de cada consumer | No garantizable desde niagara-tools. Mitigar vía `.env.local.example` advice + GOTCHAS.md row |
| R4 | Slotomatic con `-P` overrides está validado (engram #1402, wsl-build-gotchas.md Card 4). `run_slotomatic()` reusa los mismos 3 overrides ya en `build_jars()` | Sin riesgo |
| R5 | Mode B + `--with-slotomatic`: debe guardearse explícito | Documentar como WARN + skip, no error |
| R6 | Concurrent deploys racing on `.last-deploy-sha`: file-locking no warranted para solo operator | Accept la race condition (documentar como known limitation) |
| R7 | Consumers sin `SLOTOMATIC_DETECTION` en `.env.local` default a `warn` (non-breaking). Var es opcional y tiene default safe | OK |
| R8 | `.env.local.example` no existe — este SDD debe crearlo. Si proposal/spec se olvida, consumer onboarding queda roto | Spec debe incluir creación del archivo |

## Out-of-scope (confirmado)

- `ng-deploy.ps1`: no existe, SDD separado
- Chihuahua pin bump: operator ritual post-land
- Slotomatic gradle task internals: owned by Tridium plugins, no changes needed
- Mode B slotomatic execution: guard + skip (no implement)

## Listo para propuesta

Sí. Pre-decisión del operador (a)+(b) combo confirmada como sólida. Sin contradicciones en el codebase. Implementation space bien definido, punto de inserción claro, backward compatibility confirmada. Proceder a `sdd-propose`.
