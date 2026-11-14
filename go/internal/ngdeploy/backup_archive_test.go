package ngdeploy

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// These tests replace the tests/ng-deploy.bats cases that sourced the launcher and called backup on its own
// with the host's tar: the other backup tests read the arguments a stub tar received, these read the archive
// a real tar wrote.

// archiveFixture is a fixture with the host's tar (and the gzip its -z runs) on PATH in place of the stub.
func archiveFixture(t *testing.T) *fixture {
	t.Helper()
	real := map[string]string{}
	for _, tool := range []string{"tar", "gzip"} { // looked up before the fixture narrows PATH to its stubs
		path, err := exec.LookPath(tool)
		if err != nil {
			t.Fatalf("these tests need the host's %s: %v", tool, err)
		}
		real[tool] = path
	}
	f := newFixture(t)
	for tool, path := range real {
		os.Remove(f.bin + "/" + tool)
		if err := os.Symlink(path, f.bin+"/"+tool); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

// archiveMembers is the sorted list of regular files in the one backup of the module the run left.
func archiveMembers(f *fixture) []string {
	f.t.Helper()
	backups, _ := filepath.Glob(f.work + "/" + backupDir + "/demo-pre-*.tar.gz")
	if len(backups) != 1 {
		f.t.Fatalf("backups: %v, want one", backups)
	}
	file, err := os.Open(backups[0])
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		f.t.Fatal(err)
	}
	var members []string
	for r := tar.NewReader(unzipped); ; {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			f.t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			members = append(members, h.Name)
		}
	}
	sort.Strings(members)
	return members
}

func TestTheDefaultBackupHoldsOnlyTheJarsOfTheModule(t *testing.T) {
	f := archiveFixture(t)
	f.deployed()
	f.write(f.modules+"/Other-rt.jar", "a sibling module") // must not be in the backup
	f.want(f.run("--no-deploy"), ExitOK)
	if want := "[ng-deploy] backup: _backups/demo-pre-TS.tar.gz\n[ng-deploy] backup ok: _backups/demo-pre-TS.tar.gz\n"; !strings.HasPrefix(stamp(f.out), want) || f.err != "" {
		t.Errorf("stdout %q, stderr %q", f.out, f.err)
	}
	if got := strings.Join(archiveMembers(f), " "); got != "demo-rt.jar demo-ux.jar" {
		t.Errorf("archived %q", got)
	}
}

func TestAFullBackupHoldsTheWholeModulesDirectory(t *testing.T) {
	f := archiveFixture(t)
	f.write(f.deployedJar("rt"), "old-rt")
	f.write(f.modules+"/Other-rt.jar", "a sibling module")
	t.Setenv("FULL_BACKUP", "1")
	f.want(f.run("--no-deploy"), ExitOK, "[ng-deploy] backup ok: _backups/demo-pre-")
	if got := strings.Join(archiveMembers(f), " "); got != "modules/Other-rt.jar modules/demo-rt.jar" || f.err != "" {
		t.Errorf("archived %q, stderr %q", got, f.err)
	}
}

func TestAFifthBackupLeavesTheThreeNewest(t *testing.T) {
	f := archiveFixture(t)
	f.write(f.deployedJar("rt"), "old-rt")
	dir := f.work + "/" + backupDir
	old := func(day string) string { return dir + "/demo-pre-2026010" + day + "-000000.tar.gz" }
	for i, day := range []string{"1", "2", "3", "4"} {
		f.write(old(day), "")
		when := time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(old(day), when, when); err != nil {
			t.Fatal(err)
		}
	}
	f.want(f.run("--no-deploy"), ExitOK, "[ng-deploy] backup ok: _backups/demo-pre-")
	left, _ := filepath.Glob(dir + "/demo-pre-*.tar.gz")
	if len(left) != 3 || f.exists(old("1")) || f.exists(old("2")) || !f.exists(old("3")) || !f.exists(old("4")) {
		t.Errorf("backups left: %v", left)
	}
	for _, name := range left {
		if info, err := os.Stat(name); name != old("3") && name != old("4") && (err != nil || info.Size() == 0) {
			t.Errorf("the new backup %s is empty", name)
		}
	}
}
