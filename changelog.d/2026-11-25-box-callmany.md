### Changed

- `go/internal/mcpn4/box`: `Client.CallMany([]Msg) ([]Reply, error)` sends several BOX messages in one frame
  (`r` = 0..n-1) and matches the replies by `r`, so a tool that loads several ORDs no longer pays one HTTP round trip
  per message (#1025). `BBoxService` answers each message of a frame on its own: a message the station refuses comes
  back as that entry's `Reply.Err` (a `*Error` with `Station` set) and never fails the others (`BBoxService.java`
  `handleMessage` catch → `handleMessageError`, decompiled `BBoxService.java:480-511`, `:521-543`, 4.14.0.162 corpus); only a frame the station
  cannot take at all (session lost on a message that needs one, `:423-433`) answers every entry with the error. The replies may
  come in any order; a missing, repeated, out-of-range or non-integer `r` is a malformed-reply error. More than
  `MaxFrameMessages` (50) messages go in several frames. The reply-size (32 MB), nesting, seq, auth (401/403 → one
  `AuthError`, latched once) and transport guards are the ones `Call` has; `Call` now shares them (`exchange`).
- `box.Client.SSCMany(key, args)` is `SSC` once per arg in one frame; `box.Client.LoadMany(ords, depth, opt)` is
  `LoadTree` of several ords with their `loadSlots` requests in one frame (poll, one `loadSlots` frame per 50 ords,
  poll: 3 requests for 10 ords instead of 30). A load is an event, not a reply, so its ops are matched to the ords in
  request order and against a cached handle when there is one; an ord whose op does not match or does not come is
  loaded on its own by `LoadTree`, and a load still outstanding from an earlier request makes the whole call plain
  `LoadTree` per ord. Handles are remembered as `LoadTree` does.
- Round trips are countable: `Client.Frames()` is the number of BOX frames the client has posted, and a client made
  with `MCP_N4_BOX_TRACE=1` in the environment prints one stderr line per frame (`box frame seq=3 messages=50
  key=callssc`); count the lines of one tool call.
- `n4_read_tags` (`go/internal/mcpn4/readtools`) is not switched to `LoadMany` by this change; see #1025.
