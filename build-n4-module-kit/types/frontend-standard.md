# Type: frontend standard (servlet-SPA `-ux` dashboards) — SEED

The single frontend standard for the browser side of a dashboard module (`-ux` `src/rc/`). The
servlet, reader, facade, RBAC and deploy rules stay in [`dashboard.md`](dashboard.md); this file
covers structure, naming, data, memory, transfer and library choice for the SPA itself.
`[ev: retro dashboard-frontend-standard Δ1]`

Context the generic web rules are adapted to: no framework and no bundler by default, a fixed kiosk
WebView (Chromium 83 at 1280×800 on the HMI panel), 24/7 unattended runtime, ~5 s REST polling, a
Niagara `BWebServlet` as the only backend. Rules that only constrain one audience carry a profile tag.

## Profiles — tag every rule `[ev: retro dashboard-deployment-profiles Δ1]` `[ev: retro dashboard-deployment-profiles Δ2]`

One servlet dashboard can serve two audiences. Declare the module's `ui_profile` (BUILD-STATE row +
module docs, see `BUILD-STATE.md` § How to read this file) before writing a rule or a line of CSS.
This section is the single definition of `ui_profile`; `BUILD-STATE.md` and `dashboard.md` reference it.
`[ev: retro dashboard-deployment-profiles Δ1]`

| `ui_profile` | Audience | Browser floor |
|---|---|---|
| `hmi` | the HMI panel: one kiosk WebView, touch, 1280×800, 24/7, nobody reloads it | Chromium 83 |
| `lan` | anyone on the site LAN with their own browser (desktop/laptop/tablet/phone), several concurrent users, each with their own Niagara login, intermittent use | current evergreen Chrome/Edge/Firefox/Safari |
| `both` | both of the above | see below |
| `unknown` | a module that predates the field (honest, not a default); record the real value the first time its `-ux` is touched | not declared: the tools enforce no profile (`rc-scan.sh` browser-floor stays WARN) |

- `both` with ONE build ⇒ the HMI floor governs everything (Chromium 83, the panel CSS rules in
  `dashboard.md` § HMI kiosk). Alternative: two entry points, `hmi.html` (panel) and `index.html`
  (LAN), sharing `js/` with a per-entry `config.js`, each linted against its own floor.
- Tags used below: **[both]** applies to every profile; **[HMI]** applies to `hmi` and to a
  single-build `both`; **[LAN]** applies to `lan` (and to the LAN entry point of a two-entry `both`).
- A rule without a profile tag is not folded into this file. `[ev: retro dashboard-deployment-profiles Δ2]`

## Rule set `[ev: retro dashboard-frontend-standard Δ1]`

### 1. Layout and visual consistency
- [both] Fixed shell: header (logo, site/area name, global status, active alarms, active write
  session) → nav → content. On the HMI the nav is a top bar; a LAN dashboard may add a side nav and
  responsive breakpoints.
- [both] One design-token set in `css/base.css` (`--ok`, `--warn`, `--alarm`, `--offline`,
  `--manual`, spacing, radii, fonts); no hard-coded colors outside it.
- [both] One component per visual pattern (button, chip, card, table row, modal, HOA selector,
  toast), styled once and reused everywhere.
- [HMI] Target the panel resolution exactly (1280×800); no page scroll (budget in `dashboard.md`
  § HMI kiosk).
- [both] No decorative animation; motion only as feedback for a state change.

### 2. Structure and organization
- [both] File layout per § Component layout below: `index.html` is markup only; no inline
  `<style>`/`<script>` beyond a boot tag. `[ev: retro dashboard-rc-file-split Δ1]`
- [both] Layers: services (transport) → store (normalize/validate) → components/pages (DOM) →
  `main.js` (wiring). Render never fetches; services never touch the DOM.
- [both] No free globals: each file exposes one namespace object; live state lives in one store.
- [both] Functions stay under ~60 lines with one responsibility each; no dead code, no commented-out
  code, no temporary `console.log`. Gated by the kit ESLint config (§ Enforcement).
  `[ev: retro dashboard-frontend-standard Δ4]`

### 3. Naming
- [both] Identifiers in English (camelCase functions/variables, PascalCase constructors,
  UPPER_SNAKE constants); user-visible text in the site's operator language, kept in one
  `strings.js` table. Files are kebab-case (`room-detail.js`).
