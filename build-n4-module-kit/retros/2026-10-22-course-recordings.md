<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 8 -->
# 2026-10-22 · kit · course-recordings

**Session**: niagara-research quality round 8 (`odd/tasks/quality-round8.md`, task T9f), branch
`docs/r8-t9f-course-rules` from origin/main c9d1215; one commit per topic group
**Delta count**: 8

## What happened
The four recordings of the N4 Developer Certification Course (Day 1-4) carry spoken lessons the slide decks and
labs do not. Research blocks B1416-B1419 took them in and checked each one against code, the official guides or
our own hosts. This retro folds the build, packaging, signing, Slot-o-Matic, lexicon, XSS, testing, service and
N5-readiness rows into the kit as short rules that cite the block instead of copying its evidence. Where a block
corrected the instructor (the default store is `niagara.signing.jceks`, not `.jks`; `AlarmService` registers only
`{TYPE}`), the kit carries the corrected fact. One new check: `toolbelt/lint-service-types.sh`.

## Evidence
- Lessons and their verification: corpus B1416 (Day 1), B1417 (Day 2), B1418 (Day 3), B1419 (Day 4); row ids from
  the T10 digest (D1-*, D2-*, D3-*, D4-*) map to the block sections cited in each rule `[ev: corpus B1416]` `[ev: corpus B1419]`
- Kit grep before the change (`types/`, `METHODOLOGY.md`, `BUILD-LOOP.md`, `build-verify.md`): `getHtmlSafeText`,
  `Niagara4Modules`, `Werror` → 0 hits; `allowDefaultProfile` only in `build-verify.md` (sample preflight) and the
  pending retro n4-course-build-deps-permissions-signing Δ4; `N5` only in `BUILD-STATE.md` and `odd/` notes
- Kit CORRECTED: `types/structure.md` § Palette assembly templates example used Java class names
  (`t="com.angeles.coldroomrt.BColdRoomPan"`); real palettes use `m="alias=module"` + `t="alias:Type"` without the `B`
  (fixtures/MinimalPan, CompPan, DashboardFord) `[ev: corpus B1418 §1418.5 e]`
- `lint-service-types.sh`: RED 0/9 before the script existed, GREEN 11/11 (ST1-ST9, ST-awkfail, ST-finderr); each
  declared mutation observed flipping its own test. Review follow-up: every `return` is checked and an assignment is
  read from its own column (RED ST10/ST11 2 failing → GREEN 13/13, both mutations observed). Real-tree smoke: 3 client roots (0 rows); decompiled vendor code,
  590 files declaring `getServiceTypes`, 0 FAIL, 6 WARN no own type (registered only under an interface type,
  genuine by the rule), 4 WARN unresolved `[ev: corpus B1419 §1419.1]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | One Java package per module part (duplicate or extract a utility module; N5 refuses split packages); delete `module-include.xml` or its stale line after a type rename/delete | `types/structure.md` § Layout, § module-include.xml | `[ev: corpus B1417 §1417.3]` |
| Δ2 | Palette type spec `alias:Type` without the `B`, `m=` once on the first `<p>`; correct the class-name example | `types/structure.md` § Palette assembly templates | `[ev: corpus B1418 §1418.5 e]` `[ev: corpus B1419 §1419.4 d]` |
| Δ3 | Default dev key (`LocalSigningProfile`, alias `Niagara4Modules`, `niagara.signing.jceks`, RSA-3072, 365/183 days); default profile off for releases; an untrusted module stops the station; N5 drops `low` mode | `types/distribution.md` §5 Development path | `[ev: corpus B1417 §1417.1]` |
| Δ4 | Version and N5 readiness checklist (public vs private API, target-version docs, alarm/history APIs only, N5 items); second local station ports; New Module wizard location | `METHODOLOGY.md` § Version and N5 readiness; `BUILD-LOOP.md` §0.b, §1.a | `[ev: corpus B1416 §1416.5]` |
| Δ5 | Slot-o-Matic: read the output every run, never code inside the generated region, recovery for "failed to process", lower-case `string`, lost `package` line; Gradle-sync the IDE after a `*.gradle.kts` edit | `types/structure.md` § Slot-o-Matic runs; `BUILD-LOOP.md` §2 | `[ev: corpus B1418 §1418.5]` |
| Δ6 | Lexicon per whole key, lexicon module jar, locale check in WB and browser; `getHtmlSafeText` for lexicon text in HTML; BQL `LIKE` case-sensitive, NEQL `(?i)` route, client-side result size | `types/dashboard.md` § Module packaging, § BajaScript; `types/security.md` §4.3 | `[ev: corpus B1418 §1418.1]` `[ev: corpus B1418 §1418.3]` |
| Δ7 | Secrets in a `BPassword` slot (a private field is not persisted); no stack trace on security log messages; native `test` runner; no network-dependent tests, no `gradle -t` with autosave | `types/security.md` §3.2, §3.3; `types/moduleTest.md` | `[ev: corpus B1416 §1416.4 b]` `[ev: corpus B1418 §1418.2 d]` |
| Δ8 | `getServiceTypes()` returns its own `TYPE` + new `lint-service-types.sh`; text bytes name their charset; compile warning-clean (`-Werror` vendors, scoped `@SuppressWarnings`) | `METHODOLOGY.md` § rt, § Build; `toolbelt/lint-service-types.sh` | `[ev: corpus B1419 §1419.1]` `[ev: corpus B1419 §1419.4]` |

Each Δ was folded by the commit that names its topic group on this branch; this commit flips the retro and its
INDEX row.

**Rows owed elsewhere (outside this task's edit surface).** They are not deltas of this retro (Δ1-Δ8 all landed), so
the retro stays folded; the owed rows are tracked as an open issue in `BUILD-STATE.md` (kit `open_issues`, "OWED
quality-round8 T9f course-recording rows") until a session with those files in scope folds them:
- `types/observability.md` §1 Level mapping — D3-6 (messages at SEVERE/WARNING/INFO, stack traces from FINE,
  CONFIG rare) and the D3-8 reason (users cannot switch `System.out` off; `lint-no-system-out.sh` already enforces
  the rule); §3 Fault surfacing — D4-15 lexiconized `configFail` cause. `[ev: corpus B1418 §1418.2]` `[ev: corpus B1419 §1419.4 c]`
- `types/logic-authoring.md` § BQL from code — a pointer to the BQL case rules in `types/dashboard.md` (D3-23).
- Component lifecycle rows deferred until T9b merges (`types/logic-authoring.md`): D1-12, D1-14, D1-15, D1-16,
  D1-17, D1-18, D2-11, D2-12, D2-13.

## Lessons
- A spoken demo, the official devguide and our own older block can agree and still be wrong for our hosts: the
  signing store is `.jceks` on both our machines. Fold the live-verified fact and name the disagreement.
- A kit example is doctrine too: the palette example with Java class names contradicted every real palette in our
  modules and fixtures; the scaffold emits the right form, so nothing exercised the example.
- A new lint smoked on vendor code first finds its own blind spots: the first vendor run gave 145 WARN rows from
  `this.getType()`, parenthesized returns and fields assigned in static blocks; three structural rules brought it to
  10, each pinned by a test.

---
**Status**: FOLDED 2026-10-22 — Δ1-Δ8 folded by the three topic commits on `docs/r8-t9f-course-rules`; INDEX row flipped to folded.
