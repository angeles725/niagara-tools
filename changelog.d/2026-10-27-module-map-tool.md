### Added — module-map tool: add-concern, where, tests

- `toolbelt/module-map.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]` (also
  `scaffold-module.sh add-concern ...`): creates the concern folder under the part's role folder, a pure
  `<Name>Logic` class and its `<Name>LogicTest`, an rt `<Name>Adapter` stub with `--adapter`, and inserts the
  concern's MODULE-MAP.md row; an existing concern is refused with nothing written. A fresh scaffold plus
  add-concern stays clean under lint-module-map, strict L3 and lint-size.
- `module-map.sh where <file|concern>` prints the owning concern, its files, tests and "change here when";
  `module-map.sh tests <concern>` runs only that concern's pure JUnit tests through `run-pure-test.sh` and names
  the station tests and JS specs it skips.
