package modjar

import (
	"os"
	"os/user"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
	"github.com/angeles725/niagara-tools/go/internal/verifymodule"
)

const (
	// AliasProperty is the gradle property the kit's root signing block takes the signing alias from
	// (templates/signing-root.gradle.kts); GradleEnvPrefix is how the environment sets a gradle property.
	AliasProperty   = "niagaraSigningAlias"
	GradleEnvPrefix = "ORG_GRADLE_PROJECT_"
	// GradleProperties is the properties file of a gradle root and of the gradle user home.
	GradleProperties = "gradle.properties"
	// DevSigner is the name the signature files of the signing plugin's default development key carry
	// (META-INF/NIAGARA4.SF): a signer a deployed jar never makes the expected one.
	DevSigner = "NIAGARA4"
)

// Why a tool has no signer to hand the gate; each tool words the remedy with its own options after it.
const (
	// WhyNoSigner is the warning when no alias is configured and no deployed jar names one signer.
	WhyNoSigner = "no niagaraSigningAlias is configured and no deployed jar gives a signer, so the gate accepts any signature: " +
		"the jar is signed with the signing plugin's default development signer unless the gradle root names another, " +
		"and a station's Software Manager shows that one as self-signed"
	// WhyDevSigner is the warning when the deployed jar carries the development signer; %s is its name.
	WhyDevSigner = "no niagaraSigningAlias is configured and the deployed jar itself carries the signing plugin's default development signer (%s), " +
		"which is never taken as the expected signer, so the gate accepts any signature: " +
		"a station's Software Manager shows that signer as self-signed"
)

// Reason says why ExpectedSigner named no signer.
type Reason int

const (
	Known       Reason = iota // a signer is expected
	NoSigner                  // no alias, and the deployed jars name no single signer
	DevDeployed               // no alias, and the deployed jars carry the development signer
)

// ExpectedSigner is the signer the verify gate must find on every jar (verify-module.sh --expected-signer):
// the configured niagaraSigningAlias as given, else the one signer of the deployed jars. When neither names
// one, signer is "" and why says which warning applies. The signing plugin's default development signer is
// never taken from a deployed jar: a home that only ever held development builds would pin it and the gate
// would pass them without a word. Without an expected signer a jar signed with that key passes the gate and a
// station's Software Manager then rejects it as self-signed. (source: delta B5 of an external session retro, 2026-11-13)
func ExpectedSigner(alias string, deployed [][]string) (signer string, why Reason) {
	if alias != "" {
		return alias, Known
	}
	// Compared as the gate compares a signer, so the development signer filed in another case is it too.
	switch name := OneSigner(deployed); {
	case name == "":
		return "", NoSigner
	case verifymodule.SigfileName(name) == DevSigner:
		return "", DevDeployed
	default:
		return name, Known
	}
}

// ConfiguredAlias is the niagaraSigningAlias gradle property as gradle resolves it for a build: the last
// -PniagaraSigningAlias= of gradleProps (the properties the tool hands gradle, as "-Pk=v"), else the
// ORG_GRADLE_PROJECT_ environment variable, else GRADLE_USER_HOME/gradle.properties, else the gradle root's
// gradle.properties (read by signingprofile.GradleProperty, as lint-structure L16 and run-niagara-test read
// niagaraSigningProfile). An alias a root build.gradle.kts hardcodes is not seen.
func ConfiguredAlias(gradleProps []string, env func(string) string, gradleRoot string) string {
	alias, given := "", false
	for _, prop := range gradleProps {
		if v, ok := strings.CutPrefix(prop, "-P"+AliasProperty+"="); ok {
			alias, given = v, true
		}
	}
	if given {
		return alias
	}
	if v := env(GradleEnvPrefix + AliasProperty); v != "" {
		return v
	}
	for _, dir := range []string{GradleUserHome(env), gradleRoot} {
		data, err := os.ReadFile(dir + "/" + GradleProperties)
		if err != nil {
			continue
		}
		if v, ok := signingprofile.GradleProperty(data, AliasProperty); ok {
			return v
		}
	}
	return ""
}

// Signers are the names of the complete signatures of a jar (META-INF/<NAME>.SF with its block file), none
// when it is unsigned or cannot be read: what a deployed jar says about who signs the module.
func Signers(jar string) []string { return verifymodule.Signers(jar) }

// OneSigner is the one signer of the deployed jars, each given as its Signers, in the name its signature
// files carry. An unsigned jar names nobody; a jar with two signers, or two jars with different ones, give
// no signer to expect.
func OneSigner(deployed [][]string) string {
	name := ""
	for _, signers := range deployed {
		if len(signers) == 0 {
			continue
		}
		if len(signers) != 1 || (name != "" && signers[0] != name) {
			return ""
		}
		name = signers[0]
	}
	return name
}

// GradleUserHome is the gradle user home of a build: $GRADLE_USER_HOME, else ~/.gradle.
func GradleUserHome(env func(string) string) string {
	if dir := env("GRADLE_USER_HOME"); dir != "" {
		return dir
	}
	return HomeDir(env) + "/.gradle"
}

// HomeDir is $HOME as bash reads it: the variable, else the user's home directory.
func HomeDir(env func(string) string) string {
	if h := env("HOME"); h != "" {
		return h
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return ""
}
