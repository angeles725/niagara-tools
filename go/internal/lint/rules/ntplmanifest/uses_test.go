package ntplmanifest

import (
	"strings"
	"testing"
)

// usesBog is a template.bog whose components use baja, control (the manifest lists control-rt), vendorx
// (nobody lists or ships it), kitIo (the installation ships kitIo-rt) and ModA (the module's own part),
// each through a `m="alias=module"` attribute, plus a module:// ORD to vendorw in a value.
const usesBog = `<bajaObjectGraph version="4.0"><p m="b=baja" t="b:Folder">` +
	`<p n="a" m="c=control" t="c:NumericWritable"/><p n="b" m="v=vendorx" t="v:Thing"/>` +
	`<p n="c" m="k=kitIo k2=kitIo" t="k:Io"/><p n="d" m="o=ModA" t="o:Own"/>` +
	`<p n="e" t="b:String" v="module://vendorw/icon.png"/></p></bajaObjectGraph>` + "\n"

// usesPx is a Px file that imports gx (shipped as gx-ux) and vendory, and points at module://control.
const usesPx = `<?xml version="1.0" encoding="UTF-8"?>` + "\n<px version=\"1.0\">\n<import>\n  <module name=\"baja\"/>\n" +
	"  <module name=\"gx\"/>\n  <module name=\"vendory\"/>\n</import>\n<content><img src=\"module://control/x.png\"/></content>\n</px>\n"

// usesTemplate is an archive with usesBog, usesPx and a manifest listing deps.
func usesTemplate(t *testing.T, deps ...string) string {
	return ntpl(t, entry{bogEntry, usesBog}, entry{manifestEntry, manifestXML("", deps...)}, entry{"px/View.px", usesPx})
}

// usesHome is an installation shipping the stock modules, kitIo-rt and gx-ux.
func usesHome(t *testing.T) string {
	return installation(t, append([]string{"kitIo-rt", "gx-ux"}, stock...)...)
}

// TestModulesTheTemplateUsesNeedADependency (retro coordination-stretch K4): a module that template.bog
// (an alias, a module:// ORD) or a Px file (an import, a module:// ORD) uses needs a <dependency> row naming
// it or one of its parts. One the installation ships (--niagara-home, as for ntpl-deps), baja and the
// module's own parts are no finding. The rest is one FAIL ntpl-uses row per entry, names sorted.
func TestModulesTheTemplateUsesNeedADependency(t *testing.T) {
	t.Parallel()
	nh := usesHome(t)
	root := moduleRoot(t, map[string]string{tpl: usesTemplate(t, "control-rt", "local-rt")})
	want := "FAIL  ntpl-uses  " + root + "/" + tpl + ": template.bog uses vendorw, vendorx: no <dependency> of template-manifest.xml names them, and the installation " + nh + " does not ship them\n" +
		"FAIL  ntpl-uses  " + root + "/" + tpl + ": px/View.px uses vendory: no <dependency> of template-manifest.xml names them, and the installation " + nh + " does not ship them\n"
	if code, out, errText := run(homeOption, nh, root); code != 1 || out != want || errText != "" {
		t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit 1\nstdout %q", code, out, errText, want)
	}
	listed := moduleRoot(t, map[string]string{tpl: usesTemplate(t, "control-rt", "vendorx-rt", "vendorw", "vendory-ux"),
		"ModA-rt/ModA-rt.gradle.kts": partBuild + "dependencies { api(\":vendorx-rt\")\n api(\":vendorw\")\n api(\":vendory-ux\") }\n"})
	if code, out, errText := run(homeOption, nh, listed); code != 0 || out != "" || errText != "" {
		t.Errorf("every module listed: exit %d stdout %q stderr %q, want a clean pass", code, out, errText)
	}
}

