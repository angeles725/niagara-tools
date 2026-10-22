// Package jarversion implements lint-bundled-jar-class-version: it flags a bundled ext-jar under a module
// source tree that carries Java 9+ bytecode (class-file major version > 52), which an N4 station cannot
// load. The rules are those of the bash tool it replaces: the same file walk (find -P with build/ and
// dot-directories pruned), the same sample of class entries, the same version read and the same rows.
package jarversion

import (
	"fmt"
	"strings"
)

// SampleLimit is how many class entries of one jar are read, in archive order.
const SampleLimit = 20

// Java8Major is the class-file major version of Java 8, the highest an N4 module may carry.
const Java8Major = 52

// Sample returns the entry names the bash tool read: those ending in ".class" (grep '\.class$'), minus
// the multi-release META-INF/versions/ tree, at most SampleLimit of them, in archive order.
func Sample(names []string) []string {
	var out []string
	for _, n := range names {
		if len(out) == SampleLimit {
			break
		}
		if strings.HasSuffix(n, ".class") && !strings.HasPrefix(n, "META-INF/versions/") {
			out = append(out, n)
		}
	}
	return out
}

// Major is the class-file major version in head, the first bytes of a class entry: bytes 6-7 big-endian
// (after the CAFEBABE magic and the minor version). The magic is not checked, as the bash tool's
// `od -j6 -N2` did not. A seven-byte entry gives its seventh byte (od printed one value); a shorter one
// gives -1 (od could not skip past its end).
func Major(head []byte) int {
	switch {
	case len(head) >= 8:
		return int(head[6])<<8 | int(head[7])
	case len(head) == 7:
		return int(head[6])
	default:
		return -1
	}
}

// AboveJava8 reports whether major is bytecode an N4 (Java 8) station cannot load.
func AboveJava8(major int) bool {
	// Mutation: BJCV2 -- dropping the > 52 comparison makes every bundled Java 9+ class pass.
	return major > Java8Major
}

// FailRow is the report row for a jar whose class entry carries major version major.
func FailRow(jar, entry string, major int) string {
	return fmt.Sprintf("FAIL  lint-bundled-jar-class-version  %s  class %s is major %d (> 52 = Java 8)"+
		" — bundled Java 9+ bytecode throws UnsupportedClassVersionError at load\n", jar, entry, major)
}
