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

---

## Knowledge base index

Topic files with full detail, examples, and back-links to this index:

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
