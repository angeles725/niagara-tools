package stationpull

import (
	"reflect"
	"testing"
)

func TestDefaultExcludesAreTheBackupServiceDefaults(t *testing.T) {
	d := DefaultExcludes()
	if want := []string{"*.hdb", "*.adb", "*.lock", "*backup*", "console.*", "config.bog.b*", "config_backup*"}; !reflect.DeepEqual(d.Files, want) {
		t.Errorf("files %v", d.Files)
	}
	if want := []string{"history", "alarm", "webFileCache", "dataRecovery"}; !reflect.DeepEqual(d.Dirs, want) {
		t.Errorf("dirs %v", d.Dirs)
	}
}

func TestSplitList(t *testing.T) {
	if got := SplitList(" a ;b;; c d ;"); !reflect.DeepEqual(got, []string{"a", "b", "c d"}) {
		t.Errorf("got %q", got)
	}
	if got := SplitList(""); len(got) != 0 {
		t.Errorf("got %q", got)
	}
}

func TestExcludedMatchesFileNamesAndDirectoryComponents(t *testing.T) {
	ex := DefaultExcludes()
	for rel, want := range map[string]bool{
		"config.bog": false, "sub/my file.bog": false, "alarmish/x.txt": false, "history.txt": false,
		"a.hdb": true, "x/y/b.ADB": false, "x/y/b.adb": true, "app.lock": true, "old_backup_1.zip": true,
		"console.log": true, "sub/console.txt": true, "config.bog.b1": true, "config_backup2.bog": true,
		"history/a.txt": true, "x/webFileCache/y/z.js": true, "dataRecovery/q": true, "alarm/a.bog": true,
		"x/history": false, // a FILE named like an excluded directory is kept
	} {
		if got := ex.Excluded(rel); got != want {
			t.Errorf("Excluded(%q) = %v, want %v", rel, got, want)
		}
	}
}

func TestOverriddenExcludesReplaceTheDefaults(t *testing.T) {
	ex := Excludes{Files: []string{"*.tmp"}, Dirs: []string{"cache"}}
	if ex.Excluded("a.hdb") || !ex.Excluded("x.tmp") || !ex.Excluded("cache/z") {
		t.Fatal("override not applied as given")
	}
}

func TestValidPatterns(t *testing.T) {
	if err := (Excludes{Files: []string{"[a"}}).Validate(); err == nil {
		t.Error("a malformed pattern must be refused")
	}
	if err := DefaultExcludes().Validate(); err != nil {
		t.Error(err)
	}
}
