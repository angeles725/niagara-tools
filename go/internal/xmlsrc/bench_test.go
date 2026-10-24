package xmlsrc

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// syntheticBog is a station-shaped document of about size bytes: nested components with the
// attributes, escapes and character references a saved config.bog carries.
func syntheticBog(size int) []byte {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<bajaObjectGraph version=\"4.0\">\n<p n=\"station\" t=\"b:Station\">\n")
	for i := 0; sb.Len() < size; i++ {
		fmt.Fprintf(&sb, " <p n=\"Folder$20%d\" t=\"b:Folder\" h=\"%x\">\n", i, i)
		for j := range 8 {
			fmt.Fprintf(&sb, "  <p n=\"pt%d\" t=\"control:NumericWritable\" v=\"%d.5\" f=\"r\" x=\"a&amp;b&#x41;\"/>\n", j, j)
		}
		sb.WriteString("  <a n=\"l\" t=\"b:Link\"><p n=\"sourceOrd\" v=\"h:1a\"/></a>\n </p>\n")
	}
	sb.WriteString("</p>\n</bajaObjectGraph>\n")
	return []byte(sb.String())
}

// XMLSRC_BENCH_FILE benchmarks Open on a real file instead (read-only, outside the repository).
func BenchmarkOpen(b *testing.B) {
	if path := os.Getenv("XMLSRC_BENCH_FILE"); path != "" {
		st, err := os.Stat(path)
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(st.Size())
		for b.Loop() {
			if _, err := Open(path); err != nil {
				b.Fatal(err)
			}
		}
		return
	}
	data := syntheticBog(10 << 20)
	b.SetBytes(int64(len(data)))
	var n int
	for b.Loop() {
		d, err := Parse("synthetic.bog", data)
		if err != nil {
			b.Fatal(err)
		}
		n = len(d.Elements())
	}
	b.ReportMetric(float64(n), "elements")
}
