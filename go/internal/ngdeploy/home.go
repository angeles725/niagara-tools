package ngdeploy

import (
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// guardHome looks at NIAGARA_HOME before the backup and before gradle runs. A value that is not a directory
// is refused (homeIsDir). gradle's :jar step installs the built jars into <NIAGARA_HOME>/modules before any check of the deploy has run, and in a real installation
// that directory is a station's: a home that is there and is not a mirror (modjar.IsMirror, the marker
// build.sh reads) is refused with exit 10 unless --install-into-home, the option build.sh has.
//
// Then the run locks the directories it changes, in a fixed order so that two runs cannot wait on each
// other: <NIAGARA_HOME>/modules, then the station directory when it is another one and the run deploys
// (lock.go). Only with the locks held does it look: a jar of the module left aside in the home by a run
// that was killed is refused as the copy refuses one in the station. When STATION_MODULES_DIR is that same
// directory (sameDirectory), a line says so, the copy step only verifies (copySame), and what the copy
// refuses under the name of a jar is refused here, before the build moves it; --no-deploy there is refused
// (exit 10): it keeps what gradle installed, and the station would get jars no check has passed.
func (d *deploy) guardHome() int {
	home := d.env("NIAGARA_HOME")
	if code := d.homeIsDir(home); code != proceed {
		return code
	}
	if !modjar.IsMirror(home) && !modjar.IsMirror(home) && !d.isOn(optIntoHome) {
		return d.fail(ExitEnv, msgRealHome, home, modjar.MirrorMarker, intoHomeFlag)
	}
	if isDir(home + homeModules) { // else there is nothing there for gradle to replace
		d.homeDir = home + homeModules
		d.sameDir = sameDirectory(d.station, d.homeDir)
		if code := d.lock(d.homeDir); code != proceed {
			return code
		}
	}
	if !d.sameDir && !d.isOn(optNoDeploy) {
		if code := d.lock(d.station); code != proceed {
			return code
		}
	}
	if d.homeDir == "" {
		return proceed
	}
	if code := d.refuseAsideIn(d.homeDir, backupProfiles, ExitEnv, msgHomeAsideFound); code != proceed {
		return code
	}
	if !d.sameDir {
		return d.homeJarFiles()
	}
	if d.isOn(optNoDeploy) { // it would keep unchecked jars in the station (bench Δ23)
		return d.fail(ExitEnv, msgSameDirNoDeploy, d.homeDir)
	}
	d.say(msgSameDir, d.homeDir)
	for _, profile := range profilesOf[d.raw[optMode]] {
		if _, code := d.jarFileAt(d.jarName(profile), d.station+"/"+d.jarName(profile)); code != proceed {
			return code
		}
	}
	return proceed
}

// homeIsDir refuses a NIAGARA_HOME that names no directory, or names a file, with exit 10 (bench Δ25): gradle
// builds against it and installs the built jars in its modules/, and what it does with a value that is not
// a directory is not the deploy's to find out after the build. A symbolic link to a directory is one.
func (d *deploy) homeIsDir(home string) int {
	switch info, err := os.Stat(home); {
	case err != nil:
		return d.fail(ExitEnv, msgHomeMissing, home)
	case !info.IsDir():
		return d.fail(ExitEnv, msgHomeNotADir, home)
	}
	return proceed
}

// homeJarFiles refuses, before the build and with or without --no-deploy, anything that is not a jar file
// under the name of a jar the build installs in <NIAGARA_HOME>/modules when that is not the station: the
// move aside would take it, and putting it back over the installed jar fails. The same directory is
// looked at as the station (guardHome).
func (d *deploy) homeJarFiles() int {
	for _, profile := range profilesOf[d.raw[optMode]] {
		name := d.jarName(profile)
		if _, code := d.jarFileIn(homeSite(d.homeDir), msgHomeNotAJar, name, d.homeDir+"/"+name); code != proceed {
			return code
		}
	}
	return proceed
}

// sameDirectory reports whether two paths name one directory, however each is spelled: relative, through
// symbolic links, in another case on a mount that ignores case (os.SameFile asks the file system). When
// one of them is not there to ask about, the two are compared as absolute paths with their links followed.
func sameDirectory(a, b string) bool {
	x, errX := os.Stat(a)
	y, errY := os.Stat(b)
	if errX == nil && errY == nil {
		return os.SameFile(x, y)
	}
	return resolved(a) == resolved(b)
}

// resolved is an absolute path with every symbolic link followed; one that cannot be resolved is itself,
// absolute and clean.
func resolved(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return filepath.Clean(dir)
}

// asideHome moves the module's own jars in <NIAGARA_HOME>/modules aside just before the build, with the
// move the copy moves the station's jars aside with (moveAside: it never replaces a file already aside), so
// that settleHome can put them back whatever gradle installs in their place. A jar that cannot be moved
// aside stops the deploy with exit 10 before gradle runs, the ones already aside moved back.
func (d *deploy) asideHome() int {
	if d.homeDir == "" {
		return proceed
	}
	for _, profile := range backupProfiles {
		jar := &stagedJar{name: d.jarName(profile), target: d.homeDir + "/" + d.jarName(profile)}
		jar.aside, jar.had = jar.target+asideSuffix, present(jar.target)
		d.homeJars = append(d.homeJars, jar)
		if !jar.had {
			continue
		}
		if err := moveAside(jar.target, jar.aside); err != nil {
			jars := d.homeJars
			d.homeJars = nil
			return d.failHeld(err, d.homeDir, ExitEnv, msgHomeAsideFailed, jar.name, d.homeDir, strerror.Text(err), undo(jars, homeSite(d.homeDir)))
		}
		jar.moved = true
	}
	return proceed
}

// deployedJar is the jar the station held under a name before this deploy: in the station, or aside when
// the station directory is <NIAGARA_HOME>/modules, where the build has replaced it by then.
func (d *deploy) deployedJar(name string) string {
	if d.sameDir {
		return d.station + "/" + name + asideSuffix
	}
	return d.station + "/" + name
}

// interrupted turns the exit code of a phase into 128 + the signal when a SIGINT or SIGTERM arrived since
// the run took its first lock and the copy has not gone through: the run stops there, with a line,
// whatever the phase answered (a build the signal ended says so itself first). The copy reports its own.
func (d *deploy) interrupted(code int) int {
	if d.signals == nil || d.copied || !d.signals.caught() || code == d.signals.exitCode() {
		return code
	}
	return d.fail(d.signals.exitCode(), msgInterrupted, d.signals.name())
}
