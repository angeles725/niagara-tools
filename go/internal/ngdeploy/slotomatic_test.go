package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

const (
	rtSlotomatic = "gradlew" + gradleProps + " <:demo-rt:slotomatic>\n"
	uxSlotomatic = "gradlew" + gradleProps + " <:demo-rt:slotomatic> <:demo-ux:slotomatic>\n"
)

func TestWithSlotomaticRunsSlotomaticBeforeTheBuild(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C", "--with-slotomatic"), ExitOK,
		"[ng-deploy] slotomatic: running :demo-rt:slotomatic\n[ng-deploy] slotomatic ok\n[ng-deploy] build: mode C")
	if want := gitDiffCalls + rtSlotomatic + "gradlew" + gradleProps + " <:demo-rt:clean> <:demo-rt:jar>\n"; f.calls() != want {
		t.Errorf("calls:\n%s", f.calls())
	}
}

func TestUxSlotomaticRunsOnlyInModeAWithAnnotatedUxSources(t *testing.T) {
	for _, tc := range []struct{ name, file, text, mode, want string }{
		{"annotated", "com/x/BFoo.java", "package x;\n@NiagaraType\nclass BFoo {}\n", "A", uxSlotomatic},
		{"each annotation counts", "BFoo.java", "  @NiagaraSingleton", "A", uxSlotomatic},
		{"plain java", "com/x/Plain.java", "class Plain {}\n", "A", rtSlotomatic},
		{"not a java file", "com/x/notes.txt", "@NiagaraType\n", "A", rtSlotomatic},
		{"no ux sources", "", "", "A", rtSlotomatic},
		{"mode C", "com/x/BFoo.java", "@NiagaraAction\n", "C", rtSlotomatic},
	} {
		f := newFixture(t)
		if tc.file != "" {
			f.write(f.src+"/demo/demo-ux/src/"+tc.file, tc.text)
		}
		f.want(f.run("--no-backup", "--no-deploy", "--mode", tc.mode, "--with-slotomatic"), ExitOK)
		if !strings.Contains(f.calls(), tc.want) {
			t.Errorf("%s: calls:\n%s", tc.name, f.calls())
		}
		line := "[ng-deploy] slotomatic: running :demo-rt:slotomatic\n"
		if tc.want == uxSlotomatic {
			line = "[ng-deploy] slotomatic: running :demo-rt:slotomatic :demo-ux:slotomatic (-ux has @Niagara annotations)\n"
		}
		if !strings.HasPrefix(f.out, line) {
			t.Errorf("%s: stdout:\n%s", tc.name, f.out)
		}
	}
}

// Declared difference: bash read unreadable -ux sources as "no annotation" and skipped the -ux slotomatic.
func TestUxSourcesThatCannotBeReadAreRefused(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	f := newFixture(t)
	f.write(f.src+"/demo/demo-ux/src/com/x/BFoo.java", "@NiagaraType\n")
	f.chmod(f.src+"/demo/demo-ux/src/com", 0o000)
	code := f.run("--no-backup", "--with-slotomatic")
	want := warnNoBackup + "[ng-deploy] ERROR cannot scan the -ux sources for @Niagara annotations (" + f.src + "/demo/demo-ux/src/com: Permission denied); deploy aborted\n"
	if code != ExitSlotomatic || f.err != want || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

func TestModeBIgnoresWithSlotomaticWithAWarning(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "B", "--with-slotomatic"), ExitOK)
	want := warnNoBackup + "[ng-deploy] WARN --with-slotomatic ignored for mode B (ux-only): ng-deploy runs slotomatic for -rt only.\n" +
		"[ng-deploy] WARN to regenerate -ux slots, deploy with --mode A --with-slotomatic, or build via toolbelt/build.sh.\n"
	if f.err != want || strings.Contains(f.calls(), "slotomatic") {
		t.Errorf("stderr %q, calls %q", f.err, f.calls())
	}
}

func TestAFailedSlotomaticExits15WithoutBuilding(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_SLOT_EXIT", "7")
	code := f.run("--no-backup", "--with-slotomatic")
	if code != ExitSlotomatic || f.err != warnNoBackup+"[ng-deploy] ERROR slotomatic failed (gradlew exited non-zero); deploy aborted\n" || f.calls() != gitDiffCalls+uxSlotomatic[:len(rtSlotomatic)-1]+"\n" {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

func TestAFileSlotomaticSkippedExits15NamingIt(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_SLOT_OUT", "WARNING: Could not process /x/src/BDemoStatus.java: Found multiple metadata blocks")
	code := f.run("--no-backup", "--with-slotomatic")
	want := warnNoBackup + "FAIL slotomatic-skip /x/src/BDemoStatus.java (Found multiple metadata blocks)\n" +
		"[ng-deploy] ERROR Slot-o-Matic skipped the file(s) above (METHODOLOGY K15: the generated region needs exactly one stamp — delete it and re-run slotomatic); deploy aborted\n"
	if code != ExitSlotomatic || f.err != want || f.calls() != gitDiffCalls+rtSlotomatic {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
	// both gradle streams are shown on stdout, as `2>&1 | tee` showed them
	if f.out != "[ng-deploy] slotomatic: running :demo-rt:slotomatic\nWARNING: Could not process /x/src/BDemoStatus.java: Found multiple metadata blocks\ngradle stderr line\n" {
		t.Errorf("stdout:\n%s", f.out)
	}
}

func TestASkippedTestSourceIsNotAFailure(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_SLOT_OUT", "WARNING: Could not process /x/srcTest/BFooTest.java: nope")
	f.want(f.run("--no-backup", "--no-deploy", "--with-slotomatic"), ExitOK, "[ng-deploy] slotomatic ok\n")
}