- [both] No magic numbers or ordinals: name every enum map (`HOA = {AUTO:0, ON:1, OFF:2}`) and
  every sentinel ("0 = disabled") once, matching the rt contract.
- [both] Slot keys come from one `slots.js` table, never built by string concatenation spread
  through render code.

### 4. Data and state
- [both] No hard-coded site data (rooms, equipment, limits, names) that the station can provide;
  static layout config (positions, images) lives in one frozen config file.
- [both] Poll cycle: fetch → validate (plain object, expected keys/types) → build one immutable
  model → commit once → render. Report parse/contract errors separately from transport errors.
- [both] Accessors are fail-closed: a missing value or a missing status is "no data", never "ok".
- [both] Never persist server-owned state in `localStorage`; browser storage only for per-viewer
  conveniences (last tab).
- [both] Time series store `(t, v)` pairs (or explicit nulls) so a sensor gap never misaligns
  timestamps.

### 5. Network and transfer
- [both] One `apiFetch` helper (base URL, `X-Requested-With`, timeout, error mapping); no `fetch`
  anywhere else. Timeout and scheduling rules: `dashboard.md` § Poll loop reliability.
- [both] No overlapping polls; poll scope follows the visible page; secondary data (alarms) at its
  own slower cadence.
- [both] Images ship as separate cacheable files under `rc/img/`, referenced with `?v=`; measure
  page weight before reasoning about performance (base64 data URIs made ~95% of one audited page).
  Budgets are stricter on the HMI.
- [both] No secrets in JS; config (base path, poll period, timeouts) in one `config.js`.

### 6. Writes and commands
- [both] Validate client-side before sending (type, range from the slot facets, required); the
  server re-validates (DWS1 in `dashboard.md`).
- [both] Confirmation dialog for critical commands (equipment ON/OFF, alarm reset, setpoint outside
  the normal band, delete).
- [both] A write is pending until the server confirms it; a rejected value is shown as rejected,
  never silently reverted by the next poll. Optimistic UI only with a pending flag and a rollback.
- [both] Hiding a button is UX, not security: authorization is server-side (`dashboard.md`
  § Critical-write step-up auth).

### 7. Rendering, memory and 24/7 operation
- [both] Render only the visible page; hidden pages are marked dirty and rendered on show.
- [both] Patch interactive panels in place; never rebuild a panel that holds focus or an active press.
- [both] Cache DOM references; write text/classes only on change; batch reads before writes in one
  `requestAnimationFrame`.
- [both] Every timer and listener has a known owner and is created once; buffers are bounded and
  the cap is documented.
- [HMI] Unattended recovery after a station restart (see the kiosk checklist in `dashboard.md`
  § Poll loop reliability). [LAN] Recovery is the user's reload plus session handling (§ LAN profile).

### 8. Robustness and security
- [both] Server-derived text goes through `textContent` or an `esc()` helper only — never
  concatenated into `innerHTML`.
- [both] No swallowed errors; log on state transitions, not on every failed poll.
- [both] User-facing errors are plain operator-language messages; technical detail goes to the console.

### 9. Industrial states and units
- [both] One state vocabulary with one visual treatment everywhere (online, off, on, alarm, manual,
  automatic, defrost, no communication).
- [both] Never rely on color alone: state text or an icon accompanies the color.
- [both] One formatter per unit (°C 1 decimal, psig 0-1 decimal, A 1 decimal, h integer, durations
  `h:mm`); dates/times in the site locale, 24 h, via native `Intl`.

### 10. Accessibility and touch
- [both] Labels on every input; touch targets ≥ 44 px. [HMI] Contrast is checked on the panel, not
  the desktop.
- [both] Error feedback persists until acknowledged (a 3 s toast is too short on a kiosk).

### 11. Quality, tests and versioning
- [both] Pure logic (classification, unit conversion, timing) is unit-tested in Node (DJS1);
  source-structural JUnit tests only for wiring.
- [both] Bump `defaultModuleVersion` on any `rc/` change; the API contract is versioned and the SPA
  tolerates unknown keys.
- [both] No large change goes to a station without review, the verify gate and a preview pass.

## Component layout (`rc/`, classic scripts, no bundler) `[ev: retro dashboard-frontend-standard Δ11]` `[ev: retro dashboard-rc-file-split Δ1]`

