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

// firstSuffix names the pointer to the first jar a cycle replaced: <name>.first beside the kept jars.
const firstSuffix = ".first"

// keptName is the name of a kept jar: the modules/ file name, then its hash.
var keptName = regexp.MustCompile(`^(.+\.jar)\.([0-9a-f]{12})$`)

// keepReplaced is retro row B6: gradle's :jar step installs the new jar into <mirror>/modules, so after a
// build that the chain does not roll back the mirror no longer holds what the station runs, and the next
// build's drift gate would take the new jar as the deployed baseline. For every profile whose modules/ jar now
// differs from the pre-build snapshot, the snapshot is kept as <mirror>/.mirror-replaced/<jar>.<sha12> (it
// survives the run; the snapshot does not). A kept jar is never overwritten or deleted, so a module built twice
// keeps both replaced jars; the first one of the cycle (firstReplaced) is the one that matches the station, and
// the line names it with its restore command. Only a mirror (the .niagara-mirror marker) is touched; a run
// without a snapshot (--no-drift-check, a first deploy) or one that restored modules/ keeps nothing. A jar that
// cannot be kept is one WARN; the exit code never changes. [ev: retro buildsh-mirror-baseline-b6 Δ1]
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
		dir, name := b.home+"/"+replacedDir, b.jarName(p)+".jar"
		oldSum := fileSum(snap.backup)
		if why := b.keep(snap.backup, dir, name, oldSum); why != "" {
			b.warn(msgKeepFailed, b.jarName(p), dir, why)
			continue
		}
		first := b.firstReplaced(dir, name, oldSum)
		firstKept := dir + "/" + name + "." + first
		b.say(msgReplaced, b.jarName(p), b.home, shown(snap.version), oldSum, shown(vendorversion.ModuleVersion(now)),
			fileSum(now), shown(vendorversion.ModuleVersion(firstKept)), first, b.here+"/"+scriptName, firstKept)
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
// when --restore-mirror puts a jar back. A pointer that cannot be written leaves sum as the answer.
func (b *build) firstReplaced(dir, name, sum string) string {
	pointer := dir + "/" + name + firstSuffix
	if data, err := os.ReadFile(pointer); err == nil {
		if first := strings.TrimSpace(string(data)); keptName.MatchString(name+"."+first) && lint.IsFile(dir+"/"+name+"."+first) {
			return first
		}
	}
	_ = os.WriteFile(pointer, []byte(sum+"\n"), 0o644)
	return sum
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
