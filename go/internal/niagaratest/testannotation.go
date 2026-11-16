package niagaratest

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// errEnabledValue is an enabled attribute that is no true/false literal: the scan does not evaluate constants.
var errEnabledValue = errors.New("@Test enabled is not a true or false literal")

// testDisabled reads the annotations and modifiers before the declaration token at (a class keyword or a method
// name) and says whether a @Test (any qualifier) among them sets enabled = false.
func testDisabled(toks []javasrc.Token, at int) (bool, error) {
	var args []javasrc.Token // the arguments of the last ")...(" walked over
	for i := at - 1; i >= 0; {
		t := toks[i]
		switch {
		case t.Text == ")" || t.Text == ">" || t.Text == "]":
			open := openOf(toks, i)
			if open < 0 {
				return false, nil
			}
			if t.Text == ")" {
				args = toks[open+1 : i]
			}
			i = open - 1
		case t.Kind == javasrc.TokIdent:
			j := i // the first identifier of a qualified name a.b.Name
			for j >= 2 && toks[j-1].Text == "." && toks[j-2].Kind == javasrc.TokIdent {
				j -= 2
			}
			if j >= 1 && toks[j-1].Text == "@" && t.Text == "Test" {
				if v, found, err := enabledArg(args); err != nil || (found && v == "false") {
					return err == nil, err
				}
			}
			args = nil
			i = j - 1
			if i >= 0 && toks[i].Text == "@" {
				i--
			}
		default:
			return false, nil // the end of the previous member or statement
		}
	}
	return false, nil
}

// openOf is the index of the token opening the bracket closed at toks[i] (")", ">" or "]"), -1 when unbalanced.
func openOf(toks []javasrc.Token, i int) int {
	closer := toks[i].Text
	opener := map[string]string{")": "(", ">": "<", "]": "["}[closer]
	depth := 0
	for j := i; j >= 0; j-- {
		switch toks[j].Text {
		case closer:
			depth++
		case opener:
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return -1
}

// enabledArg is the value of the enabled attribute among an annotation's top-level arguments.
func enabledArg(args []javasrc.Token) (string, bool, error) {
	depth := 0
	for k := 0; k < len(args); k++ {
		switch args[k].Text {
		case "(", "{", "[":
			depth++
		case ")", "}", "]":
			depth--
		case "enabled":
			if depth != 0 || k+1 >= len(args) || args[k+1].Text != "=" {
				continue
			}
			if k+2 < len(args) && (args[k+2].Text == "true" || args[k+2].Text == "false") && (k+3 == len(args) || args[k+3].Text == ",") {
				return args[k+2].Text, true, nil
			}
			return "", true, errEnabledValue
		}
	}
	return "", false, nil
}

// hasModifier says whether the keyword mod stands among the modifiers before the declaration token at.
func hasModifier(toks []javasrc.Token, at int, mod string) bool {
	for i := at - 1; i >= 0; i-- {
		switch t := toks[i]; {
		case t.Text == ")":
			if i = openOf(toks, i); i < 0 {
				return false
			}
		case t.Kind == javasrc.TokIdent:
			if t.Text == mod && (i == 0 || (toks[i-1].Text != "@" && toks[i-1].Text != ".")) {
				return true
			}
		case t.Text != "@" && t.Text != ".":
			return false
		}
	}
	return false
}
