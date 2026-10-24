# jsonToolkit — when to use it and how to extend it

The stock `jsonToolkit-rt` module (vendor Tridium, packages `com.tridiumx.jsonToolkit.*`; `META-INF/module.xml` of
the 4.15.6.20 jar) turns station components, alarms and histories into JSON and routes inbound JSON to points.
Guide paths are `niagara-help/versions/4.15.6.20/guides-clean/JsonToolkit/`. Library choice for hand-written JSON
code (json-smart, Gson): `types/third-party-libraries.md`.

---

## 1 · Decision rule — schema before a hand-rolled JSON exporter `[ev: retro devguide415-r2-shipped-libraries Δ9]`

Before writing a JSON servlet or exporter for off-station consumers, check whether a jsonToolkit component does it:

| Need | Stock component |
|---|---|
| a JSON document of point values | `BJsonSchema` and its `output` slot |
| alarms as JSON | `BJsonAlarmRecipient` |
| history rows as JSON, "send since last" | a bound history query plus `BHistoryExportMarkerFilter` (guide `ExportMarkers-94E0333F.txt`) |
| inbound JSON to points | `BJsonRouter` / `BJsonSelector` / `BJsonHandler`; the stock setpoint handler requires `runAsUser` (`SetpointHandlerAndWritingToPoints-A651B788.txt:32`) |

- **Licence:** the host needs `DR-JSON` or `DR-S-JSON`, and a production licence also needs an active SMA
  (`LicenseRequirementsJson-6DBD2640.txt:7-13`). Owed Go lint G89 `jsontoolkit-licence-note` (ledger).
- **No transport:** "The JSON Toolkit itself does not mandate the transport protocol used"
  (`TransportProtocols-Json-94D71FFE.txt:7`). The `output` string feeds an MQTT String publish point
  (`types/cloud-connector.md` § 9), an HTTP client or a servlet.
- **Cost:** a change regenerates the whole schema synchronously on the engine thread (corpus B348); keep schemas
  small and set the schema tuning policy `Min Write Time` (`jsonToolkit-JsonSchemaTuningPolicy.txt:18`).
- **Stay on the servlet-SPA route** (`types/dashboard.md`) for a bespoke UI, or when the host has no `DR-JSON`.

`[ev: corpus B1322 rows 1-8]`

## 2 · Extension points

| Extend | Implement | Evidence |
|---|---|---|
| `BJsonSchemaProperty<T>` (custom schema member) | `T getJsonValue()` only (§ 3) | `JsonSchemaTypes-Json-70BA9870.txt:35-50`; javap 4.15.6.20 |
| `BQueryResultWriter` (custom query output style) | register it as an agent on `jsonToolkit:JsonSchemaBoundQueryResult` | `CustomQueryStyle-Json-70E179E6.txt:9,33` |
| `BJsonRouter` / `BJsonSelector` / `BJsonHandler` (inbound) | `routeValue(BString message, Context cx)`, throw `RoutingFailedException` | `InboundComponents-Json-715A7D3D.txt:9` |
| `BTypeOverride` (per-type value encoding) | `overrideValueEncoding(BValue)` (§ 3) | javap 4.15.6.20 |
| `BAbstractInlineJsonWriter` | preferred to a Program object when it is distributed | `jsonToolkit-InlineJsonWriter.txt:9` |
| `JsonSchemaBuilder` | fluent API for schemas built in code | `BuilderClassAPI-Json-70E2DA19.txt` |

Depend on it with `api("Tridium:jsonToolkit-rt")`; the compiler enforces the dependency for every class above.

## 3 · Extending jsonToolkit — the exact contract `[ev: retro devguide415-r3-packaging-extension-control Δ2]`

- **Property:** `BJsonSchemaProperty<T>` is abstract and leaves one abstract method, `T getJsonValue()` from
  `BIJsonProperty`; `process(JSONWriter, boolean)` is a default method that writes the key and the converted value
  (javap of `jsonToolkit-rt.jar` 4.15.6.20). It refuses placement directly under the schema root ("Properties not
  valid at the top level", vineflower 4.14.0.162 `outbound/schema/BIJsonProperty.java:20-31`): put it inside an
  object or array member. `getJsonValue()` runs on the regeneration thread, so never block in it.
- **Type override:** extend the concrete `BTypeOverride` and override `overrideValueEncoding(BValue)`. The guide
  says to override `doOverride(BValue)` (`TypeOverrideExample-94E2114D.txt:42`), but `doOverride` is `public final`
  in the 4.15.6.20 jar and delegates to `overrideValueEncoding` (javap; vineflower `program/BTypeOverride.java:38-46`),
  so javac rejects the guide's version. Return the same instance to leave a value unchanged `[INFER]`.

`[ev: corpus B1350]` `[ev: retro devguide415-r2-shipped-libraries Δ9]`
