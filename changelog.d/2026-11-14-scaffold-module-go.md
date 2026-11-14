### Changed — toolbelt/scaffold-module.sh is a launcher over go/cmd/scaffold-module
- The scaffolder (fixture copy, name substitution, station-test overlay, version ledger, `add-concern` passthrough) runs in Go (`go/internal/scaffoldmodule`) with the same emitted trees, messages and exit codes; the script only builds and execs the binary (exit 3 without a Go toolchain).
- An option value sed rejected as replacement text (an unescaped `|`, a newline, a trailing `\`, `\1`-`\9`) and the GNU-only case escapes (`\L \U \l \u \E` and `\c \d \o \x`) exit 2 and write nothing; the bash tool printed sed errors, left a broken tree and exited 0.
- A write fault while emitting (a read-only out-dir) exits 3 with one line; the bash tool exited 0 with "emitted".
