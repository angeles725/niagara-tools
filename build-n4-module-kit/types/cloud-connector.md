# Cloud / IoT connector — `BCloudConnectionService`, transport, auth, channel SPI

How to author a cloud/IoT connector module in the cloudLink family. This doc covers the
four extension seams: the central service, the transport layer, the authenticator, and
the per-data-type channel. Protocol-specific message formats and backend telemetry schemas
are out of scope.

Choosing between MQTT, OPC UA, oBIX, BACnet/IP export and a custom servlet for getting data out of a
station: `types/driver-authoring.md` § 10 (egress decision). `[ev: retro t9c-reference-gaps Δ7]`

Building a JSON payload: check the stock jsonToolkit schema before hand-rolling one (`types/json-toolkit.md` § 1;
it has no transport, so § 9 MQTT or § 7 HTTP carries its `output`). `[ev: retro devguide415-r2-shipped-libraries Δ9]`

---

## 1 · Core service — `BCloudConnectionService`

```java
public final class BCloudConnectionService
        extends BAbstractService
        implements BIRestrictedComponent, LicenseLimit { ... }
```

The service holds three **static SPI factory maps** that backend modules register into
via the Baja type registry at service start:

| Static field | Type | Key |
|---|---|---|
| `msgFactoryMap` | `Map<String, BAbstractCloudLinkHandlerFactory>` | `platform + ';' + transportType` |
| `configFactoryMap` | `Map<String, BAbstractChannelConfigFactory>` | `platform` string |
| `fileUploadManagerMap` | `Map<String, BAbstractFileUploadManager>` | `platform` string |

Registration happens lazily at first lookup (config factories also in `descendantsStarted()`) via `Sys.getRegistry().getConcreteTypes(Factory.TYPE.getTypeInfo())` — the registry scans every loaded module for concrete subtypes
and populates the map. [ev: code cloudLink-rt BCloudConnectionService.registerHandlerFactories/registerConfigFactories/registerFileUploadManagers]

Three `@NiagaraProperty` folders hang off the service as `flags=256` (hidden) children:
`authenticators` (`BClientAuthenticatorsFolder`), `transports` (`BTransportsFolder`),
`channels` (`BClientChannelsFolder`). [ev: code cloudLink-rt BCloudConnectionService @NiagaraProperties]

**Gotcha:** `BIRestrictedComponent` forces placement inside a `ServiceContainer` (the
Services component). Do NOT add the service as a free-standing component. [ev: code
BCloudConnectionService.checkParentForRestrictedComponent]

---

## 2 · Transport layer — dual model

`BAbstractTransport extends BAbstractService` is the common base; it owns the shared
store-and-forward queue, GZIP compression, retry logic, and send-metrics.

| Slot | Default | Effect |
|---|---|---|
| `pendingMessageLimit` | 50 | back-pressure ceiling (in-flight messages) |
| `messageRetries` | 2 | per-message retry count (max 10) |
| `compression` | `none` | `BCompressionMode`: `none` / `gzip` |
| `messageThrottlingLimit` | 0 (off) | messages per second rate cap |
| `defaultMessageTimeout` | 60 s | per-message send timeout |

[ev: code cloudLink-rt BAbstractTransport @NiagaraProperties]

### 2.1 · HTTP transport — `BHttpTransport`

```java
public class BHttpTransport extends BAbstractConnectionlessTransport { ... }
```

Backed by **okhttp3**, which the NRE ships in `bin/ext` (`okhttp-4.12.0.jar` + okio + kotlin-stdlib in
4.15.6.20; `net-rt.jar` holds no okhttp class). Do NOT bundle okhttp3 yourself: compile against the shipped
copy with `nre("com.squareup.okhttp3:okhttp")` (the `nre` configuration = jars in `!bin/ext`, devguide
`build.txt:186`). `[ev: corpus B1320 §1320.3 C1-C2]` `[ev: retro devguide415-r2-shipped-libraries Δ2]`

**MQTT: native first.** The stock driver is the `abstractMqttDriver` module, a licensed
feature, capped by default at 25 devices on a JACE and 50 on a Supervisor (guide
`Mqtt/MQTTRequirements-74179811.txt:11,17`). Model one broker connection with many points before many devices. A
bundled Paho client bypasses the licence, the cap and the User Key Store; record why when you choose it.
`[ev: corpus B1323 §1323.1-2]` `[ev: retro devguide415-r2-drivers-protocols Δ1]`

