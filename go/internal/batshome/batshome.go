// Package batshome is the lint on the bats suites of the Go launchers: a launcher run under a changed (or
// unset) HOME rebuilds its binary with a fresh Go build cache, so a suite that does that builds the binary
// once, in setup_file, with the real environment (retro go-toolbelt-small-port Δ4).
package batshome

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// launcherDirs are the directories (under the repository root) whose *.sh files can be Go launchers.
var launcherDirs = []string{"build-n4-module-kit/toolbelt", "scripts"}

var (
	// changesHome is a command that sets, unsets or removes HOME: an assignment or an export at a word
	// start (SKILL_HOME= is another variable), `unset HOME`, `env -u HOME`.
	changesHome = regexp.MustCompile(`(?:^|[\s;&|(])(?:HOME=|unset\s+HOME\b|-u\s+HOME\b)`)
	// quoted is a quoted string, dropped before changesHome reads a line: an assertion that spells
	// "HOME=" inside its text is no command. (A command inside bash -c '...' is not read: an accepted limit.)
	quoted    = regexp.MustCompile(`"[^"]*"|'[^']*'`)
	setupFile = regexp.MustCompile(`^setup_file\s*\(\)`)
	// heredoc opens a here-document: its lines are data (a fixture), not commands of the suite.
	heredoc  = regexp.MustCompile(`<<-?\s*(?:'([A-Za-z_]+)'|"([A-Za-z_]+)"|([A-Za-z_]+))`)
	prebuild = "go_tool_bin"
)

// Finding is one suite that runs a launcher under a changed HOME with no prebuild.
type Finding struct {
	File     string
	Line     int // the first line that changes HOME
	Launcher string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: runs %s under a changed HOME without a setup_file that calls %s", f.File, f.Line, f.Launcher, prebuild)
}

// Launchers are the base names of the *.sh scripts of the kit toolbelt and scripts/ that call go_tool_bin
// (a lib/ below them is sourced, never run), sorted.
func Launchers(root string) ([]string, error) {
	var names []string
	for _, dir := range launcherDirs {
		files, err := filepath.Glob(filepath.Join(root, dir, "*.sh"))
		if err != nil {
			return nil, err
		}
		for _, p := range files {
			text, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			if strings.Contains(string(text), prebuild) {
				names = append(names, filepath.Base(p))
			}
		}
	}
	sort.Strings(names)
	return names, nil
}

// Lint checks every tests/*.bats of the repository at root.
func Lint(root string) ([]Finding, error) {
	names, err := Launchers(root)
	if err != nil {
		return nil, err
	}
	suites, err := filepath.Glob(filepath.Join(root, "tests", "*.bats"))
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, p := range suites {
		text, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		libs, err := loaded(root, string(text))
		if err != nil {
			return nil, err
		}
		if f := Check(filepath.Join("tests", filepath.Base(p)), string(text), names, libs...); f != nil {
			findings = append(findings, *f)
		}
	}
	return findings, nil
}

// loadCmd is a `load <name>` line of a suite: bats runs tests/<name>.bash, whose setup_file is the suite's.
var loadCmd = regexp.MustCompile(`(?m)^load\s+(\S+)\s*$`)

// loaded are the texts of the helper files a suite loads from the tests directory.
func loaded(root, text string) ([]string, error) {
	var libs []string
	for _, m := range loadCmd.FindAllStringSubmatch(text, -1) {
		lib, err := os.ReadFile(filepath.Join(root, "tests", m[1]+".bash"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		libs = append(libs, string(lib))
	}
	return libs, nil
}

// Check is the finding for one suite (name is how it is reported), nil when it is clean: it changes HOME
// in no command, runs none of the launchers, or builds in setup_file (its own or one of the loaded helper
// texts libs). Comment lines, here-document
// bodies and quoted strings (for HOME) are not read.
func Check(name, text string, launchers []string, libs ...string) *Finding {
	lines := strings.Split(text, "\n")
	first, launcher, built, inSetup, until := 0, "", false, false, ""
	for _, lib := range libs {
		built = built || buildsInSetupFile(lib)
	}
	for i, line := range lines {
		code := strings.TrimSpace(line)
		if until != "" {
			if code == until {
				until = ""
			}
			continue
		}
		if m := heredoc.FindStringSubmatch(line); m != nil && !strings.HasPrefix(code, "#") {
			until = m[1] + m[2] + m[3]
		}
		if code == "" || strings.HasPrefix(code, "#") {
			continue
		}
		switch {
		case setupFile.MatchString(line):
			inSetup = true
		case line == "}" || strings.HasPrefix(line, "@test "):
			inSetup = false
		}
		if inSetup && strings.Contains(line, prebuild) {
			built = true
		}
		if first == 0 && changesHome.MatchString(quoted.ReplaceAllString(line, `""`)) {
			first = i + 1
		}
		for _, l := range launchers {
			if launcher == "" && names(line, l) {
				launcher = l
			}
		}
	}
	if first == 0 || launcher == "" || built {
		return nil
	}
	return &Finding{name, first, launcher}
}

// names reports whether line mentions the script called script: its whole base name, not the tail of a
// longer one.
func names(line, script string) bool {
	for rest, off := line, 0; ; {
		i := strings.Index(rest, script)
		if i < 0 {
			return false
		}
		at, end := off+i, off+i+len(script)
		before := at == 0 || !isNameByte(line[at-1])
		after := end == len(line) || !isNameByte(line[end])
		if before && after {
			return true
		}
		rest, off = line[end:], end
	}
}

func isNameByte(b byte) bool {
	return b == '_' || b == '-' || b == '.' || '0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}

// buildsInSetupFile reports whether a setup_file function of text calls go_tool_bin.
func buildsInSetupFile(text string) bool {
	in := false
	for _, line := range strings.Split(text, "\n") {
		switch {
		case setupFile.MatchString(line):
			in = true
		case line == "}" || strings.HasPrefix(line, "@test "):
			in = false
		}
		if in && strings.Contains(line, prebuild) {
			return true
		}
	}
	return false
}
