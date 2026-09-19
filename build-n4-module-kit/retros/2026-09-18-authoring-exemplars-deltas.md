<!-- review-status: pending -->
# 2026-09-18 · kit · authoring-exemplars-deltas

**Session**: corpus-mining for build-n4-module (operator: mine B772–B785 + B817 for concrete kit improvements — types/, toolbelt/, build-verify.md, BUILD-LOOP.md; NOT corpus-index.md pointers, which have a same-day retro).
**Delta count**: 8

## What happened

Mined the niagara-research corpus for B772–B785 (12 authoring dimensions from
first-party Tridium/Honeywell modules) and B817 (module STRUCTURE STANDARD +
L1–L11 conformance checklist) against the current kit state, looking for CONTENT
gaps in `types/`, `toolbelt/`, `build-verify.md`, and `BUILD-LOOP.md`.

**Already folded — skipped:**

- B817 L1–L11 → `lint-structure.sh` (now L1–L13) + `types/structure.md` fully documents the checklist.
- B787 timer-ticket cancel → `lint-timers.sh` `timer-ticket` check.
- B788 palette/lexicon non-empty → `verify-module.sh` `palette` check + `lint-structure.sh` L4/L5.
- B789 container legality / OMV5-1 poll-vs-subscribe → advisory only, no static lint candidate.
- B772–B776, B778–B785 authoring dimensions → all folded into `types/logic-authoring.md`.
- B812 liveness watchdog → `types/logic.md` §Liveness watchdog recipe.
- B960 build scaffold (defaultModuleVersion, gradle.properties.EXAMPLE) → already in `build-verify.md` + `types/structure.md` L10 doctrine.

**Gaps found — 8 proposed deltas:**

Four corpus blocks are NOT yet folded into kit content:

1. **B819** (zero-demand/idle-state doctrine) — 6-point staging/control doctrine whose kit home
   `types/logic.md` is explicit in B819 §819.5 but the section does not exist yet.
2. **B958** (link-lifecycle callbacks + BTestNg moduleTest) — `doCheckLink`/`added`/`removed`
   recipe + INDIRECT-link rule; BTestNg wiring details beyond what BUILD-LOOP.md §4.b has today.
3. **B777** (security-module SPI) — cited as `[ev: corpus B777]` in the N4-extension-idiom line of
   `types/logic-authoring.md` but the module-permissions.xml inlining mechanic and
   `@AgentOn "baja:AuthenticationScheme"` registration are never spelled out.
4. **B817 §817.6** OEM signing convention — the OEM `SERVER1.SF/RSA` vs Tridium `NIAGARA4.SF/RSA`
   distinction is absent from `build-verify.md` and `verify-module.sh` has an undocumented
   failure mode when run against a third-party Honeywell jar.

## Evidence

