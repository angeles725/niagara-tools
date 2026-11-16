package ngdeploy

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

// detection is what the annotation change heuristic found.
type detection int

const (
	noChange  detection = iota // the diff shows no annotation change, or the detection is off
	changed                    // a @Niagara annotation line was added or removed since the last deploy
	unchecked                  // the detection could not run: no git, not a repository, a failed diff, an unreadable record
)

// remindSlotomatic warns when Java annotations changed since the last deploy and --with-slotomatic was not
// given; in strict mode (--strict-slotomatic or SLOTOMATIC_DETECTION=strict) a change aborts the deploy,
// and so does a detection that could not run (under bash that passed like "no change"). --with-slotomatic
// satisfies strict mode: the slots are about to be regenerated.
func (d *deploy) remindSlotomatic() int {
	strict := (d.isOn(optStrictSlot) || d.env("SLOTOMATIC_DETECTION") == detectionStrict) && !d.isOn(optWithSlot)
	switch d.detectAnnotationChanges() {
	case changed:
		if strict {
			return d.fail(ExitSlotomatic, msgStrict)
		}
		if !d.isOn(optWithSlot) {
			for _, line := range warnAnnotations {
				d.warn(line)
			}
		}
	case unchecked:
		if strict {
			return d.fail(ExitSlotomatic, msgStrictUnchecked)
		}
	}
	return proceed
}

// detectAnnotationChanges reads `git diff <baseline>..HEAD` over the Java sources, in the working directory.
func (d *deploy) detectAnnotationChanges() detection {
	if d.env("SLOTOMATIC_DETECTION") == detectionOff {
		return noChange
	}
	if _, err := exec.LookPath("git"); err != nil {
		d.warn(noticeNoGit)
		return unchecked
	}
	if d.run(io.Discard, io.Discard, "git", "rev-parse", "--git-dir") != 0 {
		d.warn(noticeNoRepo)
		return unchecked
	}
	base, known := d.baseline()
	var diff bytes.Buffer
	status := d.run(&diff, io.Discard, "git", "diff", base+"..HEAD", "--", diffPathspec)
	for _, line := range strings.Split(diff.String(), "\n") {
		if annotationChange.MatchString(line) {
			return changed
		}
	}
	if status != 0 {
		d.warn(noticeDiffFailed, base, status)
		return unchecked
	}
	if !known {
		return unchecked
	}
	return noChange
}

// baseline is the commit the last deploy recorded in .last-deploy-sha when git knows it as a commit, else
// HEAD~1. known is false when a record is there but cannot be read, or holds something other than a full
// object name (a notice says so, and the text never reaches git, which would read a leading "-" as an
// option): the diff then starts at HEAD~1 and may miss older changes.
func (d *deploy) baseline() (base string, known bool) {
	is, err := regular(lastDeployFile)
	if err != nil || !is {
		return fallbackBaseline, err == nil
	}
	text, err := os.ReadFile(lastDeployFile)
	if err != nil {
		return fallbackBaseline, false
	}
	candidate := strings.TrimRight(string(text), "\n")
	switch {
	case candidate == "":
		return fallbackBaseline, true
	case !objectName.MatchString(candidate):
		d.warn(noticeBadRecord, lastDeployFile, fallbackBaseline)
		return fallbackBaseline, false
	case d.run(io.Discard, io.Discard, "git", "cat-file", "-e", candidate+"^{commit}") == 0:
		return candidate, true
	}
	return fallbackBaseline, true
}
