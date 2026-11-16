// Package register writes a station's mcp-n4 launcher and its MCP server entry for one agent
// harness (`mcp-n4 register`), so every harness runs the stable binary `mcp-n4 install` put in place.
package register

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/envfile"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
)

// Usage is the help text of `mcp-n4 register`.
const Usage = `usage: mcp-n4 register --station NAME --harness claude|pi|gentle-shell
                       [--env PATH] [--bin PATH] [--project DIR] [--force] [-- SERVER-FLAGS...]

Writes ~/.config/mcp-n4/NAME-run.sh (sources the env file, default ~/.config/mcp-n4/NAME.env,
then execs --bin, default ~/.local/bin/mcp-n4, with SERVER-FLAGS; it holds no credential) and the
MCP server entry mcp-n4-NAME for the harness: claude in ~/.claude.json for the project --project
(default: the current directory), pi in ~/.pi/agent/mcp.json, gentle-shell in
~/.gentle-shell/agent/mcp.json. Without SERVER-FLAGS an existing launcher is kept as is; an
explicit --bin or --env must then match what it execs or sources, else pass SERVER-FLAGS to rewrite it.
Re-running is a no-op; a differing launcher or entry is refused unless --force.
`

var stationName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type options struct {
	station, harness, env, bin, project, home string
	force                                     bool
	serverArgs                                []string
	explicit                                  map[string]bool // the flags given on the command line
}

// Main is `mcp-n4 register`: exit 0 when written or unchanged, 1 on a conflict or a failed
// write, 2 on a usage error. Nothing is written unless every check passed.
func Main(args []string, s kitcli.Stdio) int {
	o, err := parse(args, s)
	if err == nil {
		err = register(o, s.Out)
	}
	var ue *usageError
	switch {
	case err == flag.ErrHelp:
		fmt.Fprint(s.Out, Usage)
		return 0
	case errors.As(err, &ue):
		fmt.Fprintf(s.Err, "%smcp-n4 register: error: %s\n", Usage, err)
		return 2
	case err != nil:
		fmt.Fprintf(s.Err, "mcp-n4 register: %s\n", err)
		return 1
	}
	return 0
}

func parse(args []string, s kitcli.Stdio) (options, error) {
	var o options
	fs := flag.NewFlagSet("mcp-n4 register", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.station, "station", "", "")
	fs.StringVar(&o.harness, "harness", "", "")
	fs.StringVar(&o.env, "env", "", "")
	fs.StringVar(&o.bin, "bin", "", "")
	fs.StringVar(&o.project, "project", "", "")
	fs.BoolVar(&o.force, "force", false, "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return o, err
		}
		return o, &usageError{err.Error()}
	}
	o.serverArgs = fs.Args()
	o.explicit = map[string]bool{}
	fs.Visit(func(f *flag.Flag) { o.explicit[f.Name] = true })
	if o.home = s.Env("HOME"); o.home == "" {
		return o, usageErrorf("HOME is not set")
	}
	if !stationName.MatchString(o.station) {
		return o, usageErrorf("--station must match %s, got %q", stationName, o.station)
	}
	cfgDir := filepath.Join(o.home, ".config", "mcp-n4")
	if o.env == "" {
		o.env = filepath.Join(cfgDir, o.station+".env")
	}
	if o.bin == "" {
		o.bin = filepath.Join(o.home, ".local", "bin", "mcp-n4")
	}
	if o.project == "" {
		o.project, _ = os.Getwd()
	} else if abs, err := filepath.Abs(o.project); err == nil {
		o.project = abs // Claude Code keys projects by absolute path
	}
	return o, nil
}

func register(o options, out io.Writer) error {
	name := "mcp-n4-" + o.station
	launcher := filepath.Join(o.home, ".config", "mcp-n4", o.station+"-run.sh")
	tgt, err := harnessTarget(o.harness, o.home, o.project, name, launcher)
	if err != nil {
		return err
	}
	text, err := launcherText(o, launcher)
	if err != nil {
		return err
	}
	_, next, err := tgt.plan(o.force)
	if err != nil {
		return err
	}
	if err := writeIfChanged(out, "launcher "+launcher, launcher, text, 0o755); err != nil {
		return err
	}
	label := fmt.Sprintf("%s entry %s in %s", o.harness, name, tgt.file)
	if next == nil {
		fmt.Fprintf(out, "unchanged %s\n", label)
		return nil
	}
	if err := writeFile(tgt.file, next, 0); err != nil {
		return err
	}
	fmt.Fprintf(out, "written %s\n", label)
	return nil
}

// launcherText is the launcher to write, or nil to keep the existing one (no server flags).
func launcherText(o options, launcher string) ([]byte, error) {
	old, rerr := os.ReadFile(launcher)
	if len(o.serverArgs) == 0 {
		if rerr != nil {
			return nil, usageErrorf("no server flags after -- and no launcher %s to keep", launcher)
		}
		return nil, checkKept(o, launcher, string(old))
	}
	env, found, err := envfile.Load(o.env, func(string) (string, bool) { return "", false })
	if err != nil || !found {
		return nil, usageErrorf("env file %s: not readable as plain assignments (%v)", o.env, err)
	}
	cfg, _, uerr := mcpcli.Parse(o.serverArgs)
	if uerr != nil {
		return nil, usageErrorf("server flags: %s", uerr.Msg)
	}
	if err := mcpcli.Check(cfg); err != nil {
		return nil, usageErrorf("server flags: %s", err)
	}
	if keys := secretInArgs(o.serverArgs, env); len(keys) > 0 {
		return nil, usageErrorf("server flags carry the value of %v from the env file; credentials stay in the env file", keys)
	}
	if git := gitTreeOf(o.bin); git != "" && !o.force {
		return nil, fmt.Errorf("--bin %s is inside a git checkout (%s): a worktree removed or rebuilt breaks every registered server; run 'mcp-n4 install' and use its path, or rerun with --force",
			o.bin, git)
	}
	text := []byte(renderLauncher(o.station, o.env, o.bin, o.serverArgs))
	if rerr == nil && string(old) != string(text) && !o.force {
		return nil, fmt.Errorf("launcher %s of station %s differs from the one to write (first at line %d); rerun with --force to replace it",
			launcher, o.station, firstDiffLine(string(old), string(text)))
	}
	return text, nil
}
