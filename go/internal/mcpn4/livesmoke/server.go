package livesmoke

import (
	"os"
	"path/filepath"
)

// executable is the running binary, symlinks resolved ("" when unknown).
func executable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe
}

// DefaultGoBin is the mcp-n4 server next to the running binary (go/bin/mcp-n4 when both are
// built with `go build -o bin/`).
func DefaultGoBin() string { return filepath.Join(filepath.Dir(executable()), "mcp-n4") }

// kitDir is the directory the server runs in: mcp-n4-kit, two levels above go/bin, when it
// exists; "" (the current directory) otherwise.
func kitDir() string {
	dir := filepath.Join(filepath.Dir(executable()), "..", "..", "mcp-n4-kit")
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return filepath.Clean(dir)
	}
	return ""
}

// ResolveServer fills a.GoBin from the flag, then MCP_N4_GO_BIN, then DefaultGoBin, and
// describes the problem when no executable server is there ("" when there is one).
func ResolveServer(a *Args, getenv func(string) string) string {
	if a.GoBin == "" {
		a.GoBin = getenv("MCP_N4_GO_BIN")
	}
	if a.GoBin == "" {
		a.GoBin = DefaultGoBin()
	}
	if st, err := os.Stat(a.GoBin); err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {
		return "no executable Go server at " + a.GoBin + ": build it (cd go && go build -o bin/ ./cmd/mcp-n4) " +
			"or pass --go-bin"
	}
	return ""
}

// ServerArgv is the server command line: writes allowed, scoped to WriteScopes, with the
// station, home, TLS, state-dir and HTTP flags passed through in order.
func ServerArgv(a *Args) []string {
	argv := []string{a.GoBin, "--allow-writes", "--credential-env", a.CredentialEnv}
	for _, scope := range WriteScopes() {
		argv = append(argv, "--write-scope", scope)
	}
	for _, group := range []struct {
		flag  string
		items []string
	}{{"--station", a.Stations}, {"--station-home", a.StationHomes}, {"--insecure-tls", a.InsecureTLS}} {
		for _, it := range group.items {
			argv = append(argv, group.flag, it)
		}
	}
	if a.StateDir != "" {
		argv = append(argv, "--state-dir", a.StateDir)
	}
	if a.AllowHTTPForTests {
		argv = append(argv, "--allow-http-for-tests")
	}
	return argv
}
