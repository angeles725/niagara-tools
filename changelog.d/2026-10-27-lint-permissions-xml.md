### Added — lint-permissions-xml

- New `toolbelt/lint-permissions-xml.sh <module-root>...` (G21): FAILs a `module-permissions.xml` whose root is not
  `<permissions>` (baja ignores every request under any other root) or whose `<niagara-permission-groups>` type is
  not `station`, `workbench` or `all`; WARNs a `<req-permission>`/`<opt-permission>` without `<name>` or
  `<purposeKey>` and a `new File("/…")` or `Paths.get("/…")` with an absolute literal in `-rt` code. A file that
  does not parse exits 3. Not yet run by `report-module`.
