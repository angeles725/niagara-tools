// Package safety holds the write-safety machinery of the mcp-n4 server (mcp-n4-kit/mcp_n4/
// safety.py): so far the state dir, its trust check and the os.path rules its paths follow.
// The server never chmods a directory or file it did not create.
package safety

// DefaultStateDir is DEFAULT_STATE_DIR (expanded by the caller).
const DefaultStateDir = "~/.local/state/mcp-n4"
