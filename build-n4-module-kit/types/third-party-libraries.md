# Third-Party Java Libraries in N4 Modules

> **This document covers the Java-in-JVM path only.** Browser-side JavaScript
> libraries (e.g. three.js loaded from an `rc/` resource folder) are a separate
> mechanism covered in `types/dashboard.md`.

> **Compact3 SE-API constraint**: the rt/ux Compact3 gate (`java.awt`,
> `javax.swing`, `java.sql` absent on JACE) is already enforced at the
> source-check gate by `check_compact3_imports` (WARN in
> `toolbelt/verify-module.sh`). No new check is needed; the viability table
> below identifies which libraries trigger this constraint. `[ev: Δ3 TPL-COMPACT3-XREF1]`

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

First-party precedent: Niagara's own `jsonToolkit-wb` bundles Gson 2.9.0 as an
embedded extJar — bundling a 3rd-party JAR is an accepted, working pattern.
`[ev: corpus B347 §347.3]`

---

## Two packaging paths

### Path A — Bundle / shade into the module JAR (Shadow plugin)

The standard approach for a single module that needs one or more external
libraries:

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
   is for cross-module Niagara deps only. `[ev: corpus B756 §756.1]`

3. Do **not** use `compileOnly` for a bundled library — it must be `implementation`
   so the classes are included in the output jar (gradle-niagara requires the jar
   to be self-contained / packaged). `[ev: corpus B756]`

4. The resulting merged jar is signed end-to-end by the `niagara-signing` plugin
   under `NIAGARA4.SF` — no extra signing step needed. The only constraint: do
   **not** place bundled classes under `com/tridium/` or `javax/baja/` package
   paths; those namespaces trigger the cert-chain signature check for
   non-elevated modules. `[ev: corpus B617 §617.5, B18 §18.1.1]`

**When to shade (relocate)**: always when there is any chance the NRE boot
parent carries the same library (Guava, commons-lang3, SLF4J API are common
candidates). A plain bundle without relocation works when you are certain the
NRE does not ship that package at the boot level.

### Path B — Wrap as a dedicated library Niagara module

Use this when the library is shared across many modules in the same product (the
overhead is only worth it at scale; for a single module, Path A is simpler):

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
   signing rules as any other module (NIAGARA4.SF). `[ev: corpus B18]`

**Trade-off**: separate jar, module.xml, signing, and deploy chain; adds
operational overhead. Worthwhile only when 3+ consumer modules share the same
heavy library.

---

## Hard constraints

### Compact3 gate — rt/ux JVM `[ev: corpus B1023 §ND4, build-verify.md §"Profile constraints — rt/ux Compact 3"]`

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

### Native JNI caveat `[ev: corpus B26 §26.7, §26.4]`

The JACE is QNX7 ARM EABI5. A library that bundles native `.so` files (Linux
x64, Windows DLL, macOS dylib) will:

- Load and run correctly on a Supervisor (Linux x64 / Windows x64).
- Throw `UnsatisfiedLinkError` at runtime on a JACE — there is no QNX7/ARM .so
  in any general-purpose Maven artifact.

This is **not** a packaging problem; the binary simply does not exist. The
solution is to use the equivalent Niagara native driver (platform serial, rdb,
modbus-driver, mqtt-driver) instead of the JNI library.

### Elevated-permission signing note

If a bundled library calls `AccessibleObject.setAccessible()` (e.g. MVEL via
reflection), your module may need `REFLECTION` or `ACCESS_CLASS` in
`module-permissions.xml`. Those permission groups force cert-chain validation
regardless of `niagara.moduleVerificationMode` — a DEV cert no longer suffices;
a Tridium-trusted cert is required at install time. `[ev: corpus B18 §18.3.1]`

---

## Library viability table

| Library | Maven artifact (representative) | Compact3-safe rt/ux? | Supervisor-only? | Native-blocked on JACE? | N4 verdict |
|---------|--------------------------------|---------------------|-----------------|------------------------|------------|
| Jackson | `com.fasterxml.jackson.core:jackson-databind:2.14.x` | **Yes** | No | No | Bundle (shade recommended); use 2.14.x (Java 8 compat; 2.15+ dropped Java 7, still Java 8-safe) |
| Gson | `com.google.gson:gson:2.10.x` | **Yes** | No | No | Bundle; first-party precedent: `jsonToolkit-wb` ships Gson 2.9.0 `[ev: B347]`; safe to bundle own copy (classloaders are isolated) |
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
| Eclipse Paho MQTT | `org.eclipse.paho:org.eclipse.paho.client.mqttv3:1.2.x` | **Yes** | No | No | Bundle; NOTE: Niagara ships a native mqtt-driver — prefer the native driver for production; Paho for custom publish-from-rt-logic scenarios |
| modbus4j | `com.infiniteautomation:modbus4j:3.0.x` | **Yes** | No | No | Bundle; NOTE: Niagara has a native modbus-driver — prefer it; modbus4j for custom protocol logic only |
| jSerialComm | `com.fazecast:jSerialComm:2.10.x` | N/A | Supervisor (Linux/Win) only | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE: bundles Linux-x64/Win/macOS .so only; `UnsatisfiedLinkError` at runtime on QNX ARM `[ev: B26]`; use Niagara's platform serial driver instead |
| OkHttp | `com.squareup.okhttp3:okhttp:4.12.x` | **Yes** | No | No | Bundle (+ okio + kotlin-stdlib transitives); for outbound HTTPS calls from rt logic; kotlin-stdlib adds ~1.5 MB to the jar |
| RxJava2 | `io.reactivex.rxjava2:rxjava:2.2.21` | **Yes** | No | No | Bundle; ~900 KB; fine for complex async orchestration |
| H2 | `com.h2database:h2:2.2.x` | **No** (`java.sql` throughout) | **Supervisor-only** (-wb or -se profile) | No | `java.sql`-BLOCKED on rt/ux (station Compact3 has no `java.sql`); Supervisor only; use Niagara's native rdb module for database needs |
| SQLite-JDBC | `org.xerial:sqlite-jdbc:3.x` | **No** (JNI + `java.sql`) | Neither (no QNX .so) | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE (no QNX .so) AND `java.sql`-blocked on rt/ux; neither path works on JACE; use Niagara's rdb driver |

`[ev: corpus B617, B347, B1023, B26, B18, B756]`
