# GOTCHAS — Cross-project anti-patterns and knowledge base

## Anti-patterns table

Common mistakes across Niagara N4 module projects. Each entry links to
the KB topic with full detail and workaround.

| Anti-pattern | Symptom | Fix | See |
|---|---|---|---|
| `cd .../chihuahua/chihuahua/` (extra dir) | `gradlew: command not found` despite file existing | Use top-level module path: `cd .../chihuahua` | [wsl-build-gotchas.md](knowledge-base/wsl-build-gotchas.md) |
| Build without `clean` after Java edit | `writeModuleXml UP-TO-DATE`, station doesn't see new types | Always `:clean` when changing `.java` files | [wsl-build-gotchas.md](knowledge-base/wsl-build-gotchas.md) |
| Missing `-Pniagara_home` / `-Pniagara_user_home` / `-Porg.gradle.java.installations.paths` in WSL | `BUILD FAILED — Could not resolve C:\Niagara\...` or `URISyntaxException` | Always pass the 3 mandatory `-P` overrides | [wsl-build-gotchas.md](knowledge-base/wsl-build-gotchas.md) |
| `gradlew` no exec bit after NTFS clone | `permission denied: ./gradlew` | `chmod +x gradlew` once post-clone | [wsl-build-gotchas.md](knowledge-base/wsl-build-gotchas.md) |
| Deploy jar without verifying `<type` count | Jar deployed, station doesn't load module types | `unzip -p jar META-INF/module.xml \| grep -c "<type"` before + after | (this index) |
| `BAlarmService.ackAlarm(rec)` where rec came from BQL cursor | HTTP 200, `ackedCount > 0` but ack does NOT persist (N4.14) | Server collects UUIDs only; BajaScript `baja.Ord.make('alarm:').get().then(svc => svc.ackAlarms({ids}))` | [bql-gotchas.md](knowledge-base/bql-gotchas.md) |
| Hand-editing AUTO GENERATED CODE region | Next slotomatic run silently overwrites changes | Never hand-edit; use coordinated-edit pattern for removals | [slotomatic.md](knowledge-base/slotomatic.md) |
| Java change deployed without station restart | New class never loaded; feature silently non-functional | Restart station (classloader re-init required) | [hot-reload-rules.md](knowledge-base/hot-reload-rules.md) |
| JS/CSS change without bumping `?v=N` cache-buster | Browser serves cached old version (HTTP 304) | Bump `?v=N` in `index.html`; `ng-deploy.sh` checks this when `BUILD_ID` is set | [hot-reload-rules.md](knowledge-base/hot-reload-rules.md) |
| `BQL WHERE ackState = 'unacked' OR ackState = 'ackPending' AND timestamp > $T` without parentheses | Returns ALL unacked regardless of timestamp (AND binds tighter than OR) | Parenthesize OR groups; place time filter last in WHERE clause | [bql-gotchas.md](knowledge-base/bql-gotchas.md) |
| Deploy with stale slotomatic (annotation changed but slotomatic not run) | Station loads module but `@NiagaraProperty`/`@NiagaraType` slot is missing or mismatched; may cause silent BComponent errors | Use `--with-slotomatic` flag or run `:MODULE-rt:slotomatic` before deploy; or use `SLOTOMATIC_DETECTION=strict` to abort on detection | [slotomatic.md](knowledge-base/slotomatic.md#card-4-ng-deploysh-integration-v030) |
| Re-signing a gradle-built jar in Workbench | `JarFileSigner … ZipException: invalid entry compressed size (expected N but got M)`; clean rebuild recurs, local `jarsigner -verify` says "verified" (false negative) | Repackage STORED before the re-sign: `build-n4-module-kit/toolbelt/stored-repack.sh in.jar out.jar` (manifest first, SF/RSA next, `zip -0`); check with `verify-module.sh --stored out.jar` | [build-verify.md](../build-n4-module-kit/build-verify.md) |
| Building with `-Pniagara_home=<live install>` while the station runs | `Unable to delete file …/modules/<module>.jar` from `clean`/`jar` (the station locks the jar); temptation to stop a live supervisor | Build against a mirror with a writable `modules/`: `build-n4-module-kit/toolbelt/mirror-niagara-home.sh <install> <mirror> <module>.jar` (refuses the real install or any non-mirror dir, exit 20) | [build-verify.md](../build-n4-module-kit/build-verify.md) |
| Deploying a jar that only "built OK" | Station rejects or boot-loops: major-65 bytecode (default JDK 21), unsigned jar, `module.xml` type with no class, baja `vendorVersion` newer than the station | Run the gate on every jar before it leaves `build/libs`: `build-n4-module-kit/toolbelt/verify-module.sh --src <module-dir> --target-version 4.14 *.jar` (exit 0 only when every check passes); `build.sh` runs it automatically | [build-verify.md](../build-n4-module-kit/build-verify.md) |

---

## Knowledge base index

Topic files with full detail, examples, and back-links to this index:

- [build-n4-module-kit build & verify](../build-n4-module-kit/build-verify.md) —
  The verify gate (`toolbelt/verify-module.sh`), the recommended WSL build (`toolbelt/build.sh`),
  the running-station mirror (`toolbelt/mirror-niagara-home.sh`) and the STORED repack for the
  Workbench re-sign path (`toolbelt/stored-repack.sh`). Suites: `tests/{verify-module,build-sh,mirror-niagara-home,stored-repack}.bats`.

- [BQL gotchas in Niagara N4.14](knowledge-base/bql-gotchas.md) —
  3 confirmed N4.14 bugs (`ackState` transitory, `sourceState` frozen snapshot,
  OR precedence) + the only proven persistent-ack path (BajaScript).

- [WSL build gotchas](knowledge-base/wsl-build-gotchas.md) —
  Mandatory `-P` overrides, `gradlew` exec bit post-NTFS-clone,
  path-depth confusion, slotomatic-in-WSL myth busted.

- [Hot-reload rules](knowledge-base/hot-reload-rules.md) —
  Decision table: Java changes require station restart; JS/CSS/HTML assets
  are hot-served (browser hard-reload only). Why `*-ux` subproject is ambiguous.

- [Slot-O-Matic patterns](knowledge-base/slotomatic.md) —
  When to run slotomatic, AUTO GENERATED CODE region rules (and the one
  allowed exception), 4-step coordinated-edit recipe for safe slot removal.
