# Type: Reading data from module code (queries)

Which read to pick (BQL, NEQL, a history or alarm query, a lease or a subscription), the Java recipe that is safe on
a live station, what each query costs, and the idioms for adding slots. Station-side Java only: a browser or an
off-station client has its own transports (`types/dashboard.md`, `types/frontend-standard.md`). The long BQL
consumer pattern stays in `types/logic-authoring.md` § BQL from code; this card adds the cost model and the history
and alarm reads.

Evidence: `[CERT]` source of the 4.14.0.162 corpus (vineflower copies under `organized/`; `docSource` is the source
Tridium ships with the docs, named where a line is from it); `[CERT-doc]` the developer guide and javadoc copy
4.15.6.20 under niagara-research (`niagara-help/versions/4.15.6.20/`); `[ev: corpus B<n> §…]` a research block;
`[INFER]` the kit's reading, not stated by a source. Nothing here was run on a station, and the Java
snippets were checked against the signatures in the source, not compiled.

## 1. Which read

| You need | Pick | Cost | Avoid |
|---|---|---|---|
| One known component or point | `BOrd.make("station:\|slot:/…").get(base, cx)` | one lookup | a BQL for one row |
| Every component of a type | BQL `select … from <module:Type>` | tree walk (§3) | BQL in a poll loop |
| Components chosen by tag | NEQL, then `\|bql:select …` | visits all | NEQL in a poll loop |
| A history window | `HistorySpaceConnection.timeQuery(h, a, b)` | the window | `where value > x` for years |
| Last N history records | `timeQuery(h, a, null, true)`, stop at N | one descending read | `order by` over all |
| A downsampled history | BQL `HistoryRollup.rollup(…)` on a window | streams the window | rollup of all of it |
| Open or unacked alarms | `from openAlarms` / `from ackPendingAlarms` | index-backed | `where sourceState` |
| One source's alarms, one alarm | `source = '<ord>'` / `uuid = '<uuid>'` | index-backed | a scan with a `like` |
| One fresh proxy value | `BComponent.lease(points, 0)` (§7) | a lease | BQL: it does not subscribe |
| Every change of a slot | `Subscriber` + `unsubscribeAll()` | one subscription each | polling with a query |

## 2. The safe recipe

```java
// cx = the CALLER's Context (servlet request, action invoke). TOP bounds the rows; the try closes the cursor.
BOrd q = BOrd.make("station:|slot:/Drivers|bql:select top 500 name, out.value from control:NumericPoint");
BITable<?> t = (BITable<?>) q.get(Sys.getStation(), cx);
try (TableCursor<?> c = t.cursor()) {
  Column name = t.getColumns().get("name");
  while (c.next()) {
    String n = ((BString) c.cell(name)).getString();      // a history or alarm row: keep r.newCopy(), not r
  }
}
```

- **Pass the user's Context and bound the query.** `BOrd.get()` and `get(base)` resolve with a null Context, so no
  user filters the rows; the Context is what makes a tree walk skip components the user cannot read, and without a
  user the cursor is the raw walk. `[CERT]` `bql-rt` `com/tridium/bql/collection/BBogCollection.java:68-72`;
  `[CERT-doc]` `docSource` `baja` `javax/baja/naming/BOrd.java:197-217`. `toolbelt/lint-null-context-write.sh` WARNs
  on it.
- **Close every cursor and connection with try-with-resources.** `HistorySpaceConnection` and `AlarmSpaceConnection`
  extend `AutoCloseable`; "failing to close a cursor" is a documented leak. `[CERT-doc]`
  `devguide-clean/collections.txt:101`; `history-rt` `javax/baja/history/HistorySpaceConnection.java:22-23,251-255`.
