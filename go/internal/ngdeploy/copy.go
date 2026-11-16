package ngdeploy

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// rename moves a jar of the copy and remove deletes one; a test replaces them. A jar is moved aside with
// moveAside (aside.go), which never replaces a file already aside.
var (
	rename = os.Rename
	remove = os.Remove
)

// stagedJar is one jar of the mode on its way into the station.
type stagedJar struct {
	name   string // <MODULE>-<profile>.jar
	target string // the jar in the station modules directory
	staged string // the new jar beside it, until it is renamed in
	aside  string // the station's jar beside it, from the move aside until the copy is done or undone
	had    bool   // the station held a jar file of that name before the copy
	moved  bool   // it was moved aside
	placed bool   // the new jar was renamed in
}

// copyJars puts the built jars of the mode into the station modules directory, all or none: it runs only
// after every precheck, verify and the gate passed. Each jar is copied with cp to a staged name beside its
// target; then the station's jars are moved aside, the staged ones renamed in, and the ones aside removed.
// Every step after the cp is a rename within one directory, so a jar is replaced whole or not at all, and
// a step that fails, or a SIGINT or SIGTERM between two steps (interrupt.go), is undone from the jars
// moved aside (restore.go), with or without a backup. A failure is exit 40 with a line that says what the
// station holds; no staged file outlives the run, and a jar stays aside only when it could not be moved
// back or removed. The copy starts only in a station that holds no jar aside and nothing odd under the
// name of a jar (aside.go).
func (d *deploy) copyJars() int {
	if d.sameDir {
		return d.copySame()
	}
	jars, code := d.jarsToCopy()
	if code != proceed {
		return code
	}
	signals := watchInterrupts()
	defer signals.stop()
	defer func() {
		for _, jar := range jars {
			os.RemoveAll(jar.staged)
		}
	}()
	for i, jar := range jars {
		staged := !signals.caught() && d.stage(jar, d.builtJars()[i])
		if signals.caught() {
			return d.copyInterrupted(jars, signals)
		}
		if !staged {
			return d.fail(ExitCopy, msgCopyFailed, jar.name, stationSite.unchanged())
		}
	}
	for _, jar := range jars {
		if signals.caught() {
			return d.copyInterrupted(jars, signals)
		}
		if !jar.had {
			continue
		}
		if err := moveAside(jar.target, jar.aside); err != nil {
			return d.copyFailed(jars, jar, err)
		}
		jar.moved = true
	}
	for _, jar := range jars {
		if signals.caught() {
			return d.copyInterrupted(jars, signals)
		}
		if err := rename(jar.staged, jar.target); err != nil {
			return d.copyFailed(jars, jar, err)
		}
		jar.placed = true
	}
	if signals.caught() {
		return d.copyInterrupted(jars, signals)
	}
	for _, jar := range jars {
		if !jar.moved {
			continue
		}
		if err := remove(jar.aside); err != nil {
			d.warn(warnAsideLeft, jar.name+asideSuffix, strerror.Text(err))
		}
	}
	d.copied = true
	d.say(msgCopyOK)
	return proceed
}

// stage copies a built jar to its staged name, over whatever a killed run left there, and gives it the
// mode of the jar it replaces (cp wrote into that jar, so the mode stayed). cp says why when it fails.
func (d *deploy) stage(jar *stagedJar, built string) bool {
	if err := os.RemoveAll(jar.staged); err != nil {
		fmt.Fprintf(d.s.Err, msgCannotRun+"\n", jar.staged, strerror.Text(err))
		return false
	}
	if d.show("cp", built, jar.staged) != 0 {
		return false
	}
	if st, err := os.Stat(jar.target); err == nil {
		os.Chmod(jar.staged, st.Mode().Perm()) // best effort: a file system without modes keeps its own
	}
	return true
}
