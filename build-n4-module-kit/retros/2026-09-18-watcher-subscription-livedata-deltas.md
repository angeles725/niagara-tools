<!-- review-status: pending -->
# 2026-09-18 · kit · watcher-subscription-livedata-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — watcher/live-data).
**Delta count**: 7

## What happened

Mined the WATCHER/subscription/live-data vein across the five primary blocks (B4, B408, B867, B763, B752)
plus the two transport-consolidation blocks referenced from them (B512, B553). Focus areas:
`BWatch`/lease, `SubscribeCallbacks`, `ComponentSpaceConnection`, poll-vs-subscribe in `-ux` servlets and
`-wb` managers, subscribe/unsubscribe discipline, and a "how to feed live data" recipe.

**Dedup run against:**
- `retros/INDEX.md` (all entries, full-text grep on subscribe/watch/lease/BWatch/TypeSubscriber)
- `retros/2026-09-18-module-mechanics-deltas.md` — Δ1 (TypeSubscriber recipe) and Δ2 (default-mask
  gotcha) already proposed from B867; Δ7 (BPollScheduler) from B872 — skipped here.
- `retros/2026-09-18-ux-wb-writesurface-rbac-deltas.md` — Δ2 (bajaux data-channel dialects, incl.
  `subscriberMixIn`) already proposed from B752 §752.2 — skipped here.
- `types/logic-authoring.md §Author-side SPIs` — instance/depth `Subscriber` recipe + `unsubscribeAll()`
  on stop + cross-module subscribe = same as same-module [B778/B802] already folded — skipped.

**BWatch / ComponentSpaceConnection**: neither term appears in the five primary evidence blocks nor in
B512/B553. `BObixWatch` / oBIX server Watch lease model lives in B509 (not a primary block here) — deferred
to a future B509-focused retro; `ComponentSpaceConnection` is absent from the corpus evidence entirely.

**Net-new vein surface mapped:** `SubscribeCallbacks` as the driver demand-hook (B408 §408.4, never
surfaced in the kit); `BComponent.subscribed()`/`unsubscribed()` callbacks on the observed side (B4
§4.3.1 + B408 §408.4, distinct from the observer-side `Subscriber`); TypeSubscriber supertype-walk
unbounded-scope gotcha (B867 §867.3); BOX `ProxyBroker` + `BrokerPoller` as the actual live-data
substrate and its 2s rate-limit (B512 §512.4); a live-data decision matrix connecting B752 §752.4 to
B553 §553.3; a `verify-module.sh` lint rule for TypeSubscriber leak; and the causal chain
BOX-sub → space-subscriber-count → `SubscribeCallbacks` → `component.subscribed()` (B512 + B408).

## Evidence

- `SubscribeCallbacks.subscribe(BComponent[] c, int depth)` fires on the 0→1 subscriber-count transition;
  `unsubscribe(BComponent[] c)` fires on 1→0; `update(BComponent c, int depth)` is a one-time snapshot (no
  ongoing subscription). Default = no-op; replaced via `setSubscribeCallbacks()`. Dispatch order: space hook
  → `fw(17, ...)` → `component.subscribed()` (direct callback if `fireDirectCallbacks()`).
  `[ev: corpus B408 §408.4, BComponentSpace.java:1742-1762, ComponentSlotMap.java:657-693]`

- `BComponent.subscribed()` / `unsubscribed()` are lifecycle callbacks on the WATCHED component — they fire
  when a remote FOX or local subscriber starts/stops watching it. Different from `Subscriber.subscribe()`,
  which is the OBSERVER API (I am watching). Verified in B4 §4.3.1 lifecycle table (L633-L660 in the
  decompiled BComponent.java) and confirmed by the dispatch sequence in B408 §408.4.
  `[ev: corpus B4 §4.3.1, B408 §408.4.2, ComponentSlotMap.java:657-673]`

- `BComponentSpace.event(BComponentEvent)` recurses up `getSuperType()` on dispatch: a `TypeSubscriber` on
  `BNumericPoint.TYPE` catches ALL subtype instances across the space. Subscribing to `BComponent.TYPE` = an
  unbounded "watch everything" subscription for the masked event-ids. `isSubscribed(type)` also recurses up
  the supertype chain. `[ev: corpus B867 §867.3, BComponentSpace.java:302-314,349-373]`

