# Retro — cambiar el TIPO de un slot con datos guardados ROMPE el .bog: la station no arranca · 2026-09-03

Noveno retro del día, y el más importante: un cambio de tipo de slot en una station VIVA tumbó el arranque
(producción de refrigeración caída). [CERT-live]. PROPOSED kit delta (propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **Cambiar el TIPO de un slot EXISTENTE de un BComplex (`BStatusNumeric`) a un simple (`double`) en una
   station que ya tiene ese slot con datos guardados ROMPE la decodificación del `.bog` y la station NO
   ARRANCA ("Cannot load station / App Failed").** El `.bog` guardado tenía `BRoomPanel.setpoint` como una
   ESTRUCTURA compleja (`baja:StatusNumeric` con hijos `value`/`status`/facets). El módulo nuevo definía
   `setpoint` como `double` simple. Al cargar, el decoder no puede meter el complejo guardado en un slot
   simple, y el parseo se DESALINEA: los `<p>` hijos del viejo StatusNumeric se leen como si fueran slots
   del padre, produciendo warnings en cascada y, finalmente, un SEVERE que impide cargar la station.
   Firma exacta observada (17:04, Supervisor N4.14):
   - `WARNING [sys.xml] Cannot set property RoomPanel.setpoint: BStatusNumeric cannot be cast to BDouble`
   - `WARNING [sys.xml] Missing frozen property: differentialUp / zoneHighLimit / zoneLowLimit / evapLowLimit`
   - `WARNING [sys.xml] Missing slot StatusNumeric.startDelay`  ← el decoder ya cree que startDelay es hijo del StatusNumeric
   - `SEVERE [sys] Cannot load station — java.lang.ClassCastException: BRelTime cannot be cast to BComplex
     at javax.baja.io.ValueDocDecoder.parseSlots(...)`  → App Failed.
   → **PROPOSED METHODOLOGY.md / types/* delta (regla dura):** **NUNCA cambies el TIPO de un slot existente**
     en un módulo que ya está instanciado con datos en una station — sobre todo BComplex↔simple
     (`BStatusNumeric`↔`double`, `BStatusBoolean`↔`boolean`, etc.). El `.bog` codifica el slot según su tipo
     al guardar; un tipo nuevo incompatible no decodifica y tumba el boot, no solo "resetea el valor".
     Agregar un slot NUEVO es seguro; RE-TIPAR uno existente no lo es.

2. **Para hacer un config numérico escribible por oBIX SIN romper el `.bog`: agregar un slot `double` NUEVO
   y aparte (no re-tipar el existente).** El objetivo original era que `setpoint` (BStatusNumeric, no
   escribible por oBIX) se pudiera escribir por oBIX (un `double` sí sale `obix:real writable`). La vía
   correcta NO es cambiarle el tipo a `setpoint`, sino agregar `setpointCmd` (`double`, SUMMARY|OPERATOR)
   que el integrador linkea al setpoint del control; oBIX escribe `setpointCmd` y el link lo propaga. El
   `setpoint` original queda intacto (el `.bog` no se rompe) y sigue reportando estado.
   → **CORRIGE** el enfoque del retro/bitácora previos del día que proponían "convertir setpoint a double"
     (`bitacora/2026-09-03-setpoint-double-obix-writable.md`, engram #8041): ese enfoque es INSEGURO en una
     station viva. La forma segura es slot nuevo, o —si hay que re-tipar— MIGRAR el `.bog` primero (borrar
     el valor guardado del slot con el módulo VIEJO cargado, ANTES de instalar el módulo con el tipo nuevo).

3. **git salvó la recuperación porque el repo trackea los jars firmados — pero también fue la trampa: los
   backups locales se habían borrado.** Recovery: el jar bueno (setpoint=StatusNumeric) estaba en el commit
   anterior; `git checkout <commit-previo> -- BRoomPanel.java` revirtió SOLO ese archivo (el resto del
   trabajo del día quedó intacto), rebuild `-rt` (gate PASS), redeploy, reinicio → arrancó. Sin los jars en
   git (o con los `_backups` ya borrados) la recuperación habría sido mucho más lenta.
   → **PROPOSED BUILD-LOOP.md §6 delta:** ante un cambio RIESGOSO (tipo de slot, borrado de slot, rename),
     COMMITEAR el estado bueno ANTES de deployar, y NO borrar el respaldo del jar previo hasta confirmar que
     la station arrancó con el cambio. El rollback = redeploy del jar previo + reinicio.

## Cost / evidence
- Evidence: log real de la station del Supervisor (arriba). Recovery verificado: `getSetpoint()` volvió a
  `BStatusNumeric` en el jar deployado; la station reinició y arrancó (confirmado por el operador y por el
  monitoreo en vivo del equipo del viewer 3D contra el JACE: setpoints intactos C1=3.5/C2=3/C3=-6/C4=3/C5=0,
  cero pérdida). El JACE NUNCA recibió el double (el operador no lo instaló), por eso solo cayó el Supervisor.
- Marcador: [CERT-live] — comportamiento observado en una station N4.14 real, no inferido.

## Nota de alcance
Delta de metodología/proceso (regla dura + BUILD-LOOP.md §6). Incidente de producción resuelto: la station
del Supervisor está arriba; el JACE nunca se afectó. El `setpoint`-por-oBIX queda PENDIENTE, a rehacer con
la vía segura (slot `setpointCmd` nuevo). Yo (el asistente) advertí "se resetean los setpoints" pero
subestimé el riesgo real (tumbar el boot); esta retro corrige esa evaluación para futuras sesiones.
