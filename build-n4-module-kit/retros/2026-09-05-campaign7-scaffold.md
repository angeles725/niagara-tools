<!-- review-status: pending -->
# Retro: scaffold-module.sh + fixtures/MinimalPan (Campaign 7 PR4)

**Date**: 2026-09-05 | **PR**: feat/c7-scaffold | **SDD change**: build-n4-module-campaign7

---

## Summary

PR4 lands the fixture-driven module scaffolder (D1). The bundled
`fixtures/MinimalPan/` tree is the B793-verified, gate-green B790 skeleton in
**pre-slotomatic** state — no `//region … //endregion` AUTO block. `build.sh`
injects it at build time (`:slotomatic` before `:jar`). `scaffold-module.sh`
copies the fixture and substitutes MinimalPan/Angeles/mp tokens in paths and
text; binary files (`gradlew`, `gradle-wrapper.jar`) are copied verbatim.

TC1–TC3 and TC-K8 passed on the QA RED pins. TC4 round-trip ran locally:
scaffold → preflight (PASS 3/3) → build.sh (slotomatic + jar, BUILD SUCCESSFUL) →
verify-module (7 PASS / 0 FAIL / 1 SKIP) → lint-timers (PASS). Four named
mutations confirmed (see PR body).

---

## Key findings (proposed kit additions)

**K15 — pre-slotomatic convention: NO `//region` block in the source**
The correct pre-slotomatic state for a new `BComponent` class is annotations
present and NO `//region /*+ … BEGIN BAJA AUTO … +*/` block at all. Slotomatic
inserts the entire block on its first run. An empty `//region … //endregion`
skeleton causes a slotomatic error: "Found multiple metadata blocks" (the
`/*+` on both region and endregion lines looks like two distinct blocks).
[ev: B793 §793.3 C1 corrected by TC4 wall — see §793.4]

**K16 — gradle.properties machine coupling must be cleaned for fixtures**
A committed `gradle.properties` with a live `niagara_home=C:\…` path creates
machine-coupling and breaks the K8 (no-HOME) promise. The fixture drops both
`niagara_home=` and `niagara_user_home=` lines, leaving only commented
placeholders. `build.sh` passes `-Pniagara_home=…` on the command line, making
the file purely declarative.

**K17 — gradle-wrapper.jar ≤ 100 KB stays bundled**
`gradle-wrapper.jar` is 60 KB (Gradle 7.6). The design threshold of 100 KB is
not exceeded, so the jar ships in the fixture. Copying it verbatim avoids sed
corrupting binary bytes (the binary exclusion pattern `*.jar | */gradlew |
*/gradlew.bat` in the copy loop).

**K18 — TC3 byte-equality pins the fixture as the module contract**
Running `scaffold-module.sh MinimalPan <out>` with default options produces a
byte-identical copy of `fixtures/MinimalPan/`. This makes the fixture the
single source of truth (D1): reviewer diffs show Java/Kotlin, not heredoc bash;
TC3 catches regressions in the copy-rename-substitute logic.

---

## Named mutations run (PR4 evidence)

| Mutation | Test flipped | Reverted |
|---|---|---|
| Skip `.gradle.kts` in copy loop (script change) | TC3+TC-K8 FAIL | ✓ |
| Emit `module.xml` path instead of `module-include.xml` (script path remap) | TC3+TC-K8 FAIL | ✓ |
| Use `$HOME` in FIXTURE_ROOT instead of BASH_SOURCE-relative path | TC-K8 FAIL | ✓ |
| Remove `ticket.cancel()` from `stopped()` (on built file) | lint-timers FAIL | ✓ |

---

## TC4 local round-trip output

```
=== Step 1: scaffold ===
scaffold-module: emitted MinimalPan -> /tmp/tmp.tE1CEFwOxn/MinimalPan
Scaffold exit: 0

=== Step 2: preflight ===
PASS  jdk8          JDK 8 found: /usr/lib/jvm/java-1.8.0-openjdk-amd64/
PASS  plugin-pin    plugin 7.6.17 found in niagara_home/etc/m2
PASS  jar-lock      no locked jars under /mnt/c/Honeywell/OptimizerSupervisor-N4.14.0.162/modules/
Preflight exit: 0

=== Step 3: build.sh ===
==> build (Java 8 + slotomatic): :MinimalPan-rt:clean :MinimalPan-rt:slotomatic :MinimalPan-rt:jar
> Task :MinimalPan-rt:slotomatic
> Task :MinimalPan-rt:compileJava
> Task :MinimalPan-rt:jar
BUILD SUCCESSFUL in 5s
==> verify gate (verify-module.sh):
PASS  bytecode   2 classes, all major 52
PASS  signed     META-INF/NIAGARA4.SF present
PASS  types      1 declared types resolve to classes
PASS  baja       stamped baja 4.14 <= target 4.14
SKIP  stored     no --stored
PASS  typecount  jar declares 1 types == module-include.xml
PASS  facets     no raw-number MIN/MAX facet
PASS  palette    module.palette has 1 component entries
verify-module: 7 passed, 0 failed, 1 skipped, 0 warned -> ALL PASS
build.sh exit: 0

=== Step 4: verify-module.sh --src ===
verify-module: 4 passed, 0 failed, 4 skipped, 0 warned -> ALL PASS

=== Step 5: lint-timers.sh ===
PASS  timer-ticket  BMinimalPan.java: timer cancelled in stopped()
lint-timers exit: 0
```

---

## CI evidence

TC4 is SKIP-gated in CI via `[ -n "${NIAGARA_HOME:-}" ] || skip "..."`. The
`run-pure-test.bats` strict SKIP guard is deliberately not applied to
`scaffold-module.bats` (D9). The CI scaffold-diff step scaffolds MinimalPan
into `$RUNNER_TEMP` and `diff -r` vs the fixture — deterministic, exit 0.

---

## Proposed delta (propose, never apply)

Add K15, K16, K17, K18 to `METHODOLOGY.md` §Kit-maintenance after K14.
