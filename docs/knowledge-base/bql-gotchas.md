# BQL Gotchas — Niagara N4.14

Confirmed bugs and anti-patterns in BQL on Niagara N4.14.
Sourced from production smoke tests on Windows station (2026-05-17).

---

## Bug 1: `ackState = 'ackPending'` is transitory — `'unacked'` is the stable state

**Symptom**: Alarm appears with `ackState = 'ackPending'` in a BQL query for
a moment, then disappears or changes back to `'unacked'`.

**Cause**: `ackPending` is a transient in-flight state during the acknowledgment
handshake. It does NOT mean the ack succeeded — it means the station is processing
a request that may or may not commit. Querying `ackState = 'ackPending'` in a
polling loop will produce intermittent false positives.

**Workaround**: Filter on `ackState = 'unacked'` (the stable pre-ack state)
or `ackState = 'acked'` (the stable post-ack state). Never poll on `ackPending`
as a reliable indicator of anything.

---

## Bug 2: `sourceState` is a frozen per-record snapshot, not current source state

**Symptom**: BQL returns `sourceState = 'normal'` for an alarm, but the source
component is clearly in a fault state in Workbench.

**Cause**: `sourceState` in a BQL alarm record is a snapshot taken at the moment
the alarm was generated, not the current state of the source component. The field
is frozen at alarm-creation time and does NOT update as the source changes.

**Workaround**: To get the current state of the alarm source, query the source
component directly (via `baja.Ord.make('station:|slot:/path/to/component')`)
rather than reading `sourceState` from the alarm record.

---

## Bug 3: BQL OR precedence + time filter placement causes silent misqueries

**Symptom**: A BQL WHERE clause combining OR conditions with a time filter
returns unexpected results — either too many records or too few.

**Cause**: BQL OR has non-intuitive precedence when combined with AND and
comparison operators. A clause like:
```
WHERE ackState = 'unacked' OR ackState = 'ackPending' AND timestamp > $T
```
binds the AND first, making it equivalent to:
```
WHERE ackState = 'unacked' OR (ackState = 'ackPending' AND timestamp > $T)
```
which includes ALL unacked alarms regardless of timestamp.

**Workaround**: (1) Always parenthesize OR groups explicitly:
```
WHERE (ackState = 'unacked' OR ackState = 'ackPending') AND timestamp > $T
```
(2) Place the time filter LAST in the WHERE clause so it applies to the full
filtered set, not just the last OR branch.

---

## CRITICAL pattern: The only proven persistent-ack path in N4.14

**Problem**: `BAlarmService.ackAlarm(BAlarmRecord)` where `rec` came from a
BQL cursor is a **silent no-op** in Niagara N4.14. The call returns HTTP 200
and reports `ackedCount > 0`, but the ack does NOT persist to the alarm database.

**Root cause**: The `BAlarmRecord` returned by a BQL cursor is a detached snapshot.
The ack is applied to the snapshot in memory — it is never propagated back to the
live alarm DB. Confirmed by production smoke test (2026-05-17): 413 records
"acked" according to server response, all still showing `unacked` on reload.

**Sibling-module risk**: `angeles`, `casino`, `sanluis`, and `casinod` all
copy-paste the same broken Java pattern without a smoke test. Assume broken
until proven otherwise with an end-to-end state-transition check.

**Proven working path — use this**:
```javascript
// BajaScript (frontend) — the ONLY path confirmed to persist in N4.14
baja.Ord.make('alarm:').get().then(function(svc) {
    return svc.ackAlarms({ ids: uuidStringArray });
});
```
Collect UUIDs server-side (Java BQL cursor is fine for READ-ONLY collection),
return them to the frontend as JSON, and execute the ack via BajaScript.

**Engram source**: obs #1788 (`niagara/balarmservice-ackalarm-detached-snapshot`),
project `honeywell-mx60-chihuahua`.

---

← Back to [GOTCHAS index](../GOTCHAS.md)
