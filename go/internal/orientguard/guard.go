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
	if optionLike(args[0]) { // a mistyped flag is never judged as a path
		fmt.Fprintf(s.Err, msgOption, args[0], args[0])
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
		row(fail, detailFail, path, prefix) // the prefix that was compared
		return ExitFail
	}
	return ExitOK
}

// defaultPrefix is the legal prefix below home ("" without one). Every final slash of home is dropped, so
// a home of "/x//" gives "/x/modulos_niagara_n4/Cliente/" (the bash tool's ${HOME%/} dropped only one).
func defaultPrefix(home string) string {
	if home == "" {
		return ""
	}
	return strings.TrimRight(home, "/") + legalSuffix
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
// The longest existing ancestor of arg is resolved (every symlink followed) and the rest is applied by its
// text (Reachable), so a new directory several levels deep inside the prefix passes, and a ".." after a
// missing directory is judged where it leads. A path that cannot be judged at all (a link loop, a directory
// that cannot be searched, an empty argument) fails closed and the row names it as given.
func locate(arg, prefix string) (path string, inside bool) {
	cwd, _ := syscall.Getwd() // the physical directory, as realpath reads it
	if path, ok := canonical(arg, cwd); ok {
		path = strings.TrimRight(path, "\n") // lost in the bash tool's $(realpath …)
		return path, under(path, prefix)
	}
	leads, ok := Reachable(arg, cwd)
	if !ok {
		return arg, false
	}
	return leads, under(leads, prefix)
}

// optionLike reports an argument that starts with a dash ("-" alone is a path): a usage error.
func optionLike(arg string) bool { return strings.HasPrefix(arg, "-") && arg != "-" }
