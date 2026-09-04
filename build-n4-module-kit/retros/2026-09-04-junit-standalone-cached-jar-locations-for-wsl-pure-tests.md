# Retro (PROPOSED delta — propose-never-apply) — pin the exact cached JUnit/Hamcrest jar locations for the WSL standalone pure-test run

- **Date**: 2026-09-04
- **Origin**: operator asked "where is the JUnit for the standalone pure tests?" — the recipe was documented (`build-verify.md:96`, qa-stack retro) but it says only "`junit-4.13.2.jar` cacheado" without a concrete path, so each session re-hunts it.
- **Status**: PROPOSED. Adds no rule; pins a factual path + a ready command so the pure tests are runnable without a search. Following the skill's step 6.

---

## Finding

The pure-logic tests (`CompressorControl`, `ColdRoomControl` — the ZERO-Baja `final class` decision cores) run standalone in WSL because `niagaraTest` needs the native `bin/test` + a dev license and does NOT run in WSL (`build-verify.md:96`, B741/B743). The recipe needs **JUnit 4.13.2 + Hamcrest 1.3** (JUnit 4 API: `org.junit.Test`, `org.junit.Assert.*`, run via `org.junit.runner.JUnitCore`). These jars are NOT vendored in the client repo or in `niagara-tools`, and NOT in the Honeywell N4 install — they live in the **Gradle cache** (pulled once by the `niagara-jacoco`/test wiring). Verified present 2026-09-04 (junit 384 KB, hamcrest 45 KB, dated 2026-08-31):

```
JUnit    : ~/.gradle/caches/modules-2/files-2.1/junit/junit/4.13.2/8ac9e16d933b6fb43bc7f576336b8f4d7eb5ba12/junit-4.13.2.jar
Hamcrest : ~/.gradle/caches/modules-2/files-2.1/org.hamcrest/hamcrest-core/1.3/42a25dc3219429f0e5d060061f71acb49bf010a0/hamcrest-core-1.3.jar
```

Second copy (same versions, works identically) ships inside the Gradle distribution:
```
~/.gradle/wrapper/dists/gradle-7.6-bin/9l9tetv7ltxvx3i8an4pb86ye/gradle-7.6/lib/junit-4.13.2.jar
~/.gradle/wrapper/dists/gradle-7.6-bin/9l9tetv7ltxvx3i8an4pb86ye/gradle-7.6/lib/hamcrest-core-1.3.jar
```

The SHA-1 dir names are content hashes and are stable for a given version; if a path ever misses, resolve it dynamically (delta #2 below) rather than hardcoding — the version (4.13.2 / 1.3) is the durable fact, the hash path is incidental.

## Ready command (CompPan example)

The pure class and its test share `package com.angeles.CompPan` (the test is same-package to reach the package-private `final class CompressorControl`). From `Compresores/CompPan/CompPan-rt/`:

```bash
JU=$(find ~/.gradle -name 'junit-4.13.2.jar' | head -1)
HC=$(find ~/.gradle -name 'hamcrest-core-1.3.jar' | head -1)
javac -source 8 -target 8 -cp "$JU" -d out \
  src/com/angeles/CompPan/CompressorControl.java \
  srcTest/test/com/angeles/CompPan/CompressorControlTest.java
java -cp "out:$JU:$HC" org.junit.runner.JUnitCore com.angeles.CompPan.CompressorControlTest
```

(ColdRoomPan is identical: `ColdRoomControl` + `ColdRoomControlTest`, package `com.angeles.ColdRoomPan`.)

## Proposed deltas (propose-never-apply — for human review)

1. **Enrich `build-verify.md:96`** with the concrete cached paths above (or the `find ~/.gradle …` resolver), so the "`<junit>`/`<hamcrest>`" placeholders are copy-paste-ready.
2. **Add a tiny `toolbelt/run-pure-test.sh`** (sketch below) — resolve the jars from the Gradle cache, compile the pure class + its test, run `JUnitCore`. It removes the per-session hunt entirely:
   ```bash
   # run-pure-test.sh <module-rt-dir> <pkg> <PureClass> <TestClass>
   JU=$(find ~/.gradle -name 'junit-4.13.2.jar' | head -1)
   HC=$(find ~/.gradle -name 'hamcrest-core-1.3.jar' | head -1)
   [ -z "$JU" ] && { echo "junit-4.13.2 not in ~/.gradle cache — run a gradle build once to fetch it"; exit 1; }
   pkgpath=${2//.//}
   javac -source 8 -target 8 -cp "$JU" -d "$1/out" \
     "$1/src/$pkgpath/$3.java" "$1/srcTest/test/$pkgpath/$4.java"
   java -cp "$1/out:$JU:$HC" org.junit.runner.JUnitCore "$2.$4"
   ```
3. If the cache is ever empty (fresh machine), a single `./gradlew :<module>-rt:compileTestJava` (or any build touching the test wiring) re-fetches junit-4.13.2 + hamcrest into `~/.gradle/caches`.

No file above is edited by this retro.

## Evidence
- `build-verify.md:96` (the standalone recipe), qa-stack retro `2026-09-03-qa-stack-pure-tests-and-defrost-untested-gap.md` ("`junit-4.13.2.jar` cacheado").
- Test header `CompressorControlTest.java:6-17` (`import org.junit.Test`; "run this standalone: javac + junit-4.13.2.jar").
- Cached jars verified present 2026-09-04 at the paths above (junit 384581 B, hamcrest 45024 B).
- Package confirmed: both `CompressorControl.java` and `CompressorControlTest.java` = `package com.angeles.CompPan`.
