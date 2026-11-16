package ngdeploy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// recordDeploy writes the deployed commit (git rev-parse HEAD) to .last-deploy-sha in the working
// directory, the baseline of the next annotation diff, then says what the operator must restart. The jars
// are in the station by then, so the run goes on to that line whatever happens; a commit that could not be
// read or recorded is an ERROR line and exit 41. With SLOTOMATIC_DETECTION=off nothing reads the record: it
// stays best effort, and a file that cannot be written is only reported.
func (d *deploy) recordDeploy() int {
	code := proceed
	optional := d.env("SLOTOMATIC_DETECTION") == detectionOff
	sha, why := d.headCommit()
	if why == "" {
		if path, err := writeRecord(sha); err != nil {
			why = fmt.Sprintf(whyNotWritten, path, strerror.Text(err))
			if optional {
				fmt.Fprintf(d.s.Err, msgCannotRun+"\n", path, strerror.Text(err))
			}
		}
	}
	if why != "" && !optional {
		code = d.fail(ExitRecord, msgRecordFailed, lastDeployFile, why)
	}
	if d.hasProfile("rt") {
		d.say(msgRestart)
	} else {
		d.say(msgHardReload)
	}
	return code
}

// checkRecord looks, before the copy, whether the record can be written at all: a .last-deploy-sha that is
// a directory or read-only, or a working directory that takes no new file, stops the deploy with exit 10
// while the station still holds the old jars. Nothing is written: a record that is there is only opened,
// and one the look had to create is removed again. With SLOTOMATIC_DETECTION=off the record is optional
// and nothing is looked at.
func (d *deploy) checkRecord() int {
	if d.env("SLOTOMATIC_DETECTION") == detectionOff {
		return proceed
	}
	path := recordPath()
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	created := false
	if errors.Is(err, fs.ErrNotExist) {
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		created = err == nil
	}
	if err != nil {
		return d.fail(ExitEnv, msgRecordUnwritable, lastDeployFile, fmt.Sprintf(whyNotWritten, path, strerror.Text(err)))
	}
	file.Close()
	if created {
		os.Remove(path)
	}
	return proceed
}

// headCommit is the full object name of HEAD. why says what stopped the read; "" when nothing did. Only a
// full object name is ever recorded: the next deploy hands the record to git.
func (d *deploy) headCommit() (sha, why string) {
	var head bytes.Buffer
	if status := d.run(&head, io.Discard, "git", "rev-parse", "HEAD"); status != 0 {
		return "", fmt.Sprintf(whyNoHead, status)
	}
	if sha = strings.TrimRight(head.String(), "\n"); !objectName.MatchString(sha) {
		return "", whyHeadNotAName
	}
	return sha, ""
}

// writeRecord writes the commit to .last-deploy-sha in the working directory; path names the file in a
// message.
func writeRecord(sha string) (path string, err error) {
	path = recordPath()
	return path, os.WriteFile(path, []byte(sha+"\n"), 0o644)
}

// recordPath is .last-deploy-sha in the working directory, as a message names it.
func recordPath() string {
	if dir, err := os.Getwd(); err == nil {
		return dir + "/" + lastDeployFile
	}
	return lastDeployFile
}
