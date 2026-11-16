package cli

import (
	"reflect"
	"strings"
	"testing"
)

// The --niagara-home flag; the licence tool's use of it is pinned black-box in
// blackbox/license_files_test.go.

func TestNiagaraHomeIsRepeatableAndDefaultsEmpty(t *testing.T) {
	t.Parallel()
	if cfg, _, _ := Parse(nil); len(cfg.NiagaraHomes) != 0 {
		t.Errorf("default = %q", cfg.NiagaraHomes)
	}
	cfg, _, err := Parse([]string{"--niagara-home", "A=/x", "--niagara-home", "B=/y=z", "--niagara-home", "A=/w"})
	if err != nil || !reflect.DeepEqual(cfg.NiagaraHomes, []string{"A=/x", "B=/y=z", "A=/w"}) {
		t.Fatalf("Parse = %q %v", cfg.NiagaraHomes, err)
	}
	// NAME -> PATH: the path keeps every "=" after the first, the last repeat wins
	if got, want := niagaraHomes(cfg), map[string]string{"A": "/w", "B": "/y=z"}; !reflect.DeepEqual(got, want) {
		t.Errorf("homes = %q", got)
	}
}

func TestMalformedNiagaraHomeExits2WithAClearMessage(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"nopath", "=/x", "A="} {
		code, _, errOut := run([]string{"--niagara-home", bad}, "")
		if code != 2 || !strings.Contains(errOut, "--niagara-home expects NAME=PATH") {
			t.Errorf("%q: %d %q", bad, code, errOut)
		}
	}
}

func TestHelpNamesTheNiagaraHomeOption(t *testing.T) {
	t.Parallel()
	code, out, _ := run([]string{"--help"}, "")
	if code != 0 || !strings.Contains(out, "[--niagara-home NAME=PATH]") || !strings.Contains(out, "\n  --niagara-home NAME=PATH\n") {
		t.Errorf("help (%d) does not name --niagara-home", code)
	}
}
