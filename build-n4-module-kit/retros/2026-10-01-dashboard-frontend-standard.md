<!-- review-status: pending -->
# 2026-10-01 · kit · dashboard-frontend-standard

**Session**: PANCCADIA HMI freeze triage → user-requested frontend standard (structure, naming, data, memory, transfer); read-only audit of DashboardPan-ux `rc/index.html`
**Delta count**: 14

## What happened
The user asked for a complete frontend standard for dashboard modules (structure, distribution, organization,
naming, data handling, memory, transfer) and shared a generic web-frontend rule list and a generic BMS layout
tree as references. A read-only audit of DashboardPan-ux `src/rc/index.html` (agent audit, three high-severity
findings re-verified by the parent) measured the real state of our only production servlet-SPA. The kit has
scattered dashboard rules (`types/dashboard.md`, `rc-scan.sh`) plus today's two retros
(`2026-10-01-dashboard-rc-file-split.md` = S, `2026-10-01-dashboard-frontend-reliability-rules.md` = R), but no
single frontend standard. This retro proposes one file, `types/frontend-standard.md` (Appendix A), adapting the
generic list to our context (no framework, no build step, Chrome 83 kiosk WebView at 1280×800, 24/7, 5 s polling,
Niagara servlet), and the lints/harness changes that make its checkable rules bite. Rules already proposed in S/R
are referenced, not repeated. DashboardPan is NOT changed by this retro; its findings are listed in Appendix B as a
backlog that needs explicit user authorization (decision-logic-decomposition Δ3).