// TestModulesTheTemplateUsesWithoutAnInstallation: without the installation's module list a stock module
// cannot be told from a missing one: the rows are WARN (after the one note), exit 0, as for ntpl-deps.
func TestModulesTheTemplateUsesWithoutAnInstallation(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{tpl: usesTemplate(t, "control-rt", "local-rt")})
	code, out, _ := run(root)
	if code != 0 || !strings.Contains(out, "WARN  lint-ntpl-manifest-deps  "+root+": the installation's module list is not known") ||
		!strings.Contains(out, "WARN  ntpl-uses  "+root+"/"+tpl+": template.bog uses kitIo, vendorw, vendorx: no <dependency> of template-manifest.xml names them\n") ||
		!strings.Contains(out, "WARN  ntpl-uses  "+root+"/"+tpl+": px/View.px uses gx, vendory: no <dependency> of template-manifest.xml names them\n") {
		t.Errorf("exit %d stdout %q, want exit 0 with the note and two WARN ntpl-uses rows", code, out)
	}
}

// TestEntryTheUsesCheckCannotRead: a template.bog or a Px file that is not well-formed XML is a FAIL
// ntpl-uses row naming it, never a template that uses nothing.
func TestEntryTheUsesCheckCannotRead(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, "<bajaObjectGraph><p m=\"v=vendorx\">"},
		entry{manifestEntry, manifestXML("")}, entry{"px/Bad.px", "<px><import></px>"})})
	expect(t, "two entries that do not parse", root, 1,
		"FAIL  ntpl-uses  @ROOT@/"+tpl+": template.bog is not well-formed XML: 1:35: element <p> opened at 1:18 is never closed\n"+
			"FAIL  ntpl-uses  @ROOT@/"+tpl+": px/Bad.px is not well-formed XML: 1:13: element <import> opened at 1:5 closed by </px>\n")
}

// TestApplicationArchiveIsATemplate (K4): a .napl application archive is the same zip as a .ntpl and is
// checked the same way, the extension in any letter case.
func TestApplicationArchiveIsATemplate(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t, map[string]string{"ModA-rt/src/app/App.napl": ntpl(t, entry{bogEntry, bog}), "ModA-rt/src/app/B.NAPL": template(t, "", "ghost-rt")})
	expect(t, "two application archives", root, 1, "FAIL  ntpl-entries  @ROOT@/ModA-rt/src/app/App.napl: lacks template-manifest.xml\n"+
		undeclaredRow("ModA-rt/src/app/B.NAPL", "ghost-rt"))
}

// pxOf is a Px file that imports module and is padded with a comment to size bytes in all.
func pxOf(module string, size int) string {
	head := "<px version=\"1.0\"><import><module name=\"" + module + "\"/></import><!--"
	return head + strings.Repeat(" ", size-len(head)-len("--></px>")) + "--></px>"
}

// TestUsesReadBounds (review of #702): one bog or Px entry over maxUsesRead is a FAIL row naming it and the
// bound, the other entries still read; and the entries of one archive share a budget of maxUsesTotal inflated
// bytes: past it one FAIL row names the archive's limit and the first entry not read, and nothing more of the
// archive is read, however many entries it holds.
func TestUsesReadBounds(t *testing.T) {
	t.Parallel()
	big := moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML("")},
		entry{"px/Big.px", pxOf("vendorx", maxUsesRead+1)}, entry{"px/Small.px", pxOf("vendory", 200)})})
	expect(t, "one entry over the bound", big, 1,
		"FAIL  ntpl-uses  @ROOT@/"+tpl+": px/Big.px inflates past 33554432 bytes -- not read\n"+
			"FAIL  ntpl-uses  @ROOT@/"+tpl+": px/Small.px uses vendory: no <dependency> of template-manifest.xml names them, and the installation @HOME@ does not ship them\n")
	many := []entry{{bogEntry, bog}, {manifestEntry, manifestXML("")}}
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		many = append(many, entry{"px/" + n + ".px", pxOf("baja", maxUsesTotal/4)})
	}
	expect(t, "entries over the archive's budget", moduleRoot(t, map[string]string{tpl: ntpl(t, many...)}), 1,
		"FAIL  ntpl-uses  @ROOT@/"+tpl+": the bog and Px entries inflate past 134217728 bytes in all, the bound of one archive -- px/d.px and every entry after it not read\n")
}