- **Bound it three ways:** `top N` right after `select`; a `timestamp` window on a history or alarm; `depth = n` or
  `stop` on a tree walk. BQL has no `skip` or `offset`: the tokenizer knows `depth`, `stop` and `top` and no other
  limiter. `[CERT]` `bql-rt` `com/tridium/bql/compiler/BqlTokenizer.java:253,323,328`. `stop` is a documented
  keyword (`select slotPath, out from schedule:AbstractSchedule stop`, `[CERT-doc]`
  `devguide-clean/bqlExamples.txt:37`).
- **A cursor reuses one record object.** A history or alarm record you keep past `next()` is `newCopy()`; a list of
  the bare `cursor.get()` ends up holding the last row N times. `[CERT-doc]` `devguide-clean/history.txt:34`;
  `docSource` `alarm-rt` `javax/baja/alarm/AlarmDbConnection.java:49-50`.
- **Run it off the engine thread.** `BFileAlarmDatabase.getDbConnection` logs "called from Nre:Engine Thread"
  ("Potential Performance Degradation") at FINE, so the platform itself marks the engine thread as the wrong place
  for an alarm read. `[CERT]` `alarm-rt` `com/tridium/alarm/db/file/BFileAlarmDatabase.java:94-99`. Read from a
  servlet thread or a `BSimpleJob` (`types/logic-authoring.md` § Background jobs); never from `changed()`,
  `execute()`, `atSteadyState()` or an `ASYNC` action, which runs on the engine thread (`types/actions.md` § 2).
  For BQL and history the same advice is `[INFER]` (nothing in the code names the engine thread).
  `[ev: corpus B806 §806.1]`
- **Do not build the query text from user input** (`lint-bql-string-concat.sh`, `types/security.md` § 4).

## 3. Cost table: what is indexed, what scans

| Query shape | What the station does | Evidence |
|---|---|---|
| Tree BQL (`from <module:Type>`) | walks every component below the base; `where` filters after | below |
| `where …` on any extent | `BFilteredTable` tests each row after the read | `SelectQuery.java:62-64` |
| `order by`, `distinct`, aggregates | materialize the table, then `top` cuts it | `SelectQuery.java:66-81` |
| `top N` | stops after N rows; the work before an `order by` is done | `BTopTable.java:57-63` |
| History BQL, no predicate | `scan`: reads the whole history | `BHistoryDbTable.java:249-250` |
| History BQL, `timestamp` range | `timeQuery(min, max)`; other columns filter per row | `:252-261` |
| History BQL, `order by timestamp desc` | descending read pushed down; `order by` still sorts | `:236-246` |
| Alarm `openAlarms`, `ackPendingAlarms` | returns the index-backed set | `AlarmDbConnection.java:59-62` |
| Alarm `uuid =`, `source =` (one) | point lookups, or the source's records | `:70-90` |
| Alarm `timestamp` range | `timeQuery(min, max)` | `:92-104` |
| Any other alarm `where` | `scan` of the whole alarm database | `:64-66,94-95` |
| NEQL | iterates every component of the scope | below |
| Rollup | folds the cursor into buckets; maybe one extra full pass | below |
| Archive provider | merged unless excluded; limit 50000 per query | below |
| Alarm class permission | row loaded first, then dropped: no IO saved | below |
| BOX paging (`of`, `lm`) | re-resolves the ORD, skips `of` rows each call | below |

Rows are `[CERT]`: `bql-rt` `com/tridium/bql/SelectQuery.java` and `collection/BTopTable.java`, `history-rt`
`com/tridium/history/db/BHistoryDbTable.java` (vineflower copies); the alarm rows cite `docSource` `alarm-rt`
`javax/baja/alarm/AlarmDbConnection.java`, the source Tridium ships with the docs, whose line numbers differ from
the decompiled copy. The bullets below give the other paths and lines.

- **Tree BQL:** `BogCursor` walks properties depth-first and `depth` is its only counter, so the work is the number of
  components visited, whatever the `where`; with a user in the ORD target it is wrapped in `AccessSlotCursor`.
  `[CERT]` `bql-rt` `com/tridium/bql/cursor/BogCursor.java:34-48,112-120`; `BBogCollection.java:56-73`. A projection
  (`select a, b`) shapes the output; the walk is the same (`SelectQuery.java:83-85,107-114`).
