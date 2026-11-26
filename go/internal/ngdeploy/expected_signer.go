package ngdeploy

import (
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// buildStateFlag names the file that holds the module's build-state.v1 envelope, as build.sh's --build-state
// does; without it the kit's BUILD-STATE.src.md is read (kitStateFile). It is a flag only: --gradle-prop and
// --plugin-version, the other options of the command line that have no env-file key, are its siblings.
const buildStateFlag = "--build-state"

// kitStateFile is the kit's committed BUILD-STATE source, from the scripts directory.
const kitStateFile = "../build-n4-module-kit/BUILD-STATE.src.md"

// buildStateFile reads the value of --build-state into the file applyExpectedSigner reads; an empty one is no
// file, not the default one (build.sh refuses it the same way).
func (d *deploy) buildStateFile(rest []string) int {
	if len(rest) == 0 {
		return d.fail(ExitEnv, msgNeedsArgument, buildStateFlag)
	}
	if rest[0] == "" {
		return d.fail(ExitEnv, msgBuildStateEmpty, buildStateFlag)
	}
	d.buildState = rest[0]
	return proceed
}

// applyExpectedSigner reads the module's optional expected_signer (buildstatefields.ExpectedSigner, the
// reader build.sh uses) before anything runs, and decides the signer as build.sh does (#1149): an alias
// configured by --gradle-prop, ORG_GRADLE_PROJECT_niagaraSigningAlias or gradle.properties
// (modjar.ConfiguredAlias) must equal the record, or the deploy stops (exit 10) naming both; with none
// configured the record becomes the configured alias, handed to gradle as -PniagaraSigningAlias, so the build
// signs with it and expectedSigner (signers.go) returns it, never the signer of a deployed jar. A field that
// is absent or `unknown`, a module with no envelope, or a default file that is missing change nothing; a
// malformed record of the module, and a --build-state file that cannot be read, are exit 10.
func (d *deploy) applyExpectedSigner() int {
	file, explicit := d.buildState, d.buildState != ""
	if !explicit {
		file = filepath.Join(d.scriptDir, kitStateFile)
	}
	text, err := os.ReadFile(file)
	if err != nil {
		if explicit {
			return d.fail(ExitEnv, msgStateUnreadable, file, strerror.Text(err))
		}
		return proceed
	}
	recorded, err := buildstatefields.ExpectedSigner(string(text), d.module)
	if err != nil {
		return d.fail(ExitEnv, msgSignerInvalid, err.Error(), file)
	}
	if recorded == "" {
		return proceed
	}
	switch alias := modjar.ConfiguredAlias(d.gradleProps, d.env, dirname(d.env("GRADLEW_PATH"))); alias {
	case recorded:
	case "":
		d.gradleProps = append(d.gradleProps, "-P"+modjar.AliasProperty+"="+recorded)
		d.say(msgSignerFromRecord, recorded, d.module, file)
	default:
		return d.fail(ExitEnv, msgSignerMismatch, alias, d.module, recorded, file)
	}
	return proceed
}
