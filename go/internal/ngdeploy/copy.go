package ngdeploy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// copyJars copies the built jars of the mode into the station modules directory with cp, one by one: it
// runs only after every precheck, verify and the gate passed. A cp that fails is exit 40 and stops there.
func (d *deploy) copyJars() int {
	for _, profile := range profilesOf[d.raw[optMode]] {
		if d.show("cp", d.builtJar(profile), d.station+"/") != 0 {
			return d.fail(ExitCopy, msgCopyFailed, d.jarName(profile))
		}
	}
	d.say(msgCopyOK)
	return proceed
}

// recordDeploy writes the deployed commit (git rev-parse HEAD) to .last-deploy-sha in the working
// directory, the baseline of the next annotation diff, then says what the operator must restart. The
// record is not critical: without git, or a commit, nothing is written, and a file that cannot be written
// is reported and leaves the deploy a success, as in the bash tool.
func (d *deploy) recordDeploy() int {
	var head bytes.Buffer
	d.run(&head, io.Discard, "git", "rev-parse", "HEAD")
	if sha := strings.TrimRight(head.String(), "\n"); sha != "" {
		path := lastDeployFile
		if dir, err := os.Getwd(); err == nil {
			path = dir + "/" + lastDeployFile
		}
		if err := os.WriteFile(path, []byte(sha+"\n"), 0o644); err != nil {
			fmt.Fprintf(d.s.Err, msgCannotRun+"\n", path, strerror.Text(err))
		}
	}
	if d.hasProfile("rt") {
		d.say(msgRestart)
	} else {
		d.say(msgHardReload)
	}
	return proceed
}
