package ngdeploy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// stamp replaces the timestamp of a backup name, so a test compares whole lines.
func stamp(s string) string {
	return regexp.MustCompile(`[0-9]{8}-[0-9]{6}`).ReplaceAllString(s, "TS")
}

func TestNoBackupSkipsTheBackupWithAWarning(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
	if f.err != warnNoBackup || strings.Contains(f.calls(), "tar") || f.exists(f.work+"/_backups") {
		t.Errorf("stderr %q, calls %q", f.err, f.calls())
	}
}

func TestBackupArchivesOnlyTheJarsOfTheModule(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	f.write(f.modules+"/demo-wb.jar", "")
	f.write(f.modules+"/other-rt.jar", "")
	f.want(f.run("--no-deploy"), ExitOK)
	if !strings.HasPrefix(stamp(f.out), "[ng-deploy] backup: _backups/demo-pre-TS.tar.gz\n[ng-deploy] backup ok: _backups/demo-pre-TS.tar.gz\n[ng-deploy] build:") {
		t.Errorf("stdout:\n%s", f.out)
	}
	if want := "tar <-czf> <_backups/demo-pre-TS.tar.gz> <-C> <" + f.modules + "> <demo-rt.jar> <demo-ux.jar> <demo-wb.jar>\n"; !strings.HasPrefix(stamp(f.calls()), want) {
		t.Errorf("calls:\n%s", f.calls())
	}
}

// An archive is named only when one is written: a module with no jar in the station gets the one line that
// says so, no archive name, and no empty _backups directory.
func TestBackupOfAModuleNeverDeployedArchivesNothing(t *testing.T) {
	f := newFixture(t)
	f.write(f.modules+"/other-rt.jar", "")
	f.want(f.run("--no-deploy"), ExitOK)
	want := "[ng-deploy] backup: no existing demo-{rt,ux,wb}.jar in " + f.modules + "; nothing to archive\n[ng-deploy] build:"
	if !strings.HasPrefix(f.out, want) || strings.Contains(f.calls(), "tar") || f.exists(f.work+"/_backups") {
		t.Errorf("stdout:\n%scalls:\n%s", f.out, f.calls())
	}
}

func TestFullBackupArchivesTheWholeModulesDirectory(t *testing.T) {
	want := func(f *fixture) {
		t.Helper()
		if want := "tar <-czf> <_backups/demo-pre-TS.tar.gz> <-C> <" + f.dir + "> <modules>\n"; !strings.HasPrefix(stamp(f.calls()), want) {
			t.Errorf("calls:\n%s", f.calls())
		}
	}
	f := newFixture(t) // an empty modules directory is archived too
	f.want(f.run("--no-deploy", "--full-backup"), ExitOK, "[ng-deploy] backup ok: ")
	want(f)
	f = newFixture(t)
	t.Setenv("FULL_BACKUP", "1")
	f.want(f.run("--no-deploy"), ExitOK)
	want(f)
	f = newFixture(t)
	f.set("STATION_MODULES_DIR", f.modules+"/") // dirname and basename as bash computes them
	f.set("FULL_BACKUP", "1")
	f.want(f.run("--no-deploy"), ExitOK)
	want(f)
}

func TestAFailedBackupExits20BeforeTheBuild(t *testing.T) {
	for _, flag := range []string{"--no-deploy", "--full-backup"} {
		f := newFixture(t)
		f.deployed()
		t.Setenv("FAKE_TAR_EXIT", "2")
		code := f.run(flag)
		if code != ExitBackup || stamp(f.err) != "[ng-deploy] ERROR backup failed: _backups/demo-pre-TS.tar.gz\n" || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%s: exit %d, stderr %q, calls %q", flag, code, f.err, f.calls())
		}
	}
}

func TestABackupDirectoryThatCannotBeMadeExits1(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	f.write(f.work+"/_backups", "a file in the way")
	code := f.run("--no-deploy")
	if code != ExitFault || f.out != "" || !strings.Contains(f.err, "mkdir: ") || f.calls() != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, calls %q", code, f.out, f.err, f.calls())
	}
}

