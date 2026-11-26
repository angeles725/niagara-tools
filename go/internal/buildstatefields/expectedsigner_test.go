package buildstatefields

import (
	"fmt"
	"strings"
	"testing"
)

// signerState is a BUILD-STATE with one module envelope (module Foo) carrying the given expected_signer line
// ("" leaves the field out).
func signerState(line string) string {
	s := "<!-- build-state.v1 -->\nmodule: Foo\nmodule_root: /work/Foo\nsource_of_truth: unknown\n" +
		"retro_required: false\nretro_pending: false\n"
	if line != "" {
		s += line + "\n"
	}
	return s + "<!-- /build-state.v1 -->\n"
}

const wantSigner = "sweep: envelope (Foo) expected_signer invalid: '%s' " +
	"(want a signing alias of letters, digits, '.', '_' or '-', or unknown)\n"

// An envelope without expected_signer stays valid and silent (the field is optional); a well-formed value is
// accepted with or without a trailing comment, and `unknown` is an honest value.
func TestSweepAcceptsAnOptionalExpectedSigner(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, line string }{
		{"absent", ""},
		{"an alias", "expected_signer: acme"},
		{"an alias with a comment", "expected_signer: acme-ca.2_x   # DECLARED: the release signer"},
		{"unknown", "expected_signer: unknown"},
	} {
		f := newKit(t, signerState(c.line), hand, nil)
		if code, out, errs := f.sweep(); code != ExitOK || out != "" || errs != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want 0 with no output", c.name, code, out, errs)
		}
	}
}

// A value that is not one alias token (or unknown) fails the sweep with exit 1 and names the module and the value.
func TestSweepRejectsAnInvalidExpectedSigner(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, line, value string }{
		{"empty", "expected_signer:", ""},
		{"only a comment", "expected_signer:  # none yet", ""},
		{"two words", "expected_signer: acme two", "acme two"},
		{"a slash", "expected_signer: acme/x", "acme/x"},
		{"quotes", `expected_signer: "acme"`, `"acme"`},
	} {
		f := newKit(t, signerState(c.line), hand, nil)
		code, out, errs := f.sweep()
		want := fmt.Sprintf(wantSigner, c.value)
		if code != ExitFail || out != "" || errs != want {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 1 and %q", c.name, code, out, errs, want)
		}
	}
}

// The check reads the field of every envelope, the kit's too, and the first bad one is the one named.
func TestSweepChecksTheExpectedSignerOfEveryEnvelope(t *testing.T) {
	t.Parallel()
	other := strings.Replace(signerState("expected_signer: acme"), "module: Foo", "module: Bar", 1)
	f := newKit(t, other+signerState("expected_signer: a b"), hand, nil)
	if code, _, errs := f.sweep(); code != ExitFail || errs != fmt.Sprintf(wantSigner, "a b") {
		t.Errorf("exit %d, stderr %q; want exit 1 naming 'a b'", code, errs)
	}
}

const (
	wantTwice = "sweep: envelope (Foo) expected_signer is given twice: '%s' and '%s'\n"
	wantStray = "sweep: envelope (Foo) has a line that is not the expected_signer field: '%s' " +
		"(write expected_signer: <alias> at column 0)\n"
	wantTwoEnvelopes = "sweep: envelope (Foo) is declared twice and expected_signer is set in one of them: " +
		"keep one envelope\n"
)

// The key twice in one envelope is an error, whatever the values (a first `unknown` or a bad second one
// must not hide the other).
func TestSweepRejectsTheSignerKeyTwice(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, first, second string }{
		{"unknown then an alias", "unknown", "acme"},
		{"an alias then a bad value", "acme", "a b"},
		{"the same alias twice", "acme", "acme"},
	} {
		f := newKit(t, signerState("expected_signer: "+c.first+"\nexpected_signer: "+c.second), hand, nil)
		code, out, errs := f.sweep()
		if want := fmt.Sprintf(wantTwice, c.first, c.second); code != ExitFail || out != "" || errs != want {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 1 and %q", c.name, code, out, errs, want)
		}
	}
}

// Two envelopes for one module are an error when either records expected_signer; two without it stay valid.
func TestSweepRejectsTwoEnvelopesOfOneModuleWithASigner(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, state string }{
		{"field in the second", signerState("") + signerState("expected_signer: acme")},
		{"field in the first", signerState("expected_signer: acme") + signerState("")},
	} {
		f := newKit(t, c.state, hand, nil)
		if code, out, errs := f.sweep(); code != ExitFail || out != "" || errs != wantTwoEnvelopes {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 1 and %q", c.name, code, out, errs, wantTwoEnvelopes)
		}
	}
	f := newKit(t, signerState("")+signerState(""), hand, nil)
	if code, _, errs := f.sweep(); code != ExitOK || errs != "" {
		t.Errorf("two envelopes without the field: exit %d, stderr %q; want 0", code, errs)
	}
}

// A line that is clearly the key mistyped (a space before the colon, an indented key) is an error, not a field
// that is silently not there; other keys keep the convention that an indented key is no field.
func TestSweepRejectsAMistypedSignerKey(t *testing.T) {
	t.Parallel()
	for _, line := range []string{"expected_signer : acme", "  expected_signer: acme", "\texpected_signer:acme"} {
		f := newKit(t, signerState(line), hand, nil)
		want := fmt.Sprintf(wantStray, line)
		if code, out, errs := f.sweep(); code != ExitFail || out != "" || errs != want {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want exit 1 and %q", line, code, out, errs, want)
		}
	}
	for _, line := range []string{"  retro_x: 1", "expected_signers: acme", "# expected_signer : acme", "  - expected_signer: x"} {
		f := newKit(t, signerState(line), hand, nil)
		if code, _, errs := f.sweep(); code != ExitOK || errs != "" {
			t.Errorf("%q: exit %d, stderr %q; want 0 (not the signer key)", line, code, errs)
		}
	}
}
