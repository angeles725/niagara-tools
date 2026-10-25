package niagaratest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// gitEnv keeps the fixture's git hermetic: no system or global config, and no repository above the gradle root.
func (fx *fixture) gitEnv() []string {
	return []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CEILING_DIRECTORIES=" + filepath.Dir(fx.gradle)}
}

// git runs a real git in the fixture's gradle root, as the test's own setup.
func (fx *fixture) git(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=fixture", "-c", "user.email=fixture@example.com",
		"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Dir, cmd.Env = fx.gradle, append(os.Environ(), fx.gitEnv()...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %q: %v\n%s", args, err, out)
	}
}

// realGit answers the runner's git calls with a real git, hermetic as fx.git.
func (fx *fixture) realGit(ctx context.Context, c Cmd) (int, error) {
	c.Env = append(slices.Clone(c.Env), fx.gitEnv()...)
	return execRunner{}.Run(ctx, c)
}

// gitVerb is the git subcommand of a call: the first argument that is no option, -C's value skipped.
func gitVerb(args []string) string {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-C":
			i++
		case !strings.HasPrefix(a, "-"):
			return a
		}
	}
	return "?"
}

const moduleInclude = "ModuleA-rt/module-include.xml"

// The gradle root is snapshotted with git before anything is copied; a root that is no git worktree cannot be
// verified, so the run fails closed with exit 3 before any copy.
func TestRunGradleRootNotAWorktree(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	if err := os.RemoveAll(filepath.Join(fx.gradle, ".git")); err != nil {
		t.Fatal(err)
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if want := "FAIL  run-niagara-test  gradle root is not a git worktree: cannot verify it"; code != exitEnv || !strings.Contains(out, want) {
		t.Errorf("exit %d, want 3 with %q:\n%s", code, want, out)
	}
	if n := len(fx.f.of("robocopy")) + len(fx.f.of("gradle")) + len(fx.f.of("test")); n != 0 {
		t.Errorf("%d copy/build/run calls without a gradle root snapshot: %q", n, fx.f.kinds())
	}
	if l := fx.leftovers(t); len(l) != 0 {
		t.Errorf("left in %%TEMP%%: %q", l)
	}
}

// A build that leaves the gradle root as it found it, dirty files included, passes the gate.
func TestRunGradleRootUnchanged(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		dirty func(fx *fixture)
		n     int
	}{
		{"clean root", func(*fixture) {}, 0},
		{"dirty root left alone", func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, moduleInclude), "<types>wip</types>\n")
			write(t, filepath.Join(fx.gradle, "notes.txt"), "mine\n")
		}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			tc.dirty(fx)
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			want := "PASS  run-niagara-test  gradle root unchanged (" + strconv.Itoa(tc.n) + " tracked changes before, same after)"
			if code != exitPass || !strings.Contains(out, want) {
				t.Errorf("exit %d, want 0 with %q:\n%s", code, want, out)
			}
			ks := strings.Join(fx.f.kinds(), " ")
			if want := "temp profile preflight git-rev-parse git-status robocopy gradle test procs git-status"; ks != want {
				t.Errorf("steps %q, want %q", ks, want)
			}
		})
	}
}

