### Added — `types/data-access.md` and a `doPrivileged` rule

- `types/data-access.md`: which channel a browser or off-station client uses, the BOX limits in code (WebSocket idle
  60 000 ms, text 262 144 B, binary 131 072 B, session 90 s, one unbounded-pool thread per WebSocket session),
  gzip off by default with the MIME include list, the 30-day `FileServlet` cache with no ETag, an audio recipe, an
  oBIX Watch recipe for an off-station reader (`alarmFeed`, a client-side watermark, `<err>` in the body),
  `/box/query` and `/ord` CSV, and `baja.rpc` riding BOX `sys/rpc` rather than `/rpc`.
- `types/security.md` § 11: when `AccessController.doPrivileged` is the fix for an `AccessControlException`, how small
  the block stays, and the shipped examples; § 3.2 points to it.

### Changed — `types/dashboard.md`

- The live-data matrix says REST-poll is the chosen default, not a platform limit, and gives the BOX tab threshold;
  its query row passes the caller's Context.
- The alarm read is `alarm:|bql:select top 200 * from openAlarms` with the state filtered in Java, not an unindexed
  `where sourceState … order by` scan.
- A bullet on the 30-day, no-ETag cache of `/module/*` and `/file/*`; the off-station row points to the Watch recipe.
- `go/internal/cardwidth/baseline.txt`: `types/dashboard.md` 146 to 144 overlong lines.
- `README.md` and `skill/SKILL.md` list `types/data-access.md` the way `types/queries.md` is listed.
