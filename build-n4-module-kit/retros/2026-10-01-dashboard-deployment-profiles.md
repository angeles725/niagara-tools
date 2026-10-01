<!-- review-status: pending -->
# 2026-10-01 · kit · dashboard-deployment-profiles

**Session**: PANCCADIA HMI freeze triage → frontend standard; user clarified that one servlet dashboard serves two different audiences
**Delta count**: 7

## What happened
The user pointed out that a dashboard module is used in two distinct ways: (A) the Honeywell 10" WEB-HMI panel
(Chrome 83 WebView, 1280×800 touch, one kiosk, 24/7, nobody reloads it) and (B) a general module that anyone
on the site LAN opens from their own browser at `https://<JACE or Supervisor IP>/<servlet>/` (current
Chrome/Edge/Firefox/Safari, desktop/laptop/tablet/phone, several concurrent users, each with their own
Niagara login, used intermittently). The kit has no such distinction: grep of `types/dashboard.md`,
`METHODOLOGY.md` and today's retros finds only one line about dashboards "also opened from desktop" and one
lesson stating "the panel browser is the target". Today's frontend retros are therefore HMI-biased: some rules
are panel-only (Chrome 83 floor, fixed 1280×800, unattended recovery), some are LAN-only concerns the kit never
names (responsive layout, hidden-tab polling, concurrent-session load, per-user audit, session expiry). The same
triage also surfaced three cross-profile improvements adapted to our current stack: panel-faithful testing, CI on
client repos, and published module versions (deployed-version drift was found today only from a Workbench
screenshot). The user asked whether NiagaraMods Reflow already does version publishing: per our corpus it
embeds and shows its own version and migrates config between versions with a pre-migration backup, but no
block documents it reporting installed versions to an external system.

