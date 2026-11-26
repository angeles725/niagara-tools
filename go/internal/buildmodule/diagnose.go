package buildmodule

import (
	"os"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/signblock"
	"github.com/angeles725/niagara-tools/go/internal/slotomaticskip"
)

// afterGradlePassed reads the log of a passing run before anything trusts the jar. Slot-o-Matic logs a file
// it could not process at WARNING level and gradle still ends BUILD SUCCESSFUL, so the jar then carries
// stale generated code: that FAILs, the pre-build modules/ jars are restored as for a drift FAIL, and the
// gate is not run (METHODOLOGY K15). [ev: retro r10-slotomatic-skip-fail-open Δ1]
func (b *build) afterGradlePassed(log string) int {
	if rows := slotomaticskip.Scan(log); len(rows) > 0 {
		b.warn("%s", strings.Join(rows, "\n"))
		b.warn(msgSkippedFile)
		for _, p := range b.profiles {
			b.restoreBaseline(p, restoredPlain)
		}
		return ExitSlotomatic
	}
	b.warnDevKey()
	return proceed
}

// warnDevKey says when the jar was signed with a throwaway dev key: a root without a signing block makes the
// plugin create a profile in the mirror user home and sign with it, and the build still passes. Every
// profile there is plugin-created (never the user's) and it persists, so this warns on every build, not only
// the first (#316). A root whose uncommented code turns the default profile off (signblock.Parts[0], the
// lint-structure L15 rule) cannot fall back to it, so a leftover profile is no evidence there: no warning.
// An unreadable root keeps the warning.
func (b *build) warnDevKey() {
	if b.signingXML == "" || !exists(b.signingXML) {
		return
	}
	if root := b.gradleRoot + "/build.gradle.kts"; lint.IsFile(root) && disablesDefaultProfile(root) {
		return
	}
	b.warn(msgDevKey, b.signingXML)
}

// disablesDefaultProfile reports whether the code of a root build.gradle.kts, comments removed, turns the
// signing plugin's default profile off.
func disablesDefaultProfile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	lines, err := javasrc.Lines(f)
	if err != nil {
		return false
	}
	code, _ := javasrc.Strip(lines)
	for _, line := range code {
		if signblock.Parts[0].Re.MatchString(line) {
			return true
		}
	}
	return false
}

// afterGradleFailed names what the log of a failed run shows and returns the exit code of its cause.
func (b *build) afterGradleFailed(log string, started time.Time) int {
	switch {
	case logCopyLocked.MatchString(log):
		// The plugin's post-jar copy into <niagara_home>/modules failed because a (Windows-side) process
		// holds the old jar open; the jar task itself already assembled build/libs/<jar>.
		// [ev: client A retro "restart sequence, lockout and hours" Δ3]
		b.warn(msgCopyLocked, b.home)
		for _, p := range b.profiles {
			if jar := b.builtJar(p); builtSince(jar, started) {
				b.warn(msgJarBuilt, jar)
			} else {
				b.warn(msgJarStale, jar)
			}
		}
		b.warn(msgCopyLockedNext, b.home)
		return ExitCopyLocked
	case logCleanLocked.MatchString(log):
		// A running station LOCKS modules/<mod>.jar so :clean fails: tell the operator how to free it.
		b.warn(msgCleanLocked)
		return ExitCleanLocked
	}
	if strings.Contains(log, logSlotErrors) {
		// Slot-o-Matic's own task succeeded after skipping a file, so the failure surfaces later in
		// compileJava ("… have you run slot-o-matic?"). Name the skipped file. (METHODOLOGY K15)
		b.warn(msgSlotErrors)
		if rows := slotomaticskip.Scan(log); len(rows) > 0 {
			b.warn("%s", strings.Join(rows, "\n"))
			b.warn(msgSlotErrorsFile)
		} else {
			b.warn(msgSlotErrorsRerun)
		}
	}
	if m := logNoPrivateKey.FindStringSubmatch(log); m != nil {
		b.explainNoPrivateKey(m[1])
	}
	if b.opt.werror {
		b.warn(msgWerror)
	}
	b.warn(msgGradleFailed)
	return ExitGradle
}
