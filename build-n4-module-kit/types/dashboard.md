# Type: dashboard (facade + servlet + SPA) — MATURE

Proven end-to-end on DashboardPan (2026-08). A browser dashboard for an HMI, served by the module itself. NOT PX — a `BWebServlet` + a static ES5 SPA. Exemplar to copy: chihuahua-ux; cleaner reference: DashboardPan.

## rt — the facade (no logic)
- One `BComponent` per equip/entity (`BRoomPanel`-style): **link-in display slots** (`BStatusNumeric`, `SUMMARY`) the integrator links from control points; **writable config slots** (`SUMMARY|OPERATOR`). No control logic.
- A `BAbstractService` root (`getServiceTypes()`) holding the entity components as frozen children — the ORD the servlet walks. Must sit at a fixed path (e.g. `station:|slot:/Services/DashboardService`).

## ux — servlet + SPA
- `BWebServlet` subclass; `getServletName()` = the mount prefix → `/<prefix>/`. Registered as a `<type>` in module-include.xml + a palette instance under a Servlets folder.
- `doGet/doPost` delegate to a **pure router** (WSL-unit-testable, no Niagara deps). Guards in order: path-traversal→404; **XHR guard** on `/api/*` (missing `X-Requested-With: XMLHttpRequest` → 302); unknown `/api/*`→404; static fallback (`/`→index.html).
- **Reader**: walk the facade → flat JSON keyed by slot-path, each value `{"v":num|bool|null,"st":"<status>"}`, fault-aware, `Locale.ROOT` doubles, escaped strings.
- **RBAC**: `checkCanWrite` first line of every write: `BPermissions.has(OPERATOR_WRITE)` (bit, not role name), fail-closed, `SlotPath.unescape` the username. Audit each mutation.
- Static assets in `src/rc/`; gradle copies `src/rc→rc/`; served via `getClassLoader().getResourceAsStream("rc/"+path)`.
- Frontend: ES5 + `fetch`, REST poll. **Every fetch (reads too) sends `X-Requested-With`** or the guard 302s it → page loads but data shows "--". Use ABSOLUTE `/<prefix>/...` URLs (relative break without a trailing slash). Write via `POST /api/setpoint {ord,value}`.

## HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)
- **No page scroll** at the panel resolution. Chrome (header+nav+footer) ≈180px → page content ≤ ~620px. If an entity has many fields, use a **per-entity selector** (one at a time) + a 2-column grid, not all side-by-side.
- **Touch:** targets ≥44px (52px for primary); edit via **+/− steppers**, not type-a-number (no keyboard); toggle switch for booleans; one "Guardar cambios" per entity. Neutralize hover-only states.
- Kiosk meta: `user-scalable=no`, `touch-action:manipulation`, `overscroll-behavior:none`.
- Verify every page fits at exactly 1280×800 (headless Chrome or dashboard-preview.py `/hmi`).

## Real alarms (Phase B — see corpus B6/B8, chihuahua B166)
- A `BAlarmSourceExt` can only host on a **control point** (`BNumericPoint`), not a plain `BComponent` → add child points fed from the display slots; attach `BAlarmSourceExt`(`offnormalAlgorithm=BOutOfRangeAlgorithm`, a fault algorithm). Limits configured from the dashboard = writable config slots (per-role-per-room) linked/fed into the algorithm.
- Read via `GET /api/alarms` = BQL over `station:|alarm:|bql:select * where sourceState='offnormal' or 'fault' order by timestamp desc`. Station needs a `BAlarmService` + `BAlarmClass`.
- ACK from a plain servlet is the hard part (chihuahua only got it via BajaScript) — spike `svc.getAlarmDb().getDbConnection().getRecord(uuid)` → `svc.ackAlarm(live)` before committing.
