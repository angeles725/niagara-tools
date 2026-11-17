package buildmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/reportmodule"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// The kept-jar naming (reportmodule.ReplacedDir, FirstSuffix, KeptName) is shared with report-module --ready,
// which reads the first-replaced pointer to say whether the mirror holds the deployed jar again.
const (
	replacedDir = reportmodule.ReplacedDir
	firstSuffix = reportmodule.FirstSuffix
)

var keptName = reportmodule.KeptName

// keepReplaced is retro row B6: gradle's :jar step installs the new jar into <mirror>/modules, so after a
// build that the chain does not roll back the mirror no longer holds what the station runs, and the next
// build's drift gate would take the new jar as the deployed baseline. For every profile whose modules/ jar now
// differs from the pre-build snapshot, the snapshot is kept as <mirror>/.mirror-replaced/<jar>.<sha12> (it
// survives the run; the snapshot does not). A kept jar is never overwritten or deleted, so a module built twice
// keeps both replaced jars; the first one of the cycle (firstReplaced) is the one that matches the station, and
// the line names it with its restore command. Only a mirror (the .niagara-mirror marker) is touched; a run
// without a snapshot (--no-drift-check, a first deploy) or one that restored modules/ keeps nothing. A jar that
// cannot be kept is one WARN; the exit code never changes. It runs under modulesMu. [ev: retro buildsh-mirror-baseline-b6 Δ1]
func (b *build) keepReplaced() {
	// modulesMu: a signal handler restoring modules/ at this moment must finish first, or the restored jar
	// would be compared against a half-restored modules/ (retro p1-build-deploy-safety Δ11).
	b.modulesMu.Lock()
	defer b.modulesMu.Unlock()
	if b.home == "" || !b.isMirror() {
		return
	}
	for _, p := range b.profiles {
		snap, ok := b.baseline[p]
		now := b.deployedJar(p)
		if !ok || !lint.IsFile(now) || sameContent(snap.backup, now) {
			continue
		}
		dir, name := b.home+"/"+replacedDir, b.jarName(p)+".jar"
		oldSum := fileSum(snap.backup)
		if why := b.keep(snap.backup, dir, name, oldSum); why != "" {
			b.warn(msgKeepFailed, b.jarName(p), dir, why)
			continue
		}
		script := reportmodule.ShellWord(b.here + "/" + scriptName)
		newVersion, newSum := shown(vendorversion.ModuleVersion(now)), fileSum(now)
		first, why := b.firstReplaced(dir, name, oldSum)
		if why != "" {
			b.warn(msgFirstUnrecorded, b.jarName(p), dir+"/"+name+firstSuffix, why)
			b.say(msgReplacedNoFirst, b.jarName(p), b.home, shown(snap.version), oldSum, newVersion, newSum,
				reportmodule.ShellWord(dir+"/"+name+"."+oldSum), reportmodule.ShellWord(dir), script)
			continue
		}
		firstKept := dir + "/" + name + "." + first
		b.say(msgReplaced, b.jarName(p), b.home, shown(snap.version), oldSum, newVersion, newSum,
			shown(vendorversion.ModuleVersion(firstKept)), first, script, reportmodule.ShellWord(firstKept))
	}
}

// keep copies the pre-build jar to dir/<name>.<sum> unless a kept jar of that hash is already there; it returns
// why the jar is not kept, "" when it is.
func (b *build) keep(backup, dir, name, sum string) string {
	if sum == unreadable {
		return "the pre-build copy cannot be read"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return strerror.Text(err)
	}
	kept := dir + "/" + name + "." + sum
	if lint.IsFile(kept) {
		return "" // the same jar was replaced before: never overwrite a kept jar
	}
	if reason, code := b.copyPreserving(backup, kept); code != 0 {
		if reason == "" {
			reason = restoreNoReason
		}
		return reason
	}
	return ""
}

// firstReplaced is the hash of the first jar of name replaced this cycle, read from dir/<name>.first. A cycle
// starts at the first replacement with no live pointer (none, or one naming a kept jar that is gone) and ends
// when --restore-mirror puts a jar back; nothing else ends it, so after deploying a build to the station the
// operator deletes the pointer (the build.sh header says so). why is set when a new pointer cannot be written:
// then no jar can be vouched for as the first, and the caller must not name one.
func (b *build) firstReplaced(dir, name, sum string) (first, why string) {
	if first, ok := reportmodule.LiveFirst(dir, name); ok {
		return first, ""
	}
	pointer := dir + "/" + name + firstSuffix
	if err := os.WriteFile(pointer, []byte(sum+"\n"), 0o644); err != nil {
		return "", strerror.Text(err)
	}
	return sum, ""
}

// restoreMirror is --restore-mirror <kept>: it copies a jar keepReplaced kept back into the modules/ of the
// mirror that holds it, and builds nothing. It refuses anything but <mirror>/.mirror-replaced/<name>.<sha12>
// (exit 2), a kept jar that is not a file, whose sha256 prefix is not the one in its name, or a home without
// the .niagara-mirror marker (exit 10: a real install is never written); a failed copy is exit 1. The kept jar
// stays, so the restore can be repeated; the cycle's first-replaced pointer is removed.
func (b *build) restoreMirror(kept string) int {
	dir := filepath.Dir(kept)
	m := keptName.FindStringSubmatch(filepath.Base(kept))
	if m == nil || filepath.Base(dir) != replacedDir {
		b.warn(msgNotKept, kept)
		return ExitUsage
	}
	if !lint.IsFile(kept) {
		b.warn(msgNoKept, kept)
		return ExitEnv
	}
	if sum := fileSum(kept); sum != m[2] {
		b.warn(msgKeptChanged, kept, sum, m[2])
		return ExitEnv
	}
	home := filepath.Dir(dir)
	if !modjar.IsMirror(home) {
		b.warn(msgRestoreNotMirror, home)
		return ExitEnv
	}
	dst := home + "/modules/" + m[1]
	if reason, code := b.copyPreserving(kept, dst); code != 0 {
		if reason == "" {
			reason = restoreNoReason
		}
		b.warn(msgRestoreMirrorFailed, dst, reason)
		return ExitFault
	}
	_ = os.Remove(dir + "/" + m[1] + firstSuffix) // the cycle ends: the next replaced jar starts a new one
	b.say(msgMirrorRestored, home, m[1], kept, shown(vendorversion.ModuleVersion(dst)), fileSum(dst))
	return ExitOK
}

// unreadable is what fileSum gives for a file it cannot read.
const unreadable = "unreadable"

// fileSum is the first 12 hex digits of a file's sha256, unreadable when it cannot be read.
func fileSum(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return unreadable
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// shown is a vendorVersion as the mirror lines print it: "unknown" when it cannot be read.
func shown(version string) string {
	if strings.TrimSpace(version) == "" {
		return "unknown"
	}
	return version
}
