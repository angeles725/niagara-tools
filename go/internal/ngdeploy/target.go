package ngdeploy

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

// checkTargetVersion refuses a jar compiled against another Niagara version than the installation it is about
// to be copied into (exit 55), after the build and the version-bump precheck (which is first to say a build left no jar), before the schema-risk precheck, the gate and the copy: a
// module built against 4.14 in a 4.15 installation loads wrongly or not at all, and nothing at the copy says
// so. The version of the target is that of its modules/baja.jar (niagaraversion.InstallVersion); the version a
// jar was compiled against is the highest Tridium dependency floor of its module.xml (niagaraversion.CompiledFor),
// and the major and the minor must be equal. Every jar of the mode is compared. A target that holds no baja.jar
// is not an installation's modules directory: one NOTICE says the version is not checked. A version that cannot
// be read on either side is exit 55 too, never a pass. --no-deploy and a station directory that is
// <NIAGARA_HOME>/modules (the jars were built against it) copy nothing to compare with, and check nothing.
// [ev: retro deploy-target-version Δ1]
func (d *deploy) checkTargetVersion() int {
	if d.isOn(optNoDeploy) || d.sameDir {
		return proceed
	}
	installed, root, err := niagaraversion.InstallVersion(d.station)
	if errors.Is(err, niagaraversion.ErrNotInstallation) {
		d.say(noticeTargetNone, d.station, niagaraversion.BajaJar)
		return proceed
	}
	if err != nil {
		return d.fail(ExitTargetVersion, msgTargetTarget, err)
	}
	for _, built := range d.builtJars() {
		name := basename(built)
		compiled, err := niagaraversion.CompiledFor(built)
		if err != nil {
			return d.fail(ExitTargetVersion, msgTargetJar, name, err)
		}
		same, err := niagaraversion.Match(compiled, installed)
		switch {
		case err != nil:
			return d.fail(ExitTargetVersion, msgTargetJar, name, err)
		case !same:
			return d.fail(ExitTargetVersion, msgTargetMismatch, name, compiled, root, installed, niagaraversion.MajorMinor(installed), compiled)
		}
		d.say(msgTargetOK, name, compiled, root, installed)
	}
	return proceed
}
