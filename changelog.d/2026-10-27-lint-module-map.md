### Added — lint-module-map and lint.ModuleMode

- `toolbelt/lint-module-map.sh` (Go rule `go/internal/lint/rules/modulemap`) checks a module root's `MODULE-MAP.md`
  (`types/structure.md` § Module map): MM1 no map (WARN), MM2 header/table (FAIL), MM3 listed path missing (FAIL),
  MM4 source file unlisted (FAIL strict, WARN legacy) or in two concerns (FAIL), MM5 concern without a test (FAIL
  strict, WARN legacy), MM6 test in no concern (WARN). `lint.ModuleMode(path)` returns the strict/legacy/none mode
  of the nearest map for per-file lints. report-module does not run it yet.
