package skillinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// unit is one file install-skill places: the skill launcher (copied from source) or a prompt file
// (data rendered from the template source).
type unit struct {
	source, target string
	data           []byte
	copyFrom       bool     // write with installtools.Copy(source, target); else write data
	rendered       bool     // data is source rendered for this harness (the prompt file)
	links          []string // paths that must not be symbolic links before the write
}

// planned is a unit with the state of its target: installed (present) and its digest.
type planned struct {
	u                  unit
	installed          bool
	wantSHA, targetSHA string
}

func (p planned) current() bool { return p.installed && p.wantSHA == p.targetSHA }

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// assess reads the target of u; the error is a target that exists but cannot be read.
func assess(u unit) (planned, error) {
	p := planned{u: u, wantSHA: digest(u.data)}
	p.installed, _ = installtools.IsFile(u.target) // a target that cannot be examined fails at the write
	if p.installed {
		sum, err := installtools.SHA256(u.target)
		if err != nil {
			return p, err
		}
		p.targetSHA = sum
	}
	return p, nil
}

// symlinkIn is the first of paths that is a symbolic link, "" when none is.
func symlinkIn(paths []string) string {
	for _, path := range paths {
		if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
			return path
		}
	}
	return ""
}

// reportDryRun prints what a real run would do with p.
func reportDryRun(p planned, s cli.Stdio) {
	say := func(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }
	switch {
	case !p.installed:
		say("%s: [dry-run] would install to: %s", installTool, p.u.target)
	case p.current():
		say("%s: [dry-run] already current: %s", installTool, p.u.target)
	default:
		say("%s: [dry-run] would update (diverged): %s", installTool, p.u.target)
		if p.u.rendered {
			say("  rendered sha256: %s", p.wantSHA)
		} else {
			say("  tracked sha256: %s", p.wantSHA)
		}
		say("  current sha256: %s", p.targetSHA)
	}
}

// reportDiverged is the refusal of a diverged target without --force (stderr).
func reportDiverged(p planned, s cli.Stdio) {
	fmt.Fprintf(s.Err, "%s: installed copy diverges from tracked source\n", installTool)
	if p.u.rendered {
		fmt.Fprintf(s.Err, "  template: %s (rendered for this harness)\n  installed: %s\n  rendered sha256:  %s\n  installed sha256: %s\n",
			p.u.source, p.u.target, p.wantSHA, p.targetSHA)
	} else {
		fmt.Fprintf(s.Err, "  tracked:  %s\n  installed: %s\n  tracked sha256:   %s\n  installed sha256: %s\n",
			p.u.source, p.u.target, p.wantSHA, p.targetSHA)
	}
	fmt.Fprint(s.Err, "Re-run with --force to overwrite.\n")
}

// writeUnit places p (already checked: --force for a diverged copy, no symbolic link on the way) and reports it.
func writeUnit(p planned, s cli.Stdio) int {
	fail := func(format string, a ...any) int {
		fmt.Fprintf(s.Err, installTool+": "+format+"\n", a...)
		return exitEnv
	}
	say := func(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }
	if p.current() {
		say("%s: already current: %s", installTool, p.u.target)
		return exitOK
	}
	if p.installed {
		say("%s: --force: overwriting diverged copy", installTool)
	}
	dir := filepath.Dir(p.u.target)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return fail("cannot create target directory: %s", dir)
	}
	var err error
	if p.u.copyFrom {
		err = installtools.Copy(p.u.source, p.u.target)
	} else {
		err = os.WriteFile(p.u.target, p.u.data, 0o644)
	}
	if err != nil {
		return fail("cannot write %s: %s", p.u.target, installtools.Reason(err))
	}
	say("%s: installed %s", installTool, p.u.target)
	return exitOK
}

// units are the files a skill installs into one harness: the launcher, then the prompt file if the
// harness needs one. Their links are the target, its skill directory and the skills/ or prompts/ parent.
func units(h harness, root, skill, tracked string, trackedData []byte) ([]unit, error) {
	dir, file := h.skillFile(root, skill)
	all := []unit{{source: tracked, target: file, data: trackedData, copyFrom: true,
		links: []string{h.skillsDir(root), dir, file}}}
	if prompt := h.promptFile(root, skill); prompt != "" {
		data, err := renderPrompt(skill, tracked, file)
		if err != nil {
			return nil, err
		}
		all = append(all, unit{source: promptSource(tracked), target: prompt, data: data, rendered: true,
			links: []string{h.promptsDir(root), prompt}})
	}
	return all, nil
}
