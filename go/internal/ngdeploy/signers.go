package ngdeploy

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// expectedSigner is the signer the gate must find on every jar (verify-module.sh --expected-signer), as
// build.sh resolves it (modjar.ExpectedSigner): the niagaraSigningAlias gradle property of the gradle root
// that holds gradlew; else the one signer of the module's jars of the mode the station holds before the copy
// (deployedJar: the ones it runs, bench Δ24); else, when those give none (none signed, two signers, or the
// development key), the one signer of the jars that were in <NIAGARA_HOME>/modules before the build (home.go
// moved them aside), as before the station was read. The two sources are never mixed in one decision, and
// the development signer is never expected. A station jar whose signer cannot be told (unknownSigner) gives
// no signer and no fallback to the home. When the deploy knows none, signer is "" and warning says why: the
// gate then accepts any signature.
func (d *deploy) expectedSigner() (signer, warning string) {
	if alias := modjar.ConfiguredAlias(d.gradleProps, d.env, dirname(d.env("GRADLEW_PATH"))); alias != "" {
		return alias, ""
	}
	var station, home [][]string
	for _, profile := range profilesOf[d.raw[optMode]] {
		jar := d.deployedJar(d.jarName(profile))
		if why := unknownSigner(jar); why != "" {
			return "", fmt.Sprintf(warnSignerUnknown, jar, why)
		}
		station = append(station, modjar.Signers(jar))
	}
	for _, jar := range d.homeJars {
		if jar.moved && d.inMode(jar.name) {
			home = append(home, modjar.Signers(jar.aside))
		}
	}
	fromStation, whyStation := modjar.ExpectedSigner("", station)
	if whyStation == modjar.Known {
		return fromStation, ""
	}
	switch fromHome, whyHome := modjar.ExpectedSigner("", home); {
	case whyHome == modjar.Known:
		return fromHome, ""
	case whyStation == modjar.DevDeployed || whyHome == modjar.DevDeployed:
		return "", fmt.Sprintf(warnDevSigner, modjar.DevSigner)
	default:
		return "", warnNoSigner
	}
}

// unknownSigner says why the signer of a jar the station holds cannot be told, or "" when it can: a jar that
// is not there, or is an unsigned zip, names nobody, but one that cannot be read, is not a zip, or holds a
// signature file without its block file is not an unsigned jar.
func unknownSigner(jar string) string {
	if _, err := os.Lstat(jar); errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	r, err := zip.OpenReader(jar)
	if err != nil {
		return "it cannot be read as a jar: " + err.Error()
	}
	defer r.Close()
	complete := map[string]bool{}
	for _, name := range modjar.Signers(jar) {
		complete[name] = true
	}
	for _, entry := range r.File {
		name, ok := strings.CutPrefix(entry.Name, "META-INF/")
		dot := strings.LastIndexByte(name, '.')
		if ok && dot > 0 && !strings.Contains(name, "/") && strings.EqualFold(name[dot:], ".SF") && !complete[name[:dot]] {
			return entry.Name + " has no signature block file"
		}
	}
	return ""
}
