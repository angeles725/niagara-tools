package ngdeploy

import "github.com/angeles725/niagara-tools/go/internal/strerror"

// probeStation tests, before the build, that every station jar the run will replace can be moved: each is
// moved to <jar>.ng-probe with moveAside (which never replaces a file already there) and straight back. On a
// Windows mount a jar a running station or Workbench holds open refuses the move; that is exit 40 with the
// line naming a running station, before gradle runs, instead of after a minute of building. A jar with no
// write permission still passes: a rename needs the directory's, not the file's. A jar that cannot be moved
// back is exit 40 naming both paths. --no-deploy replaces nothing in the station and probes nothing.
func (d *deploy) probeStation() int {
	if d.isOn(optNoDeploy) {
		return proceed
	}
	for _, profile := range profilesOf[d.raw[optMode]] {
		name := d.jarName(profile)
		jar := d.station + "/" + name
		if is, err := regular(jar); err != nil || !is {
			continue // nothing to replace, or something the copy refuses with its own line
		}
		probe := jar + probeSuffix
		if err := moveAside(jar, probe); err != nil {
			return d.failHeld(err, ExitCopy, msgProbeRefused, name, d.station, strerror.Text(err))
		}
		if err := rename(probe, jar); err != nil {
			return d.fail(ExitCopy, msgProbeStuck, jar, probe, strerror.Text(err), probe, jar)
		}
	}
	return proceed
}
