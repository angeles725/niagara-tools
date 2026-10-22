<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-15 · kit · n4-course-build-deps-permissions-signing

**Session**: N4 Developer Certification Course (Tridium, N4.15.6.20) mined against the kit — decks 5–6, Labs 5/6/9/10, Lab 11 permissions, corpus B1234/B1235/B1242; the user compiled Labs 6–10 live (`training-rt`)
**Delta count**: 6

## What happened
Compiling the course labs surfaced build-time rules the kit does not state. Lab 10 needed three module dependencies for
one type: `api(":history-rt")` for `BHistoryToCsv`, then `api(":file-rt")` because its superclass `BITableToCsv` lives in
`file` (IntelliJ showed no class hierarchy and "BValue cannot be cast"), then `api(":bql-rt")` because
`BHistoryDatabase` implements `BIRelational` (javac: "class file for BIRelational not found"). The kit's dependency
table (`types/module-wiring.md:13-16`) has no supertype rule. Labs 6/9 requested `NETWORK_COMMUNICATION`, `REFLECTION`
and `MANAGE_EXECUTION`; the kit names only NETWORK_COMMUNICATION by example (`types/structure.md:23`) and the sign-always
groups (`build-verify.md:132`). Lab 10 used `com.tridium.history.exporters.BHistoryToCsv`, which the lab itself calls
private API that "could change between minor version updates"; the kit has no private-API rule. Deck 5 / B1234 teach
`allowDefaultProfile.set(false)` + `RestrictedSigningProfile`, the TSA and Module Info check, none of which the kit has
(it documents only the failure mode, `types/issues-and-gotchas.md:234` D4). B1235 teaches the JDK-8 toolchain check. Lab 10
also used two facets the kit's BFacets card lacks (`multiLine`, `targetType="baja:IDirectory"`) and the `@Facet`
expression syntax.

