package scaffoldmodule

// options is the parsed command line; the option values are raw, as typed.
type options struct {
	module, outDir                            string
	vendor, targetVersion, pluginVersion, typ string
	vendorGiven                               bool // --vendor has no default: checkValues refuses a run without it
	withDoc                                   bool // --with-doc: add the -doc help part (doc.go)
}

// parseArgs reads `<ModuleName> <out-dir> --vendor v [--target-version v] [--plugin-version v]
// [--type t] [--with-doc]`. Options come after the two positionals; a repeated option keeps its last value and a value
// is taken as it is, even when it looks like an option.
func parseArgs(args []string) (options, *failure) {
	if len(args) < 2 {
		return options{}, failf(ExitUsage, usageText)
	}
	o := options{
		module: args[0], outDir: args[1],
		targetVersion: defaultTargetVersion, pluginVersion: defaultPluginVersion, typ: defaultType,
	}
	valued := map[string]*string{
		optVendor: &o.vendor, optTargetVersion: &o.targetVersion, optPluginVersion: &o.pluginVersion, optType: &o.typ,
	}
	pluginGiven := false
	for rest := args[2:]; len(rest) > 0; {
		if rest[0] == optWithDoc {
			o.withDoc, rest = true, rest[1:]
			continue
		}
		target, known := valued[rest[0]]
		if !known {
			return options{}, failf(ExitUsage, unknownOption+usageText, rest[0])
		}
		if len(rest) < 2 {
			return options{}, failf(ExitUsage, usageText)
		}
		*target = rest[1]
		o.vendorGiven = o.vendorGiven || rest[0] == optVendor
		pluginGiven = pluginGiven || rest[0] == optPluginVersion
		rest = rest[2:]
	}
	if p, known := pluginForTarget[o.targetVersion]; known && !pluginGiven {
		o.pluginVersion = p
	}
	return o, nil
}
