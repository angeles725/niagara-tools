### Fixed — n4_remove_component on a simple slot plans an inverse that re-adds that slot

- Removing a simple or struct slot (a direct tag such as `tst$3acap`, a relation `r`) planned an inverse that
  re-added the *parent component* under the slot's name: the station answers a simple slot's ORD with the component
  that holds it, and the snapshot was taken from that load. A rollback would have created a copy of the parent.
- When the child's load comes back with the parent's handle, the snapshot now comes from the parent's load: the
  inverse re-adds that slot with its type, value and flags (`{"nm":"a",…,"b":{"nm":"p","t":"baja:Double","v":"4.5"},
  "f":"m"}`), so a rolled-back tag keeps its metadata flag. Components are planned as before.
- The links of a removed simple slot are checked like a component's: a link it sources into another component
  refuses the plan until `acknowledge_links: true`, is listed in `outgoing_links_broken` and is re-created by
  `n4_rollback` (only the links read from that slot: the holder's other links never refuse a tag remove); a link of
  the holder that writes the slot is recorded as an `unlinked_input` with the slot's last value (`prior`), which the rollback reports. A struct slot (a
  `StatusNumeric`) is snapshotted with its fields whatever depth the parent was loaded at.
- `n4_rollback` accepts the escaped slot names such a remove records (`tst$3acap`) in an outgoing link or an unlinked
  input, and its read-back of a re-created outgoing link now finds a `baja:ConversionLink` (a `Double` into a
  `StatusNumeric` input), which it reported missing.
