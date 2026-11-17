package scaffoldmodule

import "regexp"

// The shapes of the version options: a Niagara release <major>.<minor> and a niagara-module plugin version
// <major>.<minor>.<patch>, digits only. A value of another shape would be written into settings.gradle.kts
// and only fail later, in gradle.
var (
	targetVersionShape = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	pluginVersionShape = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

// checkValues is the one gate of the option values a run writes into the emitted tree the vendor and the two
// versions, in that order: --vendor must be given (it has no default) and be one Java package segment (checkVendor);
// --target-version and --plugin-version must have their shape. Nothing is written before it passes.
func checkValues(o options) *failure {
	if !o.vendorGiven {
		return failf(ExitUsage, vendorRequired+usageText)
	}
	if f := checkVendor(o.vendor); f != nil {
		return f
	}
	if !targetVersionShape.MatchString(o.targetVersion) {
		return failf(ExitUsage, targetShape, o.targetVersion)
	}
	if !pluginVersionShape.MatchString(o.pluginVersion) {
		return failf(ExitUsage, pluginShape, o.pluginVersion)
	}
	return nil
}