- **History:** only `timestamp` narrows the read, and `order by timestamp` is the one ordering pushed down; any other
  column is evaluated per row after the read. Nothing in `bqlQuery` caps the rows. `[CERT]` `history-rt`
  `com/tridium/history/db/BHistoryDbTable.java:232-262`.
- **Alarm:** `doBqlQuery` is `synchronized` per connection and the cursor may hold more than the query asked for; the
  query engine re-filters. `[CERT-doc]` `docSource` `alarm-rt` `javax/baja/alarm/AlarmDbConnection.java:43-52`. A
  `where sourceState = 'offnormal'` has no index, and with `order by` it loads the whole database before any cap.
- **Alarm permissions:** the cursor takes the user from the thread (`BUser.getCurrentAuthenticatedUser()`), not from
  the Context, and loads each record before it tests operator read on the alarm class. A servlet thread has a user;
  an engine or job thread has none, so nothing is filtered there. `[CERT]` `alarm-rt`
  `com/tridium/alarm/db/file/AlarmStore.java:898,935-957`; `getDbConnection(cx)` itself needs operator read on the
  alarm database for the Context's user (`BFileAlarmDatabase.java:91-92`).
- **NEQL:** no tag index in the handler; the scope needs operator read. NEQL `like` is a regex, BQL `like` a glob.
  `[CERT]` `neql-rt` `com/tridium/neql/component/BNeqlComponentQueryHandler.java:31-50`; `[CERT-doc]`
  `devguide-clean/neql.txt:57,79`, `devguide-clean/bqlExamples.txt:23`.
- **Rollup is computed on the fly:** the constructor reads `startTime` and `endTime` from the table facets and, when
  either is missing, walks the whole cursor once to find them. It saves bytes, not scan time: narrow the window first.
  `[CERT]` `history-rt` `com/tridium/history/rollup/RollupCursor.java:42-78`. `[ev: corpus B33 §33.11.5]` calls it
  a pre-aggregated table; the code is a cursor wrapper.
- **Archive:** `maxArchiveResultsPerQuery` is `newProperty(0, 50000, …)`; a Context from
  `HistoryQuery.makeArchiveQueryLimitContext(cx, n)` caps the archive rows, and `makeExcludeArchiveDataContext(cx)`
  skips the archive. `[CERT]` `history-rt` `javax/baja/history/db/BArchiveHistoryProvider.java:55,69`;
  `javax/baja/history/HistoryQuery.java:339-361`.
- **BOX paging:** `encodeTableContents` resolves the ORD, then `stream().skip(of).limit(lm)`: page N re-iterates N ×
  size rows, so keep `lm` large and `of` small, or page by time (§5). `[CERT]` `box-rt`
  `com/tridium/box/BOrdChannel.java:281-316`.
  The HTTP table endpoints and the BOX limits are in `types/data-access.md` §§ 2, 7.

## 4. History recipes

```java
BHistoryService hs = (BHistoryService) Sys.getService(BHistoryService.TYPE);
BAbsTime end = Clock.time();
BAbsTime start = end.subtract(BRelTime.makeHours(6));
try (HistorySpaceConnection conn = hs.getDatabase().getConnection(cx)) {
  BIHistory h = conn.getHistory(BHistoryId.make("MyStation", "Temp1"));      // null when the history does not exist
  if (h == null) return;
  try (TableCursor<BHistoryRecord> c = conn.timeQuery(h, start, end).cursor()) {
    while (c.next()) {
      BHistoryRecord r = c.get();                    // the same object every pass: r.newCopy() to keep it
    }
  }
}
```

- **A window: `timeQuery`.** The result includes both endpoints, a `null` end is open, and the fourth argument reads
  descending. `[CERT-doc]` `history-rt` `javax/baja/history/HistorySpaceConnection.java:119-155`. This is the
  guide's own N4 recipe (`[CERT-doc]` `devguide-clean/history.txt:223-245`).
