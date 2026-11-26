package structure

import (
	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
	"github.com/angeles725/niagara-tools/go/internal/signblock"
)

// AppliesSigning: the script calls id("com.tridium.niagara-signing").
func AppliesSigning(toks []ktssrc.Token) bool {
	for k := 0; k+2 < len(toks); k++ {
		if toks[k].Kind == ktssrc.Ident && toks[k].Text == "id" && toks[k+1].Text == "(" &&
			toks[k+2].Kind == ktssrc.String && toks[k+2].Value == "com.tridium.niagara-signing" {
			return true
		}
	}
	return false
}

// LacksSigningBlock reports whether the source of a root build.gradle.kts applies the signing plugin
// (AppliesSigning) and no line of its code, comments and literal contents dropped, opens a niagaraSigning {}
// block (signblock.Parts[1], the L15 pattern): a root the plugin reads no alias or profile from. A script it
// cannot scan is the error L15 turns into its WARN.
func LacksSigningBlock(src string) (bool, error) {
	toks, err := ktssrc.Tokens(src)
	if err != nil || !AppliesSigning(toks) {
		return false, err
	}
	for _, line := range ktssrc.Code(toks) {
		if signblock.Parts[1].Re.MatchString(line) {
			return false, nil
		}
	}
	return true, nil
}
