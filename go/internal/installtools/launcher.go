// Package installtools holds what the install and bootstrap commands under scripts/ share
// (install-skill, check-skill-drift, install-hooks, install-client-hooks, kit-resolve): where the
// invoked launcher lives and its header (launcher.go), file digests and comparison (digest.go), the
// `cp` and `chmod +x` they write with (copy.go), the file tests of a shell `[ -f ]` with an answer for
// "could not check" (filetest.go) and the git config read (gitconfig.go).
package installtools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// LauncherEnv names the variable a scripts/<tool>.sh launcher sets to its own absolute path. The
// repository the command works for is the one holding that script, as it was for the bash tool, and the
// script's header comment is the usage text.
const LauncherEnv = "INSTALL_TOOLS_LAUNCHER"

// Launcher is the scripts/<tool>.sh that started the command.
type Launcher struct{ Path string }

// LauncherFrom reads the launcher from the environment; the error says how to run the tool when the
// variable is unset or not an absolute path.
func LauncherFrom(s cli.Stdio) (Launcher, error) {
	path := s.Env(LauncherEnv)
	if !filepath.IsAbs(path) {
		return Launcher{}, fmt.Errorf("%s is not the absolute path of the invoked script (%q); run the scripts/ launcher", LauncherEnv, path)
	}
	return Launcher{Path: path}, nil
}

// Dir is the scripts directory holding the launcher.
func (l Launcher) Dir() string { return filepath.Dir(l.Path) }

// RepoRoot is the parent of the scripts directory, taken lexically as a shell `cd <dir>/.. && pwd` does.
func (l Launcher) RepoRoot() string { return filepath.Dir(l.Dir()) }

// Sibling is another script of the same scripts directory.
func (l Launcher) Sibling(name string) string { return filepath.Join(l.Dir(), name) }

// Header is the launcher's comment block from line 2 up to its first line that is not a comment, without
// the leading "#" and one space: the usage text, which has one home (the script).
func (l Launcher) Header() (string, error) {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	var header strings.Builder
	for _, line := range lines[min(1, len(lines)):] {
		if !strings.HasPrefix(line, "#") {
			break
		}
		header.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "#"), " ") + "\n")
	}
	return header.String(), nil
}
