### Added — lint-rdbms-dialect-descriptor

- New `toolbelt/lint-rdbms-dialect-descriptor.sh [--strict] <module-root>...` (WARN, advisory, ledger item G88): a
  `new B(SqlServer|MySQL|Oracle|HsqlDb)HistoryExport(` in `-rt` source fixes the database dialect in code, and a
  `jdbc:postgresql:` literal in a module with no class extending `BRdbms` reaches for a dialect the platform does not
  ship (`types/cloud-connector.md` § 11). Not run by `report-module`.
