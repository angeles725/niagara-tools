package scaffoldmodule

// options is the parsed command line; the option values are raw, as typed.
type options struct {
	module, outDir                            string
	vendor, targetVersion, pluginVersion, typ string
}

// parseArgs reads `<ModuleName> <out-dir> [--vendor v] [--target-version v] [--plugin-version v]
// [--type t]`. Options come after the two positionals; a repeated option keeps its last value and a value
// is taken as it is, even when it looks like an option.
func parseArgs(args []string) (options, *failure) {
	if len(args) < 2 {
		return options{}, failf(ExitUsage, usageText)
	}
	o := options{
		module: args[0], outDir: args[1],
		vendor: defaultVendor, targetVersion: defaultTargetVersion, pluginVersion: defaultPluginVersion, typ: defaultType,
	}
	valued := map[string]*string{
		optVendor: &o.vendor, optTargetVersion: &o.targetVersion, optPluginVersion: &o.pluginVersion, optType: &o.typ,
	}
	for rest := args[2:]; len(rest) > 0; rest = rest[2:] {
		target, known := valued[rest[0]]
		if !known {
			return options{}, failf(ExitUsage, unknownOption+usageText, rest[0])
		}
		if len(rest) < 2 {
			return options{}, failf(ExitUsage, usageText)
		}
		*target = rest[1]
	}
	return o, nil
}
