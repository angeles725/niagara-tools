# Retro — un BComponent con timer auto-armado necesita `started()`, no solo `atSteadyState()`, o NUNCA arma en commissioning · 2026-09-03

Décimo retro del día. Un `BDefrostController` (ColdRoomPan-rt) con `Modo = Intervalo` que "nunca entra" al
deshielo — reportado por el operador y confirmado en vivo. Causa raíz: arma su ticket de `Clock` **solo en
`atSteadyState()`**. [CERT] docSource + [CERT-live]. PROPOSED kit delta (propose-never-apply). Sólo lo NUEVO.
Investigación completa en el corpus niagara-research: **B729**.

## What this PROVED

1. **`atSteadyState()` es un callback SOLO de bootstrap; `started()` corre cada vez que el componente pasa a
   running.** Verbatim `BComponent.java` (docSource) [CERT]:
   - `started()` (:333-341): *"called when a component's running state moves to true. Components are started
     top-down, children after their parent."*
   - `atSteadyState()` (:380-384): *"invoked during station bootstrap after the steady state timeout has
     expired."* (timeout default 10s, `niagara.steadystate`.)
   → Un componente **montado/movido/habilitado en una station YA corriendo (commissioning)** recibe
   `started()` pero **NO** `atSteadyState()`. Si toda la lógica de arranque de su timer vive en
   `atSteadyState()`, **nunca se ejecuta para ese instance → el ticket nunca se arma → la acción periódica
   nunca dispara. Sin error: el componente queda mudo.** Encaja 100% con el síntoma real: el operador colgó
   el `BDefrostController` bajo el cuarto con la station ya en steady-state; el intervalo jamás disparó.

2. **El patrón canónico de Tridium (`BTimeTrigger`, `javax.baja.control.trigger`) override TRES hooks, no
   uno.** [CERT] `BTimeTrigger.java` docSource:
   ```java
   public void atSteadyState() { if (isRunning()) init(); }          // :213  boot
   public void started()       { if (Sys.atSteadyState()) init(); }  // :221  montaje tardío
   public void clockChanged(BRelTime shift) { init(); }              // :294  ajuste de reloj
   ```
   La guarda `started(){ if(Sys.atSteadyState()) init(); }` da armado **exactamente-una-vez** across ambos
   caminos: en boot, `started()` corre ANTES del steady-state → `Sys.atSteadyState()` es false → no-op (arma
   `atSteadyState()`); en montaje tardío, la station ya está steady → arma. Además esa misma guarda evita
   programar `Clock` antes de que el engine acepte schedules (el `NotRunningException` de boot).

   → **PROPOSED types/rt-patterns delta (regla dura):** **Todo `BComponent` que auto-arme un `Clock.schedule`
   / `schedulePeriodically` (timer "hacer X cada N", polling, watchdog) DEBE override `started()` además de
   `atSteadyState()`**, con el idiom de Tridium:
   ```java
   @Override public void started() throws Exception { super.started(); if (Sys.atSteadyState()) arm(); }
   @Override public void atSteadyState() { if (isRunning()) arm(); }
   @Override public void clockChanged(BRelTime shift) { try { arm(); } catch (Throwable t){ logError("clockChanged", t);} } // opcional pero barato
   ```
   Anti-patrón a marcar en el checklist rt: **"timer armado SOLO en `atSteadyState()`"** → falla silenciosa
   en commissioning.

3. **Ordenar `setNextX()` ANTES de `Clock.schedule()`, no después.** En `armTrigger()` del BDefrostController
   el `Clock.schedule(...)` estaba antes de `setNextDefrostTime(...)`. Si el schedule lanza (p. ej.
   `NotRunningException` en boot vía `activateLinks`), el ancla de display nunca se setea → queda null Y el
   ticket sin armar. Mismo acoplamiento que tumbó `BEvaporatorUnit.applyRunCmd` (link-target `runCmd`
   propagado en `activateLinks` → `Clock.schedule` → `NotRunningException`, tragado por el try/catch de
   `BColdRoom.changed`, cosmético pero real). → **PROPOSED:** setear los slots de estado/ancla ANTES de
   programar el ticket; y guardar todo `Clock.schedule` fuera de `atSteadyState/started` con
   `if(!Sys.atSteadyState()) return;` cuando el método pueda alcanzarse vía `activateLinks`.

4. **`schedulePeriodically` lanza si `period ≤ 0`.** [CERT] `Clock.java:223` (`IllegalArgumentException if
   period less than or equal to zero`). Si se migra de one-shot+re-arm a periódico, validar el período > 0
   antes (config de usuario puede venir en 0). `Clock.millis()` = `System.currentTimeMillis()` (epoch), por
   eso un `now-last` es válido pero expuesto a saltos de reloj → de ahí `clockChanged`.

## Net delta propuesto al kit (propose-never-apply)

- **Checklist rt (nuevo ítem):** para cualquier componente con timer auto-armado, exigir `started()` +
  `atSteadyState()` (+ `clockChanged` opcional), con el idiom `Sys.atSteadyState()`. Anti-patrón:
  "solo `atSteadyState()`".
- **Regla de orden:** `setSlot(...)` de anclas/estado ANTES de `Clock.schedule(...)`.
- **Regla de guarda:** todo `Clock.schedule` alcanzable por `activateLinks`/`changed` en boot va con
  `if(!Sys.atSteadyState()) return;`.

Referencia completa con citas file:line y tabla self-verify: corpus niagara-research **B729**. Caso vivo:
PANCCADIA León, ColdRoomPan-rt `BDefrostController`, 2026-09-03.
