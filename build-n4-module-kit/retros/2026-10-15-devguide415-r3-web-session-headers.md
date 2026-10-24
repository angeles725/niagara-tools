<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-15 · kit · devguide415-r3-web-session-headers

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 3 closed the web-side static child gaps from source (4.14 decompile + `javap` of the 4.15.6.20 jars); this retro carries corpus B1344 (session lifetime), B1349 (headers/themes on a module servlet) and B1347 (oBIX hidden slots). It CORRECTS kit `types/security.md:366-377` and EXTENDS `2026-10-15-devguide415-r2-web-ui-hmi.md` (**wuh**) Δ1/Δ6 and `2026-10-15-devguide415-r2-drivers-protocols.md` Δ9.
**Delta count**: 6

## What happened
(1) The web idle timeout is Niagara's own timer on the user's super session: Auto Logoff Enabled = false makes the period −1 and no web or Fox session ever idles out; only `POST /timeout` (the `activityMonitor`, CSRF-protected) and `UserActivityFilter` (mapped only on web-rt's `file` servlet) reset it — a REST poll to our servlet never does. 4.15 adds an absolute logoff counted from login (7 days; per-user flag default TRUE; template stations enable the global switch) that activity cannot reset. (2) Jetty runs `TridiumSecurityFilter` on every `BWebServlet` context, so our servlets already carry the station's CSP, X-Frame-Options and nosniff headers; the kit's "DashboardPan omits them" gap and its prescribed `setHeader` CSP are wrong — our CSP would REPLACE the station policy. Theme CSS is never injected into a servlet SPA. (3) oBIX drops HIDDEN slots from child listings only; a hidden slot is still served by path to a user who can read it.

## Evidence
- Idle timer `[ev: code organized/baja/baja/vineflower/com/tridium/session/NiagaraSuperSession.java:184-198,217-244]` `[ev: code organized/web/web-rt/vineflower/com/tridium/web/servlets/SessionTimeoutServlet.java:25-38]` `[ev: code organized/web/web-rt/vineflower/WEB-INF/web.xml:59-68]` `[ev: corpus B1344 §1344.1-1344.2]`
- Absolute logoff `[ev: javap /mnt/c/Niagara/Niagara-4.15.6.20/modules/baja.jar!com/tridium/session/NiagaraSuperSession scheduleAbsoluteTimeout]` `[ev: niagara-help 4.15.6.20 guides-clean/StationSecurity/StationAutoLogoff-4CBC23A4.txt:15-23 "As of Niagara 4.15, the Absolute Logoff"]` `[ev: corpus B1344 §1344.3]`
- Headers `[ev: code organized/jetty/jetty-rt/vineflower/com/tridium/jetty/BJettyWebServer.java:1178-1180,1462-1468]` `[ev: code organized/web/web-rt/vineflower/com/tridium/web/filters/TridiumSecurityFilter.java:22-27]` vs `[ev: code types/security.md:366-377]` `[ev: corpus B1349 C1]`
- Themes `[ev: code organized/bajaux/bajaux-rt/vineflower/com/tridium/ux/WbWebWidgetServlet.java:143-149]` `[ev: corpus B1349 §1349.2]`; oBIX `[ev: code organized/obixDriver/obixDriver-rt/vineflower/javax/baja/obix/io/ObixEncoder.java:234-236,588-589]` `[ev: corpus B1347]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Session lifetime rule: "a poll is not activity"; ladder for a 24/7 HMI — (1) panel user Auto Logoff Enabled = false, (2) SPA keep-alive = `POST /timeout` (offset 0, CSRF token) or `activityMonitor.keepAlive()`, (3) `UserActivityFilter` only for a war-style module. Supersedes wuh Δ1's open `[INFER]` for the static part. Lint: partial — a module with a poll loop must reference `/timeout` or `keepAlive`, or document route (1). | `types/dashboard.md` § Session lifetime (new) | `[ev: corpus B1344 §1344.4]` |
| Δ2 | 4.15 absolute logoff: per-user `autoLogoffSettings.absoluteLogoffEnabled` defaults true; template stations enable `UserService.absoluteLogoffEnabled` (7 days); a kiosk user needs BOTH Auto Logoff and Absolute Logoff off or accepts a weekly re-login. Extends wuh Δ6. Lint: partial — commissioning check reads both user slots. | `types/security.md` + commissioning checklist | `[ev: corpus B1344 §1344.3]` |
| Δ3 | CORRECT the header section: station-hosted servlets inherit the WebService Http Header Providers via `TridiumSecurityFilter` (CSP `default-src 'self' workbench`, X-Frame Sameorigin, nosniff by default); remove "DashboardPan omits X-Frame/nosniff/CSP — gaps ODA2-G1/G2" and the `setHeader` snippet; set a header only to deviate deliberately (it replaces the station value). Lint: yes — downgrade `lint-servlet.sh api-response-headers` from WARN to info; add a commissioning `curl -I` check. | `types/security.md:360-380` | `[ev: corpus B1349 C1]` |
| Δ4 | A servlet SPA is not themed automatically: link `/module/theme<Name>/ux/theme.css` (and the core `web/rc/theme/theme.css`) explicitly, or render through a bajaux view. Lint: partial — grep the SPA HTML for `theme.css`. | `types/theme.md` + `types/frontend-standard.md` | `[ev: corpus B1349 §1349.2]` |
| Δ5 | Embedding: never set X-Frame `Any`; add CSP `frame-ancestors` on the station's CSP provider (operator setting), not in module code. Extends wuh Δ5. Lint: no. | `types/security.md` headers | `[ev: corpus B1349 §1349.1]` |
| Δ6 | oBIX: HIDDEN removes a slot from listings only; it is still served by path to any user who can read it — use categories/permissions as the boundary for the read user. Extends r2 drivers-protocols Δ9. Lint: no. | `types/security.md` (oBIX exposure) | `[ev: corpus B1347 §1347.1]` |

## Lessons
- A 4.14 decompile can miss a 4.15 security feature (absolute logoff): `javap` the real install jar before calling a session rule complete.
- Before prescribing response headers, check the container's filter chain; a module header can silently weaken the station policy.
- Slot flags are presentation, not access control — say so wherever a kit card hides slots from an external protocol.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 `types/dashboard.md:158`, Δ2 `types/dashboard.md:168` + `types/security.md:263`, Δ3 `types/security.md:434`, Δ4 `types/theme.md:171`, Δ5 `types/security.md:445`, Δ6 `types/security.md:163`. INDEX row generated from the header: `| 2026-10-15-devguide415-r3-web-session-headers.md | kit | 2026-10-15 | folded | 6 |`
