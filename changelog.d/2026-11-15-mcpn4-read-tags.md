### Added — n4_read_tags: direct tags, tag groups, relations and implied tags in one call

- `n4_read_tags(ord | ords, implied=true, relations="direct")` reads up to 50 components: the direct tags (metadata
  slots, the id unescaped from the slot name: `hs$3apoint` is `hs:point`), the tag groups (`n:tagGroup` relations),
  the direct relations and, through the session handler's `impliedTags` call (the one BajaScript makes), the
  implied tags with their types. `relations="all"` adds the implied relations; `implied=false` makes no session call.
- A client that cannot call the session handler answers `implied: null` with `implied_unavailable`, never an empty
  list. The kit README gains a Tags section (how a tag, a tag group and an implied tag are stored).
