package reportmodule

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// bundleGzipCheck is the check name of the pre-compressed bundle row (retro kit-gzip-bundle-writer-briefs Δ1).
const bundleGzipCheck = "bundle-gzip"

// bundleGzipLimit is the size over which a dist/*.js must ship with a .gz twin: 300 KB. An 832 KB SPA bundle served
// plain through a slow site tunnel delayed the whole page; its deterministic gzip twin was 241 KB.
// Mutation: RM-bundle-gzip-limit -- 300 000 bytes (not 300 KB) WARNs a bundle of exactly 300 KB
const bundleGzipLimit = 300 * 1024

// bundleGzip is a native -ux step (no toolbelt script): every *.js under src/rc/dist/ (a .js.map is not one) over
// bundleGzipLimit with no regular <name>.gz next to it is a WARN row; none is one PASS row. No src/rc/dist/
// directory (a part with no bundle step) prints no row; a dist/ the walk cannot read is an ERROR row, never PASS.
// The rows are advisory: WARN never fails the report (types/dashboard.md § Static serving).
func (r *report) bundleGzip(a artifact) {
	dist := filepath.Join(a.dir, "src", "rc", "dist")
	if !isDir(dist) {
		return
	}
	var rows []string
	err := filepath.WalkDir(dist, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".js") {
			return nil
		}
		st, err := os.Stat(p)
		if err != nil {
			return err
		}
		// Mutation: RM-bundle-gzip-twin -- dropping the twin test WARNs a bundle that ships its .gz
		if st.Size() <= bundleGzipLimit || isFile(p+".gz") {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Join(a.dir, "src", "rc"), p)
		rows = append(rows, fmt.Sprintf("%s  %d KB over 300 KB with no %s.gz twin: pre-compress it in the bundle step"+
			" and serve it with Content-Encoding: gzip (types/dashboard.md § Static serving)",
			filepath.ToSlash(rel), st.Size()/1024, d.Name()))
		return nil
	})
	if err != nil {
		r.envFault(a.name, bundleGzipCheck, "cannot read src/rc/dist/: "+err.Error())
		return
	}
	for _, row := range rows {
		r.emit(a.name, warn, bundleGzipCheck, row)
	}
	if len(rows) == 0 {
		r.emit(a.name, pass, bundleGzipCheck, "no dist/*.js over 300 KB without a .gz twin")
	}
}
