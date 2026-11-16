package verifymodule

import (
	"fmt"
	"strings"
)

// The details of the signed row. The first three are the bash tool's; the last two are added when
// --expected-signer names the signer: its alias as given and the file jarsigner names after it.
const (
	signedBy       = "signed by alias %s (META-INF/<ALIAS>.SF + signature block)"
	signedOrphan   = "%s without its .RSA/.DSA/.EC signature block (jar not signed)"
	signedNone     = "no META-INF/*.SF (jar not signed)"
	signedExpected = "signed by alias %s, the expected signer %s (META-INF/%s.SF + signature block)"
	signedOnly     = "%s; expected only the signer %s (META-INF/%s.SF + signature block)"

	msgNoAlias = "verify-module: --expected-signer needs an alias\n"
)

// sigfileLen is how many characters of an alias jarsigner keeps in a signature file name.
const sigfileLen = 8

// sigRe is a signature file: an .SF or a block file directly under META-INF/. The extension is read in any
// case, as the JDK verifier reads it, so a signer filed as NAME.sf + NAME.rsa is a signer here too.
var sigRe = rx(`^META-INF/[^/]+\.(?i:SF|RSA|DSA|EC)$`)

// signers lists the jar's signature files: complete holds the name of every META-INF/<NAME>.SF that has its
// block file, orphan every .SF entry without one.
func (j *jarFile) signers() (complete, orphan []string) {
	blocks, sfs := map[string]bool{}, []string(nil)
	for _, n := range j.list.lines {
		if !sigRe.MatchString(n) || !j.list.shown(n) {
			continue
		}
		dot := strings.LastIndexByte(n, '.')
		if strings.EqualFold(n[dot:], ".SF") {
			sfs = append(sfs, n)
		} else {
			blocks[n[:dot]] = true
		}
	}
	for _, sf := range sfs {
		base := sf[:strings.LastIndexByte(sf, '.')]
		if blocks[base] {
			complete = append(complete, strings.TrimPrefix(base, "META-INF/"))
		} else {
			orphan = append(orphan, sf)
		}
	}
	return complete, orphan
}

// Signers is the names of the complete signatures of the jar at path, as the signed row reports them: none
// for an unsigned jar and for one the gate cannot open. build.sh reads the deployed jar's signer with it.
func Signers(path string) []string {
	j, msg := openJar(path)
	if msg != "" {
		return nil
	}
	defer j.close()
	complete, _ := j.signers()
	return complete
}

// SigfileName is the name jarsigner gives the signature files of an alias: the first 8 characters,
// upper-cased, each one outside A-Z 0-9 _ - as "_" (observed by signing jars with the jarsigner of JDK 8 and
// of JDK 21). Alias acmesigner signs as META-INF/ACMESIGN.SF. It is the form --expected-signer compares in,
// and build.sh recognises the development signer of a deployed jar in it.
func SigfileName(alias string) string {
	name := []rune(alias)
	if len(name) > sigfileLen {
		name = name[:sigfileLen]
	}
	return strings.Map(func(c rune) rune {
		if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			return c
		}
		return '_'
	}, strings.ToUpper(string(name)))
}

// signed is the signed row. Without --expected-signer a jar with at least one complete signature passes,
// whoever signed it; with it the complete signatures must be exactly the expected signer's, so a jar signed
// with the development key, or with a second key, FAILs.
func (r *run) signed() bool {
	complete, orphan := r.jar.signers()
	detail := signedNone
	switch {
	case len(complete) > 0:
		detail = fmt.Sprintf(signedBy, strings.Join(complete, ", "))
	case len(orphan) > 0:
		detail = fmt.Sprintf(signedOrphan, strings.Join(orphan, ", "))
	}
	if r.expected == "" {
		return r.verdict(len(complete) > 0, "signed", detail)
	}
	want := SigfileName(r.expected)
	// The name as given also matches: a signer other than jarsigner may name its files after the whole alias.
	if len(complete) == 1 && (complete[0] == want || complete[0] == r.expected) {
		return r.verdict(true, "signed", fmt.Sprintf(signedExpected, complete[0], r.expected, complete[0]))
	}
	if len(complete) > 0 {
		detail = "signed by alias " + strings.Join(complete, ", ")
	}
	return r.verdict(false, "signed", fmt.Sprintf(signedOnly, detail, r.expected, want))
}

// verdict prints a PASS or FAIL row and returns pass.
func (r *run) verdict(pass bool, check, detail string) bool {
	status := "FAIL"
	if pass {
		status = "PASS"
	}
	r.row(status, check, detail)
	return pass
}
