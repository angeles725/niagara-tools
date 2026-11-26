package previewshot

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const installHint = "npx playwright install chromium-headless-shell"

var revRe = regexp.MustCompile(`-([0-9]+)$`)

// shotTimeout is the real time a browser run may take on top of its virtual-time budget.
var shotTimeout = 2 * time.Minute

func isExec(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0
}

func homeDir(s cli.Stdio) string {
	if h := s.Env("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

// onPath finds name in the PATH the command was given (not the process one, so tests control it).
func onPath(s cli.Stdio, name string) string {
	for _, d := range strings.Split(s.Env("PATH"), string(os.PathListSeparator)) {
		if d != "" && isExec(filepath.Join(d, name)) {
			return filepath.Join(d, name)
		}
	}
	return ""
}

// findBrowser: PREVIEW_SHELL, else the newest Playwright chromium_headless_shell (both layouts), else
// chrome-headless-shell on PATH. Nothing found is exit 3 with the install command.
func findBrowser(s cli.Stdio) (string, error) {
	if p := s.Env("PREVIEW_SHELL"); p != "" {
		if !isExec(p) {
			return "", errf(exitUsage, "PREVIEW_SHELL=%s is not an executable file", p)
		}
		return p, nil
	}
	root := s.Env("PLAYWRIGHT_BROWSERS_PATH")
	if root == "" {
		root = filepath.Join(homeDir(s), ".cache", "ms-playwright")
	}
	dirs, _ := filepath.Glob(filepath.Join(root, "chromium_headless_shell-*"))
	rev := func(d string) int {
		m := revRe.FindStringSubmatch(d)
		if m == nil {
			return -1
		}
		n, _ := strconv.Atoi(m[1])
		return n
	}
	sort.Slice(dirs, func(i, j int) bool { return rev(dirs[i]) > rev(dirs[j]) })
	for _, d := range dirs {
		for _, pat := range []string{"chrome-headless-shell-linux*/chrome-headless-shell", "chrome-linux*/headless_shell"} {
			hits, _ := filepath.Glob(filepath.Join(d, pat))
			sort.Strings(hits)
			for _, h := range hits {
				if isExec(h) {
					return h, nil
				}
			}
		}
	}
	if p := onPath(s, "chrome-headless-shell"); p != "" {
		return p, nil
	}
	return "", errf(exitUsage, "no headless browser: set PREVIEW_SHELL, or install one with: %s "+
		"(looked in %s and for chrome-headless-shell on PATH)", installHint, root)
}

// defaultGL is the software WebGL the shot needs on a host with no GPU. Measured on this host, Chrome for Testing
// 151 to 159 (Playwright chromium_headless_shell 1234 to 1255): `--use-gl=swiftshader` alone gives
// getContext('webgl') === null (a black canvas, GL errors on stderr), while the ANGLE pair gives WebGL 1.0.
// Chrome 159 also warns that falling back to software WebGL on its own is deprecated and asks for
// --enable-unsafe-swiftshader (acceptable here: the page is a local preview of the user's own module).
const defaultGL = "--use-gl=angle --use-angle=swiftshader --enable-unsafe-swiftshader"

// glFlags is PREVIEW_SHOT_GL when set (a build that wants another GL backend), else defaultGL.
func glFlags(s cli.Stdio) []string {
	if v := strings.Fields(s.Env("PREVIEW_SHOT_GL")); len(v) > 0 {
		return v
	}
	return strings.Fields(defaultGL)
}

// takeShot runs the browser and returns the size of the screenshot; a missing or empty file is an error.
func takeShot(ctx context.Context, s cli.Stdio, shell, target string, o options) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(o.out), 0o755); err != nil {
		return 0, err
	}
	// A file left by an earlier run must never pass as this run's shot.
	if err := os.Remove(o.out); err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(o.budget)*time.Millisecond+shotTimeout)
	defer cancel()
	argv := append(glFlags(s), "--virtual-time-budget="+strconv.Itoa(o.budget),
		"--window-size="+strconv.Itoa(o.w)+","+strconv.Itoa(o.h), "--screenshot="+o.out, target)
	cmd := exec.CommandContext(ctx, shell, argv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, errf(exitFail, "%s failed: %v\n%s", filepath.Base(shell), err, tail(string(out)))
	}
	st, err := os.Stat(o.out)
	if err != nil || st.Size() == 0 {
		return 0, errf(exitFail, "%s wrote no screenshot to %s\n%s", filepath.Base(shell), o.out, tail(string(out)))
	}
	return st.Size(), nil
}

// tail is the last lines of a process's output, enough to see why it failed.
func tail(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	return strings.Join(lines, "\n")
}
