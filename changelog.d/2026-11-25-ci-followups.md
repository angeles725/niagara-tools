### Changed

- PR size runs the gate scripts of the base revision (`scripts/pr-size-base.sh` extracts `pr-size.sh` and `pr-size-waiver.sh` with `git show`), so a PR cannot weaken its own gate; a script new in the PR falls back to the head's with a notice (#979).
- `ci-local.sh --changed`: a hard gate whose command is missing (exit 127) or whose `go test` shows `[build failed]`/`[setup failed]` prints `FAIL <gate> (environment: ...)` instead of looking like a size finding; it still counts as failed, exit 1 (#973).
- BUILD-LOOP §0.c says CI also gates the PR size (400 lines, `size:exception` waiver) and the local pre-push only warns (#979).
- `go/README.md`: the `internal/filesize` row is a short cell that links to a `### filesize` section (#983).
