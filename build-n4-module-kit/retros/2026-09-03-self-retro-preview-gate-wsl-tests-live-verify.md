# Retro (mejora del proceso /build-n4-module) — preview-gate para UI · correr tests puros en WSL · verificación en vivo · 2026-09-03

Self-retro del BUILD-LOOP tras la sesión larga del deshielo/interlock/freeze de PANCCADIA (ColdRoomPan + CompPan + DashboardPan). PROPOSED kit deltas (propose-never-apply). Sólo lo NUEVO respecto a los retros técnicos del día.

## Qué del PROCESO funcionó y hay que formalizar

1. **Preview-approval como GATE obligatorio para cambios `-ux`** `[ev: esta sesión, repetido]`.
   El flujo que funcionó una y otra vez: editar `index.html` + ajustar el mock del `preview-server.py` → levantar `preview-server.py <port>` → operador prueba en `http://localhost:<port>/dashboardpan/hmi` y APRUEBA → recién entonces `build.sh`. Evitó compilar UI no aprobada en cada iteración (LED, interlock, countdown, no-scroll). La lógica `-rt` no tiene preview (se aprueba por diseño + tests).
   → **PROPOSED BUILD-LOOP.md §3 delta:** para cualquier cambio con superficie `-ux`, el preview + OK explícito del operador es un GATE previo al build, no opcional. Incluir en el mock el estado que dispara el comportamiento nuevo (p. ej. `Cuarto3/evapNValveState` para probar los LEDs de salida). Recordar la limitación: el mock NO ejecuta la lógica `-rt`, así que el preview aprueba el comportamiento del DASHBOARD, no el efecto físico — decirlo al operador.

2. **Correr los tests puros EN WSL — el comando exacto (el kit dice "hazlo" pero no cómo)** `[CERT, esta sesión]`.
   METHODOLOGY dice "niagaraTest no corre en WSL; unit-test el modelo puro con JUnit" pero no da el comando, y sí se puede. Receta probada (ColdRoomControl 22 tests, <0.05s):
   ```
   JUNIT=~/.gradle/wrapper/dists/gradle-7.6-bin/*/gradle-7.6/lib/junit-4.13.2.jar
   HAM=~/.gradle/caches/modules-2/files-2.1/org.hamcrest/hamcrest-core/1.3/*/hamcrest-core-1.3.jar
   javac -d OUT -cp "$JUNIT:$HAM" <ClasePura>.java <ClasePuraTest>.java   # test en el MISMO package para acceso package-private
   java -cp "OUT:$JUNIT:$HAM" org.junit.runner.JUnitCore com.angeles.<Mod>.<TestClass>
   ```
   Requiere que la clase de control sea PURA (sin imports de Baja) y el test en su mismo package. `rm -rf` del OUT puede gatillar prompt de permiso — usar un OUT nuevo por corrida en vez de borrar.
   → **PROPOSED build-verify.md delta:** agregar esta receta + la nota "clase pura + test same-package". Correr los tests puros ANTES del build cuando se toca la lógica de control es barato y agarra regresiones (aquí: al cambiar `applyHoa`, los 21 tests confirmaron que no rompí nada y el test nuevo protegió el fix).

3. **Verificación RUNTIME contra el sistema vivo — el código correcto NO garantiza el comportamiento** `[ev: freeze-stat + timer]`.
   El `applyHoa`/freeze-stat y el `armTrigger` se LEÍAN correctos en el source y aun así no funcionaban en la station (bug de AUTO-ignora-inhibit; timer no armado en mount tardío). El oráculo fue leer el estado en vivo (oBIX/Slot Sheet), no releer el código.
   → **PROPOSED BUILD-LOOP.md §6 delta:** tras el deploy, añadir un paso "verificar el COMPORTAMIENTO en vivo" (leer el slot-ancla / la salida por oBIX o Slot Sheet), distinto de "el jar cargó". Un slot derivado leído en vivo (ancla null, salida OFF/ON) arbitra "lógica mala" vs "no se disparó".

4. **Distinguir bug COSMÉTICO de causa raíz antes de invertir esfuerzo** `[ev: applyRunCmd]`.
   El `WARNING NotRunningException` de `applyRunCmd` parecía la causa del deshielo muerto y NO lo era (lo atrapa `changed()`). Se perdió tiempo hasta separar el ruido de la causa (montaje tardío sin `started()`).
   → **PROPOSED METHODOLOGY.md delta:** ante un WARNING en el log, primero clasificar atrapado-y-cosmético vs propaga-y-aborta (¿el `changed()`/`try` lo traga?) ANTES de tratarlo como causa raíz.

## Referencias
- Retros técnicos hermanos del día: `self-firing-timer-needs-started-not-only-atsteadystate.md`, `hidden-actions-not-invocable-and-runtime-anchor-verification.md`, `station-corre-en-atlas-snap-no-en-el-pc-de-deploy.md`, `qa-stack-pure-tests-and-defrost-untested-gap.md`.
- Bitácora del episodio: `Cliente/Leon-Guanjuato/bitacora/2026-09-03-deshielo-cuarto3-no-arma-diagnostico-multi-sesion.md`.
