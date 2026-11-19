### Fixed — mcp-n4: a second lost session is reported, and BQL flags a column that is no slot

- After a station restart the station answers "Could not find Server Session". A read or dry run
  already reconnected once with the last `n4_connect` and was retried once; a second loss in a row now
  answers a named error (reconnected, lost again, check the station and call `n4_connect`) instead of the
  raw BOX text, and is never looped. A write execution that loses its session answers
  "session lost: ... redo the dry run" and is not retried (delta U4).
- `n4_bql_query` adds a `warnings` entry for a projected column that is `NULL` on every row and names no
  slot of the queried type, read from the type contract (e.g. `maxOverrideDuration`, a facet key, gave
  135 rows of `"NULL"`). A column null on some rows only, or a real slot null everywhere, gives no warning;
  a normal reply is unchanged (delta M1).
