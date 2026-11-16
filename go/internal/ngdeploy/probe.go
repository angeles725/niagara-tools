package ngdeploy

import (
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// probeRegular is regular for the probe; a test swaps it to send a signal while the probe looks at a jar.
var probeRegular = regular

// probeStation tests, before the build, that every station jar the run will replace can be moved: each is
// moved to <jar>.ng-probe with moveAside (which never replaces a file already there) and straight back. On a
// Windows mount a jar a running station or Workbench holds open refuses the move; that is exit 40 with the
// line naming a running station (failHeld), before gradle runs, instead of after a minute of building. A jar
// with no write permission still passes: a rename needs the directory's, not the file's. A jar that cannot
// be moved back is exit 40 naming both paths. --no-deploy replaces nothing in the station and probes nothing.
//
// SIGINT, SIGTERM and SIGHUP are held across the two moves: a signal that arrives between them waits until
// the jar is back under its name, then ends the run with 128 + the signal; one that arrives while the probe
// looks at a jar it does not move ends it too (heldSignal runs after every jar). A probe file a killed run
// left is handled first (staleProbe).
func (d *deploy) probeStation() int {
	if d.isOn(optNoDeploy) {
		return proceed
	}
	held := make(chan os.Signal, 1)
	signal.Notify(held, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(held)
	for _, profile := range profilesOf[d.raw[optMode]] {
		if code := d.probeJar(d.jarName(profile)); code != proceed {
			return code
		}
		if code := d.heldSignal(held); code != proceed {
			return code
		}
	}
	return proceed
}

// probeJar moves one station jar to its probe name and straight back.
func (d *deploy) probeJar(name string) int {
	jar := d.station + "/" + name
	probe := jar + probeSuffix
	if code := d.staleProbe(name, jar, probe); code != proceed {
		return code
	}
	if is, err := probeRegular(jar); err != nil || !is {
		return proceed // nothing to replace, or something the copy refuses with its own line
	}
	if err := moveAside(jar, probe); err != nil {
		return d.failHeld(err, d.station, ExitCopy, msgProbeRefused, name, d.station, strerror.Text(err))
	}
	if err := rename(probe, jar); err != nil {
		return d.fail(ExitCopy, msgProbeStuck, jar, probe, strerror.Text(err), probe, jar)
	}
	return proceed
}

// heldSignal ends the run with 128 + a signal the probe held, after every jar, whether it was moved or not.
func (d *deploy) heldSignal(held <-chan os.Signal) int {
	select {
	case sig := <-held:
		s := sig.(syscall.Signal)
		return d.fail(signalBase+int(s), msgInterrupted, signalNames[s])
	default:
		return proceed
	}
}

// staleProbe looks at a <jar>.ng-probe a killed run left. With no jar beside it, it is the jar: it is moved
// back under its name and a NOTICE says so. With the jar beside it the run cannot tell which one the station
// should load: exit 40 with the manual fix, nothing changed.
// A probe file or jar that cannot be examined is left to the checks after it, which report it.
func (d *deploy) staleProbe(name, jar, probe string) int {
	if _, err := os.Lstat(probe); err != nil {
		return proceed
	}
	_, err := os.Lstat(jar)
	switch {
	case err == nil:
		d.fail(ExitCopy, msgProbeStale, probe, name)
		d.warn(indent+msgProbeStaleFix, jar, probe, probe, jar)
		return ExitCopy
	case !errors.Is(err, fs.ErrNotExist):
		return proceed
	}
	if err := rename(probe, jar); err != nil {
		return d.fail(ExitCopy, msgProbeStuck, jar, probe, strerror.Text(err), probe, jar)
	}
	d.warn(noticeProbeRestored, probe, jar)
	return proceed
}
