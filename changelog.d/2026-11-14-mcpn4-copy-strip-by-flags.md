### Fixed — mcp-n4 n4_copy_component no longer drops configuration by slot name
- A copy leaves out slots by their effective flags (r readonly, t transient, d default-on-clone, R remove-on-clone), read from the station's type registry, instead of by a list of names: a child named `status` or `health`, or a fallback's status, is copied.
- The plan notes list every node left out with its flags, and call out a readonly slot that holds a non-default value.
- The read-back loads each copy and compares its whole configured subtree with the plan, listing the paths that differ or are missing.
- A copy is refused when the client cannot read the station's type definitions or the station does not describe a type it must evaluate.
- Fixes #612.

### Changed — box client caches type definitions per session
- `LoadTypes` keeps every type entry the station sent; a request for types it already holds makes no registry call.
