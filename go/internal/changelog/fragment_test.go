package changelog

import (
	"strings"
	"testing"
)

func TestCheckFragment(t *testing.T) {
	cases := []struct {
		body string
		ok   bool
	}{
		{"### Added — x\n\n- one\n", true},
		{"### Fixed\n- one\n", true},
		{"### Added — x\n\n- one\n\n### Removed — y\n\n- two\n", true},
		{"", false},
		{"  \n\n", false},
		{"\n### Added — x\n\n- one\n", false},              // line 1 must be the heading
		{"### Added — x\n\nprose\n", false},                // no bullet
		{"### Added — x\n\n### Fixed — y\n\n- b\n", false}, // empty first subsection
		{"### Improved — x\n\n- one\n", false},             // not a Keep a Changelog type
		{"### Added — x\n\n- one\n\n## [v1.0.0]\n", false}, // a release header would split the section
		{"### Added —\n\n- one\n", false},                  // a dash with no title
	}
	for _, c := range cases {
		if why := CheckFragment(c.body); (why == "") != c.ok {
			t.Errorf("CheckFragment(%q) = %q, want ok=%v", c.body, why, c.ok)
		}
	}
}

func TestCheckName(t *testing.T) {
	for name, ok := range map[string]bool{
		"2026-10-25-changelog-gate.md": true,
		"2026-10-25-a1.md":             true,
		"2026-13-01-x.md":              false,
		"2026-10-25-Upper.md":          false,
		"2026-10-25-.md":               false,
		"notes.md":                     false,
		"2026-10-25-x.txt":             false,
	} {
		if why := CheckName(name); (why == "") != ok {
			t.Errorf("CheckName(%q) = %q, want ok=%v", name, why, ok)
		}
	}
}

func TestCheckReason(t *testing.T) {
	for reason, ok := range map[string]bool{
		"test-only rename":                  true,
		"":                                  false,
		"   ":                               false,
		"tab\there":                         false,
		strings.Repeat("r", MaxReasonLen):   true,
		strings.Repeat("r", MaxReasonLen+1): false,
	} {
		if why := CheckReason(reason); (why == "") != ok {
			t.Errorf("CheckReason(%q) = %q, want ok=%v", reason, why, ok)
		}
	}
}

func TestUnreleasedBounds(t *testing.T) {
	lines := strings.Split("# C\n\n## [Unreleased]\n\n- a\n\n---\n\n## [v1] - 2026-01-01\n", "\n")
	s, e, ok := UnreleasedBounds(lines)
	if !ok || s != 2 || e != 8 {
		t.Fatalf("UnreleasedBounds = %d, %d, %v; want 2, 8, true", s, e, ok)
	}
	if _, e, ok := UnreleasedBounds(lines[:6]); !ok || e != 6 {
		t.Fatalf("an open Unreleased section ends at EOF: got end %d ok %v", e, ok)
	}
	if _, _, ok := UnreleasedBounds([]string{"# C", "## [v1]"}); ok {
		t.Fatal("a missing header must report ok=false")
	}
}

func TestIsBehaviour(t *testing.T) {
	for p, want := range map[string]bool{
		"go/internal/x/x.go":                         true,
		"go/internal/x/x_test.go":                    true,
		"build-n4-module-kit/toolbelt/a.sh":          true,
		"build-n4-module-kit/BUILD-LOOP.md":          true,
		"tools/t.py":                                 true,
		"go/bin/tool":                                false,
		"build-n4-module-kit/retros/2026-01-01-r.md": false,
		"build-n4-module-kit/ledger.d/2026-01-01.md": false,
		"build-n4-module-kit/BUILD-STATE.src.md":     false,
		"scripts/x.sh":                               true,
		"docs/a.md":                                  false,
	} {
		if got := IsBehaviour(p); got != want {
			t.Errorf("IsBehaviour(%q) = %v, want %v", p, got, want)
		}
	}
}
