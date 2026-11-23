// Package nretool runs the build-time tools a Niagara install ships as nre classes (issue #851), so the build loop
// calls them by name instead of by hand. nre.exe embeds only the bootstrap: the class comes from the arguments, as
// nre <module>:<class> (research block B1453), and nre boots its own runtime from the install, no station needed.
//
// deps: devkit:com.tridium.devkit.util.DependencyChecker <module> lists the module parts a module depends on, read
// from the install's registry (read-only). The output is the parts, one per line; --raw passes nre's output through.
package nretool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	exitFound = 1 // the tool ran and found nothing (a module the install does not hold)
	exitUsage = 3
	usage     = "usage: nre-tool deps [--raw] <niagara_home> <module>"
)

var (
	moduleName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	depLine    = regexp.MustCompile(`^ {4}(\S+)$`)
)

// tools maps a tool name to its nre class.
var tools = map[string]string{"deps": "devkit:com.tridium.devkit.util.DependencyChecker"}

// Main is the nre-tool command.
func Main(args []string, s cli.Stdio) int {
	if len(args) == 0 {
		fmt.Fprintln(s.Err, usage)
		return exitUsage
	}
	class, ok := tools[args[0]]
	if !ok {
		fmt.Fprintf(s.Err, "nre-tool: unknown tool %s (known: deps)\n%s\n", args[0], usage)
		return exitUsage
	}
	rest, raw := args[1:], false
	if len(rest) > 0 && rest[0] == "--raw" {
		rest, raw = rest[1:], true
	}
	if len(rest) != 2 {
		fmt.Fprintln(s.Err, usage)
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
	cmd := exec.Command(nre, class, module)
	cmd.Dir = filepath.Dir(nre)
	out, err := cmd.CombinedOutput()
	if raw {
		_, _ = s.Out.Write(out)
		return exitCode(err)
	}
	deps, found := parseDeps(out, module)
	if !found {
		fmt.Fprintf(s.Err, "nre-tool: no module %s in %s (nre exit %d); the end of its output:\n%s\n",
			module, home, exitCode(err), tail(out, 5))
		return exitFound
	}
	for _, d := range deps {
		fmt.Fprintln(s.Out, d)
	}
	return 0
}

// findNre is the install's nre launcher: nre.exe (Windows install, run from WSL) or nre.
func findNre(home string) string {
	for _, name := range []string{"nre.exe", "nre"} {
		p := filepath.Join(home, "bin", name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
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

func exitCode(err error) int {
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	if err != nil {
		return exitUsage
	}
	return 0
}

// tail is the last n non-empty lines of out.
func tail(out []byte, n int) string {
	var lines []string
	for _, l := range strings.Split(string(bytes.ReplaceAll(out, []byte("\r"), nil)), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, "  "+l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
