<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 9 -->
# 2026-10-15 · kit · devguide415-r2-shipped-libraries

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 built a shipped-libraries catalog (gap SL1) from the real 4.15.6.20 install and the HttpClient, Rdbms and JsonToolkit guides; this retro carries corpus B1320, B1321, B1322, B1333 and B1340. Complements `2026-10-15-n4-course-authoring-files-and-libraries.md` (**afl** Δ1 uberjar Path 0) and `…-horizons-authoring-security.md` (**has** Δ7).
**Delta count**: 9

## What happened
afl Δ1 and has Δ7 made `uberjar` the path for external jars; neither covers the jars Niagara already ships. The 4.15.6.20 `bin/ext` holds httpclient5 5.3.1, okhttp 4.12.0 + okio 3.9.1 + kotlin-stdlib 2.0.20, jose4j 0.9.6, commons-lang3 3.19.0, commons-codec, slf4j 2.0.17 (→ JUL), the OWASP encoder, servlet APIs and BouncyCastle; the devguide declares them version-less with `nre("group:artifact")`, and shipped modules (`httpClient-rt`, `cloudLink-rt`) use okhttp/jose4j with no bundled copy. The kit never mentions `nre("g:a")`, tells authors to BUNDLE okhttp and commons-lang3, and says okhttp ships in `net-rt` (it does not). Tridium wrapper modules (jsonSmart-rt, commonsIo-rt, commonsCompress-rt, apachePoi-rt, jodaTime-rt, oauth2-rt, abstractMqttDriver-rt) expose libraries at original package names via `api(":x-rt")`; `commonsLang-rt` is an empty shell in 4.15; the kit's Gson 2.9.0 is now 2.13.2. The HttpClient driver adds a licensed, gated, CertManager-trusted client with a public `HttpClientBuilder` API. The RDBMS driver supports only MySQL/Oracle/SQL Server/HSQLDB on a Supervisor — no Postgres.

