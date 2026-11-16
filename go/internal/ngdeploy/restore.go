package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// copyFailed undoes the copy after a move aside or a rename in failed for one jar; it is the exit 40 line.
func (d *deploy) copyFailed(jars []*stagedJar, failed *stagedJar, why error) int {
	return d.fail(ExitCopy, msgCopyFailed, failed.name+": "+strerror.Text(why), undo(jars))
}

// copyInterrupted undoes the copy after a SIGINT or SIGTERM; the run ends with 128 + the signal.
func (d *deploy) copyInterrupted(jars []*stagedJar, signals *interrupts) int {
	return d.fail(signals.exitCode(), msgCopyInterrupted, signals.name(), undo(jars))
}

// undo puts the station back as the copy found it: every jar moved aside is moved back over whatever took
// its place, and a new jar the station did not have is removed. It returns what the line says after the
// reason: nothing was replaced, or what was replaced was put back, or, when a jar could not be moved back
// or removed, the mixed state: jar by jar, which is the new one, which is missing and which is the
// previous one, with each file aside that still holds previous bytes (it is left where it is).
func undo(jars []*stagedJar) (outcome string) {
	outcome = outcomeUnchanged
	var states []string
	stuck, aside := false, false
	for _, jar := range jars {
		if jar.placed {
			outcome = outcomeUndone
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
				states, stuck = append(states, fmt.Sprintf(stateExtra, jar.name, strerror.Text(err))), true
			}
		case jar.had:
			states = append(states, fmt.Sprintf(statePrevious, jar.name))
		}
	}
	switch {
	case !stuck:
		return outcome
	case aside:
		return fmt.Sprintf(outcomeMixed+mixedPutBack, strings.Join(states, ", "), asideSuffix)
	}
	return fmt.Sprintf(outcomeMixed, strings.Join(states, ", "))
}
