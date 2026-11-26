package stationpull

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Verifier regression tests (PR for #1169): names, unreadable CRCs, an unverified copy and the
// archive's mode.

func TestPasswordMaskedInFlistOutputAndNames(t *testing.T) {
	home, env, _ := fakePlat(t, "a.txt  5  2026-11-25  1\n"+secret+".txt  1  2026-11-25  5\n"+
		"javax.baja.security.AuthenticationException: "+secret+"\n", true)
	code, out, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", t.TempDir())
	if code != 1 || strings.Contains(out+errs, secret) || !strings.Contains(errs, "<masked>") {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
}

func TestStationNameOfOnlyDotsIsRefused(t *testing.T) {
	home, env := fixture(t, "exit 0\n")
	for _, n := range []string{".", "..", "..."} {
		if code, _, _ := run(t, home, "--env", env, "--station", n, "--dest", t.TempDir()); code != 3 {
			t.Errorf("%q: exit %d, want 3", n, code)
		}
	}
}

func TestAnUnparsableCRCRowFailsClosed(t *testing.T) {
	home, env, _ := fakePlat(t, "a.txt  5  2026-11-25  "+fmt.Sprint(crc("alpha"))+
		"\nmy file.bog  5  2026-11-25  99999999999\n", true)
	code, _, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", t.TempDir())
	if code != 1 || !strings.Contains(errs, "my file.bog") {
		t.Errorf("exit %d err %q, want 1 naming the row", code, errs)
	}
}

func TestAllZeroCRCListingIsNotASilentSuccess(t *testing.T) {
	home, env, _ := fakePlat(t, "a.txt  5  2026-11-25  0\nmy file.bog  5  2026-11-25  0\n", true)
	d := t.TempDir()
	code, _, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", d)
	if code != 1 || !strings.Contains(errs, "no file") {
		t.Errorf("exit %d err %q, want 1: nothing was verified", code, errs)
	}
	if zips, _ := filepath.Glob(filepath.Join(d, "*.zip")); len(zips) != 0 {
		t.Errorf("an unverified copy wrote %v", zips)
	}
}

func TestTheArchiveIsPrivate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.bog"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "s.zip")
	if _, _, err := Zip(root, archive, "s", DefaultExcludes()); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(archive); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("archive mode %v err %v, want 0600", fi.Mode().Perm(), err)
	}
}
