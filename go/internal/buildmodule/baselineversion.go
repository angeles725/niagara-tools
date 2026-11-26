package buildmodule

import (
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// warnBaselineVersion checks the schema-baseline ref's module version (defaultModuleVersion of the group build
// file at the ref, in <module-root> or <module-root>/<MOD>). Against a recorded deploy (--target) the two records
// of one deploy must agree, else one WARN. Without one, nothing says what a station runs: the vendorVersion of each
// jar under <niagara_home>/modules is only what the mirror holds (a build), so a difference is one WARN saying
// that and that the deployed version is unknown. The exit code never changes. A ref with no readable version, or
// a profile with no jar in the mirror, is not checked.
func (b *build) warnBaselineVersion(ref string, rec deployRecord) {
	want := b.refVersion(ref)
	if want == "" {
		return
	}
	if rec.version != "" {
		if c, ok := vendorversion.Cmp(want, rec.version); !ok || c != 0 {
			b.warn(msgTargetVersion, deployTargetsFile, b.opt.mod, rec.version, b.opt.deployTarget, want)
		}
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
