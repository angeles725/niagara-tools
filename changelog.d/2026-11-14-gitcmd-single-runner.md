### Changed — one git runner for the install tools, kit-resolve and the push gate
- `go/internal/gitcmd` gains `Command{Dir, Env, Stderr}.Run`, which returns git's stdout, its exit code and the error, and tells a git that could not be started from one that exited non-zero. `installtools.GitStatus`, the `kit-resolve` fetch and the git calls of `pre-push` run on it; no tool changes its output, its messages or its exit codes.
- `kit-resolve` passes `GIT_TERMINAL_PROMPT=0` to its fetch only, instead of setting and restoring the variable in its own process.
### Removed — scripts/lib/live-doctrine-set.sh
- The live-doctrine set is defined once, in `go/internal/prepush/paths.go`. The shell copy had no caller left; it is deleted with the test that compared the two copies, and the Go patterns are pinned by the paths they select.