- **The last N records:** `timeQuery(h, start, null, true)`, count rows, leave the loop after N (the `try` closes
  the cursor). `getLastTimestamp(h)` and `getRecordCount(h)` are on the same connection (`:55,71`). `[INFER]` that a
  cursor closed early stops reading; the descending read itself is `[CERT-doc]`.
- **One connection per batch, not per record.** `getConnection` makes a new connection on each call, so open one per
  batch and do not share it across threads (`[INFER]` for the sharing rule). `[CERT]` `history-rt`
  `javax/baja/history/db/BHistoryDatabase.java:53-61`; `[ev: corpus B33 §33.4]`. `niagara.history.localDb.lingerTime`
  defaults to 5 min (`BHistoryService.java:106-107,470`); `[INFER]` that it is the idle time before a table closes.
- **A history ORD in BQL:** `history:/<device>/<history>?period=<id>;start=<time>;end=<time>` with `;` between the
  parameters and `period` one of `today, yesterday, lastWeek, last7Days, monthToDate, lastMonth, yearToDate,
  lastYear`; `^name` is the local device. `[CERT-doc]` `bajadoc-clean/javax/baja/history/HistoryQuery.txt:7-19`. Then
  `|bql:select timestamp, value from …` runs over the window. Resolve it with a base of `null` and the caller's
  Context: `BOrd.make("history:" + id + "|bql:select *").get(null, cx)`.
  `[CERT-doc]` course `BProcessHistoryExample.java:91-96` (unbounded as shipped: add `?period=` or `top`).
- **Rollup** is a BQL function, not the `com.tridium` class (private API). Window first, then roll up; it returns
  every column and the interval is a `RelTime` in milliseconds, or a `RollupInterval` (`none … yearly`,
  `BRollupInterval.java:12`):

  ```
  history:/demo/BldgKw?period=lastWeek|bql:history:HistoryRollup.rollup(baja:RelTime '86400000')
  ```

  `[CERT-doc]` `guides-clean/EngNotes/WeeklyElectricalDemandReport.txt:25-29`;
  `devguide-clean/releaseNotes.txt:3379-3389`.
- **A local history read has no row cap:** nothing in `bqlQuery` or `timeQuery` limits the rows, so in Java the cap is
  the one you write (a window, `top`, or a counter). `[CERT]` `BHistoryDbTable.java:232-262`.

## 5. Paging

BQL has no `skip`, so a page is a new query from where the last one stopped. Read a window or N rows, remember the
last timestamp, and ask again from that timestamp plus 1 ms (the endpoints are inclusive, so the plus avoids a
repeat):

```java
BAbsTime from = start;
while (true) {
  BAbsTime last = null; int n = 0;
  try (TableCursor<BHistoryRecord> c = conn.timeQuery(h, from, end).cursor()) {
    while (n < PAGE && c.next()) { handle(c.get()); last = c.get().getTimestamp(); n++; }
  }
  if (n < PAGE) break;                               // a short page is the last page
  from = last.add(BRelTime.make(1L));                // restart one millisecond after the last record seen
}
```

- **Two records in one millisecond:** if a page ends between them the second is skipped; dedupe by timestamp in the
  caller, or cut pages on a time window (one day) instead of a row count. `[INFER]`
- **Incremental ("since the last run"):** keep the last timestamp in a slot or a file and start there; `?delta=true`
  is the ORD form. `[CERT-doc]` `bajadoc-clean/javax/baja/history/HistoryQuery.txt:7-10`.
- **For a tree-walk table** no re-query exists: bound it with `top`, `depth` or a narrower base ORD.

## 6. Alarm recipes

```java
BAlarmService svc = (BAlarmService) Sys.getService(BAlarmService.TYPE);
try (AlarmDbConnection conn = svc.getAlarmDb().getDbConnection(cx);
     Cursor<BAlarmRecord> cur = conn.getOpenAlarms()) {        // getOpenAlarms declares IOException
  int n = 0;
  while (n < 200 && cur.next()) {
    BAlarmRecord a = (BAlarmRecord) cur.get().newCopy();       // a copy, because the cursor reuses its record
    n++;
  }
}
```

