# Retro — HMI 10" (1280x800): el panel de Control se REPARTE en 2 columnas, nunca scroll · 2026-09-03

Sexto retro del día. Al agregar una salida más ("Intercambiador") a la pestaña Control del Cuarto 3
(DashboardPan), el contenido pasó del alto de la HMI y la última fila quedó CORTADA bajo el footer. El
primer intento (scrollbar interno) el operador lo rechazó por feo. La solución correcta fue repartir en 2
columnas. Regla explícita del operador: **seguimos trabajando a 1280x800 (WEB-HMI10/CF)** — dejarlo asentado.
PROPOSED kit delta (propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **En la HMI de 10" (1280x800), cuando las salidas de un panel HOA/Control superan el alto, se REPARTEN en
   un grid de 2 columnas por GRUPO — no se agrega scroll ni se achican los targets.** El Cuarto 3 (2
   evaporadoras × [válvula+ventilador] + 2 resistencias + intercambiador = 4 grupos, 7 filas) no entra en
   una sola columna de ~660px, y sobraba toda la mitad derecha vacía. Un scrollbar vertical en un panel
   táctil fijo es no-ergonómico (mismo criterio que ya rechazó el scroll en la página de Configuración) y
   además desperdicia el ancho libre. El fix: envolver cada GRUPO (sub-encabezado + sus filas) en un
   `.hoa-group` indivisible y volcarlos en `display:grid; grid-template-columns:1fr 1fr` — el auto-flow los
   reparte (Evap1 | Evap2 / Resistencias | Intercambiador), la altura cae a la mitad y entra sin scroll.
   Claves de implementación que importaron:
   - El grupo debe ser UNA unidad (`break-inside:avoid`, `align-items:start`): si el sub-encabezado y sus
     filas son hermanos sueltos, el reparto los separa. Por eso se refactorizó `buildHoa` a un helper
     `addGroup(fillSub, fillRows)` que arma el grupo entero antes de colgarlo del grid.
   - Ensanchar el contenedor: el `max-width:660px` (una columna) tenía que subir (~1100px) para que las 2
     columnas tengan lugar; el encabezado + selector de cuarto quedan full-width ARRIBA del grid.
   - Dejar `overflow-y:auto` en el wrap como RED DE SEGURIDAD (no muestra scrollbar si entra), nunca como
     la solución.
   - Botones táctiles intactos: el reparto NO achica los targets (siguen ≥42px); esa es la ventaja sobre
     "meter una 3ª columna" o "encoger".
   → **PROPOSED types/dashboard.md delta (reforzar el §"HMI kiosk" y §"Config panel UX"):** la regla de
     "2-column grid, no page scroll a 1280x800" que ya está para la página de Configuración aplica IGUAL a
     la pestaña de Control/HOA — y la forma correcta es un grid de GRUPOS indivisibles (`addGroup` +
     `break-inside:avoid`), no columnas CSS sueltas sobre filas hermanas (parten el encabezado de sus
     filas). Regla de diseño para el HMI de 10": si un panel no entra en el alto, se REPARTE en columnas por
     grupo antes que scrollear.

2. **Un `hoaRow`/fila de salida reusable debe permitir un set de botones a medida, sin romper el default.**
   El intercambiador es AUTO/OFF (2 botones), no el HOA de 3 (Auto/Encender/Apagar). Se parametrizó
   `hoaRow(..., { buttons:[["auto","AUTO"],["off","OFF"]], valueMap:{auto:"1",off:"0"} })` — los botones por
   defecto siguen siendo los 3, y las demás salidas no se tocan. OJO con el prefill: una clave que termina
   en `Mode` cae en el mapeo genérico `["auto","on","off"][ordinal]`; una salida con otra convención (acá
   AUTO=1/OFF=0) necesita su propio caso ANTES del genérico, o el estado se prefill-ea mal.
   → **PROPOSED types/dashboard.md delta:** al agregar una salida con esquema distinto al HOA 0/1/2,
     parametrizar los botones del row Y special-casear su prefill de estado; no asumir el mapeo 0/1/2 sólo
     porque el slot se llama `*Mode`.

## Cost / evidence
- **Delta 1** evidence: captura del operador con "Intercambiador" cortado bajo el footer (scroll), y luego
  con scrollbar rechazado. Fix verificado: `node --check` OK sobre el `<script>`; el HTML servido trae
  `.hoa-grid`/`.hoa-group`/`addGroup`. Refactor en `DashboardPan-ux/src/rc/index.html` (`buildHoa`).
- **Delta 2** evidence: la fila del intercambiador con `buttons`+`valueMap` a medida, y el prefill
  special-case `if (/intercambiadorMode$/.test(k)) hoaState[k] = mv===1 ? "auto" : "off"` antes del
  genérico `/Mode$/`.

## Nota de alcance
Ambos deltas son de UI (`types/dashboard.md`) y salieron de código en preview (aún sin build/deploy al
momento del retro). La regla de resolución (1280x800, WEB-HMI10/CF) es un requisito FIJO del cliente para
todo lo que siga — cualquier panel nuevo se valida a ese tamaño (dashboard-preview.py `/hmi`) antes de dar
por bueno el layout.
