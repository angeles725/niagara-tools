package buildmodule

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
)

// expectedSigner is the signer the verify gate must find on every jar (verify-module.sh --expected-signer):
// the configured niagaraSigningAlias as given, else the signer of the deployed jars. When the build knows
// none, signer is "" and warning is the line that says why. The signing plugin's default development signer
// is never taken from a deployed jar: a mirror that only ever held development builds would pin it and the
// gate would pass them without a word. Without an expected signer a jar signed with that key passes the gate
// and a station's Software Manager then rejects it as self-signed. [ev: external session retro, delta B5]
func (b *build) expectedSigner() (signer, warning string) {
	if alias := b.configuredAlias(); alias != "" {
		return alias, ""
	}
	switch name := b.deployedSigner(); name {
	case "":
		return "", msgNoSigner
	case devSigner:
		return "", fmt.Sprintf(msgDevSignerDeployed, devSigner)
	default:
		return name, ""
	}
}

// configuredAlias is the niagaraSigningAlias gradle property as gradle resolves it for the build: the last
// --gradle-prop, else the ORG_GRADLE_PROJECT_ environment variable, else GRADLE_USER_HOME/gradle.properties,
// else the gradle root's gradle.properties (read by signingprofile.GradleProperty, as lint-structure L16 and
// run-niagara-test read niagaraSigningProfile). An alias a root build.gradle.kts hardcodes is not seen.
func (b *build) configuredAlias() string {
	alias, given := "", false
	for _, prop := range b.opt.gradleProps {
		if v, ok := strings.CutPrefix(prop, "-P"+aliasProperty+"="); ok {
			alias, given = v, true
		}
	}
	if given {
		return alias
	}
	if v := b.s.Env(gradleEnvPrefix + aliasProperty); v != "" {
		return v
	}
	for _, dir := range []string{b.gradleUserHome(), b.gradleRoot} {
		data, err := os.ReadFile(dir + "/" + gradleProperties)
		if err != nil {
			continue
		}
		if v, ok := signingprofile.GradleProperty(data, aliasProperty); ok {
			return v
		}
	}
	return ""
}

// deployedSigner is the one signer of the jars the drift gate snapshotted before gradle replaced them (none
// under --no-drift-check), in the name its signature files carry. An unsigned deployed jar names nobody; a
// deployed jar with two signers, or two deployed jars with different ones, give no signer to expect.
func (b *build) deployedSigner() string {
	name := ""
	for _, p := range b.profiles {
		snap, ok := b.baseline[p]
		if !ok || len(snap.signers) == 0 {
			continue
		}
		if len(snap.signers) != 1 || (name != "" && snap.signers[0] != name) {
			return ""
		}
		name = snap.signers[0]
	}
	return name
}

// gradleUserHome is the gradle user home of the build: $GRADLE_USER_HOME, else ~/.gradle.
func (b *build) gradleUserHome() string {
	if dir := b.s.Env("GRADLE_USER_HOME"); dir != "" {
		return dir
	}
	return b.homeDir() + "/.gradle"
}
