### Added — sweep-build-state: ledger evidence tags, module-field WARN, `--age --strict`

- The content check FAILs an `[ev:` / `[deferred:` tag of a `ledger.d/` fragment that is left open (no closing `]`
  before the next tag or the end of the text), named as `ledger.d/<file>:<line>`; backticked code is not a tag.
- A module envelope (not `module: kit`) with no or an empty `module_root` or `source_of_truth` prints a
  `sweep: WARN` line on stderr; the exit is unchanged (`unknown` is a value).
- `--age --strict` exits 1 when a pending retro is older than `--max-age` days (the report is unchanged; one stderr
  line names the count). Without `--strict` the report still always exits 0.
