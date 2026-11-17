package writeops

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// stationTree writes a station directory with files the BackupService keeps and leaves out.
func stationTree(t *testing.T, home string) {
	t.Helper()
	for rel, body := range map[string]string{
		"shared/px/Main.px": "<px/>", "user/notes.txt": "n", "config.bog.lock": "", "config_backup_261116_1414.bog": "old",
		"config.bog.bak-2026": "b", "history/x/data.hdb": "h", "alarm/alarm.adb": "a", "webFileCache/c": "c",
		"shared/console.log": "l", "sub/history/kept.txt": "k",
	} {
		p := filepath.Join(home, rel)
		os.MkdirAll(filepath.Dir(p), 0o700)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSaveWithBackupZipsTheStationDirectoryWithoutTheBackupServiceExclusions(t *testing.T) {
	r := newSRig(t, "bog-before")
	stationTree(t, r.home)
	old := backupNow
	backupNow = func() time.Time { return time.Date(2026, 11, 17, 6, 1, 2, 0, time.UTC) }
	t.Cleanup(func() { backupNow = old })
	text, isErr, out := r.write(t, "n4_save_station", obj("backup", true))
	if isErr {
		t.Fatal(text)
	}
	if get(out, "verdict") != "verified" || get(out, "persisted") != true {
		t.Fatalf("save %s", pyjson.Repr(out))
	}
	b, ok := get(out, "backup").(*pyjson.Object)
	if !ok {
		t.Fatalf("no backup in %s", pyjson.Repr(out))
	}
	want := filepath.Join(r.dir, "backups", "FakeStation-20261117T060102Z.zip")
	if get(b, "path") != want || get(b, "files") != 4 {
		t.Fatalf("backup %s", pyjson.Repr(b))
	}
	if info, err := os.Stat(want); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("zip %v %v", info, err)
	}
	zr, err := zip.OpenReader(want)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	if got := strings.Join(names, ","); got != "config.bog,shared/px/Main.px,sub/history/kept.txt,user/notes.txt" {
		t.Errorf("zip holds %s", got)
	}
	if got := pyjson.Repr(get(b, "excluded_dirs")); got != "['alarm', 'history', 'webFileCache']" {
		t.Errorf("excluded_dirs %s", got)
	}
	if got := pyjson.Repr(get(b, "excluded_files")); got != "['config.bog.bak-2026', 'config.bog.lock', 'config_backup_261116_1414.bog', 'shared/console.log']" {
		t.Errorf("excluded_files %s", got)
	}
	// A second backup in the same second never overwrites the first: the zip fails, the save
	// stands, the verdict says the backup asked for does not exist, and the first zip is kept.
	_, _, out = r.write(t, "n4_save_station", obj("backup", true))
	if e := get(get(out, "backup").(*pyjson.Object), "error"); e == nil || !strings.Contains(e.(string), "file exists") {
		t.Errorf("second backup %s", pyjson.Repr(get(out, "backup")))
	}
	if get(out, "verdict") != "mismatch" || get(out, "persisted") != true {
		t.Errorf("failed backup: verdict %v persisted %v", get(out, "verdict"), get(out, "persisted"))
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("the first zip was removed: %v", err)
	}
}

func TestBackupIsRefusedBeforeAnyCallWithoutAStationHome(t *testing.T) {
	r := newSRig(t, "")
	text, isErr, _ := r.call(t, "n4_save_station", obj("backup", true))
	if !isErr || !strings.Contains(text, "backup needs --station-home FakeStation=") || r.st.saves != 0 {
		t.Fatalf("%v %s (%d saves)", isErr, text, r.st.saves)
	}
}

func TestSaveWithoutBackupWritesNoZip(t *testing.T) {
	r := newSRig(t, "bog-before")
	out := r.save(t)
	if get(out, "backup") != nil {
		t.Errorf("backup %s", pyjson.Repr(get(out, "backup")))
	}
	if _, err := os.Stat(filepath.Join(r.dir, "backups")); !os.IsNotExist(err) {
		t.Errorf("backups dir exists: %v", err)
	}
}

func TestBackupPlanNoteNamesTheDestinationAndTheExclusions(t *testing.T) {
	r := newSRig(t, "bog")
	_, _, out := r.call(t, "n4_save_station", obj("backup", true))
	notes := pyjson.Repr(get(get(out, "plan").(*pyjson.Object), "notes"))
	if !strings.Contains(notes, "is zipped to "+r.dir+"/backups/<station>-<UTC time>.zip") || !strings.Contains(notes, "'webFileCache'") {
		t.Errorf("notes %s", notes)
	}
}

var probeNow = time.Date(2026, 11, 17, 6, 0, 0, 0, time.UTC)

func zipNames(t *testing.T, p string) string {
	t.Helper()
	zr, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// Verifier probe 1: a --station-home that is a symbolic link to the station directory is walked
// as that directory (filepath.WalkDir alone would zip nothing and call it success).
func TestZipFollowsASymlinkedStationHome(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	os.MkdirAll(filepath.Join(real, "shared"), 0o700)
	os.WriteFile(filepath.Join(real, "config.bog"), []byte("b"), 0o600)
	os.WriteFile(filepath.Join(real, "shared", "a.px"), []byte("p"), 0o600)
	link := filepath.Join(tmp, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	out, err := zipStationHome(link, filepath.Join(tmp, "state"), "S", probeNow)
	if err != nil || get(out, "files") != 2 || zipNames(t, get(out, "path").(string)) != "config.bog,shared/a.px" {
		t.Fatalf("%v %s", err, pyjson.Repr(out))
	}
}

// A directory without config.bog is not a station directory: an error and no zip left behind.
func TestZipRefusesAHomeWithoutConfigBogAndLeavesNoZip(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	os.MkdirAll(home, 0o700)
	os.WriteFile(filepath.Join(home, "x.txt"), []byte("x"), 0o600)
	_, err := zipStationHome(home, filepath.Join(tmp, "state"), "S", probeNow)
	if err == nil || !strings.Contains(err.Error(), "no config.bog") {
		t.Fatalf("err %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(tmp, "state", "backups", "*.zip")); len(left) != 0 {
		t.Errorf("left %v", left)
	}
}

// Verifier probe 3: the backups directory must be private and not a link.
func TestZipRefusesAnOpenOrLinkedBackupsDirectory(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	os.MkdirAll(home, 0o700)
	os.WriteFile(filepath.Join(home, "config.bog"), []byte("b"), 0o600)
	open := filepath.Join(tmp, "open")
	os.MkdirAll(filepath.Join(open, "backups"), 0o700)
	os.Chmod(filepath.Join(open, "backups"), 0o777)
	if _, err := zipStationHome(home, open, "S", probeNow); err == nil || !strings.Contains(err.Error(), "open to other users") {
		t.Errorf("open dir: %v", err)
	}
	linked := filepath.Join(tmp, "linked")
	os.MkdirAll(linked, 0o700)
	os.MkdirAll(filepath.Join(tmp, "elsewhere"), 0o700)
	os.Symlink(filepath.Join(tmp, "elsewhere"), filepath.Join(linked, "backups"))
	if _, err := zipStationHome(home, linked, "S", probeNow); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("linked dir: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(tmp, "elsewhere", "*")); len(left) != 0 {
		t.Errorf("wrote through the link: %v", left)
	}
}

// Verifier probe 4: a state dir inside the station home is not zipped into itself.
func TestZipLeavesOutAStateDirInsideTheStationHome(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, "config.bog"), []byte("b"), 0o600)
	state := filepath.Join(home, "st")
	out, err := zipStationHome(home, state, "S", probeNow)
	if err != nil || zipNames(t, get(out, "path").(string)) != "config.bog" || pyjson.Repr(get(out, "excluded_dirs")) != "['st']" {
		t.Fatalf("%v %s", err, pyjson.Repr(out))
	}
}
