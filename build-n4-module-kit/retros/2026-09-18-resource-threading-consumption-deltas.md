<!-- review-status: folded -->
# 2026-09-18 · kit · resource-threading-consumption-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas).
**Delta count**: 6

---

## What happened

Mined corpus blocks B31 (thread pools per subsystem · GC tuning · I/O buffering · polling limits · engine
thread + HogsPage diagnostic), B729 (timer lifecycle), B730 (rt authoring idioms: execute/changed/timers/
off-thread IO), and B737 (engine thread + watchdog chain + composition vs flat-slot wall) for material
that is absent or only partially expressed in the current kit.

The three previously-folded retros that touched adjacent ground (`process-timers-and-defrost-audit`,
`self-firing-timer-needs-started-not-only-atsteadystate`, `soft-start-staggered-startup`) fully cover
timer arming lifecycle, anchor-vs-reltime patterns, and staged-start defaults — those are NOT re-proposed
here.  What follows are the six distinct gaps left open after those folds.

---

## Evidence

- **B31 §31.1.5** — `$HogsPage` interpretation: spy URL, column semantics, severity thresholds (10 ms /
  50 ms / 200 ms / 1000 ms), and the jstack thread name `"Niagara Engine"`.  Not in any kit file.
- **B31 §31.2.1 table + B15 cross-ref** — empirical polling limits: 1-2 k points @1 s safe; 5 k @5 s
  safe; 5 k @1 s marginal; beyond that, engine overload accumulates.  `logic.md` has "links, not polling"
  in one line but no numbers.
- **B31 §31.3.8** — ParallelGC (JDK 8 default in Niagara) Full GC stop-the-world pauses ALL threads
  including the engine thread; a Full GC of 500 ms–2 s on a 1 GB heap causes timer skips, schedule drift,
  and BACnet COV timeouts.  G1GC (`-XX:+UseG1GC`) reduces the risk but does not eliminate it (Full GC
  evades the pause target).  Absent from `logic.md` and `build-verify.md`.
- **B730 §730.8 + §G2** — `BWorker` is the named framework pattern for off-engine blocking work: post
  `Runnable`s to it, receive results back on the engine thread via `post()/postAsync()`.  `logic.md`
  mentions "off-thread IO via BWorker" in one compound bullet; the concrete recipe (when to reach for it,
  why `Thread.start()` is wrong, the 503-cascade consequence) is absent.
- **B737 §A.3** — watchdog chain: engine discipline (first) → engine watchdog (hung engine) → niagarad
  daemon (process exit + auto-restart with crash-loop backoff).  Key nuance not in any kit file: watchdogs
  recover only a DEAD or HUNG station; a merely SLOW station is invisible to them — discipline is the
  only protection against slow callbacks.  `METHODOLOGY.md` already distinguishes the native
  `EngineWatchdog` from author-level monitors but does not state the dead/hung vs slow distinction.
- **B737 §B.2-B.3** — above ~12-15 flat slots on one BComponent the property sheet and Link picker sprawl;
  group concerns into child BComponents.  `logic.md` L21 states this as doctrine; `verify-module.sh` has
  no automated signal for it (the smell is silently accepted until a code-review).

---

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | Add engine-thread HogsPage diagnostic recipe: spy URL `/spy/sysManagers/engineManager?hogs`, jstack thread name `"Niagara Engine"`, five-tier severity table (<10 ms OK · 10-50 ms attention · 50-200 ms move to driver worker · 200-1000 ms critical · >1000 ms probable blocking I/O), and note that `$HogsPage` resets on station restart | `types/logic.md` — new para after "engine-thread cost = Σ…" in the §RT control engine section | ET-DIAG1 |
| Δ2 | Expand "Links, not polling" with explicit numbers: 1-2 k points @1 s safe; 5 k @5 s safe; 5 k @1 s marginal; above 5 k @1 s engine overload risk accumulates; these are empirical limits from N4 decompilation + B15 (not Tridium-published) | `types/logic.md` — expand existing "Links, not polling" bullet | POLL-LIMITS1 |
| Δ3 | Add GC-induced engine timer drift WARNING: Niagara ships with ParallelGC (JDK 8 default); a Full GC pause (500 ms–2 s on a 1 GB heap) stops the engine thread and can skip a timer callback entirely; timer periods <1 s are at risk; for safety-critical timing add G1GC flags to `nre.properties` (`-XX:+UseG1GC -XX:MaxGCPauseMillis=200`) but note it is a soft target — Full GC can still exceed it | `types/logic.md` — add note in §"Safety fail-modes & timers" after timer defense layer list | GC-TIMER-DRIFT1 |
| Δ4 | Add BWorker off-engine recipe: when a BComponent needs direct hardware/device IO (not via BLink/proxy), never block the engine thread — post `Runnable`s to a `BWorker` child component and deliver results back via `post()/postAsync()`; `Thread.start()` directly is wrong (unmanaged lifetime, no Baja fault propagation); connect to the 503-cascade failure mode: a blocking engine callback stalls Jetty workers that call `.get()` on BComponent state, which saturates the Jetty pool and produces HTTP 503 | `types/logic.md` — new bullet in §engine-thread discipline block | BWORKER-OFFTHREAD1 |
| Δ5 | Sharpen the watchdog chain note: add one sentence to the existing watchdog content stating "the watchdogs recover a DEAD or HUNG station; a merely SLOW station is invisible to them — engine-thread discipline (never block, catch(Throwable)) is the ONLY protection against slow callbacks"; link niagarad daemon (PID poll → process exit → auto-restart with crash-loop backoff) → engine watchdog (hung engine heartbeat) → discipline | `METHODOLOGY.md` §Watchdog note — extend the existing sentence | WATCHDOG-CHAIN1 |
| Δ6 | Add slot-wall WARN check to `verify-module.sh`: for each `.java` file under `-rt/src`, count `@NiagaraProperty` annotations; if any single file has more than 15, emit WARN "flat slot wall on `<ClassName>` (N slots) — consider grouping into child BComponents per L21"; exit 0 (WARN only, not FAIL) to stay compatible with existing modules | `toolbelt/verify-module.sh` — new check block after existing slot checks | SLOT-WALL-LINT1 |

---

## Lessons

1. B31 is NOT yet in the `corpus-index.md` range (that starts at B729); it is a performance-tuning deep-dive from a separate investigation axis.  The HogsPage and GC-timing material are operationally critical but were written months before the kit campaign started.  Indexing B31 as a supplementary corpus reference would make these findable from kit context.
2. "Links, not polling" without numbers is actionable only for people who already know the numbers.  Explicit limits ("1-2 k @1 s") turn a principle into a checkable design constraint.
3. The GC-timer coupling is counter-intuitive to module authors who think of GC as a "JVM concern": a Full GC that pauses for 800 ms looks like a random engine stall on the Workbench spy page without GC logs enabled.  Adding the note next to timer defense layers makes the connection explicit.
4. The 503-cascade failure mode (slow engine → Jetty pool saturation) bridges the threading doc to an observable production symptom.  Tying BWorker to that symptom gives the recipe a stake-in-the-ground motivation.
5. Δ5 (watchdog chain) is deliberately a small targeted addition, not a rewrite; the `METHODOLOGY.md` note is already correct — only the DEAD/HUNG vs SLOW distinction is missing.
6. Δ6 (lint) is the complement to L21 doctrine: doctrine tells authors what to do; the lint tells the review gate when they didn't.

---

**Status**: PENDING — INDEX row appended: `| 2026-09-18-resource-threading-consumption-deltas.md | kit | 2026-09-18 | pending | 6 |`
