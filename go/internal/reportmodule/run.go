package reportmodule

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// maxOutput bounds what one member run may hand back (bash kept all of it), so a runaway member cannot exhaust
// memory; a run past it lost rows and is reported as an env fault.
const maxOutput = 64 << 20

// runner executes the toolbelt members: tb is the toolbelt directory, env the members' environment.
type runner struct {
	tb  string
	env []string
	max int // output bound of one run (0: maxOutput); a test seam
}

// bound is the output bound of one run.
func (x runner) bound() int {
	if x.max > 0 {
		return x.max
	}
	return maxOutput
}

// member runs <tb>/<script>.sh with args, like the shell's `out=$("$TOOLBELT/<script>.sh" args 2>&1)`.
func (x runner) member(script string, args ...string) (string, int, bool) {
	return x.run("", x.tb+"/"+script+".sh", args...)
}

// run executes path in dir ("" the cwd) and returns its stdout and stderr interleaved, like "$(cmd 2>&1)" (NUL
// bytes and trailing newlines dropped), and its exit status: 127 not found, 126 not executable or not an
// executable format, 128+n killed by signal n. Bash's own "No such file or directory" / "Permission denied"
// line for those is not in the output, and a file with no #! line is never run as a shell script (ENOEXEC is
// 126). ok is false when the output passed the bound and its rest was dropped.
func (x runner) run(dir, path string, args ...string) (out string, code int, ok bool) {
	w := &capWriter{max: x.bound()}
	code = x.exec(dir, w, path, args...)
	return strings.TrimRight(strings.ReplaceAll(w.b.String(), "\x00", ""), "\n"), code, w.dropped == 0
}

// quiet runs path with its output discarded (`cmd >/dev/null 2>&1`) and returns its exit status.
func (x runner) quiet(path string, args ...string) int { return x.exec("", io.Discard, path, args...) }

func (x runner) exec(dir string, w io.Writer, path string, args ...string) int {
	cmd := exec.Command(path, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr, cmd.Env = dir, w, w, x.env
	return exitCode(cmd.Run())
}

// firstLine runs path in the cwd and returns the first line of its stdout, like `cmd 2>/dev/null | head -1`
// in "$(...)": the pipe is closed after that line (the writer gets SIGPIPE, as when head exits) and the line
// is bounded by maxOutput.
func (x runner) firstLine(path string, args ...string) string {
	cmd := exec.Command(path, args...)
	cmd.Env = x.env
	out, err := cmd.StdoutPipe()
	if err != nil {
		return ""
	}
	if cmd.Start() != nil {
		return ""
	}
	line, _ := bufio.NewReaderSize(io.LimitReader(out, maxOutput), 4096).ReadString('\n')
	out.Close()
	_ = cmd.Wait()
	return strings.TrimRight(strings.ReplaceAll(line, "\x00", ""), "\n")
}

func exitCode(err error) int {
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return ee.ExitCode()
	case errors.Is(err, fs.ErrNotExist):
		return 127
	default:
		return 126
	}
}

// capWriter keeps the first max bytes written to it and counts the rest.
type capWriter struct {
	b       bytes.Buffer
	max     int
	dropped int
}

func (c *capWriter) Write(p []byte) (int, error) {
	keep := min(len(p), max(c.max-c.b.Len(), 0))
	c.b.Write(p[:keep])
	c.dropped += len(p) - keep
	return len(p), nil
}

// commandV is bash's `command -v <name>` for a name without a slash: the first PATH entry (an empty entry is
// ".") holding an executable non-directory <name>, else the first one holding a non-executable file of that
// name (bash's last resort), else "".
func commandV(path, name string) string {
	lose := ""
	for _, d := range strings.Split(path, ":") {
		if d == "" {
			d = "."
		}
		full := strings.TrimSuffix(d, "/") + "/" + name
		st, err := os.Stat(full)
		if err != nil || st.IsDir() {
			continue
		}
		if isExec(full) {
			return full
		}
		if lose == "" {
			lose = full
		}
	}
	return lose
}

// isDir, isFile and isExec are the shell's [ -d ], [ -f ] and [ -x ] (symlinks followed; [ -x ] holds for a
// searchable directory too).
func isDir(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

func isFile(p string) bool { st, err := os.Stat(p); return err == nil && st.Mode().IsRegular() }

func isExec(p string) bool { return p != "" && syscall.Access(p, 1) == nil }

// isRealDir is a directory that is not a symlink: what find -P descends into.
func isRealDir(p string) bool { st, err := os.Lstat(p); return err == nil && st.IsDir() }
