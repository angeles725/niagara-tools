<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-15 · kit · devguide415-r3-drivers-integration

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 3 traced driver health, the HTTP client gate and RDBMS export from source; this retro carries corpus B1348 (ping/statusFlags), B1346 (httpClient-rt) and B1352 (RDBMS export). It CORRECTS kit `types/logic-authoring.md:646` and REFINES `2026-10-15-devguide415-r2-shipped-libraries.md` (**shl**) Δ7(b) and `2026-10-15-devguide415-r2-drivers-protocols.md` (**drp**) Δ4.
**Delta count**: 6

## What happened
(1) Generic driver-rt never turns a failed poll into a ping fail: `BProxyExt.readFail` only marks the point and `Tuning.readFail()` is empty; the guide's "immediate ping fail" is per-driver (BACnet calls `ping()` on a poll timeout and fails points only after `maxPollTimeouts`). `pingEnabled` sits on `<network>/monitor` and defaults TRUE — the kit tells authors to "enable" it. (2) BACnet `Status_Flags` merge on COV and polled Present_Value proxies alike when the point facet `statusFlags=true`. (3) httpClient-rt has no OAuth2 auth type (only static Bearer); every `HttpClientBuilder` client is a non-driver client refused at send time unless `HttpClientService.enableNonDriverClients` (default false), with only a health fault as symptom; an unmounted client likely NPEs in the audit. `oauth2-rt` (client-credentials, JWT-bearer) is the shipped token library, used by `email-rt`. (4) A module can add RDBMS history exports with the dialect's concrete descriptor under the database's `histories` ext; the licence gate is on the database.

## Evidence
- Ping `[ev: code organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/point/BProxyExt.java:923-930]` `[ev: code organized/bacnet/bacnet-rt/vineflower/javax/baja/bacnet/BBacnetDevice.java:1096-1102]` `[ev: code organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/ping/BPingMonitor.java:128]` vs `[ev: code types/logic-authoring.md:646]` `[ev: corpus B1348]`
- statusFlags `[ev: code organized/docSource/docSource-doc/extracted/bacnet-rt/javax/baja/bacnet/point/BBacnetProxyExt.java:1404-1409]` `[ev: code organized/bacnet/bacnet-rt/vineflower/com/tridium/bacnet/stack/server/CovHandler.java:266-270]` `[ev: corpus B1348 §1348.2]`
- HTTP gate `[ev: code organized/httpClient/httpClient-rt/vineflower/com/tridium/httpClient/BHttpClientService.java:231-252,270]` `[ev: code organized/httpClient/httpClient-rt/vineflower/com/tridium/httpClient/BHttpClient.java:277,304-306]` `[ev: corpus B1346]` vs `[ev: retro 2026-10-15-devguide415-r2-shipped-libraries Δ7]`
- RDBMS `[ev: code organized/driver/driver-rt/vineflower/javax/baja/driver/history/BHistoryExport.java:19-32]` `[ev: code organized/rdb/rdb-rt/vineflower/javax/baja/rdb/BRdbms.java:256-301]` `[ev: corpus B1352]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Custom drivers: `readFail` does not affect device health — on a transport timeout call `device.ping()` (or `pingFail(cause)`) from the poll path and fail points only after N consecutive timeouts (BACnet `maxPollTimeouts` pattern). Lint: partial — grep our comm timeout handler for `ping(`/`pingFail(`. | `types/driver-authoring.md` §5 Health | `[ev: corpus B1348 §1348.1]` |
| Δ2 | CORRECT: `pingEnabled` is ON by default at `<network>/monitor/pingEnabled`; the rule is "never disable it; tune `pingFrequency`/`numRetriesUntilPingFail`". Lint: yes — grep the "enable `pingEnabled`" sentence. | `types/logic-authoring.md:646` | `[ev: corpus B1348 C1]` |
| Δ3 | Refine drp Δ4: `statusFlags=true` applies to COV and polled Present_Value proxies alike; cost = one extra Status_Flags read per poll, or a metadata read when a COV omits flags. Lint: yes (drp Δ4 bog check). | `types/driver-authoring.md:129` | `[ev: corpus B1348 §1348.2]` |
| Δ4 | Refine shl Δ7(b): `HttpClientBuilder` clients are non-driver clients — the station's `HttpClientService.enableNonDriverClients` must be true (default false) or every send is skipped with only a health fault; mount with `mountAtOrd` (an unmounted client likely NPEs in the request audit, `[INFER]` pending B1346-G1). Commissioning item. Lint: partial — grep `HttpClientBuilder` without `mountAtOrd`. | `types/cloud-connector.md` §7 decision ladder | `[ev: corpus B1346 §1346.2]` |
| Δ5 | OAuth2: no auth type ships; implement a `BAuthorizationHeaderScheme` subclass whose `getAuthHeaderValue` fetches/caches a token through `oauth2-rt` `OAuth2ClientCredentialsGrantRequest` (the `email-rt` pattern), plug it in with `withAuthenticator(BTypeSpec)`, depend on `oauth2-rt`. Lint: no. | `types/cloud-connector.md` + `types/security.md` | `[ev: corpus B1346 §1346.3]` |
| Δ6 | RDBMS export from code: add an instance of `histories.getExportDescriptorType()` under the database's `histories` ext (never `new BRdbmsHistoryExport`, it is abstract); the database must be licensed or it goes config-fault at start. Postgres/Supabase stays HTTPS. Lint: partial — grep `new BRdbmsHistoryExport`. | `types/cloud-connector.md` (RDBMS alternative) | `[ev: corpus B1352]` |

## Lessons
- A guide sentence that says "in most drivers" describes driver policy, not framework behaviour; the kit must tell driver authors to implement it.
- "Inherits the gates" is not enough for a gate whose default is closed: name the switch and its default.
- Prefer the shipped library a sibling Tridium module already uses (`oauth2-rt` via `email-rt`) over bundling a new one.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-15-devguide415-r3-drivers-integration.md | kit | 2026-10-15 | pending | 6 |`
