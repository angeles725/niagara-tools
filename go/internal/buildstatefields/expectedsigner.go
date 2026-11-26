package buildstatefields

import (
	"errors"
	"fmt"
)

// SignerKey is the optional envelope field that records the alias a module's releases are signed with
// (BUILD-STATE.src.md § How to read). Its value is one alias token (SignerPattern) or `unknown`, which
// records no expectation, the same as leaving the field out.
const (
	SignerKey     = "expected_signer"
	SignerUnknown = "unknown"
	// SignerPattern is the shape of a valid value, as the messages word it.
	SignerPattern = "a signing alias of letters, digits, '.', '_' or '-', or unknown"
)

// SignerInvalid is the message for a value that is not an alias; format it with the envelope's module and
// the value. The sweep prefixes it with "sweep: ", build.sh with its own name.
const SignerInvalid = "envelope (%s) " + SignerKey + " invalid: '%s' (want " + SignerPattern + ")"

// ValidSigner reports whether v is one alias token: letters, digits, '.', '_' or '-', at least one of them.
func ValidSigner(v string) bool {
	for i := 0; i < len(v); i++ {
		c := v[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return v != ""
}

// Messages of the structural errors of the field (the sweep prefixes them with "sweep: ", build.sh with its
// own name and adds the file): %s is the module (and the values or the line).
const (
	signerTwice     = "envelope (%s) " + SignerKey + " is given twice: '%s' and '%s'"
	signerStray     = "envelope (%s) has a line that is not the " + SignerKey + " field: '%s' (write " + SignerKey + ": <alias> at column 0)"
	signerTwoModule = "envelope (%s) is declared twice and " + SignerKey + " is set in one of them: keep one envelope"
)

// problem is the first error of the envelopes of one module (every envelope of a text for the sweep), "" when
// they are well formed: a line that mistypes the key, the key twice in one envelope, a value that is not an
// alias, then two envelopes of the module when any of them sets the field.
func problem(envs []signerEnv) string {
	for _, e := range envs {
		switch {
		case len(e.strays) > 0:
			return fmt.Sprintf(signerStray, e.module, e.strays[0])
		case len(e.values) > 1:
			return fmt.Sprintf(signerTwice, e.module, e.values[0], e.values[1])
		case len(e.values) == 1 && e.values[0] != SignerUnknown && !ValidSigner(e.values[0]):
			return fmt.Sprintf(SignerInvalid, e.module, e.values[0])
		}
	}
	seen := map[string]bool{}
	for _, e := range envs {
		if e.module != "" && seen[e.module] {
			for _, o := range envs {
				if o.module == e.module && len(o.values) > 0 {
					return fmt.Sprintf(signerTwoModule, e.module)
				}
			}
		}
		seen[e.module] = true
	}
	return ""
}

// CheckSigners returns the first error of the expected_signer fields of every envelope ("" when none).
func CheckSigners(text string) string { return problem(signerEnvelopes(text)) }

// CheckState is the content check of BUILD-STATE.md envelopes: CheckEnvelopes, CheckSigners, then CheckTargets.
func CheckState(text string) string {
	if msg := CheckEnvelopes(text); msg != "" {
		return msg
	}
	if msg := CheckSigners(text); msg != "" {
		return msg
	}
	return CheckTargets(text)
}

// ExpectedSigner is the alias the envelope naming module records, "" when it has none (the module has no
// envelope, no field, or `unknown`). Only the envelopes of module count: the first error among them (the
// messages above) is returned as the error; another module's malformed envelope is the sweep's business.
func ExpectedSigner(text, module string) (string, error) {
	var envs []signerEnv
	for _, e := range signerEnvelopes(text) {
		if e.module == module {
			envs = append(envs, e)
		}
	}
	if msg := problem(envs); msg != "" {
		return "", errors.New(msg)
	}
	if len(envs) == 0 || len(envs[0].values) == 0 || envs[0].values[0] == SignerUnknown {
		return "", nil
	}
	return envs[0].values[0], nil
}
