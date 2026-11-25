package scaffoldmodule

const docTypeToken = "@DOCTYPE@"

// docRules are the template rules plus the registered name of the type the help page documents.
func (n names) docRules(typ moduleType) []rule {
	return append(n.loginTextRules(), rule{docTypeToken, n.module + typ.docSuffix})
}

// docOverlay adds templates/doc for --with-doc: the <Module>-doc part (its gradle file and src/doc: the TOC, the
// index and one Guide-on-Target page, doc/<Module>-<TypeName>.html, named by the registered type name, which
// has no B: types/structure.md § -doc help profile) and its row in MODULE-MAP.md, before the plumbing row.
func docOverlay(toolbelt, dest string, n names, typ moduleType, withDoc bool) *failure {
	if !withDoc {
		return nil
	}
	root := toolbelt + "/../" + docDir
	if !isFile(root + "/" + docProbe) {
		return failf(ExitEnv, docMissing, root)
	}
	base := dest + "/" + n.module
	pathRules := append(n.loginPathRules(), rule{docTypeToken, n.module + typ.docSuffix})
	if err := copyTree(root+"/files", base, pathRules, n.docRules(typ), rewriteAll, nil); err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	row := edit{file: "MODULE-MAP.md", anchor: "| plumbing |", fragment: "module-map-row.txt", name: "-doc row"}
	return row.apply(root, base, n, n.docRules(typ))
}