- B819 §819.4: six-point zero-demand/idle-state doctrine; §819.5 explicitly says "kit implication: new `types/logic.md` section" `[ev: corpus B819 §819.4–819.5]`
- B819 §819.3: NaN setpoint guard anti-pattern — unguarded `suctionSetpoint` causes silent modulation freeze while demand gate still turns off the rack `[ev: corpus B819 §819.3]`
- B958 §958.1: `doCheckLink()` returns `LinkCheck.makeInvalid(reason)` / `makeValid()` to validate a proposed link before it is established `[ev: corpus B958 §958.1]`
- B958 §958.2: INDIRECT link rule — when creating a reciprocal link inside `added()`, use `new BLink(sessionOrd, ..., true)` (indirect = ORD-resolved) so it survives station restart ordering `[ev: corpus B958 §958.2]`
- B958 §958.3–958.4: BTestNg moduleTest full wiring: extend `BTestNg`, `@BeforeClass createTestStation()` / `@AfterClass releaseStation()`, separate `moduleTest-include.xml` for test-only types, `moduleTestImplementation("Tridium:test-wb")` dep spelling `[ev: corpus B958 §958.3–958.4]`
- B961 §961.3 explicitly proposes: (a) WARN when a module has production types but no `moduleTest-include.xml`, (b) link-lifecycle in `types/logic-authoring.md`, (c) moduleTest wiring in `BUILD-LOOP.md §4.b` `[ev: corpus B961 §961.3]`
- B777 §777.2–777.4: security-module SPI pattern — `BAbstractService → BAuthenticationScheme` subclass; permissions go in a `module-permissions.xml` that the Gradle plugin INLINES into the jar `module.xml` (NOT a separate file shipped); `@AgentOn "baja:AuthenticationScheme"` registration; jar-signed `NIAGARA4.RSA/SF` mandatory `[ev: corpus B777 §777.2–777.4]`
- B817 §817.6: Honeywell/OEM modules use `SERVER1.SF`/`SERVER1.RSA`; `verify-module.sh` `check_signed()` only recognises `NIAGARA4.SF` — expected FAIL on third-party OEM jars if ever run against them `[ev: corpus B817 §817.6]`
- `types/logic-authoring.md` line 6: B777 cited as `[ev: corpus B778/B782/B785/B777]` in the N4-extension-idiom intro but no §Security module section follows `[ev: kit types/logic-authoring.md §Idiom]`
- `BUILD-LOOP.md §4.b`: BTestNg mention exists with `createTestStation()` scaffold and `testImplementation(project(":test-wb"))` — missing `moduleTest-include.xml` registration and exact `moduleTestImplementation` dep spelling `[ev: kit BUILD-LOOP.md §4.b]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add `## Zero-demand / idle state doctrine` section: (1) demand is a first-class gate — never skip it on a valid setpoint; (2) NaN/invalid setpoint ≠ demand, guard with `Double.isFinite` or `getStatus().isValid()`; (3) every staged process declares an explicit idle state; (4) expose "why running" surface (a read-only slot or log); (5) HOA OFF lockout dominates — respect it even when demand is non-zero; (6) minOn/stageDelay guard applies in BOTH staging directions | `types/logic.md` new §Zero-demand / idle state | `[ev: corpus B819 §819.4]` |
| Δ2 | Within Δ1 section, add NaN-guard anti-pattern: "The NaN setpoint hazard (B819 §819.3): an unguarded numeric setpoint (`suctionSetpoint`, `tempSetpoint`) causes silent modulation freeze — the staging gate turns the rack off, but the PID/band logic silently continues emitting control signals because demand was never NaN-gated. Always guard: `if (!Double.isFinite(setpoint)) { → idle }` before the staging entry." | `types/logic.md` §Zero-demand / idle state sub-bullet | `[ev: corpus B819 §819.3]` |
| Δ3 | Add `## Link lifecycle callbacks` section with three subsections: (a) `doCheckLink(Context, BLink)` — validate before accept, return `LinkCheck.makeInvalid(reason)` to reject, `makeValid()` to accept; (b) `added(BLink)` / `removed(BLink)` — react to link establishment/removal; (c) INDIRECT link rule: when creating a reciprocal link inside `added()`, always use `new BLink(sessionOrd, targetOrd, true)` (`indirect = true` = ORD-resolved) so restart ordering cannot leave a dangling direct handle | `types/logic-authoring.md` new §Link lifecycle callbacks | `[ev: corpus B958 §958.1–958.2]` |
| Δ4 | Expand §4.b BTestNg recipe with the missing wiring details: (a) test class extends `BTestNg`, annotate `@Module("yourmodule")` on the class; (b) `@BeforeClass` calls `createTestStation()` via `TestStationHandler` try-with-resources; (c) register test types in a SEPARATE `moduleTest-include.xml` (never mix into production `module-include.xml`); (d) gradle dep is `moduleTestImplementation("Tridium:test-wb")` (NOT `testImplementation`); (e) WSL-unsafe: needs the NRE live station, cannot run in pure WSL JUnit | `BUILD-LOOP.md §4.b` expansion | `[ev: corpus B958 §958.3–958.4]` |
| Δ5 | Add new WARN check `moduletest` to `verify-module.sh` (advisory, never FAIL): after the `palette` check, scan all `module-include.xml` files in the checked jar tree; for each jar that has `<type …/>` entries (production types registered) but whose source tree has no `moduleTest-include.xml`, emit `WARN  moduletest  <jar>  production types with no moduleTest-include.xml`. Skip jars with zero declared types (skeleton profiles). | `toolbelt/verify-module.sh` new check after `palette` | `[ev: corpus B961 §961.3]` |
| Δ6 | Add `## Security module skeleton` subsection under §Author-side SPIs: extend `BAbstractService` → implement `BAuthenticationScheme`; place permissions in a `module-permissions.xml` in the jar source (Gradle plugin INLINES it into the generated `module.xml` — do NOT author a `META-INF/module.xml` manually); register via `@AgentOn("baja:AuthenticationScheme")` in `module-include.xml`; the jar MUST be signed with `NIAGARA4.RSA`/`NIAGARA4.SF` (same signing path as any production jar). | `types/logic-authoring.md §Author-side SPIs` new subsection | `[ev: corpus B777 §777.2–777.4]` |
| Δ7 | Add OEM signing note to the Signing section: "Honeywell/OEM modules use `SERVER1.SF`/`SERVER1.RSA` instead of `NIAGARA4.*`. A jar signed this way will FAIL the `signed` check in `verify-module.sh` — this is expected and correct when verifying a third-party OEM jar, not ours. Our modules must always use `NIAGARA4.*` regardless of the station's OEM tier." | `build-verify.md §Signing` | `[ev: corpus B817 §817.6]` |
| Δ8 | Add `types/moduleTest.md` (new file): the full moduleTest authoring reference — when to write a station test vs a pure JUnit test, the BTestNg lifecycle (createTestStation, TestStationHandler, @BeforeClass/@AfterClass), moduleTest-include.xml schema, gradle wiring (`moduleTestImplementation`), WSL-safety caveat, one minimal worked example (one @Test calling a component's execute method on a live station). | `types/moduleTest.md` new file | `[ev: corpus B958 §958.3–958.4, B961 §961.3]` |

## Lessons

- **B819 was actionable and unmapped.** The corpus block explicitly named `types/logic.md` as the kit target in §819.5; checking the target file first would have found the gap in one read.
- **B777 cited-but-not-expanded is a pattern to watch.** A corpus block can appear in a reference list (`[ev: corpus B777]`) in a logic-authoring intro without its content being folded. Future mining should grep `[ev: corpus BN]` lines and check whether BN has a corresponding kit section.
- **OEM signing is a silent footgun.** `verify-module.sh check_signed()` looks for `NIAGARA4.SF`; a Honeywell jar signed with `SERVER1.SF` causes a spurious FAIL with no explanation. Documenting this before it is encountered is low-cost insurance.
- **moduleTest-include.xml is a separate authoring concern from module-include.xml.** The kit `BUILD-LOOP.md §4.b` mentioned BTestNg but omitted the separate registration file and the exact dep key — exactly the detail a first-timer misses. A dedicated `types/moduleTest.md` removes the lookup trip entirely.

---

**Status**: PENDING — INDEX row appended:
`| 2026-09-18-authoring-exemplars-deltas.md | kit | 2026-09-18 | pending | 8 |`
