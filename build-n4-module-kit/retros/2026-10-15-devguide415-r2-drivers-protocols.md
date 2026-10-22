<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 10 -->
# 2026-10-15 · kit · devguide415-r2-drivers-protocols

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 read the 4.15.6.20 Mqtt, Bacnet, Modbus, Drivers, Obix and EngNotes user guides plus driver-rt source; this retro carries corpus B1323, B1327, B1336, B1337 and B1343. No earlier 2026-10-15 retro covers drivers or protocols (`rg -i 'mqtt|bacnet|modbus|obix|ping'` over them → 0 relevant).
**Delta count**: 10

## What happened
The kit's driver card has the mechanics but misses the rules that decide field behaviour. MQTT: the stock driver is `abstractMqttDriver` (the kit says "mqtt-driver"), licensed `tridium:mqtt`, capped at 25 devices on a JACE, one scalar per topic. BACnet: `staleTime=0` makes points stale immediately when unsubscribed (the kit says "0 (off)"); `Status_Flags` reach Niagara status only with the `statusFlags` facet; each enabled export priority links an `inN` (contention with our own writes); the `BACnet` station user has no permissions until granted. Modbus: wrong byte order gives 0 or huge values with `{ok}`. Driver framework: `BPingMonitor` never pings a faulted device, alarms only after a 5-minute startup delay, and `pollEnabled` lives on the poll service, not the network. oBIX: the server needs `export=true` + `web`, returns 410/500 when disabled/unlicensed, and external priority writes go through `ObixExport` descriptors. EngNotes add the watch-lease formula and the "503 = peer out of web threads" diagnosis.

