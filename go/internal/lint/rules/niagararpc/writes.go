package niagararpc

// The write detection of an RPC body: the first write call, and whether a write check stands before it.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

var (
	setters   = map[string]bool{"set": true, "setBoolean": true, "setInt": true, "setLong": true, "setFloat": true, "setDouble": true, "setString": true, "setFlags": true, "setFacets": true}
	guards    = map[string]bool{"hasOperatorWrite": true, "hasAdminWrite": true, "checkWrite": true, "checkCanWrite": true, "requireTargetWrite": true, "canWrite": true}
	guardBits = map[string]bool{"OPERATOR_WRITE": true, "ADMIN_WRITE": true}
	// callWords are the words after which a name is a call, not a declaration: `return doReset(x)`.
	callWords = map[string]bool{"return": true, "else": true, "throw": true, "yield": true, "case": true, "do": true}
)

// firstWrite is the token index and name of the first write call in toks[from:to), -1 when there is none.
func firstWrite(toks []javasrc.Token, from, to int) (int, string) {
	for p := from; p+1 < to; p++ {
		t := toks[p]
		if t.Kind != javasrc.TokIdent || toks[p+1].Text != "(" {
			continue
		}
		kind := 0
		switch {
		case setters[t.Text]:
			kind = 1
		case t.Text == "invoke":
			kind = 2
		case t.Text != "doPrivileged" && len(t.Text) > 2 && strings.HasPrefix(t.Text, "do") && t.Text[2] >= 'A' && t.Text[2] <= 'Z':
			kind = 3
		}
		if kind == 0 || (p > from && !isCall(toks[p-1])) {
			continue
		}
		cl := matchTok(toks, p+1, "(", ")")
		if cl < 0 {
			cl = to
		}
		n, first := args(toks[p+2 : cl])
		if kind == 1 && (n < 2 || first == "number") || kind == 2 && n < 2 {
			continue
		}
		return p, t.Text
	}
	return -1, ""
}

// isCall: the token before a name leaves it a call (a "." receiver, an operator, a keyword like return), not a
// declaration (a type or modifier before it).
func isCall(prev javasrc.Token) bool {
	if prev.Text == "." {
		return true
	}
	if prev.Kind == javasrc.TokIdent {
		return callWords[prev.Text]
	}
	return prev.Text != ">" && prev.Text != "]"
}

// args counts the top-level arguments of an argument list; first is "number" when the first one is a number.
func args(list []javasrc.Token) (n int, first string) {
	if len(list) == 0 {
		return 0, ""
	}
	n, depth := 1, 0
	for _, t := range list {
		if t.Kind != javasrc.TokPunct {
			continue
		}
		switch t.Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				n++
			}
		}
	}
	h := list[0]
	if (h.Text == "-" || h.Text == "+") && len(list) > 1 {
		h = list[1]
	}
	if h.Kind == javasrc.TokNumber {
		first = "number"
	}
	return n, first
}

// guarded: a write check stands in toks[from:before).
func guarded(toks []javasrc.Token, from, before int) bool {
	for p := from; p < before; p++ {
		t := toks[p]
		if t.Kind != javasrc.TokIdent {
			continue
		}
		if guardBits[t.Text] || guards[t.Text] && p+1 < len(toks) && toks[p+1].Text == "(" {
			return true
		}
	}
	return false
}
