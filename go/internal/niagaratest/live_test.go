package niagaratest

import (
	"bytes"
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// TestLiveRun runs `run` end to end on a real install from WSL. It boots a test NRE (about 4.5 min the first
// time), so it is opt-in: RUN_NIAGARA_TEST_LIVE=1 plus RNT_LIVE_INSTALL, RNT_LIVE_GRADLE_ROOT, RNT_LIVE_PART and
// RNT_LIVE_MODULE, e.g. RNT_LIVE_INSTALL='C:\Niagara\Niagara-4.x' RNT_LIVE_PART=ModuleA-rt RNT_LIVE_MODULE=ModuleA.
func TestLiveRun(t *testing.T) {
	if os.Getenv("RUN_NIAGARA_TEST_LIVE") != "1" {
		t.Skip("live Windows run: set RUN_NIAGARA_TEST_LIVE=1 and RNT_LIVE_* (see the comment)")
	}
	args := []string{"run"}
	for _, f := range []struct{ flag, env string }{{"--install", "RNT_LIVE_INSTALL"}, {"--gradle-root", "RNT_LIVE_GRADLE_ROOT"},
		{"--part", "RNT_LIVE_PART"}, {"--module", "RNT_LIVE_MODULE"}} {
		v := os.Getenv(f.env)
		if v == "" {
			t.Fatalf("RUN_NIAGARA_TEST_LIVE=1 needs %s", f.env)
		}
		args = append(args, f.flag, v)
	}
	var o, e bytes.Buffer
	if code := Main(args, cli.Stdio{Out: &o, Err: &e}); code != exitPass {
		t.Fatalf("exit %d\n%s%s", code, o.String(), e.String())
	}
	t.Log(o.String())
}
