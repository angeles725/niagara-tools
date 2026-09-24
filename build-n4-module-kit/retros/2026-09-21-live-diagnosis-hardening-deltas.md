<!-- review-status: folded -->
# 2026-09-21 · kit · live-diagnosis-hardening-deltas

**Session**: niagara-research focus `module-hardening`, blocks **B1158–B1160** — a LIVE-DIAGNOSIS deepening of the hardening focus. The PANCCADIA JACE-9000 (client station, 4.15.3.28) was restarting every 2–3 days from tenured-gen heap exhaustion (engine watchdog `terminate`, rc=-119). A read-only oBIX diagnosis + a source review of the operator's three custom modules (DashboardPan v2.4.2, CompPan v2.1.0, ColdRoomPan v2.1.2) traced it to a control-callback anti-pattern, not a textbook growing-collection leak. This is the FIRST time the hardening focus closed `[CERT-live]` gaps against a real station (the focus had parked its live residues for exactly this).

**Delta count**: 6 (3 lint candidates + 3 doc entries)

## What happened
Three net-new module-hardening failure modes, each backed by live-station evidence AND client source, none covered by the kit's current lint/doc set:

1. **RUN8 — hot `changed()` + non-transient write (B1158).** `CompPan.BCompressorControl.changed()` re-runs the full control cycle on every NRIO-linked input (suction/discharge pressure, amps, room calls). Under a churning field bus (~18 msg/s from a flapping IO-34 relay) it ran `execute()` ~18×/s instead of its 5-s tick — 25–90× over-execution on the single engine thread (no backpressure, B31) — and each call wrote three **non-transient** persisted slots. Fix applied to the client: a leading-edge time debounce on the callback path (`Clock.millis()` gate, `faultReset` bypass); hour integration is time-based (`ctl.step(Clock.millis(),…)`) so throttling costs nothing.

2. **PER8 — persisted accumulator written every cycle (B1159).** The persistence half: `condenserNHours` are correctly non-transient (must survive restart for fair lead/lag rotation) but were WRITTEN every cycle, so each call marked the station config dirty and fired the DashboardPan links. Rule: integrate in memory, checkpoint the non-transient slot on the tick + in `stopped()`; seed once on start.

3. **UXS7 — static `-ux` session store, lazy-only eviction (B1160).** `DashboardConfigSession.SESSIONS` is a `static ConcurrentHashMap<jsessionId,Entry>` that inserts on login, removes on explicit logout, and evicts expired entries only when the same id is queried. Unbounded (a session that never logs out is stranded — negligible heap, explicitly NOT the PANCCADIA leak) AND, being static, it survives servlet re-mount → a reused JSESSIONID can inherit an authenticated write session (session-fixation shape). Rule: sweep-on-insert; question `static`; bind auth to user+issue-time not the bare container id; audit the write.

## Evidence
- RUN8: `BCompressorControl.changed():1850-1871` (≈30 `p==` branches → `execute()`, no rate guard); `execute():2019-2021` writes `setCondenserNHours`; decl `:1403/1426/1449` = `newProperty(Flags.SUMMARY|Flags.READONLY,…)` (non-transient), comment `:326` "Persisted run-hours (NOT transient…)"; integration `ctl.step(Clock.millis(),…):2003`. Live: PANCCADIA oBIX diagnosis 2026-09-20/21 (tenured→~100%/2-3d, watchdog rc=-119, ~18 NRIO msg/s). `[ev: corpus B1158]`
- PER8: same slots; read side already correct `seedHours():2063` (once, `hoursSeeded` guard); cost mechanism B402/B411 (config.bog save) + B754/B795/B799 (survival matrix). `[ev: corpus B1159]`
- UXS7: `DashboardConfigSession.java:31-32` static map; `:83-85` put on login; `:94-95` remove on logout; `:126` lazy evict; `:29` 5-min sliding TTL. `[ev: corpus B1160]`

