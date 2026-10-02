<!-- review-status: pending -->
# 2026-10-01 · DashboardPan · dashboard-rc-file-split

**Session**: PANCCADIA HMI freeze triage (Cliente/panccadia-leon, read-only); user question on HTML/CSS/JS distribution of the servlet-SPA
**Delta count**: 6

## What happened
While triaging an HMI freeze after a station restart (watchdog fix in `index.html` §15b did not fire on
restart #11), the user asked whether keeping almost all of the dashboard's visual layer in one HTML file is
wrong. Measurement: `DashboardPan-ux/src/rc/index.html` is 3,179 lines with 3 inline `<style>` blocks
(~600 lines CSS) and 2 inline `<script>` blocks (~2,350 lines JS); `rc/` holds no other file. The kit already
names this the anti-pattern (DJS1, JS only) but does not say how to lay the files out, does not cover CSS,
cache invalidation after a `-ux` deploy, the test retargeting, or how this interacts with the existing-module
no-restructure rule. A small reliability change (fetch timeout + second watchdog) would land as a few lines
inside a 3,179-line diff surface with no executable JS test. No code was changed; DashboardPan stays as is
until the user explicitly requests the split (decision-logic-decomposition Δ3).

## Evidence
- Size/blocks: `wc -l` = 3179; 3 `<style>`, 2 `<script>`; awk block-line count CSS 601 / JS 2357; `find src/rc -type f` = `index.html` only `[ev: Cliente/panccadia-leon 42fa82e]`
- Static serving already supports split assets: `DashboardDispatch.java:210-211` (`.css` → `text/css`, `.js` → `application/javascript`) `[ev: Cliente/panccadia-leon 42fa82e]`
- XHR guard is `/api/*`-only, so `<script src>` / `<link href>` GETs (no `X-Requested-With`) are NOT redirected: `DashboardDispatch.java:29-30` (guard order javadoc) and the `/api/` prefix check in `route()` `[ev: Cliente/panccadia-leon 42fa82e]`
- Source-structural tests read `rc/index.html` as text: `ConfigLoginGateTest.java:17`, `DashboardIndexHtmlPhaseTest` `[ev: Cliente/panccadia-leon 42fa82e]`
- Existing kit rule: `types/dashboard.md:33` DJS1 (split inline JS + dual-export shim + Node harness; DashboardPan named as the anti-pattern) `[ev: corpus B762]`
- Existing-module rule: `retros/2026-09-28-decision-logic-decomposition.md` Δ3 (no restructure of working modules unless explicitly requested) `[ev: odd/tasks/panccadia-persistent-config-hoa.md T2b]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Extend DJS1 to a full `rc/` layout for NEW servlet-SPAs: `index.html` = markup only; `css/` split by layer (`base.css` tokens/reset, `layout.css`, `components.css`); `js/` split by responsibility (`api.js` reads/writes, `watchdog.js` reload/recovery, `render.js` painting, one file per feature page e.g. `rooms.js`/`condensers.js`/`config-login.js`, `nav.js`, `main.js` boot only). No inline `<style>`/`<script>` beyond a boot tag. | `types/dashboard.md` § `ux — servlet + SPA` (DJS1 bullet) | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ2 | Cache invalidation: every `<link>`/`<script src>` in `index.html` carries `?v=<module version>` matching `defaultModuleVersion` in the group `build.gradle.kts`, bumped on every `-ux` deploy, so a kiosk panel does not keep stale JS after a `-ux`-only deploy (no station restart, browser keeps its cache). [INFER: browser caching of servlet static responses not measured on the panel; relate to `lint-servlet.sh` cache-nofinger.] | `types/dashboard.md` § `Deploy on a JACE` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ3 | State that the split needs NO Java change in a DashboardDispatch-style router: static fallback already serves `.css`/`.js` and the XHR guard is `/api/*`-only; the split ships as a `-ux` static-asset deploy. Verify both facts per module before splitting (a router whose XHR guard covers all GETs would 302 every `<script src>`). | `types/dashboard.md` § `ux — servlet + SPA` (after DUX1) | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ4 | Script format for the HMI kiosk browser (Chromium 83 on WEB-HMI10/CF): use classic scripts + the DJS1 dual-export shim (not ES `type="module"`) so the same files load in the panel AND in the Node logic harness without a bundler; load order is explicit in `index.html`. | `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` | `[ev: corpus B762]` |
| Δ5 | A split of an existing SPA is its own behavior-neutral work unit: same commit retargets every source-structural test that reads `rc/index.html` as text to the new files; prove neutrality with `DashboardPan-ux/tools/hmi-sweep.js` (or the module's equivalent) before/after plus `node --check` per file; no feature change rides in the same commit. | `METHODOLOGY.md` § `Schema / upgrade safety` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ6 | Advisory lint (WARN-only): `rc-scan.sh` flags an inline `<script>` or `<style>` block over ~300 lines in any `rc/**/*.html` ("inline-block: split per DJS1"). Existing deployed modules (DashboardPan) are exempt from restructuring per decision-logic-decomposition Δ3 unless the user explicitly requests the split; NEW dashboard modules start split (scaffold-module.sh `-ux` skeleton emits the Δ1 layout). | `toolbelt/rc-scan.sh` header + `METHODOLOGY.md` § `Conformance rules — lintable vs advisory` | `[ev: Cliente/panccadia-leon 42fa82e]` |

## Lessons
- A one-file SPA is not a bug, but past a few hundred lines it hides small reliability changes inside a huge diff and blocks JS unit tests.
- Split CSS as well as JS; DJS1 alone only addressed JS.
- Check the router's static serving and XHR-guard scope before splitting — that decides whether the split is `-ux`-only.
- Version every asset URL, or a kiosk keeps stale code after a no-restart `-ux` deploy.
- For deployed modules the split is a separate, behavior-neutral, explicitly requested work unit — never bundled with a fix.
