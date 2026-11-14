### Fixed — mcp-n4 n4_copy_component no longer drops configuration by slot name
- A copy leaves out runtime slots by flags instead of by a list of names, read from the station's type registry: a frozen slot by its declared flags (r readonly, t transient, d default-on-clone) or when made transient, a dynamic slot only when flagged t or R (remove-on-clone). Flags an integrator set are configuration and travel with the copy; a child named `status` or `health`, or a fallback's status, is copied.
- The plan notes list every node left out with its flags, and call out a readonly slot that holds a non-default value.
- The read-back loads each copy and compares its whole configured subtree with the plan, listing the paths that differ or are missing.
- A copy is refused when the client cannot read the station's type definitions or the station does not describe a type it must evaluate.
- Fixes #612.

### Changed — box client caches type definitions per session
- `LoadTypes` keeps every type entry the station sent; a request for types it already holds makes no registry call.
