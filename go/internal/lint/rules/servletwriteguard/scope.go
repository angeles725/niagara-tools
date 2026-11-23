package servletwriteguard

import (
	"regexp"
	"strings"
)

// Which calls are component writes. The lint reads source text, not types, so `write` (patterns.go) also
// matches a JDBC setInt(1, v), a List.set(i, x) or an AtomicReference.set(v). A call is dropped from the
// writes only on evidence that it is not a BComponent write (retro kit-p1-security-guards Δ6); a call it
// cannot read stays a write, so the lint keeps its false positives rather than its false negatives:
//   - a bare set takes a slot and a value (set(Property, BValue[, Context])): set with one argument or none is
//     not a write (AtomicReference.set(v), ThreadLocal.set(v), BitSet.set(i)); a generated property setter
//     (setDouble(42.0), setBoolean(true)) takes one argument and stays a write;
//   - the first argument of a slot write is a slot, never a numeric literal when a value follows it
//     (JDBC setInt(1, v), List.set(0, x)); setDouble(42.0) alone is a generated setter;
//   - a receiver that is a variable of this file declared only with a type that has no slots (plainTypes).
//
// invoke(…) is always an action on a component.

// plainTypes are the types whose set*(…) calls are not component slot writes.
var plainTypes = map[string]bool{
	"PreparedStatement": true, "CallableStatement": true, "List": true, "ArrayList": true, "LinkedList": true,
	"Vector": true, "CopyOnWriteArrayList": true, "ListIterator": true, "AtomicReference": true,
	"AtomicInteger": true, "AtomicLong": true, "AtomicBoolean": true, "AtomicIntegerArray": true,
	"AtomicLongArray": true, "AtomicReferenceArray": true, "ThreadLocal": true, "Calendar": true,
	"GregorianCalendar": true, "BitSet": true,
}

var (
	// typeDecl: a declaration `Type name` followed by what ends a declarator (a local, a field, a parameter);
	// group 3 is the type, group 6 the name. A qualified type (java.util.List) counts by its last segment.
	typeDecl = regexp.MustCompile(`(^|[^A-Za-z0-9_$.])((?:[a-z][A-Za-z0-9_$]*\.)*)([A-Z][A-Za-z0-9_$]*)(<[^;=(){}]*>)?(\[\])*` +
		sp + `(` + ident + `)` + sp + `[=;,):]`)
	// plainRecv: a receiver that is one variable, with or without `this.`; group 2 is the name.
	plainRecv = regexp.MustCompile(`^(this\.)?(` + ident + `)$`)
	// numericArg: an integer literal, an index.
	numericArg = regexp.MustCompile(`^-?[0-9][0-9_]*[LlFfDd]?$`)
)

// plainReceivers are the names this file declares only with a plainTypes type: a name also declared with any
// other type (in another method, say) is not one, since the lint does not tell the scopes apart.
func plainReceivers(lines []string) map[string]bool {
	plain, other := map[string]bool{}, map[string]bool{}
	for _, ln := range lines {
		for _, g := range typeDecl.FindAllStringSubmatch(ln, -1) {
			if plainTypes[g[3]] {
				plain[g[6]] = true
			} else {
				other[g[6]] = true
			}
		}
	}
	for n := range other {
		delete(plain, n)
	}
	return plain
}

// firstWrite is the first component write on line ln as the byte range of its `.name(` token, or nil.
func firstWrite(ln string, plain map[string]bool) []int {
	for _, loc := range write.FindAllStringIndex(ln, -1) {
		if componentWrite(ln, loc, plain) {
			return loc
		}
	}
	return nil
}

// hasWrite reports whether some line holds a component write.
func hasWrite(lines []string, plain map[string]bool) bool {
	for _, ln := range lines {
		if firstWrite(ln, plain) != nil {
			return true
		}
	}
	return false
}

// componentWrite reports whether the call at loc (the `.name(` token of ln) may be a component write.
func componentWrite(ln string, loc []int, plain map[string]bool) bool {
	name := ln[loc[0]+1 : loc[1]-1]
	if name == "invoke" {
		return true
	}
	open := loc[1] - 1
	if end := argEnd(ln, open); end < len(ln) { // a call that spans lines has an argument count nobody read
		args := topLevelArgs(ln[open+1 : end])
		// Niagara's generated property setters are one argument (comp.setDouble(42.0)) and ARE slot writes:
		// only the bare set is dropped for having fewer than two arguments, and a numeric first argument drops
		// a call only when a second argument follows (JDBC setInt(1, v), List.set(0, x)).
		if name == "set" && len(args) < 2 {
			return false
		}
		if len(args) >= 2 && numericArg.MatchString(args[0]) {
			return false
		}
	}
	recv := strings.TrimSpace(ln[receiverStart(ln, loc[0]):loc[0]])
	if m := plainRecv.FindStringSubmatch(recv); m != nil && plain[m[2]] {
		return false
	}
	return true
}

// topLevelArgs splits the text between a call's parentheses at the commas outside any bracket pair; blank
// text is no argument.
func topLevelArgs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	depth, from := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[from:i]))
				from = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[from:]))
}
