# Third-Party Java Libraries in N4 Modules

> **This document covers the Java-in-JVM path only.** Browser-side JavaScript
> libraries (e.g. three.js loaded from an `rc/` resource folder) are a separate
> mechanism covered in `types/dashboard.md`.

> **Compact3 SE-API constraint**: the rt/ux Compact3 gate (`java.awt`,
> `javax.swing`, `java.sql` absent on JACE) is already enforced at the
> source-check gate by `check_compact3_imports` (WARN in
> `toolbelt/verify-module.sh`). No new check is needed; the viability table
> below identifies which libraries trigger this constraint. `[ev: retro third-party-library-integration-deltas Δ3]`

---

## Background: the N4 classloader model `[ev: corpus B617]`

Niagara creates one fresh `ModuleClassLoader` per module JAR (`ModuleManager.java:320`).
Delegation order:

1. Boot/NRE parent (tried **first** — NRE-shipped classes always win here)
2. Embedded extJars (`ModuleExtClassLoader`, reached before the module's own classes)
3. Module JAR entries
4. Declared `<dependency>` modules (cross-module, fully gated by `module.xml`)

This isolation means two modules can each bundle different versions of the same
library without clashing. It also means an NRE-shipped class (e.g. an older
Guava, a commons-* version) **shadows your bundled copy** unless you relocate
(shade) it to a private package. `[ev: corpus B617 §617.5]`

First-party precedent: Niagara's own `jsonToolkit-wb` bundles Gson 2.13.2 (4.15.6.20) as
classes unpacked into its own jar (`com/google/gson/*`, no nested jar) — bundling a 3rd-party library is an accepted, working pattern.
`[ev: corpus B347 §347.3, B1321 §1321.2 C1]`

---

## Before bundling: `bin/ext` first, then a wrapper module `[ev: retro devguide415-r2-shipped-libraries Δ1]`

**Step 1 — `ls <niagara_home>/bin/ext`.** A library there is visible to every module (boot delegation, above). Declare it
with `nre("group:artifact")` and NO version — the devguide's `nre` configuration is "dependencies on jars in !bin/ext"
(4.15.6.20 `build.txt:186`; `upgradingBuild.txt:281`: "libraries in !bin/ext and its subfolders") — and never bundle it.
**The versions drift between releases**, so never pin them and re-check after an upgrade
`[ev: retro devguide415-r2-shipped-libraries Δ3]` (listing of each install's `bin/ext`):

| Library | 4.13.2.18 (iC-Niagara) | 4.14.0.162 (Honeywell OptimizerSupervisor) | 4.15.6.20 | Declare as |
|---|---|---|---|---|
| okhttp (+ okio, kotlin-stdlib) | 4.11.0 | 4.12.0 (kotlin 1.9.10) | 4.12.0 (okio 3.9.1, kotlin 2.0.20) | `nre("com.squareup.okhttp3:okhttp")` |
| Apache HttpClient | httpclient 4.5.13 | httpclient 4.5.13 | **httpclient5 5.3.1** (no `org.apache.http`) | 4.15.6: `nre("org.apache.httpcomponents.client5:httpclient5")` |
| jose4j | 0.9.3 | 0.9.6 | 0.9.6 | `nre("org.bitbucket.b_c:jose4j")` |
| commons-codec | 1.15 | 1.16.1 | 1.19.0 | `nre("commons-codec:commons-codec")` |
| commons-lang3 | — | not in `bin/ext`; in the `commonsLang-rt` module (424 `lang3` entries) | 3.19.0 | 4.15: `nre("org.apache.commons:commons-lang3")` |
| slf4j-api | 2.0.3 | 2.0.9 | 2.0.17 (+ slf4j-jdk14) | `nre("org.slf4j:slf4j-api")` (routes to JUL) |
| OWASP encoder | 1.2.3 | 1.2.3 | 1.3.1 | `nre("org.owasp.encoder:encoder")` |

So `import org.apache.http.` that compiled against the 4.13/4.14 copy breaks on 4.15.6, and a module that must run on
4.14 and 4.15 either bundles one HTTP client itself or uses okhttp. "—" = not checked. The full 4.15.6.20 catalog with
the permissions each needs is corpus B1320 §1320.2. `[ev: corpus B1320 §1320.1-§1320.2]`

**Step 2 — a Tridium wrapper module**, declared with `api(":<module>")` (4.15.6.20, corpus B1321 §1321.1)
`[ev: retro devguide415-r2-shipped-libraries Δ4]`:

| Module | Wraps | Prefer when |
|---|---|---|
| `jsonSmart-rt` | json-smart 2.6.0 (`net.minidev.json`) | small JSON parse/write in rt code |
| `jsonToolkit-rt` | Tridium JSON components (jsonpath inside) | station-level JSON exchange by components |
| `commonsIo-rt` / `commonsCompress-rt` | commons-io 2.20.0 / commons-compress 1.28.0 | stream, file, archive helpers |
| `apachePoi-rt` | Apache POI | Excel read/write |
| `jodaTime-rt` | Joda-Time 2.12.7 | interop only; new code uses `java.time` |
| `oauth2-rt` | nimbus-jose-jwt, oauth2-oidc-sdk | client-credentials / JWT tokens (`types/cloud-connector.md` § 7) |
| `abstractMqttDriver-rt` | Paho mqttv3, Jackson databind | a cloud MQTT driver |
| `httpClient-rt` | no library; `javax.baja.httpClient` over `bin/ext` okhttp | REST through the station service (`types/cloud-connector.md` § 7) |

`commonsLang-rt` is EMPTY in 4.15.6.20 (6 entries: manifest, `module.xml`, lexicon; `unzip -l modules/commonsLang-rt.jar`):
use the `bin/ext` commons-lang3 there. **A wrapper does not lend its permissions:** your module's own
`module-permissions.xml` still requests every group its code needs (`types/structure.md` § module-permissions.xml).
`[ev: retro devguide415-r2-shipped-libraries Δ6]` `[ev: corpus B1321 §1321.3]` The bundling WARN (okhttp, okio,
kotlin-stdlib, commons-lang3, commons-codec, slf4j, jose4j, httpclient5, bouncycastle in `uberjar(`/`implementation(`, and
`import org.apache.http.`) is owed as Go lint G35 `shipped-library-bundle`.

**Step 3 — bundle** (the paths below) only when neither exists.

## Packaging paths

### Path 0 — the plugin's `uberjar` (Tridium default; use this first)

The devguide's form: declare the library under the `uberjar` configuration
(`testUberjar` for a test-only library such as Mockito 4.x) and the plugin
embeds it in the part jar, with no `module.xml` entry:

```kotlin
dependencies {
    uberjar("com.fasterxml.jackson.core:jackson-databind:2.21.2")
    testUberjar("org.mockito:mockito-inline:4.5.1")
    api(":baja")            // api(...) only for Niagara modules
}
```

"You should only use the configurations listed above. Other Gradle configurations
may not interact correctly with the Niagara build environment" (devguide
`build.txt:187-193`). `uberjar` does not relocate, so the boot-parent shadowing
rule below still applies; Path A is the fallback only when a library must be
relocated. `[ev: corpus B1313 §1313.3 C1]` `[ev: retro devguide415-horizons-authoring-security Δ7]`
`[ev: retro n4-course-authoring-files-and-libraries Δ1]`

### Path A — Shade with relocation (Shadow plugin; unsupported fallback)

Only when a bundled library collides with a class the NRE boot parent ships and
must move to a private package. It uses Gradle configurations the devguide does
not support, so the build is ours to keep working:

1. Apply the Shadow plugin in the part's `build.gradle.kts`:

   ```kotlin
   plugins {
       id("com.github.johnrengelman.shadow") version "7.1.2"
   }

   dependencies {
       implementation("com.fasterxml.jackson.core:jackson-databind:2.14.3")
       // add other 3rd-party coords here; NOT api() — these are internal
   }

   tasks.named<com.github.jengelman.gradle.plugins.shadow.tasks.ShadowJar>("shadowJar") {
       // Relocate to avoid boot-parent shadowing (recommended for widely-used libs)
       relocate("com.fasterxml.jackson", "com.acme.mymodule.shaded.jackson")
       archiveClassifier.set("")  // replace the regular jar output
   }

   // Wire shadowJar into the Niagara build task so gradle-niagara picks it up
   tasks.named("jar") { dependsOn("shadowJar") }
   ```

2. Do **not** declare the 3rd-party library as a `module.xml <dependency>` — that
   is for cross-module Niagara deps only. `[ev: corpus B617 §617.3]`

3. Do **not** use `compileOnly` for a bundled library — it must be `implementation`
   so the classes are included in the output jar (gradle-niagara requires the jar
   to be self-contained / packaged). `[INFER]`

4. The resulting merged jar is signed end-to-end by the `niagara-signing` plugin
   under the signing alias's `META-INF/<ALIAS>.SF` — no extra signing step needed. The only constraint: do
   **not** place bundled classes under `com/tridium/` or `javax/baja/` package
   paths; those namespaces trigger the cert-chain signature check for
   non-elevated modules. `[ev: corpus B617 §617.5, B18 §18.1.1]`

**When to shade (relocate)**: when the NRE boot parent carries the same
library (Guava, commons-lang3, SLF4J API are common candidates). A plain
`uberjar` bundle (Path 0) works when the NRE does not ship that package at the
boot level.

### Path B — Wrap as a dedicated library Niagara module

Use this when the library is shared across many modules in the same product (the
overhead is only worth it at scale; for a single module, Path 0 is simpler):

1. Create a new Niagara module (e.g. `myProduct-lib`) with:
   - `module.xml`: `runtimeProfile="rt"`, no `<type>` entries (zero-types), a
     `<vendor>` block, and a `<version>`.
   - The 3rd-party JAR packed into the module jar (via Shadow or a manual
     `from(configurations.runtimeClasspath)` copy).
   - A `module-permissions.xml` if elevated access is needed.

2. Consumer modules declare a `<dependency name="myProduct-lib"/>` in their own
   `module.xml`. The N4 classloader makes the library module's classes visible to
   dependents through step 4 of the delegation order. `[ev: corpus B617 §617.3]`

3. Deploy, sign, and version this library module separately — it follows the same
   signing rules as any other module (`META-INF/<ALIAS>.SF`). `[ev: corpus B18]`

**Trade-off**: separate jar, module.xml, signing, and deploy chain; adds
operational overhead. Worthwhile only when 3+ consumer modules share the same
heavy library.

---

## Hard constraints

### Compact3 gate — rt/ux JVM `[ev: corpus B1023 §1023.3, build-verify.md §"Profile constraints — rt/ux Compact 3"]`

The JACE JVM is Azul ZRE `azul-zre-compact3-qnx7-arm` v1.8.0 (Java 8,
Compact3 profile). The following packages are **absent** on the station:

- `java.awt.*`
- `javax.swing.*`
- `java.sql.*`

Any library that references these at class-load time (even in code paths you
never call) will throw `NoClassDefFoundError` on a JACE `-rt` or `-ux` module.
`check_compact3_imports` in `toolbelt/verify-module.sh` WARNs on source-level
references; the viability table below calls out the libraries that trigger this.

### Java-8 class-file ceiling (major version 52)

The JACE JVM is Java 8 only. A library JAR compiled for Java 9+ (class file
major ≥ 53) will throw `UnsupportedClassVersionError` at load time. Always
verify with `javap -verbose` or check the library's release notes. Most popular
libraries still ship a Java 8–compatible artifact; look for a `-jre8` or
`-android` classifier, or check `Automatic-Module-Name` in the manifest.

**Multi-release jars are fine:** a library may ship Java 9+ classes under
`META-INF/versions/<N>/` (Jackson 2.21.2 ships major-55 ones) — a Java 8 runtime
never loads them, so the base classes decide. Path 0's `uberjar("com.fasterxml.jackson.core:jackson-databind:2.21.2")`
embedded databind + core + annotations (1107 classes) and ran live; "pin to Java-8
builds" means a Java-8 BASE, not a jar without `META-INF/versions/`.
`verify-module.sh`'s bytecode check and `lint-bundled-jar-class-version.sh` both
skip `META-INF/versions/**` (the bytecode row counts them).
`[ev: retro n4-course-signed-jars-gate-fix Δ3]`

### Native JNI caveat `[ev: corpus B26 §26.7, §26.4]`

The JACE is QNX7 ARM EABI5. A library that bundles native `.so` files (Linux
x64, Windows DLL, macOS dylib) will:

- Load and run correctly on a Supervisor (Linux x64 / Windows x64).
- Throw `UnsatisfiedLinkError` at runtime on a JACE — no general-purpose Maven
  artifact is known to ship a QNX7/ARM .so [INFER]; JNI itself loads on a JACE when built for it
  (`libciper.so`, ARM EABI5, B26 §26.7).

This is **not** a packaging problem; the binary simply does not exist. The
solution is to use the equivalent Niagara native driver (platform serial, rdb,
modbus-driver, `abstractMqttDriver`) instead of the JNI library.

### Elevated-permission signing note

If a bundled library calls `AccessibleObject.setAccessible()` (e.g. MVEL via
reflection), your module may need `REFLECTION` or `ACCESS_CLASS` in
`module-permissions.xml`. Both are among the six groups that need a signed module
whose cert the installation trusts (public CA, internal CA root or self-signed cert in
the user trust store); unsigned, the group is dropped with a WARNING
(`build-verify.md` § Signing per deploy target). `[ev: corpus B18 §18.3.1]` `[ev: corpus B1312 §1312.3 C1-C3]`

---

## Library viability table

| Library | Maven artifact (representative) | Compact3-safe rt/ux? | Supervisor-only? | Native-blocked on JACE? | N4 verdict |
|---------|--------------------------------|---------------------|-----------------|------------------------|------------|
| Jackson | `com.fasterxml.jackson.core:jackson-databind:2.14.x` | **Yes** | No | No | Bundle (shade recommended); use 2.14.x (Java 8 compat; 2.15+ dropped Java 7, still Java 8-safe) |
| Gson | `com.google.gson:gson:2.10.x` | **Yes** | No | No | Bundle; first-party precedent: `jsonToolkit-wb` ships Gson 2.13.2 in 4.15.6.20 (`META-INF/maven/…/gson/pom.properties`) `[ev: B347]` `[ev: corpus B1321 §1321.2]`; safe to bundle own copy (classloaders are isolated) |
| org.json | `org.json:json:20240303` | **Yes** | No | No | Bundle; zero runtime deps, minimal footprint |
| json-schema-validator (networknt) | `com.networknt:json-schema-validator:1.0.87` | **Yes** | No | No | Bundle + shade; heavy transitive tree (Jackson, SLF4J, commons-lang3); shade to avoid boot-parent conflicts |
| Commons Math | `org.apache.commons:commons-math3:3.6.1` | **Yes** | No | No | Bundle; pure math, no forbidden API; no transitives |
| Aviator | `com.googlecode.aviator:aviator:5.4.x` | **Yes** | No | No | Bundle; uses ASM for runtime bytecode generation (classloaded from your module loader — safe) |
| MVEL | `org.mvel2:mvel2:2.4.x` | **Yes** (`java.beans` is in Compact3) | No | No | Bundle; reflection-heavy — if it calls `setAccessible()` your module may need `ACCESS_CLASS` in `module-permissions.xml`, which forces cert-chain signing |
| Squirrel state machine | `org.squirrelframework:squirrel-foundation:0.3.x` | **Yes** | No | No | Bundle + shade; heavy transitives (Guava, Reflections); shade to avoid boot-parent Guava shadowing `[ev: B617 §617.5]` |
| Commons Lang | `org.apache.commons:commons-lang3:3.12.x` | **Yes** | No | No | Bundle; widely used utility; no forbidden API |
| Guava | `com.google.guava:guava:31.0-jre` | **Yes** | No | No | Bundle + shade; risk: NRE boot parent may carry an older Guava that wins if unshaded `[ev: B617 §617.5]`; shade to private package |
| SLF4J + Logback | `org.slf4j:slf4j-api` + `ch.qos.logback:logback-classic` | SLF4J-api: **Yes**; Logback-classic: **conditional** (java.sql in DBAppender) | Logback: Supervisor preferred | No | Avoid Logback on rt/ux: DBAppender references `java.sql` at class-load time even when no DB appender is configured; use `java.util.logging.Logger` directly (see `types/logic.md`) |
| Typesafe Config | `com.typesafe:config:1.4.x` | **Yes** | No | No | Bundle; pure config parsing (HOCON/properties); no forbidden API |
| Eclipse Paho MQTT | `org.eclipse.paho:org.eclipse.paho.client.mqttv3:1.2.x` | **Yes** | No | No | Bundle; NOTE: Niagara ships the licensed `abstractMqttDriver` (Paho 1.2.5, Jackson 2.22.1, JJWT 0.12.7 inside in 4.15.6.20) — prefer it for production (`types/cloud-connector.md` § 2.1); a bundled Paho bypasses its licence and device cap `[ev: retro devguide415-r2-shipped-libraries Δ5]` |
| modbus4j | `com.infiniteautomation:modbus4j:3.0.x` | **Yes** | No | No | Bundle; NOTE: Niagara has a native modbus-driver — prefer it; modbus4j for custom protocol logic only |
| jSerialComm | `com.fazecast:jSerialComm:2.10.x` | N/A | Supervisor (Linux/Win) only | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE: bundles Linux-x64/Win/macOS .so only; `UnsatisfiedLinkError` at runtime on QNX ARM `[INFER]`; use Niagara's platform serial driver instead |
| OkHttp | `com.squareup.okhttp3:okhttp:4.12.x` | **Yes** | No | No | Do NOT bundle: the NRE ships okhttp 4.12.0 + okio + kotlin-stdlib in `bin/ext`; compile with `nre("com.squareup.okhttp3:okhttp")` (`types/cloud-connector.md` § 2.1) |
| RxJava2 | `io.reactivex.rxjava2:rxjava:2.2.21` | **Yes** | No | No | Bundle; ~900 KB; fine for complex async orchestration |
| H2 | `com.h2database:h2:2.2.x` | **No** (`java.sql` throughout) | **Supervisor-only** (-wb or -se profile) | No | `java.sql`-BLOCKED on rt/ux (station Compact3 has no `java.sql`); Supervisor only; use Niagara's native rdb module for database needs, within its limits (`types/cloud-connector.md` § 11) |
| SQLite-JDBC | `org.xerial:sqlite-jdbc:3.x` | **No** (JNI + `java.sql`) | Neither (no QNX .so) | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE (no QNX .so) AND `java.sql`-blocked on rt/ux; neither path works on JACE; use Niagara's rdb driver |
| **Weka 3.8.x** | `nz.ac.waikato.cms.weka:weka-stable:3.8.6` | Yes (core; shade if GUI code paths touch AWT — absent on Compact3) | No (core algos) | No | **GPL-3.0 — NOT safe for distributed closed-source module.** Viable for offline training only (dev env; never ships in module). See `types/ml-libraries.md`. |
| **Smile 2.6.x** | `com.github.haifengl:smile-core:2.6.0` | Yes — core tabular algos (RF, logistic, k-means, SVM, LDA) are pure Java; avoid `smile-nlp` (native) | No (tabular core; JACE: BWorker required for heavy models) | No | Apache 2.0; Java 8 compat. **Pin to 2.x — Smile 3.x+ requires Java 11+.** Preferred tabular ML library. Bundle + shade. See `types/ml-libraries.md`. |
| **Deeplearning4j (DL4J) 1.0-M2.1+** | `org.deeplearning4j:deeplearning4j-core:*` | Yes (API) | **No — JNI (ND4J native backend)** | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE `[INFER]`; Apache 2.0; Supervisor-only. See `types/ml-libraries.md`. |
| **ONNX Runtime Java** | `com.microsoft.onnxruntime:onnxruntime:1.x` | Yes (API) | **No — JNI** | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE `[INFER]`; MIT; Supervisor-only. **Preferred inference path** for offline-trained Python models. See `types/ml-libraries.md`. |
| **XGBoost4J** | `ml.dmlc:xgboost4j:1.x` | Yes (API) | **No — JNI** | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE `[INFER]`; Apache 2.0; Supervisor-only. Train in Python, export to ONNX, infer via ONNX Runtime Java. See `types/ml-libraries.md`. |

**ML libraries:** see `types/ml-libraries.md` for the placement rule
(Supervisor vs. JACE-rt), the full library verdict table with safety architecture,
the train-offline/infer-ONNX pattern, and data / operating-mode discipline.

`[ev: corpus B617, B347, B1023, B26, B18, B756, B67]`
