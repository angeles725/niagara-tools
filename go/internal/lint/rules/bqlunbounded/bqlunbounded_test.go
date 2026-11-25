package bqlunbounded

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule; tests/fixtures/lint-bql-unbounded holds the same
// sources for tests/lint-bql-unbounded.bats). bqlu-notop and bqlu-strict: a tree query with and without top, WARN
// and FAIL under --strict; bqlu-history, bqlu-alarm and bqlu-orderby: the window, the index paths and order by;
// bqlu-waived and bqlu-waiver-bare: the waiver comment; bqlu-dynamic: built strings are counted, never judged;
// bqlu-notcode, bqlu-edges and bqlu-js: comments, text blocks, templates, case and a `|view:` suffix;
// bqlu-clean, bqlu-strict-clean, bqlu-minjs and bqlu-two-roots: the clean run, the skipped files, the one summary;
// bqlu-usage and bqlu-empty-root: the argument and root errors.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 17) }