### 2.2 · AMQP transport — `BAmqpTransport`

```java
public class BAmqpTransport extends BAbstractConnectedTransport
        implements IAmqpCallbacks { ... }
```

Backed by **Apache Qpid Proton** (+ Microsoft Azure AMQP stack). Because no platform
module ships Qpid Proton, cloudLink-rt fat-jars it directly — the `cloudLink-rt.jar`
contains the full `org.apache.qpid.*` and `com.microsoft.azure.*` class trees.
[ev: code cloudLink-rt BAmqpTransport imports org.apache.qpid.proton.*; META-INF/maven
lists `org.apache.qpid` + `com.microsoft.azure`; DEPENDENCIES lists "Proton-J"]

`connectionType` (`BAmqpConnectionType`: `amqp` / `amqpWs`) selects plain AMQP or
WebSocket tunneling. `trustAnchors` (`BVector`, `security=true`) pins remote CA
certificates for mutual TLS. [ev: code cloudLink-rt BAmqpTransport @NiagaraProperties]

**See `types/module-wiring.md`** (uberjar vs api-dep rules) and
**`types/third-party-libraries.md`** for the fat-jar/provided decision.

---

## 3 · Auth SPI — `BAbstractClientAuthenticator`

```java
public abstract class BAbstractClientAuthenticator extends BAbstractService {
    public abstract void addMessageHandlerProperties(
            BAbstractTransport transport, String messageType, Map<String, Object> props);
    public abstract Map<String, Object> getPlatformProperties();
    public abstract boolean canAuthenticate();
    public abstract boolean isRegistered();
}
```

[ev: code cloudLink-rt BAbstractClientAuthenticator]

**Security pattern — credentials live in the platform KeyRing, not in component
properties.** Only the Azure SAS authenticator stores its token parameters in the platform KeyRing
(`KeyRingPermission("*")` in cloudLinkAzure-rt); cloudLinkForge-rt declares `KeyRingPermission("cloudLinkNcs")`
and `("cloudLinkHonSbp")` for those backends, which declare none. The other authenticators keep a `BPassword`
property or a key-store alias.

| Concrete class | Backend | Credential store |
|---|---|---|
| `BPasswordAuthenticator` | core | `usernameAndPassword` property (BPassword) |
| `BTokenAuthenticator` | core | `tokenProvider` child |
| `BAzureSasAuthenticator` | cloudLinkAzure | `KeyRingPermission("*")` |
| `BFederatedIdentityAuthenticator` | cloudLinkForge | key store (X.509) |
| `BRpkAuthenticator` | cloudLinkForge | key store (cert alias + password) |
| `BRpkAuthenticatorHonSbp` | cloudLinkHonSbp | as Rpk (extends `BRpkAuthenticator`) |

[ev: code cloudLink-rt module.xml types list; cloudLinkAzure module.xml
`KeyRingPermission("*")`; cloudLinkForge module.xml `KeyRingPermission("cloudLinkNcs")` +
`KeyRingPermission("cloudLinkHonSbp")` (:114-115)]

**Gotcha:** `BUsernameAndPassword` is a helper struct — it is NOT `BAbstractClientAuthenticator`.
The actual authenticator calls `canAuthenticate()` / `isRegistered()` before any message
send; a false return from either will gate outbound traffic. [INFER: method contract
derived from abstract signature + call site in BHeartbeatChannel:112 (isRegistered)]

---

## 4 · Channel SPI — `BAbstractClientChannel`

```java
public abstract class BAbstractClientChannel
        extends BAbstractService
        implements BIRestrictedComponent {
    @NiagaraProperty String channelType;       // logical name, READ_ONLY
    @NiagaraProperty BChannelConfig channelConfig;  // backend-supplied config
}
```

[ev: code cloudLink-rt BAbstractClientChannel @NiagaraProperties]

Named channel types registered in `BClientChannelsFolder`:

