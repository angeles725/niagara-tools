package blackbox

import (
	"os"
	"sort"
	"strconv"
)

// Options are the server's command-line options (go/internal/mcpn4/cli/flags.go). A zero field
// adds no flag; Extra is appended as given, for any flag without a field.
type Options struct {
	Stations      map[string]string // --station NAME=URL, sorted by name
	StationHomes  map[string]string // --station-home NAME=DIR, sorted by name
	NiagaraHomes  map[string]string // --niagara-home NAME=DIR, sorted by name
	AllowHTTP     bool              // --allow-http-for-tests
	AllowWrites   bool              // --allow-writes
	StateDir      string            // --state-dir
	WriteScopes   []string          // --write-scope, repeated
	MaxWrites     int               // --max-writes
	CredentialEnv string            // --credential-env
	AllowTierB    []string          // --allow-tier-b, repeated
	AllowTierC    []string          // --allow-tier-c, repeated
	Extra         []string
	// Exe is the binary to run as the server instead of the test binary (a copy of it).
	Exe string
	// Env is the child's environment on top of baseEnv: the developer's own MCP_N4_* variables
	// never leak into a test.
	Env map[string]string
}

// baseEnv are the variables a child keeps from the test process.
var baseEnv = []string{"PATH", "HOME", "TMPDIR", "LANG", "SYSTEMROOT", "GOCOVERDIR"}

// Argv is the command-line form of o.
func (o Options) Argv() []string {
	var args []string
	flag := func(on bool, name string) {
		if on {
			args = append(args, name)
		}
	}
	value := func(name, v string) {
		if v != "" {
			args = append(args, name, v)
		}
	}
	repeat := func(name string, vs []string) {
		for _, v := range vs {
			args = append(args, name, v)
		}
	}
	pairs := func(name string, m map[string]string) {
		for _, k := range sortedKeys(m) {
			args = append(args, name, k+"="+m[k])
		}
	}
	flag(o.AllowWrites, "--allow-writes")
	flag(o.AllowHTTP, "--allow-http-for-tests")
	value("--state-dir", o.StateDir)
	repeat("--write-scope", o.WriteScopes)
	if o.MaxWrites != 0 {
		value("--max-writes", strconv.Itoa(o.MaxWrites))
	}
	value("--credential-env", o.CredentialEnv)
	repeat("--allow-tier-b", o.AllowTierB)
	repeat("--allow-tier-c", o.AllowTierC)
	pairs("--station", o.Stations)
	pairs("--station-home", o.StationHomes)
	pairs("--niagara-home", o.NiagaraHomes)
	return append(args, o.Extra...)
}

// Environ is the child's environment: baseEnv from this process, then o.Env.
func (o Options) Environ() []string {
	var env []string
	for _, k := range baseEnv {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	for _, k := range sortedKeys(o.Env) {
		env = append(env, k+"="+o.Env[k])
	}
	return env
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
