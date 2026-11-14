package buildmodule

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// snapshot is what was deployed for one profile before gradle ran.
type snapshot struct {
	hash    string // content hash of the deployed jar, "" when it cannot be listed
	version string // its own vendorVersion, "" when unreadable
	backup  string // a full copy, restored into modules/ when the run must not leave the new jar there
}

// snapshotBaseline is part 1 of the deployed-baseline drift gate: gradle's :jar task installs the new jar
// into <niagara_home>/modules as its LAST step, so what is currently deployed is captured BEFORE gradle
// runs. A full copy is kept too, so a FAIL can restore modules/ to this exact baseline — without that, a bare
// re-run would snapshot the NEW jar as its own baseline, a deterministic rebuild would match it byte for
// byte, and the gate would silently pass on the retry (R4-drift-gate-not-retry-safe).
// [ev: client A retro "defrost sequencing and HMI reload" Δ2]
func (b *build) snapshotBaseline() int {
	if b.opt.skipDrift {
		return proceed
	}
	dir, code := b.makeTemp(true)
	if code != 0 {
		return code
	}
	b.baseline = map[string]snapshot{}
	for _, p := range b.profiles {
		old := b.deployedJar(p)
		if !lint.IsFile(old) {
			continue
		}
		snap := snapshot{version: vendorversion.ModuleVersion(old), backup: dir + "/" + p + ".old.jar"}
		snap.hash, _ = b.contentHash(old)
		if reason, code := b.copyPreserving(old, snap.backup); code != 0 {
			b.warn("%s", reason)
			return code // the chain cannot go on without the backup; cp's own status, as under bash's set -e
		}
		b.baseline[p] = snap
	}
	return proceed
}

// contentHash is a deterministic hash of the CONTENT of every jar entry outside META-INF/, in bytewise name
// order. META-INF/* (module.xml buildMillis, MANIFEST.MF, signatures) is excluded on purpose: those bytes
// change on every rebuild even when nothing shipped changed. ok is false when the jar cannot be listed
// (unzip missing or failing, an unreadable or empty jar): two empty listings would hash EQUAL and the gate
// would silently pass, so an unlistable jar is "cannot verify", never "unchanged". A jar holding only
// META-INF entries lists fine and hashes its (empty) payload (#226).
func (b *build) contentHash(jar string) (hash string, ok bool) {
	listing, listed := b.output("unzip", "-Z1", jar)
	names := strings.TrimRight(string(listing), "\n")
	if !listed || names == "" {
		return "", false
	}
	var payload []string
	for _, name := range strings.Split(names, "\n") {
		if !strings.HasPrefix(name, "META-INF/") {
			payload = append(payload, name)
		}
	}
	sort.Strings(payload)
	sum := sha256.New()
	for _, name := range payload {
		content, _ := b.output("unzip", "-p", jar, name) // an entry unzip cannot print adds what it did print
		sum.Write(content)
	}
	return hex.EncodeToString(sum.Sum(nil)), true
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

// sameContent is cmp -s: both files readable and byte-identical.
func sameContent(a, b string) bool {
	x, errX := os.ReadFile(a)
	y, errY := os.ReadFile(b)
	return errX == nil && errY == nil && bytes.Equal(x, y)
}
