// Package gen is the check-or-write flow every kit generator shares (gen-retro-index,
// gen-lint-index, gen-build-state): the `[--check] [<kit-root>]` command line, the kit
// sub-directory guard, and writing a generated file or comparing it with the committed one.
// Each generator only renders its content; the messages and exit codes live here once.
package gen

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes shared by the generators (the gen-retro-index contract).
const (
	ExitOK    = 0 // file written, or fresh under --check
	ExitStale = 1 // stale or missing file under --check, or an input violation
	ExitEnv   = 3 // usage or environment error; nothing written
)

// ParseArgs reads `[--check] [<kit-root>]` for tool. With no root argument (an empty one counts as
// none, as the bash tools did) the root is the rootEnv variable the toolbelt wrapper exports. On a
// usage error or no root it prints why to s.Err and returns ok false: exit ExitEnv.
func ParseArgs(tool, rootEnv string, args []string, s cli.Stdio) (check bool, root string, ok bool) {
	for _, a := range args {
		switch {
		case a == "--check":
			check = true
		case strings.HasPrefix(a, "-") || root != "":
			fmt.Fprintf(s.Err, "usage: %s.sh [--check] [<kit-root>]\n", tool)
			return false, "", false
		default:
			root = a
		}
	}
	if root == "" {
		if root = s.Env(rootEnv); root == "" {
			fmt.Fprintf(s.Err, "%s: no <kit-root> given and %s is unset (run toolbelt/%s.sh)\n", tool, rootEnv, tool)
			return false, "", false
		}
	}
	return check, root, true
}

// SubDir is root/sub when it is a directory; otherwise it prints "no <sub>/ under <root>" to s.Err
// and returns ok false: exit ExitEnv.
func SubDir(tool, root, sub string, s cli.Stdio) (string, bool) {
	dir := root + "/" + sub
	if !IsDir(dir) {
		fmt.Fprintf(s.Err, "%s: no %s/ under %s\n", tool, sub, root)
		return "", false
	}
	return dir, true
}

// IsRegular reports whether path is a regular file (symlinks followed).
func IsRegular(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

// IsDir reports whether path is a directory (symlinks followed).
func IsDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// File is one generated file: what a generator rendered and how it is compared and written.
type File struct {
	Tool    string // command name; prefixes every message and names the <tool>.sh to rerun
	Path    string
	Content string // the exact bytes a write stores
	// TrimNewlines compares ignoring trailing newlines on both sides, as the bash $(...) comparison
	// did, and reports a target that is not a regular file as missing (the INDEX.md generators).
	TrimNewlines bool
	StaleNote    string // appended to "is stale", e.g. " vs ledger.d/"
	// Atomic writes through a temporary file in the target's directory, keeping the target's mode,
	// so a failed write never leaves a truncated file behind.
	Atomic bool
}

// Run compares f with the committed file under check, or writes it, and returns the exit code.
func (f File) Run(check bool, s cli.Stdio) int {
	if check {
		return f.check(s)
	}
	return f.write(s)
}

func (f File) check(s cli.Stdio) int {
	if f.TrimNewlines && !IsRegular(f.Path) {
		fmt.Fprintf(s.Err, "%s: %s is missing; run %s.sh\n", f.Tool, f.Path, f.Tool)
		return ExitStale
	}
	committed, err := os.ReadFile(f.Path)
	// Mutation: GRI-stale, GLI-stale, GBS-stale -- dropping the generated-vs-committed compare makes --check vacuously clean
	if err != nil || !f.same(string(committed)) {
		fmt.Fprintf(s.Err, "%s: %s is stale%s; run %s.sh and commit the result\n", f.Tool, f.Path, f.StaleNote, f.Tool)
		ShowDiff(f.Content, f.Path, s.Err)
		return ExitStale
	}
	fmt.Fprintf(s.Out, "%s: %s is fresh\n", f.Tool, f.Path)
	return ExitOK
}

func (f File) same(committed string) bool {
	if f.TrimNewlines {
		return strings.TrimRight(committed, "\n") == strings.TrimSuffix(f.Content, "\n")
	}
	return committed == f.Content
}

func (f File) write(s cli.Stdio) int {
	var err error
	if f.Atomic {
		err = writeAtomic(f.Path, f.Content)
	} else {
		err = os.WriteFile(f.Path, []byte(f.Content), 0o666)
	}
	if err != nil {
		fmt.Fprintf(s.Err, "%s: cannot write %s: %v\n", f.Tool, f.Path, err)
		return ExitEnv
	}
	fmt.Fprintf(s.Out, "%s: wrote %s\n", f.Tool, f.Path)
	return ExitOK
}

// writeAtomic replaces path through a temporary file in its directory, so a failed write never
// leaves a truncated file behind.
func writeAtomic(path, content string) error {
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmp, []byte(content), mode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ShowDiff writes `diff - <path>` (generated vs committed, POSIX normal format) to w; it is
// diagnostic only, so a missing diff binary prints nothing.
func ShowDiff(generated, path string, w io.Writer) {
	cmd := exec.Command("diff", "-", path)
	cmd.Stdin = strings.NewReader(generated)
	cmd.Stdout, cmd.Stderr = w, w
	_ = cmd.Run()
}
