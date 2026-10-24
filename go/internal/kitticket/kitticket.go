// Package kitticket is kit-ticket (toolbelt/kit-ticket.sh): it opens a kit defect issue with `gh issue create`,
// or, when gh is absent, writes the ticket to retros/tickets/<date>-<slug>.md instead. A failing gh keeps the
// ticket in that file too. Output, files and exit codes are those of the bash script it replaces, including the
// messages its shell and coreutils printed on a filesystem failure.
package kitticket

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// Exit codes of kit-ticket (unchanged from the bash tool).
const (
	ExitOK    = 0 // issue opened, or gh absent and the ticket written
	ExitUsage = 3 // usage/env error or any gh failure not typed as ExitLocal; the ticket file is kept
	ExitLocal = 4 // gh has no host, no auth or no repository context; the ticket file is kept
)

// DefaultRootEnv names the kit root used when $KIT is not a directory; ScriptEnv names the script path that the
// bash shell put in front of its own error messages. toolbelt/kit-ticket.sh exports both (its ../ and itself).
const (
	DefaultRootEnv = "KIT_TICKET_DEFAULT_ROOT"
	ScriptEnv      = "KIT_TICKET_SCRIPT"
)

const usage = "usage: kit-ticket.sh \"<title>\" [--from <retro-file>] [--repo <owner/repo>]\n"

// Host is what kit-ticket reads besides its arguments: the date, the pid that names its temp files, the
// environment gh runs with and gh's stdin.
type Host struct {
	Today func() string
	Pid   int
	Env   []string
	Stdin io.Reader
}

// Main is kit-ticket on the real host.
func Main(args []string, s cli.Stdio) int {
	return Command(Host{
		Today: func() string { return time.Now().Format("2006-01-02") },
		Pid:   os.Getpid(),
		Env:   os.Environ(),
		Stdin: os.Stdin,
	})(args, s)
}

// Command is kit-ticket over h.
func Command(h Host) cli.Main {
	return func(args []string, s cli.Stdio) int {
		r, msg := parse(args)
		if msg != "" {
			fmt.Fprint(s.Err, msg)
			return ExitUsage
		}
		t := ticket{req: r, host: h, s: s, root: kitRoot(s), script: s.Env(ScriptEnv)}
		t.dir = t.root + "/retros/tickets"
		t.name = h.Today() + "-" + slug(r.title)
		t.file = t.dir + "/" + t.name + ".md"
		gh, found := lookPath("gh", s.Env("PATH"))
		if !found {
			return t.offline()
		}
		return t.online(gh)
	}
}

type request struct{ title, from, repo string }

// parse reads `<title> [--from F] [--repo R]...` (the last of a repeated option wins); a non-empty message is a
// usage error.
func parse(args []string) (request, string) {
	if len(args) < 1 {
		return request{}, usage
	}
	r := request{title: args[0]}
	for rest := args[1:]; len(rest) > 0; {
		switch rest[0] {
		case "--from", "--repo":
			if len(rest) < 2 {
				return r, "kit-ticket: " + rest[0] + " requires a value\n"
			}
			if rest[0] == "--from" {
				r.from = rest[1]
			} else {
				r.repo = rest[1]
			}
			rest = rest[2:]
		default:
			return r, "kit-ticket: unknown option: " + rest[0] + "\n"
		}
	}
	return r, ""
}

// kitRoot is $KIT when it is a directory (as spelled), else DefaultRootEnv, else the cwd.
func kitRoot(s cli.Stdio) string {
	if k := s.Env("KIT"); k != "" && gen.IsDir(k) {
		return k
	}
	if d := s.Env(DefaultRootEnv); d != "" {
		return d
	}
	wd, _ := os.Getwd()
	return wd
}

// slug is the bash pipeline `awk tolower | sed 's/[^a-z0-9._-]/-/g' | sed 's/--*/-/g;s/^-//;s/-$//'` in the C
// locale, line by line, with the trailing newlines command substitution drops; under 6 bytes it gets "-ticket".
func slug(title string) string {
	lines := strings.Split(title, "\n")
	if strings.HasSuffix(title, "\n") {
		lines = lines[:len(lines)-1] // awk sees no record after a final newline
	}
	for i, l := range lines {
		lines[i] = slugLine(l)
	}
	raw := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if len(raw) < 6 {
		return raw + "-ticket"
	}
	return raw
}

func slugLine(l string) string {
	var b strings.Builder
	for i := 0; i < len(l); i++ {
		c := l[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			c = '-'
		}
		if c == '-' && b.Len() > 0 && b.String()[b.Len()-1] == '-' {
			continue
		}
		b.WriteByte(c)
	}
	return strings.TrimSuffix(strings.TrimPrefix(b.String(), "-"), "-")
}

// ticket is one run: the request and where its ticket goes.
type ticket struct {
	req                           request
	host                          Host
	s                             cli.Stdio
	root, dir, name, file, script string
}

// bashErr is the message bash printed when a redirection on line `line` of kit-ticket.sh failed.
func (t ticket) bashErr(line int, path string, err error) {
	fmt.Fprintf(t.s.Err, "%s: line %d: %s: %s\n", t.script, line, path, errText(err))
}

// offline is the gh-absent path: the ticket goes straight to its file and the run passes (SKIP row, exit 0).
func (t ticket) offline() int {
	if !mkdirP(t.dir, t.s.Err) {
		fmt.Fprintf(t.s.Err, "kit-ticket: cannot create tickets dir: %s\n", t.dir)
		return ExitUsage
	}
	retro := t.req.from
	if retro == "" {
		retro = "<retro file — pass --from <path> to link>"
	}
	if err := writeFile(t.file, placeholderBody(t.req.title, retro)); err != nil {
		t.bashErr(117, t.file, err)
		fmt.Fprintf(t.s.Err, "kit-ticket: cannot write ticket file: %s\n", t.file)
		return ExitUsage
	}
	fmt.Fprintf(t.s.Out, "ticket  SKIP  %s  gh absent; wrote retros/tickets/%s.md\n", retro, t.name)
	return ExitOK
}
