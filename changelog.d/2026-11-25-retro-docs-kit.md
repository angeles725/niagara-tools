### Changed

- `types/dashboard.md` (retro client-e-dashboard-live-check, issue #986): § Write path: a timing claim is measured
  through an independent observer, the points snapshot (Δ4); § Critical-write step-up auth: the documented call is
  named `CsrfUtil.verifyCsrfToken(req)` in place of `CSRF_TOKEN_HTTP_HEADER` (Δ21), and `WebService.sameSite` (Lax) is
  the baseline layer (Δ20); § Deploy on a JACE: `BWebServlet` answers 410/403 before `service()` (Δ19) and a write
  with the user's Context needs `operatorInvoke` on every written point (Δ33); § Real alarms: open alarms query and
  BajaScript `AlarmSpace.ackAlarms` (Δ24); § Charts on an HMI: new "History queries" subsection (Δ23).
- `types/dashboard.md` § Write path (retro R10, issue #989): schedule ranges are written as explicit ON and OFF ranges
  with a null default output, never OFF inferred from "outside the ranges" (P5).
- `types/driver-authoring.md` (issue #986): § 11.1 a Fallback write reaches BACnet at priority 16 (Δ14); § 4 a pulse
  source's tuning policy needs Min Write Time 0 (Δ15).
- `types/logic-authoring.md` (issue #986): § Authoring a driver: a pulse source keeps `minActiveTime` and
  `minInactiveTime` at 0 (Δ5) and `in1`/`in8` are READONLY link targets (Δ6); § Watchdogs and timers: reuse kitControl
  before hand-rolling (Δ26).
