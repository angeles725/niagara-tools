// Package cli is the mcp-n4 command line (mcp-n4-kit/mcp_n4/server.py:191-312): argparse's
// flags, bounded ranges and error text, the start checks main() and Server(...) make, then the
// stdio loop. With --allow-writes the write state is checked at start and the ported write
// tools are served (writeops); the others answer that they are not ported yet.
package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// Prog is the argparse prog name every message starts with.
const Prog = "mcp_n4.server"

// Main is the kitcli.Main entry point: it serves the process stdin.
func Main(args []string, s kitcli.Stdio) int { return Run(args, os.Stdin, s) }

// Run parses args, makes the start checks and serves in until EOF. Exit codes are server.py's:
// 0 after EOF (or --help), 2 on a flag error or a refused start, 1 when the stdio loop fails.
func Run(args []string, in io.Reader, s kitcli.Stdio) int {
	reg := registry.Default()
	cfg, help, uerr := Parse(args)
	if uerr != nil {
		fmt.Fprintf(s.Err, "%s%s: error: %s\n", reg.Usage, Prog, uerr.Msg)
		return 2
	}
	if help {
		fmt.Fprint(s.Out, reg.Help)
		return 0
	}
	if err := Check(cfg); err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", Prog, err)
		return 2
	}
	srv, err := NewServer(cfg)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", Prog, err)
		return 2
	}
	if err := srv.Serve(in, s.Out); err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", Prog, err)
		return 1
	}
	return 0
}

// NewServer is Server(...) for a cfg Check accepted: the read tools and, with --allow-writes,
// the write state (WriteState's start checks: limits, state dir, scopes) and the write tools.
func NewServer(cfg Config) (*jsonrpc.Server, error) {
	reg := registry.Default()
	srv := jsonrpc.New(reg)
	ctx := NewContext(cfg)
	tools := &readtools.Tools{Ctx: ctx, NewClient: readtools.BoxFactory, ServerVersion: reg.Version}
	if cfg.AllowWrites {
		state, err := NewWriteState(cfg)
		if err != nil {
			return nil, err
		}
		tools.Audit = state.Audit
		writeops.Register(srv, &plan.Engine{Ctx: ctx, State: state})
	}
	readtools.Register(srv, tools)
	return srv, nil
}

// NewWriteState is the WriteState Server(...) makes for a cfg Check accepted: its start checks,
// and the station homes main() passes (NAME -> PATH, the last repeat wins).
func NewWriteState(cfg Config) (*plan.State, error) {
	state, err := plan.NewState(cfg.WriteScopes, stateDir(cfg.StateDir), int(cfg.TokenTTL), int(cfg.MaxWrites))
	if err != nil {
		return nil, err
	}
	for _, item := range cfg.StationHomes {
		name, path, _ := strings.Cut(item, "=")
		state.StationHomes[name] = path
	}
	return state, nil
}

type startError struct{ msg string }

func (e *startError) Error() string { return e.msg }

// NewContext is the session state of a server started with cfg, which Check accepted.
func NewContext(cfg Config) *session.Context {
	urls, _, _ := stations(cfg)
	return session.New(session.Config{AllowWrites: cfg.AllowWrites, AllowHTTP: cfg.AllowHTTP,
		Stations: urls, CredentialEnv: cfg.CredentialEnv, InsecureTLS: cfg.InsecureTLS,
		AllowTierB: cfg.AllowTierB, AllowTierC: cfg.AllowTierC, AuthCooldown: float64(cfg.AuthCooldown),
		HTTPTimeout: float64(cfg.HTTPTimeout), LoadWait: cfg.LoadWait, StateDir: stateDir(cfg.StateDir),
		ProgressPath: progressPath(cfg.ProgressFile)})
}

// stateDir is os.path.expanduser(state_dir or DEFAULT_STATE_DIR).
func stateDir(dir string) string {
	if dir == "" {
		dir = safety.DefaultStateDir
	}
	return safety.ExpandUser(dir)
}

// progressPath is the expanded --progress-file, "" when none.
func progressPath(file string) string {
	if file == "" {
		return ""
	}
	return safety.ExpandUser(file)
}

// stations is the stations dict of main(): NAME -> URL (the last repeat wins) and the names in
// insertion order.
func stations(cfg Config) (map[string]string, []string, error) {
	urls := map[string]string{}
	var names []string
	for _, item := range cfg.Stations {
		name, url, ok := strings.Cut(item, "=")
		if !ok || name == "" || url == "" {
			return nil, nil, &startError{"--station expects NAME=URL, got " + pyjson.StrRepr(item)}
		}
		if _, seen := urls[name]; !seen {
			names = append(names, name)
		}
		urls[name] = url
	}
	return urls, names, nil
}

// Check makes the start checks of main() and Server(...) in their order: NAME=URL and
// NAME=PATH pairs, https station URLs (http:// only with --allow-http-for-tests), station names
// the per-station flags may name.
func Check(cfg Config) error {
	urls, names, err := stations(cfg)
	if err != nil {
		return err
	}
	for _, item := range cfg.StationHomes {
		name, path, ok := strings.Cut(item, "=")
		if !ok || name == "" || path == "" {
			return &startError{"--station-home expects NAME=PATH, got " + pyjson.StrRepr(item)}
		}
	}
	for _, name := range names {
		url := urls[name]
		if !strings.HasPrefix(url, "https://") && !(cfg.AllowHTTP && strings.HasPrefix(url, "http://")) {
			return &startError{"station " + name + ": URL must start with https://"}
		}
	}
	for _, f := range []struct {
		flag  string
		names []string
	}{{"--insecure-tls", cfg.InsecureTLS}, {"--allow-tier-b", cfg.AllowTierB}, {"--allow-tier-c", cfg.AllowTierC}} {
		unknown := map[string]bool{}
		for _, n := range f.names {
			if _, ok := urls[n]; !ok {
				unknown[n] = true
			}
		}
		if len(unknown) > 0 {
			list := make([]string, 0, len(unknown))
			for n := range unknown {
				list = append(list, n)
			}
			sort.Strings(list)
			return &startError{f.flag + " names unconfigured station(s): " + strings.Join(list, ", ")}
		}
	}
	return nil
}
