package ngdeploy

import (
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// guardHome looks at NIAGARA_HOME before the backup and before gradle runs. gradle's :jar step installs the
// built jars into <NIAGARA_HOME>/modules before any check of the deploy has run, and in a real installation
// that directory is a station's: a home that is there and is not a mirror (modjar.IsMirror, the marker
// build.sh reads) is refused with exit 10 unless --install-into-home, the option build.sh has. A jar of the
// module left aside there by a run that was killed is refused as the copy refuses one in the station. When
// STATION_MODULES_DIR is that same directory (both resolved), a line says so and the copy step only
// verifies (copySame).
func (d *deploy) guardHome() int {
	home := d.env("NIAGARA_HOME")
	if isDir(home) && !modjar.IsMirror(home) && !d.isOn(optIntoHome) {
		return d.fail(ExitEnv, msgRealHome, home, modjar.MirrorMarker, intoHomeFlag)
	}
	if !isDir(home + homeModules) {
		return proceed // nothing there for gradle to replace
	}
	d.homeDir = home + homeModules
	if code := d.refuseAsideIn(d.homeDir, backupProfiles, ExitEnv, msgHomeAsideFound); code != proceed {
		return code
	}
	if resolved(d.station) == resolved(d.homeDir) {
		d.sameDir = true
		d.say(msgSameDir, d.homeDir)
	}
	return proceed
}

// resolved is a directory with every symbolic link followed; a path that cannot be resolved is itself.
func resolved(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return filepath.Clean(dir)
}

// asideHome moves the module's own jars in <NIAGARA_HOME>/modules aside just before the build, with the
// rename the copy moves the station's jars aside with, so that settleHome can put them back whatever gradle
// installs in their place. From here to the copy a SIGINT or SIGTERM is caught (interrupted). A jar that
// cannot be moved aside stops the deploy with exit 10 before gradle runs, the ones already aside moved back.
func (d *deploy) asideHome() int {
	if d.homeDir == "" {
		return proceed
	}
	d.signals = watchInterrupts()
	for _, profile := range backupProfiles {
		jar := &stagedJar{name: d.jarName(profile), target: d.homeDir + "/" + d.jarName(profile)}
		jar.aside, jar.had = jar.target+asideSuffix, present(jar.target)
		d.homeJars = append(d.homeJars, jar)
		if !jar.had {
			continue
		}
		if err := rename(jar.target, jar.aside); err != nil {
			jars := d.homeJars
			d.homeJars = nil
			return d.fail(ExitEnv, msgHomeAsideFailed, jar.name, d.homeDir, strerror.Text(err), undo(jars, homeSite(d.homeDir)))
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
// the home's jars were moved aside and the copy has not gone through: the run stops there, with a line,
// whatever the phase answered (a build the signal ended says so itself first). The copy reports its own.
func (d *deploy) interrupted(code int) int {
	if d.signals == nil || d.copied || !d.signals.caught() || code == d.signals.exitCode() {
		return code
	}
	return d.fail(d.signals.exitCode(), msgInterrupted, d.signals.name())
}

// settleHome ends what asideHome began, with the exit code of the run. A deploy that went through (or a
// build with --no-deploy) keeps what gradle installed: the jar aside of each jar it replaced is removed, and
// a jar gradle did not install again is moved back. Any other end puts <NIAGARA_HOME>/modules back as it
// was before the build with the copy's putBack: every jar aside is moved back over whatever took its
// place, and a jar the directory did not have is removed. A NOTICE says so; a jar that could not be put
// back or removed is an ERROR line in the copy's jar-by-jar form, and the exit code stays the one given.
func (d *deploy) settleHome(code int) {
	if d.signals != nil {
		d.signals.stop()
	}
	if d.homeJars == nil {
		return
	}
	if code == ExitOK || d.copied {
		d.keepHome()
		return
	}
	touched := false
	for _, jar := range d.homeJars {
		jar.placed = present(jar.target)
		touched = touched || jar.moved || jar.placed
	}
	switch _, mixed := putBack(d.homeJars, homeSite(d.homeDir)); {
	case mixed != "":
		d.fail(code, "%s", mixed)
	case touched:
		d.warn(noticeHomeRestored, d.homeDir)
	}
}

// keepHome tidies <NIAGARA_HOME>/modules after a deploy that went through; what it cannot do is a warning.
func (d *deploy) keepHome() {
	for _, jar := range d.homeJars {
		switch {
		case !jar.moved:
		case present(jar.target):
			if err := remove(jar.aside); err != nil {
				d.warn(warnHomeAsideLeft, jar.name+asideSuffix, d.homeDir, strerror.Text(err))
			}
		default:
			if err := rename(jar.aside, jar.target); err != nil {
				d.warn(warnHomeNotBack, jar.name, d.homeDir, jar.name+asideSuffix, strerror.Text(err))
			}
		}
	}
}

// copySame is the copy step when STATION_MODULES_DIR is <NIAGARA_HOME>/modules: the build installed the
// jars there, so nothing is copied over them. Each jar of the mode must be there with the bytes of the
// built one; a jar that is not is exit 40, and settleHome puts the previous jars back.
func (d *deploy) copySame() int {
	var names []string
	for _, built := range d.builtJars() {
		name := basename(built)
		if !sameBytes(built, d.station+"/"+name) {
			return d.fail(ExitCopy, msgCopyNotInstalled, name)
		}
		names = append(names, name)
	}
	d.copied = true
	d.say(msgCopySame, strings.Join(names, " "), d.homeDir)
	d.say(msgCopyOK)
	return proceed
}
