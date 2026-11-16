package skillinstall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// driftCheck is one harness of a check-skill-drift run.
type driftCheck struct {
	o            driftOptions
	h            harness
	rel, tracked string // the tracked launcher, relative to the repository and absolute
}

// checkHarness prints the rows of one harness: SKIP when an optional harness is absent, else the
// launcher row and, where the harness needs one, the prompt row. It returns 0, 1 (a FAIL row) or 3.
func checkHarness(c driftCheck, s cli.Stdio) int {
	name := label(c.o.skill, c.h, c.o.explicit)
	root := c.h.root(c.o.home, c.o.homeGiven, s)
	if c.h.optional {
		if present, _ := installtools.IsDir(root); !present {
			fmt.Fprintf(s.Out, "SKIP  skill-drift  %s  absent: %s not found\n", name, root)
			return exitOK
		}
	}
	install := "scripts/" + installerScript + " --skill " + c.o.skill + harnessFlag(c.h)
	force := "scripts/" + installerScript + " --force --skill " + c.o.skill + harnessFlag(c.h)
	_, file := c.h.skillFile(root, c.o.skill)
	code := exitOK
	same, err := sameAs(file, c.tracked, nil, s)
	switch {
	case err != nil:
		return exitEnv
	case same == nil:
		fmt.Fprintf(s.Out, "FAIL  skill-drift  %s  not installed at %s — run %s\n", name, file, install)
		code = exitDiverged
	case *same:
		fmt.Fprintf(s.Out, "PASS  skill-drift  %s  installed copy matches %s\n", name, c.rel)
	default:
		fmt.Fprintf(s.Out, "FAIL  skill-drift  %s  installed copy %s differs from %s — run %s\n", name, file, c.rel, force)
		code = exitDiverged
	}
	prompt := c.h.promptFile(root, c.o.skill)
	if prompt == "" {
		return code
	}
	want, err := renderPrompt(c.o.skill, c.tracked, file)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: cannot render the prompt template: %v\n", driftTool, err)
		return exitEnv
	}
	same, err = sameAs(prompt, "", want, s)
	switch {
	case err != nil:
		return exitEnv
	case same == nil:
		fmt.Fprintf(s.Out, "FAIL  skill-drift  %s  prompt not installed at %s — run %s\n", name, prompt, install)
		code = exitDiverged
	case *same:
		fmt.Fprintf(s.Out, "PASS  skill-drift  %s  prompt %s matches its template\n", name, prompt)
	default:
		fmt.Fprintf(s.Out, "FAIL  skill-drift  %s  prompt %s differs from its template — run %s\n", name, prompt, force)
		code = exitDiverged
	}
	return code
}

// sameAs compares the installed file with the tracked file (or, when tracked is "", with want): nil when
// the installed file is absent; an error (already reported) when a digest cannot be computed.
func sameAs(installed, tracked string, want []byte, s cli.Stdio) (*bool, error) {
	if isFile, _ := installtools.IsFile(installed); !isFile {
		return nil, nil
	}
	wantSHA := digest(want)
	paths := []string{installed}
	if tracked != "" {
		paths = []string{tracked, installed}
	}
	var sums []string
	for _, path := range paths {
		sum, err := installtools.SHA256(path)
		if err != nil {
			fmt.Fprintf(s.Err, "%s: cannot read %s: %s\n", driftTool, path, installtools.Reason(err))
			fmt.Fprintf(s.Err, "%s: could not compute a digest\n", driftTool)
			return nil, err
		}
		sums = append(sums, sum)
	}
	if tracked != "" {
		wantSHA = sums[0]
	}
	same := wantSHA == sums[len(sums)-1]
	return &same, nil
}
