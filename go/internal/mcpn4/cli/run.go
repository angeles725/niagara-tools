// Package cli is the mcp-n4 command line (mcp-n4-kit/mcp_n4/server.py:191-312): argparse's
// flags, bounded ranges and error text, the start checks main() and Server(...) make, then the
// stdio loop. With --allow-writes the write state is checked at start and the write tools are
// served (writeops).
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// Prog is the program name the usage line and every message start with: the binary.
const Prog = "mcp-n4"

// Main is the kitcli.Main entry point: it serves the process stdin until EOF or the first SIGINT
// or SIGTERM (watchSignals). SIGPIPE is ignored, as Python ignores it: a client gone mid-reply
// fails the write (exit 1, session closed) instead of killing the server.
func Main(args []string, s kitcli.Stdio) int {
	signal.Ignore(syscall.SIGPIPE)
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	return RunContext(watchSignals(sigs, os.Exit), args, os.Stdin, s)
}

// Run is RunContext with no interrupt.
func Run(args []string, in io.Reader, s kitcli.Stdio) int {
	return RunContext(context.Background(), args, in, s)
}

// RunContext parses args, makes the start checks and serves in until EOF or until interrupt is
// done. Exit codes are server.py's: 0 after EOF, an interrupt (or --help), 2 on a flag error or a
// refused start, 1 when the stdio loop fails.
func RunContext(interrupt context.Context, args []string, in io.Reader, s kitcli.Stdio) int {
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
	srv, err := newServer(interrupt, cfg)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", Prog, err)
		return 2
	}
	if err := srv.ServeContext(interrupt, in, s.Out); err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", Prog, err)
		return 1
	}
	return 0
}

// NewServer is Server(...) for a cfg Check accepted: the read tools and, with --allow-writes,
// the write state (WriteState's start checks: limits, state dir, scopes) and the write tools.
func NewServer(cfg Config) (*jsonrpc.Server, error) { return newServer(context.Background(), cfg) }

// newServer is NewServer whose station requests interrupt cancels.
func newServer(interrupt context.Context, cfg Config) (*jsonrpc.Server, error) {
	reg := registry.Default()
	srv := jsonrpc.New(reg)
	ctx := NewContext(cfg)
	factory := func(bc box.Config, onAuthError func(*box.Error)) (readtools.Client, error) {
		bc.Interrupt = interrupt
		return readtools.BoxFactory(bc, onAuthError)
	}
	tools := &readtools.Tools{Ctx: ctx, NewClient: factory, ServerVersion: reg.Version, NiagaraHomes: niagaraHomes(cfg)}
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
