package orientguard

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Main implements `orient-guard <module-root>`.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 1 {
		fmt.Fprint(s.Err, msgUsage)
		return ExitUsage
	}
	row := func(status, detail string, a ...any) {
		fmt.Fprintf(s.Out, rowFormat, status, tool, fmt.Sprintf(detail, a...))
	}
	byDefault := defaultPrefix(s.Env("HOME"))
	prefix := byDefault
	if seam := s.Env(PrefixEnv); seam != "" {
		prefix = withSlash(seam)
		row(warn, detailSeam, prefix) // the seam is never silent
	} else if byDefault == "" {
		fmt.Fprint(s.Err, msgNoHome) // "/modulos_niagara_n4/Cliente/" would be a guess
		return ExitUsage
	}
	path, inside := locate(args[0], prefix)
	switch {
	case inside:
		row(pass, "%s", path)
	case s.Env(OverrideEnv) == OverrideOn:
		row(warn, detailOther, path)
	default:
		row(fail, detailFail, path, byDefault)
		return ExitFail
	}
	return ExitOK
}

// defaultPrefix is the legal prefix below home ("" without one). One final slash of home is dropped, as
// the bash tool's ${HOME%/} did.
func defaultPrefix(home string) string {
	if home == "" {
		return ""
	}
	return strings.TrimSuffix(home, "/") + legalSuffix
}

// withSlash is p ending in a slash.
func withSlash(p string) string {
	if strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}

// under reports whether path is prefix or below it, by whole components: the slash appended to path
// keeps ClienteX/ out of Cliente/.
func under(path, prefix string) bool {
	return strings.HasPrefix(strings.TrimSuffix(path, "/")+"/", prefix)
}

// locate is the path the rows name for arg and whether it is under prefix.
//
// A resolved path (canonical) decides by itself. A path that cannot be resolved is compared as given,
// as the bash tool did when realpath failed; but its text alone never passes: what does resolve of it
// (reachable) must stay under prefix too, else the row names where it leads.
func locate(arg, prefix string) (path string, inside bool) {
	cwd, _ := syscall.Getwd() // the physical directory, as realpath reads it
	if optionLike(arg) {
		return arg, under(arg, prefix)
	}
	if path, ok := canonical(arg, cwd); ok {
		path = strings.TrimRight(path, "\n") // lost in the bash tool's $(realpath …)
		return path, under(path, prefix)
	}
	if !under(arg, prefix) {
		return arg, false
	}
	leads, ok := reachable(arg, cwd)
	if !ok {
		return arg, false
	}
	if !under(leads, prefix) {
		return leads, false
	}
	return arg, true
}

// optionLike reports an argument realpath and cd took as an option: the bash tool compared it as given.
func optionLike(arg string) bool { return strings.HasPrefix(arg, "-") && arg != "-" }