| Channel | Class | Purpose |
|---|---|---|
| Alarms | `BAlarmsChannel` | alarm record upload |
| Backup | `BBackupChannel` | station backup files |
| Commands | `BCommandsChannel` | cloud→station commands |
| Events | `BEventsChannel` | audit / event stream |
| Heartbeat | `BHeartbeatChannel` | liveness ping |
| Histories | `BHistoriesChannel` | trend log export |
| Messaging | `BMessagingChannel` | generic messages |
| Model | `BModelChannel` | component-tree model publish |
| Points | `BPointsChannel` | COV / poll data |
| Schedule | `BScheduleChannel` | schedule sync |

[ev: code cloudLink-rt module.xml type list]

Each backend supplies its own `B<Backend><ChannelType>Config` and a
`BAbstractChannelConfigFactory` (e.g. `BForgeChannelConfigFactory`,
`BNcsChannelConfigFactory`) which is auto-discovered via the registry map (§1).
[ev: code cloudLinkForge module.xml BForgeAmqpAlarmChannelConfig … BForgeChannelConfigFactory;
cloudLinkNcs module.xml BNcsChannelConfigFactory]

---

## 5 · Backend plugin chain and packaging

```
cloudLink-rt (core service + transport + channel SPI)
    └─ cloudLinkAzure-rt   (Azure SAS auth, Azure AMQP stack)
        └─ cloudLinkForge-rt (Forge handler factories, RPK auth, full channel configs)
            ├─ cloudLinkNcs-rt   (NCS handler factories, subset of channels)
            └─ cloudLinkHonSbp-rt (HonSBP handler factories, schedule specialization)
```

[ev: cloudLinkAzure-rt module.xml dep `cloudLink-rt`; cloudLinkForge-rt module.xml
dep `cloudLinkAzure-rt`; cloudLinkNcs-rt and cloudLinkHonSbp-rt module.xml dep
`cloudLinkForge-rt`]

Each backend registers two handler factories (AMQP + HTTP), one channel-config factory,
and optionally a file-upload manager:

```java
// cloudLinkForge example — registered automatically via BAbstractCloudLinkHandlerFactory.TYPE
public class BForgeAmqpHandlerFactory extends BAbstractCloudLinkHandlerFactory { ... }
public class BForgeHttpHandlerFactory  extends BAbstractCloudLinkHandlerFactory { ... }
public class BForgeChannelConfigFactory extends BAbstractChannelConfigFactory   { ... }
```

[ev: code cloudLinkForge module.xml]

**Module permissions per backend** (template for your own module.xml):

```xml
<!-- repeat per backend, name scoped to the backend -->
<java-permission class="com.tridium.nre.security.KeyRingPermission" name="myBackend"/>
<java-permission action="read,write" class="java.io.FilePermission"
    name="${protected.station.home}${/}cloudLinkModel${/}-"/>
<java-permission action="read,write" class="java.io.FilePermission"
    name="${protected.station.home}${/}cloudLinkSchedule${/}-"/>
```

[ev: cloudLinkForge module.xml `KeyRingPermission("cloudLinkNcs")`; cloudLink-rt module.xml
FilePermission cloudLinkModel + cloudLinkSchedule]

---

## 6 · Point tagging — the `nc` SmartTagDictionary

The `Niagara Cloud` tag dictionary (`nc` prefix, `BSmartTagDictionary`) ships in
cloudLink-rt's palette. Drop it on a point to opt it into cloud sync. It counts against the tags licence like
any dictionary: only the first two are enabled (`Dictionary.limit`) and `n` takes one (corpus B1329 §1329.1 row 1),
so on a station that already has `n` and another dictionary, `nc` would be the third and go to fault [INFER from
the licence rule]; see `types/logic.md` § Ship a tag dictionary. `[ev: retro devguide415-r2-security-tags-analytics Δ5]`

| Tag | Purpose |
|---|---|
| `nc:cloudId` | stable UUID identifying the point to the cloud backend |
| `nc:telemetryId` | telemetry channel identifier |
| `nc:commandReadable` | marks the point as cloud-readable via Commands channel |
| `nc:commandWritable` | marks the point as cloud-writable |
| `nc:schedulable` | marks the point for Schedule channel sync |

[ev: code cloudLink-rt module.palette `Niagara$20Cloud` SmartTagDictionary entry with
cloudId/telemetryId/commandReadable/commandWritable/schedulable tags]

