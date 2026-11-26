package buildstatefields

import (
	"fmt"
	"strings"
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

// signerEnvelopes are the module and the expected_signer value of every build-state.v1 envelope of a text, in
// order; set is false for an envelope without the field. Read like the other envelope readers: keys start at
// column 0, a `# comment` ends the value, a byte-order mark that opens the text is dropped and a CR ends a line.
func signerEnvelopes(text string) (found []signerField) {
	text = strings.ReplaceAll(strings.TrimPrefix(text, "\xef\xbb\xbf"), "\r", "\n")
	var cur *signerField
	for _, line := range Lines(text) {
		switch {
		case strings.HasPrefix(line, OpenMarker):
			found = append(found, signerField{})
			cur = &found[len(found)-1]
		case strings.HasPrefix(line, CloseMarker):
			cur = nil
		case cur != nil:
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			if i := strings.IndexByte(val, '#'); i >= 0 {
				val = val[:i]
			}
			switch key {
			case "module":
				if cur.module == "" {
					cur.module = trimSpace(val)
				}
			case SignerKey:
				if !cur.set {
					cur.value, cur.set = trimSpace(val), true
				}
			}
		}
	}
	return found
}

type signerField struct {
	module, value string
	set           bool
}

// CheckSigners returns the first envelope whose expected_signer is neither an alias nor `unknown`, worded as
// SignerInvalid ("" when every one is valid or absent).
func CheckSigners(text string) string {
	for _, e := range signerEnvelopes(text) {
		if e.set && e.value != SignerUnknown && !ValidSigner(e.value) {
			return fmt.Sprintf(SignerInvalid, e.module, e.value)
		}
	}
	return ""
}

// CheckState is the content check of BUILD-STATE.md envelopes: CheckEnvelopes, then CheckSigners.
func CheckState(text string) string {
	if msg := CheckEnvelopes(text); msg != "" {
		return msg
	}
	return CheckSigners(text)
}

// ExpectedSigner is the alias the first envelope naming module records, "" when it has none (the module has no
// envelope, no field, or `unknown`). A value that is not an alias is an error worded as SignerInvalid.
func ExpectedSigner(text, module string) (string, error) {
	for _, e := range signerEnvelopes(text) {
		if e.module != module {
			continue
		}
		if !e.set || e.value == SignerUnknown {
			return "", nil
		}
		if !ValidSigner(e.value) {
			return "", fmt.Errorf(SignerInvalid, e.module, e.value)
		}
		return e.value, nil
	}
	return "", nil
}
