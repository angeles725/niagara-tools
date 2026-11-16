### Changed — mcp-n4 write scope: create the scope root, scope a link on its target

- `n4_create_component` may create the `--write-scope` root itself under its parent, so a scope can
  name a folder that does not exist yet; it is refused, naming the root, when that root already
  exists, because the station would rename the new component outside the scope. Retro
  niagara-research client A 2026-11-13 M5.
- `n4_create_link` checks the write scope on `target_ord` only (the link is stored on the target; the
  source is only read). The source must still be a plain slot ORD. A refusal now names the end:
  `target_ord: ord '…' is outside the write scope (…)` or `source_ord: ord '…' is not a plain station
  ORD (…)`. Before, a source outside the scope refused the link. Retro M10.
