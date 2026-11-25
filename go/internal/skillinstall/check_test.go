package skillinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `install-skill --check`: the read-only drift check of every skill of the SKILLS table in every harness.

// installAll installs both fixture skills into the harness (every harness's directory made first for "all").
func installAll(t *testing.T, f *fixture, harness string) {
	t.Helper()
	for _, s := range []string{"build-n4-module", "mcp-n4"} {
		if code, out, errText := f.install("--skill", s, "--harness", harness); code != 0 {
			t.Fatalf("install %s: exit %d\n%s\n%s", s, code, out, errText)
		}
	}
}

// cleanRows is the rows of a clean claude-only home: PASS for the copy, SKIP for the absent harnesses.
func cleanRows(f *fixture, skills ...string) string {
	var b strings.Builder
	for _, s := range skills {
		b.WriteString("PASS  skill-drift  " + s + "[claude]  installed copy matches " + s + "-kit/skill/SKILL.md\n")
		b.WriteString("SKIP  skill-drift  " + s + "[pi]  absent: " + f.home + "/.pi/agent not found\n")
		b.WriteString("SKIP  skill-drift  " + s + "[gentle-shell]  absent: " + f.home + "/.gentle-shell/agent not found\n")
	}
	return b.String()
}

// TestCheckCleanHomeIsExit0: every skill of the table, every harness, in table then harness order.
func TestCheckCleanHomeIsExit0(t *testing.T) {
	f := newFixture(t)
	installAll(t, f, "claude")
	code, out, errText := f.install("--check")
	expect(t, "clean", code, out, errText, 0, cleanRows(f, "build-n4-module", "mcp-n4"), "")
}

// TestCheckReportsDriftAndNeverWrites: a diverged copy is a FAIL row and exit 1; the other skill is still
// checked and the diverged copy is left as it was.
func TestCheckReportsDriftAndNeverWrites(t *testing.T) {
	f := newFixture(t)
	installAll(t, f, "claude")
	target := write(t, f.installed("build-n4-module"), "stale\n")
	code, out, errText := f.install("--check")
	want := "FAIL  skill-drift  build-n4-module[claude]  installed copy " + target +
		" differs from build-n4-module-kit/skill/SKILL.md — run scripts/install-skill.sh --force --skill build-n4-module\n" +
		strings.SplitN(cleanRows(f, "build-n4-module"), "\n", 2)[1] + cleanRows(f, "mcp-n4")
	expect(t, "diverged", code, out, errText, 1, want, "")
	if got := read(t, target); got != "stale\n" {
		t.Errorf("--check wrote the installed copy: %q", got)
	}
}

// TestCheckNotInstalledWritesNothing: every copy missing is FAIL rows and exit 1, and the home stays absent.
func TestCheckNotInstalledWritesNothing(t *testing.T) {
	f := newFixture(t)
	code, out, errText := f.install("--check")
	if code != 1 || errText != "" || strings.Count(out, "FAIL  skill-drift  ") != 2 ||
		!strings.Contains(out, "build-n4-module[claude]  not installed at "+f.installed("build-n4-module")+
			" — run scripts/install-skill.sh --skill build-n4-module\n") {
		t.Errorf("exit %d\nstdout %q\nstderr %q", code, out, errText)
	}
	absent(t, f.home)
}

// TestCheckReadsEveryHarness: a pi agent directory adds the launcher row and the prompt row, and a diverged
// prompt is a FAIL row with the --harness remedy.
func TestCheckReadsEveryHarness(t *testing.T) {
	f := newFixture(t)
	piDir := filepath.Join(f.home, ".pi", "agent")
	if err := os.MkdirAll(piDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installAll(t, f, "all")
	code, out, errText := f.install("--check")
	if code != 0 || errText != "" || !strings.Contains(out, "PASS  skill-drift  mcp-n4[pi]  prompt "+piDir+"/prompts/mcp-n4.md matches its template\n") {
		t.Errorf("clean pi: exit %d\nstdout %q\nstderr %q", code, out, errText)
	}
	write(t, filepath.Join(piDir, "prompts", "mcp-n4.md"), "stale\n")
	code, out, errText = f.install("--check")
	if code != 1 || errText != "" || !strings.Contains(out, "FAIL  skill-drift  mcp-n4[pi]  prompt "+piDir+
		"/prompts/mcp-n4.md differs from its template — run scripts/install-skill.sh --force --skill mcp-n4 --harness pi\n") {
		t.Errorf("diverged prompt: exit %d\nstdout %q\nstderr %q", code, out, errText)
	}
}

// TestCheckFollowsTheTable: a skill added to the SKILLS line is checked too.
func TestCheckFollowsTheTable(t *testing.T) {
	f := newFixture(t)
	write(t, f.installer, "#!/usr/bin/env bash\nSKILLS=\"extra:extra-kit/skill/SKILL.md\"\n")
	write(t, filepath.Join(f.root, "extra-kit", "skill", "SKILL.md"), "stub extra skill\n")
	write(t, f.installed("extra"), "stub extra skill\n")
	code, out, errText := f.install("--check")
	expect(t, "extra", code, out, errText, 0, cleanRows(f, "extra"), "")
}

// TestCheckUsageErrors: --check takes only --home; the install options with it are exit 2, nothing written.
func TestCheckUsageErrors(t *testing.T) {
	f := newFixture(t)
	for _, opt := range [][]string{{"--force"}, {"--dry-run"}, {"--skill", "mcp-n4"}, {"--harness", "pi"}} {
		code, out, errText := f.install(append([]string{"--check"}, opt...)...)
		expect(t, opt[0], code, out, errText, 2, "", "install-skill: --check cannot be combined with "+opt[0]+"\n"+wantUsage)
	}
	absent(t, f.home)
}

// TestCheckEnvironmentErrors: a missing tracked launcher and no home are exit 3 with no row printed.
func TestCheckEnvironmentErrors(t *testing.T) {
	f := newFixture(t)
	installAll(t, f, "claude")
	if err := os.Remove(f.tracked("mcp-n4")); err != nil {
		t.Fatal(err)
	}
	code, out, errText := f.install("--check")
	expect(t, "tracked missing", code, out, errText, 3, "", "install-skill: tracked source not found: "+f.tracked("mcp-n4")+"\n")

	f.env["HOME"] = ""
	code, out, errText = f.installNoHome("--check")
	expect(t, "no home", code, out, errText, 3, "", "install-skill: no home directory (pass --home)\n")
}
