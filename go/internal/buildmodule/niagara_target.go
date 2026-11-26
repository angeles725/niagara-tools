package buildmodule

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

// applyNiagaraTarget reads the module's optional niagara_target and plugin_version (the build-state.v1 envelope
// of <MOD>, looked up as stateRecord says) and holds the build to them before anything runs (#1158):
//   - the Niagara version of niagara_home (and, for a mirror, of the installation it was made from, when that
//     one can be read) must equal niagara_target in major.minor, else exit 10 naming both and the file; a home
//     whose version cannot be read is a WARN, never a pass in silence;
//   - --target-version defaults to the major.minor of niagara_target, and another major.minor is exit 10;
//   - --plugin-version (or $NIAGARA_PLUGIN_VERSION) defaults to plugin_version, and another value is exit 10.
//
// One INFO line says which record set what. A module with no record, or `unknown`, builds as before; a malformed
// record of the module is exit 10.
func (b *build) applyNiagaraTarget() int {
	file, text, code := b.stateRecord()
	if code != proceed || file == "" {
		return code
	}
	rec, err := buildstatefields.ModuleTarget(text, b.opt.mod)
	if err != nil {
		b.warn(msgSignerInvalid, err.Error(), file)
		return ExitEnv
	}
	if rec.Version == "" && rec.Plugin == "" {
		return proceed
	}
	var set []string
	if rec.Version != "" {
		line := niagaraversion.MajorMinor(rec.Version)
		if code := b.checkTargetHome(rec.Version, line, file); code != proceed {
			return code
		}
		if b.opt.target == "" {
			b.opt.target = line
		} else if niagaraversion.MajorMinor(b.opt.target) != line {
			b.warn(msgTargetFlagDiffers, b.opt.target, rec.Version, line, b.opt.mod, file)
			return ExitEnv
		}
		set = append(set, "--target-version "+b.opt.target)
	}
	if rec.Plugin != "" {
		switch b.plugin {
		case "":
			b.plugin = rec.Plugin
		case rec.Plugin:
		default:
			b.warn(msgPluginFlagDiffers, b.plugin, rec.Plugin, b.opt.mod, file)
			return ExitEnv
		}
		set = append(set, "--plugin-version "+b.plugin)
	}
	b.say(msgNiagaraRecord, file, b.opt.mod, recordedFields(rec), strings.Join(set, ", "))
	return proceed
}

// recordedFields words what the record holds: "niagara_target 4.15.6.20, plugin_version 7.6.22".
func recordedFields(rec buildstatefields.Target) string {
	var fields []string
	if rec.Version != "" {
		fields = append(fields, buildstatefields.TargetField+" "+rec.Version)
	}
	if rec.Plugin != "" {
		fields = append(fields, buildstatefields.PluginField+" "+rec.Plugin)
	}
	return strings.Join(fields, ", ")
}

// checkTargetHome compares the Niagara version of niagara_home with the recorded target, and for a mirror also
// that of its source. A version that cannot be read is a WARN naming the reason.
func (b *build) checkTargetHome(target, line, file string) int {
	homes := []struct{ dir, what string }{{b.home, "niagara_home " + b.home}}
	if modjar.IsMirror(b.home) {
		if source := modjar.MirrorSource(b.home); source != "" {
			homes = append(homes, struct{ dir, what string }{source, "the installation " + source + " (the source of mirror " + b.home + ")"})
		}
	}
	for i, h := range homes {
		version, _, err := niagaraversion.InstallVersion(h.dir)
		switch {
		case err != nil && i == 0:
			b.warn(msgTargetHomeUnread, h.what, target, file, err)
		case err != nil:
		case niagaraversion.MajorMinor(version) != line:
			b.warn(msgTargetHomeDiffers, b.opt.mod, target, line, file, h.what, version)
			return ExitEnv
		}
	}
	return proceed
}
