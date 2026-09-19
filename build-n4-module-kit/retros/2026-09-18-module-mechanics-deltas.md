<!-- review-status: pending -->
# 2026-09-18 · kit · module-mechanics-deltas

**Session**: corpus-mining for build-n4-module (operator: generate kit deltas from niagara-research).
**Delta count**: 7

## What happened

Mined the module-mechanics focus (B867–B932, MM1–MM36 + children) for concrete, kit-actionable
mechanics not yet folded into the kit guides. Ran dedup against the full retro INDEX.md and
the two same-day retros (`corpus-index-refresh-b761-b1028.md`, `sdk-examples-kit-deltas.md`)
and against the existing `types/logic.md`, `types/logic-authoring.md`, and the driver §s.

**Blocks read in full:** B867, B869, B871, B872, B873, B874, B900, B909, B923, B927.
**Partial / title scan:** B870, B875–B899, B901–B928, B929–B932.

**Focus:** subscription model (B867/B900), link converters (B871), event routing (B873),
schedule integration (B874), driver threading (B872/B909). Driver wire-protocol internals
(B875–B886, B903–B916, B924–B925 etc.) were scanned and skipped — no kit-actionable rule.

**Dedup findings:**
- Instance/depth `Subscriber` already in `logic-authoring.md §Author-side SPIs` [B778].
- `BTuningPolicy` write-on-up/start/enabled already in `logic-authoring.md §Authoring a driver`.
- Action protection `@NiagaraAction(flags=OPERATOR)` already in `logic-authoring.md §Action protection`.
- Link unidirectionality rule already in `logic.md §Links` (B869 substance, cross-module double).
- corpus-index.md pointer entries for B867/B871/B873/B874/B900/B909 already proposed by
  `2026-09-18-corpus-index-refresh-b761-b1028.md` (Δ12–18 of that retro) — not re-proposed here.

## Evidence

- `TypeSubscriber` abstract, ctor binds one space, default mask = `SELF_EVENTS`; abstract `event(BComponentEvent)`;
  `subscribe(Type[],cx)` validates `t.is(BComponent.TYPE)`, calls `space.subscribe(t,this)`; `unsubscribeAll()` on stop.
  `[ev: corpus B867 §867.2-3]`
- `BComponentEventMask.SELF_EVENTS = 1701888` covers component lifecycle ids 11–20 (parented…stopped), NOT
  `PROPERTY_CHANGED` (id=0) nor any `PROPERTY_EVENTS` set (mask 395263). A default-mask `TypeSubscriber` receives
  NO value-change callbacks — only component-lifecycle events. `[ev: corpus B867 §867.4, B900 §900.1]`
- `BConversionLink extends BLink`; converter fires inside `propagate`, in-flight on the unidirectional leg.
  Three status patterns: PROPAGATE (Status→Status copies bitmask), STRIP (Status→plain: null source → return
  unchanged `to`, NOT null/NaN), INJECT (plain→Status). The STRIP pattern silently holds last-good value when
  source is null/fault. No framework `isLossy` flag; lossiness is per-converter body.
  `[ev: corpus B871 §871.4]`
- One converter per link; no framework multi-hop: `BConversionLink` holds exactly one `BConverter` slot.
  `@Adapter(from,to)` registry auto-selects; an enum→double bridge loses the tag (ordinal-only, lossy).
  `[ev: corpus B871 §871.2-3, B928 §928]`
- Event routing IS the Baja link graph: `BEventSource.routeTo(name,routable)` = `add(name, consumer)` +
  `linkTo(source.event → consumer.process)`. No imperative registry. `BEventFilter` chains
  (consumer=process action + producer=event topic). `BEventRecipient` is terminal async via `ThreadPoolWorker`.
  Uniform `BEvent` envelope (uuid/timestamp/source ORD/open value). License: `tridium:eventService`.
  `[ev: corpus B873 §873.1-3]`
- `BComponentEventSource` wraps `Subscriber.event(BComponentEvent)` into a `BEvent` — the event module is
  an OVERLAY on B867 primitives; `BComponentTypeEventSource` uses `TypeSubscriber` internally.
  `[ev: corpus B873 §873.4]`
- `BControlSchedule.doExecute()`: checks `in` slot FIRST (manual-override link wins over all tree rules);
  then `getOutputSource(now)` = FIRST-SLOT-WINS positional walk (no numeric priority field); ONE `Clock.Ticket`
  rescheduled at each transition (no polling loop). `BWeeklySchedule` wires `specialEvents` at slot 0
  (beats `week` at slot 1). Scan horizon = 90 days (`scanLimit` default).
  `[ev: corpus B874 §874.2-4]`
- DDF comm splits solicited (request→response) from unsolicited via two SPI managers. Default strategy:
  ONE outstanding transaction (blocking-queue dequeue + `Clock.Ticket` timeout, serialized).
  Multiple strategy: tag-correlated concurrent (`Hashtable<tag, requests>` two-level map).
  `[ev: corpus B909 §909.1-3]`
