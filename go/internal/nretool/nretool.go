// Package nretool runs the build-time tools a Niagara install ships as nre classes (issue #851), so the build loop
// calls them by name instead of by hand. nre.exe embeds only the bootstrap: the class comes from the arguments, as
// nre <module>:<class> (research block B1453), and nre boots its own runtime from the install, no station needed.
//
// deps: devkit:com.tridium.devkit.util.DependencyChecker <module> lists the module parts a module depends on, read
// from the install's registry (read-only). The output is the parts, one per line; --raw passes nre's output through.
// palette (palette.go) and recompile (recompile.go) write files, so each carries a rule for what it may touch.
// list names the tools. An unknown tool is exit 3 with the supported list; the others return nre's exit code.
package nretool

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	exitFound = 1 // the tool ran and found nothing (a module the install does not hold)
	exitUsage = 3
	depsUsage = "usage: nre-tool deps [--raw] <niagara_home> <module>"
)

var (
	moduleName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	depLine    = regexp.MustCompile(`^ {4}(\S+)$`)
)

// depsClass is the class deps runs (B1453 §1453.4: DependencyChecker.java:21-24, one <moduleName> argument).
const depsClass = "devkit:com.tridium.devkit.util.DependencyChecker"

// tool is one supported nre tool: its name, a one-line purpose for list, and its entry point.
type tool struct {
	name, purpose string
	run           cli.Main
}

var tools = []tool{
	{"deps", "the module parts a module depends on, one per line (DependencyChecker; read-only)", deps},
	{"palette", "write <moduleDir>/module.palette from its src/ gifs (PaletteGenerator; replaces the file)", palette},
	{"recompile", "recompile and re-sign the Program objects of a bog or folder (RecompileTool; prompts)", recompile},
}

// names is every name Main accepts, sorted.
func names() string {
	n := []string{"list"}
	for _, t := range tools {
		n = append(n, t.name)
	}
	sort.Strings(n)
	return strings.Join(n, ", ")
}

// Main is the nre-tool command.
func Main(args []string, s cli.Stdio) int {
	if len(args) == 0 {
		fmt.Fprintf(s.Err, "%s\n       %s\n       %s\n       nre-tool list\n", depsUsage, paletteUsage[7:], recompileUsage[7:])
		return exitUsage
	}
	if args[0] == "list" {
		for _, t := range tools {
			fmt.Fprintf(s.Out, "%-10s %s\n", t.name, t.purpose)
		}
		return 0
	}
	for _, t := range tools {
		if t.name == args[0] {
			return t.run(args[1:], s)
		}
	}
	fmt.Fprintf(s.Err, "nre-tool: unknown tool %s (known: %s)\n", args[0], names())
	for _, t := range tools {
		fmt.Fprintf(s.Err, "  %-10s %s\n", t.name, t.purpose)
	}
	return exitUsage
}

// deps lists the module parts a module depends on; --raw prints nre's whole output.
func deps(args []string, s cli.Stdio) int {
	rest, raw := args, false
	if len(rest) > 0 && rest[0] == "--raw" {
		rest, raw = rest[1:], true
	}
	if len(rest) != 2 {
		fmt.Fprintln(s.Err, depsUsage)
		return exitUsage
	}
	home, module := rest[0], rest[1]
	if !moduleName.MatchString(module) {
		fmt.Fprintf(s.Err, "nre-tool: bad module name %q\n", module)
		return exitUsage
	}
	nre := findNre(home)
	if nre == "" {
		fmt.Fprintf(s.Err, "nre-tool: no bin/nre.exe or bin/nre under %s\n", home)
		return exitUsage
	}
	cmd := exec.Command(nre, depsClass, module)
	cmd.Dir = filepath.Dir(nre)
	out, err := cmd.CombinedOutput()
	if raw {
		_, _ = s.Out.Write(out)
		return exitCode(err)
	}
	parts, found := parseDeps(out, module)
	if !found {
		fmt.Fprintf(s.Err, "nre-tool: no module %s in %s (nre exit %d); the end of its output:\n%s\n",
			module, home, exitCode(err), tail(out, 5))
		return exitFound
	}
	for _, d := range parts {
		fmt.Fprintln(s.Out, d)
	}
	return 0
}

// parseDeps reads the indented module parts after DependencyChecker's "Dependencies for <module>:" header.
func parseDeps(out []byte, module string) (deps []string, found bool) {
	for _, l := range strings.Split(string(bytes.ReplaceAll(out, []byte("\r"), nil)), "\n") {
		if strings.TrimSpace(l) == "Dependencies for "+module+":" {
			found = true
			continue
		}
		if m := depLine.FindStringSubmatch(l); found && m != nil {
			deps = append(deps, m[1])
		}
	}
	return deps, found
}
