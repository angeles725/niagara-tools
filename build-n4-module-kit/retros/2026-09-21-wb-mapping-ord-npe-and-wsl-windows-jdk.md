<!-- review-status: folded -->
# 2026-09-21 · kit · wb-mapping-ord-npe-and-wsl-windows-jdk

**Session**: Live bidirectional-gateway testing of the Apillm client module (Cliente/LLM/Apillm). The operator tried to add a new DB→N4 key mapping (`demo2`) through the `-wb` "Add Mapping" wizard and hit a `NullPointerException`; fixing it surfaced a runtime ORD gotcha, and rebuilding surfaced a WSL Java-toolchain gotcha.
**Delta count**: 2

## What happened
Two distinct, reproducible gotchas, both new as concrete symptoms:

1. **`getSlotPathOrd()` returns null → `setTargetOrd(null)` → `BComplex.set` NPE.** The wizard creates a writable point, adds it to the target folder, then reads `point.getSlotPathOrd()` to build the mapping's `targetOrd`. In the manager-view context that call returned `null` (BComponent.getSlotPathOrd returns null when `getSlotPath()==null`, i.e. the component is not mounted/resolvable at that instant). The code passed the null straight to `BApillmImportMap.setTargetOrd(BOrd)`, which does `set(targetOrd, v, null)`; `BComplex.set` dereferences `value.getSlotMap()` and NPEs on the null value. This is a RUNTIME NPE, distinct from the already-documented null-ORD *picker* gotcha (`wb-widgets.md §Field editors`, the file-space `BOrdFE` default) — that one is about a chooser opening `C:\`; this one is about crashing `set()` with a null ORD.

2. **On WSL, Gradle auto-detects the WINDOWS Zulu JDK and the build dies with a cryptic URI error.** A direct `./gradlew :Apillm-wb:jar` (bypassing the kit `build.sh`) failed in 2 s with `java.net.URISyntaxException: Illegal character in opaque part at index 2: C:\Program Files\Zulu\zulu-8`. Gradle's toolchain auto-detection found a Windows JDK 8 install whose backslash/colon path is not a legal toolchain URI on Linux. `build.sh` never hits this because it passes `-Porg.gradle.java.installations.paths=$J8` explicitly; a developer running `gradlew` by hand does not.

## Evidence
- NPE chain: `BApillmImportMap.setTargetOrd:90` → `BComplex.set:850-851` = `slotMap.set(property, oldValue, oldMap, value, value.getSlotMap(), context)` — `value.getSlotMap()` NPEs when `value==null`. `[ev: code javax/baja/sys/BComplex.java:850-851]`
- Null source: `BComponent.getSlotPathOrd()` = `{ SlotPath p = getSlotPath(); if (p==null) return null; return BOrd.make(p); }` — returns null for an unmounted/unresolved component. `[ev: code javax/baja/sys/BComponent.java:630-635]`
- Fix applied (Apillm-wb `BApillmImporterManager.java`): (a) `targetOrd` fallback — `if (targetOrd==null || targetOrd.isNull()) { BOrd base = folder.getSlotPathOrd(); if (base null/isNull) base = folderOrd; targetOrd = BOrd.make(base.toString()+"/"+slotName); }`; (b) `addMapping` hardened — `if (targetOrd==null||isNull) return;` + MOUNT first (`importer.add(name, m)`) THEN `m.setTargetOrd(targetOrd)`. Built green in 8 s, signed (`NIAGARA4.SF/RSA`), auto-placed into `modules/`. `[ev: Apillm fix 2026-09-21]`
- Build error verbatim: `java.net.URISyntaxException: Illegal character in opaque part at index 2: C:\Program Files\Zulu\zulu-8`; fixed by `-Porg.gradle.java.installations.paths=/usr/lib/jvm/java-8-openjdk-amd64 -Porg.gradle.java.installations.auto-detect=false`. `[ev: gradle run 2026-09-21]`

### Verified ALREADY COVERED (not re-proposed)
| Topic | Covered by |
|---|---|
| JDK-pinning FIX (`installations.paths` + `auto-detect=false`, "wrong JDK major is NOT a build" hazard) | `types/structure.md §L10` (:71-101) — Δ2 only adds the WSL Windows-JDK SYMPTOM that points here |
| null-ORD *picker* gotcha (file-space `BOrdFE`, `targetType` facet) | `types/wb-widgets.md §Field editors` (PD-FE1) — different failure (chooser, not `set()` NPE) |
| `set()`/`invoke()` null-Context write hazard | `types/security.md §1.1`, `lint-null-context-write.sh` — that is null CONTEXT, not null VALUE |

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | NEW gotcha: **a `-wb` manager that creates a point then maps `point.getSlotPathOrd()` can get a NULL ORD, and passing it to a `BOrd`-property `set()` NPEs in `BComplex.set` (`value.getSlotMap()`).** Rule: (a) never pass a possibly-null ORD to `set()` — guard `if (ord==null || ord.isNull()) …`; (b) when `getSlotPathOrd()` is null, build the ORD from the resolved parent-folder ORD + the new slot name; (c) mount the mapping component (`parent.add(name, m)`) BEFORE setting a `BOrd` property that carries a `TARGET_TYPE` facet. Sibling to the existing null-ORD *picker* gotcha (cross-link), but this is a runtime `set()` NPE. Optional lint candidate `lint-set-null-ord`: a `setXxxOrd(getSlotPathOrd())` / `set(<ordProp>, <expr that can be null>)` with no null guard. Check it does not overlap `lint-null-context-write`. | `types/issues-and-gotchas.md` (new §A/B gotcha) + `types/wb-widgets.md §create-point-from-wb` cross-link | `[ev: code BComplex.java:850-851, BComponent.java:630-635; Apillm fix]` |
| Δ2 | Add a recognizable SYMPTOM entry under **§D — Build & tooling**: on WSL, `gradlew` (run directly, not via `build.sh`) can auto-detect a **Windows** JDK and die with `java.net.URISyntaxException: Illegal character in opaque part at index 2: C:\…`. Cause = Gradle toolchain auto-detect picked a Windows JDK path. Fix = pass `-Porg.gradle.java.installations.paths=<linux JDK8> -Porg.gradle.java.installations.auto-detect=false` (or use `build.sh`, which already does). Points to the existing `structure.md §L10` pinning recipe. | `types/issues-and-gotchas.md §D` (new D-entry) | `[ev: gradle URISyntaxException; structure.md §L10]` |

## Lessons
- `getSlotPathOrd()` is NOT guaranteed non-null even right after `parent.add(slotName, point)` in a view/editor context — a manager wizard that maps a freshly-created point's ORD must guard it and have a folder-ORD-based fallback.
- The kit already had the JDK-pinning *rule* but not the *symptom*; a cryptic `URISyntaxException` on a Windows path is not obviously "pin your JDK", so the symptom→fix link is worth one line.
- Building with `build.sh` (which pins the JDK path) vs raw `gradlew` (which does not) is exactly why "before it took 5 s": the fast path is `gradlew` with the pin flag OR `build.sh`; raw `gradlew` without the pin fails on WSL.

---
**Status**: FOLDED — types/issues-and-gotchas.md §H3 (Δ1) + §D5 (Δ2), types/wb-widgets.md cross-link
(Δ1), INDEX row flipped. Optional `lint-set-null-ord` candidate DEFERRED (see BUILD-STATE.md kit
open_issue) — not implemented in this fold.
