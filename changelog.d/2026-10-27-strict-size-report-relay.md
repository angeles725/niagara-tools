### Fixed — report-module relays a strict module's size FAIL rows

- `report-module` relays an advisory member that exits 1 with a FAIL row (`lint-size` in a `modularity: strict`
  module) as FAIL rows with no PASS verdict; before, it printed one `ERROR env fault (exit 1)` row and hid the
  size rows. An exit 1 with no FAIL row is still an env fault.
- `lint-size` `adapter-branches` counts a static import of a `model.` member (`import static …model.X.y;`) as the
  adapter's pure peer.
