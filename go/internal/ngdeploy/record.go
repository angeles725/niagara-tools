package ngdeploy

import (
	"bytes"
	"fmt"
	"io"
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
	path = lastDeployFile
	if dir, err := os.Getwd(); err == nil {
		path = dir + "/" + lastDeployFile
	}
	return path, os.WriteFile(path, []byte(sha+"\n"), 0o644)
}
