# Retro — la station de producción corre en el ATLAS (snap), NO en el PC donde compilas/deployas · 2026-09-03

Del episodio del deshielo Cuarto 3: verificar el jar en `modules/` del PC de Workbench NO prueba qué corre en la station real. PROPOSED kit delta (propose-never-apply). Sólo lo NUEVO.

## What this PROVED

1. **La station de producción PANCCADIA corre en un dispositivo ATLAS con Niagara como paquete `snap`, en OTRA versión de NRE que el PC de build/deploy.** `[CERT-live]` (log de arranque):
   - Runtime real: `niagara_home = /snap/tridium-niagara/7149/niagara_home`, `config.bog` en `/var/snap/tridium-niagara/common/niagara_user_home/stations/PANCCADIA/`, host `ATLAS-SD-3FFB-3757-0DAB-DFC7`, **NRE 4.15.3.28**.
   - PC de build/deploy (donde apunta `ng-deploy` y el `niagara_home` del kit): `/mnt/c/Honeywell/OptimizerSupervisor-N4.14.0.162` → **N4.14.0.162**.
   → El `niagara_home` de RUNTIME ≠ el `niagara_home` de BUILD/DEPLOY, y son versiones distintas (4.15 vs 4.14).

2. **Verificar el jar en `modules/` del PC NO dice qué código corre en el ATLAS.** `[CERT-live]` Durante el diagnóstico se miró `/mnt/c/Honeywell/.../modules/ColdRoomPan-rt.jar` (18:23, con `nextDefrostTime`) asumiendo que era lo que ejecutaba la station — pero la instancia viva está en el ATLAS. El oráculo autoritativo del código VIVO NO es el jar en disco del PC, sino los **slots del tipo leídos en la station** (oBIX contra el JACE, o el Slot Sheet en Workbench conectado por Fox al ATLAS). Fue la lectura oBIX (`nextDefrostTime`/`defrostStart` PRESENTES en el tipo vivo) la que confirmó que el ATLAS SÍ tenía los slots nuevos — no el jar del PC.
   → **PROPOSED BUILD-LOOP.md / preflight delta:** antes de diagnosticar "qué corre", identifica el `niagara_home` REAL de la station (del log de arranque `Niagara runtime booted ("...")`, o de la Platform). Para PANCCADIA la station NO está en el PC de Workbench; está en el ATLAS (snap). Un jar en `modules/` del PC es lo que Workbench tiene disponible para EMPUJAR, no necesariamente lo que corre.

3. **Build target-version vs runtime-version:** se compila contra el `niagara_home` del PC (N4.14) pero la station corre 4.15.3.28. Aquí cargó bien (compatibilidad hacia adelante del bytecode 52 + tipos), pero es una diferencia real a tener presente.
   → **PROPOSED delta:** cuando el runtime de la station difiere del `niagara_home` de build, pasar `--target-version` al verify gate para validar contra la versión que realmente corre, no solo la de compilación.

4. **El deploy real al ATLAS es empujar el jar al DISPOSITIVO + reiniciar su station, no `ng-deploy` a `modules/` del PC.** `[CERT-live]` el hecho de que el jar 18:23 llegó al ATLAS (slots presentes en vivo) prueba que hubo un push al dispositivo; `[INFER]` el mecanismo (Software Manager / Provisioning Niagara) no se verificó en este episodio.
   → **PROPOSED delta:** documentar el paso de deploy device-side para stations que NO corren en el PC de Workbench (JACE/ATLAS): construir → (`ng-deploy` deja el jar en el PC) → Software Manager/Provisioning empuja al dispositivo → reiniciar la station del dispositivo. Confirmar el mecanismo exacto en el próximo deploy y cerrarlo como `[CERT]`.

## Referencias
- Episodio: `bitacora/2026-09-03-deshielo-cuarto3-no-arma-diagnostico-multi-sesion.md`.
- Retros hermanos: `2026-09-03-self-firing-timer-needs-started-not-only-atsteadystate.md`, `2026-09-03-hidden-actions-not-invocable-and-runtime-anchor-verification.md`.
