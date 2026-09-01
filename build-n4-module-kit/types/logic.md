# Type: pure logic (rt control) — SEED (feed as built)

A control module: `BComponent`s with real control logic, no UI. Exemplar to READ before building: ColdRoomPan-rt (`BColdRoom`, `BEvaporatorUnit`, `BDefrostController`).

Not yet fully documented — seed pointers, feed via the retro step when you build one:

- **Control engine:** logic runs on `execute()` / `changed()` / `atSteadyState()` on the engine thread — guard so it never throws there. Study the Control Engine in corpus (`corpus-nav show 6`) + kitControl (`BTstat` hysteresis: `sp ± diff/2`, HOLD between = deadband). `differential` is a band in degrees.
- **Links, not polling:** inputs arrive via Niagara links; read input `BStatus` and fail safe on invalid/null.
- **Slots:** computed outputs = `TRANSIENT|SUMMARY|READONLY`; config = `SUMMARY|OPERATOR`. Same facet/BDouble rules as checklist-common.
- **No servlet, no `-ux`** unless paired with a dashboard (then also read type-dashboard.md).

Verify with checklist-common.md + build-verify.md. TODO: flesh out the control-engine cycle, staging, safety fail-modes from a real build.
