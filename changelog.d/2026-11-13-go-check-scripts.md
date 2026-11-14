### Changed — three scripts/ checks ported to Go

- `scripts/check-client-source.sh`, `scripts/check-installed-line.sh` and `scripts/check-bats-skips.sh` are thin
  wrappers over `go/cmd/<tool>` (`go/internal/checkclientsource`, `checkinstalledline`, `checkbatsskips`): same
  arguments, rows, messages and exit codes; they now need `go` on `PATH` (exit 3 without it).
- `check-bats-skips`: an allowlist line is a Go POSIX ERE. A GNU grep extension (a back-reference, `\w`, `\<`) is
  an invalid line (exit 3), and an invalid line no longer prints grep's own error before the check's message.
- `check-installed-line`: the version read and compare are Go copies of `toolbelt/lib/vendor-version.sh`
  (`vv_source_version`, `vv_cmp`), pinned to that lib by `TestMatchesTheShellLib`.
