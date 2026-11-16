package buildmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// replacedDir is where a mirror keeps the modules/ jars its builds replaced, named <MOD>-<profile>.jar.<sha12>
// after the first 12 hex digits of the replaced jar's sha256.
const replacedDir = ".mirror-replaced"

// keptName is the name of a kept jar: the modules/ file name, then its hash.
var keptName = regexp.MustCompile(`^(.+\.jar)\.([0-9a-f]{12})$`)

// keepReplaced is retro row B6: gradle's :jar step installs the new jar into <mirror>/modules, so after a
// build that the chain does not roll back the mirror no longer holds what the station runs, and the next
// build's drift gate would take the new jar as the deployed baseline. For every profile whose modules/ jar now
// differs from the pre-build snapshot, the snapshot is copied to <mirror>/.mirror-replaced/ (it survives the
// run; the snapshot does not) and one line names both versions and hashes and the command that puts it back.
// Only a mirror (the .niagara-mirror marker) is touched; a run without a snapshot (--no-drift-check, a
// first deploy) or one that restored modules/ keeps nothing. [ev: client A module build 2026-11-16]
func (b *build) keepReplaced() {
	if b.home == "" || !b.isMirror() {
		return
	}
	for _, p := range b.profiles {
		snap, ok := b.baseline[p]
		now := b.deployedJar(p)
		if !ok || !lint.IsFile(now) || sameContent(snap.backup, now) {
			continue
		}
		oldSum, newSum := fileSum(snap.backup), fileSum(now)
		dir := b.home + "/" + replacedDir
		kept := dir + "/" + b.jarName(p) + ".jar." + oldSum
		if oldSum == unreadable {
			b.warn(msgKeepFailed, b.jarName(p), dir, "the pre-build copy cannot be read")
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.warn(msgKeepFailed, b.jarName(p), dir, strerror.Text(err))
			continue
		}
		if reason, code := b.copyPreserving(snap.backup, kept); code != 0 {
			b.warn(msgKeepFailed, b.jarName(p), dir, reason)
			continue
		}
		b.say(msgReplaced, b.jarName(p), b.home, shown(snap.version), oldSum, shown(vendorversion.ModuleVersion(now)),
			newSum, b.here+"/"+scriptName, kept)
	}
}

// restoreMirror is --restore-mirror <kept>: it copies a jar keepReplaced kept back into the modules/ of the
// mirror that holds it, and builds nothing. It refuses anything but <mirror>/.mirror-replaced/<name>.<sha12>
// (exit 2), a kept jar that is not a file or a home without the .niagara-mirror marker (exit 10: a real
// install is never written); a failed copy is exit 1. The kept jar stays, so the restore can be repeated.
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
