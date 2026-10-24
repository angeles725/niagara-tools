// Package signblock is the one Go table of the root signing-block requirements: toolbelt/lib/signing-block.sh
// SIGN_BLOCK_NAME and SIGN_BLOCK_ERE, entry by entry and in order (TestPartsMatchTheShellTable and
// tests/lint-structure.bats LS15-sync keep them equal). Lint-structure L15 and preflight sample-sign read it.
// Parts[0] (allowDefaultProfile off) stops the signing plugin's default-profile fallback: without it a missing
// profile or a failed HSM/token login signs with the self-signed default key (issue #266).
package signblock

import "regexp"

// Part is one requirement: the name a row gives it and the pattern a code line must match.
type Part struct {
	Name string
	Re   *regexp.Regexp
}

// Parts is the table; each Re is matched against single code lines (comments removed).
var Parts = []Part{
	{"allowDefaultProfile.set(false) (or = false)", regexp.MustCompile(`allowDefaultProfile[[:space:]]*(\.set\([[:space:]]*false[[:space:]]*\)|=[[:space:]]*false([^[:alnum:]_]|$))`)},
	{"niagaraSigning {", regexp.MustCompile(`niagaraSigning[[:space:]]*\{`)},
	{"aliases", regexp.MustCompile(`(^|[^[:alnum:]_])aliases([^[:alnum:]_]|$)`)},
	{"signingProfileFile", regexp.MustCompile(`signingProfileFile`)},
}

// Missing is the names of the parts no line of code matches, in table order.
func Missing(code []string) []string {
	var miss []string
	for _, p := range Parts {
		found := false
		for _, ln := range code {
			if found = p.Re.MatchString(ln); found {
				break
			}
		}
		if !found {
			miss = append(miss, p.Name)
		}
	}
	return miss
}
