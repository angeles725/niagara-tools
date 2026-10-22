<!-- review-status: pending -->
# 2026-10-15 · kit · devguide415-r2-web-ui-hmi

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 read the shipped `WEB-INF/web.xml` of 14 Tridium modules and the Graphics, JavaWebClients, User (web profile/theme), Touch, WebCharts, SeriesTransform, Mobile and EngNotes (dashboard) guides; this retro carries corpus B1318 (web.xml part), B1328, B1334, B1336, B1339 and B1342. It CORRECTS `2026-10-15-devguide415-horizons-authoring-security.md` (**has**) Δ1 and Δ2.
**Delta count**: 10

## What happened
(1) Of 14 shipped modules with a `WEB-INF/web.xml`, only `web-rt` maps `UserActivityFilter` — by `<servlet-name>file`, never on a polling servlet — and `CsrfProtectedFilter` guards only mutating endpoints in 6 of 14; has Δ1's "map it on the poll servlet" has no precedent (keep `[INFER]` until B1315-G1), and has Δ2's lint "every `<servlet-class>` needs `CsrfProtectedFilter`" would fail 8 of 14 Tridium files. (2) A theme module is found by the registry def `themeName` whose VALUE is the module name, listed only if `nss/*.nss` exists, and loaded as `module://theme<Tag>/…` — the kit calls the value a display name and places the picker under a non-existent Platform menu; the theme is chosen per web profile. (3) The Touch guide's panel recipe: one user per display, Auto Logoff off, Allow Concurrent Sessions off (default true — a second tab evicts the kiosk), client-certificate SSO. (4) The stock webChart `ChartWidget` and query servlets can replace hand-rolled charts. (5) The Mobile profile and Mobile Px are deprecated since 4.6. (6) The vendor DashboardService stores per-user state in `~stations/<name>/dashboards/<user>/db*.json`; a DashboardPane holds one WebWidget.

