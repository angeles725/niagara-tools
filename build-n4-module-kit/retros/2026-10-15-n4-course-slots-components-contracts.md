<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-15 · kit · n4-course-slots-components-contracts

**Session**: N4 Developer Certification Course (Tridium, N4.15.6.20) mined against the kit — decks 2–4, Labs 1–4, Review Assignments 1–2, Slide Code, corpus B1231/B1232/B1253/B1254
**Delta count**: 6

## What happened
The course's slot and component decks, plus the corpus code-reading behind them (B1231, B1232) and the answer keys for the
labs and Review Assignments (B1253, B1254), yield slot/component contracts the kit lacks — and one the kit states wrongly.
`types/actions.md:24-26` says a wrongly named action handler "dispatches to nothing — a silent no-op"; for a frozen
(`@NiagaraAction`) action the 4.14 type introspector rejects the type at load time ("Missing action do method"), and when
both `doX()` and `doX(Context)` exist the no-Context one wins, so the Context variant never runs. The kit also has no rule
that READONLY / CONFIRM_REQUIRED are Workbench-only (enforce with `checkSetFlags`), that `IPropertyValidator` is skipped by
link writes (null Context), that post-change callbacks need an `isRunning()` first line (only the narrower
"Clock.schedule reachable from changed()" rule exists, `types/logic.md:38`, `toolbelt/lint-timers.sh:563`), that
`get(name)` throws for actions, that a status struct written field by field propagates twice, or that run-state slots and
single-flight guards need TRANSIENT + a serialising executor because each Workbench session invokes on its own thread.

