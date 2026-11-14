package ngdeploy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/slotomaticskip"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// slotomatic is the step before the build for a mode with Java (A, C): the annotation reminder, then the
// slotomatic run when --with-slotomatic. Mode B has no -rt profile: the flag is ignored with a warning.
func (d *deploy) slotomatic() int {
	if !d.hasProfile("rt") {
		if d.isOn(optWithSlot) {
			for _, line := range warnSlotModeB {
				d.warn(line)
			}
		}
		return proceed
	}
	if code := d.remindSlotomatic(); code != proceed || !d.isOn(optWithSlot) {
		return code
	}
	return d.runSlotomatic()
}

// runSlotomatic runs :MODULE-rt:slotomatic, and :MODULE-ux:slotomatic too when the mode has a -ux profile
// whose sources carry a @Niagara annotation. Both gradle streams are shown on stdout. A gradle that fails,
// or that passes after Slot-o-Matic skipped a file (METHODOLOGY K15), is exit 15.
func (d *deploy) runSlotomatic() int {
	tasks := []string{":" + d.module + "-rt:slotomatic"}
	if d.hasProfile("ux") {
		annotated, code := d.uxHasAnnotations()
		if code != proceed {
			return code
		}
		if annotated {
			tasks = append(tasks, ":"+d.module+"-ux:slotomatic")
		}
	}
	if len(tasks) > 1 {
		d.say(msgSlotRTUX, d.module, d.module)
	} else {
		d.say(msgSlotRT, d.module)
	}
	var log bytes.Buffer
	shown := io.MultiWriter(d.s.Out, &log)
	if d.run(shown, shown, d.env("GRADLEW_PATH"), d.gradleArgs(tasks)...) != 0 {
		return d.fail(ExitSlotomatic, msgSlotFailed)
	}
	if rows := slotomaticskip.Scan(log.String()); rows != nil {
		fmt.Fprintln(d.s.Err, strings.Join(rows, "\n"))
		return d.fail(ExitSlotomatic, msgSlotSkipped)
	}
	d.say(msgSlotOK)
	return proceed
}

// uxHasAnnotations reports whether a .java file under the -ux profile's src directory holds a @Niagara
// annotation: presence, not a diff, because a fresh checkout has no diff and its slots still need
// generating. Sources that cannot be read are refused (under bash they read as "no annotation").
func (d *deploy) uxHasAnnotations() (found bool, code int) {
	src := d.profileDir("ux") + "/src"
	if !isDir(src) {
		return false, proceed
	}
	var unread string
	note := func(path string, err error) {
		if unread == "" {
			unread = fmt.Sprintf(msgUxScan, path, strerror.Text(err))
		}
	}
	findwalk.Walk(src, true, findwalk.NoLimit, func(e findwalk.Entry) bool {
		if found || e.Dir || !strings.HasSuffix(e.Name, javaSuffix) {
			return !found
		}
		if st, err := os.Lstat(e.Path); err != nil || !st.Mode().IsRegular() {
			return true // grep -r reads regular files only
		}
		text, err := os.ReadFile(e.Path)
		if err != nil {
			note(e.Path, err)
		}
		found = annotation.Match(text)
		return true
	}, note)
	if !found && unread != "" {
		return false, d.fail(ExitSlotomatic, "%s", unread)
	}
	return found, proceed
}
