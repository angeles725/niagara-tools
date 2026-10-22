package javasrc

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDifferentialAwk is the equivalence proof against the awk lib this package ports
// (build-n4-module-kit/toolbelt/lib/method-boundary.sh): every awk found on the host (awk, mawk,
// gawk --posix) runs mb_strip + mb_parse over the table-test inputs and every *.java under
// tests/fixtures/ and build-n4-module-kit/, and its dump must equal the Go dump byte for byte.
// JAVASRC_DIFF_ROOTS (a path list) adds more source trees, e.g. client checkouts; CI never sets it.
// It skips when bash or awk is missing.

// awkDriver dumps one block per input file: "F\t<path>\tE<state>", "S\t<stripped line>" per line,
// then "M\t<name>\t<start>\t<end>" per method. An empty file has no records, so awk prints no block.
const awkDriver = `
function flush(    st, c, i, k) {
    if (f == "") return
    st = mb_strip(l, n, o); c = mb_parse(o, n, s, e, m)
    printf "F\t%s\tE%d\n", f, st
    for (i = 1; i <= n; i++) printf "S\t%s\n", o[i]
    for (k = 0; k < c; k++) printf "M\t%s\t%d\t%d\n", m[k], s[k], e[k]
    n = 0
}
FNR == 1 { flush(); f = FILENAME }
{ l[++n] = $0 }
END { flush() }
`

func repoRoot(t testing.TB) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// javaFiles lists every *.java under the roots that exist, in walk order.
func javaFiles(t testing.TB, roots ...string) []string {
	t.Helper()
	var files []string
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".java") {
				files = append(files, p)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

// goDump renders the same dump as awkDriver and returns the line and method counts.
func goDump(t testing.TB, files []string) (dump string, lines, methods int) {
	t.Helper()
	var b strings.Builder
	for _, p := range files {
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		src, err := Lines(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(src) == 0 {
			continue
		}
		stripped, state := Strip(src)
		ms := Methods(stripped)
		fmt.Fprintf(&b, "F\t%s\tE%d\n", p, state)
		for _, s := range stripped {
			fmt.Fprintf(&b, "S\t%s\n", s)
		}
		for _, m := range ms {
			fmt.Fprintf(&b, "M\t%s\t%d\t%d\n", m.Name, m.Start, m.End)
		}
		lines += len(src)
		methods += len(ms)
	}
	return b.String(), lines, methods
}

// firstDiff names the first file block whose dump differs.
func firstDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	file := ""
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if strings.HasPrefix(wl, "F\t") {
			file = wl
		}
		if wl != gl {
			return fmt.Sprintf("in %q at dump line %d:\n go  %q\n awk %q", file, i+1, wl, gl)
		}
	}
	return ""
}

// tableInputs writes every table-test input to its own file.
func tableInputs(t *testing.T) []string {
	dir := t.TempDir()
	var srcs []string
	for _, tc := range stripCases {
		srcs = append(srcs, strings.Join(tc.in, "\n")+"\n")
	}
	for _, tc := range methodsCases {
		srcs = append(srcs, tc.src+"\n")
	}
	srcs = append(srcs, "a(); \"x\"; // c\n", "b(); /* open\n", "String t = \"\"\"\n", "a\r\nb() {\r\n", "x();")
	files := make([]string, len(srcs))
	for i, s := range srcs {
		files[i] = filepath.Join(dir, fmt.Sprintf("case%02d.java", i))
		if err := os.WriteFile(files[i], []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func TestDifferentialAwk(t *testing.T) {
	root := repoRoot(t)
	lib := filepath.Join(root, "build-n4-module-kit", "toolbelt", "lib", "method-boundary.sh")
	if _, err := os.Stat(lib); err != nil {
		t.Skipf("awk lib not found: %v", err)
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not found")
	}
	var awks []string
	for _, a := range []string{"awk", "mawk", "gawk --posix"} {
		if _, err := exec.LookPath(strings.Fields(a)[0]); err == nil {
			awks = append(awks, a)
		}
	}
	if len(awks) == 0 {
		t.Skip("no awk found")
	}
	roots := []string{filepath.Join(root, "tests", "fixtures"), filepath.Join(root, "build-n4-module-kit")}
	roots = append(roots, filepath.SplitList(os.Getenv("JAVASRC_DIFF_ROOTS"))...)
	files := append(tableInputs(t), javaFiles(t, roots...)...)
	want, lines, methods := goDump(t, files)
	t.Logf("compared %d files, %d lines, %d methods against %v", len(files), lines, methods, awks)

	for _, a := range awks {
		cmd := exec.Command("bash", append([]string{"-c", `. "$1"; a=$2; shift 2; exec $a "$MB_AWK$DRIVER" "$@"`, "bash", lib, a}, files...)...)
		cmd.Env = append(os.Environ(), "DRIVER="+awkDriver)
		var out, errs bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errs
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: %v\n%s", a, err, errs.String())
		}
		if got := out.String(); got != want {
			t.Errorf("%s dump differs from Go %s", a, firstDiff(want, got))
		}
	}
}
