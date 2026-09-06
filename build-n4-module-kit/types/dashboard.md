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

- **A per-room link-in boolean alarm (e.g. door-open) is a `SUMMARY` slot — an INPUT, not OPERATOR config:** one `boolean` on the facade (SUMMARY link-in), one entry in the reader's boolean loop, one generic frontend read; the zone polygon force-reds via a CSS class. Keep it on every room so the frontend stays generic. [ev: retro dashboardpan-detail-render-doors · U4]
- **A process/defrost timer shown on the dashboard is a facade+reader path — the facade gains the anchor as a READONLY link-in slot AND the reader gains a `BAbsTime` type-reader (without the reader entry the slot is invisible):** store the event instant as `TRANSIENT|SUMMARY|READONLY` `BAbsTime`; the reader emits DERIVED keys (elapsed/remaining ms from a single `Clock.millis()`) so the SPA ticks between polls. [ev: retro process-timers-and-defrost-audit + L16]
- **Off-station consumers (oBIX batch, 3D viewer) must recompute servlet-derived keys from the anchor slots:** `*ElapsedMs`/`*RemainingMs` are computed by `DashboardReader` at read-time and do NOT exist in the oBIX facade — a facade poller will never find them. Read the ANCHOR slots (`coolingSince`, `defrostStart`, `defrostDuration`, `nextDefrostTime`) and recompute `now − anchor`. Read the reader source to tell a real oBIX slot from a derived value. [ev: retro dashboardpan-2d-to-3d-port Δ2]

