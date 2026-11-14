# mcp-n4 Modbus writes
Objective: let mcp-n4 build Modbus RTU devices and points end to end (addresses, data types, scaling, ping, copy), then use it to wire the client-C IR gateways into the dashboard.
Branch: feat/mcpn4-modbus-writes (stacked on feat/mcpn4-set-slot-types 6bf99311) · Delivery: ask-on-risk · Runner: `go test ./...` in go/.

## Specs
- S1 (from retro 2026-11-12 Δ1): `n4_set_slot` accepts `baja:Integer`, `modbusCore:FlexAddress` (display form `Modbus:40005`), enums (`baja:DynamicEnum`/`baja:FrozenEnum`, by tag or ordinal) and `baja:Facets`, and reads the value back.
  - Evidence for the design (decompiled Niagara 4.14 source): `modbusCore:FlexAddress` is a BStruct with `addressFormat` (FrozenEnum `AddressFormatEnum`, default `hex`) and `address` (String, default "0") (`modbusCore-rt/.../datatypes/BFlexAddress.java:15-27`); `ModbusClientNumericProxyExt` has `regType` (default holding) and `dataType` (default integerType) enums (`.../client/point/BModbusClientNumericProxyExt.java:33-39`); the proxy `conversion` slot is a `BProxyConversion` simple, and `driver:LinearConversion` encodes as `"<scale>;<offset>"` (`driver-rt/.../point/conv/BLinearConversion.java:117-123`). So S1 reduces to: enums by tag, `baja:Integer`, and a generic "encoded simple of type T" write (covers Facets and LinearConversion).
- S2 (Δ2): `n4_invoke_action` allows `ping` (no argument) on a driver network, device or point device ext; dry run + confirmation token as for every write.
- S3 (Δ5): a tool copies one configured component N times under the same parent with new names, as one batch with rollback.
- S4: with S1-S3, create the IR gateway devices, their 7 points each with addresses, data type and scaling, and the links to the dashboard minisplits, on the live JACE.
- User choice (2026-11-12, verbatim): "Mejorar mcp-n4 primero (Recomendado)"; then "y despues de eso ve con lo demas".

## Tasks
| ID | Specs | Route | Status | Commit |
|---|---|---|---|---|
| T1 | S1 | inline (explorer map first) | done: RED 7 fail -> GREEN; 20 suites, vet, gofmt clean | 5901d57f |
| T2 | S2 | inline | done: RED 2 fail -> GREEN | 5901d57f |
| T3 | S3 | inline | done: RED (undefined) -> GREEN 5 tests | 41e9b35e |
| T3b | S4 prerequisite | inline | done: create_component proxy_ext (read-only Modbus points); RED 2 fail -> GREEN | 006d00cf |
| T4 | S4 | inline, live + 2 link writers | done live (L9) | 2a714404 |

## Log
- L1 (2026-11-12, user, verbatim): "Necesito que me ayudes con algo del dashbaord de client B puedes entrar y conectar todo, o checar que todo este bien conectado, tambien no se si puedas utilizar modbus para agregar los puntos del cuby que se tendrian que tener, no se si me explique o sea que con mcp puedas ver si se puede hacer eso"
- L2 (2026-11-12): live read-only audit. Unit1-7 linked (49 links), Unit8 and MS01-MS26 unlinked; Modbus network COM2 9600 8N1 ok; gateway "Cajero" (address 2) down, pingFail since 2026-11-11 23:17. mcp-n4 0.9.5 cannot write Integer, FlexAddress, enums or facets, nor invoke ping.
- L3 (2026-11-12, user, verbatim): "Mejorar mcp-n4 primero (Recomendado)" · "y despues de eso ve con lo demas".
- L4 (2026-11-12): T1-T3b committed (5901d57f, 41e9b35e, 006d00cf). RDD assess on 6bf99311..HEAD: medium, review_due false, under_budget (slice stays open). Design: baja:Simple sends the encoded string under the slot type or simple_type; struct slots refused (set children); readback checks the type on a type change. Copy depth 6 (Modbus point address is level 5); runtime slots status/health/faultCause/out/readValue/writeValue and links not copied. Launcher ~/.config/mcp-n4/<client-b>-run.sh repointed to this worktree binary, Cajero/points scope widened to /Drivers/ModbusAsyncNetwork (backup in session scratchpad). Python oracle server is not touched (declared differences in differential_test.go).
- L5 (2026-11-12, user answer, verbatim): "Dirección = número MS (Recomendado)": Cuby address N feeds MSnn (address 2 Cajero -> MS02), MS16-MS26 stay unlinked for now. Live check: the running server still has the old scope (set_slot on Cajero refused "outside the write scope"), so /mcp reconnect is pending.
- L6 (2026-11-12, user answer, verbatim): "Respaldo nulo (Recomendado)": writable Cuby points (Power, Mode, FanSpeed, Setpoint) get a null fallback in the template and every copy, so Niagara writes nothing until an operator commands.
- L7 (2026-11-12, user, verbatim): "O mejor ya aplica ese delta La copia falla porque el nombre Numeric$20Writable40001 tiene un espacio escapado que la validación de mcp-n4 rechaza" · "Porque si es necesario para lo que tienes que hacer, hazlo ahora mismo". Done: plan.ExistingName accepts station escapes ($ + two hex digits) for an existing source in n4_copy_component and n4_remove_component; new names stay plain; refusal text unchanged (Python oracle parity). RED 2 fail -> GREEN; all suites, vet, gofmt clean.
- L8 (2026-11-12): live dry run of copying Cajero showed name-only children (upload, download actions; regType at its default) sent without a type, and the transient read-only absoluteAddress (flags 3, modbusCore BModbusClientProxyExt.java:29). Fix: the copy leaves out name-only children and absoluteAddress. RED 1 fail -> GREEN; all suites green. Also observed: passing a confirmation_token without dry_run=false returns a new dry run instead of an error (delta for the retro).
- L9 (2026-11-12): S4 live done. Cajero template finished (Mode 40002, FanSpeed 40003, modbus format, null fallbacks); n4_copy_component made Cuby01, Cuby03..Cuby15 (verified), deviceAddress 1, 3..15 set (verified); 105 links Cuby -> MS01..MS15 (7 slots each, power/occupancy as station-made conversion links), BQL count 105; RoomTemp NumericInterval history (5 min, %parent.parent.parent.name%_%parent.name%) on all 15; station saved. A station restart mid-wiring (module 2.0.1 install) kept the unsaved work: the station saved on stop. Field state: only Cuby01 answers (RoomTemp/RoomRH read 0.0, to check); Cajero and Cuby03..15 pingFail.