## Evidence
- web.xml scan `[ev: /mnt/c/Niagara/Niagara-4.15.6.20/modules/web-rt.jar!WEB-INF/web.xml:84-91]` `[ev: code organized/web/web-rt/vineflower/javax/baja/web/filters/UserActivityFilter.java:21]` `[ev: corpus B1318 §1318.2]` vs `[ev: retro 2026-10-15-devguide415-horizons-authoring-security Δ1-Δ2]`
- Themes `[ev: code organized/baja/baja/vineflower/com/tridium/util/CustomThemeModuleManager.java:24,47]` `[ev: code organized/hx/hx-wb/vineflower/javax/baja/hx/BHxProfile.java:291]` vs `[ev: code types/theme.md:41-43]` `[ev: corpus B1328]`
- Touch `[ev: niagara-help 4.15.6.20 guides-clean/Touch/SettingUpADisplayUser-EntsecInstall-4CB4BEC4.txt:43]` `[ev: …/Touch/UpdatingTheIntrusionDisplayUserEnts-5053926B.txt:41]` `[ev: corpus B1334]`
- Charts `[ev: niagara-help 4.15.6.20 guides-clean/WebCharts/AddingDataToAChart-959F4DCC.txt:40,44]` `[ev: /mnt/c/Niagara/Niagara-4.15.6.20/modules/seriesTransform-rt.jar!WEB-INF/web.xml]` `[ev: corpus B1339]`; Mobile `[ev: niagara-help 4.15.6.20 guides-clean/Mobile/Preface-22B940F3.txt:23]` `[ev: corpus B1342]`; dashboards `[ev: niagara-help 4.15.6.20 guides-clean/EngNotes/dashboard-DashboardService.txt:14]` `[ev: corpus B1336]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Correct has Δ1/Δ2: shipped precedent maps `UserActivityFilter` by `<servlet-name>` (web-rt file servlet) — keep "map it on the poll servlet" `[INFER]` until B1315-G1; scope the CSRF lint to servlets accepting POST/PUT (no shipped module maps DELETE). Lint: yes (scoped). | `types/security.md:138-152` + `types/dashboard.md` | `[ev: corpus B1318 §1318.3 C3-C4]` |
| Δ2 | Theme module rules: `themeName` value = module name; module named `theme<Name>`; ship `nss/*.nss` (else not listed), `hx/theme.css`, `ux/theme.css`; a missing theme falls back silently; fix the picker location (Workbench Options / per-profile Theme dropdowns). Lint: yes — `module.xml` def value == moduleName, `theme` prefix, `nss/*.nss` present. | `types/theme.md` §2-§3, §7 | `[ev: corpus B1328 §1328.2 C1]` |
| Δ3 | Web profiles: theme chosen per profile (`selectedHxTheme`, default Zebra) on the user/prototype; Kiosk profiles unsupported in N4; Handheld Hx; profile switches hide sidebars/views. Lint: no. | `types/dashboard.md` web-profile paragraph | `[ev: corpus B1328 rows 3-4]` |
| Δ4 | Px rules: choose target media deliberately (Ux for controllers); responsive panes for wall panels (absolute layout is only the CanvasPane default); relative ORDs, `$(var)` via `PxInclude`, NEQL bindings; `module://` icons; Hx `visible` refreshes every 5 s; `WebWarmupConfig`. Lint: yes — grep shipped `.px` for `ord="station:|slot:/`. | `types/wb-widgets.md:360` PX section | `[ev: corpus B1328 rows 6-8]` |
| Δ5 | Hx headers: X-Frame `Sameorigin` and the default CSP sources are required by Hx views; embed via CSP `frame-ancestors`, not X-Frame `Any`. Lint: partial. | `types/security.md` headers | `[ev: corpus B1328 row 9]` |
| Δ6 | Kiosk panel user (extends has Δ1): Auto Logoff Enabled = false AND `allowConcurrentSessions` = false (default true); read+invoke-only role; `navFile` landing page; stripped profile; dimming not 0; TLS + configuration password. Complementary to `UserActivityFilter`. Lint: partial — commissioning check of the user slots. | `types/dashboard.md` § HMI kiosk | `[ev: corpus B1334 §1334.1]` |
| Δ7 | Prefer client-certificate SSO for unattended panels (re-login on reboot); assigning the cert restarts Fox/Web services. Lint: no. | `types/security.md` | `[ev: corpus B1334 row 3]` |
| Δ8 | Charts: first choice for Px/Hx viewers is the stock `ChartWidget`; a custom SPA may call `/webChart/query/data/<ord>` (user's ORD read applies); series-transform graphs for server-side rollups (timestamp + non-String value, no live mode); points folder caps at 10 histories; `defaultOptions.chart` for shared styling; keep SVG on the wall panel. Lint: partial. | `types/dashboard.md` § Charts + `types/frontend-standard.md:175` | `[ev: corpus B1339]` |
| Δ9 | Never build on the Mobile Web Profile / Mobile Px (deprecated since 4.6); phone use = responsive ux view under the `lan` profile; BOX server cache is a station tuning item. Lint: partial — grep `BMobile*` imports. | `types/frontend-standard.md:22` + `types/dashboard.md` | `[ev: corpus B1342]` |
| Δ10 | Coexisting with the Tridium DashboardService: per-user state in `~stations/<name>/dashboards/<user>/db*.json`; one HTML5 WebWidget per DashboardPane; do not change the per-user pane Id. Lint: partial. | `types/dashboard.md` | `[ev: corpus B1336 rows 1-2]` |

## Lessons
- A proposed lint must be run against the shipped Tridium files first; one that fails 8 of 14 vendor files is mis-scoped.
- Registry defs carry module names, not labels; read the consumer code before writing what a def "means".
- Kiosk robustness is mostly user configuration (logoff, concurrent sessions, SSO), not module code.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-15-devguide415-r2-web-ui-hmi.md | kit | 2026-10-15 | pending | 10 |`
