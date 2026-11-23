// Package gradleuserhome is lint-gradle-userhome (G17): a gradle.properties that tries to move the user home of the
// gradle build through a setting an already-running daemon never reads. toolbelt/lint-gradle-userhome.sh is the
// wrapper and its header is the user-facing contract.
//
// A per-file rule over every gradle.properties under the root, read as java.util.Properties reads it: a line whose
// first non-blank character is # or ! is a comment, a line ending in a backslash continues on the next one, and a key
// ends at the first = or : or blank. Rows:
//   - GU1: org.gradle.jvmargs whose value holds -Duser.home= (the daemon is forked with the jvmargs of its first
//     start; a daemon that is already up keeps its own user.home).
//   - GU2: a systemProp.user.home key (gradle sets it as a system property of the build JVM, which the daemon has
//     already started, so the signing plugin still reads the real ~/.tridium).
//
// toolbelt/build.sh mirror mode does it the working way: JAVA_TOOL_OPTIONS and --no-daemon
// (go/internal/gradleenv). [ev: retro r8-mirror-userhome]
package gradleuserhome

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-gradle-userhome"

// Spec registers the rule: [--strict] [--] <root>, WARN rows (exit 1 under --strict), left out of n4lint --all (a
// gradle or module root).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.One,
		ModuleRoot: true, // a gradle or module root, not a source root: n4lint --all leaves it out
		Usage:      "usage: lint-gradle-userhome.sh [--strict] <root>",
		Select:     lint.Selection{Names: []string{"gradle.properties"}, Prune: []string{"build", "node_modules"}},
		EmptyError: lint.EmptyRow{Msg: "no gradle.properties found (wrong path: pass a module or gradle root)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	viaEnv = "through JAVA_TOOL_OPTIONS and run with --no-daemon (toolbelt/build.sh mirror mode does)"
	gu1    = "GU1 org.gradle.jvmargs sets -Duser.home=: an already-running gradle daemon never sees it; set it " + viaEnv
	gu2    = "GU2 systemProp.user.home sets the home of no already-running gradle daemon; set -Duser.home= " + viaEnv
)

// Check reports GU1 and GU2 at the first physical line of the property.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i := 0; i < len(f.Raw); i++ {
		line, start := strings.TrimRight(f.Raw[i], "\r"), i+1
		if t := strings.TrimLeft(line, " \t\f"); t == "" || t[0] == '#' || t[0] == '!' {
			continue
		}
		for strings.HasSuffix(line, "\\") && !strings.HasSuffix(line, "\\\\") && i+1 < len(f.Raw) {
			i++
			line = strings.TrimSuffix(line, "\\") + strings.TrimLeft(strings.TrimRight(f.Raw[i], "\r"), " \t\f")
		}
		key, val := splitProperty(line)
		switch {
		case key == "org.gradle.jvmargs" && strings.Contains(val, "-Duser.home="):
			out = append(out, lint.Finding{Path: f.Path, Line: start, Msg: gu1})
		case key == "systemProp.user.home":
			out = append(out, lint.Finding{Path: f.Path, Line: start, Msg: gu2})
		}
	}
	return out, nil
}

// splitProperty is a logical property line's key and value: the key ends at the first = : or blank, and the value
// starts after that separator (blanks around an = or : are not part of it).
func splitProperty(line string) (key, val string) {
	line = strings.TrimLeft(line, " \t\f")
	end := strings.IndexAny(line, "=: \t\f")
	if end < 0 {
		return line, ""
	}
	key, rest := line[:end], strings.TrimLeft(line[end:], " \t\f")
	if rest != "" && (rest[0] == '=' || rest[0] == ':') {
		rest = strings.TrimLeft(rest[1:], " \t\f")
	}
	return key, rest
}
