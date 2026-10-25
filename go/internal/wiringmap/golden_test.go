package wiringmap

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden replays every testdata case (recorded from the bash generate-wiring-map.sh by
// golden/record/record.sh, or a hand-written truth-* case) through golden.Runner, in its tree as cwd
// (record.env CHDIR=1, so a relative --output replays too): exact output and the whole tree the run leaves.
func TestGolden(t *testing.T) {
	golden.Runner{Cases: 65, Main: func(_, args []string, s cli.Stdio) int { return Main(args, s) }}.Run(t)
}
