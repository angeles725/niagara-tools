package props

import (
	"fmt"
	"strings"
	"testing"
)

func show(es []Entry) string {
	var b []string
	for _, e := range es {
		b = append(b, fmt.Sprintf("%d[%s]=[%s]", e.Line, e.Key, e.Value))
	}
	return strings.Join(b, " ")
}

// TestRead: java.util.Properties.load(Reader) semantics, line by line.
func TestRead(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"equals", "a=1\nb = 2", "0[a]=[1] 1[b]=[2]"},
		{"colon and space separators", "a:1\nb 2\nc\t: 3", "0[a]=[1] 1[b]=[2] 2[c]=[3]"},
		{"comments and blanks", "# a=1\n  ! b=2\n\n   \nc=3", "4[c]=[3]"},
		{"hash inside a value", "a=x # y", "0[a]=[x # y]"},
		{"key only", "a\nb=", "0[a]=[] 1[b]=[]"},
		{"escapes", `p=C:\\Users\\me\tx\u0041\q`, "0[p]=[C:\\Users\\me\tx" + "Aq]"},
		{"escaped separator in key", `a\=b\ c=d`, "0[a=b c]=[d]"},
		{"continuation", "p=/home/\\\n    me/x\nq=1", "0[p]=[/home/me/x] 2[q]=[1]"},
		{"even backslashes do not continue", "p=a\\\\\nq=1", `0[p]=[a\] 1[q]=[1]`},
		{"comment line ending in a backslash does not continue", "# x\\\nq=1", "1[q]=[1]"},
		{"crlf", "a=1\r\nb=2\r", "0[a]=[1] 1[b]=[2]"},
		{"leading whitespace", "  \t\fa = 1", "0[a]=[1]"},
	} {
		if got := show(Read(strings.Split(tc.src, "\n"))); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

// TestBuildLocationKey (R10-D8): the four host build-location keys, read by lint-structure L10 and
// lint-machine-default-path alike, and nothing else (user_home is an ordinary host path).
func TestBuildLocationKey(t *testing.T) {
	for _, k := range []string{"niagara_home", "niagara_user_home", "nodeHome", "org.gradle.java.installations.paths"} {
		if !BuildLocationKey(k) {
			t.Errorf("%s: not a build-location key", k)
		}
	}
	for _, k := range []string{"user_home", "niagara_homeX", "org.gradle.java.installations.auto-detect", ""} {
		if BuildLocationKey(k) {
			t.Errorf("%s: a build-location key", k)
		}
	}
	if n := len(BuildLocationKeys()); n != 4 {
		t.Errorf("BuildLocationKeys: %d keys, want 4", n)
	}
}