### Verified DISTINCT from existing kit lints (not duplicated)
| Existing lint | Covers | Why the proposed delta is distinct |
|---|---|---|
| `lint-timers.sh` | Clock/BTicket scheduled without a reachable `.cancel()` in `stopped()` | RUN8 is WRITE FREQUENCY from a callback, not a ticket-lifecycle leak (both client modules are ticket-disciplined) |
| `lint-subscribe-without-unsubscribe.sh` | RUN4 Subscriber leak | RUN8/UXS7 are not subscriber retention; different mechanism |
| `lint-null-context-write.sh` | `set()`/`invoke()` with a null Context (RBAC/audit bypass) | PER8 is non-transient write CADENCE, not null Context |
| `lint-write-path.sh` | server-side RBAC on the write path | UXS7 is the session-STORE lifecycle (eviction/static), upstream of the request gate |
| `lint-clock-zero-floor.sh` | `Clock.schedule` delay ≤ 0 | unrelated (delay flooring, not callback rate) |

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | **Lint `lint-changed-hot-write`**: a `changed(Property,Context)` override with N≥6 `p == <prop>` branches calling a heavy `execute()`/control method, WITH no rate guard (no `Clock.millis()` delta gate / no coalescing ticket) AND the class writes ≥1 non-transient slot reachable from it → WARN "hot changed() re-runs at input rate + writes persisted slots → engine+persistence flood under field-bus churn; debounce or move periodic writes to a tick." | `toolbelt/lint-changed-hot-write.sh` (new) + `tests/*.bats` | `[ev: B1158; BCompressorControl.java:1850-1871,2019-2021]` |
| Δ2 | **Doc: RUN8 gotcha** — issues-and-gotchas §B (runtime): symptom (slow tenured climb + watchdog restart under bus churn), root (callback rate == bus rate × non-transient writes), fixes (debounce; periodic writes on the tick; time-based integration is rate-independent). Cross-link B31 (no backpressure), B729/B730 (changed discipline). | `types/issues-and-gotchas.md §B` | `[ev: B1158]` |
| Δ3 | **Lint `lint-persist-hot-write`**: a `setX(...)` on a NON-transient Property (decl without `Flags.TRANSIENT`) called from `changed()` or a method reachable from it, with no cadence guard → WARN. Pairs with Δ1. | `toolbelt/lint-persist-hot-write.sh` (new) + bats | `[ev: B1159; BCompressorControl.java:326,1403,2019]` |
| Δ4 | **Doc: PER8 gotcha** — issues-and-gotchas §C (persistence): "persisted accumulator written hot" — integrate in memory, checkpoint the non-transient slot on a periodic tick + in `stopped()`, seed once on start. Cross-link B754/B795/B799 (declare side) + B402/B411 (cost side). | `types/issues-and-gotchas.md §C` | `[ev: B1159]` |
| Δ5 | **Lint `lint-session-store-lazy-evict`**: a `static` `Map<String,*>` field in a `-ux` class with `.put(` on a login/auth path and removal ONLY via an explicit-logout method or a query-time check (no sweep reachable from insert, no scheduled purge) → WARN (advisory). "-ux retention" family with the subscriber lint. | `toolbelt/lint-session-store-lazy-evict.sh` (new) + bats | `[ev: B1160; DashboardConfigSession.java:31-126]` |
| Δ6 | **Doc: UXS7 gotcha** — issues-and-gotchas §D (ux/servlet security): "-ux session/token store lifecycle" — sweep expired on insert (not lazy-only), instance-scope unless cross-instance sharing is intended (static survives re-mount → fixation window), bind auth to user+issue-time not bare `JSESSIONID`, audit the write. Cross-link UXS5/B1120 + B829. | `types/issues-and-gotchas.md §D` | `[ev: B1160]` |

## Lessons
- The hardening focus mapped these failure modes from CODE months ago (RUN1/RUN2 callback semantics, PER survival matrix, UXS write gate) but had no way to see the WRITE-FREQUENCY family until a real station churned. A live diagnosis is what turned "changed() semantics" into "changed() rate is a heap failure mode."
- The leak was NOT any of the textbook shapes the kit already lints (no growing collection, no subscriber leak, no uncancelled ticket — all three client modules are disciplined there). The new seam is: correct declarations (non-transient slot, standard `changed()`) written at the WRONG RATE. Worth three advisory lints because the shape is invisible to a structural review.
- Honesty: RUN8/PER8 are the strongest CODE seam and a confirmed 25–90× over-execution defect; the DOMINANT retained class still needs a live `jmap -histo` to attribute the heap definitively (JACE ships a JRE, blocked). UXS7 as a heap leak is negligible — its value is the session-fixation rule.

---
**Status**: FOLDED — `toolbelt/lint-changed-hot-write.sh` (Δ1) + `types/issues-and-gotchas.md`
§I1 (Δ2), `toolbelt/lint-persist-hot-write.sh` (Δ3) + §I2 (Δ4), `toolbelt/lint-session-store-
lazy-evict.sh` (Δ5) + §F2 (Δ6). All three lints have bats coverage (fixtures + a `# Mutation:`
guard-pin) and are named in `BUILD-LOOP.md`. INDEX row flipped.
