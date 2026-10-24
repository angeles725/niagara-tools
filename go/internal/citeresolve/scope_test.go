package citeresolve

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifier(t *testing.T) {
	t.Parallel()
	kit := filepath.Join("repo", "build-n4-module-kit")
	classify := NewClassifier(kit, HistoryRules)
	cases := []struct {
		card   string
		scope  Scope
		family string
	}{
		{filepath.Join(kit, "retros", "2026-10-24-go-kit-cite-resolve.md"), History, "retros"},
		{filepath.Join(kit, "retros", "INDEX.md"), History, "retros"},
		{filepath.Join(kit, "ledger.d", "2026-10-24-x.md"), History, "ledger.d"},
		{filepath.Join(kit, "BUILD-STATE.md"), History, "BUILD-STATE"},
		{filepath.Join(kit, "BUILD-STATE.src.md"), History, "BUILD-STATE"},
		{filepath.Join("repo", "CHANGELOG.md"), History, "CHANGELOG"},
		{filepath.Join(kit, "types", "platform.md"), Live, ""},
		{filepath.Join(kit, "METHODOLOGY.md"), Live, ""},
		{filepath.Join("repo", "odd", "tasks", "x.md"), Live, ""},
		{filepath.Join("repo", "elsewhere", "retros", "x.md"), Live, ""}, // a directory rule is kit-relative
		{filepath.Join(kit, "types", "retros.md"), Live, ""},
	}
	for _, tt := range cases {
		if scope, family := classify(tt.card); scope != tt.scope || family != tt.family {
			t.Errorf("classify(%s) = %s %q, want %s %q", tt.card, scope, family, tt.scope, tt.family)
		}
	}
}

func TestClassifierInjectedRules(t *testing.T) {
	t.Parallel()
	classify := NewClassifier("kit", []PathRule{{"odd/", "odd"}})
	if scope, family := classify(filepath.Join("kit", "odd", "x.md")); scope != History || family != "odd" {
		t.Errorf("injected rule: %s %q", scope, family)
	}
	if scope, _ := classify(filepath.Join("kit", "retros", "x.md")); scope != Live {
		t.Errorf("rules replace the default table, got %s for retros/", scope)
	}
}

func TestMainScopeSplit(t *testing.T) {
	t.Parallel()
	drift := filepath.Join("testdata", "kit", "ledger.d", "2026-10-24-drift.md")
	live := filepath.Join("testdata", "kit", "types", "unverifiable.md")
	changelog := filepath.Join("testdata", "CHANGELOG.md")
	cases := []struct {
		name  string
		cards []string
		code  int
		out   []string // substrings of stdout
		err   string   // exact stderr
	}{
		{"history drift passes --strict", []string{drift}, 0,
			[]string{"\tHISTORICAL_DRIFT\twas MISSING_DELTA (history ledger.d): ", "\tUNVERIFIABLE_KIND\t", "\tRESOLVED\t"},
			"kit-cite-resolve: 4 pointers: RESOLVED 1, HISTORICAL_DRIFT 2, UNVERIFIABLE_KIND 1\n" +
				"kit-cite-resolve: LIVE 0 pointers: none\n" +
				"kit-cite-resolve: HISTORY 4 pointers: RESOLVED 1, HISTORICAL_DRIFT 2, UNVERIFIABLE_KIND 1\n" +
				"kit-cite-resolve: strict findings: LIVE 0, HISTORY 0\n"},
		{"live miss fails --strict, template and unverifiable do not", []string{live}, 1,
			[]string{"\tSKIPPED_TEMPLATE\t", "\tUNVERIFIABLE_KIND\t", "\tMISSING_SECTION\t"},
			"kit-cite-resolve: 3 pointers: MISSING_SECTION 1, UNVERIFIABLE_KIND 1, SKIPPED_TEMPLATE 1\n" +
				"kit-cite-resolve: LIVE 3 pointers: MISSING_SECTION 1, UNVERIFIABLE_KIND 1, SKIPPED_TEMPLATE 1\n" +
				"kit-cite-resolve: HISTORY 0 pointers: none\n" +
				"kit-cite-resolve: strict findings: LIVE 1, HISTORY 0\n"},
		{"a history BARE_ID still fails --strict", []string{changelog, drift}, 1, []string{"\tBARE_ID\t"},
			"kit-cite-resolve: 5 pointers: RESOLVED 1, BARE_ID 1, HISTORICAL_DRIFT 2, UNVERIFIABLE_KIND 1\n" +
				"kit-cite-resolve: LIVE 0 pointers: none\n" +
				"kit-cite-resolve: HISTORY 5 pointers: RESOLVED 1, BARE_ID 1, HISTORICAL_DRIFT 2, UNVERIFIABLE_KIND 1\n" +
				"kit-cite-resolve: strict findings: LIVE 0, HISTORY 1\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, out, errs := run(t, rootArgs(t, append([]string{"--strict"}, tt.cards...)...)...)
			if code != tt.code || errs != tt.err {
				t.Errorf("exit %d, stderr\n%s\nwant exit %d, stderr\n%s", code, errs, tt.code, tt.err)
			}
			for _, s := range tt.out {
				if !strings.Contains(out, s) {
					t.Errorf("stdout lacks %q:\n%s", s, out)
				}
			}
		})
	}
}
