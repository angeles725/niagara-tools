# mcp-n4: BACnet device discover, plus the 2026-11-17 retro deltas

Objective: mcp-n4 runs a BACnet device discover (Who-Is) on a station and reports what it finds, and closes the retro deltas of the Spyder registration session.
Branch: feat/mcpn4-rename-spyder-register (worktree niagara-tools-worktrees/mcpn4-device-tools, stacked on 0bcecbc4, base origin/main d9b04787). Delivery: ask-on-risk. Test runner: `go test ./...` in go/ (memcap shim).

## Specs

- S1 (L1): "necesito que me digas cuantos SPYDERS puedes ver, intenta hacer discover" → a tool that runs the station's BACnet device discovery and lists found devices (instance, network, MAC, vendor/model, already in the database or not, duplicate instance), without adding devices to the station database.
- S2 (L1): "vamos a empezar a trabajar mejorando el mcp-n4" → the retro deltas of 2026-11-17 (the 2026-11-17 rename and Spyder registration retro in niagara-research, client B), in priority order:
  - R1 (HIGH): `n4_set_slot` read-back of a `bacnet:BacnetOctetString` is case-sensitive: requested "0C", station stores "0c", verdict `mismatch` for an equal value.
  - R2 (MEDIUM): `writeDeviceInstance` read-back via `getOnlineDeviceInstanceNumber` instead of `unverified`.
  - R3 (MEDIUM): composite `n4_register_bacnet_spyder` (create + address + objectId check + writeDeviceInstance + ping, one token).
  - R4-R6 research (Spyder app copy, compile/download actions, points discover), R7 doc (stale Workbench views), R8 close retro U1/U2.
- S3 (L3): "y tambien trabaja con el mcp para la parte de cargarle codigo a los spyder, pero primero es el discover" → mcp-n4 loads a Spyder application without Workbench (retro R4 copy the logic from a programmed Spyder; R5 compile / validate / spyderDownload actions; R6 points discover). Order: after S1.

## Tasks

| ID | Specs | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S1 | explore (delegated, read-only) | done: map (L5) | explorer handoff 2026-11-18 |
| T2 | S1 | inline | probe done (L6); next: RED tests + tool n4_discover_bacnet_devices (plan + token, global by default) | |
| T3 | S2 R1 | inline | done: octet strings compared by octets (hex case and spaces ignored), single and batch set_slot | RED: 3 of 6 cases mismatch ("0C" vs "0c"); GREEN; go test ./internal/mcpn4/... ok |
| T4 | S2 R2 | inline | todo | |
| T5 | S2 R3 | inline | todo | |
| T7 | S3 | explore → probe → inline | todo after T2: map Spyder app copy + download actions, single-device PoC on 2012 (no program yet) |
| T6 | S2 R7 R8 | inline | todo: docs | |

## Log

- L1 (2026-11-18, user, verbatim): "Vamos a retomarlo de nuevo, necesito que me digas cuantos SPYDERS puedes ver, intenta hacer discover, y tambien vamos a empezar a trabajar mejorando el mcp-n4, dale vamos a ello"
- L2 (2026-11-18 live, client B station 4.14.0.162 tier A): 10 BACnet Spyders in the database: 2004-2009, 2012-2014 health Ok; 2010 down (TransactionTimeoutException, same as 2026-11-17). n4_invoke_action cannot run submitDeviceManagerJob (typed argument): discover is the gap.
- L3 (2026-11-18, user, verbatim): "Pero dime cuantos mas spyder puedes ver que esten en linea, o sea que tu realizaces como un discover" — "y tambien trabaja con el mcp para la parte de cargarle codigo a los spyder, pero primero es el discover". Priority: S1 first, then S3; R1 paused.
- L4 (2026-11-18, user, verbatim): "recuerda que tiene que ser como un discover en todas las redes" — "en networks, send global". networks "all;" = BDiscoveryNetworks all=true → the job sends Who-Is to BBacnetAddress.GLOBAL_BROADCAST_ADDRESS (BBacnetDiscoverDevicesJob.java:72-80): the Workbench "Send Global". Default of the tool.
- L5 (explorer map): action submitDeviceManagerJob (BBacnetNetwork.java:359-374, flags 4 = admin invoke) with bacnet:DeviceDiscoveryConfig {deviceLowLimit 0, deviceHighLimit 4194302, networks "all;", waitResponseTime 10 s}; submits a TRANSIENT BacnetDiscoverDevicesJob<N> under Services/JobService; results are dynamic DiscoveryDevice children of the job; no device is added to the database; any I-Am can still rebind an existing device's address when its instance matches (B1165 §1165.4).
- L6 (live probe 2026-11-18 13:00, temporary go/cmd/discover-probe, not committed): BOX invokeAction {h, a:"submitDeviceManagerJob", b:{nm:p, t:bacnet:DeviceDiscoveryConfig, s:[4 children baja:Integer / bacnet:DiscoveryNetworks]}} accepted; the probe printed the reply as {} only because encoding/json cannot marshal *pyjson.Object (probe bug); the 2026-11-16 lab proof shows the reply is {"t":"baja:Ord","v":"slot:/Services/JobService/BacnetDiscoverDevicesJob<N>"}. Job reached success in ~10-15 s. Children are honeywellDeviceManager:HonDiscoveryDeviceEx (Honeywell module extends DiscoveryDevice); its "d" text carries name, id, addr, vNm, mNm, fwR, asR. Two jobs ran (Job1 13:00:25, Job2 13:00:37; one was not the probe's) and both found the same 9 devices: device:2004-2009 and 2012-2014, all Honeywell Spyder on network 2; 2010 (2:0a) did not answer; no unknown or duplicate instance (no 9999). 2012-2014 report an empty device name (no application loaded).
- L7: prior design and lab proofs already cover this: niagara-research `odd/tasks/discovery-propose-apply.md` T2 plans `n4_discover_devices` + `n4_discover_points` (BACnet) and `odd/tasks/driver-discovery-remote-design-4.15.md` section G #3/#5 + "Live proof against the simulator" (wire forms, row fields, `--` rows, readLog, timings). Implement T2 of that feature here, with its tool names, instead of a new name; point discovery (`submitPointDiscoveryJob` on `<device>/points`, rows `dc0..dcN`) also serves S3/R6.
- L8: tool name `n4_discover_devices` (the planned name from discovery-propose-apply T2; BACnet only, refuses another network type). It is a token-gated write (field traffic, I-Am rebind risk), default Send Global, wait 10 s (1..60), read-back waits wait+30 s, rows marked registered / new / address_differs plus not_answering. Also fixed a README line misplaced by 1893da34 (the invoke refusal sentence had landed under the rename bullet).
