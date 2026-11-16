### Changed — the pre-push CI lints shellcheck the change with its sourced files, not CI's whole set

- `scripts/ci-local.sh --changed` (run by `.githooks/pre-push`) shellchecked all ~285 files of CI's shellcheck set
  whenever one shell file changed (about 30-43 s per push). It now checks the changed files of that set together
  with the files they source and the files that source them, followed transitively through `.`/`source` lines,
  so a sourced lib still gives its sourcer's context (no false SC2034).
- When a source line names a path that matches no file of the set, it falls back to the whole set and names
  `<file>:<line>` on stderr.
