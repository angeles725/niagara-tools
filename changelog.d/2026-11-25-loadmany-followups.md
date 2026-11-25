### Changed

- `box.Client.LoadMany` (`go/internal/mcpn4/box/loadmany.go`, `loadmany_wire.go`): ords beyond
  `MaxFrameMessages` (50) are loaded in batches, one loadSlots frame per poll cycle (poll, frame, polls until its ops
  are in, next batch). A timeout on frame 2 asks frame 2 again alone, never frame 1 (no duplicate ops), and
  `loadsOutstanding` counts only the frame in doubt. A failure that is not a timeout keeps the results of the batches
  done before it: the call returns them (a prefix of the ords) with the error, `(nil, err)` only when the first batch
  failed; the failed batch alone is taken off `loadsOutstanding`. Issue #1049 item 2.
- `LoadMany` order matching: the load op carries the handle of the component it loaded (`SyncOp.encode`,
  `h`), never its ord, so an ord without a cached handle is matched by arrival order. The station keeps that order:
  `BBoxService.handleMessages` runs a session's messages in order (`BBoxService.java:450-464`) into the FIFO of
  `SyncBuffer` loads (`SyncBuffer.java:309`, `OpLinkedList.append`). An ord with a cached handle must get its op, and an
  uncached ord no longer takes an op of the cached handle of another ord of the batch: the batch stops matching and
  the rest load alone with `LoadTree`. Two uncached ords swapped on the wire cannot be told apart (no ord, no
  handle yet). Issue #1049 item 1.
- Tests: mixed cached/uncached batch in order and with the op stream reordered (four orders), 120 ords as three
  sequential poll cycles, timeout and refusal on the second batch (`box/loadmany_match_test.go`,
  `loadmany_batches_test.go`), and `ReadTags` over the real client against its `LoadTree`-only twin: same reply, same
  first error (`readtools/tags_batch_internal_test.go`). Issue #1049 item 3.
