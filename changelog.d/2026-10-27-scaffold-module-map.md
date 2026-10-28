### Changed — the scaffold emits a strict module map and role packages

- `scaffold-module.sh` now emits `<module-root>/MODULE-MAP.md` (`modularity: strict`) for every type: logic
  (`control`, `station-test`, `plumbing`), dashboard (`control`, `api-routing`, `frontend`, `station-test`,
  `plumbing`) and driver (`codec`, `network-device`, `points`, `discovery`, `point-manager`, `plumbing`). A fresh
  scaffold passes `lint-module-map` with no row.
- The fixtures' Baja-free classes moved to role packages: `<MOD>Logic` and the driver's `<MOD>Codec` (and their
  tests) to `com.<vendor>.<MOD>.model`, now public; the dashboard's router and servlet to
  `com.<vendor>.<MOD>.ux.route` (the ux `module-include.xml` type class and bajadoc package follow).
