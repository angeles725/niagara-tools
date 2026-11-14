### Fixed — live-smoke: a relative `--go-bin` starts the file that was checked
- `live-smoke --go-bin NAME` (or `MCP_N4_GO_BIN`) with a name that has no directory was checked as a file of the working directory and then started through `PATH`, so another `NAME` could run or the start could fail. The resolved path is now absolute.