The `BCloudIdManager` service child assigns and persists `nc:cloudId` UUIDs.
[ev: code cloudLink-rt BCloudConnectionService @NiagaraProperty cloudIdManager]

---

---

## 7 · Outbound HTTP — the decision ladder and the `enableNonDriverClients` gate `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ18]`

**Ladder** — take the first route that fits `[ev: retro devguide415-r2-shipped-libraries Δ7]` `[ev: corpus B1333 §1333.3]`:
1. **No code:** the HttpClient driver (needs the `http` licence and an active SMA; JSONPath parsing).
2. **Java through the station:** `HttpClientBuilder…build()` then `sendSync()` (`IHttpClient`), which inherits the
   service gates, the licence, the throttle, the request audit and CertManager trust. Mount it with
   `mountAtOrd(slotPathOrd, name)` — it throws `IllegalArgumentException` outside a running station
   (vineflower httpClient-rt 4.14.0.162 `HttpClientBuilder.java:167-181`); `withoutSendOnStart()` stops the
   first send at start.
3. **okhttp from `bin/ext`** via `nre(...)` (§ 2.1): unlicensed and outside every service gate, so trust,
   retries and timeouts are the module's job.
4. An uberjar only when 2 and 3 cannot do it.

**The gate is CLOSED by default and fails quietly.** `BHttpClientService.enableNonDriverClients` and
`enableInsecureHttp` both default false (vineflower httpClient-rt 4.14.0.162 `BHttpClientService.java:88-89`; javap
of `httpClient-rt.jar` 4.15.6.20 static init: `iconst_0`). Every builder client is a non-driver client:
`BHttpClient.isDriverBasedClient()` is `final` and returns false (same copy `BHttpClient.java:302-306`), whatever
module calls it. With the gate closed, `permittedByService` catches the refusal, writes it to the client's
health as a fault ("Non driver clients are disabled in the HttpClientService") and returns false
(`BHttpClientService.java:231-252`); `sendSync()` then returns `""` (`BHttpClient.java:277-285`, `sendSync` = `commonPoll(null)`).
No exception reaches the caller. Same message and health call in the 4.15.6.20 jar (javap). `[ev: corpus B1346 §1346.2]`
`[ev: retro devguide415-r3-drivers-integration Δ4]`
- **Commissioning item:** a module on route 2 states "set `HttpClientService.enableNonDriverClients=true`" (and
  `enableInsecureHttp` only for a plain-http endpoint) in its runbook, and the module reads its client's health
  so an empty reply is not mistaken for an empty payload.
- An unmounted client that passes the gate reaches the request audit, which dereferences the component's slot
  path (`BHttpClientService.java:270`); a null there is a likely NPE `[INFER]` (not executed, niagara-research
  B1346-G1). Mount it.
- **Raw `HttpURLConnection` / okhttp bypass the gate entirely** — they are plain JVM calls and are not routed
  through `BHttpClientService`. Our Apillm importer uses `HttpURLConnection` for this reason. Document in the
  module README which route applies.
- **OAuth2:** no OAuth2 auth type ships. Subclass `BAuthorizationHeaderScheme` (abstract `getAuthHeaderValue`,
  vineflower httpClient-rt 4.14.0.162 `datatypes/auth/BAuthorizationHeaderScheme.java:29`), fetch and cache the token in
  `getAuthHeaderValue` through `oauth2-rt` `OAuth2ClientCredentialsGrantRequest` (the `email-rt` pattern: `BOAuthEmailClientSecretAuthenticator.java:6`, email-rt 4.14.0.162), plug
  it in with `withAuthenticator(BTypeSpec)` (`HttpClientBuilder.java:93`) and depend on `oauth2-rt`; the client
  keeps the gate, throttle and audit. A static `withBearerAuthenticator(token)` (`:116`) with a token our code
  refreshes is the simpler route. `[ev: retro devguide415-r3-drivers-integration Δ5]` `[ev: corpus B1346 §1346.3]`
- **Security axis note:** this gate is OUTBOUND-only — it does NOT affect inbound auth. See `types/security.md §10` for the full outbound-vs-inbound axis separation.

