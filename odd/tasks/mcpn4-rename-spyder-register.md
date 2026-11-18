# mcp-n4: rename components and register Honeywell BACnet Spyders

Objective: give mcp-n4 the tools to rename components and to register new Spyders (duplicate factory instance 9999), then use them live on client B station.
Branch: feat/mcpn4-rename-spyder-register (worktree niagara-tools-worktrees/mcpn4-device-tools, from origin/main d9b04787). Delivery: ask-on-risk. Test runner: `go test ./...` in go/.

## Specs

- S1 (L1): "Renombrar 30 puntos (Numeric Writable40001 → Power y 40005 → Setpoint en los 15 Cubys). Mcp-n4 no tiene herramienta para renombrar" → mcp-n4 gains a rename tool; then the 30 points are renamed live (15 Cubys incl. the cashier unit: the 15 Modbus devices).
- S2 (L1): "intenta agregando 3 spyders, para mejorar mcp-n4, porque ahora mismo tenemos en linea 3 spyders, y quiero ver si mcp-n4 puede registrarlos, y si no puede, que se implemente esa herramienta." → mcp-n4 can register the three new Spyders (MAC 12, 13, 14 on MS/TP network 2, all at device:9999).
- S3 (L2): "el 12, 13, y 14 debes nombrarlos como 2012, 2013, 2014" → components named 2012/2013/2014 (stored $32012..), instances device:2012..2014.
- S4 (L1): field review: "solo Cuby01 responde ... Hay que repetir el Full Download de la Spyder 2006. Los sensores con +inf ... los CT leen ~0.01 y DUCT_P está por revisar." → live diagnosis report (field fixes are physical/Workbench).
- S5 (L3): "trabaja con ODD+RDD(GRANTED)+ SI GENTLE-AI PIDE REVISION, dale permiso y trabaja con ENGRAM".

## Tasks

| ID | Specs | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S1 | inline | done (code): n4_rename_component (RenameOp r, batch ≤50, inverse, read-back, rollback read-back), registry, README | rename_test.go, rename_internal_test.go GREEN; go test ./... GREEN |
| T2 | S2, S3 | inline | done (code): create accepts escaped display names (2012 → $32012); invoke allows writeDeviceInstance only on honeywellBacnetSpyder:BacnetSpyder | spyder_register_test.go GREEN; differentials declare both differences |
| T3 | S1 | inline (live) | done: 30/30 renamed verified; links (154) and history ids intact; saved persisted | batches e7effcd9 (pilot), eed733b1 (29); save d19210be |
| T4 | S2, S3 | inline (live) | done: 2012/2013/2014 created, address 2:0c/0d/0e, objectId device:2012..2014, writeDeviceInstance sent, health Ok on all three; saved persisted | create 371629c9, 667980f0, 53351e56; wDI 058c2ae2, ecae1745, 5aaa5e9d; save 854394c6 |
| T5 | S4 | inline (live) | done (read): see Log L4 | BQL 2026-11-17 |
| T6 | S5 | inline | pending: commit, RDD assess, retro | |

## Log

- L1 (2026-11-17, user, verbatim): "Esta el tunnel levantado, y no se si recuerdes que ibamos a hacer?? lo de los spyders, y cubys, que bamos a probar la parte de los cubys y los spuder. Intenta trabajar con esto: Pendiente 1. Renombrar 30 puntos (Numeric Writable40001 → Power y 40005 → Setpoint en los 15 Cubys). Mcp-n4 no tiene herramienta para renombrar, así que quedó para hacerlo en Workbench. No sé si llegaste a hacerlo. 2. Campo: solo Cuby01 responde, y su RoomTemp/RoomRH en 0.0 es sospechoso. El Cajero y los Cuby03 a 15 dan pingFail. Hay que repetir el Full Download de la Spyder 2006. Los sensores con +inf están en la 2009 (SA_T) y la 2010 (RA_T y SA_T), los CT leen ~0.01 y DUCT_P está por revisar. y tambien intenta agregando 3 spyders, para mejorar mcp-n4, porque ahora mismo tenemos en linea 3 spyders, y quiero ver si mcp-n4 puede registrarlos, y si no puede, que se implemente esa herramienta." Screenshot: Discover shows device:9999 at MAC 12, 13, 14 (net 2, Honeywell Spyder, orange = duplicate).
- L2 (user): "el 12, 13, y 14 debes nombrarlos como 2012, 2013, 2014"; "TIenes permitidos escribir, y hacer todo lo que tu quieras"; "puedes tratar de hacerlo y cuando lo logres intentar utilizarlo".
- L3 (user): "trabaja con ODD+RDD(GRANTED)+ SI GENTLE-AI PIDE REVISION, dale permiso y trabaja con ENGRAM".
- L4 (live read 2026-11-17, client B station 4.14.0.162 tier A): Modbus: the cashier unit and Cuby01 up, Cuby03..15 down; 30 writables still `Numeric Writable40001/40005`; RoomTemp/RoomRH 0.0 {ok} on both live units (suspicious). Spyders 2004..2010 {ok}; +inf on 2009 SA_T, 2010 RA_T/SA_T; CTs ~0.01 (2004 FAN_A 1.91); DUCT_P 10.20 (2006), 10.24 (2009). Licence tridium:globalcapacity device.limit 26, now 22 devices (+3 = 25).
- L5: map (explorer): RenameOp `{nm:r,h:parent,o,n}` (javax.baja.sync.RenameOp), dynamic slots only; Spyder `writeDeviceInstance` (BBacnetSpyder.doWriteDeviceInstance) reads the instance via wildcard 0x3FFFFF and writes objectId to the component's address; create at the final instance (never 9999: I-Am rebinding, B1165). Launcher write-scope widened to /Drivers/BacnetNetwork (backup the client B launcher (backup kept)).
- L6: Risk: item 3 is not hit (tools/actions added, no existing contract removed; create accepts more names: refusal of escaped display names is a changed behaviour, declared in the differentials). Field write (writeDeviceInstance) is gated to the Spyder type.
- L7 (2026-11-17 live): rename is now live-certified over BOX (RenameOp r): pilot verified, then a 29-item batch verified; history names use %parent.parent.parent.name%_Power (fixed text) so ids did not change. Spyders: a new BacnetSpyder gets the next free instance as objectId (2011 for the first, then 2013, 2014), never 9999. Address set through its struct children (addressType 3, networkNumber 2, macAddress octet string). Workbench Device Manager showed device:-1 for the new components until the operator refreshed the view (stale client proxy; station value was right). 2010 went down at 1:03 PM (TransactionTimeout) during the save, before any Spyder write; still down at 1:08 PM: field check.
- L8 kit deltas found: (a) set_slot read-back compares a BacnetOctetString case-sensitively: requested "0C", station stores "0c" -> verdict mismatch for an equal value (batch 176b9aba); compare octet strings case-insensitively or normalise to lowercase. (b) writeDeviceInstance read-back is unverified; a read of the device health after a ping (or getOnlineDeviceInstanceNumber) would verify it. (c) a register-Spyder tool could compose create + address + writeDeviceInstance + ping in one plan.
