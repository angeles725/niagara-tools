<!-- review-status: folded -->
# 2026-09-18 · kit · third-party-library-integration-deltas

**Session**: corpus-mining for build-n4-module (operator: how to integrate helper libraries into N4 modules).
**Delta count**: 3

---

## What happened

Investigated the feasibility and mechanics of integrating external Maven libraries into a custom N4 module,
grounding the guidance in the N4 classloader topology (B617), the Compact3 JVM constraint (B1023,
build-verify.md §"Profile constraints"), the module signing pipeline (B18), and the build scaffold (B756/B960).

The two load-bearing corpus references are:
- **B617** — decompiled `ModuleClassLoader` / `ModuleExtClassLoader` / `ModuleManager`: one ClassLoader per
  module JAR; parent-first (boot/NRE) then embedded extJars then own jar then declared deps; cross-module
  visibility is `module.xml <dependency>`-gated; signature check for com/tridium/javax/baja entries and
  elevated-permission modules.
- **B347** — Niagara itself bundles Gson 2.9.0 inside `jsonToolkit-wb` as an embedded extJar: real
  first-party precedent that bundling a 3rd-party JAR is an accepted and working pattern.

No existing kit doc covers third-party library integration as a topic. The Compact3 gate check (source-level
`check_compact3_imports` WARN on java.awt/javax.swing/java.sql in -rt/-ux source) already exists in the
verify gate; this retro does NOT re-propose it. The gaps are: no how-to for packaging paths, no viability
guidance for the 19-library candidate list, and no build-verify.md pointer.

---

## Evidence

- **B617 §617.1** — `ModuleManager.java:320`: fresh `ModuleClassLoader` per `NModule`; no shared loader pool.
  Each module JAR has its own class namespace. `[ev: corpus B617]`
- **B617 §617.2** — delegation order: (1) parent/boot first; (2) embedded extJars
  (`extClassLoadersByResourcePath → ModuleExtClassLoader`); (3) own module JAR entries; (4) declared deps.
  A bundled 3rd-party JAR at an ext path is reached in step 2, BEFORE the module's own classes. `[ev: corpus B617]`
- **B617 §617.3** — cross-module visibility is fully dependency-gated: no ambient classpath; undeclared modules
  are invisible. `[ev: corpus B617]`
- **B617 §617.4** — a standalone module bundling a 3rd-party lib (graphql-java was the test case) is fully
  isolated; two modules can even bundle different versions of the same library with no clash, because each
  loader is independent. `[ev: corpus B617]`
- **B617 §617.5 — parent-shadowing caveat**: the boot parent is tried FIRST; if the NRE ships a conflicting
  class (e.g. an older Guava, a commons-* version) the platform copy wins. Shade (relocate to a private
  package) eliminates this. `[ev: corpus B617]`
- **B617 §617.5 — signing gate**: `verifyJarEntrySignature` is enforced for entries under `com/tridium/` or
  `javax/baja/` AND for modules that request elevated permissions (`ACCESS_CLASS`, `REFLECTION`,
  `MBEAN_PERMISSION`). For ordinary 3rd-party packages in a non-elevated module, the Tridium PKI chain is NOT
  applied to extJar entries. When you shade (merge all classes into your module jar), the entire output jar is
  signed by the niagara-signing plugin under NIAGARA4.SF — no extra step needed. Do NOT place bundled classes
  under `com/tridium/` or `javax/baja/` package paths. `[ev: corpus B617, B18 §18.1.1]`
- **B347 §347.3** — Niagara's own `jsonToolkit-wb` bundles Gson 2.9.0 as a packaged extJar: real precedent
  (first-party ship). `[ev: corpus B347]`
- **B1023 §ND4 table** — JACE JVM: Azul ZRE `azul-zre-compact3-qnx7-arm` v1.8.0.392.18 (Java 8, Compact3
  profile). The QNX7 ARM EABI5 ABI is the only JACE native target. `[ev: corpus B1023]`
- **build-verify.md §"Profile constraints — rt/ux Compact 3"** — java.awt, javax.swing, java.sql absent on the
  station; -rt and -ux must not use them. `check_compact3_imports` already WARN-flags these in source.
  `[ev: kit build-verify.md]`
- **B26 §26.7 / §26.4** — native JNI `.so` files are ELF 32-bit ARM EABI5 (QNX7) for JACE; any library
  bundling JNI native code for Linux-x64/Windows but NOT QNX7-ARM (e.g. jSerialComm, SQLite-JDBC) will load on
  Supervisor but throw `UnsatisfiedLinkError` on JACE at runtime. `[ev: corpus B26]`
