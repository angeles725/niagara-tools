# Retro — un deploy `-ux`-only NO reinicia la station (solo hard-reload del navegador) · 2026-09-03

Séptimo retro del día. Al iterar la UI del DashboardPan (config sin scroll + campos de Cuarto 5) se
deployó SOLO el perfil `-ux` con `ng-deploy.sh --mode B`, y el propio deploy avisó que no hacía falta
reiniciar la station. Hallazgo de toolchain reutilizable (no de internals de N4). PROPOSED kit delta
(propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **Un deploy que cambia SOLO el perfil `-ux` (assets estáticos del SPA en `rc/`, sin clases Java) NO
   requiere reinicio de la station — basta un hard-reload del navegador en la HMI.** `ng-deploy.sh --mode B`
   (ux-only) copió el jar y su `print_restart_reminder` imprimió literalmente
   `[ng-deploy] browser hard-reload only (no station restart needed)`. En cambio, un deploy que toca clases
   Java (perfiles `rt`/`wb`, mode A o C) imprime `station restart required (Java classes deployed)` y SÍ
   necesita el reinicio para cargar las clases nuevas. La razón: el servlet sirve `rc/index.html` (y demás
   assets) desde el classloader del módulo en cada request, así que un jar `-ux` nuevo se toma al recargar;
   las clases Java, en cambio, ya están cargadas en la JVM de la station y solo se recargan al reiniciarla.
   → **PROPOSED BUILD-LOOP.md §6 (deploy) delta:** al planificar un cambio, separar UI de lógica:
     - Cambio SOLO de UI/SPA (`-ux`, `rc/`) → `ng-deploy --mode B` → **sin ventana de parada de planta**,
       el operador solo hace hard-reload del navegador. Ideal para iterar diseño en producción.
     - Cambio de lógica (`-rt`) o widgets wb (`-wb`) → mode A/C → **requiere reinicio de station**
       (ventana de parada). Agrupar los cambios de rt para no reiniciar de más.
   Regla práctica: iterar la UI en producción es barato (sin reinicio); tocar el rt cuesta una ventana.

## Cost / evidence
- Evidence: salida real de `ng-deploy.sh --mode B` sobre DashboardPan-ux:
  `[ng-deploy] copy ok` · `[ng-deploy] verify ok (DashboardPan-ux.jar: 2/2)` ·
  `[ng-deploy] browser hard-reload only (no station restart needed)`. Contrastar con los deploys mode A/C
  del mismo día, que imprimieron `station restart required (Java classes deployed)`.
- El mecanismo (assets servidos por classloader vs clases en la JVM) es [INFER] consistente con el diseño
  del servlet (`getClassLoader().getResourceAsStream("rc/"+path)`, ver types/dashboard.md §ux); el
  comportamiento observado (no-restart) es [CERT] por la salida de ng-deploy.

## Nota de alcance
Delta de tooling/proceso (BUILD-LOOP.md §6). No es conocimiento de internals de N4 → no va al corpus de
investigación; su hogar es este kit + un pointer en Engram (capturado vía research-sdd document mode).
