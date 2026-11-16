package ngdeploy

import (
	"bytes"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// lockFile is the run lock of a modules directory.
func lockFile(dir string) string { return dir + "/.ng-deploy.lock" }

// held is a gradlew that stops where a test wants it: at the call whose arguments match pattern it writes a
// line to one named pipe, which waits for the test to read it (arrived), and then reads another, which
// waits for the test to write (release). Nothing depends on how long anything takes.
type held struct{ at, on string }

func holdGradlew(t *testing.T, f *fixture, pattern, then string) *held {
	t.Helper()
	h := &held{at: f.dir + "/at.fifo", on: f.dir + "/on.fifo"}
	for _, p := range []string{h.at, h.on} {
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.write(f.dir+"/do/gradlew", `case "$*" in `+pattern+`) echo here > "`+h.at+`"; read -r _ < "`+h.on+`" ;; esac`+"\n"+then)
	return h
}

func (h *held) arrived(t *testing.T) {
	t.Helper()
	if _, err := os.ReadFile(h.at); err != nil {
		t.Fatal(err)
	}
}

func (h *held) release(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(h.on, []byte("go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ran is how a run started beside the test ended.
type ran struct {
	code     int
	out, err string
}

// start is run on a goroutine of its own: a second deploy of the same env file, as another terminal starts.
func (f *fixture) start(args ...string) <-chan ran {
	args = append([]string{"--env-file", f.env}, args...)
	done := make(chan ran, 1)
	go func() {
		var out, errs bytes.Buffer
		code := Main(args, cli.Stdio{Out: &out, Err: &errs})
		done <- ran{code, out.String(), errs.String()}
	}()
	return done
}

var lockedLine = regexp.MustCompile(`^\[ng-deploy\] ERROR another ng-deploy run holds (\S+) \(pid (\d+) on host (\S+), started (\S+), (\S+) ago\): two runs on one modules directory lose jars\. ` +
	`A lock is never removed for the run that made it: when ps -p (\d+) -o comm= on (\S+) does not print (\S+), that run was killed; delete the file and deploy again\n$`)

// Two runs on one <NIAGARA_HOME>/modules: the second starts while the first is held in the build (its jars
// are aside: a second run that looked would record "the home had none") or in slotomatic (before the move
// aside: a second run would move the same jars aside over the first's). The lock refuses the second before
// it looks at anything, and when the first then fails its gate the directory holds the original jars.
func TestASecondRunOnTheSameHomeIsRefusedAndTheJarsOfTheFirstSurvive(t *testing.T) {
	for name, tc := range map[string]struct {
		pattern string
		args    []string
		aside   bool
	}{
		"the first is in the build, its jars aside":     {"*:jar*", []string{"--no-backup"}, true},
		"the first is in slotomatic, nothing aside yet": {"*slotomatic*", []string{"--no-backup", "--with-slotomatic"}, false},
	} {
		f := newFixture(t)
		modules := withHome(t, f, true)
		t.Setenv("FAKE_VM_EXIT", "1")
		before := tree(t, f.dir+"/home")
		h := holdGradlew(t, f, tc.pattern, installing(f.dir+"/home", "", "rt", "ux"))
		first := f.start(tc.args...)
		h.arrived(t)
		if f.exists(modules+"/demo-rt.jar"+asideSuffix) != tc.aside {
			t.Fatalf("%s: the first run is not where the test wants it:\n%s", name, tree(t, modules))
		}
		during := tree(t, f.dir)
		code := f.run("--no-backup")
		m := lockedLine.FindStringSubmatch(f.err)
		if code != ExitEnv || m == nil || m[1] != lockFile(modules) || f.out != "" || tree(t, f.dir) != during {
			t.Errorf("%s: the second run: exit %d, stdout %q, stderr %q", name, code, f.out, f.err)
		}
		h.release(t)
		if r := <-first; r.code != ExitVerify || !strings.Contains(r.err, "verify-module gate failed") {
			t.Errorf("%s: the first run: exit %d, stderr %q", name, r.code, r.err)
		}
		if after := tree(t, f.dir+"/home"); after != before {
			t.Errorf("%s: <NIAGARA_HOME> is not as before the two runs:\n%swas:\n%s", name, after, before)
		}
		if f.exists(lockFile(modules)) || f.exists(lockFile(f.modules)) {
			t.Errorf("%s: a lock outlived its run", name)
		}
	}
}

// While it runs, a deploy holds a lock in <NIAGARA_HOME>/modules and in the station directory that names
// its process, its host and when it started; whichever way the run ends, both are gone: a deploy that went
// through, a check that failed, a SIGTERM during the build, with a niagara_home or without one.
func TestTheLocksNameTheRunAndAreGoneWhenItEnds(t *testing.T) {
	for name, tc := range map[string]struct {
		home    bool
		arrange func(f *fixture)
		after   string
		code    int
	}{
		"a deploy that went through":      {true, func(*fixture) {}, "", ExitOK},
		"a check that failed":             {true, func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, "", ExitVerify},
		"a SIGTERM during the build":      {true, func(*fixture) {}, "kill -TERM $PPID; command -p sleep 0.5; exit 143", 128 + int(syscall.SIGTERM)},
		"no niagara_home, went through":   {false, func(*fixture) {}, "", ExitOK},
		"no niagara_home, a SIGTERM":      {false, func(*fixture) {}, "kill -TERM $PPID; command -p sleep 0.5; exit 143", 128 + int(syscall.SIGTERM)},
		"no niagara_home, the copy fails": {false, func(f *fixture) { failRename(f.t, renameInUX) }, "", ExitCopy},
	} {
		f := newFixture(t)
		rename, link = os.Rename, os.Link // what an earlier case made fail
		own := make(chan os.Signal, 1)    // keeps a signal nothing else catches from ending the test binary
		signal.Notify(own, syscall.SIGTERM)
		home, seen := f.dir+"/home", f.dir+"/locks.seen"
		if tc.home {
			withHome(t, f, true)
		}
		tc.arrange(f)
		f.write(f.dir+"/do/gradlew", `case "$*" in *:jar*)`+"\n"+`for l in "`+lockFile(home+"/modules")+`" "`+lockFile(f.modules)+`"; do if [ -e "$l" ]; then echo "$(<"$l")"; else echo absent; fi; done > "`+seen+`"`+"\n"+
			map[bool]string{true: `cp "${0%/*}/demo/demo-rt/build/libs/demo-rt.jar" "` + home + `/modules/demo-rt.jar"; cp "${0%/*}/demo/demo-ux/build/libs/demo-ux.jar" "` + home + `/modules/demo-ux.jar"` + "\n"}[tc.home]+
			tc.after+"\n;; esac\nexit 0")
		code := f.run("--no-backup")
		signal.Stop(own)
		host, _ := os.Hostname()
		holder := regexp.QuoteMeta("pid "+strconv.Itoa(os.Getpid())+"\nhost "+host+"\nstarted ") + `\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ\ncommand ngdeploy\.test\n`
		want := "^" + holder + holder + "$"
		if !tc.home {
			want = "^absent\n" + holder + "$"
		}
		if code != tc.code || !regexp.MustCompile(want).MatchString(f.read(seen)) {
			t.Errorf("%s: exit %d, stderr %q, the locks during the build:\n%s", name, code, f.err, f.read(seen))
		}
		if f.exists(lockFile(home+"/modules")) || f.exists(lockFile(f.modules)) {
			t.Errorf("%s: a lock outlived its run", name)
		}
	}
}

// A lock a killed run left is refused (exit 10) and never removed, in the home or in the station, whatever it
// holds: the line says who made it, how old it is and how to tell that the run is gone. Nothing ran, and a
// lock this run took before it met the other is released.
func TestALockAKilledRunLeftIsRefusedAndLeftAlone(t *testing.T) {
	const left = "pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n"
	for name, tc := range map[string]struct {
		inStation bool
		content   string
		holder    string
	}{
		"in the home":                   {false, left, `\(pid 4194000 on host elsewhere, started 2020-01-02T03:04:05Z, \d+h\d+m\d+s ago\)`},
		"in the station":                {true, left, `\(pid 4194000 on host elsewhere, started 2020-01-02T03:04:05Z, \d+h\d+m\d+s ago\)`},
		"a lock that names no run":      {false, "", `\(it does not say which run made it; the file is \S+ old\)`},
		"a lock with something else in": {true, "hello\n", `\(it does not say which run made it; the file is \S+ old\)`},
	} {
		f := newFixture(t)
		modules := withHome(t, f, true)
		lock := lockFile(modules)
		if tc.inStation {
			lock = lockFile(f.modules)
		}
		f.write(lock, tc.content)
		before := tree(t, f.dir)
		code := f.run("--no-backup", "--with-slotomatic")
		want := `^\[ng-deploy\] ERROR another ng-deploy run holds ` + regexp.QuoteMeta(lock) + " " + tc.holder + `: two runs on one modules directory lose jars\. A lock is never removed for the run that made it: `
		if code != ExitEnv || !regexp.MustCompile(want).MatchString(f.err) || strings.Count(f.err, "\n") != 1 || f.out != "" || f.calls() != "" || tree(t, f.dir) != before {
			t.Errorf("%s: exit %d, stdout %q, stderr %q, calls %q\n%s", name, code, f.out, f.err, f.calls(), tree(t, f.dir))
		}
	}
}

// A run that builds and does not deploy changes no station: with --no-deploy and no niagara_home it takes no
// lock, and a lock in the station does not stop it.
func TestABuildThatDeploysNothingDoesNotLockTheStation(t *testing.T) {
	f := newFixture(t)
	f.write(lockFile(f.modules), "pid 4194000\nhost elsewhere\nstarted 2020-01-02T03:04:05Z\n")
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] no-deploy: jars in build/libs/\n")
	if !f.exists(lockFile(f.modules)) {
		t.Error("the lock of another run was removed")
	}
}
