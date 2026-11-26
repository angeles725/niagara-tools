package stationpull

import (
	"archive/zip"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func crc(s string) uint32 { return crc32.ChecksumIEEE([]byte(s)) }

// fakePlat writes the station files on fget (then prints the harmless exception the real plat prints) and
// lists rows on flist; it records its argv and exits 0, as plat does.
func fakePlat(t *testing.T, rows string, files bool) (home, env, log string) {
	t.Helper()
	home, env = fixture(t, "")
	log = filepath.Join(t.TempDir(), "args.log")
	copyFiles := ""
	if files {
		copyFiles = `mkdir -p "$last/sub" "$last/history"
printf alpha > "$last/a.txt"; printf bravo > "$last/sub/my file.bog"; printf c > "$last/sub/console.txt"
printf d > "$last/history/h.hdb"; printf e > "$last/console.log"; printf f > "$last/config.bog.b1"
`
	}
	rowsFile := filepath.Join(t.TempDir(), "rows.txt")
	if err := os.WriteFile(rowsFile, []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf(`echo "$*" >> %q
echo 'INFO [nre] Booting'
case "$1" in
fget) for a; do last="$a"; done
%s;;
flist) cat %q;;
esac
echo 'javax.baja.sys.BajaRuntimeException: niagarad version ?'
exit 0
`, log, copyFiles, rowsFile)
	if err := os.WriteFile(filepath.Join(home, "bin", "plat"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return home, env, log
}

func listing(alpha, bravo uint32) string {
	return fmt.Sprintf("Name  Size  Date  CRC\na.txt  5  2026-11-25  %d\nmy file.bog  5  2026-11-25  %d\n"+
		"h.hdb  1  2026-11-25  %d\nconsole.txt  1  2026-11-25  0\n", alpha, bravo, crc("d"))
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func TestPullCopiesVerifiesAndZipsWithoutTheDefaultExclusions(t *testing.T) {
	home, env, log := fakePlat(t, listing(crc("alpha"), crc("bravo")), true)
	dest := t.TempDir()
	code, out, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", dest)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d err %q out %q", code, errs, out)
	}
	m := regexp.MustCompile(`^station-pull: (\S+/LAB-\d{8}T\d{6}Z\.zip) copied=6 verified=3 zipped=2 bytes=(\d+) excluded=4\n$`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("summary:\n%s", out)
	}
	if st, err := os.Stat(m[1]); err != nil || filepath.Dir(m[1]) != dest || fmt.Sprint(st.Size()) != m[2] {
		t.Fatalf("archive %v %v", st, err)
	}
	if got := strings.Join(zipNames(t, m[1]), ","); got != "LAB/a.txt,LAB/sub/my file.bog" {
		t.Errorf("zip holds %s", got)
	}
	logged, _ := os.ReadFile(log)
	want := "fget -recurse -overwrite:none -h:ip:localhost -p:5011 -secure -noinput -usr:admin -pwd:" + secret +
		" ~stations/LAB " + filepath.Join(dest, "station")
	if !strings.Contains(string(logged), want) || !strings.Contains(string(logged), "flist -recurse ") {
		t.Errorf("plat argv:\n%s", logged)
	}
	if strings.Contains(out+errs, secret) {
		t.Error("password printed")
	}
}

func TestExcludeFlagsReplaceTheDefaults(t *testing.T) {
	home, env, _ := fakePlat(t, listing(crc("alpha"), crc("bravo")), true)
	dest := t.TempDir()
	code, out, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", dest,
		"--exclude-files", "*.log;console.*", "--exclude-dirs", "sub")
	if code != 0 || !strings.Contains(out, "zipped=3 ") || !strings.Contains(out, "excluded=3\n") {
		t.Fatalf("exit %d err %q out %q", code, errs, out)
	}
}

func TestACRCMismatchOrAMissingFileFailsAndNamesIt(t *testing.T) {
	home, env, _ := fakePlat(t, listing(999, crc("bravo"))+"ghost file.txt  1  2026-11-25  5\n", true)
	dest := t.TempDir()
	code, out, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", dest)
	if code != 1 || out != "" || !strings.Contains(errs, "crc mismatch: a.txt") ||
		!strings.Contains(errs, "missing: ghost file.txt") || strings.Contains(errs, secret) {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
	if zips, _ := filepath.Glob(filepath.Join(dest, "*.zip")); len(zips) != 0 {
		t.Errorf("an archive was written: %v", zips)
	}
}

func TestNothingCopiedFailsWithPlatsReasonNotItsExitCode(t *testing.T) {
	home, env, _ := fakePlat(t, listing(1, 2), false)
	code, _, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", t.TempDir())
	if code != 1 || !strings.Contains(errs, "copied no files") {
		t.Fatalf("exit %d err %q", code, errs)
	}
}

func TestAnEmptyListingCannotBeVerifiedAndFails(t *testing.T) {
	home, env, _ := fakePlat(t, "", true)
	code, _, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", t.TempDir())
	if code != 1 || !strings.Contains(errs, "listed no files") {
		t.Fatalf("exit %d err %q", code, errs)
	}
}

func TestAnAuthenticationFailureIsNamedAndMasked(t *testing.T) {
	home, env := fixture(t, "echo 'javax.baja.security.AuthenticationException: bad -pwd:"+secret+"'\nexit 0\n")
	code, out, errs := run(t, home, "--env", env, "--station", "LAB", "--dest", t.TempDir())
	if code != 1 || !strings.Contains(errs, "AuthenticationException") || strings.Contains(out+errs, secret) {
		t.Fatalf("exit %d out %q err %q", code, out, errs)
	}
}
