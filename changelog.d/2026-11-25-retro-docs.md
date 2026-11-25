### Changed

- Kit docs, retro deltas of #985 and #986 that need no code. mcp-n4: one routed in-doubt recovery step in
  `METHODOLOGY.md` section 4 step 6 and in the skill (read the journaled intent, read the slot back, then applied /
  redo the dry run / undo the standing part by hand, because `n4_rollback` refuses an in-doubt batch; record the
  decision) (#985 Δ10); a timing claim is measured through an independent observer (#985 Δ7); `claude mcp list` is
  never a registration probe (#985 Δ11). build-n4-module-kit: the `types/dashboard.md` rules for the facets min/max
  check and the user Context, a named reason on a refused request, cross-check partners in a live fixture,
  Template I/O at scale, the "nohistory" cause and `/api/*` cache headers; a bare-slot-name rule in
  `types/lexicon.md`; the Clock ticket lateness in `types/logic-authoring.md`; a stop-order section in
  `types/platform.md`; and `BUILD-LOOP.md` § 6.b item 11 (live check shape) plus the first-deploy envelope note
  (#986 Δ2, Δ3, Δ7, Δ10, Δ11, Δ13, Δ16, Δ17, Δ22, Δ25, Δ27, Δ32).
