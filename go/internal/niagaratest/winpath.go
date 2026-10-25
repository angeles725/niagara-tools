package niagaratest

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// drivePath is a Windows drive path, X:\rest.
var drivePath = regexp.MustCompile(`^([A-Za-z]):\\(.*)$`)

// unsafeChars are refused in every path the run hands to cmd.exe or PowerShell: cmd metacharacters, the
// PowerShell single quote, and control characters. With them gone no command line needs escaping.
const unsafeChars = "\"%!&^|<>'`$\r\n\t\x00"

// toWSL maps X:\a\b, or a WSL path already under the drive mount (<mnt>/x/a/b), to the WSL path.
func (h host) toWSL(p string) (string, bool) {
	if m := drivePath.FindStringSubmatch(p); m != nil {
		return filepath.Join(h.mnt, strings.ToLower(m[1]), strings.ReplaceAll(m[2], `\`, "/")), true
	}
	if _, ok := h.toWin(p); ok {
		return filepath.Clean(p), true
	}
	return "", false
}

// toWin maps <mnt>/x/a/b to X:\a\b; ok is false for a path off the drive mount.
func (h host) toWin(p string) (string, bool) {
	rel, err := filepath.Rel(h.mnt, filepath.Clean(p))
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	drive, rest, _ := strings.Cut(rel, "/")
	if len(drive) != 1 || !('a' <= drive[0] && drive[0] <= 'z') {
		return "", false
	}
	return strings.ToUpper(drive) + `:\` + strings.ReplaceAll(rest, "/", `\`), true
}

// winDir is a directory on a Windows drive, by its symlink-resolved WSL path and its Windows path.
type winDir struct{ wsl, win string }

// canonical resolves a flag's path (X:\… or <mnt>/x/…) to an existing directory on a Windows drive.
func (h host) canonical(flag, p string) (winDir, error) {
	if i := strings.IndexAny(p, unsafeChars); i >= 0 {
		return winDir{}, fmt.Errorf("%s %q: unsafe character %q", flag, p, p[i])
	}
	w, ok := h.toWSL(p)
	if !ok {
		return winDir{}, fmt.Errorf("%s %s: not on a Windows drive (X:\\… or %s/x/…)", flag, p, h.mnt)
	}
	r, err := filepath.EvalSymlinks(w)
	if err != nil {
		return winDir{}, fmt.Errorf("%s %s: %v", flag, p, err)
	}
	win, ok := h.toWin(r)
	if !ok {
		return winDir{}, fmt.Errorf("%s %s: resolves off the Windows drives (%s)", flag, p, r)
	}
	return winDir{r, win}, nil
}

// within says whether the Windows path child is parent or below it, case-insensitively as Windows compares.
func within(child, parent string) bool {
	c, p := strings.ToLower(child), strings.TrimSuffix(strings.ToLower(parent), `\`)
	return c == p || strings.HasPrefix(c, p+`\`)
}

// underUsers says whether a Windows path is under X:\Users, where every real user home lives.
func underUsers(win string) bool {
	_, rest, _ := strings.Cut(win, `\`)
	top, _, _ := strings.Cut(rest, `\`)
	return strings.EqualFold(top, "Users")
}
