package previewshot

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func serverPort(t *testing.T, f *fixture) int {
	t.Helper()
	a := strings.Split(f.read(t, "server.args"), "\n")
	for i, s := range a {
		if s == "--port" {
			p, err := strconv.Atoi(a[i+1])
			if err != nil {
				t.Fatal(err)
			}
			return p
		}
	}
	t.Fatalf("no --port in %q", a)
	return 0
}

func TestServeStartsTheServerOnAFreePortAndShootsIt(t *testing.T) {
	f := newFixture(t)
	rc := filepath.Join(f.dir, "src", "rc")
	shot := filepath.Join(f.dir, "a.png")
	code, out, errs := f.run(t, "--serve", rc, shot, "--path", "/hmi?x=1", "--prefix", "/dash", "--mock", "m.json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	port := serverPort(t, f)
	if port < 1024 {
		t.Fatalf("port %d is not an OS-assigned ephemeral port", port)
	}
	args := f.read(t, "server.args")
	for _, want := range []string{"--rc\n" + rc + "\n", "--prefix\n/dash\n", "--mock\nm.json\n"} {
		if !strings.Contains(args, want) {
			t.Errorf("server args miss %q: %q", want, args)
		}
	}
	wantURL := "http://127.0.0.1:" + strconv.Itoa(port) + "/hmi?x=1"
	if got := f.browserArgs(t); got[len(got)-1] != wantURL {
		t.Errorf("shot URL %q, want %q", got[len(got)-1], wantURL)
	}
	if got := f.read(t, "requested"); got != "/hmi" {
		t.Errorf("the server was polled for %q before the shot, want /hmi", got)
	}
	if !strings.Contains(out, shot) {
		t.Errorf("stdout %q", out)
	}
}

func TestServeDefaultPathIsHmi(t *testing.T) {
	f := newFixture(t)
	if code, _, errs := f.run(t, "--serve", f.dir, filepath.Join(f.dir, "a.png")); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if got := f.browserArgs(t); !strings.HasSuffix(got[len(got)-1], "/hmi") {
		t.Fatalf("default path: %q", got)
	}
}

func TestServeRecordsThePIDAndStopsExactlyThatProcess(t *testing.T) {
	f := newFixture(t)
	// A bystander that a command-line pattern match would hit: the same fake server program.
	by := exec.Command(f.server, "--idle")
	by.Env = append(os.Environ(), "FAKE_SERVER_HELPER=1")
	if err := by.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = by.Process.Kill(); _ = by.Wait() }()

	if code, _, errs := f.run(t, "--serve", f.dir, filepath.Join(f.dir, "a.png")); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	pid := pidOf(t, f.read(t, "server.pid"))
	if got := pidOf(t, f.read(t, "state-seen")); got != pid {
		t.Errorf("the state file held %d while the shot ran, the server is %d", got, pid)
	}
	if !f.exists("alive") {
		t.Error("the server was not alive while the browser ran")
	}
	if alive(pid) {
		t.Errorf("server %d still running after the shot", pid)
	}
	if left, _ := filepath.Glob(filepath.Join(f.state, "server-*.pid")); len(left) != 0 {
		t.Errorf("state file left behind: %v", left)
	}
	if !alive(by.Process.Pid) {
		t.Error("a different process with the same command line was stopped")
	}
}

func TestServeStopsTheServerWhenTheShotFails(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_EXIT", "2")
	code, _, _ := f.run(t, "--serve", f.dir, filepath.Join(f.dir, "a.png"))
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if alive(pidOf(t, f.read(t, "server.pid"))) {
		t.Error("server left running after a failed shot")
	}
	if left, _ := filepath.Glob(filepath.Join(f.state, "server-*.pid")); len(left) != 0 {
		t.Errorf("state file left behind: %v", left)
	}
}

func TestServeThatNeverAnswersIsStoppedAndExit1(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_MODE", "hang")
	f.env["PREVIEW_SHOT_WAIT_MS"] = "400"
	code, _, errs := f.run(t, "--serve", f.dir, filepath.Join(f.dir, "a.png"))
	if code != 1 || !strings.Contains(errs, "did not answer") {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if alive(pidOf(t, f.read(t, "server.pid"))) {
		t.Error("hung server left running")
	}
	if f.exists("browser.args") {
		t.Error("the browser ran against a server that never answered")
	}
}

func TestServeThatExitsAtOnceIsExit1WithoutWaitingTheBudget(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_MODE", "exit")
	start := time.Now()
	code, _, errs := f.run(t, "--serve", f.dir, filepath.Join(f.dir, "a.png"))
	if code != 1 || !strings.Contains(errs, "exited") || time.Since(start) > 3*time.Second {
		t.Fatalf("exit %d after %v: %s", code, time.Since(start), errs)
	}
	if left, _ := filepath.Glob(filepath.Join(f.state, "server-*.pid")); len(left) != 0 {
		t.Errorf("state file left behind: %v", left)
	}
}

func TestMissingServerBinaryExit3(t *testing.T) {
	f := newFixture(t)
	f.env["PREVIEW_SERVER"] = filepath.Join(f.dir, "gone")
	code, _, errs := f.run(t, "--serve", f.dir, filepath.Join(f.dir, "a.png"))
	if code != 3 || !strings.Contains(errs, "PREVIEW_SERVER") {
		t.Fatalf("exit %d: %s", code, errs)
	}
}

func TestFreePortCanBeBoundAtOnce(t *testing.T) {
	p, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	if p < 1024 {
		t.Fatalf("port %d", p)
	}
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
	if err != nil {
		t.Fatalf("port %d from freePort is not free: %v", p, err)
	}
	_ = l.Close()
}
