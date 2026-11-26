package buildmodule

import "github.com/angeles725/niagara-tools/go/internal/vendorversion"

// driftAgainstRecord is the same-version case of the drift gate for a run with --target: the mirror's jar and the
// new one carry one vendorVersion and differ in bytes. A recorded deploy says what a station runs, which the mirror
// (a build) never does, so it decides in the place of the mirror's own-build record: when the version being shipped
// is the recorded one, the station would report "Up to Date" and skip the jar (a drift, even if the mirror's jar is
// this checkout's own build); when it is another, no station has this version and nothing is shipped under one it
// knows (an INFO line, and the new jar stays). The record names a version, not bytes, so the bytes are still those
// of the mirror's jar. A version that cannot be ordered against the record is another version.
func (b *build) driftAgainstRecord(profile, version string) (drifted bool) {
	if c, ok := vendorversion.Cmp(version, b.deployed.version); ok && c == 0 {
		b.warn(msgDrift, b.jarName(profile), version)
		b.restoreBaseline(profile, restoredRetry)
		return true
	}
	b.warn(msgRecordedElsewhere, deployTargetsFile, b.opt.mod, b.deployed.version, b.opt.deployTarget, version, b.jarName(profile))
	return false
}
