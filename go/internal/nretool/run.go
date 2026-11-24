package nretool

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// stdin is what a tool that prompts reads (nre prompts on the terminal); tests replace it.
var stdin io.Reader = os.Stdin

// findNre is the install's nre launcher: nre.exe (Windows install, run from WSL) or nre.
func findNre(home string) string {
	for _, name := range []string{"nre.exe", "nre"} {
		p := filepath.Join(home, "bin", name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// nativePath is p as the launcher reads a path: a Windows nre.exe gets the Windows form (wslpath -w), as nre.exe
// cannot open /home/... paths; with no wslpath, or for a Linux nre, p is unchanged.
func nativePath(nre, p string) string {
	if !strings.HasSuffix(nre, ".exe") {
		return p
	}
	w, err := exec.Command("wslpath", "-w", p).Output()
	if err != nil || strings.TrimSpace(string(w)) == "" {
		return p
	}
	return strings.TrimSpace(string(w))
}

// capture keeps a copy of everything written through its tees, so a caller reads the tool's output after it
// streamed to the terminal; stdout and stderr share one buffer in arrival order.
type capture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

type tee struct {
	c *capture
	w io.Writer
}

func (c *capture) tee(w io.Writer) io.Writer { return tee{c, w} }

func (t tee) Write(p []byte) (int, error) {
	t.c.mu.Lock()
	t.c.buf.Write(p)
	t.c.mu.Unlock()
	return t.w.Write(p)
}

// runStreaming runs nre class args from the install's bin directory with stdin handed through and stdout and stderr
// streamed to s; it returns nre's exit code and everything it printed (CR removed).
func runStreaming(nre string, args []string, s cli.Stdio) (int, string) {
	var c capture
	cmd := exec.Command(nre, args...)
	cmd.Dir = filepath.Dir(nre)
	cmd.Stdin = stdin
	cmd.Stdout, cmd.Stderr = c.tee(s.Out), c.tee(s.Err)
	err := cmd.Run()
	return exitCode(err), strings.ReplaceAll(c.buf.String(), "\r", "")
}

func exitCode(err error) int {
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	if err != nil {
		return exitUsage
	}
	return 0
}

// tail is the last n non-empty lines of out.
func tail(out []byte, n int) string {
	var lines []string
	for _, l := range strings.Split(string(bytes.ReplaceAll(out, []byte("\r"), nil)), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, "  "+l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
