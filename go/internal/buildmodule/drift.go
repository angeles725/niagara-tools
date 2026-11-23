package buildmodule

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// driftVerdict is part 2 of the deployed-baseline drift gate. A new jar whose shipped bytes differ from the
// pre-build snapshot while the module's own vendorVersion is UNCHANGED fails: Software Manager compares
// versions, not bytes, and would report "Up to Date" and silently skip installing the jar (a client A
// control module shipped twice at one version, the same-version trap). A jar the gate cannot list, or a changed jar whose version cannot be
// read on either side, is UNVERIFIABLE — never "unchanged" and never "bumped". Either way the pre-build jar
// is restored into modules/, so a bare re-run sees the same baseline and catches it again.
// [ev: client A retro "defrost sequencing and HMI reload" Δ2]
func (b *build) driftVerdict() int {
	if b.opt.skipDrift {
		return proceed
	}
	drifted, unverified := false, false
	for _, p := range b.profiles {
		old, snapped := b.baseline[p]
		jar := b.builtJar(p)
		if !snapped || !lint.IsFile(jar) {
			continue
		}
		hash, _ := b.contentHash(jar)
		version := vendorversion.ModuleVersionFromZip(jar)
		if why := unverifiable(old, hash, version); why != "" {
			b.warn(msgCannotVerify, b.jarName(p), why)
			b.restoreBaseline(p, restoredPlain)
			unverified = true
			continue
		}
		if hash == old.hash {
			continue // nothing shipped actually changed
		}
		// An incomparable pair (a differing suffix) reads as a changed version: no FAIL.
		switch c, comparable := vendorversion.Cmp(version, old.version); {
		case !comparable:
		case c < 0:
			// A lower version is a WARN, not a FAIL: a deliberate rollback below the deployed jar is legitimate.
			b.warn(msgWentDown, b.jarName(p), old.version, version)
		case c == 0:
			b.warn(msgDrift, b.jarName(p), version)
			b.restoreBaseline(p, restoredRetry)
			drifted = true
		}
	}
	// A real drift outranks a cannot-verify profile of the same run: the bump it asks for is needed either
	// way, and the cannot-verify rows above stay on stderr.
	switch {
	case drifted:
		b.warn(msgDriftDetected)
		return ExitDrift
	case unverified:
		b.warn(msgDriftUnverified)
		return ExitDriftUnknown
	}
	return proceed
}

// unverifiable is why the gate cannot compare a new jar (its content hash and version) with the snapshot,
// "" when it can.
func unverifiable(old snapshot, hash, version string) string {
	switch {
	case hash == "" || old.hash == "":
		return whyUnlistable
	case hash != old.hash && (version == "" || old.version == ""):
		return fmt.Sprintf(whyNoVersion, old.version, version)
	}
	return ""
}