`[ev: corpus B1069]`

---

## 8 · `BICloudConnector` — cloud integration entry point (PD-25) `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ25]`

`BICloudConnector` is the lightweight Niagara cloud-integration SPI for modules that do NOT need the full `cloudLink` stack (§1–§6). It provides a `CompletableFuture`-based API.

**Implementation via `BConnectorImpl`:**

```java
public class BMyCloudConnector extends BConnectorImpl {

    @Override protected void doConnect() throws Exception {
        // open the connection / authenticate
    }

    @Override protected void registerDevice() throws Exception {
        // register or claim the device in the cloud backend
    }

    @Override protected void doDisconnect() throws Exception {
        // graceful close
    }

    @Override protected void doPing() throws Exception {
        // liveness check; throw when the cloud endpoint is unreachable
    }

    @Override public void sendMessage(byte[] payload, Map<String, String> props, CompletableFuture<String> result) {
        // serialize to the protocol format, send, then result.complete(...)
    }
}
```

**`nCloudDriver` bridge — device/point tree:** when the cloud service must map to a standard Niagara device/point tree, add an `nCloudDriver` module that bridges the `BICloudConnector` service to a `BDeviceNetwork`/`BDevice`/`BProxyExt` hierarchy. This keeps the cloud connector stateless (session management only) and the driver side standard (tuning, poll, subscription). `[ev: corpus B1076]`

---

## 9 · MQTT — broker auth and telemetry `[ev: retro devguide415-r2-drivers-protocols Δ2]`

Stock driver first (§ 2.1 "MQTT: native first"). Guide paths are
`niagara-help/versions/4.15.6.20/guides-clean/Mqtt/`.

**Set the port explicitly.** The Generic authenticator defaults to `connectionType=UserLoginOverSSL` but
`brokerPort` defaults to 1883 (javap of `abstractMqttDriver-rt.jar` 4.15.6.20, `BAbstractMqttAuthenticator`
static init: `sipush 1883`; guide `abstractMqttDriver-GenericMqttAuthenticator.txt:19`). A TLS broker listens
on 8883: set it, or the connection fails against a TLS listener. Owed Go lint G87 `mqtt-tls-port` (ledger).

| Authenticator | Credentials | Note |
|---|---|---|
| Generic | user/password over TLS; mutual TLS with `useTlsClientAuth` + a certificate alias, optional Cert Signing Config renewal | `abstractMqttDriver-GenericMqttAuthenticator.txt:19-25`; `Anonymous` warns and is not secure |
| AWS | client certificate from the AWS IoT portal, imported into the User Key Store | `abstractMqttDriver-AwsMqttAuthenticator.txt:19` |
| AWS JITP | same, just-in-time provisioning | port preset to 8883 (`AwsJitpMqttAuthenticatorabstractMqt-061BB575.txt:18`) |
| Azure SAS | SAS token, regenerated `Advance Expiration` (10 s) before expiry | `abstractMqttDriver-AzureMqttSasAuthenticator.txt:22`; port 8883 |
| GCP (ClearBlade) | JWT keys | `AbstractMQTTDriverWithGCPAuth.txt:9`; port 8883 (`SettingUpStationToConnectToGCP-632BA3D9.txt:38`) |

**Telemetry recipe** (cold-room or compressor values to a broker):
- One publish extension per value; `Retained` and `Publish Message on Change` default true
  (`abstractMqttDriver-MqttNumericObjectPublishExt.txt:25-26`). QoS 1 (at least once) is our choice for telemetry
  `[INFER]`. Enable LWT on the device (default off, `abstractMqttDriver-AbstractMqttDevice.txt:19`) so the
  consumer sees the station drop.
- A publish while the broker link is down is dropped silently and the point still reports ok (vineflower
  abstractMqttDriver-rt 4.14.0.162 `BMqttClientAbstractPublisherPoint.java:90-110`, corpus B1275 §1275.8). If
  gaps matter, add a heartbeat (the `publish` action on a trigger) and let the consumer flag missing samples.
- One scalar per topic: there is no JSON payload. A JSON document needs a String publish point fed by a
  jsonToolkit schema `output` (`types/json-toolkit.md` § 1). `[ev: corpus B1323 rows 3-6]`

