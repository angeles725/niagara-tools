package ngdeploy

import (
	"bytes"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// precheckVersion refuses a built jar whose bytes differ from the jar already in the station unless its
// vendorVersion went up: Software Manager and the module cache key on the version, not the bytes. The
// comparison is vendorversion.Cmp (numeric per component, a missing one is 0: 1.3 is 1.3.0). The same
// version is exit 51, a lower one exit 54, and a version that cannot be read or ordered exit 52. A jar
// with no deployed counterpart, or with the same bytes, is not checked. A build that passed and left no jar
// is exit 52 before any jar is compared: there is nothing to verify, and nothing to deploy.
func (d *deploy) precheckVersion() int {
	for _, built := range d.builtJars() {
		if is, err := regular(built); err == nil && !is {
			return d.fail(ExitVersionUnknown, msgVersionNoJar, basename(built), built)
		}
	}
	for _, built := range d.builtJars() {
		name := basename(built)
		deployed := d.deployedJar(name)
		differs, code := d.jarDiffers(name, built, deployed)
		if code != proceed {
			return code
		}
		if !differs {
			continue
		}
		newer, older := vendorversion.ModuleVersion(built), vendorversion.ModuleVersion(deployed)
		if newer == "" || older == "" {
			return d.fail(ExitVersionUnknown, msgVersionUnreadable, name, newer, older)
		}
		switch order, comparable := vendorversion.Cmp(newer, older); {
		case !comparable:
			return d.fail(ExitVersionUnknown, msgVersionIncomparable, name, newer, older)
		case order == 0:
			if code := d.sameVersion(name, built, deployed, newer); code != proceed {
				return code
			}
			continue
		case order < 0:
			return d.fail(ExitVersionDown, msgVersionDown, name, older, newer, older)
		}
		d.say(msgVersionOK, name, older, newer)
	}
	return proceed
}

// sameVersion judges a built jar whose bytes differ from the deployed one under the same vendorVersion. A
// rebuild of unchanged sources differs only in what every build changes (sameContent): there is no version to
// bump, and the deploy goes on. Any other difference is exit 51. A jar whose content cannot be read is
// neither: the precheck cannot verify (exit 52).
func (d *deploy) sameVersion(name, built, deployed, version string) int {
	switch same, why := sameContent(built, deployed); {
	case why != "":
		return d.fail(ExitVersionUnknown, msgVersionContent, name, why)
	case !same:
		return d.fail(ExitVersionSame, msgVersionSame, name, version)
	}
	d.say(msgVersionSameContent, name, version)
	return proceed
}

// jarDiffers reports whether both jars exist with different bytes. A jar that cannot be checked is exit 52
// (under bash it read as absent and the precheck was skipped); one that cannot be read counts as different,
// as a failed `cmp -s` did, so the version is then required to go up.
func (d *deploy) jarDiffers(name, built, deployed string) (differs bool, code int) {
	for _, jar := range []string{built, deployed} {
		is, err := regular(jar)
		if err != nil {
			return false, d.fail(ExitVersionUnknown, msgVersionCheck, name, jar, strerror.Text(err))
		}
		if !is {
			return false, proceed
		}
	}
	return !sameBytes(built, deployed), proceed
}

// sameBytes is cmp -s: both files can be read and hold the same bytes.
func sameBytes(a, b string) bool {
	x, errX := os.ReadFile(a)
	y, errY := os.ReadFile(b)
	return errX == nil && errY == nil && bytes.Equal(x, y)
}
