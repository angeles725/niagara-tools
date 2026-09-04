# Retro (mejora del proceso /build-n4-module) — superficie de escritura del servlet + el reader como autoridad · 2026-09-03

Self-retro de una sesión de **ANÁLISIS de comisionamiento** (NO de build) sobre los 3 módulos PANCCADIA
(ColdRoomPan + CompPan + DashboardPan). No se compiló ni deployó nada; estas son deltas de **diseño /
verificación** que salieron al leer el código, no lecciones del build-loop. PROPOSED kit deltas
(propose-never-apply). Solo lo NUEVO respecto a los retros del día.

## Hallazgos de diseño para el kit

1. **El endpoint de escritura genérico por ord del servlet `-ux` NO tiene lista blanca ni valida el flag `OPERATOR`** `[CERT, esta sesión]`.
   `DashboardPan-ux` expone UN solo `POST /api/setpoint` que hace `parent.set(prop, coerceValue(...))`
   sobre cualquier propiedad settable resoluble bajo el ORD del servicio [`BDashboardServlet.java:262-274`].
   Las únicas guardas son RBAC (`OPERATOR_WRITE` global), header XHR (`X-Requested-With`) y anti-traversal
   — **NO hay whitelist de slots ni chequeo del flag `OPERATOR` del slot destino**. Consecuencia: quien
   tenga `OPERATOR_WRITE` puede escribir CUALQUIER slot settable del componente bajo el servicio, incluidos
   los "solo lectura" del HMI (temps, LEDs de estado, `doorOpen`, los anchors de tiempo). La superficie de
   ESCRITURA es más ancha que la de LECTURA; la garantía "solo display" la sostiene únicamente la UI, no el
   servlet. Además el RBAC es un único bit global, sin permiso por-cuarto ni por-slot
   [`DashboardRbacHelper.java:96-98`].
   → **PROPOSED build-verify.md / checklist `-ux` delta:** para un módulo `-ux` con un endpoint de
   escritura genérico por ord, exigir una de dos guardas ANTES del `set()`: (a) whitelist explícita de
   slots escribibles, o (b) validar que el slot destino tenga `Flags.OPERATOR`. Sin eso, el gate marca el
   módulo como **"write-surface > read-surface"**. Declarar también la limitación de RBAC de un solo bit
   global como decisión explícita, no como default silencioso.

2. **La autoridad de "qué controla/muestra un dashboard `-ux`" son los arrays de grupos del reader, NO el property sheet ni la clase fachada** `[CERT, esta sesión]`.
   En esta sesión clasifiqué mal dos veces (anti-hielo "no está en el dashboard"; `startDelay` como
   config-solo-property-sheet) al inferir desde la fachada / lecturas parciales. La lista enumerable y
   autoritativa vive en `DashboardReader.java:80-134`: `TEMP_SLOTS`, `NUM_CONFIG_SLOTS`,
   `RELTIME_CONFIG_SLOTS`, `BOOL_CONFIG_SLOTS`, `DOOR_SLOTS`, `STATE_SLOTS`, `HOA_MODE_SLOTS`. Un slot del
   control está en el HMI **solo si** aparece en uno de esos arrays.
   → **PROPOSED BUILD-LOOP.md delta:** al documentar o verificar la superficie de un módulo dashboard,
   leer los arrays de grupos del reader COMPLETOS como fuente de verdad. Distinguir lectura
   (control→fachada, grupos de estado/temp) de escritura (fachada→control, grupos de config/HOA + flag
   `OPERATOR` en la fachada). No inferir de la UI ni del property sheet.

3. **Patrón "un slot de fachada abanica a N hijos"** `[ev: startDelay]`.
   `BRoomPanel.startDelay` es uno por cuarto y baja a las 3 `BEvaporatorUnit` del cuarto
   [`DashboardReader.java:98`: *"startDelay ... is per-room (all 3)"*]. Al enlazar en comisionamiento, un
   valor de fachada puede tener N destinos en el control.
   → **PROPOSED MANUAL/commissioning delta:** en la guía de links, marcar los slots de fachada que son
   **1→N** (un control del HMI, varios destinos en el control) para que el que cablea no busque un slot por
   evaporadora.

## Nota de alcance honesta

Esta sesión **NO compiló ni deployó nada** — fue análisis de comisionamiento. No hay lecciones del
build-loop propiamente dichas (preview-gate, tests puros, verify gate) porque no se ejercieron. Las 3
deltas son de diseño/verificación de módulos `-ux`, surgidas al leer el código. El maintainer debe
confirmar que #1 (write-surface) no esté ya capturada en un retro previo del día.

## Referencias

- Bitácora del episodio (mapa de conexión verificado, §8): `Cliente/Leon-Guanjuato/bitacora/2026-09-02-commissioning-3-modulos.md`.
- Retro hermano research-sdd: `niagara-research/retros/2026-09-03-research-sdd-commissioning-map-consulting-retro.md`.
