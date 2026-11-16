package readtools

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// diskJar is one jar of the modules directory: its file name and modification time and, when its
// module.xml was read, the part it holds (key, the name the module manager lists) and version.
type diskJar struct {
	file, stem, key, version string
	modified                 time.Time
	read, unreadable         bool
}

// diskScan is the modules directory of an installation: its jars in name order, the entries
// named like jars that are not regular files (never opened), the jars whose module.xml could not
// be read, and whether the scan stopped at a cap before the end.
type diskScan struct {
	jars       []diskJar
	skipped    []string
	unreadable []string
	cut        bool
}

// scanJars lists dir and reads module.xml only of the jars it needs: those that may hold a
// loaded part (by file name) and, when boot is known, those changed after it. Reading stops at
// deadline; the jars left unread then make the scan cut.
func scanJars(dir string, loaded map[string]bool, boot time.Time, bootKnown bool, deadline time.Time) (diskScan, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return diskScan{}, err
	}
	var scan diskScan
	for _, e := range entries {
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), runtimeJarSuffix) {
			continue
		}
		if !e.Type().IsRegular() {
			scan.skipped = append(scan.skipped, name)
			continue
		}
		if len(scan.jars) >= runtimeMaxJars {
			scan.cut = true
			break
		}
		info, err := e.Info()
		if err != nil {
			scan.unreadable = append(scan.unreadable, name)
			continue
		}
		jar := diskJar{file: name, stem: strings.TrimSuffix(name, filepath.Ext(name)), modified: info.ModTime()}
		jar.key = jar.stem
		if mayHold(jar.stem, loaded) || (bootKnown && afterBoot(jar.modified, boot)) {
			if time.Now().After(deadline) {
				scan.cut = true
				break
			}
			jar.readModule(filepath.Join(dir, name))
			if jar.unreadable {
				scan.unreadable = append(scan.unreadable, name)
			}
		}
		scan.jars = append(scan.jars, jar)
	}
	return scan, nil
}

// mayHold reports whether a jar named stem may hold a loaded part: the part's own name, or the
// module name of a part with a profile suffix.
func mayHold(stem string, loaded map[string]bool) bool {
	if loaded[stem] {
		return true
	}
	for _, p := range runtimeProfiles {
		if loaded[stem+"-"+p] {
			return true
		}
	}
	return false
}

// afterBoot reports whether a jar modified at mod changed after boot (with runtimeBootSlack).
func afterBoot(mod, boot time.Time) bool { return mod.After(boot.Add(runtimeBootSlack)) }

// readModule reads the root attributes of the jar's module.xml: the part it holds and its
// version. Only that entry is read, at most runtimeMaxXMLBytes of it.
func (j *diskJar) readModule(path string) {
	z, err := zip.OpenReader(path)
	if err != nil {
		j.unreadable = true
		return
	}
	defer z.Close()
	f, err := z.Open(runtimeModuleXML)
	if err != nil {
		j.unreadable = true
		return
	}
	defer f.Close()
	dec := xml.NewDecoder(io.LimitReader(f, runtimeMaxXMLBytes))
	for {
		tok, err := dec.Token()
		if err != nil {
			j.unreadable = true
			return
		}
		if el, ok := tok.(xml.StartElement); ok {
			attrs := map[string]string{}
			for _, a := range el.Attr {
				attrs[a.Name.Local] = a.Value
			}
			if el.Name.Local != "module" || attrs["name"] == "" {
				j.unreadable = true
				return
			}
			j.key, j.version, j.read = partKey(attrs["name"], attrs["runtimeProfile"]), attrs["vendorVersion"], true
			return
		}
	}
}

// partKey is the name the module manager lists for a module.xml name and runtime profile: the
// name itself when it already ends with the profile (alarm-rt), else name-profile (baja-rt).
func partKey(name, profile string) string {
	if profile == "" || strings.HasSuffix(name, "-"+profile) {
		return name
	}
	return name + "-" + profile
}
