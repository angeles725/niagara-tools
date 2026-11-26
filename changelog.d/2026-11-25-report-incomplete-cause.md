### Fixed

- lint-vacuous-tests: an enum constant with a body (`A(1) { ... }`) or a constructor as the first member of a class
  was exit 3 `cannot find the declaration of <Class>.<name>` on a test file that compiles and passes (#1116); the
  scan now reads it.
- build.sh: a module with a vendored script under `<mod>-ux/src/rc/vendor` needs node + acorn for report-module's
  vendor-floor check. They are now resolved before gradle (exit 10 with `npm install --prefix <toolbelt>/eslint`, or
  `KIT_ACORN` / `KIT_NODE`) instead of a vendor-floor SKIP at the end of the build; `--no-preflight` skips it (#1116).

### Changed

- report-module: an INCOMPLETE verdict names its cause: `INCOMPLETE (environment)` for a member that could not run
  (an ERROR row), `INCOMPLETE (module)` for a skipped floor check, `INCOMPLETE (environment, module)` when both
  hold, and `ISSUES (+INCOMPLETE (module))` likewise. A skipped module check no longer hides among environment
  faults. The exit codes are unchanged (#1116).