// Any change the build makes to the gradle root is FAIL: a modified, deleted or new file, a content change to a
// file that was already dirty, and a dirty file the build reverted. A failed build is still checked.
func TestRunGradleRootChangedIsFail(t *testing.T) {
	t.Parallel()
	const prefix = "FAIL  run-niagara-test  gradle root changed by the build: "
	for _, tc := range []struct {
		name       string
		dirty      func(fx *fixture)
		build      func(fx *fixture)
		gradleExit int
		code       int
		want       string
	}{
		{"modifies a clean tracked file", nil, func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, moduleInclude), "<types><type name=\"BExampleStationTest\"/></types>\n")
		}, 0, exitFail, moduleInclude},
		{"deletes a tracked file", nil, func(fx *fixture) { _ = os.Remove(filepath.Join(fx.gradle, moduleInclude)) }, 0, exitFail, moduleInclude},
		{"creates an untracked file", nil, func(fx *fixture) { write(t, filepath.Join(fx.gradle, "ModuleA-rt", "new.xml"), "x\n") }, 0, exitFail, "ModuleA-rt/new.xml"},
		{"rewrites an already-dirty file", func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, moduleInclude), "<types>wip</types>\n")
		}, func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, moduleInclude), "<types>wip</types><type/>\n")
		}, 0, exitFail, moduleInclude},
		{"rewrites an already-untracked file", func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, "notes.txt"), "mine\n")
		}, func(fx *fixture) { write(t, filepath.Join(fx.gradle, "notes.txt"), "build\n") }, 0, exitFail, "notes.txt"},
		{"reverts a dirty file", func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, moduleInclude), "<types>wip</types>\n")
		}, func(fx *fixture) { write(t, filepath.Join(fx.gradle, moduleInclude), "<types/>\n") }, 0, exitFail, moduleInclude},
		{"failed build that modified a file", nil, func(fx *fixture) {
			write(t, filepath.Join(fx.gradle, moduleInclude), "<types><type/></types>\n")
		}, 1, exitEnv, moduleInclude},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			if tc.dirty != nil {
				tc.dirty(fx)
			}
			fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) { tc.build(fx); return tc.gradleExit, nil }
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			if want := prefix + tc.want; code != tc.code || !strings.Contains(out, want) {
				t.Errorf("exit %d, want %d with %q:\n%s", code, tc.code, want, out)
			}
			if strings.Contains(out, "gradle root unchanged") {
				t.Errorf("a PASS row beside the change:\n%s", out)
			}
			if n := len(fx.f.of("git-checkout")); n != 0 {
				t.Errorf("%d checkouts without --restore-gradle-root", n)
			}
		})
	}
}

// The FAIL row names at most 10 paths, sorted, then "and N more".
func TestRunGradleRootChangedListsTenPaths(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) {
		for i := range 12 {
			write(t, filepath.Join(fx.gradle, "gen", "f"+strconv.Itoa(10+i)+".txt"), "x\n")
		}
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	want := "FAIL  run-niagara-test  gradle root changed by the build: gen/f10.txt gen/f11.txt gen/f12.txt gen/f13.txt " +
		"gen/f14.txt gen/f15.txt gen/f16.txt gen/f17.txt gen/f18.txt gen/f19.txt and 2 more\n"
	if code != exitFail || !strings.Contains(out, want) {
		t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
	}
}

