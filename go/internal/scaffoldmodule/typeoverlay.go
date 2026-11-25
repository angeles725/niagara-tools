package scaffoldmodule

// typeOverlay lays templates/<overlay> over the fixture copy of a type that has one (the service): the
// template's files sit below <Module>/<Module> (the module directory, the one MODULE-MAP.md is in), carry the
// template tokens and replace a fixture file of the same path, so the library fixture's plumbing (its
// -rt gradle file, module-include.xml, `type: library` map) becomes the service's own.
func typeOverlay(toolbelt, dest string, n names, typ moduleType) *failure {
	if typ.overlay == "" {
		return nil
	}
	root := toolbelt + "/../templates/" + typ.overlay
	if !isFile(root + "/" + serviceJobProbe) {
		return failf(ExitEnv, serviceJobMissing, root)
	}
	base := dest + "/" + n.module
	if err := copyTree(root, base, n.loginPathRules(), n.loginTextRules(), rewriteAll, nil); err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	return nil
}
