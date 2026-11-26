package previewshot

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// browserScript is a fake headless shell: it records its arguments one per line, notes the PID file of the
// preview server it finds while it runs and whether that server is alive, then writes the screenshot
// FAKE_SHOT asks for (default: some bytes; "empty": a zero-byte file; "none": no file) and exits FAKE_EXIT.
const browserScript = `#!/bin/sh
printf '%s\n' "$@" > "$FAKE_DIR/browser.args"
for f in "$PREVIEW_SHOT_STATE"/server-*.pid; do
  [ -f "$f" ] || continue
  cp "$f" "$FAKE_DIR/state-seen"
  kill -0 "$(cat "$f")" 2>/dev/null && echo alive > "$FAKE_DIR/alive"
done
for a in "$@"; do case "$a" in --screenshot=*) out="${a#--screenshot=}";; esac; done
case "${FAKE_SHOT:-data}" in
  data) printf 'PNGDATA' > "$out";;
  empty) : > "$out";;
esac
exit "${FAKE_EXIT:-0}"
`

// serverScript is the fake preview server: the test binary itself, running TestHelperServer.
const serverScript = `#!/bin/sh
FAKE_SERVER_HELPER=1 exec "$TESTBIN" -test.run='^TestHelperServer$' -- "$@"
`

type fixture struct {
	dir, home, state, shell, server string
	env                             map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	f := &fixture{dir: dir, home: filepath.Join(dir, "home"), state: filepath.Join(dir, "state")}
	f.shell = writeExec(t, filepath.Join(dir, "shell"), browserScript)
	f.server = writeExec(t, filepath.Join(dir, "server"), serverScript)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TESTBIN", exe)
	t.Setenv("FAKE_DIR", dir)
	t.Setenv("PREVIEW_SHOT_STATE", f.state)
	f.env = map[string]string{"HOME": f.home, "PATH": filepath.Join(dir, "nobin"), "PREVIEW_SHELL": f.shell,
		"PREVIEW_SERVER": f.server, "PREVIEW_SHOT_STATE": f.state, "PREVIEW_SHOT_WAIT_MS": "5000"}
	return f
}

func writeExec(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func (f *fixture) run(t *testing.T, args ...string) (code int, out, errs string) {
	t.Helper()
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return f.env[k] }})
	return code, o.String(), e.String()
}

func (f *fixture) read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.dir, name))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return string(b)
}

func (f *fixture) exists(name string) bool {
	_, err := os.Stat(filepath.Join(f.dir, name))
	return err == nil
}

func (f *fixture) browserArgs(t *testing.T) []string {
	return strings.Split(strings.TrimSuffix(f.read(t, "browser.args"), "\n"), "\n")
}

func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

func pidOf(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		t.Fatalf("not a pid %q: %v", s, err)
	}
	return n
}

// TestHelperServer is the body of the fake preview server (run only through serverScript). FAKE_MODE: "exit"
// dies at once, "hang" never listens; otherwise it answers 200 on --port. "--idle" is a bystander that only
// sleeps. Its PID goes to $FAKE_DIR/server.pid and its arguments to server.args.
func TestHelperServer(t *testing.T) {
	if os.Getenv("FAKE_SERVER_HELPER") != "1" {
		t.Skip("helper process")
	}
	args, port := os.Args, ""
	for i, a := range args {
		if a == "--idle" {
			time.Sleep(time.Minute)
			os.Exit(0)
		}
		if a == "--port" && i+1 < len(args) {
			port = args[i+1]
		}
	}
	dir := os.Getenv("FAKE_DIR")
	_ = os.WriteFile(filepath.Join(dir, "server.args"), []byte(strings.Join(args, "\n")+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "server.pid"), []byte(fmt.Sprint(os.Getpid())), 0o644)
	switch os.Getenv("FAKE_MODE") {
	case "exit":
		os.Exit(2)
	case "hang":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	_ = http.ListenAndServe("127.0.0.1:"+port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = os.WriteFile(filepath.Join(dir, "requested"), []byte(r.URL.Path), 0o644)
		fmt.Fprint(w, "ok")
	}))
	os.Exit(1)
}
