package wiringmap

import (
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

const usage = "usage: generate-wiring-map.sh <facade-src-dir> [--lexicon <module.lexicon>] [--output <file>]\n"

// Main parses `<facade-src-dir> [--lexicon <file>] [--output <file>]` (options after the directory, the last
// of each wins) and prints the map, or writes it to the --output file. Exits 0, or 3 for usage and env errors:
// a directory the walk cannot list or a source or lexicon it cannot read prints no map.
func Main(args []string, s cli.Stdio) int {
	fail := func(format string, a ...any) int {
		fmt.Fprintf(s.Err, "generate-wiring-map: "+format+"\n", a...)
		return 3
	}
	if len(args) < 1 {
		fmt.Fprint(s.Err, usage)
		return 3
	}
	src, out, lexicon := args[0], "-", ""
	for rest := args[1:]; len(rest) > 0; rest = rest[2:] {
		if rest[0] != "--output" && rest[0] != "--lexicon" {
			return fail("unknown option: %s", rest[0])
		}
		if len(rest) < 2 {
			return fail("%s requires a filename", rest[0])
		}
		if rest[0] == "--output" {
			out = rest[1]
		} else if lexicon = rest[1]; !regular(lexicon) {
			return fail("lexicon not found: %s", lexicon)
		}
	}
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		return fail("not a directory: %s", src)
	}
	for _, cand := range []string{src + "/module.lexicon", src + "/../module.lexicon"} {
		if lexicon == "" && regular(cand) {
			lexicon = cand
		}
	}
	files, err := lint.Walk(src, lint.Selection{Names: []string{"*.java"}})
	if err != nil {
		return fail("cannot list every file under %s: %v", src, err)
	}
	var op, sum []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fail("cannot read %s: %s", f, strerror.Text(err))
		}
		o, m := Slots(string(data))
		op, sum = append(op, o...), append(sum, m...)
	}
	names, err := readLexicon(lexicon)
	if err != nil {
		return fail("cannot read %s: %s", lexicon, strerror.Text(err))
	}
	if out == "-" {
		render(s.Out, src, rows(op, names), rows(sum, names))
		return 0
	}
	if err := writeFile(out, func(w io.Writer) { render(w, src, rows(op, names), rows(sum, names)) }); err != nil {
		fmt.Fprintf(s.Err, "generate-wiring-map: %s: %s\n", out, strerror.Text(err))
	}
	fmt.Fprintf(s.Out, "generate-wiring-map: wrote %s\n", out)
	return 0
}

// regular is bash `[ -f p ]`: p names a regular file, a symlink to one included.
func regular(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

func writeFile(path string, fill func(io.Writer)) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	fill(f)
	return f.Close()
}
