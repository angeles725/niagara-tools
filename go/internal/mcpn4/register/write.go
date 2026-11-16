package register

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
)

// writeFile replaces path atomically. A symlink (a config kept in a dotfiles repository, say)
// stays a link: the file it resolves to is replaced, by a temporary file in that file's own
// directory. perm 0 keeps the existing mode (0644 for a new file).
func writeFile(path string, data []byte, perm os.FileMode) error {
	if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("%s is a symlink that does not resolve: %w", path, err)
		}
		path = real
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := atomicfile.Write(path, data); err != nil {
		return err
	}
	if perm != 0 {
		return os.Chmod(path, perm)
	}
	return nil
}

// writeLauncher writes the launcher text unless it is nil or already there, and returns the line to
// report and an undo that puts back the launcher's previous bytes and mode (or removes a new one),
// for when the entry write that follows fails.
func writeLauncher(label, path string, text []byte) (line string, undo func() error, err error) {
	noop := func() error { return nil }
	old, rerr := os.ReadFile(path)
	if text == nil || (rerr == nil && string(old) == string(text)) {
		return "unchanged " + label, noop, nil
	}
	undo = func() error { return os.Remove(path) }
	if rerr == nil {
		st, err := os.Stat(path)
		if err != nil {
			return "", nil, err
		}
		undo = func() error { return writeFile(path, old, st.Mode().Perm()) }
	} else if !errors.Is(rerr, fs.ErrNotExist) {
		return "", nil, rerr
	}
	if err := writeFile(path, text, 0o755); err != nil {
		return "", nil, err
	}
	return "written " + label, undo, nil
}

// checkKeptLauncher refuses, unless force, to keep a launcher that does not exec the installed
// binary bin: today's hand-written launchers run a worktree binary (the H5 failure), and a
// launcher whose exec line cannot be read cannot be shown not to.
func checkKeptLauncher(launcher, text, bin string, force bool) error {
	if force {
		return nil
	}
	target, ok := launcherWord(text, "exec ")
	switch {
	case !ok:
		return fmt.Errorf("launcher %s has no exec line naming its binary; pass server flags after -- to rewrite it, or --force to keep it", launcher)
	case filepath.Clean(target) == filepath.Clean(bin):
		return binInCheckout(bin, false) // bin may be a symlink into a checkout
	}
	where := ""
	if git := gitTreeOf(target); git != "" {
		where = fmt.Sprintf(", inside a git checkout (%s)", git)
	}
	return fmt.Errorf("launcher %s execs %s%s, not the installed binary %s; pass server flags after -- to rewrite it, or --force to keep it",
		launcher, target, where, bin)
}
