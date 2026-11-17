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
	// A second backup in the same second never overwrites the first: the zip fails, the save stands.
	_, _, out = r.write(t, "n4_save_station", obj("backup", true))
	if e := get(get(out, "backup").(*pyjson.Object), "error"); e == nil || !strings.Contains(e.(string), "file exists") {
		t.Errorf("second backup %s", pyjson.Repr(get(out, "backup")))
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
