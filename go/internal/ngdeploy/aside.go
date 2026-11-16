package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// jarsToCopy describes the jars of the mode, after two looks at the station that change nothing in it and
// stop the deploy (exit 40) before anything is copied: a jar left aside by a deploy that did not finish
// (refuseAside), and something that is not a jar file under the name of a jar (notAJar).
func (d *deploy) jarsToCopy() (jars []*stagedJar, code int) {
	if code := d.refuseAside(); code != proceed {
		return nil, code
	}
	for _, profile := range profilesOf[d.raw[optMode]] {
		jar := &stagedJar{name: d.jarName(profile), target: d.station + "/" + d.jarName(profile)}
		jar.staged, jar.aside = jar.target+stagedSuffix, jar.target+asideSuffix
		if jar.had, code = d.jarFileAt(jar.name, jar.target); code != proceed {
			return nil, code
		}
		jars = append(jars, jar)
	}
	return jars, proceed
}

// jarFileAt reports whether the station holds a jar file under the name of a jar; anything else under that
// name (notAJar) is the exit 40 line of a copy that replaced nothing.
func (d *deploy) jarFileAt(name, target string) (exists bool, code int) {
	what, exists := notAJar(target)
	if what != "" {
		return exists, d.fail(ExitCopy, msgCopyFailed, name+": "+fmt.Sprintf(whyNotAJar, what), stationSite.unchanged())
	}
	return exists, proceed
}

// link makes a second name for a file; a test replaces it, as it does rename and remove (copy.go).
var link = os.Link

// moveAside moves a jar to its name aside and never replaces a file that is already there: the second name is
// made with link, which fails when it exists, and then the first is removed, so a run killed between the two
// leaves the jar under both names. On a file system that makes no links the name is looked for and the jar
// renamed, which is as safe while the run holds the lock of the directory (lock.go).
func moveAside(target, aside string) error {
	switch err := link(target, aside); {
	case err == nil:
		if err := remove(target); err != nil {
			remove(aside)
			return err
		}
		return nil
	case errors.Is(err, fs.ErrExist):
		return err
	}
	if present(aside) {
		return &os.LinkError{Op: "rename", Old: target, New: aside, Err: fs.ErrExist}
	}
	return rename(target, aside)
}

// refuseAside stops the deploy when the station holds a <jar>.ng-deploy-old file for any jar the tool
// deploys, of the mode or not. This run did not put it there, so it may be the only copy of the previous
// module, beside its jar (which may be the new one of a deploy that was killed) or without it: nothing is
// removed or renamed. The lines say what the station holds for each such jar, and the two ways out.
func (d *deploy) refuseAside() int {
	return d.refuseAsideIn(d.station, deployable, ExitCopy, msgAsideFound)
}

// refuseAsideIn is that refusal for the jars of some profiles in one modules directory: the exit code and the
// headline (which takes the directory) are the caller's, the state lines and the ways out are the same.
func (d *deploy) refuseAsideIn(dir string, profiles []string, code int, headline string) int {
	var found []string
	for _, profile := range profiles {
		name := d.jarName(profile)
		if !present(dir + "/" + name + asideSuffix) {
			continue
		}
		state := stateAsideNoJar
		if present(dir + "/" + name) {
			state = stateAsideWithJar
		}
		found = append(found, fmt.Sprintf(state, name, name+asideSuffix, asideSuffix))
	}
	if len(found) == 0 {
		return proceed
	}
	d.fail(code, headline, dir)
	for _, line := range found {
		d.warn(indent + line)
	}
	d.warn(indent+msgAsideWaysOut, asideSuffix, asideSuffix)
	return code
}

// present reports whether a path names anything at all, a link that leads nowhere included.
func present(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

// notAJar says what the station holds under the name of a jar when that is not a jar file: a directory, a
// link to nothing, anything but a regular file or a symbolic link to one. what is "" for a jar file
// (exists is true) and for a name not in use.
func notAJar(target string) (what string, exists bool) {
	if !present(target) {
		return "", false
	}
	st, err := os.Stat(target)
	switch {
	case err != nil && errors.Is(err, fs.ErrNotExist):
		return kindDangling, true
	case err != nil:
		return kindUnreadable, true
	case st.Mode().IsRegular():
		return "", true
	}
	prefix := ""
	if link, err := os.Lstat(target); err == nil && link.Mode()&fs.ModeSymlink != 0 {
		prefix = kindLinkTo
	}
	if st.IsDir() {
		return prefix + kindDirectory, true
	}
	return prefix + kindOther, true
}
