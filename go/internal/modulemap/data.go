package modulemap

import "regexp"

const usage = "usage: module-map.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]\n" +
	"       module-map.sh where <file|concern> --root <module-root>\n" +
	"       module-map.sh tests <concern> --root <module-root>"

// defaultRole is the role folder a new concern goes under, per part (types/structure.md § Module map).
var defaultRole = map[string]string{"rt": "model", "ux": "route", "wb": "model", "se": "model"}

var javaName = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

var (
	pkgLine = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)
	// bajaRef is a javax.baja reference in code (an import line or a fully qualified name, spaces around the dots
	// allowed): what makes a srcTest class a station test. It is matched on code with comments and literal text
	// blanked (stationTest), so a mention in a comment or a string never counts.
	bajaRef = regexp.MustCompile(`\bjavax\s*\.\s*baja\s*\.`)
)