## Evidence
- Page weight: `index.html` 3,647,673 B; base64 data URIs 3,465,482 B = 95.0% (parent re-measured with `wc -c` + python regex) `[ev: Cliente/panccadia-leon 42fa82e]`
- Write result ignored: `saveRoom` POST `.then(() => { it.dirty = false; renderIt(it); })` discards the response body; `Promise.all` with a single generic status `index.html:2276-2291` (parent re-verified) `[ev: Cliente/panccadia-leon 42fa82e]`
- Server text into `innerHTML` unescaped: alarm `a.message`, `a.sourceLabel || a.source` `index.html:2493-2497` (parent re-verified) `[ev: Cliente/panccadia-leon 42fa82e]`
- Six live-state stores written in one function (`data[]`, `lastEq`, `procData`, `hoaState`, `spInputs`, `serverAlarms`); `lastEq = j` before validation `index.html:2552-2637` `[ev: Cliente/panccadia-leon 42fa82e]`
- Slot keys hand-built by concatenation and matched by hand to `DashboardReader.java:81-273` arrays `[ev: Cliente/panccadia-leon 42fa82e]`
- Hidden-tab work per poll: `renderChart` rebuilds the whole SVG `index.html:2343-2407`; full payload fetched every 5 s regardless of tab `index.html:2813` `[ev: Cliente/panccadia-leon 42fa82e]`
- Payload estimate ~620-650 keys ≈ 28-32 KB/poll [INFER from reader arrays; not measured on the wire]; `preview-mock.json` has 25 keys `[ev: Cliente/panccadia-leon 42fa82e]`
- Orphan page: `section#page-graficas` `index.html:720` has no `nav-item data-page="graficas"` (nav `index.html:547-552`) `[ev: Cliente/panccadia-leon 42fa82e]`
- Cleared suspicions (no defect): no timer accumulation, no unbounded arrays (`hist`/`chartTimes` ≤ 240), no per-poll listener leak in the common path `[ev: Cliente/panccadia-leon 42fa82e]`
- Version drift: the Chrome 83 login fix `f12f0fe` bumped Dashboard to 2.8.1; the PANCCADIA JACE runs 2.8.0 (Software Manager screenshot, 2026-10-01) `[ev: f12f0fe]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Create `types/frontend-standard.md` with the Appendix A rule set; link it from `types/dashboard.md` § `ux — servlet + SPA` and from the skill's type table row "Dashboard (facade + servlet + SPA)". | NEW `types/frontend-standard.md` + `types/dashboard.md` § `ux — servlet + SPA` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ2 | `rc-scan.sh` WARN `innerhtml-server`: an `innerHTML`/`insertAdjacentHTML` assignment that concatenates a non-literal identifier member (e.g. `+ a.message`) without an `esc(` call. | `toolbelt/rc-scan.sh` header | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ3 | `rc-scan.sh` `datauri-budget`: a `data:` URI over 20 KB in any `rc/**/*.html|css|js` → FAIL for new modules, WARN with `--legacy`; images ship as separate `rc/img/*` files with `?v=`. | `toolbelt/rc-scan.sh` header | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ4 | `rc-scan.sh` advisory WARNs: function body over ~60 lines; symbol declared once and never referenced (dead code); `typeof x !== "undefined"` guard on an own top-level symbol. | `toolbelt/rc-scan.sh` header | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ5 | `rc-scan.sh` `orphan-page`: every `section#page-<id>` has a nav entry `data-page="<id>"` and vice versa. | `toolbelt/rc-scan.sh` header | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ6 | Slot-key contract: the SPA reads keys from one generated `js/slots.js` (or a constant table), and a JUnit test diffs it against the `DashboardReader` slot arrays; no `"Cuarto" + n + "/..."` concatenation spread through render code. | `types/dashboard.md` § `Dashboard as an external API (port spec)` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ7 | Write confirmation contract: write endpoints answer `{ok, value, reason}` with the value actually stored; the client checks `ok`, shows per-field result, applies the echoed value and marks it "confirmed"; a rejected value is shown as rejected, never silently reverted by the next poll. | `types/dashboard.md` § `Critical-write step-up auth` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ8 | `dashboard-preview.py` reports bytes per `/api/*` response, requests per minute and page weight against a budget (proposed: shell HTML ≤ 200 KB, poll payload ≤ 16 KB, ≤ 1 request per poll per visible page); mocks must cover the full reader key set. | `tools/dashboard-preview.py` (niagara-research) + `types/dashboard.md` § `Deploy on a JACE` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ9 | Layout shell for dashboards: header (identity, global status, active alarms, active write session) → nav → content; top nav bar on 1280×800 kiosks (side nav only for desktop dashboards with many sections); tabs named by site system; no orphan pages. | `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ10 | Lint/format toolchain: an ESLint flat config in the kit (`ecmaVersion` matching the Chrome 83 floor, browser globals, `no-unused-vars`, `max-lines-per-function` 60 warn, `no-console` except `error`, `eqeqeq`) run with Node in WSL on `rc/js`; Prettier optional; wired into `report-module.sh` for `-ux` profiles. | `toolbelt/report-module.sh` + `types/frontend-standard.md` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ11 | Component-oriented `rc/js/` layout (Appendix C §C1), mapping the React-style tree (`components/ pages/ services/ hooks/ types/`) onto plain classic scripts: `components/` render functions taking a model slice, `pages/` one per nav tab, `services/` one per API resource on top of `apiFetch`, `store.js` replacing hooks (poll loop + subscribe), `types.js` JSDoc typedefs checked with `tsc --checkJs --noEmit` in WSL. Refines S Δ1. | `types/frontend-standard.md` § A2 | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ13 | Prototype-to-module rule: a standalone design HTML (sim data) is DECOMPOSED into the Appendix C §C1 layout when it becomes a module; live data enters as the `services/` + `store.js` layer, never as an appended block that reassigns the prototype's global functions; the sim mode survives only as a mock service selected by `config.js`. Complements the existing "module is the SKELETON" rule. | `types/dashboard.md` § `Extending an existing dashboard` | `[ev: Cliente/Juarez/Umbrella UmbrellaDashboard-ux index.html:3259-3295]` |
| Δ14 | Vendored browser libraries live as separate files in `rc/vendor/<lib>-<version>.min.js` (+ its LICENSE file), pinned, loaded with `?v=`, never pasted inline into `index.html` and never fetched from a CDN at runtime; deprecated builds (e.g. three.js legacy `build/three.min.js`, deprecated since r150) are recorded with a migration note. | `types/dashboard.md` § `ux — servlet + SPA` (third-party browser library recipe) | `[ev: Cliente/Juarez/Umbrella UmbrellaDashboard-ux index.html:23-30]` |
| Δ12 | Technology decision table (Appendix C §C2): the station servlet stays the backend for on-station HMIs (no Spring Boot on a JACE: Java 8 Compact3); a framework build (Vite + TypeScript, optionally React/Preact) is allowed for NEW `-ux` SPAs only as a pre-built bundle in `src/rc/` with `build.target` = the panel engine (Chrome 83) and a bundle budget; Spring Boot/React/WebSocket stacks belong to off-station multi-site or cloud viewers, not inside the module. | `types/frontend-standard.md` (new § Technology choices) + `types/dashboard.md` § `JS build strategy — grunt vs. no grunt` | `[ev: corpus B1023 §ND4]` |

## Lessons
- 95% of our HMI page was images; measure page weight before reasoning about performance.
- A write without a confirmed read-back is a guess shown to the operator as a fact.
- One normalized model per poll beats six stores written by one function.
- Generic web rules need adapting: no build step, fixed kiosk screen, 24/7 runtime, Chrome 83.
- Audit findings on a deployed module become a backlog, never an automatic refactor.

---

## Appendix A — proposed `types/frontend-standard.md`

Scope: servlet-SPA dashboards (`-ux` `src/rc/`), no framework, no bundler. S = rc-file-split retro, R = frontend-reliability retro.

**A1. Layout and visual consistency**
- Fixed shell: header (logo, site/area name, global status, active alarms, active write session) → nav → content (Δ9).
- One design-token set in `css/base.css` (`--ok`, `--warn`, `--alarm`, `--offline`, `--manual`, spacing, radii, fonts); no hard-coded colors outside it.
- One component per visual pattern (button, chip, card, table row, modal, HOA selector, toast) styled once, reused everywhere.
- Kiosk HMI targets its panel resolution exactly (1280×800); responsive breakpoints only for dashboards also opened from desktop/mobile browsers.
- No decorative animation; motion only for state change feedback.

**A2. Structure and organization**
- File layout per S Δ1 (`index.html` markup only; `css/` by layer; `js/` by responsibility).
- Layers: `api.js` (transport) → `model.js` (normalize/validate) → `render/*.js` (DOM) → `main.js` (wiring). Render never fetches; api never touches the DOM.
- No free globals: each file exposes one namespace object; state lives in one `store`.
- Functions ≤ ~60 lines; one responsibility each (Δ4).
- No dead code, no commented-out code, no temporary `console.log`.

**A3. Naming**
- Identifiers in English (camelCase functions/variables, PascalCase constructors, UPPER_SNAKE constants); user-visible text in Spanish, kept in one `strings.js` table.
- Files kebab-case (`room-detail.js`).
- No magic numbers or ordinals: name every enum map (`HOA = {AUTO:0, ON:1, OFF:2}`) and sentinel ("0 = disabled") once, matching the rt contract.
- Slot keys from `slots.js` only (Δ6).

**A4. Data and state**
- No hard-coded site data (rooms, equipment, limits, names) that the station can provide; static layout config (positions, images) lives in one frozen config file.
- Poll cycle: fetch → validate (plain object, expected keys/types) → build one immutable model → commit once → render. Parse/contract errors are reported separately from transport errors.
- Accessors are fail-closed: a missing value or missing status is "no data", never "ok".
- Do not persist server-owned state in `localStorage`; browser storage only for per-viewer conveniences (last tab).
- Time series store `(t, v)` pairs (or explicit nulls) so gaps never misalign timestamps.

**A5. Network and transfer**
- One `apiFetch` helper: base URL, `X-Requested-With`, timeout (R Δ1), error mapping; no `fetch` elsewhere.
- No overlapping polls (R Δ2); poll scope follows the visible page; secondary data (alarms) at its own slower cadence.
- Budgets measured in the preview harness (Δ8); images as separate cacheable files with `?v=` (S Δ2, Δ3).
- No secrets in JS; config (base path, poll period) in one `config.js`.

**A6. Writes and commands**
- Client-side validation before sending (type, range from the slot facets, required); server re-validates (DWS1).
- Confirmation dialog for critical commands (equipment ON/OFF, alarm reset, setpoint outside normal band, delete).
- Write confirmation contract (Δ7): pending → confirmed/rejected per field; never silent revert.
- Optimistic UI only with a pending flag and rollback on failure.
- Hiding a button is UX, not security: authorization is server-side (step-up auth section).

**A7. Rendering, memory and 24/7 operation**
- Render only the visible page; hidden pages are marked dirty and rendered on show.
- Patch interactive panels in place; never rebuild a panel with focus or an active press (R Δ6).
- Cache DOM references; write text/classes only on change; batch reads before writes in one `requestAnimationFrame`.
- Every timer and listener has a known owner and is created once; bounded buffers only (document the cap).
- Recovery per R Δ3/Δ4.

**A8. Robustness and security**
- Server-derived text via `textContent` or `esc()` only (Δ2).
- No swallowed errors; log on state transitions, not on every failed poll.
- User-facing errors are plain Spanish messages ("No fue posible guardar el setpoint"); technical detail goes to the console.

**A9. Industrial states and units**
- One state vocabulary with one visual treatment everywhere: En línea, Apagado, Encendido, Alarma, Manual, Automático, Deshielo, Sin comunicación.
- Never rely on color alone: state text or icon accompanies the color.
- One formatter per unit (°C 1 decimal, psig 0-1 decimal, A 1 decimal, h integer, durations `h:mm`), date/time `es-MX` 24 h.

**A10. Accessibility and touch**
- Touch targets ≥ 44 px; labels on every input; contrast checked on the panel, not the desktop.
- Feedback that persists until acknowledged for errors (a 3 s toast is too short on a kiosk).
- Do not `disabled` a gate entry control (R Δ7).

**A11. Quality, tests and versioning**
- ESLint config for the Chrome 83 floor (Δ10) and the feature-floor lint (R Δ5).
- Pure logic (classification, unit conversion, timing) is unit-tested in Node (DJS1); source-structural JUnit tests only for wiring.
- `defaultModuleVersion` bump on any `rc/` change; the API contract is versioned and the SPA tolerates unknown keys.
- No large change goes to a station without review, the verify gate and a preview pass.

## Appendix C — adapting the React/Spring reference to Niagara modules

**C1. `rc/js/` layout (classic scripts, no bundler)** — same separation as the React tree, without the build step:
```
rc/
  index.html                 markup + ordered <script src="...?v=X.Y.Z">
  css/ base.css (tokens) · layout.css · components.css
  js/
    config.js                pollMs, timeouts, base path (one object)
    types.js                 JSDoc @typedef Room, Compressor, Alarm, WriteResult  (tsc --checkJs)
    slots.js                 slot-key table generated from DashboardReader arrays (Δ6)
    services/ api.js (apiFetch) · equipment.service.js · alarms.service.js · setpoint.service.js
    store.js                 poll loop + validate + immutable model + subscribe()  (the "hooks" role)
    components/ header.js · status-badge.js · metric-card.js · equipment-card.js
                alarm-table.js · trend-chart.js · hoa-selector.js · toast.js
    pages/ plano.js · sensores.js · condensadoras.js · alarmas.js · configuracion.js · control.js
    main.js                  boot: wire store → visible page
```
Rules: a component is a pure function `(container, slice) → void` that patches DOM; a page composes components and subscribes to the store only while visible; services never touch the DOM; only `api.js` calls `fetch`.

**C2. Technology choices**
| Situation | Use | Why |
|---|---|---|
| On-station HMI / station web UI (JACE or Supervisor) | Niagara servlet (`BWebServlet`) as backend + static SPA in `-ux` `rc/` | The station already owns auth, alarms, histories and the points; the JACE JVM is Java 8 Compact3, so Spring Boot cannot run there `[ev: types/third-party-libraries.md:112]` |
| Small/medium SPA (today's dashboards) | Plain JS classic scripts + JSDoc types + ESLint (C1) | No build step; loads on Chrome 83; testable with Node |
| Large NEW SPA with many screens or 3D | Vite + TypeScript (React or Preact) built OUTSIDE gradle to a pre-built bundle in `src/rc/`, `build.target: 'chrome83'`, bundle budget enforced in preview (Δ8) | Components and typed contracts; the kit's pre-built bundle recipe already covers serving it `[ev: types/dashboard.md:82]` |
| Charts on the panel | Hand-rolled SVG per `types/dashboard.md` § Charts on an HMI; ECharts only if the bundle budget allows | ECharts adds hundreds of KB to a panel that reloads the shell [INFER] |
| Live data inside the station | REST polling (5 s) through the servlet | No BOX/Fox subscription in a plain `BWebServlet` `[ev: types/dashboard.md:27]` |
| Off-station, multi-site or cloud viewer | Separate frontend (React/Three.js) + its own backend (Supabase/Spring Boot) reading oBIX; writes through one gated write service | Different trust boundary; never expose the station to browsers directly (PANCCADIA viewer pattern) |
| Styling | CSS design tokens (A1); Tailwind/Material UI only with a framework build | Utility CSS needs a build step |

Decision rule: React does not replace the backend — in a module the backend is the station. Pick the framework only when the SPA size justifies a build pipeline, and always target the panel engine.

## Appendix D — UmbrellaDashboard-ux audit (2026-10-01, parent-measured)
| Item | Measured | Evidence |
|---|---|---|
| Shape | single `rc/index.html`, 2,734,850 B, 3,298 lines; 1 `<style>` (19 KB in 8 lines), 3 `<script>` blocks | `wc -lc`, python block split |
| Base64 | 1,899,078 B = 69.4% (one line ≈ 1.78 MB in the app block) | python regex |
| Embedded libraries | three.js legacy global build (669 KB, prints its own r150+ deprecation warning) + OrbitControls inline | `index.html:23-30` |
| App code | ~293 lines, 47 over 400 chars, ~50 functions | awk on lines 2966-3258 |
| Live wiring | appended block reassigns globals `reading`/`currentState`; `fetch` without timeout; `setInterval` polling; `.catch(function(){})`; `r.ok ? r.json() : null` drops errors silently; NO reload watchdog | `index.html:3259-3295` |
| Slot keys | built by concatenation `'Unit' + n + '/supplyTemp'` | `index.html:3261, 3272-3280` |
| Good practice to keep | CSS tokens in `:root`; central SVG `ICONS`; consistent status badges (fault/down/stale/overridden) and fail-closed `offline` when no data | style head; `index.html:2967` (ICONS); `index.html:3279` (offline) |

Backlog (needs explicit authorization): U1 add fetch timeout + success watchdog (R Δ1-Δ3); U2 vendor three.js as files and plan the migration off the legacy build (Δ14); U3 move the 1.78 MB base64 asset to a separate file (Δ3); U4 decompose per Δ13 when the module is next extended.

## Appendix B — DashboardPan backlog from the audit (needs explicit authorization)
| # | Finding | Evidence |
|---|---|---|
| B1 | Write response body ignored; partial-failure status generic; rejected values silently reverted | `index.html:2276-2291` |
| B2 | Alarm `message`/`sourceLabel` concatenated into `innerHTML` | `index.html:2493-2497` |
| B3 | 95% of page is base64 (plano 1.04 MB, room photos 2.37 MB); HTML is `no-store`, so every reload downloads 3.65 MB | `index.html:574,820-828`; `BDashboardServlet.java:658` |
| B4 | Six live-state stores; `lastEq` set before validation; missing `st` defaults to "ok" | `index.html:2552-2637` |
| B5 | Hidden-tab work every poll (`renderChart` SVG rebuild, `renderCondensadoras`); full payload regardless of tab | `index.html:2343-2407, 2634-2636, 2813` |
| B6 | `buildHoa` rebuilds the Control panel every poll while visible | `index.html:1488, 1657` |
| B7 | HOA optimistic update without rollback; `hoaSave` localStorage overwritten each poll | `index.html:1462-1480, 1402, 2625-2637` |
| B8 | `hist`/`chartTimes` misalignment on sensor gaps [INFER] | `index.html:2547, 2620, 2400` |
| B9 | Dead code (`logAlarm`, `simulate`, `readObix`); 5 duplicated XHR header builders | `index.html:1073, 2458, 2528-2543, 2292` |
| B10 | Orphan page `page-graficas` (no nav entry) | `index.html:720, 547-552` |
| B11 | `console.error` every failed poll (~17k lines/day while down) | `index.html:2823` |
| B12 | JACE runs 2.8.0; Chrome 83 login fix is in 2.8.1 | `f12f0fe` |