The React-style tree (`components/ pages/ services/ hooks/ types/`) mapped onto plain classic
scripts — same separation, no build step. Applies to NEW servlet-SPAs; a deployed single-file SPA is
split only as its own explicitly requested, behavior-neutral work unit.

```
rc/
  index.html                 markup + ordered <script src="...?v=X.Y.Z"> (no inline blocks beyond a boot tag)
  css/  base.css (tokens, reset) · layout.css · components.css
  img/  separate image files, referenced with ?v=
  vendor/ <lib>-<version>.min.js + LICENSE + rc/vendor/THIRD-PARTY.md (§ Vendored libraries)
  js/
    config.js                pollMs, timeouts, base path (one object)
    types.js                 JSDoc @typedef Room, Compressor, Alarm, WriteResult (tsc --checkJs --noEmit)
    slots.js                 slot-key table matching the reader's slot arrays
    services/ api.js (apiFetch, the only fetch) · equipment.service.js · alarms.service.js · setpoint.service.js
    store.js                 poll loop + validate + immutable model + subscribe()   (the "hooks" role)
    watchdog.js              reload / recovery
    components/ header.js · status-badge.js · metric-card.js · equipment-card.js
                alarm-table.js · trend-chart.js · hoa-selector.js · toast.js
    pages/ one file per nav tab
    nav.js                   tab switching
    main.js                  boot only: wire store → visible page
```

- [both] A component is a pure function `(container, slice) → void` that patches the DOM.
- [both] A page composes components and subscribes to the store only while it is visible.
- [both] Services never touch the DOM; only `services/api.js` calls `fetch`.
- [both] `types.js` holds JSDoc typedefs checked with `tsc --checkJs --noEmit` in WSL (no TypeScript build).
- [HMI] Classic scripts plus the DJS1 dual-export shim, not ES `type="module"`; load order is
  explicit in `index.html` (`dashboard.md` § HMI kiosk). [LAN] A LAN-only SPA may use ES modules.
  `[ev: retro dashboard-rc-file-split Δ4]`
- [both] Serving the split needs no Java change in a DashboardDispatch-style router — verify per module
  (`dashboard.md` § ux — servlet + SPA). `[ev: retro dashboard-rc-file-split Δ3]`

## Technology choices `[ev: retro dashboard-frontend-standard Δ12]`

