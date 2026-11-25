# Type: Data access from a browser or an off-station client (channel, limits, cache)

Which transport a dashboard or an off-station consumer should use to read and write station data, the limits each
one has in code, and what the station already does to static files, images and audio. Station-side Java reads (BQL,
history, alarms, leases) are `types/queries.md`; the servlet-SPA recipe is `types/dashboard.md`; BajaScript from a
page is `types/dashboard.md` § Reading the station directly with BajaScript. The practitioner decision table for
every channel (including Fox, MQTT and protocol servers) is `docs/n4-data-access-guide.md` §1 in niagara-research.

Evidence: `[CERT]` source of the 4.14.0.162 corpus under `organized/` (vineflower copies; `4.15` not checked unless
named); `[CERT-doc]` the guides and developer guide 4.15.6.20 (`niagara-help/versions/4.15.6.20/`); `[ev: corpus
B<n> §…]` a research block; `[INFER]` the kit's reading. Nothing here was run on a station.

## 1. Which channel

| Job | Pick | Avoid |
|---|---|---|
| Live values on a page, a few tabs | servlet JSON + poll, or BajaScript `Subscriber` | oBIX BQL as a poll |
| Wallboard, many unattended screens | one off-station cache, one oBIX reader (§6) | BOX or a watch per screen |
| Bulk list, rarely | `/ord/<ORD>\|bql:…\|view:file:ITableToCsv`, or §7 | BOX tree crawl; BQL on a timer |
| History export | oBIX `history query` + stored watermark (§6), or §7 | rebuilding from snapshots |
| Alarms to another system | oBIX `alarmFeed` inside a watch (§6) | the oBIX alarm `query` op |
| Write a value | servlet or RPC with the user's Context | null-Context `set` |
| Call server logic from a UI | `@NiagaraRpc` over `baja.rpc` (§8) | `unrestricted` with no check |

Why: a BOX WebSocket tab pins a thread (§2); `alarmFeed` is lossless while the watch lives; RBAC and audit need the
user's Context (`dashboard.md`, `servlet-write.md`).

[research block B1271 §1271.3] [research block B1275 §1275.2] `[CERT-doc]` the job-by-job table
`docs/n4-data-access-guide.md` §1. A row is a starting choice, not a measurement: no latency was measured here.

## 2. BOX limits (WebSocket and HTTP)

| Limit | Value | Where |
|---|---|---|
| WebSocket idle timeout | **60 000 ms**, system property `box.ws.idleTimeout` | `BoxWebSocketServlet.java:16,32` |
| Text message / binary message | 262 144 B / 131 072 B (buffers 65 536 B) | `BoxWebSocketServlet.java:17-36` |
| Reassembled envelope | 1 048 576 B, `box.mux.maxEnvelopeSize` | `mux/BoxEnvelope.java:42-44` |
| Server session expiry / poll | 90 000 ms (renewed per request) / 2000 ms | `BServerSession.java:83-86,410-413` |
| Broker threads | core 2, max `Integer.MAX_VALUE`, one per WebSocket session | `BServerSession.java:97-101,531-545` |
| Licence `session.limit` | `none` unless the licence says otherwise | `BBoxService.java:693-695` |
| Jetty workers (every HTTP channel) | `maxThreads` 30 | `BJettyWebServer.java:271` |

All `[CERT]` under `box-rt` and `jetty-rt`. Reading them:
- **A BOX WebSocket tab costs a thread and a broker.** `initEventHandler` makes the `BrokerPoller` only for a v2
  (WebSocket) session, and the pool has no upper bound (`BServerSession.java:310-316`). Few tabs are fine; many
  unattended screens are not. No tab count is a station limit `[INFER]`: measure on the target.
  [research block B1275 §1275.4]
- **A silent socket closes after 60 s.** Do not rely on one that carries no traffic for longer; reconnect and
  resubscribe. An older corpus note gives 300 000 ms: the code default is 60 000 (4.15 not checked). BajaScript's
  event-mode poll is 30 000 ms against a default of 2500 (`[CERT-doc]` `module-js/bajaScript-ux/rc/comm.js:42-43`),
  which is consistent with a 60 s limit `[INFER]`.
- **A frame over the limit is the caller's problem:** ask for fewer rows per call (`limit`) before raising a property.

## 3. Compression

- `WebService.gzipEnabled` is **false** by default and a hidden slot (`flags = 4`, `BWebService.java:171-174,262`).
  `[CERT]`
- When it is on, Jetty's `GzipHandler` is added with an include list: `text/plain`, `text/html`, `text/css`,
  `text/javascript`, `application/xml`, `application/xhtml+xml`, `application/rss+xml`, `application/javascript`,
  `application/x-javascript`, `application/json` (`BJettyWebServer.java:290-301,780-784`). `[CERT]` So a JSON endpoint
  and a `/box/query` response (`text/plain`) are eligible and an image never is. oBIX (`text/xml`) and an `/ord` CSV
  (`text/csv`) are not on the list, so they stay uncompressed even with gzip on `[INFER]`: Jetty's include-list
  semantics were not re-read.
