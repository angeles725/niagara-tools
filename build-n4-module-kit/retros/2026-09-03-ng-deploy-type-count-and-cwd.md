# Retro — dos gotchas PROBADOS de `ng-deploy.sh`: conteo de tipos y cwd de gradle · 2026-09-03

Cuarto retro del día (previos: `2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md`,
`2026-09-03-soft-start-staggered-startup.md`, `2026-09-03-dashboard-contract-port-spec.md`). Este tramo
SÍ deployó a la Honeywell (ColdRoomPan-rt + DashboardPan-rt/-ux, backup + gate ALL PASS). Dos fricciones
del wrapper `scripts/ng-deploy.sh` costaron reintentos; ambas son PROPOSED kit deltas (propose-never-apply).
Sólo lo NUEVO.

## What this PROVED

1. **`EXPECTED_RT_TYPES` / `EXPECTED_UX_TYPES` de `ng-deploy.sh` NO cuentan los tipos reales — cuentan
   `grep -c "<type"` sobre `META-INF/module.xml`, que TAMBIÉN matchea el contenedor `<types>`. Hay que
   pasar (tipos reales + 1).** El deploy abortó con `expected 6 <type entries ... found 7` en un jar cuyo
   `module-include.xml` declara exactamente 6 tipos. Causa raíz verificada: el `module.xml` del jar tiene
   `<types>` (contenedor) en una línea + 6 `<type .../>` → `grep -c "<type"` = 7. `verify-module.sh` del
   gate en cambio cuenta `grep -c "<type "` (con espacio) = 6, por eso el gate daba PASS y ng-deploy no.
   Valores reales de esta corrida: ColdRoomPan-rt = **7** (6 tipos + contenedor), DashboardPan-rt = **3**
   (2 + 1), DashboardPan-ux = **2** (1 + 1). El copy YA había ocurrido antes del verify (el jar quedó bien
   deployado); el fallo era sólo el conteo off-by-one.
   → **PROPOSED build-verify.md delta:** documentar que los `EXPECTED_*_TYPES` de ng-deploy = (tipos en
   `module-include.xml`) + 1 por el `<types>` wrapper, y que difieren del conteo de `verify-module.sh`
   (`<type ` con espacio = tipos reales). Regla práctica: contar con
   `unzip -p <jar> META-INF/module.xml | grep -c "<type"` para obtener el número que ng-deploy espera.
   (Mejor aún, arreglo de fondo para el kit: que ng-deploy use `grep -c "<type "` con espacio y así el
   valor esperado sea el nº de tipos real — quitar la trampa del contenedor.)

2. **`ng-deploy.sh` corre `gradlew` desde el CWD, no desde el dir de `GRADLEW_PATH`.** Invocarlo desde el
   directorio del repo de investigación falló con `Directory '...' does not contain a Gradle build`.
   `GRADLEW_PATH` se usa sólo para derivar el `src_dir` de la copia (`dirname`), no para elegir el dir de
   ejecución de gradle. Hay que `cd` a la raíz gradle real (donde vive `gradlew` + `settings.gradle.kts`)
   ANTES de invocar ng-deploy. En este cliente esas raíces son `Paccadia/` y `Dashboard/` (cada una un
   proyecto multi-módulo con su propio wrapper), NO la raíz `Leon-Guanjuato`.
   → **PROPOSED BUILD-LOOP.md §6 (deploy) delta:** anotar el preflight "cd a la raíz gradle (la del
   `gradlew`/`settings.gradle.kts`) antes de `ng-deploy.sh`". Complementa el mismo hallazgo de `build.sh`
   (que sí resuelve por su cuenta): en `build.sh` el arg es `<module-root>` = la raíz gradle; en ng-deploy
   hay que estar parado en ella.

## Cost / evidence
- **Delta 1** evidence: `[ng-deploy] ERROR verify failed: expected 6 <type entries ... found 7`; luego
  `unzip -p ColdRoomPan-rt.jar META-INF/module.xml | grep -c "<type"` = 7 vs `grep -c "<type "` = 6 (línea
  `<types>` + 6 `<type .../>`). Reejecución con `EXPECTED_RT_TYPES=7` → `verify ok (7/7)` + gate ALL PASS.
- **Delta 2** evidence: primer intento desde `/home/cristian/niagara-research` → gradle `does not contain a
  Gradle build`; segundo intento tras `cd .../Paccadia` → build + copy + gate OK.

## Nota de alcance
Ningún cambio de código de módulo salió de este tramo; los dos deltas son de tooling/doc (build-verify.md +
BUILD-LOOP.md §6). El deploy en sí fue correcto: backup por módulo (`_backups/*.tar.gz`), copy a
`OptimizerSupervisor-N4.14.0.162/modules/`, gate ALL PASS, "station restart required". Station detenida al
momento (sólo `niagarad.exe`), así que la copia no chocó con lock del jar.