## 10 · Watch and lease clients — leases, 503 and weak peers `[ev: retro devguide415-r2-drivers-protocols Δ10]`

For any client our module runs that polls a watch on another station (oBIX watch, a fox or HTTP poller):
- **Lease = max(2 × interval, interval + safety factor).** That is how the R2 oBIX client sizes its requested
  lease (default interval 2 s, safety factor 10 s); raise the interval to 10 s or more for a weak peer
  (guide 4.15.6.20 `EngNotes/aClientPollingTiming.txt:10-20`). Use the same rule for our own watch clients
  `[INFER]`, and re-create the watch when the server reports it expired.
- **HTTP 503 = the peer ran out of web-service threads** (`aClientPollingTiming.txt:34-36`): back off with a
  growing delay, never retry at the poll rate. Debug logging on the serving station "greatly slows" its answers
  (`:29`). `[ev: corpus B1336 row 3]`
- Field-side counterparts (capability overrides, queue depth points): `types/driver-authoring.md` § 11.4.

## 11 · RDBMS — supported databases and export from code `[ev: retro devguide415-r2-shipped-libraries Δ8]`

"Use Niagara's native rdb module" holds only inside these limits (guide 4.15.6.20
`Rdbms/DatabaseRequirement-Rdbms-7C2CBFED.txt`):
- **Four databases:** MySQL, Oracle, SQL Server and HSQLDB (`:7`); the 4.15.6.20 install ships `rdbMySQL`,
  `rdbOracle`, `rdbSqlServer` and `rdbHsqlDb` and no Postgres module. The external database runs on the
  Supervisor; a controller gets HSQLDB only (`:7-11`).
- **JDBC by hand:** the MySQL connector, renamed `mysql-connector-java.jar`, and Oracle's ojdbc go in
  `<niagara_home>/jre/lib/ext` (`:25`; `Rdbms/OracleDatabaseAndJDKCompatibility-2F50143A.txt:13`);
  `rdbSqlServer-rt.jar` bundles the Microsoft driver. Each database type needs its own licence feature
  (`Rdbms/PrereqChecklist-7B2F6B77.txt:17`).
- **Export semantics:** a daily or interval trigger appends from the last exported timestamp with no retry; the
  driver creates the tables, by history id (default) or by type (corpus B1340 row 6, B407).
- **Postgres or Supabase:** a custom `BRdbms` dialect with its own licence feature (corpus B785), or, as PANCCADIA
  does, an HTTPS connector (corpus B1273). `[ev: corpus B1340]`

**Adding an export from code** `[ev: retro devguide415-r3-drivers-integration Δ6]`: `BRdbmsHistoryExport` is
abstract (javap of `rdb-rt.jar` 4.15.6.20). Instantiate the type the database's frozen `histories` device ext
names and add it there (or in an archive folder below it); `BHistoryExport.isParentLegal` accepts only a parent
whose nearest archive folder returns a non-null `getExportDescriptorType()` (vineflower driver-rt 4.14.0.162
`javax/baja/driver/history/BHistoryExport.java:19-32`):

```java
// db = the concrete database (BSqlServerDatabase, BMySQLDatabase, ...); getHistories() is declared there
BHistoryExport e = (BHistoryExport) db.getHistories().getExportDescriptorType().getInstance();
e.setHistoryId(BHistoryId.make("/" + stationName + "/" + historyName));
db.getHistories().add("exp_" + historyName, e, 0, cx);   // engine thread, a Context allowed to add here
```

The recipe is corpus B1352 §1352.1 `[INFER]` (not executed). The licence gate is on the database:
`BRdbms.started()` runs `checkLicense()` and an unlicensed database goes `configFatal("Unlicensed: …")`
(vineflower rdb-rt 4.14.0.162 `javax/baja/rdb/BRdbms.java:256-301`; same methods in the 4.15.6.20 jar), so its
exports never run. Owed Go lint G88 `rdbms-dialect-descriptor` (ledger).

**See also:** `types/module-wiring.md` (fat-jar vs api-dep decision),
`types/third-party-libraries.md` (uberjar rules), `types/security.md` (KeyRing,
KeyRingPermission, FilePermission), `types/structure.md` (BAbstractService lifecycle).
