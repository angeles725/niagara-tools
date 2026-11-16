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
		what, exists := notAJar(jar.target)
		if what != "" {
			return nil, d.fail(ExitCopy, msgCopyFailed, jar.name+": "+fmt.Sprintf(whyNotAJar, what), outcomeUnchanged)
		}
		jar.had = exists
		jars = append(jars, jar)
	}
	return jars, proceed
}

// refuseAside stops the deploy when the station holds a <jar>.ng-deploy-old file for any jar the tool
// deploys, of the mode or not. This run did not put it there, so it may be the only copy of the previous
// module, beside its jar (which may be the new one of a deploy that was killed) or without it: nothing is
// removed or renamed. The lines say what the station holds for each such jar, and the two ways out.
func (d *deploy) refuseAside() int {
	var found []string
	for _, profile := range deployable {
		name := d.jarName(profile)
		if !present(d.station + "/" + name + asideSuffix) {
			continue
		}
		state := stateAsideNoJar
		if present(d.station + "/" + name) {
			state = stateAsideWithJar
		}
		found = append(found, fmt.Sprintf(state, name, name+asideSuffix, asideSuffix))
	}
	if len(found) == 0 {
		return proceed
	}
	code := d.fail(ExitCopy, msgAsideFound, d.station)
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
