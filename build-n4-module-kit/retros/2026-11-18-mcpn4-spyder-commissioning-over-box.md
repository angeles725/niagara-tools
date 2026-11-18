<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-11-18 · kit · mcpn4-spyder-commissioning-over-box

**Session**: mcp-n4 branch feat/mcpn4-rename-spyder-register — BACnet discovery and Honeywell Spyder commissioning on a client B 4.14 station (MS/TP)
**Delta count**: 5

## What happened
Three PUB1012S Spyders had been registered with the default model (PUB6438S) and refused a download. The session
added the tools to see and fix that over BOX (point discovery with the hardware id, `modelChange`), then programmed
the three controllers without Workbench: server-side `transfer` copy of another unit's application, delete, rename,
`modelChange`, `spyderDownload`. Two traps surfaced live: renaming an input point does not rename its BACnet object,
and renaming the object's `BacnetNIFolder` slot breaks the compiler (the download fails "due to compilation error"
with the real message lost); an `objectName` edit does not mark the program modified, so a download alone sends the
old compiled files. The working recipe is `objectName` only, then `compile`, then `spyderDownload`.

## Evidence
- `n4_invoke_action modelChange` verified on three units; download refused before the change `[ev: add622c7]`
- `spyderDownload` read-back live: `Job Success` on two units, `failed` reported as `mismatch` when the compile broke `[ev: 3cd223f9]`
- `compile` action and the "no compile step" hint `[ev: b5a42b0c]`
- Workbench MAC column = `BBacnetAddress.bytesToString` (bacnet-rt :217): IP port in hex `[ev: 6ee5589d]`
- research corpus blocks B1450-B1452 (niagara-research, focus spyder-live-ops) `[ev: corpus B1451]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | A composite Spyder registration step: create the device, read `analogValue:0` with point discovery and set the model before any program work | `go/internal/mcpn4/writeops` (new) + `mcp-n4-kit/README.md` | `[ev: add622c7]` |
| 2 | A typed Spyder program tool: copy an application from another unit, delete points, set `objectName`, compile and download in one plan with one token; never rename a `BacnetNIFolder` slot | `go/internal/mcpn4/writeops` (new) | `[ev: corpus B1451]` |
| 3 | Surface the compiler's real error: read the device's error slots or the station log when a CompilationJob fails with the `JobLog.failed` IllegalArgumentException | `writeops/spyder_download.go` | `[ev: b5a42b0c]` |
| 4 | A tool that loads a Spyder `ControlProgram` (it unloads on its own and later paths fail with the bare path as the error); writes under it should name that cause | `writeops` + `plan.Handle` error text | `[ev: corpus B1451]` |
| 5 | Commissioning report row for Spyders: model vs hardware id, `deviceStatus`, `compileStatusFlag`, object names vs point names (U15 of the 2026-11-16 deep retro) | `n4_commissioning_report` | `[ev: corpus B1450]` |

## Lessons
- A Spyder's configured model is a station slot that defaults to PUB6438S; read `analogValue:0` before programming.
- Set a Spyder object's `objectName`; never rename its `BacnetNIFolder` slot.
- A download sends what was compiled last: compile after any edit the station does not mark as modified.
- A job's read-back must return the job log: "success" and "failed" alone hid the compile step.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-18-mcpn4-spyder-commissioning-over-box.md | kit | 2026-11-18 | pending | 5 |`
