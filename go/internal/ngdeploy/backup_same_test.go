package ngdeploy

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archiveContents is name -> bytes of the regular files in the one backup of the module the run left.
func archiveContents(f *fixture) map[string]string {
	f.t.Helper()
	backups, _ := filepath.Glob(f.work + "/" + backupDir + "/demo-pre-*.tar.gz")
	if len(backups) != 1 {
		f.t.Fatalf("backups: %v, want one", backups)
	}
	file, err := os.Open(backups[0])
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		f.t.Fatal(err)
	}
	got := map[string]string{}
	for r := tar.NewReader(unzipped); ; {
		h, err := r.Next()
		if err == io.EOF {
			return got
		}
		if err != nil {
			f.t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			b, _ := io.ReadAll(r)
			got[h.Name] = string(b)
		}
	}
}

// When the station is <NIAGARA_HOME>/modules the build has replaced its jars by the time the backup runs
// (just before the copy): the backup holds the previous jars, from the files aside, under their jar names,
// and a full backup leaves out the jars the build put there.
func TestABackupOfTheHomeModulesDirectoryHoldsThePreviousJars(t *testing.T) {
	for _, full := range []bool{false, true} {
		f := archiveFixture(t)
		modules := sameDir(t, f)
		f.set("STATION_MODULES_DIR", modules) // a full backup archives the directory as named: not through a link
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		args := []string{}
		prefix := ""
		if full {
			args, prefix = append(args, "--full-backup"), "modules/"
		}
		f.want(f.run(args...), ExitOK, "[ng-deploy] copy ok\n")
		got := archiveContents(f)
		for name, want := range map[string]string{"demo-rt.jar": "home-old-rt", "demo-ux.jar": "home-old-ux", "demo-wb.jar": "home-old-wb"} {
			if got[prefix+name] != want {
				t.Errorf("full %v: %s holds %q, want %q; archive %v", full, name, got[prefix+name], want, got)
			}
		}
		for name := range got {
			if strings.Contains(name, asideSuffix) {
				t.Errorf("full %v: the archive holds %s", full, name)
			}
		}
		if f.read(modules+"/demo-rt.jar") != "rtjar" {
			t.Errorf("full %v: the deploy did not go through:\n%s", full, tree(t, modules))
		}
	}
}

// A full backup of the home modules directory renames and leaves out only this module's own jars: a jar the
// mode did not build is archived (mode C builds rt only), another module's file aside keeps its name and
// never stands in for that module's live jar, and a jar the build installed where none was is left out
// (review R1-001/R3/R4 full-backup findings).
func TestAFullBackupOfTheHomeModulesDirectoryNamesOnlyThisModulesJars(t *testing.T) {
	f := archiveFixture(t)
	modules := sameDir(t, f)
	f.set("STATION_MODULES_DIR", modules)
	f.write(modules+"/other-rt.jar"+asideSuffix, "another module's leftover")
	f.write(f.dir+"/do/gradlew", installing(f.dir+"/home", "", "rt"))
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.want(f.run("--full-backup", "--mode", "C"), ExitOK, "[ng-deploy] copy ok\n")
	got := archiveContents(f)
	for name, want := range map[string]string{
		"modules/demo-rt.jar": "home-old-rt", "modules/demo-ux.jar": "home-old-ux", "modules/demo-wb.jar": "home-old-wb",
		"modules/other-rt.jar": "another module", "modules/other-rt.jar" + asideSuffix: "another module's leftover",
	} {
		if got[name] != want {
			t.Errorf("%s holds %q, want %q; archive %v", name, got[name], want, got)
		}
	}

	f = archiveFixture(t) // the ux jar was not there: the one the build installed is not in the backup
	modules = sameDir(t, f)
	f.set("STATION_MODULES_DIR", modules)
	f.remove(modules + "/demo-ux.jar")
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.want(f.run("--full-backup"), ExitOK, "[ng-deploy] copy ok\n")
	if got := archiveContents(f); got["modules/demo-rt.jar"] != "home-old-rt" || got["modules/demo-ux.jar"] != "" {
		t.Errorf("archive %v", got)
	}
}

// The --transform expression matches the file aside literally: every character a sed basic regular
// expression or replacement reads specially is escaped.
func TestTheTransformQuotesItsNamesForSed(t *testing.T) {
	if got, want := breQuote(`a.b*c[d]^e$f\g,h+i?(j)|k{l}`), `a\.b\*c\[d\]\^e\$f\\g\,h+i?(j)|k{l}`; got != want {
		t.Errorf("breQuote = %s, want %s", got, want)
	}
	if got, want := sedReplacement(`a&b\c,d.e`), `a\&b\\c\,d.e`; got != want {
		t.Errorf("sedReplacement = %s, want %s", got, want)
	}
}
