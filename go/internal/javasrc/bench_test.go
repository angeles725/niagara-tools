package javasrc

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// benchCopies repeats the repo's Java corpus to make one large input (about 20k lines).
const benchCopies = 8

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

func benchLines(b *testing.B) []string {
	root := repoRoot(b)
	var all []string
	for _, p := range javaFiles(b, filepath.Join(root, "tests", "fixtures"), filepath.Join(root, "build-n4-module-kit")) {
		f, err := os.Open(p)
		if err != nil {
			b.Fatal(err)
		}
		src, err := Lines(f)
		f.Close()
		if err != nil {
			b.Fatal(err)
		}
		all = append(all, src...)
	}
	var big []string
	for range benchCopies {
		big = append(big, all...)
	}
	if len(big) == 0 {
		b.Skip("no Java corpus")
	}
	return big
}

func BenchmarkStrip(b *testing.B) {
	src := benchLines(b)
	b.ReportMetric(float64(len(src)), "lines")
	for b.Loop() {
		Strip(src)
	}
}

func BenchmarkMethods(b *testing.B) {
	stripped, _ := Strip(benchLines(b))
	b.ReportMetric(float64(len(stripped)), "lines")
	for b.Loop() {
		Methods(stripped)
	}
}
