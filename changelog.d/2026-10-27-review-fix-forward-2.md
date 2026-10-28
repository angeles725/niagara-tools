### Fixed — lint-module-map MM7, module-map tests, rc-scan trailing-slash root

- `lint-module-map` MM7 compares rows by concern AND part (case-insensitive): a concern with one row per part is
  clean, `plumbing` rows may repeat, and a second row of one concern in one part is still a row. Concern folders are
  gathered per row, so rows of one concern in two parts no longer merge into a false "two names" row.
- `module-map.sh tests` treats a srcTest class as a station test when its code references `javax.baja` fully
  qualified, not only through an import line; a mention in a comment or a string still runs as a pure test. The
  skip row reads `uses javax.baja`.
- `rc-scan` given an artifact dir with a trailing slash reads the path below it for the vendored skip and the map
  anchors, so an ancestor directory named `rc` no longer stops `rc/vendor/**/*.js` from being skipped.
