### Added — lint-structure L17: module.palette type specs

- `lint-structure.sh` reads each part's `module.palette`: a dotted Java class name in `t=` and a `B`-prefixed name of a
  type the module declares without it FAIL (Workbench "Invalid typespec" / "Type not found"); a later `m=` that maps a
  symbol to another module, and a palette with entries that is not well-formed XML, WARN. An identical repeated `m=` is
  clean.
