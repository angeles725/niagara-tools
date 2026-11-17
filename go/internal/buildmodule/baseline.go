package buildmodule

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
	"github.com/angeles725/niagara-tools/go/internal/verifymodule"
)

// snapshot is what was deployed for one profile before gradle ran.
type snapshot struct {
	hash    string // content hash of the deployed jar, "" when it cannot be listed
	version string // its own vendorVersion, "" when unreadable
	backup  string // a full copy, restored into modules/ when the run must not leave the new jar there
	// signers are the names of its complete signatures (META-INF/<NAME>.SF with its block file), none when
	// it is unsigned or unreadable: what the gate expects of the new jar when no alias is configured.
	signers []string
}

// snapshotBaseline is part 1 of the deployed-baseline drift gate: gradle's :jar task installs the new jar
// into <niagara_home>/modules as its LAST step, so what is currently deployed is captured BEFORE gradle
// runs. A full copy is kept too, so a FAIL can restore modules/ to this exact baseline — without that, a bare
// re-run would snapshot the NEW jar as its own baseline, a deterministic rebuild would match it byte for
// byte, and the gate would silently pass on the retry (R4-drift-gate-not-retry-safe).
// [ev: client A retro "defrost sequencing and HMI reload" Δ2]
func (b *build) snapshotBaseline() int {
	if b.opt.skipDrift {
		b.unverified.Store(true) // gradle runs next: from here on modules/ may hold an unverified jar
		return proceed
	}
	dir, code := b.makeTemp(true)
	if code != 0 {
		return code
	}
	b.baseline, b.undeployed = map[string]snapshot{}, map[string]bool{}
	for _, p := range b.profiles {
		old := b.deployedJar(p)
		if !lint.IsFile(old) {
			b.undeployed[p] = !exists(old)
			continue
		}
		snap := snapshot{version: vendorversion.ModuleVersion(old), backup: dir + "/" + p + ".old.jar"}
		snap.hash, _ = b.contentHash(old)
		snap.signers = verifymodule.Signers(old)
		if reason, code := b.copyPreserving(old, snap.backup); code != 0 {
			b.warn("%s", reason)
			return code // the chain cannot go on without the backup; cp's own status, as under bash's set -e
		}
		b.baseline[p] = snap
	}
	b.unverified.Store(true) // the snapshot is whole and gradle runs next (a failed one ends the chain)
	return proceed
}

// contentHash is a deterministic hash of every jar entry outside META-INF/, in bytewise name order: its name,
// its size and its bytes, so bytes moved between entries or a renamed entry change the hash. META-INF/*
// (module.xml buildMillis, MANIFEST.MF, signatures) is excluded on purpose: those bytes change on every
// rebuild even when nothing shipped changed. ok is false when the jar is not a readable zip, holds no entry,
// holds two entries under one name, or one of its entries cannot be read in full (a CRC or decompression error): two unreadable jars would
// hash EQUAL and the gate would silently pass, so that is "cannot verify", never "unchanged". A jar holding
// only META-INF entries reads fine and hashes its (empty) payload (#226).
func (b *build) contentHash(jar string) (hash string, ok bool) {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return "", false
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return "", false
	}
	var payload []*zip.File
	seen := map[string]bool{}
	for _, entry := range zr.File {
		if seen[entry.Name] {
			return "", false // two entries under one name: which one ships is the reader's choice
		}
		seen[entry.Name] = true
		if !strings.HasPrefix(entry.Name, "META-INF/") {
			payload = append(payload, entry)
		}
	}
	sort.SliceStable(payload, func(i, j int) bool { return payload[i].Name < payload[j].Name })
	sum := sha256.New()
	for _, entry := range payload {
		fmt.Fprintf(sum, "%s\x00%d\x00", entry.Name, entry.UncompressedSize64)
		if !hashEntry(sum, entry) {
			return "", false
		}
	}
	return hex.EncodeToString(sum.Sum(nil)), true
}

// hashEntry adds the bytes of one entry to sum; false when they cannot be read in full (archive/zip checks
// the CRC and the size at the end of the entry).
func hashEntry(sum io.Writer, entry *zip.File) bool {
	r, err := entry.Open()
	if err != nil {
		return false
	}
	defer r.Close()
	_, err = io.Copy(sum, r)
	return err == nil
}

// copyPreserving is `cp -p src dst`: cp's exit status (0 copied) and, on failure, what cp printed — or why
// cp could not be run.
func (b *build) copyPreserving(src, dst string) (reason string, code int) {
	out, err := b.command("cp", "-p", src, dst).CombinedOutput()
	why, code := exitStatus(err)
	if why != "" {
		return fmt.Sprintf(msgCannotRun, "cp", why), code
	}
	return strings.TrimRight(string(out), "\n"), code
}

// restoreBaseline puts a profile's pre-build jar back into modules/ and says so; a profile without a
// snapshot is left alone. A failed copy prints cp's own reason and what modules/<jar> really holds now (the
// pre-build jar, or something else such as the new jar gradle's :jar step installed), because a bare retry
// would snapshot a new jar as its own baseline.
func (b *build) restoreBaseline(profile, restoredSuffix string) {
	b.modulesMu.Lock()
	defer b.modulesMu.Unlock()
	b.restoreBaselineHeld(profile, restoredSuffix)
}

// restoreBaselineHeld is restoreBaseline for a caller that holds modulesMu (the signal handler).
func (b *build) restoreBaselineHeld(profile, restoredSuffix string) {
	snap, ok := b.baseline[profile]
	if !ok {
		return
	}
	dst, name := b.deployedJar(profile), b.jarName(profile)
	reason, code := b.copyPreserving(snap.backup, dst)
	if code == 0 {
		b.warn(msgRestored, name, b.home, restoredSuffix)
		return
	}
	if reason == "" {
		reason = restoreNoReason
	}
	if sameContent(snap.backup, dst) {
		b.warn(msgRestoreFailedOld, name, b.home, reason)
	} else {
		b.warn(msgRestoreFailedNew, name, b.home, reason)
	}
}

// removeFirstDeploy takes the jar gradle's :jar step installed for a profile out of modules/ when none was
// deployed there before the build, and says so: the counterpart of restoreBaseline for a first deploy, where
// there is nothing to put back. Left in place, a jar that failed the gate would be the next run's baseline
// and its signer the expected one. A profile that had a jar, or a run without the snapshot
// (--no-drift-check), is left alone.
func (b *build) removeFirstDeploy(profile string) {
	b.modulesMu.Lock()
	defer b.modulesMu.Unlock()
	b.removeFirstDeployHeld(profile)
}

// removeFirstDeployHeld is removeFirstDeploy for a caller that holds modulesMu (the signal handler).
func (b *build) removeFirstDeployHeld(profile string) {
	dst := b.deployedJar(profile)
	if !b.undeployed[profile] || !lint.IsFile(dst) {
		return
	}
	if err := os.Remove(dst); err != nil {
		b.warn(msgRemoveFailed, b.jarName(profile), b.home, strerror.Text(err))
		return
	}
	b.warn(msgRemoved, b.jarName(profile), b.home)
}

// sameContent is cmp -s: both files readable and byte-identical.
func sameContent(a, b string) bool {
	x, errX := os.ReadFile(a)
	y, errY := os.ReadFile(b)
	return errX == nil && errY == nil && bytes.Equal(x, y)
}
