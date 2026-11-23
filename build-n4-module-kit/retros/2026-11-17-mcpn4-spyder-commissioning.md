<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-11-17 · kit · mcpn4-spyder-commissioning

**Session**: a client B field job with six Honeywell BACnet Spyders, where a Spyder Full Download failed
without a word in the station (43 min lost) and every Spyder's IOComp configuration was checked by hand.
**Delta count**: 1

## What happened
A Spyder whose Full Download had not completed still looked healthy in the device list; only its
`deviceStatus` and `compileStatusFlag` showed it. The first download of each Spyder failed and left it
disabled until `Enabled=true` and a Ping. The IO configuration (sensor type, ranges, terminal) of each
IOComp was then read one by one over oBIX.

## Evidence
- Live BQL on the field station: good Spyders read `deviceStatus=1`, `compileStatusFlag=0`,
  `sErrorStatus` empty; one whose download did not complete read `deviceStatus=4`, `compileStatusFlag=4`.
- `honeywellSpyderTool` declared types: `IOComponent` (BIOComponent) carries the dynamic `SensorType`,
  `InputLow`/`InputHigh`/`OutputLow`/`OutputHigh` and the frozen `Terminal1`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. `n4_commissioning_report` section `spyder`: the download state of each BACnet Spyder (disabled, device_status, compile_status, error_status) and, with `spyder_spec`, `io_mismatch` against the caller's IO table; field runbook "Spyder Full Download" | `mcp-n4-kit/README.md` (Finishing a job), `mcp-n4-kit/METHODOLOGY.md` § 3, `build-n4-module-kit/BUILD-LOOP.md` § 6.c | `[ev: retro mcpn4-spyder-commissioning Δ1]` |
