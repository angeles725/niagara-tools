package ngdeploy

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// settleHome ends what asideHome began, with the exit code of the run. A deploy that went through (or a
// build with --no-deploy) keeps what gradle installed: the jar aside of each jar it replaced is removed, and
// a jar gradle did not install again is moved back. Any other end puts <NIAGARA_HOME>/modules back as it
// was before the build with the copy's putBack: every jar aside is moved back over whatever took its
// place, and a jar the directory did not have is removed when it is the jar this run built (notOurs). A
// NOTICE says so; a jar that could not be put back or removed is an ERROR line in the copy's jar-by-jar
// form, and the exit code stays the one given.
func (d *deploy) settleHome(code int) {
	if d.homeJars == nil {
		return
	}
	if code == ExitOK || d.copied {
		d.keepHome()
		return
	}
	moved, installed := false, false
	for i, jar := range d.homeJars {
		jar.placed = present(jar.target)
		moved, installed = moved || jar.moved, installed || jar.placed
		if jar.placed && !jar.moved && !sameBytes(d.builtJar(backupProfiles[i]), jar.target) {
			d.warn(warnHomeNotOurs, jar.name, d.homeDir)
			jar.placed = false // not this run's to remove
		}
	}
	switch _, mixed := putBack(d.homeJars, homeSite(d.homeDir)); {
	case mixed != "":
		d.fail(code, "%s", mixed)
	case installed:
		d.warn(noticeHomeRestored, d.homeDir)
	case moved:
		d.warn(noticeHomeMovedBack, d.homeDir)
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

// finish is every way out of Main once the phases began: the home is settled, then the locks are removed
// and the watch for signals ends.
func (d *deploy) finish(code int) int {
	d.settleHome(code)
	d.unlock()
	if d.signals != nil {
		d.signals.stop()
	}
	return code
}
