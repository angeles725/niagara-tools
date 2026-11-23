### Changed — lint-structure L5 row says how to declare a library

- The undeclared `-rt` row of L5 now ends with `(a library module declares `type: library` in its MODULE-MAP.md header,
  types/utility-lib.md § 6)`, as the `-ux` L9 row does for a theme. `lint.ReadMapHeader` reads the modularity and the
  `type:` kind of a module map in one pass (`lint.MapHeader` is unchanged), and `lint-structure` uses it for the kind.
