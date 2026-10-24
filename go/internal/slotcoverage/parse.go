package slotcoverage

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func isFacadeName(t string) bool {
	return strings.HasSuffix(t, "Panel") || strings.HasSuffix(t, "Facade")
}

// hasKeyLine is grep -qE '^[^#[:space:]].*=': a line that starts with neither "#" nor a blank and holds a
// later "=".
func hasKeyLine(lex []byte) bool {
	for _, ln := range lines(lex) {
		if ln != "" && ln[0] != '#' && !strings.ContainsRune(trailBlank, rune(ln[0])) && strings.Contains(ln[1:], "=") {
			return true
		}
	}
	return false
}

// dupKeys is sort | uniq -d over the raw keys: each key that occurs twice or more, once.
func dupKeys(keys []string) []string {
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	var out []string
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] && (len(out) == 0 || out[len(out)-1] != sorted[i]) {
			out = append(out, sorted[i])
		}
	}
	return out
}

// parseMode is the type-set mode: module-include.xml <type> names against "<Type>.displayName" keys.
func parseMode(xmlPath, lexPath string, strict, facade bool, s cli.Stdio) int {
	xml, ok := readInput(xmlPath, s)
	if !ok {
		return 3
	}
	if !isFile(lexPath) {
		// WB-LEX1: a missing lexicon with >=1 declared type is a FAIL, not an env fault.
		if n := len(grepO(typeTag, xml)); n >= 1 {
			fmt.Fprintf(s.Out, "slot-coverage: FAIL missing lexicon with %d declared type(s)\n", n)
			return 1
		}
		fmt.Fprintf(s.Err, "slot-coverage: not found: %s\n", lexPath)
		return 3
	}
	lex, ok := readInput(lexPath, s)
	if !ok {
		return 3
	}
	requiredCSV := strings.Join(sortU(typeNames(xml)), ",")
	raw := lexiconKeys(lex)
	keys := here(subst(trimKeys(raw)))
	var declared []string
	for _, k := range keys {
		if t, ok := strings.CutSuffix(k, ".displayName"); ok {
			declared = append(declared, t)
		}
	}
	warn, fail := false, false
	for _, t := range nonEmpty(strings.Split(requiredCSV, ",")) {
		if !contains(keys, t) || ((isFacadeName(t) || facade) && !contains(keys, t+".displayName")) {
			continue
		}
		fmt.Fprintf(s.Out, "slot-coverage: WARN ineffective type key: %s= (Workbench reads %s.displayName)\n", t, t)
		warn = true
	}
	if dups := subst(dupKeys(raw)); dups != "" {
		for _, k := range here(dups) {
			fmt.Fprintf(s.Out, "slot-coverage: FAIL dup-keys: %s\n", k)
		}
		fail = true
	}
	emptyLex := false
	if r := len(nonEmpty(strings.Split(requiredCSV, ","))); !hasKeyLine(lex) && r >= 1 {
		fmt.Fprintf(s.Out, "slot-coverage: FAIL empty lexicon with %d declared type(s)\n", r)
		emptyLex = true
	}
	cov := SetCoverage(strings.Join(sortU(declared), ","), requiredCSV)
	fmt.Fprintf(s.Out, "%s (type-set)\n%s\n%s\n", cov[0], cov[1], cov[2])
	missing := strings.TrimPrefix(cov[1], "missing=")
	facadeFail := false
	for _, t := range nonEmpty(strings.Split(missing, ",")) {
		if !isFacadeName(t) && !facade {
			continue
		}
		if contains(keys, t) {
			fmt.Fprintf(s.Out, "slot-coverage: WARN facade type with only a bare key: %s (add %s.displayName)\n", t, t)
			warn = true
			continue
		}
		fmt.Fprintf(s.Out, "slot-coverage: FAIL facade type without lexicon: %s\n", t)
		facadeFail = true
	}
	if fail || emptyLex || facadeFail || (strict && (missing != "" || warn)) {
		return 1
	}
	return 0
}