- **B756 §756.1** — `api(":baja")` is the bare-form Niagara dep; no version; manifest `<dependency>` is for
  cross-module deps (other Niagara modules), NOT for bundled 3rd-party JARs. 3rd-party libs are NOT declared
  as module `<dependency>` entries. `[ev: corpus B756]`

---

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | Add new `types/third-party-libraries.md` covering: (a) the two packaging paths (BUNDLE/shade via Shadow plugin vs. library-module wrapper); (b) Compact3 gate reminder (java.awt/javax.swing/java.sql absent on JACE rt/ux); (c) Java-8 class-file ceiling (major 52 — library must be compiled for ≤ Java 8); (d) gradle-niagara packaging: add Shadow plugin to the part `.gradle.kts`, declare the 3rd-party coord as `implementation(...)`, configure `shadowJar` to relocate to a private package, wire `shadowJar` to run as part of the Niagara `jar` task; (e) signing: the shadow-merged jar is signed end-to-end by niagara-signing under NIAGARA4.SF — no extra step; do NOT place bundled classes under com/tridium/ or javax/baja/; (f) native-JNI caveat: a lib needing a QNX7-ARM .so is not viable on JACE — use Niagara's own drivers (platform serial, rdb) instead; (g) the library-module path (zero-type Niagara module, `runtimeProfile` rt, no `<type>` entries, other modules depend on it via `module.xml <dependency>`) — trade-offs; (h) viability table below | `types/third-party-libraries.md` (new file) | TPL-DOC1 |
| Δ2 | Add pointer bullet in `build-verify.md` §"Profile constraints — rt/ux Compact 3": one sentence after the `check_compact3_imports` note: "For guidance on bundling external Maven libraries (packaging paths, signing, the Java-8 ceiling, and a per-library viability table) see `types/third-party-libraries.md`." | `build-verify.md` §"Profile constraints — rt/ux Compact 3" | TPL-BVPTR1 |
| Δ3 | Cross-reference note in `types/third-party-libraries.md` header: "The rt/ux Compact3 SE-API constraint (java.awt/javax.swing/java.sql) is already enforced at the source-check gate by `check_compact3_imports` (WARN in `toolbelt/verify-module.sh`). No new check is needed for this retro; the viability table below identifies which libraries trigger this constraint." | `types/third-party-libraries.md` (new file, header note) | TPL-COMPACT3-XREF1 |

### Viability table (to be embedded in Δ1 `types/third-party-libraries.md`)

