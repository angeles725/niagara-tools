package verifymodule

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// The station-save row texts. A station holds one config.bog, so the doctrine is one save per operator
// change per STATION (types/logic.md § Restart & persistence): the row makes a second requester visible
// before deploy. It reports and never fails, with or without --strict, so an existing module keeps building.
const (
	stationSaveCheck = "station-save"
	stationSaveCall  = "Sys.getStation().save()"
	stationSaveNone  = "no class calls " + stationSaveCall + " under %s/src"
	stationSaveOne   = "1 class calls " + stationSaveCall + ": %s"
	stationSaveMany  = "%d classes call " + stationSaveCall + ": %s" + stationSaveRule
	stationSaveRule  = " — one operator change must yield ONE station save: keep one requester, the owner of the" +
		" persisted slot (types/logic.md § Restart & persistence); reported, never a FAIL"
)

// stationSaveChain is the call as tokens: comments are blank in the stripped view and a string is one
// literal token, so neither is a call, and the chain may span lines.
var stationSaveChain = []string{"Sys", ".", "getStation", "(", ")", ".", "save", "("}

// stationSaveLine is the 1-based line of the first Sys.getStation().save( call in toks, or 0.
func stationSaveLine(toks []javasrc.Token) int {
	for i := 0; i+len(stationSaveChain) <= len(toks); i++ {
		if chainAt(toks[i:], stationSaveChain) {
			return toks[i].Line + 1
		}
	}
	return 0
}

// chainAt reports whether toks starts with the code tokens chain.
func chainAt(toks []javasrc.Token, chain []string) bool {
	for k, text := range chain {
		if toks[k].Kind == javasrc.TokLiteral || toks[k].Text != text {
			return false
		}
	}
	return true
}

// stationSave counts the classes (source files) under the profile's src that request a station save and
// names each at the file:line of its first call: none or one is a PASS, two or more one WARN.
func (r *run) stationSave() bool {
	pd, ok := r.srcDirOrSkip(stationSaveCheck)
	if !ok {
		return true
	}
	var callers []string
	for _, f := range r.javaFiles(pd + "/src") {
		v, ok := r.java(f)
		if !ok {
			return true
		}
		if ln := stationSaveLine(javasrc.Tokens(v.code)); ln > 0 {
			callers = append(callers, f+":"+itoa(ln))
		}
	}
	switch len(callers) {
	case 0:
		r.row("PASS", stationSaveCheck, fmt.Sprintf(stationSaveNone, pd))
	case 1:
		r.row("PASS", stationSaveCheck, fmt.Sprintf(stationSaveOne, callers[0]))
	default:
		r.row("WARN", stationSaveCheck, fmt.Sprintf(stationSaveMany, len(callers), strings.Join(callers, ", ")))
	}
	return true
}
