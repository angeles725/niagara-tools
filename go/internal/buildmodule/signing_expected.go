package buildmodule

import (
	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// kitStateFile is the kit's committed BUILD-STATE source, relative to the toolbelt directory: the file
// build.sh reads the module's envelope from unless --build-state names another or the module root holds
// its own BUILD-STATE.md (staterecord.go).
const kitStateFile = "../BUILD-STATE.src.md"

// applyExpectedSigner reads the module's optional expected_signer (BUILD-STATE envelope, see
// buildstatefields.ExpectedSigner) and applies it before anything runs (#1131). The precedence: an alias
// configured by --gradle-prop, $ORG_GRADLE_PROJECT_niagaraSigningAlias or gradle.properties
// (modjar.ConfiguredAlias) must equal the record, or the chain ends (exit 10) naming both; with none
// configured the record is handed to gradle as -PniagaraSigningAlias, so every later step (the signing block
// check, the key listing, the gate's signed row) sees it as the configured alias. A field that is absent or
// `unknown`, a module with no envelope, or a default file that is missing change nothing; a malformed record of
// the module (buildstatefields.ExpectedSigner) is exit 10 whatever alias is configured.
func (b *build) applyExpectedSigner() int {
	file, text, code := b.stateRecord()
	if code != proceed || file == "" {
		return code
	}
	recorded, err := buildstatefields.ExpectedSigner(text, b.opt.mod)
	if err != nil {
		b.warn(msgSignerInvalid, err.Error(), file)
		return ExitEnv
	}
	if recorded == "" {
		return proceed
	}
	switch alias := modjar.ConfiguredAlias(b.opt.gradleProps, b.s.Env, b.gradleRoot); alias {
	case recorded:
	case "":
		b.opt.gradleProps = append(b.opt.gradleProps, "-P"+modjar.AliasProperty+"="+recorded)
		b.say(msgSignerFromRecord, recorded, b.opt.mod, file)
	default:
		b.warn(msgSignerMismatch, alias, b.opt.mod, recorded, file)
		return ExitEnv
	}
	return proceed
}
