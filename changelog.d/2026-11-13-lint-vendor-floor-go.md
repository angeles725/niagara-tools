### Changed — lint-vendor-floor is a Go command

- `toolbelt/lint-vendor-floor.sh` is now a thin wrapper over `go/cmd/lint-vendor-floor` (`go/internal/vendorfloor`),
  like the other lints: the arguments, the tool resolution (`KIT_NODE`, `KIT_ACORN`, `toolbelt/eslint`), the
  vendored-file list, the API-above-floor table (`apis.go`), the verdicts and the exit codes are Go. node + acorn stay
  external tools; they run the embedded `driver.js`, which only parses and tokenizes and reports JSON lines. Rows,
  stderr lines and exit codes are unchanged.
- A `KIT_NODE` that is a directory is now the `SKIP ... unavailable: node not found` row (exit 4) instead of bash's
  own exit 126, and a node that ends without the driver's result is an env fault (exit 3), never a clean exit.
- The table check (no method name in both the member and the qualified family) is pinned by Go tests
  (`TestDefaultTableIsDisjoint`, `TestSharedMethodNameIsAnEnvFault`); the bats case `VF-disjoint`, which edited the
  bash source, and its `# Mutation:` line are removed.
- Known gaps, kept for parity (not fixed here): a `vendor/` that is a symlink is not followed ("nothing to check",
  exit 0), and a sub-directory of `vendor/` that cannot be read prints find's error line on stderr and is skipped
  (exit 0 when the rest is clean). Both are silent passes of the bash lint.
