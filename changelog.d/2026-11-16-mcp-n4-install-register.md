### Added — `mcp-n4 install` and `mcp-n4 register`

- `mcp-n4 install [--prefix DIR]` copies the running binary to `DIR/mcp-n4` (default `~/.local/bin`)
  atomically (temporary file, fsync, chmod 0755, rename), so station launchers stop depending on a worktree.
- `mcp-n4 register --station NAME --harness claude|pi|gentle-shell [--env PATH] [--bin PATH] [--project DIR]
  [--force] [-- SERVER-FLAGS]` writes `~/.config/mcp-n4/NAME-run.sh` (sources the env file, holds no
  credential) and the `mcp-n4-NAME` MCP entry for the harness, changing only the bytes of that entry; re-running
  is a no-op, and a differing launcher or entry, or a `--bin` inside a git checkout, is refused without `--force`.
