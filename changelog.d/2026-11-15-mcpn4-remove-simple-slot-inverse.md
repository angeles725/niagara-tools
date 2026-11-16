### Fixed — n4_remove_component on a simple slot plans an inverse that re-adds that slot

- Removing a simple or struct slot (a direct tag such as `tst$3acap`, a relation `r`) planned an inverse that
  re-added the *parent component* under the slot's name: the station answers a simple slot's ORD with the component
  that holds it, and the snapshot was taken from that load. A rollback would have created a copy of the parent.
- When the child's load comes back with the parent's handle, the snapshot now comes from the parent's load: the
  inverse re-adds that slot with its type, value and flags (`{"nm":"a",…,"b":{"nm":"p","t":"baja:Double","v":"4.5"},
  "f":"m"}`), so a rolled-back tag keeps its metadata flag. Components are planned as before.
