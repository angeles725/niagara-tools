package puretest

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// `run-pure-test --all <module-rt-dir>`: every *Test.java under srcTest/ is compiled by one javac and run by
// one JUnitCore, each class named from its package line, then one summary line (issue #1115).

const (
	flagAll    = "--all"
	testBase   = "srcTest"
	testSuffix = "Test.java"
	msgNoTests = "no *Test.java under %s"
	msgSummary = "%d classes, %d tests, %d failures"
)

// junitOK and junitFailed are JUnitCore's closing lines: "OK (3 tests)" and "Tests run: 3,  Failures: 1".
var (
	junitOK     = regexp.MustCompile(`(?m)^OK \((\d+) tests?\)`)
	junitFailed = regexp.MustCompile(`(?m)^Tests run: (\d+),\s+Failures: (\d+)`)
)

// all runs every test of the part in one JVM; the exit code is JUnit's.
func (r *run) all(rt string) int {
	t, code := r.tools(rt)
	if code != ExitOK {
		return code
	}
	sources := testSources(t.rt + "/" + testBase)
	if len(sources) == 0 {
		return r.die(ExitEnv, msgNoTests, t.rt+"/"+testBase)
	}
	var classes, roots []string
	used := map[string]bool{}
	for _, src := range sources {
		classes = append(classes, classOf(src))
		for _, root := range testRoots {
			if strings.HasPrefix(src, t.rt+"/"+root+"/") {
				used[root] = true
				break
			}
		}
	}
	for _, root := range testRoots {
		if used[root] {
			roots = append(roots, t.rt+"/"+root)
		}
	}
	var out bytes.Buffer
	tee := &run{cli.Stdio{Out: io.MultiWriter(r.s.Out, &out), Err: r.s.Err, Getenv: r.s.Getenv}}
	code = tee.build(t, sourcePath(t.rt, roots, r.s.Env(kitJavaEnv)), sources, classes...)
	if tests, failures, ok := summarize(out.String()); ok {
		fmt.Fprintf(r.s.Err, prefix+msgSummary+"\n", len(classes), tests, failures)
	}
	return code
}

// testSources are the *Test.java files under dir, in name order.
func testSources(dir string) []string {
	var found []string
	for _, f := range javaSources(dir) {
		if strings.HasSuffix(f, testSuffix) {
			found = append(found, filepath.ToSlash(f))
		}
	}
	return found
}

// summarize reads the tests run and the failures from JUnitCore's output; ok is false when it has neither line
// (the JVM did not get as far as running them).
func summarize(output string) (tests, failures int, ok bool) {
	if m := junitFailed.FindAllStringSubmatch(output, -1); m != nil {
		last := m[len(m)-1]
		tests, _ = strconv.Atoi(last[1])
		failures, _ = strconv.Atoi(last[2])
		return tests, failures, true
	}
	if m := junitOK.FindAllStringSubmatch(output, -1); m != nil {
		tests, _ = strconv.Atoi(m[len(m)-1][1])
		return tests, 0, true
	}
	return 0, 0, false
}
