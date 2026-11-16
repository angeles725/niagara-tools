package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// copyFailed undoes the copy after a move aside or a rename in failed for one jar, and is the exit 40 line:
// the jar and the reason, then undone (what to say when the station is as it was) or, when a jar could not
// be moved back, what each jar of the station is now.
func (d *deploy) copyFailed(jars []*stagedJar, failed *stagedJar, why error, undone string) int {
	outcome := undone
	if mixed := undo(jars); mixed != "" {
		outcome = mixed
	}
	return d.fail(ExitCopy, msgCopyFailed, failed.name+": "+strerror.Text(why), outcome)
}

// undo puts the station back as the copy found it: every jar moved aside is moved back over whatever took
// its place, and a new jar the station did not have is removed. It returns "" when that worked. When a jar
// could not be moved back the station holds a mixed pair: the text says, jar by jar, which is the new one,
// which is missing and which is the previous one, and names each file aside that still holds previous
// bytes; that file is left where it is.
func undo(jars []*stagedJar) (mixed string) {
	var states []string
	stuck := false
	for _, jar := range jars {
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
			states, stuck = append(states, fmt.Sprintf(state, jar.name, jar.name+asideSuffix, strerror.Text(err))), true
		case jar.placed:
			if err := os.Remove(jar.target); err != nil && !errors.Is(err, fs.ErrNotExist) {
				states, stuck = append(states, fmt.Sprintf(stateExtra, jar.name, strerror.Text(err))), true
			}
		case jar.had:
			states = append(states, fmt.Sprintf(statePrevious, jar.name))
		}
	}
	if !stuck {
		return ""
	}
	return fmt.Sprintf(outcomeMixed, strings.Join(states, ", "), asideSuffix)
}
