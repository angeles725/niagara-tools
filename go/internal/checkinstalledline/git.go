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

// show is the text of file at ref, "" when the ref has no such file.
func (r repository) show(ref, file string) string {
	text, _ := r.git(nil, "show", ref+":"+file)
	return text
}

// filesNaming lists the files of ref matching pathspec that hold the literal text, in git's path order.
// A path git quotes (a control or non-ASCII byte) is returned quoted, so show finds nothing for it, as
// in the bash tool.
func (r repository) filesNaming(ref, text, pathspec string) []string {
	out, _ := r.git(r.stderr, "grep", "-l", "-F", "-e", text, ref, "--", pathspec)
	var files []string
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line != "" {
			files = append(files, strings.TrimPrefix(line, ref+":"))
		}
	}
	return files
}
