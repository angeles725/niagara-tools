# Build & verify — Java 8 + slotomatic (the ONLY valid build)

A `gradle :jar` with the default JDK is NOT a valid N4 build: it can use the wrong Java (major-65 bytecode Niagara's Java-8 runtime rejects) and it skips slotomatic, so annotation bugs never surface. Chihuahua's `deploy.sh` uses `JAVA8=/usr/lib/jvm/java-8-openjdk-amd64`.

## Primary: `ng-deploy.sh` (the established wrapper)

Use `niagara-tools/scripts/ng-deploy.sh` — it already does backup → build → (slotomatic) → copy-to-station → verify, and knows the flags:
```
scripts/ng-deploy.sh --mode A --with-slotomatic        # rt+ux, regenerate from annotations, deploy+verify
scripts/ng-deploy.sh --strict-slotomatic               # ABORTS (exit 15) if annotation changes but slotomatic wasn't run
scripts/ng-deploy.sh --no-deploy --with-slotomatic     # build-only (jars stay in build/libs)
```
Set env in `.env.local` (or `--env-file`): `MODULE_NAME, GRADLEW_PATH, NIAGARA_HOME, NIAGARA_USER_HOME, JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64, STATION_MODULES_DIR`.
`--strict-slotomatic` is the guard that catches the annotation-vs-generated bug automatically — prefer it.

## Fallback: raw gradle (quick WSL iteration, no deploy)
```
cd <module-root>
J8=/usr/lib/jvm/java-8-openjdk-amd64            # confirm: ls /usr/lib/jvm
./gradlew :MOD-rt:clean :MOD-ux:clean \
          :MOD-rt:slotomatic :MOD-ux:slotomatic \
          :MOD-rt:jar :MOD-ux:jar \
  -Pniagara_home=<niagara_home> \
  -Porg.gradle.java.installations.paths=$J8
```
- `slotomatic` = "Run Slot-o-matic on a module" — regenerates the AUTO region from `@NiagaraProperty` annotations. Run it so a regen can't reintroduce a fixed bug.
- On WSL, `gradle.properties` may pin a Windows JDK path (`C:\Program Files\Zulu\zulu-8`) — that breaks WSL (`Illegal character ... C:\`). Override with `-Porg.gradle.java.installations.paths=$J8`. On Windows the pinned Zulu 8 is correct.

## niagara_home on WSL
The real install path in `gradle.properties` is a Windows path. Either point `-Pniagara_home` at `/mnt/c/...` OR at a **writable mirror** (symlinks to all real module jars + bin/ext + etc, but a writable `modules/`) so the plugin's auto-copy of the built jar succeeds — needed when a `-ux` project depends on `-rt` (the rt→ux edge + a station-locked jar otherwise blocks the build).

## Verify
```
# bytecode major 52 = Java 8 (65 = Java 21, wrong)
unzip -p MOD-rt/build/libs/MOD-rt.jar com/.../SomeClass.class | od -An -t d1 -j6 -N2   # 2nd number = 52
# signed
unzip -l .../MOD-ux.jar | grep NIAGARA4.SF
# no raw-double facet anywhere
grep -rnE 'BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]' <module> --include='*.java'    # must be empty
```

## Deploy (station)
Sign (the build auto-signs), then: stop station → replace the jars in `<niagara_home>/modules/` → start. The jar is locked while the station runs. `niagaraTest` needs the native `bin/test` + a developer license — skip it; run pure-Java JUnit instead.
