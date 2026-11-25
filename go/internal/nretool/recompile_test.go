package nretool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const wantRecompileClass = "program:com.tridium.program.ui.RecompileTool"

func bogFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.bog")
	write(t, p, "", 0o644)
	return p
}

const processed = "echo '--- Processed x!'"

func TestRecompilePassesTheTargetAndTheAllowedFlagsAndOutput(t *testing.T) {
	home, argsFile := stubHome(t, "nre.exe", processed)
	bog := bogFile(t)
	code, out, errs := run("recompile", home, bog, "-alias:lab", "-tsaUrl:http://tsa.example")
	want := filepath.Join(home, "bin") + "\n" + wantRecompileClass + "\nW:" + bog + "\n-alias:lab\n-tsaUrl:http://tsa.example\n"
	got, _ := readFile(argsFile)
	if code != 0 || got != want || out != "--- Processed x!\n" || errs != "" {
		t.Errorf("exit %d, nre ran as %q (want %q), out %q, err %q", code, got, want, out, errs)
	}
}

func TestRecompileHandsStdinToNreSoItPromptsForThePassword(t *testing.T) {
	home, _ := stubHome(t, "nre.exe", processed+"\ncat > \"$PWD/../stdin.txt\"")
	old := stdin
	t.Cleanup(func() { stdin = old })
	stdin = strings.NewReader("typed-secret\n")
	if code, _, errs := run("recompile", home, bogFile(t)); code != 0 {
		t.Fatalf("exit %d, err %q", code, errs)
	}
	if got, _ := readFile(filepath.Join(home, "stdin.txt")); got != "typed-secret\n" {
		t.Errorf("nre read %q from stdin", got)
	}
}

func TestRecompileRefusesAPasswordInArgv(t *testing.T) {
	for _, flag := range []string{"-password:hunter2", "-PASSWORD:hunter2", "-password"} {
		home, argsFile := stubHome(t, "nre.exe", processed)
		code, out, errs := run("recompile", home, bogFile(t), "-alias:lab", flag)
		if code != 3 || out != "" || exists(argsFile) || strings.Contains(errs, "hunter2") ||
			!strings.Contains(errs, "nre-tool: recompile never passes the password in argv; nre prompts for it") {
			t.Errorf("%s: exit %d, out %q, err %q, nre ran %v", flag, code, out, errs, exists(argsFile))
		}
	}
}

func TestRecompileRefusesAnUnknownFlagAndAMissingTarget(t *testing.T) {
	home, argsFile := stubHome(t, "nre.exe", processed)
	missing := filepath.Join(t.TempDir(), "none.bog")
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{bogFile(t), "-force"}, "nre-tool: recompile: unknown flag -force (allowed: -alias:, -tsaUrl:)"},
		{[]string{missing}, "nre-tool: recompile: " + missing + " does not exist"},
		{nil, "usage: nre-tool recompile <niagara_home> <dir|bog> [-alias:<a>] [-tsaUrl:<u>]"},
	} {
		code, _, errs := run(append([]string{"recompile", home}, c.args...)...)
		if code != 3 || exists(argsFile) || !strings.Contains(errs, c.want) {
			t.Errorf("%q: exit %d, err %q, nre ran %v", c.args, code, errs, exists(argsFile))
		}
	}
}

func TestRecompileExitsWithNreExitCode(t *testing.T) {
	home, _ := stubHome(t, "nre.exe", "exit 4")
	if code, _, _ := run("recompile", home, bogFile(t)); code != 4 {
		t.Errorf("exit %d, want 4", code)
	}
}

func TestRecompileFailuresThatNreExitsZeroOnAreStillFailures(t *testing.T) {
	for body, want := range map[string]string{
		processed + "\necho 'ERROR Recompiling p'": "nre-tool: RecompileTool reported ERROR",
		"echo 'ERROR: Cannot process x'":           "nre-tool: RecompileTool reported ERROR",
		"echo 'File not found: x'":                 "nre-tool: RecompileTool processed no bog file",
		"echo 'SEVERE: bad signing key'":           "nre-tool: RecompileTool processed no bog file",
	} {
		home, _ := stubHome(t, "nre.exe", body)
		code, _, errs := run("recompile", home, bogFile(t))
		if code != 1 || !strings.Contains(errs, want) {
			t.Errorf("%q: exit %d, err %q, want %q", body, code, errs, want)
		}
	}
}

func TestRecompileNamesNreAndTheErrorWhenItCannotBeStarted(t *testing.T) {
	home, _ := stubHome(t, "nre.exe", "")
	nre := filepath.Join(home, "bin", "nre.exe")
	if err := os.Chmod(nre, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run("recompile", home, bogFile(t))
	if code != exitEnv || out != "" || !strings.Contains(errs, "nre-tool: cannot start "+nre+": ") ||
		!strings.Contains(errs, "permission denied") {
		t.Errorf("exit %d (want %d), out %q, err %q", code, exitEnv, out, errs)
	}
}
