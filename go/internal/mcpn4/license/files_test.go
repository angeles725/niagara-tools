package license

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const licFileOK = `<license hostId="H" generated="2026-10-12" version="4.15" maintenanceExpiration="2027-01-01"/>`

// licDir is an installation home whose licences directory holds files (name: content).
func licDir(t *testing.T, files map[string]string) (home, dir string) {
	t.Helper()
	home = t.TempDir()
	dir = filepath.Join(home, "security", "licenses")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home, dir
}

func TestReadFilesReadsOnlyTheRootAttributesOfTheWantedFiles(t *testing.T) {
	t.Parallel()
	home, _ := licDir(t, map[string]string{"A.license": licFileOK, "other.license": licFileOK,
		"ctl.license": "<license hostId=\"H\" generated=\"a\nb\tc\" version=\"  4.15  \"/>"})
	files, skipped, err := ReadFiles(home, []string{"A.license", "ctl.license", "gone.license", "../A.license", "A.license"})
	if err != nil || len(skipped) != 0 || len(files) != 2 {
		t.Fatalf("files %v, skipped %v, err %v", files, skipped, err)
	}
	want := File{Generated: "2026-10-12", Version: "4.15", MaintenanceExpiration: "2027-01-01", hostID: "H"}
	if files["A.license"] != want || !files["A.license"].ForHost("H") || files["A.license"].ForHost("") || files["A.license"].ForHost("h") {
		t.Errorf("A = %+v", files["A.license"])
	}
	if got := files["ctl.license"]; got.Generated != "a b c" || got.Version != "4.15" {
		t.Errorf("control characters and spaces: %+v", got)
	}
}

func TestReadFilesSaysWhyAFileThatIsThereWasNotRead(t *testing.T) {
	t.Parallel()
	big := `<license hostId="H" generated="G">` + strings.Repeat(" ", 1<<20) + `</license>`
	home, dir := licDir(t, map[string]string{"big.license": big, "exact.license": big[:34] + big[34+44:], "Upper.LICENSE": licFileOK,
		"bad.license": `<license hostId="H" generated="G"`, "root.license": `<licence hostId="H"/>`,
		"doctype.license": `<!DOCTYPE license [<!ENTITY x "y">]>` + licFileOK})
	outside := filepath.Join(t.TempDir(), "outside.xml")
	if err := os.WriteFile(outside, []byte(licFileOK), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.license")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "dir.license"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "fifo.license"), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted := []string{"big.license", "exact.license", "Upper.license", "bad.license", "root.license", "doctype.license",
		"link.license", "dir.license", "fifo.license"}
	done := make(chan struct{})
	var files map[string]File
	var skipped map[string]string
	go func() {
		files, skipped, _ = ReadFiles(home, wanted)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("ReadFiles blocked (on the FIFO?)")
	}
	want := map[string]string{"big.license": SkipLarge, "Upper.license": SkipCase, "bad.license": SkipNotLicense,
		"root.license": SkipNotLicense, "doctype.license": SkipNotLicense, "link.license": SkipNotRegular,
		"dir.license": SkipNotRegular, "fifo.license": SkipNotRegular}
	for name, why := range want {
		if skipped[name] != why {
			t.Errorf("%s: skipped for %q, want %q", name, skipped[name], why)
		}
	}
	if _, read := files["exact.license"]; !read || len(files) != 1 || len(skipped) != len(want) {
		t.Errorf("files %v, skipped %v", files, skipped)
	}
}

func TestReadFilesStopsAtTheFileCapAndSaysWhichWereNotRead(t *testing.T) {
	t.Parallel()
	content, wanted := map[string]string{}, []string{}
	for n := 0; n < 150; n++ {
		name := fmt.Sprintf("f%03d.license", n)
		content[name], wanted = licFileOK, append(wanted, name)
	}
	home, _ := licDir(t, content)
	files, skipped, err := ReadFiles(home, wanted)
	if err != nil || len(files) != 100 || len(skipped) != 50 || skipped[wanted[100]] != SkipCap || skipped[wanted[149]] != SkipCap {
		t.Errorf("%d read, %d skipped (%q), err %v", len(files), len(skipped), skipped[wanted[100]], err)
	}
	if _, _, err := ReadFiles(filepath.Join(home, "nope"), wanted); err == nil {
		t.Errorf("a home without a licences directory gave no error")
	}
}
