# Retro — el CONTRATO de un módulo dashboard como superficie reusable (port spec) · 2026-09-03

Tercer retro del día (previos: `2026-09-03-coldroompan-dashboardpan-freeze-stat-leds.md`,
`2026-09-03-soft-start-staggered-startup.md`). Este tramo NO compiló módulo: fue documentar el DashboardPan
para que OTRO equipo (un viewer 3D externo) lo reuse por oBIX sin tocar la station. Salieron 2 lecciones
genuinamente de build-n4-module. PROPOSED kit deltas (propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **Un módulo dashboard tiene un CONTRATO reusable — ords + `{v,st}` + encoding de escritura + slots
   OPERATOR — que un consumidor externo reusa SIN la station; documentarlo como "port spec" es un
   entregable de primera clase.** Un equipo aparte porta el DashboardPan a un viewer 3D standalone que
   lee/escribe por oBIX (no por el servlet). Lo que necesitaban NO era el código del módulo sino su
   CONTRATO: (a) el mapa plano `CuartoN/slot → {v,st}` (idéntico entre el servlet `/api/equipment` y el
   export oBIX — 285 puntos = ~57 slots × 5 cuartos, verificado), (b) qué slots son OPERATOR-escribibles vs
   display/estado (estos son destino de link control→fachada, un write se los pisa), (c) el encoding por
   tipo (número °C / HOA `0/1/2` / bool / RelTime en **ms**). Eso vive disperso en la fachada, el reader y
   el SPA; reunirlo una vez ahorró que el otro equipo lo reingeniere.
   → **PROPOSED types/dashboard.md delta:** añadir una sección "El contrato como API externa": cuando un
   dashboard se consuma fuera de la station (oBIX, un mirror en DB, otra UI), documenta UNA vez el port
   spec — la forma `{v,st}` keyed por ord, la tabla slot→encoding, y la lista OPERATOR-escribible vs
   readonly. Es la misma info que el integrador necesita para los linkmarks, pero orientada a un consumidor
   de datos, no a Workbench.

2. **Para entregar un SPA de un solo archivo gigante a otro equipo, pasa RANGOS DE LÍNEA al source
   verbatim, no un pegado — y avisa del base64.** El `index.html` del DashboardPan son ~1928 líneas + 3.5MB
   de fotos base64 embebidas (arreglo `CUARTOS`). Pegar `SENSORS`/`classOf`/`:root`/etc. en un chat pierde
   fidelidad y arriesga transcripción; lo correcto fue un doc con el MAPA de rangos (`SENSORS` 665-688,
   `classOf` 833-855, `:root` 9-28, control 1113+, …) + el aviso de NO hacer `Read` del archivo entero ni
   del rango base64 (revienta tokens; usar `sed`/`awk 'length<300'`) + validar con `node --check`.
   → **PROPOSED types/dashboard.md delta (reforzar el §"editing technique"):** al handoff de un SPA
   single-file a otro agente/equipo, entrega un mapa de rangos al source (no un pegado) y el aviso del
   base64/tamaño. Complementa el retro de navegación de archivo gigante del kit research-sdd.

## Cost / evidence
- **Delta 1** evidence: el otro equipo pidió "port spec verbatim"; el export oBIX real (285 pts) confirmó
  que los ords coinciden 1:1 con la fachada, así que el contrato es reúso directo. Doc entregado:
  `docs/PORT-SPEC-dashboardpan-2d.md` + `docs/ARQUITECTURA-datos-obix-supabase.md` (repo del cliente).
- **Delta 2** evidence: intentos previos de `Read` del `index.html` completo fallaron por límite de tokens
  (base64); el mapa de rangos + `sed`/`awk length<300` fue lo que funcionó, y es lo que se le pasó al equipo.

## Nota de alcance
No hubo build ni cambio de gate en este tramo; ambos deltas son de documentación (`types/dashboard.md`). La
asimetría lectura/escritura de un dashboard consumido por oBIX (lectura se espeja a una DB; la escritura DEBE
llegar a la station por oBIX PUT a un endpoint server-side) quedó en el doc de arquitectura del cliente, no
es material de kit.
