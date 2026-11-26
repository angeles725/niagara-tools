### Added

- `lint-module-map.sh --base <ref> <module-root>` (#1125): new check MM8 reports every file the change added since
  `<ref>` that no row of `MODULE-MAP.md` lists, among the files MM4 does not already require: `srcTest/**/*.java`,
  `webtest/**/*.test.{mjs,js}` and vendored `src/rc` files (`vendor/`, `ext/`, `*.min.*`). WARN in a `legacy` module,
  FAIL in a `strict` one, so a legacy module stops growing its unmapped set without a full migration; the row says to
  add the file's row in the same change. A new source file in no row stays MM4's row alone. The caller names the base
  ref; the toolbelt script starts no version control (the Go rule asks `go/internal/gitcmd`). Without `--base` the
  lint prints exactly what it did before; a ref that does not resolve is one INFO row (exit 0). Tests:
  `go/internal/lint/rules/modulemap/newfiles_test.go`.
- `lint-module-map.sh` now takes only `--base` as an option: a second root or another flag is the usage line (exit 3)
  instead of being read as a root. Skill step 1e and the BUILD-LOOP parallel-writer checklist run it with `--base`.
