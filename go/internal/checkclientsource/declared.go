package checkclientsource

import "strings"

// unknown is the declared value, branch or commit that honestly says "not known": it skips its comparison.
const unknown = "unknown"

// declaration is a parsed source_of_truth value. path is "" when nothing is declared to compare (the
// value is empty or `unknown`); raw tells the two apart.
type declaration struct {
	raw                  string
	path, branch, commit string
}

// shape is how a declared value parsed.
type shape int

const (
	shapeOK        shape = iota
	shapeUsage           // three fields with an empty one, or a fourth: the usage text
	shapeMalformed       // not <path>@<branch>@<commit>: its own message
)

// parseDeclared splits `<path>@<branch>@<commit>`; "" and `unknown` are valid values with no path.
func parseDeclared(raw string) (declaration, shape) {
	d := declaration{raw: raw, branch: unknown, commit: unknown}
	if raw == "" || raw == unknown {
		return d, shapeOK
	}
	path, rest, _ := strings.Cut(raw, "@")
	branch, commit, three := strings.Cut(rest, "@")
	switch {
	case !three:
		return d, shapeMalformed
	case path == "" || branch == "" || commit == "" || strings.Contains(commit, "@"):
		return d, shapeUsage
	}
	d.path, d.branch, d.commit = path, branch, commit
	return d, shapeOK
}

// names reports whether the declared path names the work tree whose top level is top: top ends in
// /<path>, or is <path> itself.
func (d declaration) names(top string) bool {
	return strings.HasSuffix(top, "/"+strings.TrimPrefix(d.path, "/")) || top == strings.TrimSuffix(d.path, "/")
}
