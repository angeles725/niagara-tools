### Added

- mcp-n4: a write still running after 45 s answers `{state: "running", batch_id, executed: true}` before the MCP client's 60 s limit; the write goes on, journals its result and audits its end, and the server reads the next call only once it ended (no concurrent use of the station session). (#925)
