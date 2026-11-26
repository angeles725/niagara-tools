package reportmodule

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// jsTestsCheck is the check name of the JS-tests row (issue #1124).
const jsTestsCheck = "js-tests"

// jsTestsTail is how many lines of node's output a FAIL row carries (the end of a node:test report holds the
// failing test and the summary); jsTestsTailBytes bounds them, so a row stays one readable line.
const (
	jsTestsTail      = 10
	jsTestsTailBytes = 1500
)

// jsTests is a native -ux step (no toolbelt script): the part's JS tests run as `node --test <file>...` in the
// part's directory, the files listed by the tool itself because `node --test <dir>` fails on Node 22 (it loads the
// directory as a module): *.test.js and *.test.cjs under srcTest/, in byte order, never under node_modules or a
// dot directory. A non-zero exit is a FAIL row with the tail of the output; no node on PATH (KIT_NODE first) is a
// SKIP row naming the tool; a srcTest/ with no such file is a SKIP row; a part with no srcTest/ prints no row, so
// the report of a module with no JS suite is as it was. [ev: issue #1124]
func (r *report) jsTests(a artifact) {
	src := a.dir + "/srcTest"
	if !isDir(src) {
		return
	}
	files := jsTestFiles(a.dir)
	if len(files) == 0 {
		r.emit(a.name, skip, jsTestsCheck, "no *.test.js or *.test.cjs file under srcTest/ (JS tests not run)")
		return
	}
	count := plural(len(files), "file")
	bin := r.getenv("KIT_NODE")
	if bin == "" {
		bin = commandV(r.getenv("PATH"), "node")
	}
	if !isExec(bin) {
		r.emit(a.name, skip, jsTestsCheck, fmt.Sprintf("node not on PATH: %s not run (install node >= 18 or set KIT_NODE)",
			plural(len(files), "JS test file")))
		return
	}
	limit := jsTestsLimit(r.getenv("KIT_JS_TESTS_TIMEOUT"))
	out, code, ok := r.x.runLimited(a.dir, limit, bin, append([]string{"--test"}, files...)...)
	switch {
	case r.truncated(a.name, jsTestsCheck, "node --test", ok):
	case code == codeTimeout:
		r.emit(a.name, failed, jsTestsCheck, fmt.Sprintf("node --test timed out after %ds (KIT_JS_TESTS_TIMEOUT)", limit))
	case code == 126 || code == 127:
		r.envFault(a.name, jsTestsCheck, fmt.Sprintf("env fault (exit %d): cannot run %s", code, bin))
	case code != 0:
		r.emit(a.name, failed, jsTestsCheck, fmt.Sprintf("node --test exited %d (%s): %s", code, count, outputTail(out)))
	default:
		r.emit(a.name, pass, jsTestsCheck, "node --test: "+count+" passed")
	}
}

// jsTestFiles lists the JS test files of the part at dir, as paths below it ("srcTest/rc/a.test.js"), in byte
// order: regular *.test.js and *.test.cjs files under srcTest/, none under node_modules or a dot directory.
func jsTestFiles(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir+"/srcTest", func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil // an unreadable entry holds no test the run could name
		case d.IsDir():
			if n := d.Name(); p != dir+"/srcTest" && (n == "node_modules" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
		case d.Type().IsRegular() && (strings.HasSuffix(p, ".test.js") || strings.HasSuffix(p, ".test.cjs")):
			rel, _ := filepath.Rel(dir, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files
}

// outputTail is the last jsTestsTail non-blank lines of out, trimmed and joined with " | ", cut at the front to
// jsTestsTailBytes.
func outputTail(out string) string {
	var lines []string
	for _, ln := range strings.Split(out, "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			lines = append(lines, ln)
		}
	}
	if len(lines) > jsTestsTail {
		lines = lines[len(lines)-jsTestsTail:]
	}
	tail := strings.Join(lines, " | ")
	if len(tail) > jsTestsTailBytes {
		tail = "..." + tail[len(tail)-jsTestsTailBytes:]
	}
	return tail
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return itoa(n) + " " + noun + "s"
}