- **Index-backed paths:** `from openAlarms`, `from ackPendingAlarms`, `uuid = '…'`, `source = '<ord>'` (exactly one
  source) and a `timestamp` range; everything else scans. The Java methods are `getOpenAlarms`, `getAckPendingAlarms`,
  `getAlarmsForSource`, `getRecord(BUuid)`, `timeQuery`. `[CERT]` §3 rows; `[CERT-doc]` `docSource` `alarm-rt`
  `javax/baja/alarm/AlarmDbConnection.java:195-240`. The guide's N4 open-alarm loop is
  `[CERT-doc]` `devguide-clean/alarm.txt:308-315`.
- **In BQL:** `alarm:|bql:select * from openAlarms` (`devguide-clean/alarm.txt:333`). Filter the state in Java
  first and cap the reply after it: a `top n` before the filter can drop a live offnormal alarm.
  `openAlarms` is every record that is not both acked and normal, and not both acked and alert
  (`[ev: corpus B1237 §1237.8]`); the archive is `alarm:archive|bql:…` (`alarm.txt:320-327`) and is not part of
  `openAlarms`.
- **One alarm by id:** `alarm:|bql:select * where uuid = '<uuid>'` (course `BAlarmExample2.java:94-97`).
- **Cheap windows:** a `timestamp` predicate or `timeQuery(start, end)` stays on the time index; `sourceState`,
  `ackState` and `alarmClass` conditions do not. Prefer `from openAlarms` and test the state in Java. `[INFER]` that
  this beats the `order by` form of `types/dashboard.md`; the paths are `[CERT]` §3. `[ev: corpus B1237 §1237.10]`
- **The file alarm database defaults to `capacity` 500 records**, so it is small: the cost lever is the scan shape.
  `[CERT]` `alarm-rt` `com/tridium/alarm/db/file/BFileAlarmDbConfig.java:21-23`.

## 7. Fresh values: lease or subscribe

- **BQL never subscribes.** A proxy point read through a query is as fresh as its last subscription. Hold the rows
  (`stream().toArray(BControlPoint[]::new)`), then `BComponent.lease(points, 0)` for a one-off snapshot, or
  `Subscriber.subscribe(points, 0, cx)` with `unsubscribeAll()` when done for a stream. `[CERT-doc]` course
  `BPointsQuery.java:162-211`.
- **The Java lease is 60 000 ms.** `lease()` and `lease(depth)` use the space default, 60 s; `lease(depth, millis)`
  sets it. A second lease renews to the later expiry, and a sweeper thread unsubscribes expired leases every 5 s, so
  an unsubscribe can come up to 5 s late. `[CERT]` `baja` `javax/baja/sys/BComponent.java:249-283`,
  `javax/baja/space/BComponentSpace.java:121-123`, `com/tridium/sys/engine/LeaseManager.java:32-34,52,106`. The
  BajaScript default is 10 s (`[CERT-doc]` `devguide-clean/jsdoc/bajaScript-ux/baja.Component.txt:6775`), a BOX
  server session expires after 90 s unless renewed (`[CERT]` `box-rt`
  `com/tridium/box/BServerSession.java:85-86,410-413`), and `BoxService.serverCacheConfig` is `medium` by default
  (`BBoxService.java:131`).
- **Do not read the leased proxy point in the callback that leased it.** `lease` calls `Subscriber.subscribe` on your
  thread, but a driver is told to read asynchronously and to answer with `readOk()` or `readFail()`, so the value of
  a proxy point arrives later. Read it on a later tick, or react to the `Subscriber` event. `[CERT]` `baja`
  `com/tridium/sys/engine/LeaseManager.java:32-34`; `[CERT-doc]` `docSource` `driver-rt`
  `javax/baja/driver/point/BProxyExt.java:851-859`. The course's `doRunQueryAndLease` reads right after `lease`
  (`BPointsQuery.java:170-174`): fine for a plain control point, not for a proxy. `[ev: corpus B1420 §1420.6]`
