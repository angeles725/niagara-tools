### Fixed — the pre-push shellcheck closure reads compound source lines and falls back on any it cannot resolve

- `ci-steps changed` now recognises a `.`/`source` command after a separator or keyword on its line
  (`[ -f x ] && . x`, `then . x`, `{ . x; }`, `a; . b`), not only at the start of a line, and follows it.
- A source line anywhere in CI's shellcheck set that names no file of the set now checks the whole set (it might
  be the only sourcer of a changed lib); before, only a line inside the closure did.
- A `# shellcheck source=` directive holds over comment and blank lines up to its command.
- A changed path the set does not hold (a deleted script) is left out of the closure instead of being read.
