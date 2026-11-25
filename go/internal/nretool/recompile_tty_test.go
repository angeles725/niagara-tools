package nretool

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// What the stub reports about its own stdout and stderr (a pipe or a file), then touches the bog it is given.
const probe = `[ -p /dev/stdout ] && echo "out:pipe" || echo "out:file"
[ -p /dev/stderr ] && echo "err:pipe" >&2 || echo "err:file" >&2
echo "$2" | sed 's/^W://' > "$PWD/../target.txt"
`
const rewrite = probe + `printf x >> "$(cat "$PWD/../target.txt")"` + "\n"

// asTerminal makes isTerminal true for the files in the test, as a terminal would be.
func asTerminal(t *testing.T) {
	t.Helper()
	old := isTerminal
	t.Cleanup(func() { isTerminal = old })
	isTerminal = func(w io.Writer) bool { _, ok := w.(*os.File); return ok }
}

func tmpFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func runFiles(t *testing.T, args ...string) (code int, out, errs string) {
	t.Helper()
	o, e := tmpFile(t), tmpFile(t)
	code = Main(args, cli.Stdio{Out: o, Err: e})
	a, _ := os.ReadFile(o.Name())
	b, _ := os.ReadFile(e.Name())
	return code, string(a), string(b)
}

func TestRecompileOnATerminalGivesNreTheRealStdoutAndStderrAndJudgesByTheBogItRewrote(t *testing.T) {
	asTerminal(t)
	home, _ := stubHome(t, "nre.exe", rewrite)
	code, out, errs := runFiles(t, "recompile", home, bogFile(t))
	if code != 0 || out != "out:file\n" || errs != "err:file\n" {
		t.Errorf("exit %d, out %q, err %q; nre must see real files, not pipes", code, out, errs)
	}
}

func TestRecompileOnATerminalReadsABogUnderAFolder(t *testing.T) {
	asTerminal(t)
	home, _ := stubHome(t, "nre.exe", probe+`printf x >> "$(cat "$PWD/../target.txt")/sub/a.bog"`+"\n")
	dir := t.TempDir()
	write(t, filepath.Join(dir, "sub", "a.bog"), "", 0o644)
	write(t, filepath.Join(dir, "sub", "notes.txt"), "", 0o644)
	if code, _, errs := runFiles(t, "recompile", home, dir); code != 0 {
		t.Errorf("exit %d, err %q", code, errs)
	}
}

func TestRecompileOnATerminalWithNoBogRewrittenIsExit1(t *testing.T) {
	asTerminal(t)
	home, _ := stubHome(t, "nre.exe", probe)
	code, _, errs := runFiles(t, "recompile", home, bogFile(t))
	if code != 1 || !strings.Contains(errs, "nre-tool: RecompileTool rewrote no bog file") {
		t.Errorf("exit %d, err %q", code, errs)
	}
}

func TestRecompileOnATerminalPassesNresExitCodeAndStartFailure(t *testing.T) {
	asTerminal(t)
	home, _ := stubHome(t, "nre.exe", "exit 4")
	if code, _, _ := runFiles(t, "recompile", home, bogFile(t)); code != 4 {
		t.Errorf("exit %d, want 4", code)
	}
	nre := filepath.Join(home, "bin", "nre.exe")
	if err := os.Chmod(nre, 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errs := runFiles(t, "recompile", home, bogFile(t))
	if code != exitEnv || !strings.Contains(errs, "nre-tool: cannot start "+nre+": ") {
		t.Errorf("exit %d, err %q", code, errs)
	}
}

func TestRecompileOnATerminalStillHandsStdinToNre(t *testing.T) {
	asTerminal(t)
	home, _ := stubHome(t, "nre.exe", rewrite+"cat > \"$PWD/../stdin.txt\"")
	old := stdin
	t.Cleanup(func() { stdin = old })
	stdin = strings.NewReader("typed-secret\n")
	if code, _, errs := runFiles(t, "recompile", home, bogFile(t)); code != 0 {
		t.Fatalf("exit %d, err %q", code, errs)
	}
	if got, _ := readFile(filepath.Join(home, "stdin.txt")); got != "typed-secret\n" {
		t.Errorf("nre read %q from stdin", got)
	}
}

func TestRecompileWithPipedOutputKeepsJudgingByNresOutput(t *testing.T) {
	home, _ := stubHome(t, "nre.exe", "[ -p /dev/stdout ] && echo '--- Processed x!'")
	var o, e bytes.Buffer
	code := Main([]string{"recompile", home, bogFile(t)}, cli.Stdio{Out: &o, Err: &e})
	if code != 0 || o.String() != "--- Processed x!\n" {
		t.Errorf("exit %d, out %q, err %q", code, o.String(), e.String())
	}
}

func TestIsTerminalIsFalseForBuffersFilesAndDevNull(t *testing.T) {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	for name, w := range map[string]io.Writer{"buffer": &bytes.Buffer{}, "file": tmpFile(t), "devnull": null} {
		if isTerminal(w) {
			t.Errorf("%s reported as a terminal", name)
		}
	}
}
