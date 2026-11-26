package buildstatefields

import (
	"fmt"
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
	f := newKit(t, signerState("expected_signer: acme")+signerState("expected_signer: a b"), hand, nil)
	if code, _, errs := f.sweep(); code != ExitFail || errs != fmt.Sprintf(wantSigner, "a b") {
		t.Errorf("exit %d, stderr %q; want exit 1 naming 'a b'", code, errs)
	}
}
