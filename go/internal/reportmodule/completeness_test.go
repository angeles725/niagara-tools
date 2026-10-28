package reportmodule

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules"
)

// notInReport is every lint report-module deliberately does not run, each with why (issue #459: a new lint
// must be either a member or listed here, never silently skipped).
var notInReport = map[string]string{
	"lint-absolute-claims":     "a kit doc report over build-n4-module-kit/types/*.md (METHODOLOGY K33), not a module check",
	"lint-bacnet-status-flags": "reads a station config.bog, which a module root does not hold",
	"lint-bats-errexit":        "reads bats test files (the kit's own tests/), which a module root does not hold",
	"lint-intrinsic-lock":      "member wiring lands with the other course-coverage run B lints in kit-backlog B11.6",
	"lint-lifecycle-guard":     "member wiring lands with the other course-coverage run B lints in kit-backlog B11.6",
	"lint-manual-labels":       "needs an operator manual path, which a module root does not name",
	"lint-guard-pins":          "a kit meta-check of the toolbelt's own lint headers, run by the kit gates",
	"split-package-check":      "a project-level check across every module of a project, not of one module",
}

// TestEveryLintIsAMemberOrListed (issue #459): every n4lint rule in rules.All() and every
// toolbelt/lint-*.sh is in the member table or in notInReport, and notInReport names nothing that is a member
// or that no longer exists.
func TestEveryLintIsAMemberOrListed(t *testing.T) {
	t.Parallel()
	members := map[string]bool{}
	for _, s := range Scripts() {
		members[strings.TrimSuffix(s, ".sh")] = true
	}
	known := map[string]bool{}
	for _, sp := range rules.All() {
		known[sp.ID()] = true
	}
	scripts, err := filepath.Glob(filepath.Join("..", "..", "..", "build-n4-module-kit", "toolbelt", "lint-*.sh"))
	if err != nil || len(scripts) == 0 {
		t.Fatalf("no toolbelt/lint-*.sh found: %v", err)
	}
	for _, p := range scripts {
		known[strings.TrimSuffix(filepath.Base(p), ".sh")] = true
	}
	var missing []string
	for id := range known {
		if !members[id] && notInReport[id] == "" {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	for _, id := range missing {
		t.Errorf("%s is neither a report-module member (members.go) nor in notInReport with a reason", id)
	}
	for id, why := range notInReport {
		if members[id] || !known[id] || why == "" {
			t.Errorf("notInReport[%q]: a member, an unknown lint or no reason (%q)", id, why)
		}
	}
}
