package newretro

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// today is the date testdata/setup.sh's fake `date` prints.
func today() string { return "2026-01-02" }

// TestGolden replays every testdata case (recorded from the bash new-retro.sh by golden/record/record.sh, or
// a hand-written truth-* case) through golden.Runner, with DefaultRootEnv = the case dir (the recorded
// script's own ../): exact output and the whole tree the run leaves.
func TestGolden(t *testing.T) {
	golden.Runner{
		Cases: 44,
		Setup: func(_ *testing.T, dir string, env map[string]string) []string { env[DefaultRootEnv] = dir; return nil },
		Main:  func(_, args []string, s cli.Stdio) int { return Command(today)(args, s) },
	}.Run(t)
}
