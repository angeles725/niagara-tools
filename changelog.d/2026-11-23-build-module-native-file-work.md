### Changed — build-module does its file work itself

- `build.sh` (`go/cmd/build-module`) no longer runs `cp`, `mkdir`, `mktemp`, `unzip` or `df`: the pre-build jar copy
  keeps the mode, times and (where the user may) the owner, a temporary path is made under `$TMPDIR` (else `/tmp`), the
  module version of a jar is read with `archive/zip`, and the WSL 9p/drvfs probe reads the mount table (a mount
  matches by whole path components, so `/mnt/cc` is not under a mount at `/mnt/c`). A host without those commands
  builds the same; none of them is on the `PATH` the chain needs any more (`git` and `tar` stay, for
  `--schema-baseline-ref`).
- Diagnostics reworded, exit codes unchanged: a temporary path that cannot be made is
  `build.sh: cannot make a temporary directory in <dir>: <reason>` (exit 1; it was `mktemp: failed to create ...`); a copy that fails names the path
  (`cannot read <src>: ...` / `cannot write <dst>: ...`); a directory that cannot be made is
  `build.sh: cannot create directory <dir>: <reason>`. A copy onto the file it copies, or onto a directory, is refused
  and never truncates the source.
- The bats cases that stubbed `unzip` on `PATH` (`BS-drift-unverifiable-unzip`, `-restore-fail`,
  `BS-drift-empty-listing`) are Go tests with the fault injected into the jar. Retro `2026-11-14-go-build-module-port` Δ3.
