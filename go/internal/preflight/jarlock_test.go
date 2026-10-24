package preflight

import (
	"os"
	"strings"
	"testing"
)

// jarRow runs preflight over f with PATH=<d>/bin and mounts (written to a file when non-empty) and
// returns its jar-lock row.
func jarRow(t *testing.T, f *fakeHost, d, mounts string) string {
	t.Helper()
	env := map[string]string{"PATH": d + "/bin"}
	if mounts != "" {
		mk(t, d, "mounts="+mounts)
		env["N4_FSTYPE_MOUNTS_FILE"] = d + "/mounts"
	}
	_, out, _ := pf(f.host(), env, "--jvm-dir", d, d+"/nh", d+"/gr")
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "  jar-lock  ") {
			return ln
		}
	}
	return ""
}

func TestJarLockOneLsofPassOverTheJarSet(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "gr/", "nh/modules/A.jar", "nh/modules/B.jar", "nh/modules/.h.jar", "nh/modules/D.jar/", "bin/lsof")
	if err := os.Chmod(d+"/bin/lsof", 0o755); err != nil {
		t.Fatal(err)
	}
	m := d + "/nh/modules/"
	for _, c := range []struct{ listing, want string }{
		{"p1\nn" + m + "B.jar\nn" + m + ".h.jar\nn" + m + "D.jar\nn" + m + "A.jar\n\n", "WARN  jar-lock      jars held by a process (station running?): A.jar B.jar"},
		{"p1\nn" + m + "A.ja\nn" + m + "A.jar.bak\n" + m + "B.jar", "PASS  jar-lock      no locked jars under " + d + "/nh/modules/"},
		{"\n\n", "SKIP  jar-lock      lsof returned no open-file listing (failed or no permission) — jar locks not checked"},
	} {
		f := &fakeHost{out: map[string]string{"/bin/lsof": c.listing}}
		if got := jarRow(t, f, d, "ext4 / ext4 rw 0 0\n"); got != c.want || len(f.calls) != 1 || f.calls[0] != "lsof -Fn" {
			t.Errorf("listing %q: row %q calls %q, want %q after one lsof -Fn", c.listing, got, f.calls, c.want)
		}
	}
}

func TestJarLockSkipsWithoutAnAnswer(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "gr/", "nh/modules/A.jar", "bin/df")
	if err := os.Chmod(d+"/bin/df", 0o755); err != nil {
		t.Fatal(err)
	}
	skip9p := "SKIP  jar-lock      niagara_home is on a 9p/drvfs mount (WSL) — lsof cannot see a Windows-side lock here"
	for _, c := range []struct{ mounts, df, want string }{
		{"9p / 9p rw 0 0\next4 " + d + " ext4\n9p " + d + "/n 9p rw 0 0\n", "", skip9p}, // longest prefix, as a string
		{"ext4 / ext4\ndrvfs " + d + " drvfs rw 0 0\n", "", skip9p},
		{"", "Filesystem Type 1K-blocks\nC:\\ 9p 1 1 1 1% " + d + "\n", skip9p}, // no mounts file: df -T, found in PATH
		{"ext4 / ext4 rw 0 0\n", "", "SKIP  jar-lock      lsof not in PATH — install lsof to enable jar-lock detection"},
	} {
		f := &fakeHost{out: map[string]string{"df": c.df}}
		if got := jarRow(t, f, d, c.mounts); got != c.want {
			t.Errorf("mounts %q df %q: row %q, want %q", c.mounts, c.df, got, c.want)
		}
		for _, call := range f.calls {
			if strings.HasPrefix(call, "lsof") || (call != "df -T -- "+d+"/nh" && strings.HasPrefix(call, "df")) {
				t.Errorf("ran %q", call)
			}
		}
	}
}

func TestLookPathFallsBackToANonExecutableFile(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "a/lsof/", "b/lsof", "c/lsof")
	if err := os.Chmod(d+"/c/lsof", 0o755); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{d + "/a:" + d + "/b:" + d + "/c": d + "/c/lsof", d + "/a:" + d + "/b": d + "/b/lsof", d + "/a": ""} {
		if got, ok := lookPath("lsof", path); got != want || ok != (want != "") {
			t.Errorf("lookPath(%q) = %q %v, want %q", path, got, ok, want)
		}
	}
}

// TestCommandsSeeOnlyTheDeclaredEnvironment: the System host runs lsof in Host.Env, not in the test
// process's environment: the fake lists the jar only when JAR is declared and HOME is not.
func TestCommandsSeeOnlyTheDeclaredEnvironment(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "gr/", "nh/modules/A.jar", "mounts=ext4 / ext4\n", "bin/lsof=#!/bin/sh\n[ -z \"$HOME\" ] && printf 'n%s\\n' \"$JAR\"\n")
	if err := os.Chmod(d+"/bin/lsof", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", d)
	env := []string{"PATH=" + d + "/bin", "JAR=" + d + "/nh/modules/A.jar"}
	h := Host{Output: System.Output, Env: env}
	vars := map[string]string{"PATH": d + "/bin", "N4_FSTYPE_MOUNTS_FILE": d + "/mounts"}
	want := "WARN  jar-lock      jars held by a process (station running?): A.jar\n"
	if _, out, _ := pf(h, vars, "--jvm-dir", d, d+"/nh", d+"/gr"); !strings.Contains(out, want) {
		t.Errorf("rows\n%s\nwant %q: the fake saw the test process's environment", out, want)
	}
}
