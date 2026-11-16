package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// rename moves a jar of the copy; a test replaces it.
var rename = os.Rename

// stagedJar is one jar of the mode on its way into the station.
type stagedJar struct {
	name   string // <MODULE>-<profile>.jar
	target string // the jar in the station modules directory
	staged string // the new jar beside it, until it is renamed in
	aside  string // the station's jar beside it, from the move aside until the copy is done or undone
	had    bool   // the station held something of that name before the copy
	moved  bool   // it was moved aside
	placed bool   // the new jar was renamed in
}

// copyJars puts the built jars of the mode into the station modules directory, all or none: it runs only
// after every precheck, verify and the gate passed. Each jar is copied with cp to a staged name beside its
// target; then the station's jars are moved aside, the staged ones renamed in, and the ones aside removed.
// Every step after the cp is a rename within one directory, so a jar is replaced whole or not at all, and
// a step that fails is undone from the jars moved aside (restore.go), with or without a backup. Every
// failure is exit 40, with a line that says what the station holds; no staged file outlives the run, and a
// jar stays aside only when it could not be moved back.
func (d *deploy) copyJars() int {
	var jars []*stagedJar
	for _, profile := range profilesOf[d.raw[optMode]] {
		jars = append(jars, d.newStagedJar(profile))
	}
	if code := d.clearStale(jars); code != proceed {
		return code
	}
	defer func() {
		for _, jar := range jars {
			os.RemoveAll(jar.staged)
		}
	}()
	for i, jar := range jars {
		if !d.stage(jar, d.builtJars()[i]) {
			return d.fail(ExitCopy, msgCopyFailed, jar.name, outcomeUnchanged)
		}
	}
	for _, jar := range jars {
		if !jar.had {
			continue
		}
		if err := rename(jar.target, jar.aside); err != nil {
			return d.copyFailed(jars, jar, err, outcomeUnchanged)
		}
		jar.moved = true
	}
	for i, jar := range jars {
		if err := rename(jar.staged, jar.target); err != nil {
			outcome := outcomeUnchanged
			if i > 0 {
				outcome = outcomeUndone
			}
			return d.copyFailed(jars, jar, err, outcome)
		}
		jar.placed = true
	}
	for _, jar := range jars {
		if err := os.RemoveAll(jar.aside); err != nil {
			d.warn(warnAsideLeft, jar.name+asideSuffix, strerror.Text(err))
		}
	}
	d.say(msgCopyOK)
	return proceed
}

// newStagedJar describes a jar of the mode and whether the station holds one under its name.
func (d *deploy) newStagedJar(profile string) *stagedJar {
	jar := &stagedJar{name: d.jarName(profile), target: d.station + "/" + d.jarName(profile)}
	jar.staged, jar.aside = jar.target+stagedSuffix, jar.target+asideSuffix
	_, err := os.Lstat(jar.target)
	jar.had = err == nil
	return jar
}

// clearStale deals with what a killed deploy left aside, before anything is copied. Beside its jar such a
// file is a leftover and is removed. Without its jar it may be the only copy of the previous one: the
// deploy stops (exit 40) and the file is left for the operator.
func (d *deploy) clearStale(jars []*stagedJar) int {
	for _, jar := range jars {
		if _, err := os.Lstat(jar.aside); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if !jar.had {
			return d.fail(ExitCopy, msgCopyFailed, jar.name, fmt.Sprintf(outcomeStale, jar.name+asideSuffix, jar.name, jar.name))
		}
		if err := os.RemoveAll(jar.aside); err != nil {
			return d.fail(ExitCopy, msgCopyFailed, jar.name+asideSuffix+": "+strerror.Text(err), outcomeUnchanged)
		}
	}
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
	if st, err := os.Stat(jar.target); err == nil && st.Mode().IsRegular() {
		os.Chmod(jar.staged, st.Mode().Perm()) // best effort: a file system without modes keeps its own
	}
	return true
}