## ux — servlet + SPA
- **Pick the right serving recipe first (three exist):** a bespoke dashboard with a custom JSON API → **servlet-SPA** (a `BWebServlet` that self-registers by `getServletName()`); a view bound to a component TYPE → **bajaux `@AgentOn` + `BIJavaScript`** (`getJsInfo` → `module://…/rc/X.js`); engineer-authored equipment graphics → **PX** (XML bound to slot ords, no Java). **Our DashboardPan is servlet-SPA — the correct choice for a bespoke HMI; do not migrate it to bajaux.** [ev: retro corpus-index · B752]
- `BWebServlet` subclass; `getServletName()` = the mount prefix → `/<prefix>/`. Registered as a `<type>` in module-include.xml + a palette instance under a Servlets folder.
- `doGet/doPost` delegate to a **pure router** (WSL-unit-testable, no Niagara deps). Guards in order: path-traversal→404; **XHR guard** on `/api/*` (missing `X-Requested-With: XMLHttpRequest` → 302); unknown `/api/*`→404; static fallback (`/`→index.html).
- **DUX1 — `-ux` servlet testable seam (route() → RouteAction):** extract routing into a package-private `final` class taking `Function` header/param lookups and returning a sealed `RouteAction`; the servlet becomes a thin `instanceof` adapter. Test the router with plain JUnit + `HashMap` — no station required. `DashboardDispatch` (14 `@Test`, 0 Baja imports) is the proven pattern. [ev: corpus B762]
- **DUX2 — purity gradient:** keep data-shapers pure by injecting the Baja touch as a `Function`/`Predicate`; `DashboardReader.buildEquipmentResponse(BComponent)` (takes a live component) is the anti-pattern — `JsonUtil` (0 Baja imports) is the target shape. [ev: corpus B762]
- **DJS1 — testable SPA JS:** split inline JS into files + add a dual-export shim (`if(typeof module!=='undefined')module.exports=…`) + a Node harness for logic tests; `node --check` proves syntax only. DashboardPan's 2300-line inline `index.html` is the anti-pattern. [ev: corpus B762]
- **Reader**: walk the facade → flat JSON keyed by slot-path, each value `{"v":num|bool|null,"st":"<status>"}`, fault-aware, `Locale.ROOT` doubles, escaped strings.
- **RBAC**: `checkCanWrite` first line of every write: `BPermissions.has(OPERATOR_WRITE)` (bit, not role name), fail-closed, `SlotPath.unescape` the username. Audit each mutation. **Enforce this server-side even though the vendors don't** — the Honeywell React SPAs ship `permissions="unrestricted"` RPCs with NO server check; do NOT copy that, the browser is never the security boundary. [ev: retro corpus-index · B752]
- **DWS1 — Write-surface: five gates (HIGH — apply to every mutating endpoint):** (1) `checkCanWrite` first line = `OPERATOR_WRITE` fail-closed (deny on no-user / no-service / `catch(Exception)`); (2) hand-rolled `X-Requested-With` guard IN the pure `route()` (framework CSRF filter covers `/rpc/*` only) — a CRITICAL-write endpoint should ALSO verify the real x-niagara-csrfToken token (CsrfUtil double-submit), not rely on X-Requested-With alone. [ev: corpus B803]; (3) pin the write ORD under a `SERVICE_ORD` facade + traversal reject OR an explicit slot allowlist; (4) mutate under a per-Ord lock → HTTP **423** on contention; (5) audit every mutation (who/what/when/old→new), fire-and-forget, audit-failure never fails the write. [ev: corpus B763]
- **DWS2 — Pure RBAC test seam `canWrite(boolean)` (HIGH):** extract the auth DECISION as a Baja-free `canWrite(boolean)` / `getRole` / `buildForbiddenJson` seam so write-auth is unit-testable off-station. `DashboardRbacHelper`'s collapsed Baja-only form (no pure `canWrite`) is the anti-pattern; chihuahua's `ChiRbacHelper` ADR D1 is the proven seam. [ev: corpus B763]
- **A generic per-ORD write endpoint MUST gate on more than the global RBAC bit — whitelist writable slots AND/OR check the target's `Flags.OPERATOR` before `set()`:** `POST /api/setpoint` doing `parent.set(prop, coerce(...))` on ANY settable property under the service ORD, guarded only by the single global `OPERATOR_WRITE` bit + the XHR header + anti-traversal, lets anyone with that bit write display/state slots the HMI treats as read-only — the write-surface is wider than the read-surface. Before `set()`, whitelist the writable slots or check `getFlags().contains(Flags.OPERATOR)` on the target, and declare the single-global-bit RBAC as an explicit decision. [ev: retro dashboard-servlet-write-surface · U5]
- **The reader's group arrays are the AUTHORITY for what a dashboard shows/controls — not the property sheet or the facade class:** a slot is on the HMI only if it appears in `DashboardReader`'s arrays (`TEMP_SLOTS`, `NUM/RELTIME/BOOL_CONFIG_SLOTS`, `DOOR_SLOTS`, `STATE_SLOTS`, `HOA_MODE_SLOTS`). Read those to document/verify a dashboard's surface; flag 1→N facade slots (e.g. `startDelay` fanning to N evaporators) for the wirer. [ev: retro dashboard-servlet-write-surface · U6]
- **A dashboard-written value's slot type is load-bearing:** a `double`/`BRelTime`/`boolean` is oBIX-writable directly; a `BStatusNumeric` needs the wrapped `<obj><real name="value">` body (never attr-only → silent 0.0) or the servlet/action path — see `types/logic-authoring.md` §"Slot types for externally written values". `[ev: corpus B823]`
### Reference exemplar (our own module) — DashboardPan-ux `[ev: corpus B796]` [ev: retro dashboard-exemplar]

Tridium ships no vendor exemplar for the SPA/servlet split — DashboardPan-ux is the reference (B791 THIN verdict). [ev: corpus B791]

**Routing seam (DUX1):** `DashboardDispatch` is `package-private final` (`DashboardDispatch.java:30`); `route()` returns a sealed `RouteAction` hierarchy (`:41-43`); servlet (`BDashboardServlet.java:91-102,132-153`) is a thin `instanceof` adapter. `DashboardDispatchTest`: 14 `@Test`, 0 Baja imports, 0 station needed. [ev: corpus B796]

**DUX2 anti-pattern (same module):** `DashboardReader.java:66` is `public final` but impure — 15+ `javax.baja.*` imports (`:6-20`); `buildEquipmentResponse(BComponent)` takes a live component (`:143`). The exemplar exposes both seams side by side. [ev: corpus B796]

**Five DWS1 gates scored on real code:**

