### Added — strict L3 and L3b in lint-structure

- In a module whose `MODULE-MAP.md` says `modularity: strict`, `lint-structure` replaces the advisory L3 directory
  WARN with per-file FAIL rows on every `-rt`/`-ux`/`-wb`/`-se` part: a Baja-free class outside its role package
  (`model/`; for `-ux` also `route/` or `service/`), a `model/` class importing a `javax.baja` or `com.tridium`
  type, and a `model/` class with no `<C>Test.java` under `srcTest/`. New L3b (strict only): a `model/` class may
  not import the module's own non-model packages. Legacy modules and modules without a map are unchanged.
