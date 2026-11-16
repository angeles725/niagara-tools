package blackbox

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The spy pages n4_station_runtime reads, served from the recorded fixtures: sysinfo.html says the
// station has been up rtUptime seconds (its ticks), module-manager.html lists six loaded parts.
const (
	rtSysInfo  = "sysInfo"
	rtModules  = "sysManagers/moduleManager"
	rtUptime   = 14034.985
	rtLabDash  = "1.0.0"
	rtTridium  = "4.15.6.20"
	rtNewJar   = "newmod-rt.jar"
	rtOldJar   = "history-rt.jar"
	rtBadJar   = "control-rt.jar"
	rtHomeNote = "--niagara-home"
)

// rtJar is one module jar of a test installation: its file name, the root attributes of its
// META-INF/module.xml (no module.xml when name is "") and its age.
type rtJar struct {
	file, name, module, profile, version string
	age                                  time.Duration
}

// rtBeforeBoot and rtAfterBoot are jar ages on either side of the fixture's boot.
const (
	rtBeforeBoot = 24 * time.Hour
	rtAfterBoot  = 10 * time.Minute
)

// rtInSync is a jar for each loaded part of the fixture, all older than the boot, and one jar the
// station has not loaded.
func rtInSync() []rtJar {
	return []rtJar{
		{"LabDash-rt.jar", "LabDash-rt", "LabDash", "rt", rtLabDash, rtBeforeBoot},
		{"LabDash-ux.jar", "LabDash-ux", "LabDash", "ux", rtLabDash, rtBeforeBoot},
		{"alarm-rt.jar", "alarm-rt", "alarm", "rt", rtTridium, rtBeforeBoot},
		{"baja.jar", "baja", "baja", "rt", rtTridium, rtBeforeBoot},
		{"control-rt.jar", "control-rt", "control", "rt", rtTridium, rtBeforeBoot},
		{"web-rt.jar", "web-rt", "web", "rt", rtTridium, rtBeforeBoot},
		{rtOldJar, "history-rt", "history", "rt", rtTridium, rtBeforeBoot},
	}
}

// rtHome is a Niagara installation directory whose modules directory holds jars.
func rtHome(t *testing.T, jars []rtJar) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "modules")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, j := range jars {
		path := filepath.Join(dir, j.file)
		rtWriteJar(t, path, j)
		when := time.Now().Add(-j.age)
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func rtWriteJar(t *testing.T, path string, j rtJar) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if j.name == "" { // not a jar at all
		_, _ = f.WriteString("not a zip archive")
		return
	}
	z := zip.NewWriter(f)
	w, err := z.Create("META-INF/module.xml")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n"+`<module name=%q vendor="V" vendorVersion=%q moduleName=%q runtimeProfile=%q>`+"\n</module>\n",
		j.name, j.version, j.module, j.profile)
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
}

// rtRig is a connected servlet rig serving the recorded sysInfo and module manager pages; home,
// when not "", is the station's --niagara-home.
func rtRig(t *testing.T, home string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, func(o *Options) {
		if home != "" {
			o.NiagaraHomes = map[string]string{StationName: home}
		}
	})
	shxServeSpy(sv, rtSysInfo, boxstation.SpyPage{Body: shxSpyFixture(t, "sysinfo.html")})
	shxServeSpy(sv, rtModules, boxstation.SpyPage{Body: shxSpyFixture(t, "module-manager.html")})
	return r, sv
}

// rtReasons is the mismatches of out as module -> reason.
func rtReasons(t *testing.T, out map[string]any) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, m := range objects(out["mismatches"]) {
		got[fmt.Sprint(m["module"])] = fmt.Sprint(m["reason"])
	}
	return got
}
