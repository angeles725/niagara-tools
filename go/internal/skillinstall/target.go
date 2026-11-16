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
// (data rendered from a template, source names the template).
type unit struct {
	source, target string
	data           []byte
	copyFrom       bool     // write with installtools.Copy(source, target); else write data
	links          []string // paths that must not be symbolic links before the write
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
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

// place installs u under the --dry-run and --force rules and returns an exit code: 0 placed, current or
// reported; 1 diverged without --force; 3 a file that cannot be read or written.
func place(u unit, o installOptions, s cli.Stdio) int {
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(s.Err, installTool+": "+format+"\n", a...)
		return code
	}
	say := func(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }
	wantSHA := digest(u.data)
	installed, _ := installtools.IsFile(u.target) // a target that cannot be examined fails at the write below
	targetSHA := ""
	if installed {
		var err error
		if targetSHA, err = installtools.SHA256(u.target); err != nil {
			return fail(exitEnv, "cannot read %s: %s", u.target, installtools.Reason(err))
		}
	}
	switch {
	case o.dryRun:
		switch {
		case !installed:
			say("%s: [dry-run] would install to: %s", installTool, u.target)
		case wantSHA == targetSHA:
			say("%s: [dry-run] already current: %s", installTool, u.target)
		default:
			say("%s: [dry-run] would update (diverged): %s", installTool, u.target)
			say("  tracked sha256: %s", wantSHA)
			say("  current sha256: %s", targetSHA)
		}
		return exitOK
	case installed && wantSHA == targetSHA:
		say("%s: already current: %s", installTool, u.target)
		return exitOK
	case installed && !o.force:
		fail(exitDiverged, "installed copy diverges from tracked source")
		fmt.Fprintf(s.Err, "  tracked:  %s\n  installed: %s\n  tracked sha256:   %s\n  installed sha256: %s\n",
			u.source, u.target, wantSHA, targetSHA)
		fmt.Fprint(s.Err, "Re-run with --force to overwrite.\n")
		return exitDiverged
	case installed:
		say("%s: --force: overwriting diverged copy", installTool)
	}
	if link := symlinkIn(u.links); link != "" {
		return fail(exitEnv, "refusing to write through a symbolic link: %s (remove the link, then re-run)", link)
	}
	dir := filepath.Dir(u.target)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return fail(exitEnv, "cannot create target directory: %s", dir)
	}
	var err error
	if u.copyFrom {
		err = installtools.Copy(u.source, u.target)
	} else {
		err = os.WriteFile(u.target, u.data, 0o644)
	}
	if err != nil {
		return fail(exitEnv, "cannot write %s: %s", u.target, installtools.Reason(err))
	}
	say("%s: installed %s", installTool, u.target)
	return exitOK
}

// units are the files a skill installs into one harness: the launcher, then the prompt file if the
// harness needs one.
func units(h harness, root, skill, tracked string, trackedData []byte) ([]unit, error) {
	dir, file := h.skillFile(root, skill)
	all := []unit{{source: tracked, target: file, data: trackedData, copyFrom: true, links: []string{dir, file}}}
	if prompt := h.promptFile(root, skill); prompt != "" {
		data, err := renderPrompt(skill, tracked, file)
		if err != nil {
			return nil, err
		}
		all = append(all, unit{source: promptSource(tracked), target: prompt, data: data, links: []string{prompt}})
	}
	return all, nil
}
