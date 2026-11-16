package niagaratest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The gradle-root gate: niagara gradle plugin 7.3.40 (4.13) makes :<part>:compileModuleTestJava append every
// @NiagaraType test class to the PRODUCTION <part>/module-include.xml, a tracked source file, so a run would
// silently edit the client's tree. The runner records `git status` of --gradle-root before the copy and after
// cleanup, and any difference is FAIL. --restore-gradle-root then checks out only the files that were clean
// before and that the build modified; it never touches a file already dirty and never deletes an untracked one.

// maxRootListed is how many changed paths a gradle root row names before "and N more".
const maxRootListed = 10

// rootEntry is one `git status` entry: its XY code (and the source of a rename) and its content hash, so a
// file already dirty before the run still shows a content change.
type rootEntry struct{ status, sum string }

// rootSnapshot is the gradle root's git state: every entry of `git status`, keyed by its path from the
// worktree top.
type rootSnapshot map[string]rootEntry

// gitStatusArgs is the snapshot's `git status`, untracked files listed one by one.
func gitStatusArgs(root string) []string {
	return []string{"-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all"}
}

// rootTop is the worktree top of --gradle-root (status paths are relative to it), or ok false when the root is
// no git worktree.
func (h host) rootTop(ctx context.Context, root string) (string, bool) {
	out := &capped{max: maxCaptured}
	code, err := h.run.Run(ctx, Cmd{Name: "git", Args: []string{"-C", root, "rev-parse", "--show-toplevel"}, Stdout: out})
	top := strings.TrimSpace(out.buf.String())
	if err != nil || code != 0 || top == "" {
		return "", false
	}
	return top, true
}

// rootStatus takes the snapshot: `git status` through the runner, then a sha256 of every listed file.
func (h host) rootStatus(ctx context.Context, root, top string) (rootSnapshot, error) {
	var out bytes.Buffer
	code, err := h.run.Run(ctx, Cmd{Name: "git", Args: gitStatusArgs(root), Stdout: &out})
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, fmt.Errorf("git status exit %d", code)
	}
	s := rootSnapshot{}
	fields := strings.Split(out.String(), "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f == "" {
			continue
		}
		if len(f) < 4 || f[2] != ' ' {
			return nil, fmt.Errorf("git status: malformed entry %q", f)
		}
		e, p := rootEntry{status: f[:2]}, f[3:]
		if strings.ContainsAny(f[:2], "RC") && i+1 < len(fields) {
			i++
			e.status += " from " + fields[i]
		}
		if e.sum, err = pathSum(filepath.Join(top, filepath.FromSlash(p))); err != nil {
			return nil, err
		}
		s[p] = e
	}
	return s, nil
}

// pathSum hashes a status path: a file's sha256, "missing" when it is gone, "dir" for a directory (a
// submodule or a nested repository), the target of a symlink.
func pathSum(p string) (string, error) {
	info, err := os.Lstat(p)
	switch {
	case os.IsNotExist(err):
		return "missing", nil
	case err != nil:
		return "", err
	case info.Mode()&os.ModeSymlink != 0:
		t, err := os.Readlink(p)
		return "symlink " + t, err
	case info.IsDir():
		return "dir", nil
	}
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// changed lists the paths whose entry differs between s and now, sorted: new, gone, or another code or hash.
func (s rootSnapshot) changed(now rootSnapshot) []string {
	var ps []string
	for p, e := range s {
		if w, ok := now[p]; !ok || w != e {
			ps = append(ps, p)
		}
	}
	for p := range now {
		if _, ok := s[p]; !ok {
			ps = append(ps, p)
		}
	}
	slices.Sort(ps)
	return ps
}
