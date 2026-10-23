// Package cli is the mcp-n4 command line (mcp-n4-kit/mcp_n4/server.py:191-312): argparse's
// flags, bounded ranges and error text, the start checks main() and Server(...) make, then the
// stdio loop. Until the write slices land (S10), --allow-writes is refused at start.
package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// Prog is the argparse prog name every message starts with.
const Prog = "mcp_n4.server"

// AllowWritesRefusal is the start refusal of --allow-writes in this version.
const AllowWritesRefusal = "--allow-writes " + jsonrpc.NotImplemented

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
	if err := jsonrpc.New(reg).Serve(in, s.Out); err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", Prog, err)
		return 1
	}
	return 0
}

type startError struct{ msg string }

func (e *startError) Error() string { return e.msg }

// Check makes the start checks of main() and Server(...) in their order: NAME=URL and
// NAME=PATH pairs, https station URLs (http:// only with --allow-http-for-tests), station names
// the per-station flags may name; then the --allow-writes refusal of this version.
func Check(cfg Config) error {
	urls := map[string]string{}
	var names []string // insertion order of the stations dict
	for _, item := range cfg.Stations {
		name, url, ok := strings.Cut(item, "=")
		if !ok || name == "" || url == "" {
			return &startError{"--station expects NAME=URL, got " + pyjson.StrRepr(item)}
		}
		if _, seen := urls[name]; !seen {
			names = append(names, name)
		}
		urls[name] = url
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
	if cfg.AllowWrites {
		return &startError{AllowWritesRefusal}
	}
	return nil
}