| # | Gate | Status | File:line |
|---|---|---|---|
| 1 | `checkCanWrite` first — `OPERATOR_WRITE` fail-closed (no-user / exception → deny) | ✅ met | `BDashboardServlet.java:198`; `DashboardRbacHelper.java:33,40-46,55-61,98,100-104` |
| 2 | `X-Requested-With` guard inside pure `route()` | ✅ met | `DashboardDispatch.java:123-126` (POST), `:144-147` (/api GET) |
| 3 | ORD pinned under `SERVICE_ORD` + traversal reject | ✅ met | `BDashboardServlet.java:222-223,241,247-248` |
| 4 | per-Ord lock → HTTP **423** on contention | ❌ REQUIRED-but-absent → issue #49 | no 423/lock in `BDashboardServlet.java` (grep 0 hits) [ev: corpus B763] |
| 5 | audit fire-and-forget; failure never fails the write | ✅ met | `BDashboardServlet.java:286-301` |

- Static assets in `src/rc/`; gradle copies `src/rc→rc/`; served via `getClassLoader().getResourceAsStream("rc/"+path)`.
- Frontend: ES5 + `fetch`, REST poll. **Every fetch (reads too) sends `X-Requested-With`** or the guard 302s it → page loads but data shows "--". Use ABSOLUTE `/<prefix>/...` URLs (relative break without a trailing slash). Write via `POST /api/setpoint {ord,value}`.

