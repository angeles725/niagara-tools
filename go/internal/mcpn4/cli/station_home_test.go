package cli

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The Go twins of test_tools_destructive.TestStationHomeFlag.

func TestFlagIsRepeatableAndDefaultsEmpty(t *testing.T) {
	t.Parallel()
	if cfg, _, _ := Parse(nil); len(cfg.StationHomes) != 0 {
		t.Errorf("default = %q", cfg.StationHomes)
	}
	cfg, _, err := Parse([]string{"--station-home", "A=/x", "--station-home", "B=/y"})
	if err != nil || !reflect.DeepEqual(cfg.StationHomes, []string{"A=/x", "B=/y"}) {
		t.Errorf("Parse = %q %v", cfg.StationHomes, err)
	}
}

func TestMalformedValuesExit2WithAClearMessage(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"nopath", "=/x", "A="} {
		code, _, errOut := run([]string{"--station-home", bad}, "")
		if code != 2 || !strings.Contains(errOut, "--station-home expects NAME=PATH") {
			t.Errorf("%q: %d %q", bad, code, errOut)
		}
	}
}

// TestServerPassesTheHomesToTheWriteState: NAME -> PATH as main() builds it (the path keeps
// every "=" after the first, the last repeat wins), with the save's default poll timing.
func TestServerPassesTheHomesToTheWriteState(t *testing.T) {
	t.Parallel()
	cfg, _, uerr := Parse([]string{"--allow-writes", "--state-dir", filepath.Join(t.TempDir(), "s"),
		"--station-home", "A=/x", "--station-home", "B=/y=z", "--station-home", "A=/w"})
	if uerr != nil {
		t.Fatal(uerr.Msg)
	}
	state, err := NewWriteState(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"A": "/w", "B": "/y=z"}; !reflect.DeepEqual(state.StationHomes, want) {
		t.Errorf("homes = %q", state.StationHomes)
	}
	if state.SaveTimeout != 30.0 || state.SaveInterval != 0.5 || state.Sleep == nil || state.Monotonic == nil {
		t.Errorf("timing = %v %v", state.SaveTimeout, state.SaveInterval)
	}
	if a, b := state.Monotonic(), state.Monotonic(); b < a {
		t.Errorf("monotonic went back: %v then %v", a, b)
	}
	cfg.StationHomes = nil
	if state, err = NewWriteState(cfg); err != nil || state.StationHomes == nil || len(state.StationHomes) != 0 {
		t.Errorf("no homes = %v %v", state.StationHomes, err)
	}
}
