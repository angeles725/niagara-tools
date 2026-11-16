### Fixed — n4_set_tags refuses duplicates and contradictions and restores relations as stored

- A tag, tag group or relation named twice in one list, or both added and removed, is refused at plan; so is
  adding a group or relation the target already has (each used to plan two slots or an in-doubt remove).
- Only simple value types are written as tags (Marker, Boolean, Double, Float, Integer, Long, String,
  DynamicEnum, Format, RelTime), each value checked for its type: `baja:Folder` or `nonsense` is refused.
- The rollback of a removed relation or tag group re-adds the fields the slot stored (the endpoint ORD as it
  was written), not the request's form.
- `n4_set_tags` is in the destructive class (`destructiveHint: true`): it removes tags and relations.
