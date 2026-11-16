package install

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
)

// Usage is the help text of `mcp-n4 install`.
const Usage = `usage: mcp-n4 install [--prefix DIR]

Copies this mcp-n4 binary to DIR/mcp-n4 (default $HOME/.local/bin) atomically: a temporary
file in DIR, fsync, chmod 0755, rename. Station launchers written by 'mcp-n4 register' run it
from there, so removing or rebuilding a worktree never changes a registered server.
`

// Main is `mcp-n4 install [--prefix DIR]`: exit 0 on success (installed or already current),
// 1 when the copy fails, 2 on a usage error.
func Main(args []string, s kitcli.Stdio) int {
	fs := flag.NewFlagSet("mcp-n4 install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	prefix := fs.String("prefix", "", "")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(s.Out, Usage)
			return 0
		}
		fmt.Fprintf(s.Err, "%smcp-n4 install: error: %s\n", Usage, err)
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(s.Err, "%smcp-n4 install: error: unexpected argument %q\n", Usage, fs.Arg(0))
		return 2
	}
	if *prefix == "" {
		home := s.Env("HOME")
		if home == "" {
			fmt.Fprintln(s.Err, "mcp-n4 install: HOME is not set; pass --prefix")
			return 2
		}
		*prefix = filepath.Join(home, ".local", "bin")
	}
	src, err := executable()
	if err == nil {
		src, err = filepath.EvalSymlinks(src)
	}
	if err != nil {
		fmt.Fprintf(s.Err, "mcp-n4 install: cannot locate the running binary: %s\n", err)
		return 1
	}
	dst, changed, err := Install(src, *prefix)
	if err != nil {
		fmt.Fprintf(s.Err, "mcp-n4 install: %s\n", err)
		return 1
	}
	if changed {
		fmt.Fprintf(s.Out, "installed %s\n", dst)
	} else {
		fmt.Fprintf(s.Out, "unchanged %s (already this binary)\n", dst)
	}
	return 0
}
