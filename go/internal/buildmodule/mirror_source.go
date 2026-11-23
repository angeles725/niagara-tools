package buildmodule

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
	"github.com/angeles725/niagara-tools/go/internal/reportmodule"
)

// checkTargetScript is the toolbelt check that compares a jar's compile version with an installation's.
const checkTargetScript = "check-target-version.sh"

// sayMirrorSource ends a passing mirror build with one line: the real installation the mirror was made from
// (the marker's source), its Niagara version, that the jars built this run were NOT installed there (gradle's
// :jar step wrote them into the mirror only), and the command that installs them: the version check of
// check-target-version.sh on the first jar, then the copy of every jar built. A mirror whose marker names no
// source, a home that is not a mirror and a build that did not pass print nothing. A source version that cannot
// be read is "unknown". Nothing is installed by the build. [ev: retro deploy-target-version Δ2]
func (b *build) sayMirrorSource() {
	if b.home == "" || !modjar.IsMirror(b.home) || len(b.profiles) == 0 {
		return
	}
	source := modjar.MirrorSource(b.home)
	if source == "" {
		return
	}
	version, _, err := niagaraversion.InstallVersion(source)
	if err != nil {
		version = "unknown"
	}
	jars := make([]string, len(b.profiles))
	for i, p := range b.profiles {
		jars[i] = reportmodule.ShellWord(b.builtJar(p))
	}
	install := reportmodule.ShellWord(b.here+"/"+checkTargetScript) + " " + jars[0] + " " + reportmodule.ShellWord(source) +
		" && cp " + strings.Join(jars, " ") + " " + reportmodule.ShellWord(source+"/modules/")
	b.say(msgMirrorSource, b.home, source, version, install)
}
