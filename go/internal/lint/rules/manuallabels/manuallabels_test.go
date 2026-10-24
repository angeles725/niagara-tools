package manuallabels

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden replays the runs of toolbelt/lint-manual-labels.sh recorded before the port: every
// tests/lint-manual-labels.bats run and the edge trees (each quote style, entities, nested tags, a label
// split across lines, code blocks, case and whitespace, CRLF, no final newline, invalid UTF-8, surrogate
// escapes, file selection, the argument contract).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 50) }