// Declared difference: bash read "cannot check" as "no such jar" and went on without a backup.
func TestBackupRefusesADeployedJarItCannotCheck(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root checks everything")
	}
	f := newFixture(t)
	f.deployed()
	f.chmod(f.modules, 0o000)
	code := f.run("--no-deploy")
	if code != ExitBackup || f.err != "[ng-deploy] ERROR backup failed: cannot check "+f.modules+"/demo-rt.jar: Permission denied\n" || f.calls() != "" {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

// oldBackups plants five dated backups of the module and one of another module; it returns their directory.
func oldBackups(f *fixture) string {
	dir := f.work + "/_backups"
	for day := 1; day <= 5; day++ {
		name := dir + "/demo-pre-2026010" + string(rune('0'+day)) + "-000000.tar.gz"
		f.write(name, "")
		when := time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(name, when, when); err != nil {
			f.t.Fatal(err)
		}
	}
	f.write(dir+"/other-pre-20260101-000000.tar.gz", "")
	return dir
}

func TestBackupKeepsTheNewestNOfTheModule(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		env  string // KEEP_N of the environment
		file string // KEEP_N of the env file
		want []string
	}{
		{"default 3", nil, "", "", []string{"20260104", "20260105", "TS"}},
		{"--keep 5", []string{"--keep", "5"}, "", "", []string{"20260102", "20260103", "20260104", "20260105", "TS"}},
		{"--keep 1", []string{"--keep", "1"}, "", "", []string{"TS"}},
		{"KEEP_N=2", nil, "2", "", []string{"20260105", "TS"}},
		{"the flag wins over the environment", []string{"--keep", "4"}, "2", "", []string{"20260103", "20260104", "20260105", "TS"}},
		{"the env file wins over the environment", nil, "2", "4", []string{"20260103", "20260104", "20260105", "TS"}},
		{"the flag wins over the env file", []string{"--keep", "1"}, "", "4", []string{"TS"}},
	} {
		f := newFixture(t)
		f.deployed()
		dir := oldBackups(f)
		if tc.env != "" {
			t.Setenv("KEEP_N", tc.env)
		}
		if tc.file != "" {
			f.set("KEEP_N", tc.file)
		}
		f.want(f.run(append([]string{"--no-deploy"}, tc.args...)...), ExitOK)
		kept, _ := filepath.Glob(dir + "/demo-pre-*")
		var got []string
		for _, k := range kept {
			name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(k), "demo-pre-"), ".tar.gz")
			if !strings.HasSuffix(name, "-000000") {
				name = "TS"
			}
			got = append(got, strings.TrimSuffix(name, "-000000"))
		}
		if strings.Join(got, " ") != strings.Join(tc.want, " ") || !f.exists(dir+"/other-pre-20260101-000000.tar.gz") {
			t.Errorf("%s: kept %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Declared difference: under bash --keep 0 and --keep -1 deleted every backup, the new one included, and
// a word purged nothing; all three still ended "backup ok" with exit 0.
func TestKeepMustBeAPositiveWholeNumber(t *testing.T) {
	for _, keep := range []string{"0", "-1", "abc", "", "03", "1.5"} {
		f := newFixture(t)
		f.deployed()
		dir := oldBackups(f)
		code := f.run("--no-deploy", "--keep", keep)
		kept, _ := filepath.Glob(dir + "/*")
		if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR KEEP_N (--keep) must be a whole number of backups, 1 or more\n" || len(kept) != 6 || f.calls() != "" {
			t.Errorf("--keep %q: exit %d, stdout %q, stderr %q, %d backups left", keep, code, f.out, f.err, len(kept))
		}
	}
	f := newFixture(t)
	t.Setenv("KEEP_N", "") // an empty variable is the default, as ${KEEP_N:-3} is
	f.want(f.run("--no-deploy"), ExitOK)
}

// Two deploys in the same second used to give their backups the same name, and the second tar wrote over
// the first backup: the only copy of the jars the first deploy replaced.
func TestBackupNamesAreUniqueWithinOneSecond(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.Local)
	clock = func() time.Time { return at }
	t.Cleanup(func() { clock = time.Now })
	for i, name := range []string{"_backups/demo-pre-20260304-050607.tar.gz", "_backups/demo-pre-20260304-050607-2.tar.gz", "_backups/demo-pre-20260304-050607-3.tar.gz"} {
		f.reset()
		f.want(f.run("--no-deploy", "--keep", "5"), ExitOK, "[ng-deploy] backup: "+name+"\n[ng-deploy] backup ok: "+name+"\n")
		if !strings.HasPrefix(f.calls(), "tar <-czf> <"+name+"> <-C> <"+f.modules+"> <demo-rt.jar> <demo-ux.jar>\n") || f.err != "" {
			t.Errorf("backup %d: stderr %q, calls:\n%s", i+1, f.err, f.calls())
		}
		f.write(f.work+"/"+name, "backup "+name) // what a real tar would have left there
	}
	for _, name := range []string{"demo-pre-20260304-050607.tar.gz", "demo-pre-20260304-050607-2.tar.gz"} {
		if got := f.read(f.work + "/_backups/" + name); got != "backup _backups/"+name {
			t.Errorf("%s was written over: %q", name, got)
		}
	}
	at = at.Add(time.Second) // the next second starts again with the plain name
	f.want(f.run("--no-deploy", "--keep", "5"), ExitOK, "[ng-deploy] backup ok: _backups/demo-pre-20260304-050608.tar.gz\n")
}

// The numbered backups of one second are purged like any other, oldest first.
func TestBackupsOfTheSameSecondArePurgedOldestFirst(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.Local)
	clock = func() time.Time { return at }
	t.Cleanup(func() { clock = time.Now })
	for i, name := range []string{"demo-pre-20260304-050607.tar.gz", "demo-pre-20260304-050607-2.tar.gz"} {
		f.write(f.work+"/_backups/"+name, "")
		when := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(f.work+"/_backups/"+name, when, when); err != nil {
			t.Fatal(err)
		}
	}
	f.want(f.run("--no-deploy", "--keep", "2"), ExitOK, "[ng-deploy] backup ok: _backups/demo-pre-20260304-050607-3.tar.gz\n")
	kept, _ := filepath.Glob(f.work + "/_backups/*")
	if len(kept) != 2 || f.exists(f.work+"/_backups/demo-pre-20260304-050607.tar.gz") {
		t.Errorf("kept %v", kept)
	}
}
