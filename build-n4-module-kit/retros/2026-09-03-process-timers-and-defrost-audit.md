# Retro — timers "en vivo" en un dashboard (deshielo va/falta, cuánto lleva enfriando) + auditoría del disparo de deshielo · 2026-09-03

Quinto retro del día (previos hoy: freeze-stat-leds, soft-start-staggered-startup, dashboard-contract-port-spec,
ng-deploy-type-count-and-cwd). Este tramo agregó a ColdRoomPan + DashboardPan el proceso/tiempo de cada cuarto
(Cuarto 3: deshielo "va X / falta Y"; los demás: "lleva enfriando X" + estado) y, sobre el reporte del operador
"el deshielo nunca entró", auditó la lógica de disparo. Compiló + gate ALL PASS + deploy. PROPOSED kit deltas
(propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **Para un contador "en vivo" (transcurrido/restante) en un dashboard que se lee por poll, expón ANCLAS
   (timestamps `BAbsTime`) en el rt, computa los derivados en ms en el READER con UN solo `Clock.millis()`
   por respuesta, y tickea suave en el SPA entre polls — NO expongas "restante" como slot `BRelTime`
   almacenado.** Un slot `BRelTime` no se auto-actualiza: quedaría congelado entre ejecuciones del engine y
   sólo saltaría en cada poll. Diseño que funcionó: el rt guarda el instante ancla (`defrostStart`,
   `coolingSince` como `BAbsTime` TRANSIENT|SUMMARY|READONLY, seteado en el edge — `beginDefrost()` /
   subida de `roomCall` en `execute()`); el reader (`DashboardReader`) toma `now = Clock.millis()` una vez y
   emite `coolingElapsedMs`, `defrostElapsedMs`, `defrostRemainingMs = max(0, duration - elapsed)`; el SPA
   guarda `procRxAt = Date.now()` al recibir y un `setInterval(1s)` avanza el texto sin refetch. Resultado:
   reloj autoritativo en el JACE (sin desfase del navegador), contrato JSON uniforme (todo número/bool), y
   conteo fluido. Verificado en preview: Cuarto 3 "va 12m25s · falta 17m35s", Cuarto 1 "lleva 6m12s".
   → **PROPOSED types/logic.md delta:** patrón "anchor slot para display": un tiempo mostrado en una UI que
   polea = un `BAbsTime` ancla READONLY seteado en el edge del evento, NO un `BRelTime` de "restante"
   recalculado en el engine. El consumidor computa `now - ancla`.
   → **PROPOSED types/dashboard.md delta:** el reader puede emitir KEYS DERIVADOS (no sólo espejar slots):
   toma un `now` único por respuesta y publica elapsed/remaining en ms. La fachada sólo lleva las anclas
   (link-in) + la duración (config ya existente); los derivados NO son slots. Sirve igual para el consumidor
   oBIX externo (lee las anclas y computa su propio conteo).

2. **La fachada de dashboard suma anclas de proceso como slots link-in nuevos, y el reader gana un tipo
   `BAbsTime`: extiende el reader además del slot.** `BRoomPanel` ganó `coolingSince`/`defrostStart`
   (`BAbsTime`, SUMMARY) + `defrostActive` (`BStatusBoolean`, SUMMARY); `DashboardReader` necesitó un
   `readAbsTimeMillis()` (BAbsTime → epoch ms, null si `isNull()`) junto al `readRelTimeMillis()` existente,
   e imports de `BAbsTime`+`Clock`. El path completo (fachada slot → reader → SPA descriptor/render) es el
   mismo end-to-end que ya documenta dashboard.md para un config field; acá el "campo" es de sólo lectura y
   derivado. Sin la entrada en el reader el slot es invisible aunque exista y esté linkeado.
   → **PROPOSED types/dashboard.md delta (reforzar el §"end-to-end path"):** el path fachada→reader→SPA
   aplica también a display DERIVADO, no sólo a config escribible. Un slot `BAbsTime` link-in requiere un
   lector de tipo en el reader (los readers existentes cubren Numeric/Bool/RelTime, no AbsTime).

3. **Un control automático time-gated (deshielo por intervalo) que "nunca entró" casi siempre es
   PRECONDICIÓN no cumplida, no bug de la secuencia — y le falta un disparo manual para probarlo.** La
   secuencia de entrada estaba correcta (`BDefrostController.beginDefrost` → `BEvaporatorUnit.enterDefrost`:
   cierra válvula, apaga fan, energiza resistencia). Lo que lo mata en la planta: (a) `hasDefrost` default
   **false** por evaporadora y SIN campo en el dashboard → `requestDefrostCycle()` salta la unidad y
   `enterDefrost()` corta en `if (!getHasDefrost()) return;`; (b) `interval` default 8h y, mientras
   `lastDefrostTime` sea null (nunca deshieló), CADA reinicio de station rearma las 8h completas
   (`armTrigger()`: `last.isNull() → delayMs = intervalMs`) → con reinicios frecuentes el PRIMER deshielo
   nunca llega; (c) no hay acción operador "deshielar ahora", así que no se puede forzar para verificar.
   → **PROPOSED types/logic.md delta:** para todo control automático time-gated, (i) expón una acción manual
   "ejecutar ahora" operator-visible para comisionar/probar sin esperar el temporizador; (ii) haz visibles
   sus PRECONDICIONES de habilitación (un flag `hasX` default false sin superficie de UI es un "nunca entró"
   silencioso — surfacéalo o dale default seguro); (iii) ancla el primer disparo para que el reinicio no
   reinicie el conteo (si `lastEventTime` es null en `atSteadyState`, sémbralo, no rearmes el intervalo
   completo). Esto endurece el patrón "anclar intervalo a reloj persistente" que ya trae logic.md, cerrando
   el hueco del PRIMER evento.

## Cost / evidence
- **Deltas 1-2** evidence: `node --check` OK sobre el `<script>` extraído; preview-server con mock extendido
  (`coolingElapsedMs`/`defrostActive`/`defrostElapsedMs`/`defrostRemainingMs`) sirviendo el conteo; unit-test
  en node de `fmtDur()`/`procDe()` → "Enfriando · lleva 6m12s" / "En setpoint" / "Deshielo · va 12m25s ·
  falta 17m35s". Build ambos módulos: bytecode 52, firmados, gate ALL PASS; deploy con backup.
- **Delta 3** evidence: `BEvaporatorUnit.java:1045` (`if (!getHasDefrost()) return;`), `hasDefrost`
  defaultValue `false` (~línea 128); `BDefrostController.armTrigger()` `last.isNull() → delayMs = intervalMs`;
  el reader NO expone `hasDefrost` (no está en `BOOL_CONFIG_SLOTS`) → invisible desde el dashboard.

## Nota de alcance
Los deltas 1-2 son de patrón de display (logic.md + dashboard.md) y salieron de código que compiló y deployó.
El delta 3 es una lección de diseño de control derivada de una AUDITORÍA (no de un cambio aplicado): el
endurecimiento propuesto (acción manual + visibilidad de precondiciones + anclaje del 1er intervalo) quedó
PROPUESTO, pendiente de decisión del operador; no se tocó la lógica de deshielo en este tramo. La verificación
en paralelo del arranque escalonado (soft-start por posición de planta, `AUTO_STEP_MS=60000`) confirmó que los
slots nuevos NO lo alteraron.
