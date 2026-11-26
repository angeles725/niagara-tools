package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// deployTargetsFile is what each station runs, kept in the module root (the directory of GRADLEW_PATH, which
// holds the module directory) and read by build.sh --target: one line per target and module,
// "<target> <MOD>@<vendorVersion>=<commit>", "#" starts a comment (go/internal/buildmodule/deploytarget.go).
const deployTargetsFile = ".deploy-targets"

// targetName is a name a line can hold: one word, with no "#", "@" or "=", starting with a letter or a digit.
var targetName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// checkTarget refuses a target name that would break the line (exit 10) before anything runs. No name is fine:
// the deploy then records nothing and says how to.
func (d *deploy) checkTarget() int {
	if name := d.raw[optTarget]; name != "" && !targetName.MatchString(name) {
		return d.fail(ExitEnv, msgTargetName, name)
	}
	return proceed
}

// recordTarget is the phase after the deploy record: with --target <name> it writes the line for the target and
// the module into <module root>/.deploy-targets, the commit being the one recordDeploy read and the version
// that of the -rt jar (the schema baseline of build.sh --target is the -rt source of that commit). Without a
// name it writes nothing, and the last closing line says how to. A mode with no -rt jar (B) leaves the line as
// it is: the station still runs the -rt of the commit the line names. The jars are in the station by then, so a
// line that cannot be written is exit 41, as for .last-deploy-sha. Nothing runs for a deploy that stopped
// before (the phases after the copy are the only ones that get here) or for --no-deploy.
func (d *deploy) recordTarget() int {
	target := d.raw[optTarget]
	switch {
	case target == "":
		d.recordNote = noteNoTarget
		return proceed
	case !d.hasProfile("rt"):
		d.recordNote = fmt.Sprintf(noteTargetKept, target, d.raw[optMode])
		return proceed
	}
	path := d.moduleRoot() + "/" + deployTargetsFile
	version := vendorversion.ModuleVersion(d.builtJar("rt"))
	switch {
	case d.commit == "":
		return d.fail(ExitRecord, msgTargetNotRecorded, target, path, fmt.Sprintf(whyTargetNoCommit, d.noCommit))
	case version == "" || strings.ContainsAny(version, " \t#"):
		return d.fail(ExitRecord, msgTargetNotRecorded, target, path, fmt.Sprintf(whyTargetNoVersion, d.jarName("rt")))
	}
	line := fmt.Sprintf("%s %s@%s=%s", target, d.module, version, d.commit)
	if err := upsertTarget(path, target, d.module, line); err != nil {
		return d.fail(ExitRecord, msgTargetNotRecorded, target, path, fmt.Sprintf(whyNotWritten, path, strerror.Text(err)))
	}
	d.recordNote = fmt.Sprintf(noteTargetWritten, line, path, target)
	return proceed
}

// moduleRoot is the directory of gradlew: the module directory is under it (build.sh's <module-root>).
func (d *deploy) moduleRoot() string { return dirname(d.env("GRADLEW_PATH")) }

// upsertTarget puts line, the one of (target, module), into the file at path: in the place of the first line
// of that pair (a repeat of it is dropped), else after the last line. Every other line, comments and blank
// lines included, is kept as it was. The file is replaced atomically; a file that is not there is made.
func upsertTarget(path, target, module, line string) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var out []string
	placed := false
	if text := strings.TrimSuffix(string(data), "\n"); text != "" {
		for _, old := range strings.Split(text, "\n") {
			switch {
			case !namesPair(old, target, module):
				out = append(out, old)
			case !placed:
				out, placed = append(out, line), true
			}
		}
	}
	if !placed {
		out = append(out, line)
	}
	return atomicfile.Write(path, []byte(strings.Join(out, "\n")+"\n"))
}

// namesPair reports whether a line of the file is the line of target and module, read as build.sh reads it:
// the text before a "#", the first field the target, the second the module up to its "@".
func namesPair(line, target, module string) bool {
	fields := strings.Fields(strings.SplitN(line, "#", 2)[0])
	if len(fields) < 2 || fields[0] != target {
		return false
	}
	name, _, _ := strings.Cut(fields[1], "@")
	name, _, _ = strings.Cut(name, "=")
	return name == module
}
