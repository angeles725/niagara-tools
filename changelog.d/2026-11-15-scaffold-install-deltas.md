### Changed — scaffold-module writes option values as typed

- `--target-version` and `--plugin-version` are literal text: `&`, `|` and `\` are written as typed. Before, a
  value was read as sed replacement text (`&` was the replaced token, `\&` a literal `&`, `\n` a newline) and a
  value with an unescaped `|`, a trailing `\` or `\1`-`\9` was refused. A value that holds a newline is exit 2 with
  `scaffold-module: <option> value must be a single line (it holds a newline): <value>` and nothing written; the
  message `<option> value cannot be substituted (...)` is gone. `--vendor` keeps its own rule (one Java package
  segment once lowercased, exit 2). Retro `2026-11-14-go-scaffold-module-port` Δ1.
- Inserted text is never rewritten by another substitution: `--plugin-version 4.14 --target-version 4.15` keeps
  plugin `4.14` (it became `4.15`), and a ModuleName that holds the fixture vendor (`AngelesBox --vendor Acme`)
  keeps its name (it became `AcmeBox`). Ordinary values emit the same tree byte for byte.

### Fixed — scaffold-module removes the partial tree of a failed run

- A run that fails after it created `<out-dir>/<ModuleName>` (a failed write, a missing station-test or
  version-ledger template) removes that directory and adds one stderr line, `scaffold-module: removed the partial
  <path>`; when the removal fails the line is `scaffold-module: could not remove the partial <path> (<error>): its
  files stay`. The exit code stays 3. Before, the partial tree stayed and the next run was refused with "destination
  already exists". A destination that existed before the run is refused as before and never touched; the out-dir
  itself is never removed. Retro `2026-11-14-go-scaffold-module-port` Δ4.

### Removed — install-skill and check-skill-drift no longer require a sha256 tool

- `scripts/install-skill.sh` and `scripts/check-skill-drift.sh` no longer exit 3 with `<tool>: no sha256 tool found
  (sha256sum or shasum)` when neither tool is on PATH: the digests are computed by the Go command. The exit 3 that
  remains before the command runs is the Go one (`<tool>: go is not on PATH; it is needed to build <bin>` when the
  command has to be built). `scripts/lib/sha256.sh` stays for its one caller, `scripts/ci-local.sh`. Retro
  `2026-11-14-go-install-tools-port` Δ1.

### Fixed — the close bats no longer run install-skill against the real home

- `tests/c8-close.bats` to `tests/c11-close.bats` run `install-skill.sh --dry-run` with `--home` on the case's
  temporary directory; the c8 and c9 cases no longer read `$HOME/.claude/skills` to decide whether to skip.
  `tests/shell-hygiene.bats` SH2 fails on a bats case that runs a script taking `--home` with neither `--home` nor a
  `HOME=` override on the command. Retro `2026-11-14-go-install-tools-port` Δ3.

### Fixed — the PF-jarlock cases no longer depend on the shared go/bin during a parallel run

- `tests/preflight.bats`: the cases that narrow `PATH` (no `go`) run a private copy of `preflight.sh` and of its
  binary, made once per file. Before, they ran the launcher over the shared `go/bin/preflight`, so a Go source that
  changed during the run (an edit, a checkout, another build) made the launcher exit 3 and the cases fail on a row
  that was never printed. `PF-jarlock-timing` now asserts that the check ran and that its one `lsof` call slept (its
  fake `lsof` called a `sleep` the narrowed `PATH` did not have), with 9 s of margin instead of under 2 s. No lock
  path or port was shared: each case already had its own temporary directory. Retro
  `2026-11-14-go-scaffold-module-port` Δ2.
