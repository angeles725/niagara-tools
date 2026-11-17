package buildmodule

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// runGradle runs ./gradlew in the gradle root with the tasks of every selected profile, on Java 8 (a
// `gradle :jar` with the default JDK is not a build: wrong bytecode major, slotomatic skipped). Gradle's
// stdout and stderr both go to the chain's stdout, in order, and are kept for the diagnosis of the run.
func (b *build) runGradle() int {
	tasks := b.tasks()
	b.say(msgBuild, strings.Join(tasks, " "))
	started, code := b.startStamp()
	if code != 0 {
		return code
	}
	var log bytes.Buffer
	stream := io.MultiWriter(b.s.Out, &log)
	cmd := b.command("./gradlew", append(tasks, b.gradleArgs...)...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = b.gradleRoot, stream, stream
	if wd, err := os.Getwd(); err == nil {
		cmd.Env = b.environ("OLDPWD="+wd, "PWD="+b.gradleRoot) // as after bash's `cd <gradle root>`
	}
	err := b.runTracked(cmd)
	if err == nil {
		return b.afterGradlePassed(log.String())
	}
	if why, _ := exitStatus(err); why != "" {
		fmt.Fprintf(stream, msgCannotRun+"\n", cmd.Path, why) // gradlew could not be started
	}
	return b.afterGradleFailed(log.String(), started)
}

// startStamp marks the start of the build with the modification time of a temporary file created now: a
// build/libs jar NEWER than it was produced by this run. A file, not the clock, so both times come from the
// filesystem. code is not 0 when the file cannot be made.
func (b *build) startStamp() (started time.Time, code int) {
	stamp, code := b.makeTemp(false)
	if code != 0 {
		return started, code
	}
	st, err := os.Stat(stamp)
	if err != nil {
		b.warn(msgFault, err)
		return started, ExitFault
	}
	return st.ModTime(), 0
}

// builtSince reports whether path exists and was modified after started (find <path> -newer <stamp>).
func builtSince(path string, started time.Time) bool {
	st, err := os.Lstat(path)
	return err == nil && st.ModTime().After(started)
}