## Evidence
- Frozen-action contract enforced by `ComplexIntrospector` (missing/wrong `do` method → type-load error; `doX()` looked up before `doX(Context)`) — `[ev: corpus B1231 §1231.4]`; kit's contrary sentence — `[ev: code types/actions.md:24-26]`; deck: "always include [Context] in case you need it for permissions-checking" — `[ev: course Slides/2 Slots.pdf]` (via B1231 DECK:90-101)
- READONLY enforced only by the Workbench dialog; `checkSetFlags` enforces it on API/BOX/Fox paths; CONFIRM_REQUIRED is a UI hint, not security — `[ev: corpus B1231 §1231.9]` `[ev: corpus B1254 §1254.5 trap 4]`
- `IPropertyValidator`: "This only validates user-invoked sets"; link writes (null Context) skip it — `[ev: corpus B1232 §1232.4]`
- Callbacks without `isRunning()` double-count across restarts (slide `BComponentArray`) — `[ev: corpus B1232 §1232.2]` `[ev: corpus B1242 §1242.4 row 6]`; B1242 K4 proposes the lint — `[ev: corpus B1242 §1242.7 K4]`
- `get(name) != null` throws for actions → use `getSlot(name) != null`; `removeAll()` deletes links and `wsAnnotation`; fixed `add("link", …)` collides; slot-path link sources break on rename → `"link?"` + `getHandleOrd()` — `[ev: corpus B1231 §1231.6]` `[ev: corpus B1231 §1231.11]`
- `out.setValue(sum)` then `out.setStatus(status)` = two propagations and two `changed(out)`; links fire before `changed()`; `setOut(new BStatusNumeric(...))` once — `[ev: corpus B1253 §1253.3 K1]`; `[ev: course Labs/Lab3 Add Part 2/BAdd.java]`
- Lab 6 `state` persisted (READONLY only) → stuck `submitting` after a restart; check-then-set guard races across two Workbench sessions (each on its own `Fox:Dispatcher`) — `[ev: corpus B1253 §1253.3 K6]`
- Generated ids: regenerate in `started()` (DEFAULT_ON_CLONE copy keeps `""`), not in the constructor, a default, `added()` or `changed()`; never TRANSIENT — `[ev: corpus B1254 §1254.5 traps 1-3]`
- History extension `enabled` carries `DEFAULT_ON_CLONE` → every pasted point's history ext is disabled — `[ev: corpus B1253 §1253.3 K4]`
- Kit grep 2026-10-15 (non-retro): `checkSetFlags`, `IPropertyValidator`, `getSlot(`, `link?`, `if (!isRunning()) return` → 0 hits

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Correct §1: for a frozen `@NiagaraAction`, a missing/mis-signed `do` method is a TYPE-LOAD error ("Missing action do method", "mismatched signature"), not a silent no-op; keep the silent-no-op wording only for the reflective dynamic-action path it was observed on `[INFER]` (the kit cite is slot-o-matic `getMethod`, not the introspector). Add: declare exactly ONE handler — if both `doX()` and `doX(Context)` exist the no-Context one wins; Tridium recommends always taking `Context`. Lint: YES — WARN when a class declares both `do<X>()` and `do<X>(…Context)`. | `types/actions.md` § 1 (handler naming gotcha, :24-26) | `[ev: corpus B1231 §1231.4]` |
| Δ2 | Add "flags are UI, not security": READONLY and CONFIRM_REQUIRED are honoured only by the Workbench property sheet; to block API/BOX/Fox/oBIX writes override `checkSetFlags` (or gate in the setter path) and check the Context's permissions. Lint: no. | `types/security.md` § 2 · RBAC | `[ev: corpus B1231 §1231.9]` |
| Δ3 | Add the callback contract: first line of `changed/added/removed` is `if (!isRunning()) return;` (they fire during BOG decode); dispatch `changed()` on input slots only so output writes re-enter as no-ops; recompute derived/counter slots in `descendantsStarted()` and flag them TRANSIENT. Lint: YES (mechanical, B1242 K4) — WARN on a `changed(`/`added(`/`removed(` override whose first statement is not an `isRunning()` guard. | `types/logic.md` § Tridium rt idioms + new lint (or `lint-timers.sh` sibling) | `[ev: corpus B1232 §1232.2]` |
| Δ4 | Add validation scope: `IPropertyValidator.validateSet` runs only for user-context / forceValidate sets; link-fed (null Context) writes skip it, so clamp link-fed values in the logic, and use a `LocalizableRuntimeException(module, key)` whose key exists in `module.lexicon` for the user message. Lint: no. | `types/logic.md` § Safety fail-modes & timers (or logic-authoring) | `[ev: corpus B1232 §1232.4]` |
| Δ5 | Add slot-API traps: test existence with `getSlot(name) != null` (`get(name)` throws for actions); never `removeAll()` on a component with links or `wsAnnotation`; persistent dynamic links are named `"link?"` with a handle-ORD source (`getHandleOrd()`), not a slot-path ORD. Lint: YES — WARN on `get("…") != null` / `removeAll()` in rt sources. | `types/logic-authoring.md` (dynamic slots) or `types/logic.md` § Linking across custom modules | `[ev: corpus B1231 §1231.6]` |
| Δ6 | Add write/state discipline: write a status struct as ONE new value (`setOut(new BStatusNumeric(v, status))`), never `getOut().setValue()` + `setStatus()` (two propagations; links fire before `changed`); run-state slots (`submitting`, `busy`) are TRANSIENT; a single-flight action serialises on one executor or an `AtomicBoolean.compareAndSet`, because concurrent Workbench sessions invoke on separate dispatcher threads; generated ids are created in `started()`, persisted, never TRANSIENT; commissioning checklists include "re-enable collection" after pasting points with history extensions (`DEFAULT_ON_CLONE`). Lint: YES (partial) — WARN on `get<Slot>().setValue(` followed by `get<Slot>().setStatus(` in one method. | `types/logic.md` § Slots — BStatus producers + § Restart & persistence | `[ev: corpus B1253 §1253.3 K1]` |

## Lessons
- The type introspector, not slot-o-matic, decides whether an action handler is valid; trust the load-time error over reflective-dispatch folklore.
- Every Workbench session and the engine are separate threads; any "check then set" on a component needs a lock or a single executor.
- Validators and READONLY protect only the user path; link writes and API writes need their own guards.