| Situation | Use | Why |
|---|---|---|
| On-station HMI / station web UI (JACE or Supervisor) | Niagara servlet (`BWebServlet`) backend + static SPA in `-ux` `rc/` | The station already owns auth, alarms, histories and points; a JACE JVM is Java 8 Compact3, so Spring Boot cannot run there (`third-party-libraries.md`) |
| Small/medium SPA (today's dashboards) | Plain JS classic scripts + JSDoc types + ESLint (§ Component layout) | No build step; loads on Chromium 83; logic testable with Node |
| Large NEW SPA with many screens or 3D | Vite + TypeScript (optionally React/Preact) built OUTSIDE gradle into a pre-built bundle in `src/rc/`, `build.target` = the panel engine (`chrome83`), bundle budget enforced in preview | Components and typed contracts; served by the pre-built bundle recipe (`dashboard.md` § JS build strategy) |
| Charts on the panel | Hand-rolled SVG (`dashboard.md` § Charts on an HMI) or a small vetted library (§ Vendor catalog) | Large chart libraries add hundreds of KB to a panel that reloads the shell |
| Live data inside the station | REST polling (~5 s) through the servlet | No BOX/Fox subscription in a plain `BWebServlet` |
| Off-station, multi-site or cloud viewer | Separate frontend + its own backend reading oBIX; writes through one gated write service | Different trust boundary; never expose the station to browsers directly |
| Styling | CSS design tokens (§ 1); utility-CSS frameworks only with a framework build | Utility CSS needs a build step |

- [both] Decision rule: a framework does not replace the backend — in a module the backend is the station.
  Pick a framework only when the SPA size justifies a build pipeline, and always target the panel
  engine. [LAN] A LAN-only SPA may target evergreen browsers.

## Vendored libraries `[ev: retro dashboard-frontend-standard Δ14]`

- [both] A browser library lives as separate files in `rc/vendor/<lib>-<version>.min.js` plus its
  LICENSE file, pinned, loaded with `?v=`. Never paste it inline into `index.html`; never fetch it
  from a CDN at runtime (the panel and many site LANs have no internet).
- [both] A deprecated build (e.g. the three.js legacy global `build/three.min.js`, deprecated since
  r150) is recorded with a migration note next to its pin in `rc/vendor/THIRD-PARTY.md`.
- [HMI] Every vendored file passes `toolbelt/lint-vendor-floor.sh <rc-dir>` on every library bump:
  it must parse as an ES2020 classic script (FAIL otherwise — one unparsable file breaks the whole
  page on the panel) and each API above the floor (WARN — it may be feature-guarded) is reviewed.
  Record the verdict (`vendor-floor: clean` or the accepted WARN list with the reason) next to the
  pin in `rc/vendor/THIRD-PARTY.md`. [LAN] A LAN-only build may record the WARNs as accepted.
  `[ev: retro dashboard-frontend-standard Δ16]`

## Vendor catalog `[ev: retro dashboard-frontend-standard Δ15]`

One recommended library per need. Method: files pulled from the npm tarballs, sizes measured
(raw / gzip -9, KB), syntax parsed with acorn, API floor from MDN browser-compat-data. "C83" is a
static scan only — nothing was run on the panel yet, so the first use of each library needs a panel
smoke test. The pinned versions are the [HMI] picks; [LAN]-only modules may use current versions.

| Need | Pick (pinned) | Size raw / gz | C83 | License | Notes |
|---|---|---|---|---|---|
| 3D | three.js **r160** `build/three.min.js` (global) | 670 / 166 | yes | MIT | Last global build (removed in r161). Later path: ESM ≤ r183 with addon imports rewritten to relative paths (no import maps before C89); r184+ needs esbuild `target: chrome83`; r163+ needs WebGL2 (panel GPU support unverified). |
| 3D models | GLB as a separate file in `rc/models/`; meshopt decoder (29 KB) if compression is needed | — | — | — | Base64 costs +33% and blocks caching; the Draco decoder (~1 MB) is too heavy for the panel. |
| Trends | uPlot **1.6.22** IIFE + CSS | 45 / 19 | yes | MIT | 1.6.23+ uses `??=` (fails C83). |
| Charts (desktop) | Chart.js 4.5.1 UMD | 209 / 70 | yes | MIT | ECharts 6.1 = 1,122 / 368, too big for the panel. |
| Gauges | canvas-gauges 2.1.7 | 45 / 14 | yes | MIT | P&ID symbols (valves, pumps): hand-rolled SVG. |
| Plan pan/zoom | svg-pan-zoom 3.6.2 | 30 / 8 | yes | BSD-2 | Native Pointer Events, not Hammer.js (unmaintained). |
| Alarm table | Grid.js 6.2.0 | 53 / 17 | yes | MIT | List.js 2.3.1 (19 / 6.5) for simple filters; Tabulator only on desktop. |
| Sanitization | DOMPurify 3.4.16 | 29 / 11 | yes | MPL-2.0 OR Apache-2.0 | Record the Apache-2.0 election for closed deliverables. |
| Components without build | Preact 10.x UMD + hooks + htm 3.1.1 | ≈ 28 / 12 | yes | MIT / Apache-2.0 | Preact 11 has no UMD; lit 3.x and model-viewer ≥ 4.1 use `??=` (fail C83). |
| Icons | Lucide 1.49.0 sprite, trimmed to the icons used | 517 KB full | — | ISC | Never ship the full sprite. |
| Dates/numbers | native `Intl` (C83 has DateTimeFormat, RelativeTimeFormat, PluralRules, ListFormat, DisplayNames) | 0 | yes | — | dayjs 1.11.23 (7 / 3) only if `Intl` falls short. |

- [HMI] Rejected for the HMI: Babylon.js (8.6 MB raw), ECharts (size), model-viewer ≥ 4.1 and lit 3
  (syntax above C83), petite-vue (unmaintained), zod (unmeasured; hand-roll contract validation).
- [HMI] Budget proposal: the whole `rc/vendor/` ≤ 600 KB gz (the full pick list ≈ 280 KB gz with
  three.js, ≈ 115 KB without).
- [HMI] Not verified: runtime on the real panel, WebGL2 on the panel SoC, feature-guarded call sites
  (`replaceAll` in Tabulator, `toSorted` in Alpine), performance claims.

## LAN profile `[ev: retro dashboard-deployment-profiles Δ3]`

Rules that only the `lan` audience needs (the kit had none before — every earlier frontend rule was
written for the panel):

- [LAN] Responsive at ~360 / 768 / 1280+ px; no hover-only interaction (tablets).
- [LAN] Pause polling on `visibilitychange` while `document.hidden`; resume with an immediate poll.
- [LAN] Handle session expiry: a 401 shows a re-login prompt, never a silent retry loop.
- [LAN] Write audit uses the real Niagara user — per-user attribution is possible only here (the
  panel runs one shared login).
- [LAN] State the load budget per site: open sessions × payload / poll period.
- [LAN] HTTPS with the station certificate; document the self-signed-certificate warning for the
  client.

## Prototype → module `[ev: retro dashboard-frontend-standard Δ13]`

- [both] A standalone design HTML (simulated data) is DECOMPOSED into § Component layout when it
  becomes a module: live data enters as the `services/` + `store.js` layer, never as an appended
  block that reassigns the prototype's global functions. The simulation mode survives only as a mock
  service selected by `config.js`. Complements the "module is the SKELETON" rule in `dashboard.md`
  § Extending an existing dashboard.

## Enforcement
`toolbelt/rc-scan.sh` gates the checkable browser rules (row format and flags in its header):
`browser-floor` (WARN; FAIL under `--strict` or `--profile hmi|both`), `disabled-gate`,
`fetch-no-signal`, `setinterval-async`, `innerhtml-server` (§ 8 escape rule), `datauri-budget`
(FAIL over 20 KB; WARN with `--legacy` for a deployed module not yet restructured), `orphan-page`
(§ 1 layout shell) and `inline-block-size` (DJS1 split, over 300 lines). A false positive is
silenced on its line with `rc-scan: allow <check-id>` plus a reason.
`toolbelt/lint-spa-poll-no-recovery.sh` gates the success-time watchdog (`dashboard.md` § Poll loop
reliability). `[ev: retro dashboard-frontend-standard Δ2]` `[ev: retro dashboard-frontend-standard Δ3]`
`[ev: retro dashboard-frontend-standard Δ5]` `[ev: retro dashboard-rc-file-split Δ6]`

`toolbelt/report-module.sh` passes the module's `ui_profile` (read by the caller from the module's
BUILD-STATE envelope) as `--profile` and `--legacy` through to `rc-scan.sh`, so an `hmi`/`both`
module gets browser-floor FAIL in the aggregated report. `[ev: retro dashboard-frontend-standard Δ10]`