- **Lease for a snapshot, subscribe for a stream.** A lease needs no teardown; a `Subscriber` needs
  `unsubscribeAll()` in `stopped()` (`types/logic-authoring.md` § Subscribe / unsubscribe symmetry rule).

## 8. Adding slots from code

- **`add("name", v)` throws `DuplicateSlotException` when the slot exists; `add("name?", v)` appends a number to
  make a unique name; `add(null, v)` generates one.** With `?` the name stays unescaped (`"hello world?"`). Without
  it a name with a space or a non-ASCII letter throws `IllegalNameException`: wrap it in `SlotPath.escape(...)`. A
  value that already has a parent throws `AlreadyParentedException` (copy it first); `null` is never a property
  value. `[CERT-doc]` `docSource` `baja` `javax/baja/sys/BComponent.java:832-870`; course
  `BSlotAdder.java:150-181`.
- **A link:** `linkTo(source, sourceSlot, targetSlot)` makes the `BLink`, adds it as a dynamic transient property
  with a generated name and activates it. A hand-made `add("link", …)` throws `DuplicateSlotException` on the second
  link added to that component; a name ending in `?` (`"link?"`) avoids it. `[CERT-doc]`
  `javax/baja/sys/BComponent.java:1762-1782`; using `"link?"` for a link is `[INFER]` from the `?` rule
  `[ev: corpus B1231 §1231.8]`. `isLinkTarget(slot)` tells whether a slot already is a link target (`:1757`).
- **Existence tests and `removeAll()`** are in `types/logic.md` ("Slot-API traps": `getSlot(name) != null`, because
  `get(name)` rejects an action) and `types/logic-authoring.md` § Dynamic-slot orphan hygiene.
- **Set a status struct once.** Writing `out.value` and `out.status` as two sets propagates twice; build one
  `BStatusNumeric` and set the slot once. `[INFER]` `[ev: corpus B1231 §1231.11]`

## 9. Snippet bank: the course `ords/*` classes

The Tridium course ships ten classes and a test twin for each (`BTestNgStation` station tests, not pure tests; see
`types/moduleTest.md`). Copy the shape, then fix what the course leaves open:
`Slide Code/slides/slides-rt/src/com/tridiumuniversity/slides/ords/` (tests in `srcTest/test/…/ords/`). `[CERT-doc]`

| Class | Shows | Change before you ship it |
|---|---|---|
| `BBqlExample` | `select name, out.value from control:NumericPoint`, columns by name | add `top` or a narrower base |
| `BBqlExample2` | three loops over one cursor: `next()`, for-each, `stream()` | the extent is unbounded: narrow it |
| `BPointsQuery` | a stored `BOrd` query; lease and `Subscriber` over the rows | read after the lease (§7) |
| `BAlarmExample` | `alarm:\|bql:select *`, `.get(null, cx)`, `newCopy()` per row | `from openAlarms` or a window |
| `BAlarmExample2` | `where uuid = '%s'` from `getAlarmUuid().encodeToString()` | none: `uuid` is index-backed |
| `BProcessHistoryExample` | `history:<id>\|bql:select *`, `newCopy()` per row | add `?period=` or `top` (§4) |
| `BProcessHistoriesExample` | `history:\|bql:select *` lists `BHistoryConfig` rows | the rows are configs |
| `BNeqlExample` | `neql:n:history`, then `entity.tags().get(Id.newId("n","history"))` | NEQL scans: not in a loop |
| `BOrdExample` | `resolve(base, cx)`, `canRead()` before `get()`, four `slot:` forms | none |
| `BUserScheme` | a custom `userDemo:` ORD scheme, `canRead` checks | `logic-authoring.md` § ORD scheme typology |

`[ev: corpus B1237 §1237.3]`
