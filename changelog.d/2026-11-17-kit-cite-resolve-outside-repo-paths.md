### Added — kit-cite-resolve OUTSIDE_REPO and live repo paths

- `kit-cite-resolve` reports an `[ev: …]` token naming a `~/` or absolute path as `OUTSIDE_REPO` (a strict finding
  on live doctrine, `HISTORICAL_DRIFT` on a history card) instead of resolving it by existence on one machine.
- On live doctrine, a backticked `toolbelt/`, `scripts/`, `go/` or `tests/` path of prose must exist under the kit
  or the repository (`MISSING_FILE` otherwise), so deleting a file fails `--strict` until the doctrine naming it
  is reworded; the generated `toolbelt/INDEX.md` and `go/bin/` pass. Live docs naming deleted libs were reworded.
