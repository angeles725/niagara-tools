package skillinstall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// step is one line of an install run, in harness order: an absent harness, or a file to place.
type step struct {
	absent string // the "absent" row of an optional harness whose directory does not exist
	file   planned
}

// planInstall assesses every file of every selected harness before anything is written. present is
// whether at least one harness was there to install into.
func planInstall(o installOptions, selected []harness, tracked string, trackedData []byte, s cli.Stdio) (steps []step, present bool, code int) {
	for _, h := range selected {
		root := h.root(o.home, o.homeGiven, s)
		if h.optional {
			if isDir, _ := installtools.IsDir(root); !isDir {
				steps = append(steps, step{absent: fmt.Sprintf("%s: absent: %s (%s not found), nothing installed", installTool, h.name, root)})
				continue
			}
		}
		present = true
		all, err := units(h, root, o.skill, tracked, trackedData)
		if err != nil {
			fmt.Fprintf(s.Err, "%s: cannot render the prompt template: %v\n", installTool, err)
			return nil, false, exitEnv
		}
		for _, u := range all {
			p, err := assess(u)
			if err != nil {
				fmt.Fprintf(s.Err, "%s: cannot read %s: %s\n", installTool, u.target, installtools.Reason(err))
				return nil, false, exitEnv
			}
			steps = append(steps, step{file: p})
		}
	}
	return steps, present, exitOK
}

// runInstall carries out a plan: --dry-run reports it; otherwise every diverged copy without --force,
// every symbolic link on the way and every file that cannot be placed (unplaceable) is refused before the
// first write, then the files are placed in order.
func runInstall(steps []step, o installOptions, s cli.Stdio) int {
	files := make([]planned, 0, len(steps))
	for _, st := range steps {
		if st.absent == "" {
			files = append(files, st.file)
		}
	}
	if !o.dryRun {
		code := exitOK
		for _, p := range files {
			if p.installed && !p.current() && !o.force {
				reportDiverged(p, s)
				code = exitDiverged
			}
		}
		if code != exitOK {
			return code
		}
		for _, p := range files {
			if p.current() {
				continue
			}
			if link := symlinkIn(p.u.links); link != "" {
				fmt.Fprintf(s.Err, "%s: refusing to write through a symbolic link: %s (remove the link, then re-run)\n", installTool, link)
				return exitEnv
			}
			if reason := unplaceable(p.u); reason != "" {
				fmt.Fprintf(s.Err, "%s: refusing to install: %s (move it away, then re-run)\n", installTool, reason)
				return exitEnv
			}
		}
	}
	for _, st := range steps {
		switch {
		case st.absent != "":
			fmt.Fprintln(s.Out, st.absent)
		case o.dryRun:
			reportDryRun(st.file, s)
		default:
			if code := writeUnit(st.file, s); code != exitOK {
				return code
			}
		}
	}
	return exitOK
}
