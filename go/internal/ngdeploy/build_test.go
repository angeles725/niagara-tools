package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

// Kept from the bash tool: the task list of the build line is joined with newlines (its IFS).
func TestBuildRunsTheGradleTasksOfTheMode(t *testing.T) {
	for mode, tasks := range map[string][]string{
		"A": {":demo-rt:clean", ":demo-ux:clean", ":demo-rt:jar", ":demo-ux:jar"},
		"B": {":demo-ux:clean", ":demo-ux:jar"},
		"C": {":demo-rt:clean", ":demo-rt:jar"},
	} {
		f := newFixture(t)
		t.Setenv("SLOTOMATIC_DETECTION", "off")
		f.want(f.run("--no-backup", "--no-deploy", "--mode", mode), ExitOK)
		wantOut := "[ng-deploy] build: mode " + mode + " — " + strings.Join(tasks, "\n") + "\n[ng-deploy] build ok\n[ng-deploy] no-deploy: jars in build/libs/\n"
		wantCall := "gradlew" + gradleProps + " <" + strings.Join(tasks, "> <") + ">\n"
		if f.out != wantOut || f.calls() != wantCall {
			t.Errorf("mode %s: stdout %q, calls %q", mode, f.out, f.calls())
		}
	}
}

func TestTheDefaultModeIsA(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] build: mode A — :demo-rt:clean\n:demo-ux:clean\n:demo-rt:jar\n:demo-ux:jar\n")
}

func TestAFailedBuildExits30(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_GRADLEW_EXIT", "1")
	code := f.run("--no-backup")
	if code != ExitBuild || f.err != warnNoBackup+"[ng-deploy] ERROR build failed (gradlew exited non-zero)\n" || strings.Contains(f.out, "build ok") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
}

// Declared difference: the line saying why gradlew could not be started was bash's own "<script>: line N: ...".
func TestAGradlewThatCannotBeStartedExits30(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(f.src + "/gradlew"); err != nil {
		t.Fatal(err)
	}
	code := f.run("--no-backup")
	want := warnNoBackup + "ng-deploy.sh: " + f.src + "/gradlew: No such file or directory\n[ng-deploy] ERROR build failed (gradlew exited non-zero)\n"
	if code != ExitBuild || f.err != want {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}

func TestNoDeployStopsAfterTheBuild(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] no-deploy: jars in build/libs/\n")
	if f.exists(f.deployedJar("rt")) || f.exists(f.deployedJar("ux")) || strings.Contains(f.calls(), "unzip") || f.exists(f.work+"/.last-deploy-sha") {
		t.Errorf("--no-deploy went past the build: %s", f.calls())
	}
}
