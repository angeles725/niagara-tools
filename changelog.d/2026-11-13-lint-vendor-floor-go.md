### Changed — lint-vendor-floor is a Go command

- `toolbelt/lint-vendor-floor.sh` is now a thin wrapper over `go/cmd/lint-vendor-floor` (`go/internal/vendorfloor`),
  like the other lints: the arguments, the tool resolution (`KIT_NODE`, `KIT_ACORN`, `toolbelt/eslint`), the
  vendored-file list, the API-above-floor table (`apis.go`), the verdicts and the exit codes are Go. node + acorn stay
  external tools; they run the embedded `driver.js`, which only parses and tokenizes and reports JSON lines. Rows,
  stderr lines and exit codes are unchanged.
- A `KIT_NODE` that is a directory is now the `SKIP ... unavailable: node not found` row (exit 4) instead of bash's
  own exit 126, and a node that ends without the driver's result is an env fault (exit 3), never a clean exit.