## Evidence
- `nre` `[ev: niagara-help 4.15.6.20 devguide-clean/build.txt:238,250]` `[ev: niagara-help 4.15.6.20 devguide-clean/upgradingBuild.txt:289]`; okhttp location `unzip -l /mnt/c/Niagara/Niagara-4.15.6.20/modules/net-rt.jar` → 0 okhttp `[ev: corpus B1320 §1320.3]` vs `[ev: code types/cloud-connector.md:61]` `[ev: code types/third-party-libraries.md:167,174]`
- Wrapper versions (`pom.properties`): Gson 2.13.2, Jackson 2.22.1, Paho 1.2.5, Nimbus 10.6, Joda 2.12.7 `[ev: corpus B1321 §1321.1]` vs `[ev: code types/third-party-libraries.md:30,160]`
- HttpClient gates `[ev: code organized/httpClient/httpClient-rt/vineflower/com/tridium/httpClient/BHttpClientService.java:88-89]`; API `[ev: code niagara-help/versions/4.15.6.20/source/httpClient-rt/javax/baja/httpClient/IHttpClient.java:23]` `[ev: corpus B1333]`
- Rdbms `[ev: niagara-help 4.15.6.20 guides-clean/Rdbms/DatabaseRequirement-Rdbms-7C2CBFED.txt:16,46]` vs `[ev: code types/third-party-libraries.md:176]` `[ev: corpus B1340]`; JsonToolkit licence/extension `[ev: corpus B1322]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add "bin/ext first": before bundling, `ls <niagara_home>/bin/ext`; a library there is visible to every module — declare `nre("group:artifact")` with NO version and do not bundle; include the 4.15.6.20 catalog table (library · declare-as · permissions of calling code · prefer-when). Lint: yes — WARN on `uberjar(`/`implementation(` of okhttp, okio, kotlin-stdlib, commons-lang3, commons-codec, slf4j, jose4j, httpclient5, bouncycastle. | `types/third-party-libraries.md` after `:15` + `types/module-wiring.md` §1 | `[ev: corpus B1320 §1320.2]` |
| Δ2 | Fix okhttp: it ships in `bin/ext` (not `net-rt`); compile with `nre("com.squareup.okhttp3:okhttp")`; one rule in both files ("do not bundle"); Commons Lang via `nre(...)` on 4.15+, `api(":commonsLang-rt")` on 4.14. Lint: yes — grep `okhttp3.*net-rt`. | `types/cloud-connector.md:61-63` + `types/third-party-libraries.md:167,174` | `[ev: corpus B1320 §1320.3 C1-C2]` `[ev: corpus B1333 §1333.2 C1-C2]` |
| Δ3 | Version drift note: `bin/ext` versions change per release (httpclient 4.5 → httpclient5 in 4.15.6, kotlin 1.9 → 2.0); never pin them; `import org.apache.http.` relying on the 4.14 boot copy breaks on 4.15.6. Lint: yes. | `types/third-party-libraries.md` | `[ev: corpus B1320 §1320.1 row 4]` |
| Δ4 | Add the Tridium-wrapped libraries table (module · library+version · `api(":x-rt")` · permissions · prefer-when) for jsonSmart, jsonToolkit, commonsIo, commonsCompress, apachePoi, jodaTime, oauth2, abstractMqttDriver, httpClient; note commonsLang-rt is empty in 4.15. Lint: partial. | `types/third-party-libraries.md` near `:160` | `[ev: corpus B1321 §1321.1]` |
| Δ5 | Update stale facts: Gson 2.13.2 (not 2.9.0), Jackson 2.22.1 / JJWT 0.12.7 in abstractMqttDriver (no AWS SDK/Joda), stock MQTT module name `abstractMqttDriver` (not "mqtt-driver"). Lint: yes — grep `2\.9\.0`, `mqtt-driver`. | `types/third-party-libraries.md:30,143,159-160,171` | `[ev: corpus B1321 §1321.2]` |
| Δ6 | State that depending on a wrapper module does not grant its permissions; your module requests what its own code needs. Lint: no. | `types/module-wiring.md` | `[ev: corpus B1321 §1321.3 D3]` |
| Δ7 | Add the outbound-HTTP decision ladder: (a) HttpClient driver, no code (`http` licence + SMA, JSONPath parsing); (b) `HttpClientBuilder…build().sendSync()` (gates, licence, throttle, CertManager trust; `mountAtOrd` only in a station; `withoutSendOnStart`); (c) `nre` okhttp (unlicensed, own trust/retries); (d) uberjar only if needed. Lint: partial. | `types/cloud-connector.md` §7 + `types/issues-and-gotchas.md` H2 | `[ev: corpus B1333 §1333.3]` |
| Δ8 | Qualify "use Niagara's native rdb module": MySQL/Oracle/SQL Server/HSQLDB on a Supervisor only; MySQL/Oracle JDBC in `jre/lib/ext`; per-database licence; Postgres/Supabase → custom `BRdbms` dialect or HTTPS connector. Lint: partial — grep `postgres`. | `types/third-party-libraries.md:176-177` + `types/utility-lib.md:8` | `[ev: corpus B1340]` |
| Δ9 | Add a jsonToolkit decision rule and extension card: schema `output`/alarm recipient/history export before a hand-rolled JSON servlet; `DR-JSON`/`DR-S-JSON` + SMA; extension points `BJsonSchemaProperty<T>`, `BQueryResultWriter` agent, `BJsonHandler.routeValue`, `BTypeOverride`, `BAbstractInlineJsonWriter`, `JsonSchemaBuilder`. Lint: partial. | `types/dashboard.md:29` + `types/cloud-connector.md` §1 | `[ev: corpus B1322]` |

## Lessons
- The install is the authority for library versions; the corpus figures from a 4.14 decompile go stale across releases.
- "Bundle it" is wrong whenever the boot parent already ships the package — check `bin/ext` first.
- A shipped driver often replaces a hand-written client; the kit should present it before the code route.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-15-devguide415-r2-shipped-libraries.md | kit | 2026-10-15 | pending | 9 |`
