### Added — lint-port-parity: a reference frontend port drops nothing without a recorded decision

- `toolbelt/lint-port-parity.sh` (Go: `go/internal/portparity`, `go/cmd/port-parity`) diffs a reference frontend tree
  against its port (file basenames, DOM ids, named functions) and FAILs every absence not waived in
  `<module-root>/.port-parity` with `[decision: <ref>]`; a waiver without `decision:` FAILs, a stale waiver WARNs. A `file` waiver also waives
  the ids and functions only that reference file defines (INFO rows). `rename: function|id <ref> -> <port> -- <reason>`
  passes as INFO when the port has the target as the same kind; no reason or an absent target FAILs.
- `report-module` runs it on every module that has a `.port-parity` file (no row for other modules).
