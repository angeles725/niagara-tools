package ngdeploy

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// D6: the refusal names the command that tells whether the pid of a lock is still that run, ps -p <pid>
// -o comm=, and what it must print: the program the lock records, or this one's for a lock an earlier
// version wrote with no command line.
func TestTheRefusalNamesThePsCommandAndTheProgramTheLockRecords(t *testing.T) {
	for name, tc := range map[string]struct{ content, command string }{
		"a lock that records its program": {"pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\ncommand ng-deploy\n", "ng-deploy"},
		"a lock an earlier version wrote": {"pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n", "ngdeploy.test"},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(lockFile(dir), []byte(tc.content), 0o644); err != nil {
			t.Fatal(err)
		}
		code, errs, _ := lockIn(dir)
		want := `^\[ng-deploy\] ERROR another ng-deploy run holds ` + regexp.QuoteMeta(lockFile(dir)) + ` \(pid 4194000 on host elsewhere, started 2020-01-02T03:04:05Z, \d+h\d+m\d+s ago\): ` +
			`two runs on one modules directory lose jars\. A lock is never removed for the run that made it: ` +
			regexp.QuoteMeta("when ps -p 4194000 -o comm= on elsewhere does not print "+tc.command+", that run was killed; delete the file and deploy again\n") + `$`
		if code != ExitEnv || !regexp.MustCompile(want).MatchString(errs) {
			t.Errorf("%s: exit %d, stderr %q", name, code, errs)
		}
	}
}

// D6: a lock of this host whose pid is alive but runs another program (the pid was given again after the
// run that made the lock was killed) gets a second line that says so: "not an ng-deploy run".
func TestALockWhosePidRunsAnotherProgramSaysItIsNotAnNgDeployRun(t *testing.T) {
	other := exec.Command("/bin/sleep", "60")
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { other.Process.Kill(); other.Wait() }()
	pid := strconv.Itoa(other.Process.Pid)
	// Start returns after fork, maybe before exec: wait until ps names the child sleep, or the lock
	// check reads the test binary's name (a CI flake).
	for i := 0; i < 100; i++ {
		if out, _ := exec.Command("ps", "-p", pid, "-o", "comm=").Output(); strings.TrimSpace(string(out)) == "sleep" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	host, _ := os.Hostname()
	dir := t.TempDir()
	if err := os.WriteFile(lockFile(dir), []byte("pid "+pid+"\nhost "+host+"\nstarted 2020-01-02T03:04:05Z\ncommand ng-deploy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, errs, _ := lockIn(dir)
	want := `^\[ng-deploy\] ERROR another ng-deploy run holds .*when ps -p ` + pid + ` -o comm= on ` + regexp.QuoteMeta(host) + ` does not print ng-deploy, that run was killed; delete the file and deploy again\n` +
		regexp.QuoteMeta("[ng-deploy]   pid "+pid+" runs sleep, not an ng-deploy run: the run that made the lock is gone and its pid went to another program; delete the file and deploy again\n") + `$`
	if code != ExitEnv || !regexp.MustCompile(want).MatchString(errs) {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}
