### Added — lint-logger-name

- New `toolbelt/lint-logger-name.sh <module-root>...` (G40): WARNs a `Logger.getLogger("<x>")` whose name is neither
  the part's exact `moduleName` nor a `<moduleName>.<child>` logger, and a `Logger.getLogger(X.class.getName())`.
  The module name comes from the part's gradle `moduleName`, else the group gradle, else the part directory name.
  Not yet run by `report-module`.
