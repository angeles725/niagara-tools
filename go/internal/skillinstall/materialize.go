package skillinstall

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// kitDir is the kit checkout whose generated state an install of skill materializes: "" for a skill
// with no kit, $BUILD_N4_KIT when it holds METHODOLOGY.md, else the repository's own kit. A
// $BUILD_N4_KIT that cannot be examined is an error: it is never read as "no kit there".
func kitDir(skill, repoRoot string, s cli.Stdio) (string, error) {
	if skill != kitSkill {
		return "", nil
	}
	if kit := s.Env(kitEnv); kit != "" {
		marker := kit + "/" + kitMarker
		isKit, err := installtools.IsFile(marker)
		if err != nil {
			return "", fmt.Errorf("cannot check %s: %s", marker, installtools.Reason(err))
		}
		if isKit {
			return kit, nil
		}
	}
	return repoRoot + "/" + repoKit, nil
}

// materialize runs the kit's `kit-state.sh --write <kit>` (which prints each path it writes) and
// returns the command's exit code: 0 with no kit, with a kit that has no kit-state.sh (a warning) or
// after a successful run; 3 when the run fails or the script cannot be examined.
func materialize(kit string, s cli.Stdio) int {
	if kit == "" {
		return exitOK
	}
	script := kit + "/" + kitStateScript
	present, err := installtools.IsFile(script)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: cannot check %s: %s; the launcher is installed, the kit state is not\n",
			installTool, script, installtools.Reason(err))
		return exitEnv
	}
	if !present {
		fmt.Fprintf(s.Err, "%s: no %s in %s; generated kit state not materialized\n", installTool, kitStateScript, kit)
		return exitOK
	}
	cmd := exec.Command("bash", script, "--write", kit)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, s.Out, s.Err
	if err := cmd.Run(); err != nil {
		if _, exited := err.(*exec.ExitError); !exited {
			fmt.Fprintf(s.Err, "%s: bash: %v\n", installTool, err)
		}
		fmt.Fprintf(s.Err, "%s: kit-state --write failed in %s; the launcher is installed, the kit state is not\n", installTool, kit)
		return exitEnv
	}
	return exitOK
}
