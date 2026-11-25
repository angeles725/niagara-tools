package ngdeploy

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// nextStep is one closing line of a deploy that went through.
type nextStep struct {
	java bool   // only for a mode that deploys Java classes (the -rt jar): the station must restart
	text string // {jars}, {done}, {dir}, {doneTail}, {paths}, {backup}, {signers}, {module}, {triage}, and {lab} are filled in (closingValues)
}

// nextSteps are the closing lines, in order: the one table of what the author does after a deploy. They
// are printed only by a deploy that went through (exit 0), after the restart or reload line.
var nextSteps = []nextStep{
	{text: "done: {jars} {done} {dir}{doneTail}"},
	{text: "signer: {signers}; the gate checks who signed a jar, not whether the station trusts that signer: the station decides when it loads the module"},
	{java: true, text: "next: restart the station; ng-deploy did not look at whether one is running, and stops or starts none"},
	{java: true, text: "next: stop it with: {lab} stop --home <NIAGARA_HOME> <station>; start it with: {lab} start --home <NIAGARA_HOME> <station> (lab-station stops only a station it started; a Windows service or station.exe is stopped and started there)"},
	{java: true, text: "next: after the restart, run the mcp-n4 prompt post-deploy-check with station=<station> module={module} (read-only: restart_needed, module version, health, new alarms)"},
	{java: true, text: "next: after the restart, check the station console for the module: {triage} --package <the Java package of the module> --module-prefix {module} <station-dir>/console*.txt (exit 1: rows to look at)"},
	{text: "undo: to remove the module again, take its components out of the station, stop the station and delete {paths}{backup}"},
}

// The words of the signer line, and the console triage tool beside the scripts directory.
const (
	jarSignedBy    = "%s signed by %s"
	jarNotSigned   = "%s carries no complete signature"
	signerJoin     = " and "
	triageToolPath = "/../build-n4-module-kit/toolbelt/triage-console.sh"
	labToolPath    = "/../build-n4-module-kit/toolbelt/lab-station.sh"
	// doneCopied and doneSame are how the done line says the jars reached the station directory: copied, or
	// in the same-directory mode installed there by the build and only verified (#993 D11).
	doneCopied   = "copied to"
	doneSame     = "installed by the build in"
	doneSameTail = " and verified there (nothing to copy)"
	// undoBackup ends the undo line of a run that wrote a backup archive.
	undoBackup = " (for the version before this one, restore the backup this run wrote: --help, Backups)"
)

// sayNextSteps is the last phase: it prints the closing lines for the jars of the mode.
func (d *deploy) sayNextSteps() int {
	values := d.closingValues()
	for _, step := range nextSteps {
		if step.java && !d.hasProfile("rt") {
			continue
		}
		d.say("%s", values.Replace(step.text))
	}
	return proceed
}

// closingValues are what the closing lines name: the jars of the mode, the station directory and the jars
// in it, who signed each (modjar.Signers reads the built jar, the bytes that were copied), the module and
// the triage tool.
func (d *deploy) closingValues() *strings.Replacer {
	var names, paths, signers []string
	for _, built := range d.builtJars() {
		name := basename(built)
		names, paths = append(names, name), append(paths, d.station+"/"+name)
		if by := modjar.Signers(built); len(by) > 0 {
			signers = append(signers, fmt.Sprintf(jarSignedBy, name, strings.Join(by, signerJoin)))
		} else {
			signers = append(signers, fmt.Sprintf(jarNotSigned, name))
		}
	}
	backup := ""
	if d.backedUp {
		backup = undoBackup
	}
	done, doneTail := doneCopied, ""
	if d.sameDir {
		done, doneTail = doneSame, doneSameTail
	}
	return strings.NewReplacer("{backup}", backup, "{done}", done, "{doneTail}", doneTail,
		"{jars}", strings.Join(names, " "), "{dir}", d.station, "{paths}", strings.Join(paths, " "),
		"{signers}", strings.Join(signers, ", "), "{module}", d.module, "{triage}", d.scriptDir+triageToolPath,
		"{lab}", d.scriptDir+labToolPath)
}