## Evidence
- No profile distinction in the kit: `grep -niE "desktop browser|laptop|tablet|responsive|document.hidden|visibilitychange|concurrent" types/dashboard.md METHODOLOGY.md retros/2026-10-01-*.md` → only `2026-10-01-dashboard-frontend-standard.md:69` and `2026-10-01-dashboard-frontend-reliability-rules.md:45` `[ev: kit 3793ff2]`
- Local test browser is Chrome for Testing 153.0.8010.12 (`~/.cache/ms-playwright/chromium-1243`), 70 major versions above the panel; every Chrome 83 defect so far was found on the panel itself (`f12f0fe` inset/flex gap; Engram topic `panccadia/dashboard/hmi-constraints-proven-css` min()/max()/backdrop-filter, user-confirmed 2026-09-28) `[ev: f12f0fe]`
- Client repo without CI: `panccadia-leon` has 34 `*Test.java` under `srcTest/`, 172 commits, no `.github/`; `niagara-tools/.github/workflows/ci.yml` already runs Java 8 + pinned JUnit 4.13.2 + bats `[ev: Cliente/panccadia-leon 42fa82e]`
- Deployed version known only by screenshot: JACE Software Manager shows DashboardPan 2.8.0 while 2.8.1 exists (`f12f0fe`) `[ev: f12f0fe]`
- Release folders: 13 `Downloads/PANCCADIA-modulos-*` folders with suffixes `-v2`…`-v7` / `-comppan-2.6.2`; each recent one carries `SHA256SUMS.txt` + `SOURCE.txt` (good) but the name does not state module versions `[ev: Downloads/PANCCADIA-modulos-2026-09-30]`
- Reflow reference: SPA embeds `{ version:"1.7.7", rc:"RC5", number:"75" }` matching `module.xml` vendorVersion `1.7.7.75` (corpus B151, B153); config schema migration with pre-migration backup and `Client-Migration` handshake (B233); config backups (B231). Grep of B138-B155 and B228-B241 found no reporting of installed versions to an external system `[ev: corpus B153 §153, B233 §233.4]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Every `-ux` dashboard declares `ui_profile: hmi \| lan \| both` in its BUILD-STATE row and module docs. `both` with ONE build ⇒ the HMI floor governs everything (Chrome 83, panel CSS rules); alternative: two entry points `hmi.html` (panel) and `index.html` (LAN) sharing `js/` with a per-entry `config.js`, each linted against its own floor. | `BUILD-STATE.md` § `How to read this file` + `types/dashboard.md` § `ux — servlet + SPA` | `[ev: kit 3793ff2]` |
| Δ2 | Tag every rule of the three 2026-10-01 frontend retros with its profile (Appendix A) when folding them into `types/frontend-standard.md`; a rule without a tag is not folded. | `types/frontend-standard.md` (new, per frontend-standard Δ1) | `[ev: kit 3793ff2]` |
| Δ3 | LAN profile rules: responsive at ~360 / 768 / 1280+ px; no hover-only interaction (tablets); pause polling on `visibilitychange` when `document.hidden`; handle session expiry (401 → re-login prompt, no silent loop); write audit uses the real Niagara user (per-user attribution is possible only here); load budget = open sessions × payload / poll period, stated per site; HTTPS with the station certificate (self-signed warning documented for the client). | `types/dashboard.md` (new §) `LAN browser access` | `[ev: kit 3793ff2]` |
| Δ4 | Test matrix per profile: HMI → a pinned Chromium 83 binary in the preview harness at 1280×800 [INFER: obtaining a Chromium 83 build runnable in WSL is not yet verified] plus the existing panel smoke; LAN → current Chrome/Edge/Firefox/Safari at three widths. Remote DevTools on the WEB-HMI10 is a candidate to investigate, not assumed. | `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` + `tools/dashboard-preview.py` | `[ev: f12f0fe]` |
| Δ5 | Client repos get CI by reusing the `niagara-tools` workflow pattern (Java 8 Temurin, pinned JUnit/hamcrest sha256): run pure JUnit tests, `rc-scan.sh`, the frontend lints and `report-module.sh` on every push; private repo, no secrets needed. | `build-verify.md` (new §) `Client repository CI` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ6 | Version publication: each module exposes a read-only version (`defaultModuleVersion` + build millis) as a SUMMARY/READONLY slot on its service and via `/api/version`; the SPA shows it in an About/footer (Reflow pattern, B153); the existing oBIX → Supabase pipeline reads it per site and flags drift against the latest packaged release (`SOURCE.txt`). | `types/structure.md` (new §) `Deployed version publication` + `types/dashboard.md` § `Deploy on a JACE` | `[ev: corpus B153 §153]` |
| Δ7 | Release packages: folder named by site and module versions (e.g. `PANCCADIA_ColdRoomPan-2.4.2_CompPan-2.8.0_DashboardPan-2.9.2`), a `MANIFEST.md` (module versions, manual version, change summary, `SHA256SUMS.txt`, `SOURCE.txt`), and a retention rule (keep the last 3 locally; older ones archived in the client repo's releases or deleted). | `types/distribution.md` (new §) `Release package` | `[ev: Downloads/PANCCADIA-modulos-2026-09-30]` |

## Lessons
- One servlet, two audiences: declare the profile before writing a rule or a line of CSS.
- If one build serves the panel, the panel's browser is the floor for everyone.
- LAN access multiplies load and enables real per-user audit; the panel needs unattended recovery.
- A version nobody can read remotely is drift waiting to happen.
- Test in the browser the user actually has, not the newest one on the dev box.

## Appendix A — profile tags for today's frontend rules
S = `2026-10-01-dashboard-rc-file-split.md`, R = `2026-10-01-dashboard-frontend-reliability-rules.md`, F = `2026-10-01-dashboard-frontend-standard.md`.

| Rule | Tag | Note |
|---|---|---|
| S Δ1 file layout, S Δ2 `?v=`, S Δ3 no-Java split, S Δ5 split as own work unit, S Δ6 inline-block lint | both | — |
| S Δ4 classic scripts + dual-export shim | HMI (and `both`) | LAN-only may use ES modules |
| R Δ1 fetch timeout, R Δ2 no overlapping polls, R Δ6 interaction safety, R Δ7 no disabled gate button, R Δ8 one timing config | both | — |
| R Δ3 success watchdog | HMI critical; LAN recommended | — |
| R Δ4 recovery ladder | HMI | LAN: user reload + session handling (Δ3) |
| R Δ5 Chrome 83 feature floor | HMI (and `both`) | LAN-only: evergreen floor |
| R Δ9 stale-data visibility | both (new modules; per-project opt-out) | — |
| F Δ1-Δ8, Δ11, Δ13, Δ14 standard, lints, contracts, budgets, layering, prototype rule, vendoring | both | budgets stricter for HMI |
| F Δ9 layout shell | both | top nav on HMI; LAN may add side nav + responsive |
| F Δ10 ESLint | both | `ecmaVersion` per profile floor |
| F Δ12 technology table | both | LAN-only SPAs may target evergreen browsers |
| F Δ15 vendor catalog, F Δ16 vendor floor gate | HMI picks (e.g. uPlot 1.6.22) | LAN-only may use current versions |
