package niagaratest

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// goldenCases is the exact number of testdata cases: a lost or an extra case fails the run.
const goldenCases = 16

// TestGolden replays the CLI cases of testdata/ (testdata/README.md) through golden.Runner.
func TestGolden(t *testing.T) {
	golden.Runner{Cases: goldenCases, Main: func(_, args []string, s cli.Stdio) int { return Main(args, s) }}.Run(t)
}
