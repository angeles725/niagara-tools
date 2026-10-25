package niagaratest

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// nuhPassRe is the row naming the files under the scratch niagara_user_home this boot wrote.
var nuhPassRe = regexp.MustCompile(`(?m)^PASS  run-niagara-test  scratch niagara_user_home written by this boot: (.*)$`)

// nuhFailRow starts the row of a boot that wrote no file under the scratch niagara_user_home.
const nuhFailRow = "FAIL  run-niagara-test  scratch niagara_user_home not written by this boot"

// nuhOf is the WSL path of the scratch niagara_user_home a test.exe call was given.
func (fx *fixture) nuhOf(c Cmd) string { return fx.f.wsl(argValue(c.Env, "niagara_user_home=")) }

// greenOnly is a test.exe that writes its results and console log and nothing under niagara_user_home.
func (fx *fixture) greenOnly(_ context.Context, c Cmd) (int, error) {
	fx.f.testExe(c, greenXML, greenLog)
	return 0, nil
}

// (a)-(d) On a --reuse-home boot the kept nuh\ already has etc\ and security\, so their presence proves nothing:
// the NRE must have written a file under nuh\ during this boot (new or changed since just before test.exe,
// its mtime inside the boot window widened by the skew tolerance).
func TestRunReuseHomeNREWroteScratchHome(t *testing.T) {
	t.Parallel()
	t.Run("a written in the window is named", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		win, _ := fx.keepRun(t)
		fx.f.on["test"] = func(ctx context.Context, c Cmd) (int, error) {
			code, err := fx.greenOnly(ctx, c)
			write(t, filepath.Join(fx.nuhOf(c), "registry", "modules.xml"), "boot 2")
			return code, err
		}
		code, out := fx.reuse(t, win)
		m := nuhPassRe.FindStringSubmatch(out)
		if code != exitPass || m == nil {
			t.Fatalf("exit %d, want 0 with a scratch niagara_user_home PASS row:\n%s", code, out)
		}
		if !strings.HasPrefix(m[1], "1 file: registry/modules.xml (mtime 20") || !strings.Contains(m[1], "skew tolerance 2s") {
			t.Errorf("row %q does not name registry/modules.xml with its mtime and the 2s tolerance", m[1])
		}
		for _, old := range []string{"etc/nre.properties", "security/keystore.jceks"} {
			if strings.Contains(m[1], old) {
				t.Errorf("row %q names %s, which boot 1 wrote and boot 2 did not touch", m[1], old)
			}
		}
	})
	t.Run("b nothing under nuh changes", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		win, _ := fx.keepRun(t)
		fx.f.on["test"] = fx.greenOnly
		code, out := fx.reuse(t, win)
		if code != exitFail || !strings.Contains(out, nuhFailRow) || nuhPassRe.MatchString(out) {
			t.Errorf("exit %d, want 1 with %q and no PASS row:\n%s", code, nuhFailRow, out)
		}
	})
	t.Run("c written only by the runner", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		win, _ := fx.keepRun(t)
		fx.f.on["gradle"] = func(_ context.Context, c Cmd) (int, error) {
			write(t, filepath.Join(argValue(c.Args, "-Pniagara_user_home="), "signing", "build.txt"), "x")
			return 0, nil
		}
		fx.f.on["test"] = fx.greenOnly
		code, out := fx.reuse(t, win)
		if code != exitFail || !strings.Contains(out, nuhFailRow) || strings.Contains(out, "signing/build.txt") {
			t.Errorf("exit %d, want 1 with %q and the build's file not qualifying:\n%s", code, nuhFailRow, out)
		}
	})
	for _, tc := range []struct {
		name string
		off  time.Duration
	}{{"d mtime before the window", -time.Hour}, {"d mtime after the window", time.Hour}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			win, _ := fx.keepRun(t)
			fx.f.on["test"] = func(ctx context.Context, c Cmd) (int, error) {
				code, err := fx.greenOnly(ctx, c)
				p := filepath.Join(fx.nuhOf(c), "etc", "nre.properties")
				write(t, p, "rewritten")
				at := time.Now().Add(tc.off)
				if err := os.Chtimes(p, at, at); err != nil {
					t.Fatal(err)
				}
				return code, err
			}
			code, out := fx.reuse(t, win)
			if code != exitFail || !strings.Contains(out, nuhFailRow) {
				t.Errorf("exit %d, want 1 with %q (an mtime %s off the window does not qualify):\n%s", code, nuhFailRow, tc.off, out)
			}
		})
	}
}

// A fresh run carries the same proof: the NRE creates etc\nre.properties and security\ keystores in the empty
// scratch home, and a build's leftovers under nuh\ are not a write of this boot.
func TestRunFreshNREWroteScratchHome(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	m := nuhPassRe.FindStringSubmatch(out)
	if code != exitPass || m == nil || !strings.HasPrefix(m[1], "2 files: etc/nre.properties (mtime ") ||
		!strings.Contains(m[1], "security/keystore.jceks (mtime ") {
		t.Errorf("exit %d, want 0 with a PASS row naming etc/nre.properties and security/keystore.jceks:\n%s", code, out)
	}
}
