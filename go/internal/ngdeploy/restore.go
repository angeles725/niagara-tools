package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// copyFailed undoes the copy after a move aside or a rename in failed for one jar; it is the exit 40 line,
// and the line naming a running station when the file system refused as for a jar held open.
func (d *deploy) copyFailed(jars []*stagedJar, failed *stagedJar, why error) int {
	return d.failHeld(why, ExitCopy, msgCopyFailed, failed.name+": "+strerror.Text(why), undo(jars, stationSite))
}

// copyInterrupted undoes the copy after a SIGINT or SIGTERM; the run ends with 128 + the signal.
func (d *deploy) copyInterrupted(jars []*stagedJar, signals *interrupts) int {
	return d.fail(signals.exitCode(), msgCopyInterrupted, signals.name(), undo(jars, stationSite))
}

// unchanged and undone are the two outcomes that leave a site as it was.
func (at site) unchanged() string { return fmt.Sprintf(outcomeUnchanged, at.holder, at.since) }
func (at site) undone() string    { return fmt.Sprintf(outcomeUndone, at.holder, at.since) }

// undo puts a site back as the replacement found it (putBack) and returns what the line says after the
// reason: nothing was replaced, or what was replaced was put back, or the mixed state.
func undo(jars []*stagedJar, at site) (outcome string) {
	switch replaced, mixed := putBack(jars, at); {
	case mixed != "":
		return mixed
	case replaced:
		return at.undone()
	}
	return at.unchanged()
}

// putBack moves every jar that was moved aside back over whatever took its place, and removes a new jar the
// site did not have. replaced reports that a new jar had gone in. mixed is "" when the site holds again what
// it held; when a jar could not be moved back or removed, it is the mixed state: jar by jar, which is the new
// one, which is missing and which is the previous one, with each file aside that still holds previous bytes
// (it is left where it is).
func putBack(jars []*stagedJar, at site) (replaced bool, mixed string) {
	var states []string
	stuck, aside := false, false
	for _, jar := range jars {
		if jar.placed {
			replaced = true
		}
		switch {
		case jar.moved:
			err := rename(jar.aside, jar.target)
			if err == nil {
				states = append(states, fmt.Sprintf(statePrevious, jar.name))
				continue
			}
			state := stateMissing
			if jar.placed {
				state = stateNew
			}
			states = append(states, fmt.Sprintf(state, jar.name, jar.name+asideSuffix, strerror.Text(err)))
			stuck, aside = true, true
		case jar.placed:
			if err := remove(jar.target); err != nil && !errors.Is(err, fs.ErrNotExist) {
				states, stuck = append(states, fmt.Sprintf(stateExtra, jar.name, at.holder, strerror.Text(err), at.before)), true
			}
		case jar.had:
			states = append(states, fmt.Sprintf(statePrevious, jar.name))
		}
	}
	switch {
	case !stuck:
		return replaced, ""
	case aside:
		return replaced, fmt.Sprintf(outcomeMixed+mixedPutBack, at.holder, at.since, strings.Join(states, ", "), asideSuffix, at.before)
	}
	return replaced, fmt.Sprintf(outcomeMixed, at.holder, at.since, strings.Join(states, ", "))
}