- BOX subscribe flow: browser BOX `sub` op → `ProxyBroker.subscribeOp(comp, 0, isVirtual)` → `BrokerPoller`
  daemon wakes on `SyncOp`, debounces ~20 ms, then delivers one burst. Server-side rate-limit:
  `POLL_INTERVAL = 2000 ms` (default) — at most one push burst per 2 s regardless of change rate. HTTP v1 =
  client polls `pollchgs`; WebSocket v2 = server pushes unsolicited `"t":"u"` frames. This `sub` is what
  drives the space-level subscriber-count, which in turn fires `SubscribeCallbacks.subscribe()` and
  `component.subscribed()`. `[ev: corpus B512 §512.4, BServerSession.java:109, BComponentSpaceSessionHandler.java:217]`

- N4 push transport map (B553 §553.1–553.3): browser live data = BOX only (subscribe via `ProxyBroker`);
  station-to-station = Fox; custom SPA needing push = BOX WebSocket; plain HTTP servlet context = none (must
  REST-poll). `subscriberMixIn` / `Subscriber.subscribe()` in bajaux/PX = implicit BOX subscription; NOT
  available to a plain `BWebServlet`. `[ev: corpus B553 §553.1–553.3, B752 §752.4]`

- A `TypeSubscriber` subclass holds strong references in `typeSubscriptionMap` (keyed per type, per event id)
  on the space. No reference is released without an explicit `unsubscribe` call. A subscriber that omits
  `unsubscribeAll()` in `stopped()` keeps the space-map populated until station restart — a permanent leak
  per-instance of the subscriber component. `[ev: corpus B867 §867.2, BComponentSpace.java:1759,222-247,
  B408 §408.4]`

