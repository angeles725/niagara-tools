package ngdeploy

import (
	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

// checkRecordedTarget holds the deploy to the module's optional niagara_target (the build-state.v1 envelope of
// the module, looked up as stateRecord says) before anything is built, moved or copied (#1158): when the
// installation STATION_MODULES_DIR belongs to has another major.minor than the recorded Niagara version, the run
// is exit 55 (ExitTargetVersion) naming both. A station whose version cannot be read is a WARN and the check of
// the jars against it (checkTargetVersion) still runs. No record, `unknown` or --no-deploy change nothing; a
// malformed record of the module is exit 10.
func (d *deploy) checkRecordedTarget() int {
	file, text, code := d.stateRecord()
	if code != proceed || file == "" {
		return code
	}
	rec, err := buildstatefields.ModuleTarget(text, d.module)
	if err != nil {
		return d.fail(ExitEnv, msgSignerInvalid, err.Error(), file)
	}
	if rec.Version == "" || d.isOn(optNoDeploy) {
		return proceed
	}
	installed, root, err := niagaraversion.InstallVersion(d.station)
	if err != nil {
		d.warn(warnRecordedTargetUnread, rec.Version, file, d.station, err)
		return proceed
	}
	if niagaraversion.MajorMinor(installed) != niagaraversion.MajorMinor(rec.Version) {
		return d.fail(ExitTargetVersion, msgRecordedTarget, d.module, rec.Version, file, root, installed,
			niagaraversion.MajorMinor(rec.Version))
	}
	d.say(msgRecordedTargetOK, rec.Version, d.module, file, root, installed)
	return proceed
}
