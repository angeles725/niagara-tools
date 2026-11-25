### Changed

- `toolbelt/nre-tool.sh recompile` (`go/internal/nretool`): when stdout and stderr are both terminals, nre gets the
  real streams instead of pipes, so RecompileTool's signing-password prompt (a Java console read, which may need a
  terminal stdout and whose echo through a pipe was unproven) runs as nre would run from a shell. nre's output is not
  read then, so the result is judged by nre's exit code and by the bog files: exit 1 (message `RecompileTool rewrote
  no bog file`) when no `*.bog` at or under the target changed in size or modification time. Piped or redirected,
  nothing changes: the output is read for an `ERROR` line and a `--- Processed <file>!` line, as before.
- Unit tests in `recompile_tty_test.go` (nre sees files on a terminal and pipes otherwise, a bog under a folder,
  no bog rewritten, exit code and start failure, stdin) pin it. Follow-up of #851; issue #1001.
- Not verified against a real install: that RecompileTool rewrites (changes the mtime of) a bog it processes even when
  it holds no Program to recompile. Doc to update: `build-n4-module-kit/BUILD-LOOP.md` line 357.
