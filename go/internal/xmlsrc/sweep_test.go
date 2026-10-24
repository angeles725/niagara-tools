package xmlsrc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// sweepExts are the XML kinds the kit lints read; .lexicon files are key=value text, not XML.
var sweepExts = map[string]bool{".bog": true, ".xml": true, ".palette": true, ".px": true}

// XMLSRC_SWEEP_ROOTS (colon-separated, outside the repository: real client and module files) are
// opened one by one. A file may be malformed (a positioned *Error, logged and counted: some shipped
// Tridium and Honeywell files are, and expat agrees), but any other failure fails the test.
// XMLSRC_SWEEP_MAX_MALFORMED (default 0) caps the malformed count. Dot and *-worktrees directories
// are skipped, as the lint walker does.
func TestSweepRealFiles(t *testing.T) {
	roots := os.Getenv("XMLSRC_SWEEP_ROOTS")
	if roots == "" {
		t.Skip("XMLSRC_SWEEP_ROOTS not set")
	}
	maxBad := 0
	if v := os.Getenv("XMLSRC_SWEEP_MAX_MALFORMED"); v != "" {
		var err error
		if maxBad, err = strconv.Atoi(v); err != nil {
			t.Fatal(err)
		}
	}
	counts, bad := map[string]int{}, 0
	for _, root := range strings.Split(roots, ":") {
		err := filepath.WalkDir(root, func(path string, de fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if de.IsDir() {
				if path != root && (strings.HasPrefix(de.Name(), ".") || strings.HasSuffix(de.Name(), "-worktrees")) {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			if !sweepExts[ext] || !de.Type().IsRegular() {
				return nil
			}
			counts[ext]++
			var xe *Error
			if _, err := Open(path); errors.As(err, &xe) {
				bad++
				t.Logf("malformed: %v", err)
			} else if err != nil {
				t.Errorf("%v", err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	var exts []string
	for e := range counts {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	for _, e := range exts {
		t.Logf("%s: %d files", e, counts[e])
	}
	if len(counts) == 0 {
		t.Fatal("no XML file found under XMLSRC_SWEEP_ROOTS")
	}
	if t.Logf("%d malformed", bad); bad > maxBad {
		t.Errorf("%d malformed files, XMLSRC_SWEEP_MAX_MALFORMED is %d", bad, maxBad)
	}
}
