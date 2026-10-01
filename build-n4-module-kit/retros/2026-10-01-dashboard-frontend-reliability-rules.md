<!-- review-status: pending -->
# 2026-10-01 · kit · dashboard-frontend-reliability-rules

**Session**: PANCCADIA HMI freeze triage (Cliente/panccadia-leon, read-only); user asked for frontend rules judged by the agent
**Delta count**: 9

## What happened
The PANCCADIA HMI froze after station restart #11 although the in-page watchdog (DashboardPan 2.4.3+, running
2.8.0 on the JACE) had recovered the previous 10 restarts. Reading the SPA showed the watchdog counts
consecutive FAILED polls, but `readJson()` has no fetch timeout and `poll` is an async function driven by a
fixed `setInterval`, so a hung connection neither fails nor stops new polls from piling up behind it. A
Workbench `servletName` toggle meant to force the watchdog did not recover the panel (user test). Two earlier
DashboardPan fixes belong to the same frontend-discipline gap: CSS features the panel's Chrome 83 lacks
(`inset`, flex `gap`) and a poll prefill overwriting a focused input. The kit has WebView render quirks
(aspect-ratio/object-fit, power-cycle after redeploy) and `rc-scan.sh` (ord/host literals, bare catch,
null branch) but no rules for network timeouts, poll scheduling, recovery design, the browser feature floor,
or interaction safety. These deltas are kit rules for NEW dashboard modules; changes to deployed DashboardPan
remain separate, explicitly authorized work.

## Evidence
- `readJson()` fetch without timeout/signal: `DashboardPan-ux/src/rc/index.html:2553` `[ev: Cliente/panccadia-leon 42fa82e]`
- Async `poll()` (`index.html:2811`) scheduled by `setInterval(poll, N4.pollMs)` (`index.html:2871`), `pollMs: 5000` (`index.html:774`) `[ev: Cliente/panccadia-leon 42fa82e]`
- Watchdog keyed on consecutive failures + probe accepting any `r.ok`: `index.html:2777-2833` (`WATCHDOG_THRESHOLD = 6`, comment "~30 s a 5 s/poll") `[ev: Cliente/panccadia-leon 42fa82e]`
- Watchdog present in the deployed build: `f0ddfdc` (2.4.3) is an ancestor of the 2.8.0 bump `2456d41`; JACE Software Manager shows DashboardPan-rt/-ux 2.8.0 `[ev: f0ddfdc]`
- Chrome 83 CSS defect: `inset:0` and flex `gap` replaced in the login modal; disabled "Guardar cambios" fired no click so the login gate never opened `[ev: f12f0fe]`
- Focused input overwritten by poll prefill `[ev: 74e84b7]`
- Prior WebView quirks: `retros/2026-09-02-dashboardpan-detail-render-doors.md` items 1, 8 `[ev: retro dashboardpan-detail-render-doors]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Every network call has a timeout: one helper `fetchT(url, opts, ms)` (AbortController + `setTimeout`, e.g. 8 s for reads; with Δ2 self-scheduling the timeout bounds each poll cycle), used for reads, writes and watchdog probes; a timeout counts as a failure. `rc-scan.sh` WARN: `fetch(` without `signal` outside the helper. | `types/dashboard.md` § `ux — servlet + SPA` + `toolbelt/rc-scan.sh` header | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ2 | No overlapping polls: schedule the next poll with `setTimeout` after the current one settles (success, failure or timeout) plus an `inFlight` guard; never `setInterval(asyncFn)`. [INFER: hung requests also occupy the browser's per-host HTTP/1.1 connection slots, starving later polls and the watchdog probe.] `rc-scan.sh` WARN: `setInterval(<name>` where `<name>` is declared `async function`. | `types/dashboard.md` § `ux — servlet + SPA` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ3 | Recovery watchdogs key on time since last SUCCESS (`lastOkAt`), not on a count of failures; the probe uses `fetchT` and validates an app marker (e.g. the data API answers JSON), not any 200; after a hard ceiling (e.g. 10 min without success) reload unconditionally. | `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ4 | Document a kiosk recovery ladder per HMI module, each layer with its blind spot: (1) in-page timeout + success watchdog; (2) operator "Refresh" button; (3) station-side liveness (`hmiStatus` Healthy/Stale/Down from a page heartbeat) + Workbench "reload HMI" action; (4) physical (panel cable/power). In-page layers cannot recover a page replaced by a non-app page (login/error); only (3)-(4) observe or cover that. | `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ5 | Browser feature floor = the panel engine (Chrome 83 on WEB-HMI10/CF): `rc-scan.sh` WARN on CSS `inset:`, `gap:` in flex rules, `aspect-ratio`, and JS `??=` `\|\|=` `&&=` `.replaceAll(` `.at(` `structuredClone` top-level `await`; verify layout in the HMI simulator before deploy. [INFER: per-feature Chrome versions not re-verified here beyond `inset`/flex `gap` in f12f0fe and `aspect-ratio` in the 2026-09-02 retro.] | `toolbelt/rc-scan.sh` header + `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` | `[ev: f12f0fe]` |
| Δ6 | Interaction safety: background work (poll prefill, auto-reload, re-render) never overwrites a focused input or interrupts an open write session; automatic reloads defer while `document.activeElement` is an input or a write session is active, with a hard ceiling. | `types/dashboard.md` § `Config panel UX on a fixed touch panel` | `[ev: 74e84b7]` |
| Δ7 | Never `disabled` a control whose click is the entry point to an auth/write gate (a disabled button fires no click); mark it (class/`aria-disabled`) and route the click to the gate. | `types/dashboard.md` § `Critical-write step-up auth` | `[ev: f12f0fe]` |
| Δ8 | One timing config object (`pollMs`, `fetchTimeoutMs`, watchdog windows); thresholds expressed in time and derived from it (`Math.ceil(30000 / pollMs)`), never a constant whose comment assumes another constant's value. | `types/dashboard.md` § `ux — servlet + SPA` | `[ev: Cliente/panccadia-leon 42fa82e]` |
| Δ9 | Stale-data visibility for NEW modules: after N× `pollMs` without a successful read, the UI marks values as stale with the last-good timestamp. A project may opt out by explicit user decision recorded in the module docs (PANCCADIA opted out 2026-10-01; its existing `procStale` dimming stays). | `types/dashboard.md` § `HMI kiosk (e.g. WEB-HMI10/CF, 1280×800 capacitive Chromium — see corpus B724)` | `[ev: Cliente/panccadia-leon 42fa82e]` |

## Lessons
- A watchdog that counts failures is blind to requests that never finish; watch for missing success instead.
- Every fetch needs a timeout, and polls must not overlap.
- The panel browser is the target, not the desktop browser: lint the feature floor.
- Background refreshes must never fight the operator's input.
- Recovery is a ladder; each in-page layer dies with the page, so a station-side view is needed.
