package scaffoldmodule

import "strings"

// checkValues is the one gate of the option values a run writes into the emitted tree (substitutedOptions):
// --vendor must be one Java package segment (checkVendor); every other value is literal text and is only
// refused when it holds a newline, which no emitted line can carry.
func checkValues(o options) *failure {
	if f := checkVendor(o.vendor); f != nil {
		return f
	}
	typed := map[string]string{optTargetVersion: o.targetVersion, optPluginVersion: o.pluginVersion}
	for _, option := range substitutedOptions {
		if v, literal := typed[option]; literal && strings.Contains(v, "\n") {
			return failf(ExitUsage, valueNewline, option, v)
		}
	}
	return nil
}
