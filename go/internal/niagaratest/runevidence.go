package niagaratest

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// evidence is what the run log takes from the scratch dir, read before cleanup deletes it.
type evidence struct {
	stderr, stdout, listing []string
	testng                  testngEvidence // kept beside the log on a failed run only
	fullStderr              stderrEvidence // kept beside the log on a failed run only
}

// gatherEvidence reads the tails of test.exe's streams and lists its -output: dir, every read bounded.
func gatherEvidence(p *plan) evidence {
	var ev evidence
	for _, f := range []struct {
		name string
		n    int
		dst  *[]string
	}{{"stderr", diagStderrLines, &ev.stderr}, {"stdout", diagStdoutLines, &ev.stdout}} {
		lines, total, windowed, ok := tailFile(p.sub("out/"+f.name+".txt"), f.n)
		switch {
		case !ok:
			*f.dst = []string{"(no " + f.name + ".txt: test.exe did not run)"}
		case windowed:
			*f.dst = append([]string{fmt.Sprintf("(last %d lines of the file's last %d bytes)", len(lines), diagWindow)}, lines...)
		default:
			*f.dst = append([]string{fmt.Sprintf("(last %d of %d lines)", len(lines), total)}, lines...)
		}
	}
	ev.testng = gatherTestNG(p.sub("out/testng/testng-results.xml"))
	ev.fullStderr = gatherStderr(p.sub("out/stderr.txt"))
	root := p.sub("out/testng")
	if !isDir(root) {
		ev.listing = []string{"(no -output: dir)"}
		return ev
	}
	n := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if n++; n > diagMaxEntries {
			return fs.SkipAll
		}
		rel, _ := filepath.Rel(root, path)
		size := int64(-1)
		if info, err := d.Info(); err == nil {
			size = info.Size()
		}
		ev.listing = append(ev.listing, fmt.Sprintf("%s  %d bytes", filepath.ToSlash(rel), size))
		return nil
	})
	switch {
	case n > diagMaxEntries:
		ev.listing = append(ev.listing, fmt.Sprintf("(listing stopped at %d files)", diagMaxEntries))
	case n == 0:
		ev.listing = []string{"(empty)"}
	}
	return ev
}