## Critical-write step-up auth `[ev: corpus B803]`
- Niagara ships NO core credential step-up — `Flags.CONFIRM_REQUIRED` is a UX-only confirm; the `electronicSignature` module is the only true sign-before-invoke. Step-up is MODULE-level code. `[ev: corpus B803]`
- A mutating `-ux` endpoint whose target is a CRITICAL control adds, ON TOP of the B763 five gates: (1) a SERVER-SIDE criticality allowlist of target ORDs/actions that REQUIRE step-up (never client-decided); (2) re-verify the session user through their auth scheme SERVER-SIDE; (3) issue a fresh short-TTL step-up TOKEN (2–5 min) bound to `(sessionId + user + target ORD + purpose)`, checked server-side on the write — NEVER client-only. `[ev: corpus B803]`
- SAML/SSO caveat: a SAML user cannot be re-verified server-side mid-session (the browser→IdP redirect hierarchy is [CERT]; the "no in-request re-verify" runtime block is [INFER], B803-G1 pending a live-station confirm) — either reject with "re-login required" or trust the session + a short TTL. `[ev: corpus B803]`
- CSRF: verify the REAL token `x-niagara-csrfToken` (`CsrfUtil.CSRF_TOKEN_HTTP_HEADER`; double-submit against the session token), not `X-Requested-With` alone. `[ev: corpus B803]`
- The per-Ord write lock / HTTP 423 for a critical write is tracked as the client gate-4 gap (issue #49, `dashboard.md:48`). `[ev: corpus B763]`
- OPEN (requires-execution): **B803-G1** (confirm the live SAML mid-session re-auth block on a station) and **B803-G2** (whether gauth's `BPasswordCache.validate` accepts a TOTP token mid-session). `[ev: corpus B803]`

## Extending an existing dashboard
- **Before hand-editing a dashboard SPA, decide DATA vs CODE:** a data-driven render (`CUARTOS.forEach`, `SENSORS.map`) is entity-count-agnostic, so scaling 3→5 rooms is swapping the DATA arrays (`CUARTOS`/`SENSORS`/`FANS`, the plano src, `viewBox`/`IMG_W`/`IMG_H`) with zero change to the render functions. Verify with `node --check` on the extracted `<script>` blocks after. [ev: retro 5rooms #1]
- **The module is the SKELETON; the operator's standalone HTML is only the DATA:** splice the operator's calibrated data blocks into the module's `index.html` (which already carries kiosk CSS, `/api/*` wiring, status classes, alarms, setpoint save) and remap sim ords to the module's Niagara ords — never rebuild the module from the standalone, which loses the integration layer. See `METHODOLOGY.md` §editing technique for navigating the giant single-file source. [ev: retro 5rooms #2]

- **The reusable add-a-tab + wire-data recipe for extending the SPA (no servlet change):** a top-level tab = one `nav-item` + one `<section>` toggled by id (order pages by id, NOT DOM order); render read-only values from the last `/api/equipment` JSON, keyed by ord (`{v,st,nm}`); add a control row via the shared HOA-row helper and append a non-entity segment to the control selector. [ev: retro editing-base64-heavy-spa · U10]

## Config panel UX on a fixed touch panel
> These build on the `## HMI kiosk` budget below (≥44px targets, no page scroll at 1280×800).
- **When a room's fields overflow two columns at ≥44px targets, partition into named sub-tab GROUPS — not scroll, not more columns:** on a fixed capacitive panel scroll is non-ergonomic and a 3rd column shrinks touch targets (both operator-rejected). Data-drive it: give the room `groups:[{name,fields}]`, render a `.cfg-tabs` bar, keep ONE shared `items[]` + one "Guardar cambios" so edits across tabs save once. [ev: retro hmi-touch-ux Δ1]
- **Write the per-field row-builder as a standalone `buildField(room,f,items)` from the start:** it registers the input, pushes to `items`, and returns the row — so grouping, reordering, and conditional fields all stay cheap. [ev: retro hmi-touch-ux Δ2]
- **Give a config value both a typed input AND +/− steppers, validating and reverting invalid text:** the operator can type or step; min/max validation reverts a bad entry. [ev: bitácora 5cuartos §4]
- **A dense secondary feature gets its own full-width page with an entity selector, not a cramped corner of the per-room panel:** e.g. HOA control moved to a full-width page with a room selector — the same partition principle as sub-tabs, at page scale. [ev: bitácora 5cuartos §4]
- **Writable ranges (min/max/step/decimals) live in the SPA field-descriptor array, not Java — but check the facade and control facets too:** a "-40 floor" was a frontend `SP_COMMON min:-40`, not a servlet clamp; confirm the facade/control facets impose no MIN before concluding the link won't re-clamp. [ev: retro hmi-touch-ux Δ4]
- **Sync config across viewers: run prefill on EVERY poll but SKIP dirty (unsaved-editing) fields, and rebuild an open sub-panel each poll:** otherwise telemetry syncs on the 5s poll but config/HOA load once, so a change by one operator is invisible to the others until reload. [ev: bitácora 5cuartos §10]

- **The 2-column no-scroll rule applies to the Control/HOA panel too — build indivisible group units, not CSS columns over sibling rows:** wrap each group (subheader + rows) in a `.hoa-group` (`break-inside:avoid`) via an `addGroup(fillSub, fillRows)` helper laid out in `grid-template-columns:1fr 1fr`; CSS columns over sibling rows split a header from its rows. [ev: retro hmi-1280x800 · U2]
- **A reusable output row must allow custom buttons + a special-cased prefill — don't assume 0/1/2 from a `*Mode` name:** an AUTO/OFF exchanger (AUTO=1/OFF=0) needs `hoaRow(..., {buttons, valueMap})`; a key ending in `Mode` otherwise falls into the generic `["auto","on","off"][ordinal]` prefill and shows the wrong state — special-case its prefill before the generic branch. [ev: retro hmi-1280x800 · U3]

## Charts on an HMI
- **Size the SVG chart's `viewBox` to the MEASURED box in real px each render — do not carry a fixed art-board viewBox with `preserveAspectRatio="none"`:** measure `getBoundingClientRect()` (guard width>60) and set `viewBox="0 0 <w> <h>"` so 1 unit = 1px and strokes/axis text stop distorting. [ev: retro hmi-touch-ux Δ5]
- **On a capacitive HMI the chart read-out is a TOUCH crosshair (pointer events + `touch-action:none`), never hover:** `pointerdown/move` map touch X back to a reading index, drawing a vertical line + per-series dots + a tooltip with the time and each visible series' friendly name (`equipo`) and value; double-tap clears. Label points by the operator's name, never the raw tag. [ev: retro hmi-touch-ux Δ7]
- **Assign each series a distinct color explicitly and mark the shown series bold — never inherited `currentColor` or hover-only emphasis:** a `SERIE_COL` palette per series (and its legend chip); visible series get `.on` (opacity 1, wider stroke) applied on filter, not only on hover. [ev: retro hmi-touch-ux Δ6]

## Plano overlay
- **The frame must carry EXACTLY ONE `aspect-ratio` declaration, equal to `IMG_W/IMG_H` — and the fix for a stale one is to DELETE it, never to shadow it:** the zone overlay aligns only when four values agree with the plano image (the `#plano` image, `IMG_W/IMG_H`, the zones `viewBox`, and the frame `aspect-ratio`); a leftover `.frame{aspect-ratio:1247/771}` masked by a higher-specificity `#frame` rule silently returns the offset if the id is renamed. Treat the four as one atomic unit; check `grep -c 'aspect-ratio'` for the plano frame == 1. [ev: retro hmi-touch-ux Δ9]
- **Fix a label/polygon-centroid collision with an optional per-room `lbl:[x,y]` (in %) override, not by re-drawing geometry:** adjacent or concave rooms get near-identical centroids; `cu.lbl ? cu.lbl : centroid` is minimal and reversible. [ev: retro 5rooms #5]
- **Read the REAL `#plano` src before trusting an on-disk file — it may be an inline base64 image, and `rc/img/plano.png` may be orphaned:** the current build's plano is an embedded base64 (1248×891), not the stray on-disk `plano.png` (1247×771). [ev: retro hmi-touch-ux Δ10]

- **When a real SPA has several `<svg viewBox>` elements (charts, room maps, etc.), target the plano overlay by id (`id="zonas"` or the semantic id), then fall through to `viewBox` attribute match, never grep file-wide:** a file-wide search finds the first `viewBox` (often a chart) instead of the plano overlay, producing a wrong aspect ratio; the id anchor is both stable and unambiguous. [ev: retro campaign7-plano]
- **Overlay a raster and its vectors in a SINGLE `<svg viewBox>` (`<image xlink:href>` + shapes, `preserveAspectRatio="xMidYMid meet"`), never a raster `<img>` + a separately JS-fitted SVG:** the panel WebView (i.MX8M) ignores CSS `aspect-ratio`/`object-fit`, so a JS-fitted overlay is right in the desktop `/hmi` sim but wrong on the panel. Use `xlink:href` (SVG2 `href` renders blank on old WebKit); size HTML label overlays to the SVG `<image>` rect, not the frame; reproduce the panel path in the sim with `#frame{aspect-ratio:auto}`. Give the frame a DEFINITE height (`height:100%` from the stage, **not `height:auto`**) so the SVG `meet` letterboxes inside it — a near-square image at `height:auto` overflows a shorter frame and the frame's `overflow:hidden` then clips the bottom zone label; a definite height keeps the whole image and every label visible (verified across wide and square rooms). [ev: retro dashboardpan-detail-render-doors · U1]

## Triage — a UI element is "missing"
- **Before debugging a "missing" control, rule out a stale deployed `-ux` in this order: (a) confirm it renders in the CURRENT `src/rc` on `/hmi`, (b) `unzip -p` the deployable jar and grep for it, (c) THEN suspect the deployed build — do not touch code until the live-vs-source gap is ruled out.** [ev: retro hmi-touch-ux Δ3]

## Deploy on a JACE
- **Never set a raw servlet path as a User's Home Page on a JACE:** `/dashboardpan/` is not a valid ORD → `SyntaxException "Missing scheme name"` → `BUser.getHomePage` throws → `AuthenticationException: Login Failed`. Land the operator with a browser bookmark or a redirect ORD/PX instead. [CERT-live 2026-09-01 · bitácora 5cuartos §9]
- **Dashboard write access on a JACE is DEPLOY-side config, not code:** `checkCanWrite` is fail-closed and needs `OPERATOR_WRITE` GLOBAL + a non-null Web Profile + a category granting operatorRead on the service; the log line `[<mod>] checkCanWrite:` states the reason (401/403/user-not-found). [ev: bitácora 5cuartos §8]

## Dashboard as an external API (port spec)
- **A dashboard's reusable CONTRACT is a first-class deliverable when consumed off-station (e.g. a 3D viewer over oBIX):** document the port spec once — the flat `CuartoN/slot → {v,st}` map (identical between `/api/equipment` and oBIX), which slots are OPERATOR-writable vs display, and per-type encoding (°C / HOA 0-1-2 / bool / RelTime in ms). Hand off a big single-file SPA as a line-range map, not a paste, and warn about the embedded base64. [ev: retro dashboard-contract-port-spec · U7]

## HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)
- **No page scroll** at the panel resolution. Chrome (header+nav+footer) ≈180px → page content ≤ ~620px. If an entity has many fields, use a **per-entity selector** (one at a time) + a 2-column grid, not all side-by-side.
- **Touch:** targets ≥44px (52px for primary); edit via **+/− steppers**, not type-a-number (no keyboard); toggle switch for booleans; one "Guardar cambios" per entity. Neutralize hover-only states.
- Kiosk meta: `user-scalable=no`, `touch-action:manipulation`, `overscroll-behavior:none`.
- Verify every page fits at exactly 1280×800 (headless Chrome or dashboard-preview.py `/hmi`).

- **`dashboard-preview.py` has `--editor` (a preview-only overlay label editor — drag + live rx/ry readout, the sanctioned way to reposition overlays without touching the module) and `--mock <json>` (seeds `/api/*` with realistic `{v,st,nm}`); for an off-site operator, deliver a screenshot/PDF via a Playwright script (`page.click('[data-page=X]')` for a non-default tab) + `chrome --headless --print-to-pdf` with a `@page A4` block.** [ev: retro editing-base64-heavy-spa · U9]

## Real alarms (Phase B — see corpus B6/B8, chihuahua B166)
- A `BAlarmSourceExt` can only host on a **control point** (`BNumericPoint`), not a plain `BComponent` → add child points fed from the display slots; attach `BAlarmSourceExt`(`offnormalAlgorithm=BOutOfRangeAlgorithm`, a fault algorithm). Limits configured from the dashboard = writable config slots (per-role-per-room) linked/fed into the algorithm.
- Read via `GET /api/alarms` = BQL over `station:|alarm:|bql:select * where sourceState='offnormal' or 'fault' order by timestamp desc`. Station needs a `BAlarmService` + `BAlarmClass`.
- ACK from a plain servlet is the hard part (chihuahua only got it via BajaScript) — spike `svc.getAlarmDb().getDbConnection().getRecord(uuid)` → `svc.ackAlarm(live)` before committing.

## Module packaging (palette + lexicon)

- **`module.palette` convention:** bare-Type-minus-`B` instance names, plural category folders, `m="alias=module"` declared once, nested `<p>` to pre-seed ext/config child slots. `[ev: corpus B780]`
- **`module.lexicon` prefixing:** the lexicon is flat + MODULE-GLOBAL — PREFIX keys (`parent.child`, `Type.slot`) to dodge the B759 collision; a missing key renders raw camelCase via `toFriendly`. `[ev: corpus B780]`

## Web-tier exemplars — where the Tridium pattern lives (DUX-WEB1)

This section documents the web tier from OUR modules; the Tridium exemplar for each aspect is in the corpus — reach for it, do not re-derive: servlet routing (`BWebServlet`/`BServletView`) → B29; hx views (`BHxView`/`BHxProfile`/`HxOp`) → B433; module `rc/` web resources + `module://<mod>/rc/…` ORDs → B5/B752; `@AgentOn` web agents (`BIJavaScript`+`JsInfo`) → B752/B421; Tridium-servlet CSRF (`CsrfGuard`/`CsrfProtectedFilter`) → B58 (vs our hand-rolled `X-Requested-With` guard, B763); JSON/REST response shaping → B16/B66, B361–B364, B604, B509. `[ev: corpus B791]`

## DashboardPan divergences from the Tridium web pattern (DUX-WEB2)

- **CSRF via a hand-rolled `X-Requested-With` check inside the pure `route()`** (B763) rather than the framework `CsrfProtectedFilter`/`CsrfGuard` (B58) — a DELIBERATE, STRONGER-than-vendor divergence (vendor bajaux treats `requiredPermissions` as visibility-only and skips server RBAC, B752); keep it, note it as intentional. `[ev: corpus B791/B763]`
- **PUNCH-LIST: the RBAC decision is COLLAPSED into a Baja-bound helper** (DashboardPan) where chihuahua-ux keeps a pure-vs-Baja seam — re-split so the write-auth decision is Niagara-free/unit-testable. `[ev: corpus B763]`

See also: `docs/module-best-practices.md` §2 (the X-Requested-With rule + the CSRF-guard↔header pairing).