ESLint (§ 2 function size, § 11 quality): `toolbelt/eslint.config.mjs` is the kit flat config —
`ecmaVersion: 2020` + `sourceType: "script"` (the Chromium 83 floor: syntax above it is a parse
error, FAIL), browser globals, `no-unused-vars` (locals; classic scripts share one global scope, so
a top-level symbol used by another file is not flagged), `max-lines-per-function` 60 (WARN),
`no-console` except `console.error`, `eqeqeq`. `report-module.sh` runs it on every `-ux`
artifact's own `src/rc` js (vendor/, ext/, `*.min.js` excluded) and relays its rows (error → FAIL,
warning → WARN). This covers the function-length and dead-local rule; a dead TOP-LEVEL symbol and a
`typeof` guard on an own symbol are not mechanized (review). Prettier is optional.
`[ev: retro dashboard-frontend-standard Δ10]` `[ev: retro dashboard-frontend-standard Δ4]`

Vendored libraries: `toolbelt/lint-vendor-floor.sh` (§ Vendored libraries), also run by
`report-module.sh` on `src/rc/vendor`. `[ev: retro dashboard-frontend-standard Δ16]`

HMI layout fit: `toolbelt/hmi-sweep.js` sweeps every nav view (and declared sub-tabs) at 1280×800 for
document scroll, unnamed inner scrollers and an occluded target such as the alarm banner
(`build-verify.md` § Frontend verify step). `[ev: retro comppan-fase2-amps-alarms Δ3]`

Tools: node + `npm install --prefix toolbelt/eslint` (pinned eslint + acorn); `hmi-sweep.js` needs
puppeteer-core and a Chrome. A missing tool is one SKIP row naming it (exit 4 for the standalone
scripts), never a silent pass. Still DECLARED, not gated: the preview budgets (out-of-repo preview
harness, deferred).
