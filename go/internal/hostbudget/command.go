package hostbudget

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of host-budget (ledger.d/2026-10-22-r8-t41-host-safety.md).
const (
	ExitOK   = 0 // every row PASS, or WARN rows without --strict
	ExitFail = 1 // any FAIL row, or any WARN row under --strict
	ExitEnv  = 3 // usage error or an unreadable cgroup root
)

// defaultMount is checked when no --mount is given and it exists: the Windows C: drive under WSL,
// whose exhaustion stops WSL from starting.
const defaultMount = "/mnt/c"

// defaultMaxEntries bounds the GOCACHE walk; the 84 GB cache of 2026-10-22 is far below it in
// entries, and this host walks ~60k entries in well under a second.
const defaultMaxEntries = 2_000_000

const usage = `usage: host-budget [--cgroup-root DIR] [--proc-root DIR] [--pids-pct N] [--gocache DIR]
                   [--gocache-gb N] [--disk-pct N] [--mount PATH]... [--strict]
`

// goEnvGOCACHE asks the Go toolchain for its build cache, so GOCACHE set in `go env -w` counts too.
var goEnvGOCACHE = func() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "go", "env", "GOCACHE").Output()
	return strings.TrimSpace(string(out)), err
}

type mountList []string

func (m *mountList) String() string { return strings.Join(*m, ",") }
func (m *mountList) Set(v string) error {
	if v == "" {
		return errors.New("empty mount path")
	}
	*m = append(*m, v)
	return nil
}

// Main implements host-budget: it prints one `PASS|WARN|FAIL <resource> <used>/<limit>` row per
// resource on stdout and exits 0 (clean or WARN-only), 1 (any FAIL; any WARN under --strict) or 3.
func Main(args []string, s cli.Stdio) int {
	fl := flag.NewFlagSet("host-budget", flag.ContinueOnError)
	fl.SetOutput(s.Err)
	fl.Usage = func() { fmt.Fprint(s.Err, usage) }
	c := Config{GocacheMaxEntries: defaultMaxEntries}
	var mounts mountList
	var strict bool
	fl.StringVar(&c.CgroupRoot, "cgroup-root", "/sys/fs/cgroup", "cgroup v2 root")
	fl.StringVar(&c.ProcRoot, "proc-root", "/proc", "proc root")
	fl.IntVar(&c.PidsPct, "pids-pct", 50, "FAIL a pids counter above this percent")
	fl.StringVar(&c.GocacheDir, "gocache", "", "Go build cache (default: go env GOCACHE)")
	fl.IntVar(&c.GocacheGB, "gocache-gb", 20, "WARN the Go build cache above this many GB")
	fl.IntVar(&c.DiskPct, "disk-pct", 90, "FAIL a mount above this percent used")
	fl.Var(&mounts, "mount", "mount to check besides / (repeatable; default /mnt/c when present)")
	fl.BoolVar(&strict, "strict", false, "exit 1 on WARN too")
	if err := fl.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitEnv
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(s.Err, "host-budget: unexpected argument %q\n%s", fl.Arg(0), usage)
		return ExitEnv
	}
	for name, v := range map[string]int{"--pids-pct": c.PidsPct, "--disk-pct": c.DiskPct} {
		if v < 0 || v > 100 {
			fmt.Fprintf(s.Err, "host-budget: %s %d is not a percent (0-100)\n", name, v)
			return ExitEnv
		}
	}
	if c.GocacheGB < 0 {
		fmt.Fprintf(s.Err, "host-budget: --gocache-gb %d is negative\n", c.GocacheGB)
		return ExitEnv
	}
	if c.GocacheDir == "" {
		c.GocacheDir = resolveGocache(s)
	}
	c.Mounts = mounts
	if len(c.Mounts) == 0 {
		if st, err := os.Stat(defaultMount); err == nil && st.IsDir() {
			c.Mounts = []string{defaultMount}
		}
	}
	rows, err := Check(c, Probe{Statfs: statfs})
	if err != nil {
		fmt.Fprintf(s.Err, "host-budget: %v\n", err)
		return ExitEnv
	}
	worst := Pass
	for _, r := range rows {
		fmt.Fprintln(s.Out, r)
		worst = max(worst, r.Status)
	}
	if worst == Fail || strict && worst == Warn {
		return ExitFail
	}
	return ExitOK
}

// resolveGocache returns `go env GOCACHE`, else $GOCACHE, else the Go default
// (<user cache dir>/go-build); "" when none resolves.
func resolveGocache(s cli.Stdio) string {
	if dir, err := goEnvGOCACHE(); err == nil && dir != "" {
		return dir
	}
	if dir := s.Env("GOCACHE"); dir != "" {
		return dir
	}
	if base, err := os.UserCacheDir(); err == nil {
		return filepath.Join(base, "go-build")
	}
	return ""
}
