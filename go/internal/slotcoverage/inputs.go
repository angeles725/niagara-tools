package slotcoverage

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// grepLongest compiles an ERE that matches leftmost-longest, as grep -o does.
func grepLongest(expr string) *regexp.Regexp {
	re := regexp.MustCompile(expr)
	re.Longest()
	return re
}

var (
	typeTag    = grepLongest(`<type[[:space:]]+[^>]*\bname="[^"]*"`)
	nameAttr   = grepLongest(`\bname="[^"]*"`)
	skipLine   = regexp.MustCompile(`^[[:space:]]*#|^[[:space:]]*$`)
	trailBlank = " \t\n\v\f\r"
)

// lines splits a file's text the way awk and grep read it: a last line with no newline still counts.
func lines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

// grepO is grep -o: every match of re on every line of data, in order.
func grepO(re *regexp.Regexp, data []byte) []string {
	var out []string
	for _, ln := range lines(data) {
		out = append(out, re.FindAllString(ln, -1)...)
	}
	return out
}

// isFile is test -f; isDir is test -d (both follow a symlink).
func isFile(p string) bool { st, err := os.Stat(p); return err == nil && st.Mode().IsRegular() }
func isDir(p string) bool  { st, err := os.Stat(p); return err == nil && st.IsDir() }

// readInput reads one input file. A file it cannot read prints "cannot read" and returns false: an env
// error (exit 3), never a verdict on the rest.
func readInput(path string, s cli.Stdio) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(s.Err, "slot-coverage: cannot read %s: %s\n", path, strerror.Text(err))
		return nil, false
	}
	return data, true
}

// typeNames are the name="X" values of every <type ...> tag that grep -o finds on one line, in file order.
func typeNames(xml []byte) []string {
	var out []string
	for _, tag := range grepO(typeTag, xml) {
		for _, attr := range nameAttr.FindAllString(tag, -1) {
			out = append(out, strings.TrimSuffix(strings.TrimPrefix(attr, `name="`), `"`))
		}
	}
	return out
}

// lexiconKeys are the text before the first "=" of every lexicon line that is neither a comment nor blank
// and holds an "=" (grep -vE ... | grep '=' | cut -d= -f1), in file order, trailing blanks kept.
func lexiconKeys(lex []byte) []string {
	var out []string
	for _, ln := range lines(lex) {
		if skipLine.MatchString(ln) {
			continue
		}
		if i := strings.IndexByte(ln, '='); i >= 0 {
			out = append(out, ln[:i])
		}
	}
	return out
}

// trimKeys is sed 's/[[:space:]]*$//' over keys.
func trimKeys(keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = strings.TrimRight(k, trailBlank)
	}
	return out
}
