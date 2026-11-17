### Added — mcp-n4: `n4_inventory_snapshot` and `n4_inventory_diff` (station-audits S8)

- `n4_inventory_snapshot(base, depth=6, max_components=20000, slots)` writes the components of a subtree (slot path,
  type, handle, configured property values or the chosen slots) to `<state-dir>/exports/inventory-<station>-<time>.json`
  with one BOX load. The type contract decides what is runtime: transient slots (instance or contract, such as `out`,
  the inputs other than `in1` and `in8`, `lastRecord`), a computed Status `out` and linked inputs are left out; a Status config slot such as
  `fallback` is kept.
- `n4_inventory_diff(a, b, limit=200)` compares two snapshots by slot path from the local files only: `added`,
  `removed`, `changed`, `type_changed`, `handle_changed`, `counts`, `truncated`, `notes`. A name with `..` or a path
  separator, or one not under the exports dir, is refused.