// --restore-gradle-root checks out only the files that were clean before and that the build modified; a file
// already dirty keeps the build's bytes, a new untracked file is listed and kept, and the run stays FAIL.
func TestRunGradleRootRestore(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	write(t, filepath.Join(fx.gradle, "ModuleA-ux", "module-include.xml"), "<ux/>\n")
	write(t, filepath.Join(fx.gradle, "gone.txt"), "tracked\n")
	fx.git(t, "add", "-A")
	fx.git(t, "commit", "-q", "-m", "more")
	write(t, filepath.Join(fx.gradle, "ModuleA-ux", "module-include.xml"), "<ux>wip</ux>\n")
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) {
		write(t, filepath.Join(fx.gradle, moduleInclude), "<types><type/></types>\n")
		_ = os.Remove(filepath.Join(fx.gradle, "gone.txt"))
		write(t, filepath.Join(fx.gradle, "ModuleA-ux", "module-include.xml"), "<ux>build</ux>\n")
		write(t, filepath.Join(fx.gradle, "ModuleA-rt", "new.xml"), "x\n")
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args("--restore-gradle-root")...)...)
	for _, w := range []string{
		"FAIL  run-niagara-test  gradle root changed by the build: ModuleA-rt/module-include.xml ModuleA-rt/new.xml ModuleA-ux/module-include.xml gone.txt\n",
		"PASS  run-niagara-test  gradle root restored: ModuleA-rt/module-include.xml gone.txt (untracked files the build created, left in place: ModuleA-rt/new.xml)\n",
	} {
		if !strings.Contains(out, w) {
			t.Errorf("no %q in\n%s", w, out)
		}
	}
	if code != exitFail {
		t.Errorf("exit %d, want 1: the restore is hygiene, not a pass", code)
	}
	for p, want := range map[string]string{moduleInclude: "<types/>\n", "gone.txt": "tracked\n",
		"ModuleA-ux/module-include.xml": "<ux>build</ux>\n", "ModuleA-rt/new.xml": "x\n"} {
		if b, err := os.ReadFile(filepath.Join(fx.gradle, p)); err != nil || string(b) != want {
			t.Errorf("%s = %q (%v), want %q", p, b, err, want)
		}
	}
	cos := fx.f.of("git-checkout")
	if len(cos) != 1 {
		t.Fatalf("%d checkouts, want 1: %q", len(cos), fx.f.kinds())
	}
	if got, want := cos[0].Args, []string{"--literal-pathspecs", "-C", fx.gradle, "checkout", "--", moduleInclude, "gone.txt"}; !slices.Equal(got, want) {
		t.Errorf("checkout args %q, want %q", got, want)
	}
	noDestructiveGit(t, fx)
	if ks := strings.Join(fx.f.kinds(), " "); !strings.HasSuffix(ks, "procs git-status git-checkout git-status") {
		t.Errorf("steps %q: no re-snapshot after the restore", ks)
	}
}

// With nothing clean to restore, --restore-gradle-root runs no checkout and lists the untracked files.
func TestRunGradleRootRestoreUntrackedOnly(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) {
		write(t, filepath.Join(fx.gradle, "ModuleA-rt", "new.xml"), "x\n")
		return 0, nil
	}
	code, out := fx.run(t, append([]string{"run"}, fx.args("--restore-gradle-root")...)...)
	want := "PASS  run-niagara-test  gradle root restored: nothing to check out (untracked files the build created, left in place: ModuleA-rt/new.xml)\n"
	if code != exitFail || !strings.Contains(out, want) {
		t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
	}
	if n := len(fx.f.of("git-checkout")); n != 0 {
		t.Errorf("%d checkouts with nothing clean to restore", n)
	}
	noDestructiveGit(t, fx)
}

// A restore the re-snapshot does not confirm is a FAIL row of its own.
func TestRunGradleRootRestoreUnconfirmed(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.f.on["gradle"] = func(context.Context, Cmd) (int, error) {
		write(t, filepath.Join(fx.gradle, moduleInclude), "<types><type/></types>\n")
		return 0, nil
	}
	fx.f.on["git-checkout"] = func(context.Context, Cmd) (int, error) { return 0, nil }
	code, out := fx.run(t, append([]string{"run"}, fx.args("--restore-gradle-root")...)...)
	want := "FAIL  run-niagara-test  gradle root not restored: " + moduleInclude + " still differs after git checkout\n"
	if code != exitFail || !strings.Contains(out, want) || strings.Contains(out, "gradle root restored:") {
		t.Errorf("exit %d, want 1 with %q:\n%s", code, want, out)
	}
}

// noDestructiveGit: the runner never cleans, resets or checks out the whole tree.
func noDestructiveGit(t *testing.T, fx *fixture) {
	t.Helper()
	for _, c := range fx.f.calls {
		if c.Name != "git" {
			continue
		}
		v := gitVerb(c.Args)
		if v == "clean" || v == "reset" || v == "restore" || v == "stash" {
			t.Errorf("destructive git call: %q", c.Args)
		}
		if v == "checkout" && (slices.Contains(c.Args, ".") || !slices.Contains(c.Args, "--")) {
			t.Errorf("checkout of the whole tree or without --: %q", c.Args)
		}
	}
}
