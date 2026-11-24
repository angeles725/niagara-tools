package spapoll

import "github.com/angeles725/niagara-tools/go/internal/lint"

// checkRecovery is the WARN half: the file's recovery is missing or not keyed on the last success.
func checkRecovery(lines []string, prefix string, call, decl, end int) []lint.Finding {
	recovery := false
	for _, ln := range lines {
		if reload.MatchString(ln) || hrefSet.MatchString(ln) || callsHelper(ln) {
			recovery = true
			break
		}
	}
	if !recovery {
		return []lint.Finding{{Line: call + 1, Msg: prefix + tailNoRecovery}}
	}
	set, read := false, false
	for i, ln := range lines {
		set = set || i >= decl && i <= end && okSet.MatchString(ln)
		read = read || okMinus.MatchString(ln) || okCompare.MatchString(ln)
	}
	if set && read {
		return nil
	}
	return []lint.Finding{{Line: call + 1, Msg: prefix + tailNotKeyed}}
}
