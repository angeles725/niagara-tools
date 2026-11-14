package slotomaticskip

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// logs are gradle logs and the rows Scan reports; TestMatchesTheShellLib replays them on the bash lib.
var logs = []struct {
	name, log string
	want      []string
}{
	{"no skipped file", "> Task :Foo-rt:slotomatic\nBLimiter.java:12: error: cannot find symbol\nBUILD SUCCESSFUL in 3s\n", nil},
	{"one Could not process line", "WARNING: Could not process /x/src/com/a/BHmiStatus.java: Found multiple metadata blocks\n",
		[]string{"FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java (Found multiple metadata blocks)"}},
	{"the block only", "The following files failed to compile: \n  /x/src/com/a/BHmiStatus.java: Found multiple metadata blocks\n> Task :Foo-rt:jar\n",
		[]string{"FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java (Found multiple metadata blocks)"}},
	{"a block entry with no message", "WARNING: The following files failed to compile:\nWARNING:   /x/src/com/a/BHmiStatus.java\n",
		[]string{"FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java (listed under: The following files failed to compile)"}},
	{"both forms name one file once", "Could not process /x/B.java: m\nThe following files failed to compile: \n  /x/B.java: m\nBUILD SUCCESSFUL\n",
		[]string{"FAIL slotomatic-skip /x/B.java (m)"}},
	{"two files in log order, CRLF and a drive letter", "Could not process /x/A.java: m1\nCould not process C:\\m\\src\\B.java: m2\r\n",
		[]string{"FAIL slotomatic-skip /x/A.java (m1)", `FAIL slotomatic-skip C:\m\src\B.java (m2)`}},
	{"a javac line inside the block is not an entry", "The following files failed to compile: \n  /x/A.java: m1\n  /x/C.java:12: error: boom\n",
		[]string{"FAIL slotomatic-skip /x/A.java (m1)"}},
	{"srcTest files are ignored", "WARNING: Could not process /x/srcTest/com/a/BLimiterTest.java: m\nThe following files failed to compile:\n  C:\\x\\srcTest\\com\\a\\BOtherTest.java\n", nil},
	{"a src skip beside a srcTest one", "Could not process /x/srcTest/T.java: m\nCould not process /x/src/S.java: m\n",
		[]string{"FAIL slotomatic-skip /x/src/S.java (m)"}},
	{"a bare Could not process path", "Could not process /x/A.java\n", []string{"FAIL slotomatic-skip /x/A.java (Could not process)"}},
	{"an empty reason", "Could not process /x/A.java: \n", []string{"FAIL slotomatic-skip /x/A.java (skipped by Slot-o-Matic)"}},
	{"Could not process something else", "Could not process /x/A.kt: m\n", nil},
	{"the block ends at a task line", "The following files failed to compile:\n> Task :x\n  /x/A.java: m\n", nil},
	{"the block ends at BUILD FAILED", "The following files failed to compile:\nBUILD FAILED in 1s\n  /x/A.java: m\n", nil},
	{"a one-space indent is not an entry", "The following files failed to compile:\n /x/A.java: m\n\t\t/x/B.java: m\n",
		[]string{"FAIL slotomatic-skip /x/B.java (m)"}},
	{"trailing blanks and no final newline", "The following files failed to compile:\n  /x/A.java  \t", []string{"FAIL slotomatic-skip /x/A.java (listed under: The following files failed to compile)"}},
	{"empty log", "", nil},
}

func TestScan(t *testing.T) {
	t.Parallel()
	for _, c := range logs {
		if got := strings.Join(Scan(c.log), "\n"); got != strings.Join(c.want, "\n") {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, strings.Join(c.want, "\n"))
		}
	}
}

func TestScanFile(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	log := filepath.Join(d, "gradle.log")
	if err := os.WriteFile(log, []byte("Could not process /x/A.java: m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rows, status := ScanFile(log); status != Skipped || len(rows) != 1 {
		t.Errorf("a log with a skip: rows %q status %d", rows, status)
	}
	if err := os.WriteFile(log, []byte("BUILD SUCCESSFUL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rows, status := ScanFile(log); status != Clean || rows != nil {
		t.Errorf("a clean log: rows %q status %d", rows, status)
	}
	for _, bad := range []string{filepath.Join(d, "missing.log"), d} {
		rows, status := ScanFile(bad)
		if want := "FAIL slotomatic-skip " + bad + " (gradle log unreadable)"; status != Unreadable || len(rows) != 1 || rows[0] != want {
			t.Errorf("ScanFile(%s) = %q status %d, want %q status %d", bad, rows, status, want, Unreadable)
		}
	}
}

// TestMatchesTheShellLib pins this package to toolbelt/lib/slotomatic-skip.sh while that lib exists (no
// script sources it any more): slotomatic_skip_scan must print the same rows with the same status for every
// log above.
func TestMatchesTheShellLib(t *testing.T) {
	t.Parallel()
	lib, err := filepath.Abs("../../../build-n4-module-kit/toolbelt/lib/slotomatic-skip.sh")
	if err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	bash := func(log string) (string, int) {
		out, err := exec.Command("bash", "-c", `. "$1"; slotomatic_skip_scan "$2"`, "_", lib, log).Output()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("bash slotomatic_skip_scan: %v", err)
		}
		return strings.TrimRight(string(out), "\n"), code
	}
	paths := []string{filepath.Join(d, "missing.log"), d}
	for i, c := range logs {
		p := filepath.Join(d, "log"+string(rune('a'+i)))
		if err := os.WriteFile(p, []byte(c.log), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	for _, p := range paths {
		rows, status := ScanFile(p)
		if out, code := bash(p); out != strings.Join(rows, "\n") || code != int(status) {
			t.Errorf("%s: slotomatic_skip_scan = %q status %d, ScanFile = %q status %d", p, out, code, strings.Join(rows, "\n"), status)
		}
	}
}