## Evidence
- Superclass/interface modules: "BITableToCsv, the actual superclass of BHistoryToCsv, is defined in the file module … Gradle can't figure out the class hierarchy" → `api(":file-rt")` — `[ev: course Labs/Lab Instructions.pdf p81-82]`; "class file for BIRelational cannot be found … defined in the bql module" → `api(":bql-rt")` — `[ev: course Labs/Lab Instructions.pdf p86]`
- Verified on N4.15.6.20 jars with `javap`: `BHistoryToCsv extends com.tridium.file.exporters.BITableToCsv` (class in `file-rt.jar`); `BHistoryDatabase … implements javax.baja.bql.BIRelational` (class in `bql-rt.jar`) — `[ev: /mnt/c/Niagara/Niagara-4.15.6.20/modules/{history,file,bql}-rt.jar]`; user's module ends with all three — `[ev: /mnt/c/N4-Developer-Training/Core-Class/N4_Labs/niagara_dev_home/training/training-rt/training-rt.gradle.kts]`
- Permission groups requested by the course: `NETWORK_COMMUNICATION` (hosts `*`, ports `3001`), `REFLECTION` (Jackson), `MANAGE_EXECUTION` — `[ev: course Labs/Lab9 Multithreaded/module-permissions.xml]`; Lab 11 asks `hosts="*" ports="*"` — `[ev: corpus B1239 §1239.12 row 10]`; group expansion, enforcement by `AccessControlException`, unsigned module → signature-requiring group skipped with a WARNING, debug order (`security.niagaraPolicy` at FINE, then `-@Dniagara.security.manager.disable`) — `[ev: corpus B1235 §1235.5]` `[ev: corpus B1235 §1235.6]`
- Private API: "BHistoryToCsv is in the com.tridium package, making it a part of Niagara's private API … could change between minor version updates … worth writing that functionality yourself" — `[ev: course Labs/Lab Instructions.pdf p83]`
- Signing: `allowDefaultProfile.set(false)` + `RestrictedSigningProfile`, `niagara.signing.standardtsa`, verify with Tools → Module Info — `[ev: corpus B1234 §1234.3]` `[ev: corpus B1234 §1234.6]`; Lab 5 skips the restricted profile so a mistyped alias silently self-signs — `[ev: corpus B1242 §1242.4 row 9]`; `gradlew clean` before `jar` after a signing change because the unsigned jar is up to date `[INFER]` — `[ev: corpus B1253 §1253.3 K10]`
- JDK-8 toolchain: `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` — `[ev: corpus B1235 §1235.2]`
- Facets: `@Facet(name = "BFacets.MULTI_LINE", value = "true")` on a `BOrdList`; `@Facet(name = "BFacets.TARGET_TYPE", value = "\"baja:IDirectory\"")` on a `BOrd` for a folder-only picker — `[ev: course Labs/Lab10 Jobs/BHistoriesToCsvService.java]` `[ev: course Labs/Lab Instructions.pdf p82-83]`; `%lexicon(slides:enabled.value)%` as TRUE_TEXT — `[ev: course Slides/7 Logging, Lexicons, and Testing.pdf p18]`; kit card has `targetType` but no `multiLine` and no syntax note — `[ev: code types/logic-authoring.md:489-535]`
- Kit grep 2026-10-15 (non-retro `*.md`/`*.sh`): `MANAGE_EXECUTION`, `allowDefaultProfile`, `javaToolchains`, `standardtsa`, `AccessControlException`, `multiLine`, `IDirectory`, `private API` → 0 hits each

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add the supertype rule: the compile classpath needs the module of every superclass and implemented interface of each Tridium type you use, not only the type's own module (`history-rt` + `file-rt` + `bql-rt` for `BHistoryToCsv`/`BHistoryDatabase`). Symptoms: javac "class file for X not found"; IntelliJ shows no hierarchy / "BValue cannot be cast". Recipe: `javap -cp <niagara_home>/modules/*.jar <type>` for extends/implements, `unzip -l` to find the jar. Lint: YES (helper) — `build.sh` post-failure hint: grep the gradle log for `class file for (\S+) not found`, map the class to its `modules/*.jar` and print the `api(":<mod>")` line to add. | `types/module-wiring.md` § dependency table + `toolbelt/build.sh` failure hints | `[ev: course Labs/Lab Instructions.pdf p81-86]` |
| Δ2 | Add a permission-group roster with triggers: `NETWORK_COMMUNICATION` (hosts + ports, scope tightly — the course's `*` is a teaching shortcut), `REFLECTION` (Jackson databind, signature required), `MANAGE_EXECUTION` (`modifyThread`: executor `shutdownNow`, `interrupt`); unsigned module → signature-requiring group skipped with a WARNING; diagnose `AccessControlException` via `security.niagaraPolicy` FINE before any SecurityManager bypass. Lint: YES (mechanical) — `verify-module.sh --src` cross-check: `ObjectMapper`/`com.fasterxml.jackson.databind` without REFLECTION, `shutdownNow(`/`.interrupt(` without MANAGE_EXECUTION, `HttpConnection`/`Socket`/`URL.openConnection` without NETWORK_COMMUNICATION → FAIL; `hosts="*"` with `ports="*"` → WARN. | `types/structure.md` § module-permissions.xml + `toolbelt/verify-module.sh` | `[ev: corpus B1235 §1235.5]` |
| Δ3 | Add a private-API rule: `com.tridium.*` (non-`javax.baja`) imports are private API that may break on a minor upgrade; prefer the `javax.baja` equivalent or write the function; when unavoidable, record it in the module README and pin `vendorVersion` floors. Lint: YES — `verify-module.sh --src` WARN listing every `import com.tridium.` outside an allow-list (e.g. NRE/gradle-plugin classes the kit already sanctions). | `types/module-wiring.md` (new § Private API) + `toolbelt/verify-module.sh` | `[ev: course Labs/Lab Instructions.pdf p83]` |
| Δ4 | Add the signing hardening recipe: `signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }` + a `RestrictedSigningProfile` so a wrong alias fails the build instead of self-signing; `niagara.signing.standardtsa`; verify in Workbench Tools → Module Info (green check + signer); run `gradlew clean` before `jar` after changing signing config. Lint: YES — `lint-structure.sh` WARN when a GROUP `build.gradle.kts` applies `com.tridium.niagara-signing` without `allowDefaultProfile.set(false)` (B1242 K5). | `build-verify.md` § signing + `toolbelt/lint-structure.sh` | `[ev: corpus B1234 §1234.6]` |
| Δ5 | Preflight the JDK-8 toolchain for a 4.15 target: run `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` and FAIL when no full JDK 8 is listed (B1242 K6). Lint: YES (mechanical, preflight check). | `toolbelt/preflight.sh` (new check) + `build-verify.md` | `[ev: corpus B1235 §1235.2]` |
| Δ6 | Extend the BFacets card: add `multiLine` (`BFacets.MULTI_LINE`, multi-line field editor, e.g. one ORD per line for a `BOrdList`) and the `targetType` value `"baja:IDirectory"` (folder-only ORD chooser); add the `@Facet` syntax note — `name` is a Java constant expression (`"BFacets.TARGET_TYPE"`), `value` is a Java expression, so a string value needs escaped quotes (`"\"baja:IDirectory\""`) and a lexicon reference is `"\"%lexicon(<module>:<key>)%\""`. Lint: no. | `types/logic-authoring.md` § BFacets key reference card | `[ev: course Labs/Lab10 Jobs/BHistoriesToCsvService.java]` |

## Lessons
- A missing supertype module fails at compile time with a misleading message; resolve the full extends/implements chain, not just the type's own jar.
- Each sensitive JVM capability maps to one permission group; the source code that needs it can be found mechanically.
- `com.tridium.*` is private API even when Tridium's own course uses it.
- Signing must fail closed: a default profile that silently self-signs hides a mistyped alias until the station refuses the module.
