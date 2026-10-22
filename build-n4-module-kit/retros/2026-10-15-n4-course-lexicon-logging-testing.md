<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-15 · kit · n4-course-lexicon-logging-testing

**Session**: N4 Developer Certification Course (Tridium, N4.15.6.20) mined against the kit — deck 7, Lab 7, Lab 10 lexicon, the 34-test Slide Code suite, corpus B1236/B1252/B1242; the user ran the course station with a Spanish locale
**Delta count**: 5

## What happened
Deck 7 and Lab 7 teach the lexicon, logging and testing idioms; the kit enforces only that `module.lexicon` exists and is
non-empty (`types/structure.md:54` L4), checks slot keys mechanically (`toolbelt/slot-coverage.sh`), and documents
`BTestNg`/`BTestNgStation` (`types/moduleTest.md`). Missing: the `<Type>.displayName` key (and `slot-coverage.sh` counts a
bare `<Type>=` key as covering the type, which does nothing for the type display name), `LexiconModule.getText` with
MessageFormat arguments for operator-facing fault causes, `%lexicon()%` in facets, the 4.14+ test layout, Mockito static
mocking at a factory seam, `BTestNgStation`'s per-class station lifetime, seeding the alarm/history DBs, and asserting FINE
log lines. On the live course station with a Spanish locale the console printed `DETALLADO` for FINE; the kit's console
triage recognises `GRAVE`, `ADVERTENCIA` and `INFORMACI*` but not `DETALLADO`.