## Evidence
- MQTT `[ev: niagara-help 4.15.6.20 guides-clean/Mqtt/MQTTRequirements-74179811.txt:16,23]` `[ev: code organized/abstractMqttDriver/abstractMqttDriver-rt/vineflower/com/tridium/mqttClientDriver/BAbstractMqttDriverNetwork.java:64,69,125]` vs `[ev: code types/third-party-libraries.md:143,171]` `[ev: corpus B1323]`
- BACnet `[ev: niagara-help 4.15.6.20 guides-clean/Bacnet/bacnet-BacnetTuningPolicy.txt:66]` vs `[ev: code types/driver-authoring.md:129]`; `[ev: …/Bacnet/TroubleshootingBACnet-4FF09FBA.txt:86]` `[ev: …/Bacnet/ConfiguringExportedObjectsForWriteB-5410ADDB.txt:15]` `[ev: corpus B1327]`
- Ping/poll `[ev: code niagara-help/versions/4.15.6.20/source/driver-rt/javax/baja/driver/ping/BPingMonitor.java:484-488,526,532]` `[ev: code …/driver/util/BAbstractPollService.java:64]` vs `[ev: code types/driver-authoring.md:213]` `[ev: corpus B1337]`
- oBIX `[ev: niagara-help 4.15.6.20 guides-clean/Obix/LicenseRequirement-OBIXDriver-9F25675C.txt:19]` `[ev: …/Obix/obixDriver-ObixNetwork.txt:61-63]` `[ev: …/Obix/obixDriver-ObixExport.txt:61]` `[ev: corpus B1343]`; leases/503 `[ev: niagara-help 4.15.6.20 guides-clean/EngNotes/aClientPollingTiming.txt:24,66]` `[ev: corpus B1336]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add "MQTT: native first": module `abstractMqttDriver`, licence `tridium:mqtt`, 25 devices on a JACE (50→500 on a Supervisor); one broker connection with many points before many devices; a bundled Paho bypasses licence, cap and User Key Store — record why. Lint: yes — grep `mqtt-driver`. | `types/cloud-connector.md` + `types/third-party-libraries.md:143,171` | `[ev: corpus B1323 §1323.1-2]` |
| Δ2 | MQTT auth matrix (Generic TLS/mTLS, AWS, AWS JITP 8883, Azure SAS, GCP) — set 8883 explicitly; telemetry recipe (publish ext per value, QoS 1, retained, LWT on; publish-on-change drops while disconnected — heartbeat; JSON needs a String publish point fed by a jsonToolkit schema). Lint: partial. | `types/cloud-connector.md` | `[ev: corpus B1323 rows 3-6]` |
| Δ3 | Correct `staleTime`: 0 = stale immediately when unsubscribed; freshness = poll cycle + stale time; `writeOnStart=false` except critical points on large networks. Lint: partial. | `types/driver-authoring.md:127-132` | `[ev: corpus B1327 §1327.2 C1]` `[ev: corpus B1337 §1337.2 C2]` |
| Δ4 | BACnet health: add `statusFlags` to proxy facets or remote alarm/fault/override never reach Niagara status (a point in alarm reads `{ok}`). Lint: yes — bog check. | `types/driver-authoring.md` + `types/dashboard.md` health read | `[ev: corpus B1327 row 3]` |
| Δ5 | Exposing values over BACnet: never enable an export priority our module drives (`inN` contention); grant the `BACnet` station user Operator-write + password for remote writes; enum ordinals 1..n contiguous or the object silently does not export; Px virtual writes default to priority 16; a device program at priority 1-7 beats our `in8`. Lint: partial. | `types/driver-authoring.md` (new §) + `types/logic-authoring.md:641` | `[ev: corpus B1327 rows 4-7]` |
| Δ6 | MS/TP + Modbus: COV per tuning policy (15-min lifetime); `skipUpload` on large MS/TP; APDU timeout × (retries+1) (B1190); Modbus Double/Long64 = 4 registers, byte order on network/device, wrong order → 0/huge with `{ok}`, writes rounded/clamped, 0-based wire addresses. Lint: no. | `types/driver-authoring.md` | `[ev: corpus B1327 rows 8-10]` |
| Δ7 | Ping rules: a disabled/faulted device is never pinged (recovery must come from the read/config path); DOWN is immediate but the ping ALARM waits for `numRetriesUntilPingFail` and a 5-min `startupAlarmDelay`; cadence per attempt. Lint: partial. | `types/driver-authoring.md` §5 after `:182` | `[ev: corpus B1337 rows 1-3]` |
| Δ8 | Fix `pollEnabled` (poll-service property, not network); size rates to ≤75 % busy time; ship Slow/Normal/Fast + no-writeOnStart tuning policies; proxy points have no inputs and should not carry history/alarm exts (put them on the source). Lint: yes for `:213`; partial otherwise. | `types/driver-authoring.md:121-122,205-213` | `[ev: corpus B1337 §1337.2 C1, rows 5-7]` |
| Δ9 | oBIX exposure: licence `export=true` + `web`; 410 = server disabled, 500 = unlicensed; external priority writes via `ObixExport` at a level the module never drives; choose `set` (fallback, persists) vs `override` (timed) deliberately; poll watches well under the 30 s lease and re-`make` after expiry; bind each oBIX user to the scheme its client speaks (B605). Lint: partial. | `types/dashboard.md:23` + `types/logic-authoring.md:554` + `types/security.md` | `[ev: corpus B1343]` |
| Δ10 | Watch/lease clients: lease = max(2×interval, interval + safety factor); raise the interval for weak peers; treat HTTP 503 as "peer out of web threads" (back off); add per-device capability-override children (RPM, segmentation, APDU) as a last resort; expose driver queue depth/overflow as diagnostic points. Lint: partial. | `types/cloud-connector.md` + `types/driver-authoring.md` + `types/observability.md` | `[ev: corpus B1336 rows 3,4,6]` |

## Lessons
- A zero in a tuning field is not always "off" — read the javadoc of each default.
- Status that the field device knows (alarm, fault, overridden) is lost unless the proxy is told to read it.
- Every external write path (BACnet export, oBIX export) links into a priority slot; the module must reserve its own level and document the others.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-15-devguide415-r2-drivers-protocols.md | kit | 2026-10-15 | pending | 10 |`