- **Do not depend on it.** The station default is off, so a servlet that wants small JSON compresses itself
  (`Accept-Encoding` check, `GZIPOutputStream`) and a bundle ships pre-compressed (`dashboard.md` § Static serving).
  `[INFER]` for the advice; the pre-compressed twin is in the kit already.
- `FileServlet` sets `Content-Encoding: gzip` only when the file it serves is itself a gzip file type and the client's
  `Accept-Encoding` contains `gzip` (`FileServlet.java:198-205`). `[CERT]` It does not compress on the fly.
- No `gzip`, `deflate` or `compress` appears anywhere in `box-rt`, so BOX frames are not compressed by BOX
  `[CERT negative, scoped to box-rt]`; whether Jetty negotiates WebSocket `permessage-deflate` was not checked.

## 4. Static files and images: the 30-day cache

`/module/*` and `/file/*` are `com.tridium.web.servlets.FileServlet`
(`web/web-rt/vineflower/WEB-INF/web.xml:94,103,121,125`). Its headers, as coded (`FileServlet.java:240-252,270-277`):

1. `Cache-Control: no-cache, no-store` when web dev is on, when system property `niagara.web.addExpires` is not
   `true` (default `true`), or for a dynamic file.
2. `private, must-revalidate, max-age=2592000` (30 days) when the file is in a module or zip space, **or** its ORD
   matches `WebService.cacheConfig.cachedFileExtensions`: `enabled` true and `png,jpg,gif,svg` by default
   (`javax/baja/web/BCacheConfig.java:10-23`; the pattern is read once in `init`, `FileServlet.java:72-95`).
3. `private, must-revalidate, max-age=0` for every other file.

- **No ETag.** Revalidation is `Last-Modified` and `If-Modified-Since` at one-second granularity; a request whose date
  is not older than the file's second gets `304` (`FileServlet.java:208-232`). `[CERT]` A case-insensitive whole-word
  search for `ETag` and `If-None-Match` over `web-rt`, `jetty-rt` and `box-rt` Java returns nothing
  `[CERT negative, scoped]`.
- **A station image replaced under the same name is not fetched again for up to 30 days** by a browser that has it
  (`png`, `jpg`, `gif`, `svg` by default). Give each revision a new file name, or a changing query: the servlet
  ignores the query, so `?v=` is only a cache key. `jpeg`, `webp`, `ico` and `bmp` are not in the default list and
  revalidate on every request.
  `[CERT]` for the rule, `[INFER]` for the workaround; changing `cacheConfig` at runtime was not checked.
- **A restored or `cp -p` file can keep the old image:** if its mtime second is not newer than the browser's copy the
  answer is `304` (`:219-228`). `[CERT]`
- **A module image keeps its URL across a module upgrade** when the page writes `/module/<mod>/…` with no version, so
  it is stale for up to 30 days. The platform avoids that for its own URLs: `WebUtil.toUri` writes
  `/module/…?version=<registry last build time>` (`WebUtil.java:108`, `RequireJsUtil.java:69,222`). `[CERT]` Ours
  carries `?v=<module version>` (`dashboard.md` § cache-bust).
- **Your own servlet is not `FileServlet`:** it sets its own headers (`toolbelt/lint-servlet.sh` `cache-nofinger`).
- **Budget** for images is not a Tridium number; the kit's starting points are in `frontend-standard.md` (images ship
  as separate cacheable files) and `rc-scan.sh` `datauri-budget`. `[ev: corpus B38 §38.7]`

## 5. Audio

- **Resolve through the platform, not a hand-built URL:** `baja.Ord.make(ord).get()` gives a file whose
  `getReadUri()` is `/ord/<path>|view:web:FileDownloadView` for a station file (`BOrdChannel.java:106-117`), served by
  the same `FileServlet` and the same permission check. `[CERT]`
- **Start it from a user gesture and read the promise `play()` returns.** Tridium's console calls `new Audio(uri)` and
  `play()` and drops the promise (`webEditors-ux rc/wb/audio/AudioManager.js:54,61`), so a browser that blocks
  autoplay stays silent with no error `[CERT]` for the platform code; the autoplay policy itself is `[INFER]`
  (general browser behaviour, not read here).

  ```javascript
  var unlocked = false;
  button.addEventListener('click', function () { unlocked = true; });      // a user gesture
  function beep(ord) {
    if (!unlocked) { return Promise.resolve(false); }
    return baja.Ord.make(ord).get().then(function (f) {
      return new Audio(f.getReadUri()).play().then(function () { return true; }, function () { return false; });
    });
  }
  ```