## Evidence
- Slot-name key = slot display name; `<NiagaraType>.displayName` = type display name; enum tags and `%lexicon(slides:enabled.value)%` TRUE/FALSE_TEXT — `[ev: course Slides/7 Logging, Lexicons, and Testing.pdf p13-20]`; bare `<Type>=` ineffective for the type, `slot-coverage.sh` strips after the first dot so both forms count — `[ev: corpus B1236 §1236.3]` `[ev: corpus B1242 §1242.7 K1]` `[ev: code toolbelt/slot-coverage.sh:39-41]`
- `LexiconModule.make(BWorkOrderService.class)` + `getText("workOrderService.serializeFailure", context, ex.getLocalizedMessage())` into `configFail` — `[ev: course Labs/Lab7 Testing, Lexicons, and Logging/BWorkOrderService.java:107-185,272]`; MessageFormat `''` rule — `[ev: corpus B1236 §1236.4]`; log for the integrator, lexicon for the operator — `[ev: corpus B1236 §1236.7]`; job `log().message("training", key, args)` keys must exist — `[ev: course Labs/Lab Instructions.pdf p86]` `[ev: course Labs/Lab10 Jobs/module.lexicon]`
- `isLoggable(Level.FINE)` guard, default INFO, DebugService (`baja:LoggingService`) / `spy:/logSetup` to raise a category — `[ev: course Slides/7 Logging, Lexicons, and Testing.pdf p9]` `[ev: corpus B1236 §1236.2]`; Lab 11 logs every frame at INFO — `[ev: corpus B1239 §1239.12 row 6]`
- Tests: `srcTest/test/com/...` + `package test.com...` (4.14+), class-level `@Test`, `BTestNg` vs `BTestNgStation` choice table, station built in `@BeforeTest` so state persists across methods, DB seeding recipes, fixed `Thread.sleep` waits in 7 tests (do not copy; use `TestHelper.assertWillBeTrue`) — `[ev: corpus B1252 §1252.3]` `[ev: corpus B1236 §1236.5]`; kit layout example is the 4.13 `srcTest/com/<vendor>/<mod>/test/` — `[ev: code types/moduleTest.md:20]`
- `testUberjar("org.mockito:mockito-inline:4.11.0")` + `Mockito.mockStatic(HttpConnectionFactory.class)` inside `BTestNgStation` — `[ev: course Labs/Lab7 Testing, Lexicons, and Logging/BWorkOrderServiceTest.java:7,29-34,53]` `[ev: /mnt/c/N4-Developer-Training/Core-Class/N4_Labs/niagara_dev_home/training/training-rt/training-rt.gradle.kts]`; FINE assertions need the logger lowered to ALL (course `LogUtil`) — `[ev: corpus B1236 §1236.6]`
- Spanish console level names observed live: `INFORMACIÓN`, `GRAVE`, `DETALLADO` (user session 2026-10-15); `triage-console.sh` `is_level`/`norm_level` lack `DETALLADO` — `[ev: code toolbelt/triage-console.sh:99-111]`; the remaining JDK `es` names (CONFIG `CONFIGURAR`, FINER `MUY DETALLADO`, FINEST `MÁS DETALLADO`) are `[INFER]` (JDK `sun.util.logging.resources.logging_es`, not read here)
- Kit grep 2026-10-15 (non-retro): `LexiconModule`, `MessageFormat`, `mockStatic`, `mockito`, `testUberjar`, `DETALLADO`, `logSetup` → 0 hits

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add a lexicon key catalogue: `<slot>=` (slot display name), `<Type>.displayName=` (type display name; a bare `<Type>=` does nothing for the type), enum tag keys, `%lexicon(<module>:<key>)%` inside facet values (TRUE_TEXT/FALSE_TEXT), and fix `slot-coverage.sh` parse mode to require `<Type>.displayName` per `<type name>` and WARN "ineffective for type display" on a bare `<Type>=`. Lint: YES (mechanical, B1242 K1). | `types/structure.md` § lexicon (near L4) + `toolbelt/slot-coverage.sh` parse mode | `[ev: corpus B1236 §1236.3]` |
| Δ2 | Add the runtime lexicon idiom: `private static final LexiconModule LEX = LexiconModule.make(X.class)`; operator-facing text (fault cause, validator message, job log) comes from `LEX.getText(key, cx, args…)` / `log().message(module, key, args)`; such entries are MessageFormat (`''` for an apostrophe); English logs stay for the integrator. Lint: YES (mechanical) — new check: every literal key passed to `getText(`, `log().message(`, `LocalizableRuntimeException(` exists in that module's `module.lexicon` (FAIL), and a single `'` in an entry read with arguments (WARN, B1242 K11). | `types/observability.md` § 3 · Fault surfacing + new `toolbelt/lint-lexicon-keys.sh` (or `lint-lexicon-ascii.sh` companion) | `[ev: course Labs/Lab7 Testing, Lexicons, and Logging/BWorkOrderService.java]` |
| Δ3 | Update moduleTest layout and idioms: 4.14+ layout `srcTest/test/com/<vendor>/…` with `package test.com.<vendor>…` beside the 4.13 one; class-level `@Test`; pick `BTestNg` (detached logic) vs `BTestNgStation` (services, JobService, alarm/history DB, BQL, permissions); the station is built once per class (`@BeforeTest`) so fixtures must be idempotent; seed alarms/histories through `getDbConnection(null)`; mock I/O at a static factory seam with `testUberjar("org.mockito:mockito-inline:4.11.0")` + `Mockito.mockStatic(...)` in try-with-resources; lower the logger to `Level.ALL` to assert FINE lines; never copy the suite's fixed `Thread.sleep` waits. Lint: no. | `types/moduleTest.md` § Gradle dependencies + § TEST-G1/G2 + § TEST-G8 layout | `[ev: corpus B1252 §1252.3]` |
| Δ4 | Teach `triage-console.sh` the Spanish JDK level names: add `DETALLADO`→FINE (observed), and `CONFIGURAR`→CONFIG, `MUY DETALLADO`→FINER, `MÁS DETALLADO`→FINEST after confirming them from the JDK resource bundle; add a bats row per name. Lint: YES (mechanical tool fix). | `toolbelt/triage-console.sh` § is_level / norm_level (:99-111) | `[ev: code toolbelt/triage-console.sh:99-111]` |
| Δ5 | Add the log-level operating notes: default level is INFO; raise one category in Services › DebugService (`baja:LoggingService`) or `spy:/logSetup` instead of logging at INFO; per-frame / per-poll traces go at FINE behind `isLoggable`. Lint: YES (partial) — WARN on `logger.info(`/`LOG.info(` inside `receive`/`doPoll`/`sendMessage` bodies of driver classes. | `types/observability.md` § 1 · Logging | `[ev: corpus B1236 §1236.2]` |

## Lessons
- A lexicon key's shape decides what it labels; a check that only proves "some key exists" misses the type display name.
- Operator text belongs in the lexicon and is looked up with the caller's Context; that makes its keys mechanically checkable against `module.lexicon`.
- Console tooling must parse localized JDK level names; every Spanish-locale station prints them.
