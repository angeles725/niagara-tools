package niagaratest

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const sysProps = "defaults/system.properties"

// withSysProps gives the fixture install a defaults/system.properties holding body.
func (fx *fixture) withSysProps(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(fx.install, sysProps)
	write(t, p, body)
	return p
}

// copiedSysProps captures the copy's system.properties as test.exe sees it.
func (fx *fixture) copiedSysProps(t *testing.T) *string {
	t.Helper()
	var got string
	fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
		b, err := os.ReadFile(filepath.Join(c.Dir, sysProps))
		if err != nil {
			got = "<missing: " + err.Error() + ">"
		} else {
			got = string(b)
		}
		fx.f.testExe(c, greenXML, greenLog)
		fx.f.useNUH(c)
		return 0, nil
	}
	return &got
}

// Guard 1: the flag is off by default and takes only `low`.
func TestRunModuleVerificationValues(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"medium", "high", "LOW", "", "off"} {
		t.Run("value "+v, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.withSysProps(t, "a=1\n")
			code, out := fx.run(t, append([]string{"run"}, fx.args("--module-verification", v)...)...)
			if code != exitUsage || !strings.Contains(out, "--module-verification") {
				t.Errorf("exit %d, want 2 naming the flag:\n%s", code, out)
			}
			if n := len(fx.f.calls); n != 0 {
				t.Errorf("%d calls after a usage error", n)
			}
		})
	}
	t.Run("off by default", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		const body = "#niagara.moduleVerificationMode=medium\n"
		fx.withSysProps(t, body)
		got := fx.copiedSysProps(t)
		code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
		if code != exitPass || *got != body || strings.Contains(out, "module verification") {
			t.Errorf("exit %d, copy %q, want the file untouched and no row:\n%s", code, *got, out)
		}
	})
}

// Guard 3: exactly one line is appended to the copy, in the file's line-ending style.
func TestRunModuleVerificationLowAppendsOneLine(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body, want string }{
		{"LF", "a=1\n#niagara.moduleVerificationMode=medium\n", "a=1\n#niagara.moduleVerificationMode=medium\nniagara.moduleVerificationMode=low\n"},
		{"CRLF", "a=1\r\nb=2\r\n", "a=1\r\nb=2\r\nniagara.moduleVerificationMode=low\r\n"},
		{"CRLF, no final newline", "a=1\r\nb=2", "a=1\r\nb=2\r\nniagara.moduleVerificationMode=low\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			src := fx.withSysProps(t, tc.body)
			got := fx.copiedSysProps(t)
			code, out := fx.run(t, append([]string{"run"}, fx.args("--module-verification", "low")...)...)
			if code != exitPass {
				t.Fatalf("exit %d\n%s", code, out)
			}
			if *got != tc.want {
				t.Errorf("copy %q, want %q", *got, tc.want)
			}
			if b, _ := os.ReadFile(src); string(b) != tc.body {
				t.Errorf("install file %q, want it untouched %q", b, tc.body)
			}
			if !strings.Contains(out, "PASS  run-niagara-test  module verification: low (copy only)") {
				t.Errorf("no module verification row:\n%s", out)
			}
			if ks := fx.f.kinds(); ks[slices.Index(ks, "robocopy")+1] != "gradle" {
				t.Errorf("steps %q", fx.f.kinds())
			}
		})
	}
}

