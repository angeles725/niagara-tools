package buildmodule

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A signal passed to gradle's own pid does not reach what gradle started (retro p1-build-deploy-safety Δ10): a
// single-use daemon it forked can install its jar after the restore. The fake gradle leaves a background job
// that copies the new jar 0.6 s later and then ends on the signal; the handler ends the job with it, so the
// jar stays the pre-build one. Gradle stays in the terminal's foreground process group, so Ctrl-C still reaches
// it the way it did.
func TestAGrandchildOfGradleIsStoppedBeforeTheRestore(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.jar(f.deployed("rt"), "1.0.0", "old")
	f.jar(f.built("rt"), "1.0.1", "new")
	before := f.read(f.deployed("rt"))
	code := -1
	exitProcess = func(c int) { code = c }
	t.Cleanup(func() { exitProcess = os.Exit })
	f.does("gradlew", "trap 'exit 1' INT TERM HUP\n"+
		`( sleep 0.6; cp "`+f.built("rt")+`" "`+f.deployed("rt")+`" ) >/dev/null 2>&1 &`+"\n"+
		"sleep 0.1\nkill -TERM $PPID\nwait\n")
	f.build()
	if code != 143 {
		t.Errorf("exit %d, want 143\nstderr: %s", code, f.err)
	}
	time.Sleep(1200 * time.Millisecond) // the job's moment, had it been left running
	if f.read(f.deployed("rt")) != before {
		t.Errorf("modules/Foo-rt.jar is the jar gradle's background job installed after the restore\nstderr: %s", f.err)
	}
}

// procComm is the command name of a process, "" when it is gone.
func procComm(pid int) string {
	b, _ := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	return strings.TrimSpace(string(b))
}

// descendants lists every live process below a pid, the children's children too, and survivors ends them.
func TestDescendantsFindsGrandchildrenAndKillStopsThem(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 30 & wait")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	var kin []procRef
	for i := 0; i < 100 && len(kin) == 0; i++ {
		time.Sleep(20 * time.Millisecond)
		kin = descendants(cmd.Process.Pid)
	}
	if len(kin) != 1 {
		t.Fatalf("descendants of the shell: %v, want its one sleep", kin)
	}
	if !strings.Contains(procComm(kin[0].pid), "sleep") {
		t.Errorf("the descendant is %q, want sleep", procComm(kin[0].pid))
	}
	killAll(kin)
	for i := 0; i < 100 && alive(kin[0]); i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if alive(kin[0]) {
		t.Error("the descendant is still running after killAll")
	}
}

// A pid that now belongs to another process (another start time) is never killed.
func TestKillAllLeavesAReusedPidAlone(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	killAll([]procRef{{pid: cmd.Process.Pid, start: "1"}})
	time.Sleep(100 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Errorf("a process with another start time was killed: %v", err)
	}
}
