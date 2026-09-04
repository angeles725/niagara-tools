# Retro — `ng-deploy.sh` backup: liviano (solo nuestros jars) + auto-purga · 2026-09-03

Octavo retro del día. Tras ~8 deploys en la sesión, los `_backups/` acumularon **~1.9 GB** (8 tarballs de
~240 MB) y se colaron en un `.rar` que llegó a **1.8 GB**. Causa: `ng-deploy.sh backup()` respalda TODO el
modules dir en cada deploy y nunca purga. PROPOSED kit delta (propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **El backup de `ng-deploy.sh` respalda el modules dir COMPLETO (todos los módulos de Honeywell), no solo
   los jars que despliega — ~240 MB por deploy — y nunca purga los viejos, así que crecen sin techo.** El
   `backup()` hace `tar -czf "$bk" -C "$(dirname STATION_MODULES_DIR)" "$(basename STATION_MODULES_DIR)"`,
   o sea la carpeta entera. Con muchos deploys de iteración de UI (que son baratos, sin reinicio) los
   backups dominan el árbol y cualquier `.rar`/respaldo que los incluya explota (1.8 GB medido). El
   comentario del código dice "tar deployed jars" pero en realidad tar-ea todo el directorio.
   → **DECISIÓN DEL OPERADOR (2026-09-03): NO hacer backups en el deploy — los jars ya están en GitHub, que
     es el respaldo/rollback.** Por eso la recomendación PRINCIPAL es que `ng-deploy` NO respalde por
     default (o que `--no-backup` sea trivial de usar sin el `--i-know-what-im-doing`, o directamente que
     el default sea sin backup cuando el repo trackea los jars). En esta sesión se pasó a deployar con
     `--no-backup` y a borrar los `_backups/` acumulados. Las dos mejoras de abajo quedan como fallback
     para repos que NO trackean los jars en git.
   → **PROPOSED toolbelt/ng-deploy delta (dos partes, ambas seguras — fallback si se quiere backup local):**
     a) **Backup liviano:** respaldar SOLO los jars del módulo que se va a pisar, no el modules dir entero.
        Antes de copiar, guardar los `<MODULE_NAME>-{rt,ux,wb}.jar` existentes de `STATION_MODULES_DIR`
        (los que realmente se reemplazan) en el tar. Pasa de ~240 MB a pocos MB por deploy, y sigue
        permitiendo el rollback de NUESTRO módulo (que es lo único que ng-deploy toca).
     b) **Auto-purga:** tras un backup exitoso, conservar solo los últimos N por módulo (ej. N=3) y borrar
        los más viejos (`ls -t "$BKDIR"/${MODULE_NAME}-pre-*.tar.gz | tail -n +$((N+1)) | xargs -r rm -f`).
        Evita el crecimiento sin techo sin perder el rollback reciente.
   → **PROPOSED BUILD-LOOP.md §6 / METHODOLOGY nota operativa:** los `_backups/` y `build/` son artefactos
     regenerables — excluirlos de cualquier `.rar`/snapshot manual del repo (además de `.gitignore`, que ya
     los ignora). Limpieza manual de emergencia: `ls -t _backups/*.tar.gz | tail -n +2 | xargs rm` deja solo
     el último.

## Cost / evidence
- Evidence: `Dashboard/_backups` = 1.2 GB (5 tarballs), `Paccadia/_backups` = 686 MB (3 tarballs), cada uno
  ~239.5 MB y casi idéntico (snapshot del mismo modules dir). El `.rar` recreado a las 16:54 los incluyó →
  1.8 GB. Tras purgar a 1 por módulo: 229 MB + 229 MB; repo completo 527 MB. `backup()` en `ng-deploy.sh`:
  `tar -czf "$bk" -C "$(dirname "$STATION_MODULES_DIR")" "$(basename "$STATION_MODULES_DIR")"`.
- Riesgo del delta: bajo. (a) no cambia la semántica de rollback de nuestro módulo; (b) conserva los N
  últimos. Ambos son opt-in razonables; se puede exponer `--keep N` y `--full-backup` para el caso raro en
  que se quiera el respaldo del dir entero.

## Nota de alcance
Delta de tooling (`toolbelt/ng-deploy` + BUILD-LOOP.md §6). No toca módulos ni el corpus. Es propuesta:
un humano decide si cambia `ng-deploy.sh`. La limpieza puntual de esta sesión (borrar backups viejos,
dejar 1 por módulo) ya se hizo a mano; el delta busca que no vuelva a pasar.
