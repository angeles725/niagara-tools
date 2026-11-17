package mirrorhome

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// Main implements `mirror-niagara-home <source_niagara_home> <mirror_dir> [exclude-jar ...]`.
func Main(args []string, s cli.Stdio) int { return run(args, s, time.Now) }

// proceed is what a step returns when the run goes on to the next one.
const proceed = -1

// mirror is one run: the arguments as given (they are printed and linked as given) and what the steps
// learn on the way.
type mirror struct {
	s       cli.Stdio
	now     func() time.Time
	src     string   // <source_niagara_home>
	dir     string   // <mirror_dir>
	at      string   // where the mirror is written: dir, or dir as guard A judged it when it was not there
	exclude []string // jar names that get no link

	srcAbs, dirAbs string   // both resolved, for guard A and the marker
	exists         bool     // <mirror_dir> is there (through a symlink too) and has to be wiped
	entries, jars  []string // what will be linked: top-level names, and names under modules/
}

// run is Main with the clock the marker is dated by.
func run(args []string, s cli.Stdio, now func() time.Time) int {
	if len(args) < 2 {
		fmt.Fprint(s.Err, msgUsage)
		return ExitUsage
	}
	m := &mirror{s: s, now: now, src: args[0], dir: args[1], exclude: args[2:]}
	// Every refusal comes before wipe, the first step that removes anything.
	for _, step := range []func() int{m.checkSource, m.resolve, m.guardSource, m.guardWipe, m.plan, m.wipe, m.build} {
		if code := step(); code != proceed {
			return code
		}
	}
	return ExitOK
}

// stop prints a message on stderr and returns code.
func (m *mirror) stop(code int, format string, a ...any) int {
	fmt.Fprintf(m.s.Err, format, a...)
	return code
}

// fault reports a step the operating system refused: exit 1, as under the bash tool's set -e.
func (m *mirror) fault(err error) int {
	text := err.Error()
	var pe *fs.PathError
	var le *os.LinkError
	switch {
	case errors.As(err, &pe):
		text = pe.Op + " " + pe.Path + ": " + strerror.Text(pe.Err)
	case errors.As(err, &le):
		text = le.Op + " " + le.New + ": " + strerror.Text(le.Err)
	}
	return m.stop(ExitUsage, msgFault, text)
}
