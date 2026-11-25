package scaffoldmodule

import (
	"os"
	"strings"
)

// loginOverlay adds templates/dashboard-login to the -rt part of a type that ships its own login (the
// dashboard; types/dashboard.md § The module ships its own login): the BLoginTemplate subclass, its pure
// classes and tests, the asset prefix, the service that applies it at station start. The files are copied
// from templates/dashboard-login/files; the fixture's plumbing (module-include.xml, module.palette, the -rt
// gradle file, module.lexicon, MODULE-MAP.md) then receives the fragments of templates/dashboard-login/fragments
// at the places loginEdits names, so the fixture itself stays the same for every type.
func loginOverlay(toolbelt, dest string, n names, typ moduleType) *failure {
	if !typ.ownLogin {
		return nil
	}
	root := toolbelt + "/../" + loginDir
	if !isFile(root + "/" + loginProbe) {
		return failf(ExitEnv, loginMissing, root)
	}
	base := dest + "/" + n.module
	if err := copyTree(root+"/files", base+"/"+n.module+"-rt", n.loginPathRules(), n.loginTextRules(), rewriteAll); err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	for _, e := range loginEdits {
		if f := e.apply(root, base, n, n.loginTextRules()); f != nil {
			return f
		}
	}
	return nil
}

// edit puts one fragment into an emitted file: before the anchor, after it, or (no anchor) at the end of the
// file. The file and the anchor are written in template terms (ModuleA, com.example): the run's names are
// substituted into both, so the anchor is the emitted text.
type edit struct {
	file, anchor, fragment string
	after                  bool
	name                   string // what the edit adds, for the anchor error ("" = the login)
}

func what(e edit) string {
	if e.name == "" {
		return "login"
	}
	return e.name
}

// loginEdits are the fragments of templates/dashboard-login/fragments; files are relative to <Module>/<Module>.
var loginEdits = []edit{
	{file: "ModuleA-rt/module-include.xml", anchor: "</types>", fragment: "module-include.xml"},
	{file: "ModuleA-rt/module.palette", anchor: "</p>\n</bajaObjectGraph>", fragment: "module.palette"},
	{file: "ModuleA-rt/ModuleA-rt.gradle.kts", anchor: "  api(\":baja\")\n", fragment: "gradle-deps.txt", after: true},
	{file: "ModuleA-rt/ModuleA-rt.gradle.kts", anchor: "  includePackage(\"com.example.ModuleA\")\n", fragment: "gradle-bajadoc.txt", after: true},
	{file: "ModuleA-rt/ModuleA-rt.gradle.kts", fragment: "gradle-jar.txt"},
	{file: "ModuleA-rt/module.lexicon", fragment: "module.lexicon"},
	{file: "MODULE-MAP.md", anchor: "| plumbing |", fragment: "module-map-row.txt"},
}

func (e edit) apply(root, base string, n names, rules []rule) *failure {
	path := base + "/" + apply(e.file, n.templatePathRules())
	data, err := os.ReadFile(path)
	if err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	frag, err := os.ReadFile(root + "/" + loginFragmentDir + "/" + e.fragment)
	if err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	text, s := apply(string(frag), rules), string(data)
	if e.anchor == "" {
		if s != "" && !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		s += text
	} else {
		anchor := apply(e.anchor, rules)
		i := strings.Index(s, anchor)
		if i < 0 {
			return failf(ExitEnv, loginAnchor, what(e), path, anchor)
		}
		if e.after {
			i += len(anchor)
		}
		s = s[:i] + text + s[i:]
	}
	if err := writeFile(path, []byte(s), 0o666); err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	return nil
}
