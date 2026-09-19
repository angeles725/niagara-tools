# Cloud / IoT connector — `BCloudConnectionService`, transport, auth, channel SPI

How to author a cloud/IoT connector module in the cloudLink family. This doc covers the
four extension seams: the central service, the transport layer, the authenticator, and
the per-data-type channel. Protocol-specific message formats and backend telemetry schemas
are out of scope.

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
| `msgFactoryMap` | `Map<String, BAbstractCloudLinkHandlerFactory>` | `platform + "." + transportType` |
| `configFactoryMap` | `Map<String, BAbstractChannelConfigFactory>` | `platform` string |
| `fileUploadManagerMap` | `Map<String, BAbstractFileUploadManager>` | `platform` string |

Registration happens lazily at first lookup via `Sys.getRegistry().getConcreteTypes(Factory.TYPE.getTypeInfo())` — the registry scans every loaded module for concrete subtypes
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

Backed by **okhttp3**, which ships in `net-rt` (a Tridium platform module). Do NOT
bundle okhttp3 yourself — declare `net-rt` as a module dependency and let the platform
supply it. [ev: code cloudLink-rt BHttpTransport imports okhttp3.*; module.xml dep net-rt]

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
properties.** Each backend module declares a `KeyRingPermission` named after itself in
`module.xml`; the authenticator calls the platform KeyRing API at runtime to read tokens
or certificates. This keeps secrets out of the station database (config.bog).

| Concrete class | Backend | Credential store |
|---|---|---|
| `BPasswordAuthenticator` | core | KeyRing via `SYSTEM_PASSWORD` |
| `BTokenAuthenticator` | core | KeyRing |
| `BAzureSasAuthenticator` | cloudLinkAzure | `KeyRingPermission("*")` |
| `BFederatedIdentityAuthenticator` | cloudLinkForge | KeyRing |
| `BRpkAuthenticator` | cloudLinkForge | KeyRing (RPK certificate) |
| `BRpkAuthenticatorHonSbp` | cloudLinkHonSbp | KeyRing |

[ev: code cloudLink-rt module.xml types list; cloudLinkAzure module.xml
`KeyRingPermission("*")`; cloudLinkNcs module.xml `KeyRingPermission("cloudLinkNcs")`;
cloudLinkHonSbp module.xml `KeyRingPermission("cloudLinkHonSbp")`]

**Gotcha:** `BUsernameAndPassword` is a helper struct — it is NOT `BAbstractClientAuthenticator`.
The actual authenticator calls `canAuthenticate()` / `isRegistered()` before any message
send; a false return from either will gate outbound traffic. [INFER: method contract
derived from abstract signature + call site in BCloudConnectionService]

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

[ev: cloudLinkNcs module.xml `KeyRingPermission("cloudLinkNcs")`; cloudLink-rt module.xml
FilePermission cloudLinkModel + cloudLinkSchedule]

---

## 6 · Point tagging — the `nc` SmartTagDictionary

The `Niagara Cloud` tag dictionary (`nc` prefix, `BSmartTagDictionary`) ships in
cloudLink-rt's palette. Drop it on a point to opt it into cloud sync:

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

**See also:** `types/module-wiring.md` (fat-jar vs api-dep decision),
`types/third-party-libraries.md` (uberjar rules), `types/security.md` (KeyRing,
KeyRingPermission, FilePermission), `types/structure.md` (BAbstractService lifecycle).
