package puretest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// These tests write fake javac and java scripts and run them, so none may use t.Parallel (a script
// still open for writing in another goroutine's fork cannot be executed: ETXTBSY).

const testClass = "com.example.demo.AdderTest"

// fixture is a module part (<dir>/rt) with one test, a home whose gradle cache holds both jars, and a
// PATH of fake tools that record how they were called.
type fixture struct {
	t        *testing.T
	dir      string // canonical
	rt       string
	junit    string
	hamcrest string
	out, err string
	kitJava  string // the value of RUN_PURE_TEST_KIT_JAVA the run sees ("" = unset)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, dir: dir, rt: dir + "/rt",
		junit:    dir + "/home/.gradle/caches/a/junit-4.13.2.jar",
		hamcrest: dir + "/home/.gradle/wrapper/b/hamcrest-core-1.3.jar"}
	f.write("rt/src/com/example/demo/Adder.java", "")
	f.write("rt/srcTest/test/com/example/demo/AdderTest.java", "package com.example.demo;\n")
	f.write("home/.gradle/caches/a/junit-4.13.2.jar", "")
	f.write("home/.gradle/wrapper/b/hamcrest-core-1.3.jar", "")
	f.write("tmp/.keep", "")
	f.tool("javac", 0)
	f.tool("java", 0)
	t.Setenv("PATH", dir+"/bin")
	t.Setenv("TMPDIR", dir+"/tmp")
	return f
}

func (f *fixture) write(rel, content string) {
	f.t.Helper()
	p := filepath.Join(f.dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

// tool installs a fake <name> on PATH: it prints a line on stdout and one on stderr, appends its working
// directory and arguments to <dir>/<name>.log, leaves a file in the directory after -d, and exits code.
func (f *fixture) tool(name string, code int) {
	f.write("bin/"+name, `#!/bin/sh
echo "`+name+` stdout"; echo "`+name+` stderr" >&2
{ printf '%s' "$(/bin/pwd -P)"; for a; do printf ' <%s>' "$a"; done; echo; } >> '`+f.dir+`/`+name+`.log'
while [ $# -gt 1 ]; do [ "$1" = -d ] && : > "$2/Compiled.class"; shift; done
exit `+string(rune('0'+code))+"\n")
}

// call is what the fake tool logged ("" when it never ran).
func (f *fixture) call(name string) string {
	b, _ := os.ReadFile(f.dir + "/" + name + ".log")
	return string(b)
}

// run runs the tool and returns its exit code; f.out and f.err hold what it and its children printed.
func (f *fixture) run(args ...string) int {
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(k string) string {
		if k == "HOME" {
			return f.dir + "/home"
		}
		if k == kitJavaEnv {
			return f.kitJava
		}
		return ""
	}})
	f.out, f.err = out.String(), errb.String()
	return code
}

// stops runs the tool on the fixture's test and expects it to end before javac with code and one message.
func (f *fixture) stops(name string, code int, message string, args ...string) {
	f.t.Helper()
	if got := f.run(args...); got != code || f.out != "" || f.err != "run-pure-test: "+message+"\n" {
		f.t.Errorf("%s: exit %d stdout %q stderr %q, want %d and %q", name, got, f.out, f.err, code, message)
	}
	if calls := f.call("javac") + f.call("java"); calls != "" {
		f.t.Errorf("%s: a tool ran: %s", name, calls)
	}
}

// tempDir is the directory javac was given after -d.
func (f *fixture) tempDir() string {
	_, after, _ := strings.Cut(f.call("javac"), "<-d> <")
	dir, _, _ := strings.Cut(after, ">")
	return dir
}