| Library | Maven artifact (representative) | Compact3-safe rt/ux? | Supervisor-only? | Native-blocked on JACE? | N4 verdict |
|---------|--------------------------------|---------------------|-----------------|------------------------|------------|
| Jackson | `com.fasterxml.jackson.core:jackson-databind:2.14.x` | **Yes** | No | No | Bundle (shade recommended); use 2.14.x (Java 8 compat; 2.15+ dropped Java 7, still Java 8-safe) |
| Gson | `com.google.gson:gson:2.10.x` | **Yes** | No | No | Bundle; first-party precedent: jsonToolkit-wb ships Gson 2.9.0 [ev: B347]; safe to bundle own copy (classloaders are isolated) |
| org.json | `org.json:json:20240303` | **Yes** | No | No | Bundle; zero runtime deps, minimal footprint |
| json-schema-validator (networknt) | `com.networknt:json-schema-validator:1.0.87` | **Yes** | No | No | Bundle + shade; heavy transitive tree (Jackson, SLF4J, commons-lang3); shade to avoid parent-shadow conflicts |
| Commons Math (commons-math3) | `org.apache.commons:commons-math3:3.6.1` | **Yes** | No | No | Bundle; pure math, no forbidden API; no transitives |
| Aviator | `com.googlecode.aviator:aviator:5.4.x` | **Yes** | No | No | Bundle; uses ASM for runtime bytecode generation (classloaded at runtime from your module loader — safe) |
| MVEL | `org.mvel2:mvel2:2.4.x` | **Yes** (java.beans is in Compact3) | No | No | Bundle; reflection-heavy — if it calls `AccessibleObject.setAccessible()` your module may need `ACCESS_CLASS` in `module-permissions.xml`, which forces cert-chain signing |
| Squirrel state machine | `org.squirrelframework:squirrel-foundation:0.3.x` | **Yes** | No | No | Bundle + shade; heavy transitives (Guava, Reflections); shade to avoid boot-parent Guava shadowing [ev: B617 §617.5] |
| Commons Lang (commons-lang3) | `org.apache.commons:commons-lang3:3.12.x` | **Yes** | No | No | Bundle; widely used utility; no forbidden API |
| Guava | `com.google.guava:guava:31.0-jre` | **Yes** | No | No | Bundle + shade; risk: NRE boot parent may carry an older Guava version that wins if unshaded [ev: B617 §617.5]; shade to private package |
| SLF4J + Logback | `org.slf4j:slf4j-api` + `ch.qos.logback:logback-classic` | SLF4J-api: **Yes**; Logback-classic: **conditional** (java.sql in DBAppender) | Logback: Supervisor preferred | No | Avoid Logback on rt/ux (DBAppender references java.sql at class-load time); use `java.util.logging.Logger` directly (kit doctrine: logic.md "use one for logging") |
| Typesafe Config | `com.typesafe:config:1.4.x` | **Yes** | No | No | Bundle; pure config parsing (HOCON/properties); no forbidden API |
| Eclipse Paho MQTT | `org.eclipse.paho:org.eclipse.paho.client.mqttv3:1.2.x` | **Yes** | No | No | Bundle; NOTE: Niagara ships a native mqtt-driver — prefer the native driver for production; Paho is suitable for custom publish-from-rt-logic scenarios |
| modbus4j | `com.infiniteautomation:modbus4j:3.0.x` | **Yes** | No | No | Bundle; NOTE: Niagara has a native modbus-driver — prefer it; modbus4j for custom protocol logic only |
| jSerialComm | `com.fazecast:jSerialComm:2.10.x` | N/A | Supervisor (Linux/Win) only | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE: bundles Linux-x64/Win/macOS .so only; `UnsatisfiedLinkError` at runtime on QNX ARM [ev: B26]; use Niagara's platform serial driver instead |
| OkHttp | `com.squareup.okhttp3:okhttp:4.12.x` | **Yes** | No | No | Bundle (+ okio + kotlin-stdlib transitives); for outbound HTTPS calls from rt logic; kotlin-stdlib adds ~1.5 MB to the jar |
| RxJava2 | `io.reactivex.rxjava2:rxjava:2.2.21` | **Yes** | No | No | Bundle; Reactive Streams style choice; adds ~900 KB; fine for complex async orchestration |
| H2 | `com.h2database:h2:2.2.x` | **No** (java.sql throughout) | **Supervisor-only** (-wb or -se profile) | No | java.sql-BLOCKED on rt/ux (station Compact3 has no java.sql); Supervisor only; use Niagara's native rdb module for database needs |
| SQLite-JDBC | `org.xerial:sqlite-jdbc:3.x` | **No** (JNI + java.sql) | Neither (no QNX .so) | **YES — no QNX7/ARM .so** | NATIVE-BLOCKED on JACE (no QNX .so) AND java.sql-blocked on rt/ux; neither path works on JACE; use Niagara's rdb driver |

---

## Lessons

1. The classloader isolation model (one `ModuleClassLoader` per module JAR, B617) is the architectural fact that
   makes 3rd-party bundling safe: there is no risk of cross-module version clash for a standalone module.
2. The parent-shadowing caveat (B617 §617.5) is non-obvious: the boot/NRE parent is tried FIRST, so any class
   Niagara ships at the NRE level will shadow your bundled copy unless you shade to a private package. This is
   the primary reason to use Shadow plugin even for libraries Niagara does not appear to ship.
3. The signing nuance is straightforward: shade → single jar → niagara-signing signs it as-is. No separate
   step. The constraint is package naming (no com/tridium/ or javax/baja/ in your bundled classes) and
   elevated-permission implications (if you need ACCESS_CLASS, REFLECTION, or MBEAN_PERMISSION, the cert-chain
   requirement tightens regardless of moduleVerificationMode).
4. The JNI native-blocked verdict for jSerialComm and SQLite-JDBC is hard: the JACE is QNX7 ARM EABI5
   (B1023/B26); no general-purpose JNI library ships a QNX .so. This is not a packaging problem — it is an
   absent binary.
5. H2's java.sql dependency is a softer-but-real blocker on rt/ux: the station Compact3 JVM omits java.sql
   (build-verify.md), so H2 cannot initialize on a JACE rt module even if the jar is present.
6. SLF4J+Logback is the most surprising entry: Logback-classic loads DBAppender which references java.sql, so
   it fails on the JACE rt classload even if you never configure a DB appender. The kit already documents the
   alternative (java.util.logging.Logger in logic.md).
7. The library-module path (wrap a lib as a zero-type Niagara module) is a legitimate alternative when the
   library is shared by many modules in the same product. Its overhead (separate jar, module.xml, signing, deploy
   chain) is only worth it at scale; for a single module, Shadow plugin is the simpler path.

---

**Status**: PENDING — INDEX row appended: `| 2026-09-18-third-party-library-integration-deltas.md | kit | 2026-09-18 | pending | 3 |`
