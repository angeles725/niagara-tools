package javasrc

import "strings"

// Class is one named class, interface, enum or record declaration found by Classes in a token stream.
type Class struct {
	Kind    string // "class", "interface", "enum" or "record"
	Name    string // the simple name
	Extends string // a class's superclass, simple name (a qualified name's last part, type arguments dropped); "" when none
	Line    int    // the 0-based line of the name token
	Body    [2]int // the body's token range [Body[0], Body[1]): inside its braces
}

// Classes finds every named type declaration in source order, nested ones included. A class literal
// (A.class) and an anonymous class are not declarations; "extends" of an interface is not a superclass.
func Classes(toks []Token) []Class {
	var out []Class
	for k := 0; k+1 < len(toks); k++ {
		t := toks[k]
		if t.Kind != TokIdent || !classKeyword[t.Text] || toks[k+1].Kind != TokIdent || (k > 0 && toks[k-1].Text == ".") {
			continue
		}
		c := Class{Kind: t.Text, Name: toks[k+1].Text, Line: toks[k+1].Line}
		angle, paren, j := 0, 0, k+2
		for ; j < len(toks); j++ {
			switch s := toks[j].Text; {
			case s == "<":
				angle++
			case s == ">":
				angle--
			case s == "(":
				paren++
			case s == ")":
				paren--
			case s == "extends" && angle == 0 && paren == 0 && c.Kind == "class":
				c.Extends = typeName(toks, j+1)
			}
			if angle == 0 && paren == 0 && (toks[j].Text == "{" || toks[j].Text == ";") {
				break
			}
		}
		if j >= len(toks) || toks[j].Text != "{" {
			continue
		}
		c.Body = [2]int{j + 1, matchBrace(toks, j)}
		out = append(out, c)
	}
	return out
}

// typeName is the simple name of the (possibly qualified) type starting at toks[i]: its last identifier
// before type arguments or the end of the name.
func typeName(toks []Token, i int) string {
	name := ""
	for ; i < len(toks) && toks[i].Kind == TokIdent; i += 2 {
		name = toks[i].Text
		if i+1 >= len(toks) || toks[i+1].Text != "." {
			break
		}
	}
	return name
}

// matchBrace is the index of the "}" matching the "{" at toks[open], or len(toks) when it is unbalanced.
func matchBrace(toks []Token, open int) int {
	depth := 0
	for j := open; j < len(toks); j++ {
		switch toks[j].Text {
		case "{":
			depth++
		case "}":
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return len(toks)
}

// Field is one field declaration directly in a class body; a declaration of several names is its first.
type Field struct {
	Static bool   // declared static
	Type   string // the declared type as written ("Set<String>", "java.util.List<String>", "String[]")
	Name   string // the first declared name
	Line   int    // the 0-based line of the name token
	Init   [2]int // the initializer's token range, every declarator's; empty when there is none
}

// fieldModifier are the keywords that may precede a field's type.
var fieldModifier = map[string]bool{"public": true, "protected": true, "private": true, "static": true, "final": true,
	"transient": true, "volatile": true}

// Fields finds the fields declared directly in the class body body (a Class.Body): a method, a constructor,
// an initializer block and a nested type are skipped whole, so their own declarations are never returned.
// An enum's constant list is not a field.
func Fields(toks []Token, body [2]int) []Field {
	var out []Field
	start, paren := body[0], 0
	for j := body[0]; j < body[1]; j++ {
		switch toks[j].Text {
		case "(":
			paren++
		case ")":
			paren--
		case ";":
			if paren == 0 {
				if f, ok := field(toks, start, j); ok {
					out = append(out, f)
				}
				start = j + 1
			}
		case "{":
			if paren == 0 && !hasAssign(toks[start:j]) { // a member body, not an initializer
				j = matchBrace(toks, j)
				start = j + 1
			} else {
				j = matchBrace(toks, j) // an array or anonymous-class initializer: the declaration goes on
			}
		}
	}
	return out
}

func hasAssign(seg []Token) bool {
	for _, t := range seg {
		if t.Text == "=" {
			return true
		}
	}
	return false
}

// field reads toks[start:end] (one member up to its ";") as a field declaration.
func field(toks []Token, start, end int) (Field, bool) {
	var f Field
	i := start
	for i < end {
		switch t := toks[i]; {
		case t.Text == "@" && i+1 < end:
			i = skipAnnotation(toks, i, end)
		case fieldModifier[t.Text]:
			f.Static = f.Static || t.Text == "static"
			i++
		default:
			goto typ
		}
	}
typ:
	angle, typeStart := 0, i
	for ; i < end; i++ {
		t := toks[i]
		switch t.Text {
		case "<":
			angle++
		case ">":
			angle--
		case "(":
			return Field{}, false // a method or constructor without a body, or an enum constant
		}
		if angle == 0 && i > typeStart && t.Kind == TokIdent && i+1 <= end &&
			(i+1 == end || toks[i+1].Text == "=" || toks[i+1].Text == "," || toks[i+1].Text == "[") {
			break
		}
	}
	if i >= end || i == typeStart {
		return Field{}, false
	}
	f.Type, f.Name, f.Line = joinType(toks[typeStart:i]), toks[i].Text, toks[i].Line
	for k := i + 1; k < end; k++ {
		if toks[k].Text == "=" {
			f.Init = [2]int{k + 1, end}
			break
		}
	}
	return f, true
}

// skipAnnotation is the index after the annotation at toks[i] ("@" Name[.Name]... [(...)]).
func skipAnnotation(toks []Token, i, end int) int {
	i++
	for i < end && toks[i].Kind == TokIdent {
		i++
		if i < end && toks[i].Text == "." {
			i++
			continue
		}
		break
	}
	if i < end && toks[i].Text == "(" {
		depth := 0
		for ; i < end; i++ {
			switch toks[i].Text {
			case "(":
				depth++
			case ")":
				depth--
			}
			if depth == 0 {
				return i + 1
			}
		}
	}
	return i
}

// joinType writes type tokens back as source: no spaces, ", " after a comma.
func joinType(ts []Token) string {
	var b strings.Builder
	for _, t := range ts {
		b.WriteString(t.Text)
		if t.Text == "," {
			b.WriteString(" ")
		}
	}
	return b.String()
}

// Descendants is the set of class names among classes whose superclass chain, followed through classes, reaches
// one of bases (simple names). Names are simple, so two classes of one name in different packages are one.
func Descendants(classes []Class, bases ...string) map[string]bool {
	parent := map[string]string{}
	for _, c := range classes {
		if c.Extends != "" {
			parent[c.Name] = c.Extends
		}
	}
	want := map[string]bool{}
	for _, b := range bases {
		want[b] = true
	}
	out := map[string]bool{}
	for name := range parent {
		seen := map[string]bool{}
		for p, ok := parent[name]; ok && !seen[p]; p, ok = parent[p] {
			if want[p] {
				out[name] = true
				break
			}
			seen[p] = true
		}
	}
	return out
}
