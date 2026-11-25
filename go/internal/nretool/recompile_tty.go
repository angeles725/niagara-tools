package nretool

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// isTerminal is true for a terminal (a character device that is not /dev/null); tests replace it.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(st, null)
}

// onTerminal is true when the user watches both streams, so RecompileTool's password prompt gets them
// (RecompileTool.java:110-111: a Java console read that may need a terminal stdout; echo through a pipe is unproven).
func onTerminal(s cli.Stdio) bool { return isTerminal(s.Out) && isTerminal(s.Err) }

// stamp is what a bog file looked like before nre ran.
type stamp struct {
	size int64
	mod  time.Time
}

// bogStamps is every *.bog at or under target (target itself when it is a file), by path.
func bogStamps(target string) map[string]stamp {
	m := map[string]stamp{}
	_ = filepath.WalkDir(target, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".bog") {
			if st, e := d.Info(); e == nil {
				m[p] = stamp{st.Size(), st.ModTime()}
			}
		}
		return nil
	})
	return m
}

// runOnTerminal runs nre with the real stdin, stdout and stderr, so nre owns the terminal for the prompt. Its output
// cannot be read then, so success is judged from the bog files: nre's exit code, then at least one bog rewritten
// (the processed-file list, read off the disk: RecompileTool saves each bog it prints "--- Processed" for).
func runOnTerminal(nre string, args []string, target string, s cli.Stdio) int {
	before := bogStamps(target)
	cmd := exec.Command(nre, args...)
	cmd.Dir = filepath.Dir(nre)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, s.Out.(*os.File), s.Err.(*os.File)
	err := cmd.Run()
	if _, exited := err.(*exec.ExitError); err != nil && !exited {
		fmt.Fprintf(s.Err, "nre-tool: cannot start %s: %v\n", nre, err)
		return exitEnv
	}
	if code := exitCode(err); code != 0 {
		return code
	}
	for p, was := range bogStamps(target) {
		if b, ok := before[p]; !ok || b != was {
			return 0
		}
	}
	fmt.Fprintln(s.Err, "nre-tool: RecompileTool rewrote no bog file (a missing file or a bad signing key exits 0; "+
		"nre's output was on the terminal, so the helper judges by the bog files)")
	return exitFound
}
