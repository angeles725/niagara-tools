# Retro — anti-patrón SISTÉMICO: un límite con default 0 BLOQUEA en vez de deshabilitar · 2026-09-03

Apareció TRES veces en este sistema durante la sesión. Merece regla propia y una auditoría transversal. PROPOSED kit delta (propose-never-apply).

## El patrón (el bug)
Un slot de LÍMITE/UMBRAL con `defaultValue = 0` que se compara contra un sensor con `valorValido && valor > limite` (o `< limite`), donde el rango físico del sensor está siempre de un lado del 0. Con el sensor cableado y válido y el límite en su default 0, la condición es SIEMPRE verdadera (o siempre falsa) → el interlock **bloquea permanentemente** (o **queda inerte permanentemente**), en vez de significar "deshabilitado".

El operador ve una máquina muerta (o una protección que nunca actúa) SIN causa evidente, porque "no toqué nada / lo dejé en 0".

## Las tres apariciones `[CERT / CERT-live]`
1. **Deshielo — `dischargeHighLimit` (CompPan), documentado como advertencia:** `dischargeHigh = dischargeValid && discharge > limit`; descarga R404A siempre >0 → con default 0 bloquea subir etapa. (Ya estaba como nota de comisionamiento.)
2. **CompPan — `dischargeHighLimit`, EL más grave:** `target = min(target, onCount)` con onCount=0 al arrancar → **el rack no arranca NI UN compresor** + alarma de alta descarga fantasma. `CompressorControl.java:109/177`.
3. **CompPan — `suctionLowLimit`, el simétrico inerte:** `suction < limit` con succión gauge siempre >0 → la protección de baja presión **nunca actúa** hasta comisionar. `:178`. (Falla hacia seguro, pero inconsistente con #2.)

## Regla propuesta
→ **PROPOSED METHODOLOGY.md / logic.md delta:**
- **En un slot de límite/umbral de PROTECCIÓN, `0` debe significar DESHABILITADO**, no "el peor caso". Guardar explícitamente: `if (limit == 0) { /* protección off */ }` o `boolean trip = limitEnabled(limit) && sensorValid && valor > limit`, donde `limitEnabled = limit != 0` (o un flag booleano aparte tipo `freezeProtect`).
- **Semántica CONSISTENTE entre límites que se leen igual** (descarga vs succión no deben comportarse al revés con el mismo default).
- **Auditoría transversal:** buscar en cada módulo las comparaciones `sensorValid && valor {>,<} limite` con `limite` default 0 (`rg 'Valid.*[<>].*Limit'` como punto de partida) y decidir explícitamente la semántica del 0 en cada una.
- **Test del caso default:** los tests suelen usar un límite "realista" (p. ej. 100) y así NO cubren el default 0 — agregar el caso `limit == 0` explícito (aquí ningún test lo cubría, por eso llegó a producción).

## Por qué importa (regla, no anécdota)
La contramedida NO es "el operador debe acordarse de fijar el límite" — eso ya falló en producción (rack muerto). La contramedida es que el default sea SEGURO por diseño: 0 = protección deshabilitada, y el comisionador la habilita con un valor real cuando corresponde.

## Referencias
- Backlog CompPan: `Cliente/Leon-Guanjuato/bitacora/2026-09-03-deshielo-cuarto3-no-arma-diagnostico-multi-sesion.md` (§Backlog CompPan) + engram.
- `CompressorControl.java:109/177/178`, `BCompressorControl.java:114/121`.
