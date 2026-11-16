package ngdeploy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// notARepository is what git says, in the C locale, when no repository holds the working directory.
const notARepository = "not a git repository"

// outsideGit reports that the working directory is not in a git repository, the one case whose deploy has
// no commit to record and still ends 0 (record.go). git rev-parse --git-dir must fail saying so (in the C
// locale, so a translation does not hide it), and no .git may be there or in any directory above: a
// repository git refuses to open (dubious ownership, a corrupt .git) is not "outside git" and stays exit 41.
// Without git the record step cannot tell either, and stays exit 41.
func (d *deploy) outsideGit() bool {
	git, err := exec.LookPath("git")
	if err != nil {
		return false
	}
	var said bytes.Buffer
	cmd := exec.Command(git, "rev-parse", "--git-dir")
	cmd.Env, cmd.Stderr = append(os.Environ(), "LC_ALL=C", "LANGUAGE="), &said
	if cmd.Run() == nil || !strings.Contains(said.String(), notARepository) {
		return false
	}
	return !gitEntryAbove()
}

// gitEntryAbove reports whether the working directory or a directory above it holds a .git entry.
func gitEntryAbove() bool {
	dir, err := os.Getwd()
	if err != nil {
		return true // cannot tell: not "outside git"
	}
	for {
		if present(filepath.Join(dir, ".git")) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}