- **Ship WAV:** "the only format that works in both Workbench and in a browser is wav" (`[CERT-doc]`
  `guides-clean/Alarms/alarm-AlarmUxConsole.txt:25`); MP3 and OGG play in a browser only.
- **Host it as `module://<mod>/rc/sounds/x.wav`** (module space: the 30-day cache of § 4, so version the name) or
  `file:^sounds/x.wav` (needs operator read; `wav` is not in the default cached list, so each play revalidates).
  `[CERT]` for the cache rules, `[INFER]` for the sizes (a short alert tone, roughly 100 KB at most).
- An unattended kiosk needs one operator tap or a browser started with an autoplay allowance `[INFER]`.

## 6. Off-station consumer: an oBIX Watch

For a wallboard, a cache or an analytics reader: one reader, not one watch per screen. Needs the `tridium:obixDriver`
`export` licence and a read user (`types/security.md` § 2.5). [research block B1275 §1275.2]
1. **Make once:** `watchService/make`. The watch's lease is the service's `defaultLeaseTime`, **30 s**
   (`BObixWatchService.java:35-44,91`); the watch's own default is 15 s (`BObixWatch.java:75`). `[CERT]`
2. **Add the hrefs, then `pollChanges` on a fixed period** well inside the lease; every op renews the lease
   (`BObixWatch.java:151-160`). `pollChanges` is a short poll that returns at once (`:253-264`): the period sets the
   latency. A value that flips twice between polls is reported once. `[CERT]`
   [research block B1275 §1275.2]
3. **Delete on exit** (`delete` op, `BObixWatch.java:67-81`). The expiry sweep runs only when some client calls
   `/obix/watchService/…`, so an abandoned watch keeps its point subscriptions. `[CERT]`
   [research block B1275 §1275.2]
4. **Alarms and history rows:** add `~alarmFeed` (under the alarm service or one alarm class,
   `BAlarmServiceAgent.java:241-243`) or `historyFeed` to the watch: a database-backed stream with no collapsing.
   Keep the watermark on the client: the position lives in the watch and is lost with it.
   [research block B1275 §1275.3]
5. **Check the body, not the HTTP status:** an error comes back as `<err>` with the servlet's default status.
   `[CERT]` [research block B1272 §1272.4]
6. **Read the anchor slots:** a servlet-derived key (an elapsed or remaining time) does not exist in oBIX
   (`types/dashboard.md` § Facade contract).

## 7. A table over HTTP: `/box/query` and `/ord`

- **`GET /box/query/data/<ORD>?offset=&limit=&descending=&sortColumn=&filter=`** streams one JSON record per line,
  chunked `text/plain` (`QueryServlet.java:72-74`). It refuses unless `target.canRead()` (`:174-175`) and sends no
  cache headers. `limit` defaults to **-1**, so pass one (`:90`). A sort on `timestamp` of a history is read in
  order; any other sort buffers `offset + limit` rows in a `TreeSet` (`:179-184,202-207`). A GET needs no CSRF token;
  the two `POST` forms do (`:118-131`). `[CERT]` [research block B1270, code §1270.3a]
- **`GET /ord/<url-encoded ORD>` with an exporter view**, such as `…|bql:select …|view:file:ITableToCsv`, returns
  CSV; a read user can run any BQL this way, so keep it to a service account (`types/queries.md` § 3 for the cost).
  [research block B1274 §1274.5]
- Both are pulls of one query: bound it first (`types/queries.md` § 2).

## 8. Calling server logic: `@NiagaraRpc` has two wires

| | `/rpc` (web) | BOX `sys/rpc` (box) |
|---|---|---|
| Client | any HTTP client with a session and a CSRF token | `baja.rpc(ord, method, …)`, `Component.rpc(…)` |
| Server | `NiagaraRpcServlet.java:31,57` | `BSysChannel.java:65,200-201` |
| Context given | a request Context, the last argument | `new BasicContext(op.getUser(), op.getLanguage())` |
| Gate | the CSRF filter on `/rpc/*` (`web.xml:44-55`, same file) | the BOX session |

`baja.rpc` goes over **BOX**, not `/rpc` (`module-js/bajaScript-ux/rc/sys.js:580`, `comm.js:740-745`,
`[CERT-doc]`). Both wires call the same `NiagaraRpcUtil.rpc` (`NiagaraRpcServlet.java:55`,
`BSysChannel.java:200`), which throws `NoSuchMethodException` when the method does not list the transport
(`baja` `com/tridium/util/NiagaraRpcUtil.java:131-132`), so `transports` names every wire you use and the method's
last parameter is `Context`. `[CERT]` Permissions, `unrestricted` and the write check: `types/security.md` § 2.7.
