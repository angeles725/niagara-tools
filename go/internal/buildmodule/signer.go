package buildmodule

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// expectedSigner is the signer the verify gate must find on every jar (verify-module.sh --expected-signer),
// resolved by modjar.ExpectedSigner as ng-deploy resolves it: the configured niagaraSigningAlias as given
// (the last --gradle-prop first), else the signer of the deployed jars. When the build knows none, signer is
// "" and warning is the line that says why.
func (b *build) expectedSigner() (signer, warning string) {
	alias := modjar.ConfiguredAlias(b.opt.gradleProps, b.s.Env, b.gradleRoot)
	switch signer, why := modjar.ExpectedSigner(alias, b.deployedSigners()); why {
	case modjar.NoSigner:
		return "", msgNoSigner
	case modjar.DevDeployed:
		return "", fmt.Sprintf(msgDevSignerDeployed, devSigner)
	default:
		return signer, ""
	}
}

// deployedSigners are the signers of the jars the drift gate snapshotted before gradle replaced them, one
// list per profile that had a jar (none under --no-drift-check).
func (b *build) deployedSigners() (signers [][]string) {
	for _, p := range b.profiles {
		if snap, ok := b.baseline[p]; ok {
			signers = append(signers, snap.signers)
		}
	}
	return signers
}

// gradleUserHome is the gradle user home of the build: $GRADLE_USER_HOME, else ~/.gradle.
func (b *build) gradleUserHome() string { return modjar.GradleUserHome(b.s.Env) }
