package jarversion

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usageText = "usage: lint-bundled-jar-class-version.sh <module-root>\n"

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

func TestMain(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	clean, bad, none, locked := filepath.Join(d, "clean"), filepath.Join(d, "bad"), filepath.Join(d, "none"), filepath.Join(d, "locked")
	writeJar(t, clean, "libs/java8.jar", entry{name: "Foo.class", data: classHeader(52)})
	writeJar(t, clean, "build/libs/java9.jar", entry{name: "Baz.class", data: classHeader(53)})
	j9 := writeJar(t, bad, "libs/java9.jar", entry{name: "Bar.class", data: classHeader(53)})
	j11 := writeJar(t, bad, "libs/a/java11.jar", entry{name: "A.class", data: classHeader(52)}, entry{name: "B.class", data: classHeader(55)})
	writeJar(t, bad, "libs/ok.jar", entry{name: "Ok.class", data: classHeader(52)})
	touch(t, filepath.Join(none, "libs", "readme.txt"))
	lj := writeJar(t, locked, "libs/java9.jar", entry{name: "Bar.class", data: classHeader(53)})
	touch(t, filepath.Join(locked, "libs", "locked", "a.jar"))
	lock(t, filepath.Join(locked, "libs", "locked"))

	cases := []struct {
		name      string
		args      []string
		code      int
		out, errs string
	}{
		{"BJCV-usage: no argument", nil, ExitEnv, "", usageText},
		{"BJCV-nondir: not a directory", []string{filepath.Join(d, "nope")}, ExitEnv, "", "lint-bundled-jar-class-version: not a directory: " + d + "/nope\n"},
		{"an empty argument is not a directory", []string{""}, ExitEnv, "", "lint-bundled-jar-class-version: not a directory: \n"},
		{"BJCV1/BJCV4: Java 8 jars clean, build/ pruned", []string{clean}, ExitOK, "", ""},
		{"BJCV3: no jars", []string{none}, ExitOK, "", ""},
		{"BJCV2: one row per failing jar, C byte order, extra arguments ignored", []string{bad, "ignored"}, ExitFail,
			FailRow(j11, "B.class", 55) + FailRow(j9, "Bar.class", 53), ""},
		{"raw arity: a flag is taken as the root", []string{"--strict", bad}, ExitEnv, "", "lint-bundled-jar-class-version: not a directory: --strict\n"},
		{"BJCV-finderr: an unreadable sub-directory is exit 3, the rest still scanned", []string{locked}, ExitEnv,
			FailRow(lj, "Bar.class", 53), "lint-bundled-jar-class-version: cannot list every file under " + locked + ": find: '" + locked + "/libs/locked': Permission denied\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, out, errs := run(tt.args...)
			if code != tt.code || out != tt.out || errs != tt.errs {
				t.Errorf("Main(%q) = %d\nout  %q\nerr  %q\nwant %d\nout  %q\nerr  %q", tt.args, code, out, errs, tt.code, tt.out, tt.errs)
			}
		})
	}
}
