package register

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/envfile"
)

// shellQuote is s single-quoted for bash.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// renderLauncher is the station launcher: it exports the env file's assignments and execs the
// stable binary with the server flags, one option and its values per line. It holds no credential.
func renderLauncher(station, envPath, bin string, serverArgs []string) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	fmt.Fprintf(&b, "# mcp-n4 launcher for station %s, written by 'mcp-n4 register'.\n", station)
	b.WriteString("# Credentials come only from the env file sourced below (keep it chmod 600); this file holds none.\n")
	fmt.Fprintf(&b, "set -a; . %s; set +a\n", shellQuote(envPath))
	b.WriteString("exec " + shellQuote(bin))
	for i, a := range serverArgs {
		if i == 0 || strings.HasPrefix(a, "-") {
			b.WriteString(" \\\n  ")
		} else {
			b.WriteString(" ")
		}
		b.WriteString(shellQuote(a))
	}
	b.WriteString("\n")
	return b.String()
}

// secretInArgs names the env file keys whose value a server flag carries: a value of 4 bytes or
// more anywhere in a flag, or a shorter non-empty value as a whole flag. Values are never returned.
func secretInArgs(args []string, env []envfile.Assignment) []string {
	var keys []string
	for _, as := range env {
		for _, a := range args {
			if as.Value != "" && (a == as.Value || (len(as.Value) >= 4 && strings.Contains(a, as.Value))) {
				keys = append(keys, as.Key)
				break
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// firstDiffLine is the 1-based line where two texts first differ.
func firstDiffLine(a, b string) int {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return i + 1
		}
	}
	if len(la) < len(lb) {
		return len(la) + 1
	}
	return len(lb) + 1
}

// gitTreeOf is the .git (directory or worktree file) at or above the directory of path, after
// resolving symlinks when path exists; "" when there is none. A launcher must not exec a binary
// inside a git checkout: removing or rebuilding the checkout breaks every registered server.
func gitTreeOf(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return filepath.Join(dir, ".git")
		}
		if filepath.Dir(dir) == dir {
			return ""
		}
	}
}