- `BPollScheduler`: ONE poll thread per `BDeviceNetwork`, named `"Poll:<networkName>"`. Three rate buckets:
  fast=1000ms, normal=5000ms, slow=30000ms (default). `dibs` stack for immediate first-poll on subscribe.
  `[ev: corpus B872 §872.1]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add `TypeSubscriber` authoring recipe (type-level "watch every instance of Type T" SPI, distinct from instance-depth `Subscriber`); ctor, `subscribe(Type,cx)`, `event(BComponentEvent)`, `unsubscribeAll()` on stop; type check at subscribe time (non-component throws) | `types/logic-authoring.md §Author-side SPIs` | `[ev: corpus B867 §867.2-3]` |
| Δ2 | Add TypeSubscriber event-mask gotcha: default `SELF_EVENTS` = component-lifecycle ids 11–20 ONLY — does NOT include `PROPERTY_CHANGED` (id=0); must call `setMask(BComponentEventMask.PROPERTY_EVENTS)` to receive value-change callbacks; using the default mask and expecting value changes = silent miss | `types/logic-authoring.md §Author-side SPIs` | `[ev: corpus B867 §867.4, B900 §900.1]` |
| Δ3 | Add cross-type link BConverter STRIP gotcha: a Status→plain converter on a heterogeneous link returns the UNCHANGED target value (last-good) when source is null/fault — NOT null/NaN; fault does NOT surface at target; only a Status→Status converter propagates the status bitmask; one converter per link (no multi-hop); enum→double bridge loses the tag (ordinal-only, lossy) | `types/logic.md §Links, not polling` | `[ev: corpus B871 §871.4, B871 §871.2-3, B928]` |
| Δ4 | Add BEventService event routing recipe: routing = Baja link graph, no imperative registry; `BEventSource.routeTo(name, routable)` = add consumer as child + BLink(source.event → consumer.process); `BEventFilter` chains; `BEventRecipient` = terminal async via ThreadPoolWorker; uniform `BEvent` envelope; cross-station via `BStationRecipient` (Fox EventChannel); license `tridium:eventService` | `types/logic-authoring.md §Author-side SPIs` | `[ev: corpus B873 §873.1-4]` |
| Δ5 | Add BControlSchedule integration rules: (1) a link to `schedule.in` OUTRANKS ALL tree rules — check before using a linked override; (2) composite priority = FIRST SLOT WINS (positional, no numeric field); (3) ONE Clock.Ticket per transition (no polling); (4) scan horizon = 90 days | `types/logic-authoring.md` (new §Schedule integration) | `[ev: corpus B874 §874.2-4]` |
| Δ6 | Add DDF driver transaction-manager choice rule: pick `defaultComm` (serialized, one-outstanding, Clock.Ticket timeout) for devices that cannot handle overlapping requests; pick `multipleTransaction` (tag-correlated, pipelined) for capable protocols; wrong pick = poll jams or framing errors | `types/logic-authoring.md §Authoring a driver` | `[ev: corpus B909 §909.1-3]` |
| Δ7 | Add BPollScheduler diagnostics: ONE thread per network `"Poll:<networkName>"`; three rate buckets fast=1000ms/normal=5000ms/slow=30000ms; `dibs` = immediate first-poll on subscribe; debugging non-updating proxy point → check BPollFrequency bucket, check `pollEnabled` on network, check thread alive | `types/logic-authoring.md §Authoring a driver` | `[ev: corpus B872 §872.1]` |

## Lessons

- The module-mechanics focus is broad (66 blocks): driver wire-protocol internals dominate by block count but
  yield zero kit rules; genuine kit-actionable content concentrates in the subscription model (B867/B900),
  link converters (B871/B928), event routing overlay (B873), schedule integration (B874), and driver threading
  (B872/B909) — ~10 blocks out of 66.
- The DEFAULT event mask on `TypeSubscriber` (`SELF_EVENTS`) is a biting gotcha: it looks like a value-change
  subscription but fires only on component lifecycle events. This is the kind of silent miss that compiles and
  passes the verify gate; it is not obvious from the name alone.
- The BConverter STRIP pattern (null source → hold last-good) is architecturally correct (preserves last
  known-good state across a type-conversion seam) but operationally dangerous if the author expects
  null/fault to propagate; it violates the "degrade honestly" rule from `logic.md` unless the caller checks
  both source status and converter type before trusting the target value.
- `2026-09-18-corpus-index-refresh-b761-b1028.md` already proposes corpus-index pointer entries for B867,
  B871, B900 (and others). This retro proposes only the SUBSTANTIVE RULE TEXT to add to types/*.md — the two
  retros are complementary, not redundant.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-module-mechanics-deltas.md | kit | 2026-09-18 | pending | 7 |`
