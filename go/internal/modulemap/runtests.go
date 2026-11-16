package modulemap

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// stationTest: the Java source b references a javax.baja type in code (import or fully qualified), read with
// comments blanked (javasrc.Strip) and literal text masked (javasrc.MaskLiterals).
func stationTest(b []byte) bool {
	code, _ := javasrc.Strip(strings.Split(string(b), "\n"))
	for _, ln := range javasrc.MaskLiterals(code) {
		if bajaRef.MatchString(ln) {
			return true
		}
	}
	return false
}

func tests(o opts, s cli.Stdio) error {
	m, err := readMap(o.root)
	if err != nil {
		return err
	}
	r, ok := findRow(m, o.target)
	if !ok {
		return fail(1, "module-map: no concern %s in %s", o.target, lint.MapFile)
	}
	files, err := expand(o.root, r.Tests)
	if err != nil {
		return err
	}
	failed, envErr := false, false
	for _, f := range files {
		switch {
		case strings.HasSuffix(f, ".js"):
			fmt.Fprintf(s.Out, "skip  %s  Jasmine browser spec (run it with the module's Karma harness, types/dashboard.md DUX-TEST1)\n", f)
			continue
		case !strings.HasSuffix(f, ".java") || !strings.Contains("/"+f, "/srcTest/"):
			fmt.Fprintf(s.Out, "skip  %s  not a srcTest Java test\n", f)
			continue
		}
		b, err := os.ReadFile(filepath.Join(o.root, f))
		if err != nil {
			return fail(3, "module-map: %v", err)
		}
		// Mutation: MMT-import -- a substring match skips a pure test that merely names javax.baja
		// Mutation: MMT-fqn -- an import-only check runs a test that uses javax.baja fully qualified as pure
		if stationTest(b) {
			fmt.Fprintf(s.Out, "skip  %s  uses javax.baja (a station test: run it under moduleTest on a station)\n", f)
			continue
		}
		pm := pkgLine.FindSubmatch(b)
		if pm == nil {
			return fail(3, "module-map: %s has no package line", f)
		}
		fqcn := string(pm[1]) + "." + strings.TrimSuffix(path.Base(f), ".java")
		tb := s.Env("MODULE_MAP_TOOLBELT")
		if tb == "" {
			return fail(3, "module-map: MODULE_MAP_TOOLBELT is unset (run through toolbelt/module-map.sh)")
		}
		part := strings.SplitN(f, "/", 2)[0]
		fmt.Fprintf(s.Out, "run   %s  %s\n", f, fqcn)
		// The runner inherits the process environment (Stdio only reads single variables); toolbelt/module-map.sh
		// exports MODULE_MAP_TOOLBELT into it. Exit 1 is a failed test; any other exit or a failed start is exit 3.
		cmd := exec.Command(filepath.Join(tb, "run-pure-test.sh"), filepath.Join(o.root, part), fqcn)
		cmd.Stdout, cmd.Stderr = s.Out, s.Err
		if err := cmd.Run(); err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) && ee.ExitCode() == 1 {
				failed = true
			} else {
				envErr = true
				fmt.Fprintf(s.Err, "module-map: %s: %v\n", fqcn, err)
			}
		}
	}
	switch {
	case failed:
		return fail(1, "module-map: a test of concern %s failed", r.Concern)
	case envErr:
		return fail(3, "module-map: a test of concern %s could not run", r.Concern)
	}
	return nil
}
