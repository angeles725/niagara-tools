package reportmodule

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
	"github.com/angeles725/niagara-tools/go/internal/verifymodule"
)

// The target, signer and gate rows of report-module --ready (ready.go). Each returns status, check, detail.

const notChecked = "the target was not checked"

// niagaraTarget compares the jar's baja floor (the Niagara it requires at least) with --expect-niagara: a floor
// above the target FAILs, since the station would refuse the module, and so does a floor that cannot be compared
// with it (4.14-rc against 4.15): the target the operator named was not checked.
func niagaraTarget(floor, expect string) (string, string, string) {
	const check = "niagara-target"
	switch c, ok := vendorversion.Cmp(floor, expect); {
	case floor == "":
		return warn, check, "no baja dependency in META-INF/module.xml"
	case expect == "":
		return warn, check, "baja >= " + floor + " (no --expect-niagara: " + notChecked + ")"
	case !ok: // FAIL, not WARN: a READY line must never stand on a target nobody could check
		return failed, check, "baja >= " + floor + " cannot be compared with the target " + expect + ": the station may refuse the module"
	case c > 0:
		return failed, check, "baja >= " + floor + ", above the target " + expect
	default:
		return pass, check, "baja >= " + floor + " (target " + expect + ")"
	}
}

// homeNiagara is the Niagara version of a niagara_home: the vendorVersion of its modules/baja.jar ("" when
// there is no home or it cannot be read).
func homeNiagara(home string) string {
	if home == "" {
		return ""
	}
	xml, _ := moduleXMLOf(home + "/modules/baja.jar")
	return vendorversion.ModuleXMLVersion(xml)
}

// niagaraLine is the release line of a version: its first two numeric components (4.15.3.28 -> 4.15).
var niagaraLine = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?`)

// niagaraHome compares the Niagara of the compile home with --expect-niagara by release line: a newer line
// FAILs (the jar may call API the station does not have), an older one WARNs.
func niagaraHome(dir, version, expect string) (string, string, string) {
	const check = "niagara-home"
	if dir == "" {
		return skip, check, "no --niagara-home: the compile home is unknown"
	}
	if version == "" {
		return warn, check, dir + "/modules/baja.jar has no readable vendorVersion"
	}
	compiled := "compiled against Niagara " + version
	if expect == "" {
		return warn, check, compiled + " (no --expect-niagara: " + notChecked + ")"
	}
	switch c, ok := vendorversion.Cmp(niagaraLine.FindString(version), niagaraLine.FindString(expect)); {
	case !ok:
		return warn, check, compiled + " cannot be compared with the target " + expect
	case c > 0:
		return failed, check, compiled + ", newer than the target " + expect
	case c < 0:
		return warn, check, compiled + ", an older line than the target " + expect
	default:
		return pass, check, compiled + " (target " + expect + ")"
	}
}

// signerRow names the jar's complete signatures. Unsigned and the development signer FAIL always; with
// --expect-signer the only signature must be that alias's (compared as jarsigner names its files).
func signerRow(jar, expect string) (string, string, string) {
	const check = "signer"
	signers := modjar.Signers(jar)
	names := strings.Join(signers, ", ")
	for _, s := range signers {
		if s == modjar.DevSigner {
			return failed, check, s + " is the development signer: a station rejects the jar as self-signed"
		}
	}
	switch want := verifymodule.SigfileName(expect); {
	case len(signers) == 0:
		return failed, check, "unsigned (no META-INF/<SIGNER>.SF with its block file)"
	case expect == "":
		return warn, check, names + " (no --expect-signer: the signer was not checked)"
	case len(signers) == 1 && signers[0] == want:
		return pass, check, names + " (expected " + expect + ")"
	default:
		return failed, check, names + ", expected " + expect + " (" + want + ")"
	}
}

// verifyGate runs the verify-module gate on the jar (with --target-version and the expected signer when given)
// and reduces it to one row: its first FAIL line names the failure.
func verifyGate(x runner, o options, jar string) (string, string, string) {
	const check = "verify-module"
	var args []string
	if o.targetVersion != "" {
		args = append(args, "--target-version", o.targetVersion)
	}
	if o.expectSigner != "" {
		args = append(args, "--expected-signer", o.expectSigner)
	}
	out, code, ok := x.member("verify-module", append(args, "--src", o.root, jar)...)
	switch {
	case !ok:
		return errored, check, "output past the bound (rows lost)"
	case code == 0:
		return pass, check, "passed"
	case code == 3:
		return errored, check, "env fault (exit 3)"
	case code != 1:
		return errored, check, fmt.Sprintf("unexpected exit %d", code)
	}
	var fails []string
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "FAIL") {
			fails = append(fails, strings.TrimSpace(ln))
		}
	}
	switch len(fails) {
	case 0:
		return failed, check, "exit 1 with no FAIL row"
	case 1:
		return failed, check, fails[0]
	default:
		return failed, check, fmt.Sprintf("%s (+%d more FAIL rows)", fails[0], len(fails)-1)
	}
}
