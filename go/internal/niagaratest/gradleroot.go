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

	"github.com/angeles725/niagara-tools/go/internal/cli"
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

// checkGradleRoot is the gate at the end of cleanup: the root's status against the one taken before the copy,
// then the opt-in restore. It reports whether the root is unchanged.
func (h host) checkGradleRoot(ctx context.Context, p *plan, s cli.Stdio) bool {
	now, err := h.rootStatus(ctx, p.gradleRoot, p.rootTop)
	if err != nil {
		row(s, "FAIL", "gradle root unchanged: cannot re-read it: "+err.Error())
		return false
	}
	ch := p.root.changed(now)
	if len(ch) == 0 {
		row(s, "PASS", fmt.Sprintf("gradle root unchanged (%d tracked changes before, same after)", len(p.root)))
		return true
	}
	row(s, "FAIL", "gradle root changed by the build: "+listed(ch, maxRootListed))
	if p.restoreRoot {
		h.restoreGradleRoot(ctx, p, now, ch, s)
	}
	return false
}

// restoreGradleRoot checks out the changed paths that were clean before and are now modified or deleted in
// the worktree, nothing else, then re-reads the status to confirm them. Untracked files the build created are
// listed, never deleted.
func (h host) restoreGradleRoot(ctx context.Context, p *plan, now rootSnapshot, ch []string, s cli.Stdio) {
	var restore, untracked []string
	for _, path := range ch {
		if _, dirty := p.root[path]; dirty {
			continue
		}
		switch st := now[path].status; {
		case st == "??":
			untracked = append(untracked, path)
		case len(st) == 2 && st[0] == ' ' && strings.ContainsRune("MDT", rune(st[1])):
			restore = append(restore, path)
		}
	}
	left := ""
	if len(untracked) > 0 {
		left = " (untracked files the build created, left in place: " + listed(untracked, maxRootListed) + ")"
	}
	if len(restore) == 0 {
		row(s, "PASS", "gradle root restored: nothing to check out"+left)
		return
	}
	// --literal-pathspecs: a path is a file name, never pathspec magic such as :(glob) or a wildcard.
	args := append([]string{"--literal-pathspecs", "-C", p.gradleRoot, "checkout", "--"}, restore...)
	if code, err := h.run.Run(ctx, Cmd{Name: "git", Args: args}); err != nil || code != 0 {
		row(s, "FAIL", fmt.Sprintf("gradle root not restored: git checkout exit %d (%v)", code, err))
		return
	}
	after, err := h.rootStatus(ctx, p.gradleRoot, p.rootTop)
	if err != nil {
		row(s, "FAIL", "gradle root not restored: cannot re-read it: "+err.Error())
		return
	}
	var still []string
	for _, path := range restore {
		if _, ok := after[path]; ok {
			still = append(still, path)
		}
	}
	if len(still) > 0 {
		row(s, "FAIL", "gradle root not restored: "+listed(still, maxRootListed)+" still differs after git checkout")
		return
	}
	row(s, "PASS", "gradle root restored: "+listed(restore, maxRootListed)+left)
}
