### Added — mcp-n4: n4_describe_session reports the server build and a stale-binary warning
- `n4_describe_session` adds a `server` block, with or without a session: `build` (the git revision, `+modified` for a dirty tree, `unknown` without a VCS stamp), `built_at` (the commit time, else the binary's mtime at start), `started_at` and `binary` (the executable's path).
- When the binary on disk was modified after the server started, the reply carries a `warning` to restart the MCP server: new tools reach a client only after a restart.
