package kitticket

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// online is the gh-present path: the body goes to a temp file, `gh issue create` runs with its output captured
// to a second temp file, that output is printed, and on a gh failure the body is kept as the ticket file.
func (t ticket) online(gh ghPath) int {
	tmp := t.dir + "/.kit-ticket-tmp-" + strconv.Itoa(t.host.Pid)
	body, out := tmp+"-body.md", tmp+"-gh.txt"
	defer os.Remove(out)
	defer os.Remove(body)
	mkdirP(t.dir, t.s.Err) // the bash went on when this mkdir failed; the redirections below then report it
	if f, err := create(body); err != nil {
		t.bashErr(148, body, err)
	} else {
		f.WriteString(t.ghBody())
		f.Close()
	}
	rc := t.runGh(gh, body, out)
	if data, err := os.ReadFile(out); err != nil {
		fmt.Fprintf(t.s.Err, "cat: %s: %s\n", out, errText(err))
	} else {
		t.s.Out.Write(data)
	}
	if rc == 0 {
		return ExitOK
	}
	code, row := ExitUsage, "FAIL"
	if data, err := os.ReadFile(out); err != nil {
		fmt.Fprintf(t.s.Err, "grep: %s: %s\n", out, errText(err))
	} else if localFailure(data) {
		code, row = ExitLocal, "LOCAL"
	}
	retro := t.req.from
	if retro == "" {
		retro = "<no retro>"
	}
	if mv(body, t.file, t.s.Err) {
		fmt.Fprintf(t.s.Out, "ticket  %s  %s  gh issue create failed; wrote retros/tickets/%s.md (file it by hand)\n", row, retro, t.name)
	} else {
		fmt.Fprintf(t.s.Err, "kit-ticket: cannot write ticket file: %s\n", t.file)
	}
	fmt.Fprint(t.s.Err, "kit-ticket: gh issue create failed\n")
	return code
}

// runGh runs `gh issue create [--repo <words of $REPO>] --title ... --label ... --body-file <body>` with its
// stdout and stderr both in the file out, and returns its exit status as bash saw it.
func (t ticket) runGh(gh ghPath, body, out string) int {
	f, err := create(out)
	if err != nil {
		t.bashErr(157, out, err)
		return 1
	}
	defer f.Close()
	args := []string{"gh", "issue", "create"}
	if t.req.repo != "" {
		args = append(args, splitGlob("--repo "+t.req.repo)...)
	}
	args = append(args, "--title", "[kit] "+t.req.title, "--label", "kit,from-run,campaign-9", "--body-file", body)
	if !gh.exec {
		fmt.Fprintf(f, "%s: line 157: %s: Permission denied\n", t.script, gh.path)
		return 126
	}
	cmd := &exec.Cmd{Path: gh.path, Args: args, Env: childEnv(t.host.Env), Stdin: t.host.Stdin, Stdout: f, Stderr: f}
	err = cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return ee.ExitCode()
	default:
		fmt.Fprintf(f, "%s: line 157: %s: %s\n", t.script, gh.path, errText(err))
		return 126
	}
}

// localFailure is the bash `grep -qiE` (C locale, ASCII case folding) for a gh that could not even try: no
// known host, no auth, or no repository context.
func localFailure(out []byte) bool {
	lower := bytes.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, out)
	for _, p := range []string{"no known github host", "point to a known github host", "gh auth login", "not logged in", "not a git repository", "could not determine base repo"} {
		if bytes.Contains(lower, []byte(p)) {
			return true
		}
	}
	return false
}

// childEnv is the environment gh gets: the run's own, without the two variables only the wrapper sets (the bash
// never exported them to gh).
func childEnv(env []string) []string {
	var out []string
	for _, kv := range env {
		if !strings.HasPrefix(kv, DefaultRootEnv+"=") && !strings.HasPrefix(kv, ScriptEnv+"=") {
			out = append(out, kv)
		}
	}
	return out
}

// ghPath is where `command -v gh` found gh, and whether that file is executable (bash also returns a
// non-executable file when it finds no executable one, and then fails to run it).
type ghPath struct {
	path string
	exec bool
}

// lookPath is bash's `command -v name` over path (an empty entry is the cwd).
func lookPath(name, path string) (ghPath, bool) {
	var fallback *ghPath
	for _, dir := range strings.Split(path, ":") {
		if dir == "" {
			dir = "."
		}
		p := dir + "/" + name
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			if syscall.Access(p, 1) == nil {
				return ghPath{p, true}, true
			}
			if fallback == nil {
				fallback = &ghPath{p, false}
			}
		}
	}
	if fallback == nil {
		return ghPath{}, false
	}
	return *fallback, true
}

// splitGlob is bash's expansion of an unquoted $VAR: split on blanks and newlines, then each word holding *, ?
// or [ replaced by its sorted matches (dot entries only for a dot pattern), or kept when nothing matches.
func splitGlob(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		if !strings.ContainsAny(w, "*?[") {
			out = append(out, w)
			continue
		}
		m := globMatches(w)
		if len(m) == 0 {
			out = append(out, w)
			continue
		}
		out = append(out, m...)
	}
	return out
}

func globMatches(pattern string) []string {
	m, err := filepath.Glob(strings.ReplaceAll(pattern, "[!", "[^"))
	if err != nil {
		return nil
	}
	pat := strings.Split(pattern, "/")
	var keep []string
	for _, p := range m {
		hidden := false
		for i, c := range strings.Split(p, "/") {
			if i < len(pat) && strings.HasPrefix(c, ".") && !strings.HasPrefix(pat[i], ".") {
				hidden = true
			}
		}
		if !hidden {
			keep = append(keep, p)
		}
	}
	sort.Strings(keep)
	return keep
}
