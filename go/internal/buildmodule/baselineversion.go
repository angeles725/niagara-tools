package buildmodule

import (
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// warnBaselineVersion checks --schema-baseline-ref against what is deployed: the module version the ref's group
// build file declares (defaultModuleVersion, in <module-root> or <module-root>/<MOD>) is compared with the
// vendorVersion of each jar under <niagara_home>/modules. A baseline of 2.3.0 with 2.3.1 deployed makes schema-risk
// check the wrong floor, so it is one WARN per jar; the exit code never changes. A ref with no readable version, or
// a profile with no deployed jar, is not checked.
func (b *build) warnBaselineVersion(ref string) {
	want := b.refVersion(ref)
	if want == "" {
		return
	}
	for _, p := range []string{"rt", "ux", "wb"} {
		jar := b.deployedJar(p)
		if !lint.IsFile(jar) {
			continue
		}
		have := vendorversion.ModuleVersionFromZip(jar)
		if c, ok := vendorversion.Cmp(want, have); have == "" || (ok && c == 0) {
			continue
		}
		b.warn(msgBaselineRefVersion, ref, want, b.jarName(p), have)
	}
}

// refVersion is the defaultModuleVersion of the group build.gradle.kts at ref, "" when there is none.
func (b *build) refVersion(ref string) string {
	for _, dir := range []string{b.root, b.root + "/" + b.opt.mod} {
		out, err := b.command("git", "-C", dir, "show", ref+":./build.gradle.kts").Output() // stderr dropped: a file absent at ref
		if err != nil {
			continue
		}
		if v := vendorversion.SourceVersion(string(out)); v != "" {
			return v
		}
	}
	return ""
}
