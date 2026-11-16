package ngdeploy

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// rename puts a staged jar in place of its target; a test replaces it.
var rename = os.Rename

// stagedJar is one jar of the mode on its way into the station.
type stagedJar struct {
	name   string      // <MODULE>-<profile>.jar
	target string      // the jar in the station modules directory
	staged string      // the new jar beside it, until it is renamed
	had    bool        // the station held a regular file of that name before the copy
	mode   os.FileMode // its mode, kept by the new jar and by a jar put back
	size   int64       // its size: what a restore from the backup must give back
}

// copyJars puts the built jars of the mode into the station modules directory, all or none: it runs only
// after every precheck, verify and the gate passed. Each jar is copied with cp to a staged name beside its
// target (the same file system, so the rename that follows replaces the jar whole or not at all), then the
// staged jars are renamed in order. A jar that cannot be staged, and a first rename that fails, leave the
// station as it was; a later rename that fails puts the jar already renamed back (restore.go). Every
// failure is exit 40, with a line that says what the station holds; no staged file outlives the run.
func (d *deploy) copyJars() int {
	var jars []*stagedJar
	defer func() {
		for _, jar := range jars {
			os.RemoveAll(jar.staged)
		}
	}()
	for _, profile := range profilesOf[d.raw[optMode]] {
		jar := d.newStagedJar(profile)
		jars = append(jars, jar)
		if !d.stage(jar, d.builtJar(profile)) {
			return d.fail(ExitCopy, msgCopyFailed, jar.name, outcomeUnchanged)
		}
	}
	for i, jar := range jars {
		if err := rename(jar.staged, jar.target); err != nil {
			outcome := outcomeUnchanged
			if i > 0 { // a mode has two jars at most: one went in before this one
				outcome = d.putBack(jars[0], jar)
			}
			return d.fail(ExitCopy, msgCopyFailed, jar.name+": "+strerror.Text(err), outcome)
		}
	}
	d.say(msgCopyOK)
	return proceed
}

// newStagedJar describes a jar of the mode and what the station holds under its name.
func (d *deploy) newStagedJar(profile string) *stagedJar {
	jar := &stagedJar{name: d.jarName(profile), target: d.station + "/" + d.jarName(profile)}
	jar.staged = jar.target + stagedSuffix
	if st, err := os.Stat(jar.target); err == nil && st.Mode().IsRegular() {
		jar.had, jar.mode, jar.size = true, st.Mode().Perm(), st.Size()
	}
	return jar
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
	if jar.had {
		os.Chmod(jar.staged, jar.mode) // best effort: a file system without modes keeps its own
	}
	return true
}
