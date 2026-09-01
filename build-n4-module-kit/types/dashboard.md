# Type: dashboard (facade + servlet + SPA) — MATURE

Proven end-to-end on DashboardPan (2026-08). A browser dashboard for an HMI, served by the module itself. NOT PX — a `BWebServlet` + a static ES5 SPA. Exemplar to copy: chihuahua-ux; cleaner reference: DashboardPan.

## rt — the facade (no logic)
- One `BComponent` per equip/entity (`BRoomPanel`-style): **link-in display slots** (`BStatusNumeric`, `SUMMARY`) the integrator links from control points; **writable config slots** (`SUMMARY|OPERATOR`). No control logic.
- A `BAbstractService` root (`getServiceTypes()`) holding the entity components as frozen children — the ORD the servlet walks. Must sit at a fixed path (e.g. `station:|slot:/Services/DashboardService`).

## Facade contract & commissioning links
- **Link direction is fixed by role:** display telemetry flows control→facade (the integrator links each control point into a `SUMMARY` display slot); operator config and commands (setpoint, differentials, HOA modes, defrost, startDelay) flow facade→control; names and alarm limits stay in the facade, unlinked. [ev: bitácora 5cuartos §6]
- **Entity/room count is data-driven:** the reader walks an array (`ROOMS[]`) and the servlet/dispatch/RBAC take an arbitrary ORD — so adding an entity is one `@NiagaraProperty(name="CuartoN", type="BRoomPanel")` (slotomatic regenerates the AUTO region + type hash) plus one reader-array entry; every entity exposes the full slot set for free. [ev: retro 5rooms #3]
- **Keep the facade a PURE slot container — never `setOut` a child `ControlPoint` from `changed()`:** feeding points on every change re-triggers `changed()` (a re-entrancy flood) and a `ControlPoint.setOut` always logs "out set explicitly". If the module must own alarm points, guard `changed()` to real source slots + a deadband; the clean design keeps display/config/HOA/label slots only and lets the integrator build alarms in Workbench. [CERT-live 2026-09-01 · bitácora 5cuartos §7]
- **Adding one operator config field is an end-to-end path, not a UI edit:** facade `@NiagaraProperty(SUMMARY|OPERATOR)` → reader field list (e.g. `BOOL_CONFIG_SLOTS`) → SPA field descriptor (`SP_COMMON`) → a linkmark facade→control. Miss any layer and the field is invisible or unwired. [ev: bitácora 5cuartos §11]
- **Expose operator-runtime config on the HMI; leave commissioning-time enums in Workbench:** setpoints/limits/modes belong on the dashboard, but one-time commissioning choices (staging mode, defrost trigger mode) stay in Workbench unless the operator asks to surface them. [ev: bitácora 5cuartos §11]

## ux — servlet + SPA
- `BWebServlet` subclass; `getServletName()` = the mount prefix → `/<prefix>/`. Registered as a `<type>` in module-include.xml + a palette instance under a Servlets folder.
- `doGet/doPost` delegate to a **pure router** (WSL-unit-testable, no Niagara deps). Guards in order: path-traversal→404; **XHR guard** on `/api/*` (missing `X-Requested-With: XMLHttpRequest` → 302); unknown `/api/*`→404; static fallback (`/`→index.html).
- **Reader**: walk the facade → flat JSON keyed by slot-path, each value `{"v":num|bool|null,"st":"<status>"}`, fault-aware, `Locale.ROOT` doubles, escaped strings.
- **RBAC**: `checkCanWrite` first line of every write: `BPermissions.has(OPERATOR_WRITE)` (bit, not role name), fail-closed, `SlotPath.unescape` the username. Audit each mutation.
- Static assets in `src/rc/`; gradle copies `src/rc→rc/`; served via `getClassLoader().getResourceAsStream("rc/"+path)`.
- Frontend: ES5 + `fetch`, REST poll. **Every fetch (reads too) sends `X-Requested-With`** or the guard 302s it → page loads but data shows "--". Use ABSOLUTE `/<prefix>/...` URLs (relative break without a trailing slash). Write via `POST /api/setpoint {ord,value}`.

## Extending an existing dashboard
- **Before hand-editing a dashboard SPA, decide DATA vs CODE:** a data-driven render (`CUARTOS.forEach`, `SENSORS.map`) is entity-count-agnostic, so scaling 3→5 rooms is swapping the DATA arrays (`CUARTOS`/`SENSORS`/`FANS`, the plano src, `viewBox`/`IMG_W`/`IMG_H`) with zero change to the render functions. Verify with `node --check` on the extracted `<script>` blocks after. [ev: retro 5rooms #1]
- **The module is the SKELETON; the operator's standalone HTML is only the DATA:** splice the operator's calibrated data blocks into the module's `index.html` (which already carries kiosk CSS, `/api/*` wiring, status classes, alarms, setpoint save) and remap sim ords to the module's Niagara ords — never rebuild the module from the standalone, which loses the integration layer. See `METHODOLOGY.md` §editing technique for navigating the giant single-file source. [ev: retro 5rooms #2]

## Config panel UX on a fixed touch panel
> These build on the `## HMI kiosk` budget below (≥44px targets, no page scroll at 1280×800).
- **When a room's fields overflow two columns at ≥44px targets, partition into named sub-tab GROUPS — not scroll, not more columns:** on a fixed capacitive panel scroll is non-ergonomic and a 3rd column shrinks touch targets (both operator-rejected). Data-drive it: give the room `groups:[{name,fields}]`, render a `.cfg-tabs` bar, keep ONE shared `items[]` + one "Guardar cambios" so edits across tabs save once. [ev: retro hmi-touch-ux Δ1]
- **Write the per-field row-builder as a standalone `buildField(room,f,items)` from the start:** it registers the input, pushes to `items`, and returns the row — so grouping, reordering, and conditional fields all stay cheap. [ev: retro hmi-touch-ux Δ2]
- **Give a config value both a typed input AND +/− steppers, validating and reverting invalid text:** the operator can type or step; min/max validation reverts a bad entry. [ev: bitácora 5cuartos §4]
- **A dense secondary feature gets its own full-width page with an entity selector, not a cramped corner of the per-room panel:** e.g. HOA control moved to a full-width page with a room selector — the same partition principle as sub-tabs, at page scale. [ev: bitácora 5cuartos §4]
- **Writable ranges (min/max/step/decimals) live in the SPA field-descriptor array, not Java — but check the facade and control facets too:** a "-40 floor" was a frontend `SP_COMMON min:-40`, not a servlet clamp; confirm the facade/control facets impose no MIN before concluding the link won't re-clamp. [ev: retro hmi-touch-ux Δ4]
- **Sync config across viewers: run prefill on EVERY poll but SKIP dirty (unsaved-editing) fields, and rebuild an open sub-panel each poll:** otherwise telemetry syncs on the 5s poll but config/HOA load once, so a change by one operator is invisible to the others until reload. [ev: bitácora 5cuartos §10]

## Charts on an HMI
- **Size the SVG chart's `viewBox` to the MEASURED box in real px each render — do not carry a fixed art-board viewBox with `preserveAspectRatio="none"`:** measure `getBoundingClientRect()` (guard width>60) and set `viewBox="0 0 <w> <h>"` so 1 unit = 1px and strokes/axis text stop distorting. [ev: retro hmi-touch-ux Δ5]
- **On a capacitive HMI the chart read-out is a TOUCH crosshair (pointer events + `touch-action:none`), never hover:** `pointerdown/move` map touch X back to a reading index, drawing a vertical line + per-series dots + a tooltip with the time and each visible series' friendly name (`equipo`) and value; double-tap clears. Label points by the operator's name, never the raw tag. [ev: retro hmi-touch-ux Δ7]
- **Assign each series a distinct color explicitly and mark the shown series bold — never inherited `currentColor` or hover-only emphasis:** a `SERIE_COL` palette per series (and its legend chip); visible series get `.on` (opacity 1, wider stroke) applied on filter, not only on hover. [ev: retro hmi-touch-ux Δ6]

## Plano overlay
- **The frame must carry EXACTLY ONE `aspect-ratio` declaration, equal to `IMG_W/IMG_H` — and the fix for a stale one is to DELETE it, never to shadow it:** the zone overlay aligns only when four values agree with the plano image (the `#plano` image, `IMG_W/IMG_H`, the zones `viewBox`, and the frame `aspect-ratio`); a leftover `.frame{aspect-ratio:1247/771}` masked by a higher-specificity `#frame` rule silently returns the offset if the id is renamed. Treat the four as one atomic unit; check `grep -c 'aspect-ratio'` for the plano frame == 1. [ev: retro hmi-touch-ux Δ9]
- **Fix a label/polygon-centroid collision with an optional per-room `lbl:[x,y]` (in %) override, not by re-drawing geometry:** adjacent or concave rooms get near-identical centroids; `cu.lbl ? cu.lbl : centroid` is minimal and reversible. [ev: retro 5rooms #5]
- **Read the REAL `#plano` src before trusting an on-disk file — it may be an inline base64 image, and `rc/img/plano.png` may be orphaned:** the current build's plano is an embedded base64 (1248×891), not the stray on-disk `plano.png` (1247×771). [ev: retro hmi-touch-ux Δ10]

## Triage — a UI element is "missing"
- **Before debugging a "missing" control, rule out a stale deployed `-ux` in this order: (a) confirm it renders in the CURRENT `src/rc` on `/hmi`, (b) `unzip -p` the deployable jar and grep for it, (c) THEN suspect the deployed build — do not touch code until the live-vs-source gap is ruled out.** [ev: retro hmi-touch-ux Δ3]

## Deploy on a JACE
- **Never set a raw servlet path as a User's Home Page on a JACE:** `/dashboardpan/` is not a valid ORD → `SyntaxException "Missing scheme name"` → `BUser.getHomePage` throws → `AuthenticationException: Login Failed`. Land the operator with a browser bookmark or a redirect ORD/PX instead. [CERT-live 2026-09-01 · bitácora 5cuartos §9]
- **Dashboard write access on a JACE is DEPLOY-side config, not code:** `checkCanWrite` is fail-closed and needs `OPERATOR_WRITE` GLOBAL + a non-null Web Profile + a category granting operatorRead on the service; the log line `[<mod>] checkCanWrite:` states the reason (401/403/user-not-found). [ev: bitácora 5cuartos §8]

## HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)
- **No page scroll** at the panel resolution. Chrome (header+nav+footer) ≈180px → page content ≤ ~620px. If an entity has many fields, use a **per-entity selector** (one at a time) + a 2-column grid, not all side-by-side.
- **Touch:** targets ≥44px (52px for primary); edit via **+/− steppers**, not type-a-number (no keyboard); toggle switch for booleans; one "Guardar cambios" per entity. Neutralize hover-only states.
- Kiosk meta: `user-scalable=no`, `touch-action:manipulation`, `overscroll-behavior:none`.
- Verify every page fits at exactly 1280×800 (headless Chrome or dashboard-preview.py `/hmi`).

## Real alarms (Phase B — see corpus B6/B8, chihuahua B166)
- A `BAlarmSourceExt` can only host on a **control point** (`BNumericPoint`), not a plain `BComponent` → add child points fed from the display slots; attach `BAlarmSourceExt`(`offnormalAlgorithm=BOutOfRangeAlgorithm`, a fault algorithm). Limits configured from the dashboard = writable config slots (per-role-per-room) linked/fed into the algorithm.
- Read via `GET /api/alarms` = BQL over `station:|alarm:|bql:select * where sourceState='offnormal' or 'fault' order by timestamp desc`. Station needs a `BAlarmService` + `BAlarmClass`.
- ACK from a plain servlet is the hard part (chihuahua only got it via BajaScript) — spike `svc.getAlarmDb().getDbConnection().getRecord(uuid)` → `svc.ackAlarm(live)` before committing.

See also: `docs/module-best-practices.md` §2 (the X-Requested-With rule + the CSRF-guard↔header pairing).