- B752-G2 (Fox `subscriberMixIn` client contract — attach/detach lifecycle) is an explicit OPEN GAP in the
  corpus; do NOT propose a kit rule about it until B752-G2 is walked.
  `[ev: corpus B752 open-gaps §B752-G2]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add `SubscribeCallbacks` demand-subscription hook for driver/proxy-space authors: override `subscribe(BComponent[], depth)` to register a native device COV subscription when subscriber count goes 0→1; override `unsubscribe(BComponent[])` to deregister on 1→0; use `BComponentSpace.update(c, depth)` for a one-time snapshot (no ongoing sub). Note: the framework tracks the count — one `subscribe` call per 0→1 crossing, not per individual subscriber joining. Set via `space.setSubscribeCallbacks(impl)`. | `types/logic-authoring.md §Authoring a driver` (new "Demand-based subscription" bullet) | `[ev: corpus B408 §408.4]` |
| Δ2 | Add `BComponent.subscribed()` / `unsubscribed()` observed-side pattern: these callbacks fire ON a component when a remote client (Workbench, BOX browser) STARTS/STOPS watching it — distinct from `Subscriber.subscribe()` (which is the OBSERVER API). Override to do demand-based resource acquisition/release (e.g., open a serial port only while someone is watching the status component). Caveat: fires only at the 0↔1 subscriber-count crossing, not on every client join/leave. | `types/logic-authoring.md §Author-side SPIs` (note after the existing Subscriber entry) | `[ev: corpus B4 §4.3.1, B408 §408.4.2]` |
| Δ3 | Add TypeSubscriber supertype-walk gotcha: `BComponentSpace.event()` recurses up `getSuperType()` — a TypeSubscriber on a BASE type (e.g., `BNumericPoint.TYPE`) catches ALL subtype instances' events across the entire space; subscribing to `BComponent.TYPE` = "watch everything" for the masked event-ids. Rule: be specific about the Type; profile event volume before subscribing to any framework base type; a too-broad base type causes a silent event flood with no compile-time warning. | `types/logic-authoring.md §Author-side SPIs` (immediately after the TypeSubscriber recipe from module-mechanics-deltas Δ1 when folded) | `[ev: corpus B867 §867.3]` |
| Δ4 | Add BOX BrokerPoller mechanics as context for live-data choice: a browser `subscriberMixIn` / `ProxyBroker.subscribeOp` subscription goes through a `BrokerPoller` with a 2000 ms server-side rate-limit — push bursts are batched, not per-change. Implication: high-frequency slots (1 Hz control tick) are throttled to one push burst per 2 s over BOX/WebSocket. A REST-poll at 5s is therefore equivalent in effective latency for a dashboard; the extra complexity of a BOX subscription (session management, `ssession` lifecycle) is only worth it when sub-2s latency actually matters and a bajaux `@AgentOn` context is already in use. | `types/dashboard.md §ux — servlet + SPA` (add as a note under the REST-poll bullet) | `[ev: corpus B512 §512.4]` |
| Δ5 | Add "How to feed live data to your SPA/view" decision matrix — match the serving recipe to the live-data channel: (a) **Servlet-SPA** → REST-poll (`setInterval` + 1s local tick for JACE-computed fields); no BOX/Fox subscription available in a plain `BWebServlet` context; (b) **bajaux `@AgentOn` view** → `subscriberMixIn` Fox/BOX subscription (push on value change; B752-G2 still open — treat as P2); (c) **PX page** → `*Binding` handles live data implicitly (no author code); (d) **Query** (alarms, history) → BQL via `BOrd.make(bql).get(cx, null)` → `BITable` (pull, not push). Rule: do NOT try to mix channels (no BOX subscription from a servlet, no polling from a PX binding). | `types/dashboard.md §ux — servlet + SPA` (add after the three-recipe picker, before RBAC) | `[ev: corpus B752 §752.4, B553 §553.1–553.3]` |
| Δ6 | Add subscription-leak lint to `verify-module.sh`: detect a `TypeSubscriber` subclass (class extending `javax.baja.sys.TypeSubscriber`) whose owning component has no `stopped()` override — WARN (not FAIL, because there are valid architectures where the space itself is torn down on stop). Rationale: `typeSubscriptionMap` holds strong refs; an uncleaned TypeSubscriber leaks until station restart. Companion to the existing `Clock.Ticket`-without-cancel FAIL check. Rule text for METHODOLOGY advisory checklist: "A `TypeSubscriber` that omits `unsubscribeAll()` in its component's `stopped()` leaks space entries until station restart." | `toolbelt/verify-module.sh` (new CHECK: TypeSubscriber-without-stopped WARN) + `METHODOLOGY.md §Conformance rules` advisory checklist | `[ev: corpus B867 §867.2, B408 §408.4]` |
| Δ7 | Add the BOX→SubscribeCallbacks→subscribed() causal chain to the kit's conceptual map: browser/Workbench BOX `sub` op → `ProxyBroker.subscribeOp` → space subscriber-count 0→1 → `SubscribeCallbacks.subscribe()` fires → `component.subscribed()` fires. This is WHY `subscribed()` and `SubscribeCallbacks` fire — the trigger is a remote client starting a live subscription, not a local observer. A `BComponent` author who overrides `subscribed()` is therefore reacting to a Workbench panel or browser SPA opening a live view on that component. | `types/logic-authoring.md §Author-side SPIs` (or a new "Live-data chain" cross-ref box after Subscriber + subscribed() entries) | `[ev: corpus B408 §408.4.2, B512 §512.3–512.4]` |

## Lessons

- The **observed-side** callbacks (`subscribed()`/`unsubscribed()`) and the **observer-side** API
  (`Subscriber.subscribe()`) are two different surfaces for two different authoring postures. The kit
  documents the observer side well; the observed side has zero coverage — a gap that bites whenever a driver
  or service needs demand-activation.
- The BOX `BrokerPoller` 2s rate-limit makes REST-poll (at 5s) and push-on-change (via BOX) practically
  equivalent for a slow-moving dashboard. The complexity tradeoff is clear: REST-poll wins unless the
  bajaux context is already in play.
- `BWatch`/`ComponentSpaceConnection` are absent from the five primary evidence blocks — they are not
  corpus-documented at the rule level. `BObixWatch` (oBIX server Watch + lease) is in B509 and is a separate
  surface; it belongs in a dedicated B509 retro.
- The supertype-walk in `TypeSubscriber` dispatch is the most dangerous piece of the subscription model: it
  is architecturally correct (base-type subscribers get all subtype events by design) but operationally a
  footgun when a module subscribes to a framework base type expecting narrow scope.
- B752-G2 (Fox `subscriberMixIn` attach/detach lifecycle) is still an open gap in the corpus. No kit rule
  can be written until it is researched; the dashboard.md note about "bajaux `@AgentOn` MAY use
  `subscriberMixIn`" (Δ5) intentionally defers to B752-G2 for the full contract.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-watcher-subscription-livedata-deltas.md | kit | 2026-09-18 | pending | 7 |`
