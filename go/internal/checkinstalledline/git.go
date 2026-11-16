package checkinstalledline

import (
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// repository runs git in one checkout. Every call reads local objects only; nothing is fetched.
type repository struct {
	dir    string
	stderr io.Writer // where git's own diagnostics go when a call does not silence them
}

// git runs git in the repository; its stderr goes to errTo (nil drops it).
func (r repository) git(errTo io.Writer, args ...string) (string, bool) {
	return gitcmd.Run(r.dir, errTo, args...)
}

// isCommit reports whether ref names a commit of the repository.
func (r repository) isCommit(ref string) bool {
	_, ok := r.git(r.stderr, "rev-parse", "-q", "--verify", ref+"^{commit}")
	return ok
}

// show is the text of file at ref, "" when the ref has no such file. The closing `--` says that
// `<ref>:<file>` is a revision: without it git refuses the argument as "both revision and filename"
// whenever the working tree holds an entry of that very name, and the file read as absent.
func (r repository) show(ref, file string) string {
	// Mutation: ILG-ambiguous -- dropping "--" skips a group whose `<ref>:<path>` name exists in the working tree
	text, _ := r.git(nil, "show", ref+":"+file, "--")
	return text
}

// filesNaming lists the files of ref matching pathspec that hold the literal text, in git's path order.
// The names are read NUL-separated (-z): git then prints every path as it is, so a path it would quote
// (a control or non-ASCII byte) reaches show unchanged and its group is compared, never skipped.
func (r repository) filesNaming(ref, text, pathspec string) []string {
	// Mutation: ILG-quoted-path -- dropping -z returns a non-ASCII path quoted, so show finds nothing and its group is skipped
	out, _ := r.git(r.stderr, "grep", "-l", "-z", "-F", "-e", text, ref, "--", pathspec)
	var files []string
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			files = append(files, strings.TrimPrefix(name, ref+":"))
		}
	}
	return files
}
