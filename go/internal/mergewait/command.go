package mergewait

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usage = `usage: merge-when-green <pr> [--rebase|--squash|--merge] [--repo <owner/name>]
                        [--interval <duration>] [--timeout <duration>] [--log-lines <n>] [--retries <n>]
Wait for the CI checks of the PR's exact head commit, then merge that commit (gh pr merge
--match-head-commit). Default --rebase (CONTRIBUTING.md § 8: linear history), --interval 30s,
--timeout 60m, --log-lines 20, --retries 3. Every gh error is printed; one that never reached GitHub
(network unreachable, DNS, TLS handshake) is sent again up to --retries times, --interval apart. After
the merge call the PR state decides: MERGED at the checked head is success (with its merge commit),
whatever the call returned; a PR already merged at start is success too.
Exit: 0 merged; 1 a check failed (its log tail printed); 2 usage; 3 gh missing or a gh call failed;
      4 the PR is a draft; 5 the branch lacks the current base (rebase first); 6 the PR is not open
      or its head moved; 7 timed out; 8 gh pr merge refused.
`

// callDeadline ends one gh call that hangs (a stalled connection); the wait's --timeout bounds the whole run.
const callDeadline = 2 * time.Minute

var prNumber = regexp.MustCompile(`^[1-9][0-9]*$`)

// Main implements merge-when-green. Its gh is looked up on PATH; GH_REPO, or --repo, names the
// repository, else gh reads it from the git remote of the process directory.
func Main(args []string, s cli.Stdio) int {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fmt.Fprint(s.Out, usage)
			return ExitMerged
		}
	}
	c, repo, ok := parse(args, s)
	if !ok {
		return ExitUsage
	}
	if _, err := exec.LookPath("gh"); err != nil {
		fmt.Fprintf(s.Err, "%sgh is not on PATH (https://cli.github.com)\n", msgPrefix)
		return ExitGh
	}
	c.Gh = ExecGh(repo, os.Environ(), callDeadline)
	c.Sleep, c.Now = time.Sleep, time.Now
	c.Out, c.Err = s.Out, s.Err
	return Wait(c)
}

func parse(args []string, s cli.Stdio) (c Config, repo string, ok bool) {
	c = Config{Method: "rebase", Interval: 30 * time.Second, Timeout: 60 * time.Minute, LogLines: 20, Retries: 3}
	bad := func(format string, a ...any) (Config, string, bool) {
		fmt.Fprintf(s.Err, msgPrefix+format+"\n%s", append(a, usage)...)
		return c, "", false
	}
	methods := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch a {
		case "--rebase", "--squash", "--merge":
			c.Method, methods = a[2:], methods+1
		case "--repo":
			v, has := value()
			if !has || v == "" {
				return bad("--repo needs <owner/name>")
			}
			repo = v
		case "--interval", "--timeout":
			v, has := value()
			d, err := time.ParseDuration(v)
			if !has || err != nil || d < 0 {
				return bad("%s needs a duration such as 30s or 60m", a)
			}
			if a == "--interval" {
				c.Interval = d
			} else {
				c.Timeout = d
			}
		case "--retries":
			v, has := value()
			n, err := strconv.Atoi(v)
			if !has || err != nil || n < 0 {
				return bad("--retries needs a number, 0 or more")
			}
			c.Retries = n
		case "--log-lines":
			v, has := value()
			n, err := strconv.Atoi(v)
			if !has || err != nil || n < 1 {
				return bad("--log-lines needs a positive number")
			}
			c.LogLines = n
		default:
			if !prNumber.MatchString(a) || c.PR != "" {
				return bad("unexpected argument %q", a)
			}
			c.PR = a
		}
	}
	if c.PR == "" {
		return bad("no pull request number")
	}
	if methods > 1 {
		return bad("give one of --rebase, --squash, --merge")
	}
	return c, repo, true
}