// Guard 2: the line goes only into the runner's own copy; a copy whose system.properties resolves into the
// install or out of the scratch dir is refused, before any build.
func TestRunModuleVerificationRefusesOutsideCopy(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, want string
		target     func(fx *fixture) string
	}{
		{"resolves to --install", "resolves inside --install", func(fx *fixture) string { return filepath.Join(fx.install, sysProps) }},
		{"resolves inside --install", "resolves inside --install", func(fx *fixture) string {
			p := filepath.Join(fx.install, "etc", "other.properties")
			write(t, p, "x\n")
			return p
		}},
		{"resolves outside the scratch dir", "resolves outside the run's scratch dir", func(fx *fixture) string {
			p := filepath.Join(fx.temp, "elsewhere.properties")
			write(t, p, "x\n")
			return p
		}},
		{"resolves off the drives", "resolves off the Windows drives", func(fx *fixture) string {
			p := filepath.Join(fx.gradle, "wsl.properties")
			write(t, p, "x\n")
			return p
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			src := fx.withSysProps(t, "a=1\n")
			target := tc.target(fx)
			before, _ := os.ReadFile(target)
			fx.f.on["robocopy"] = func(_ context.Context, c Cmd) (int, error) {
				fx.f.robocopy(c.Args[2:])
				if dst := fx.f.wsl(c.Args[3]); strings.HasSuffix(dst, "/defaults") {
					_ = os.Remove(filepath.Join(dst, "system.properties"))
					if err := os.Symlink(target, filepath.Join(dst, "system.properties")); err != nil {
						t.Fatal(err)
					}
				}
				return 1, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args("--module-verification", "low")...)...)
			if want := "FAIL  run-niagara-test  module verification: refused: the copy's defaults/system.properties " + tc.want; code != exitUsage || !strings.Contains(out, want) {
				t.Errorf("exit %d, want 2 with %q:\n%s", code, want, out)
			}
			if len(fx.f.of("gradle")) != 0 {
				t.Errorf("built after a refusal: %q", fx.f.kinds())
			}
			if b, _ := os.ReadFile(target); string(b) != string(before) {
				t.Errorf("%s written: %q", target, b)
			}
			if b, _ := os.ReadFile(src); string(b) != "a=1\n" {
				t.Errorf("install file written: %q", b)
			}
			if l := fx.leftovers(t); len(l) > 1 || (len(l) == 1 && l[0] != "elsewhere.properties") {
				t.Errorf("left in %%TEMP%%: %q", l)
			}
		})
	}
}

// The flag needs a system.properties in the install to append to; without one it is refused before the copy.
func TestRunModuleVerificationNeedsInstallFile(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args("--module-verification", "low")...)...)
	if code != exitUsage || !strings.Contains(out, "no defaults/system.properties") || len(fx.f.of("robocopy")) != 0 {
		t.Errorf("exit %d, want 2 before any copy:\n%s", code, out)
	}
}

// Guard 4: the install's system.properties is hashed before the copy and after cleanup; a change is FAIL.
func TestRunModuleVerificationInstallFileHashed(t *testing.T) {
	t.Parallel()
	t.Run("unchanged", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		fx.withSysProps(t, "a=1\n")
		code, out := fx.run(t, append([]string{"run"}, fx.args("--module-verification", "low")...)...)
		// sha256("a=1\n") = fe3209d6…
		if want := "PASS  run-niagara-test  install defaults/system.properties unchanged (sha256 fe3209d6)"; code != exitPass || !strings.Contains(out, want) {
			t.Errorf("exit %d, want 0 with %q:\n%s", code, want, out)
		}
	})
	t.Run("changed", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		src := fx.withSysProps(t, "a=1\n")
		fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) { write(t, src, "a=2\n"); return 0, nil }
		code, out := fx.run(t, append([]string{"run"}, fx.args("--module-verification", "low")...)...)
		if want := "FAIL  run-niagara-test  install defaults/system.properties changed during the run"; code != exitFail || !strings.Contains(out, want) {
			t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
		}
	})
}

// Guard 5: a rejected signature with the flag off names the flag; with it on, no hint.
func TestRunSignatureRejectionHintsFlag(t *testing.T) {
	t.Parallel()
	const rejected = "Module ModuleA-rtTest failed signature validation\r\ninvalid test target(s): ModuleA\r\n"
	const hint = "hint: the test NRE rejected the test module's signature; on a disposable copy pass --module-verification low"
	for _, tc := range []struct {
		name  string
		extra []string
		hint  bool
	}{{"flag off", nil, true}, {"flag on", []string{"--module-verification", "low"}, false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.withSysProps(t, "a=1\n")
			fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
				fx.f.useNUH(c)
				_, _ = c.Stderr.Write([]byte(rejected))
				return 0, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args(tc.extra...)...)...)
			if code != exitFail || strings.Contains(out, hint) != tc.hint {
				t.Errorf("exit %d, want 1, hint %v (%q):\n%s", code, tc.hint, hint, out)
			}
		})
	}
}
