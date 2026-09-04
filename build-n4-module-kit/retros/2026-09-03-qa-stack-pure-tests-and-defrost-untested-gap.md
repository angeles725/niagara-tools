# Retro — cómo saber que un módulo rt está bien: pila de QA de 4 capas + el gap del deshielo sin tests · 2026-09-03

Undécimo retro del día. Auditoría de aseguramiento de calidad de nuestros módulos frente al framework de
tests de Tridium. [CERT]. PROPOSED kit delta (propose-never-apply). Sólo lo NUEVO. Detalle completo con citas:
corpus niagara-research **B741** (plan) + **TI** (infra de tests). Cierra el "¿cómo saben que están bien?".

## What this PROVED

1. **`niagaraTest` (BTestNg de Tridium) NO es un gate ejecutable para nosotros.** Corre TestNG DENTRO de una
   station forkeada (kernel NRE + `bin/test.exe` Windows + licencia dev) — no corre en WSL; y el plugin
   `com.tridium.niagara-module` 7.6.17 tiene un bug en `moduleTestAnnotationProcessor` que hace que
   `niagaraTest` descubra **0 tests** (`Total tests run: 0`, política `tests-are-docs`). `[CERT · bloque TI]`
   → **La vía real de cobertura ejecutable = tests PUROS JUnit** sobre lógica extraída sin tipos Baja
   (template-method: `final class` de decisión + `BComponent` adapter delgado), compilados con `javac -source 8`
   + `junit-4.13.2.jar` cacheado, SIN station.

2. **La pila de aseguramiento correcta para un módulo rt (4 capas):**
   (1) **tests unitarios puros** (JUnit, WSL) — la capa principal; (2) **build-verify gate** (Java8 +
   slotomatic + firma ALL PASS); (3) **smoke test en vivo** (cold-boot + secuencia de salidas — lo que el
   test puro no alcanza: lifecycle, links, IO de campo); (4) **review adversarial** (judgment-day).
   → **PROPOSED METHODOLOGY/build-verify delta:** documentar estas 4 capas como el estándar de "cómo saber
   que está bien"; el pure-test es obligatorio para TODA lógica de decisión/seguridad.

3. **Base actual buena: 63 tests puros** — `ColdRoomControl` 21 (enfriamiento/histéresis/freeze/sensor-fault),
   `CompressorControl` 28 (staging/dew-point R404A/start-prove/lead-lag), `DashboardDispatch` 14 (routing/
   RBAC/JSON). El patrón template-method (clase pura + adapter) está probado y corre en WSL.

4. **GAP PELIGROSO: la lógica del DESHIELO no tiene tests.** `BDefrostController` tiene su lógica (armTrigger
   interval math, interlock FIFO, terminate-by-temp, stagger) **INLINE en el `BComponent`, sin clase pura ni
   test** — y es JUSTO el subsistema que shipeó el bug `started()`/intervalo/`lastDefrostTime` futuro.
   `[CERT]` → **PROPOSED types/logic delta (regla):** toda lógica de TIMING/SEGURIDAD que hoy esté inline en
   un `BComponent` debe extraerse a una clase pura testeable. Plan para el deshielo: extraer `DefrostControl`
   con `nextDelayMs(lastMs,nowMs,intervalMs)` [null→full, elapsed<interval→resto, overdue→0,
   **|elapsed|>3·interval→interval** = guarda anti-reloj-futuro], el interlock como máquina de estados pura, y
   `terminateOnTemp(resistTemp,threshold,enabled,valid)`. Unos tests de `nextDelayMs` habrían agarrado el bug
   antes de campo.

5. **Verificar** que los 28 tests de `CompressorControl` cubran start-prove + lead/lag-por-horas (el `tick`
   que se congela en montaje tardío, retro started()/B737); si no, agregarlos.

## Net delta propuesto al kit (propose-never-apply)
- **build-verify.md / METHODOLOGY:** estandarizar la pila de 4 capas; pure-test obligatorio para lógica de
  decisión/seguridad; `niagaraTest` es documentación, no gate (en 7.6.17 / WSL).
- **types/logic.md:** regla — no dejar timing/seguridad inline en el `BComponent`; extraer a clase pura +
  test (con el ejemplo `DefrostControl`).
- **Backlog de módulo:** `DefrostControl` + tests = ALTA prioridad (subsistema sin tests que ya falló).

Referencia con citas file:line y self-verify: corpus niagara-research **B741** (+ **TI**, **B729/B730/B737**).
Caso vivo: PANCCADIA León, 2026-09-03.
