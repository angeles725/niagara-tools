package javasrc

import (
	"fmt"
	"strings"
	"testing"
)

func showTokens(toks []Token) string {
	var b []string
	for _, t := range toks {
		b = append(b, fmt.Sprintf("%d:%s:%s", t.Line, t.Kind, t.Text))
	}
	return strings.Join(b, " ")
}

// TestTokens: Tokens reads Strip output: comments are already blanks, literals are single tokens.
func TestTokens(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"declaration", `int a = 10;`, `0:ident:int 0:ident:a 0:punct:= 0:number:10 0:punct:;`},
		{"comments are gone", "a /* b; */ c // d;\ne", `0:ident:a 0:ident:c 1:ident:e`},
		{"string with braces and an escaped quote", `s = "{\";}";`, `0:ident:s 0:punct:= 0:literal:"{\";}" 0:punct:;`},
		{"char", `c = '}';`, `0:ident:c 0:punct:= 0:literal:'}' 0:punct:;`},
		{"text block is one token", "s = \"\"\"\n  void f() { a(); }\n  \"\"\";\nb", "0:ident:s 0:punct:= 0:literal:\"\"\"\n  void f() { a(); }\n  \"\"\" 2:punct:; 3:ident:b"},
		{"annotation and generics", `@Override Map<K, V> m()`, `0:punct:@ 0:ident:Override 0:ident:Map 0:punct:< 0:ident:K 0:punct:, 0:ident:V 0:punct:> 0:ident:m 0:punct:( 0:punct:)`},
		{"number forms", `x = 1.5e3 + 0xFFL;`, `0:ident:x 0:punct:= 0:number:1.5e3 0:punct:+ 0:number:0xFFL 0:punct:;`},
		{"crlf", "a;\r\nb", `0:ident:a 0:punct:; 1:ident:b`},
		{"unterminated string ends at the line", "s = \"x\nb", `0:ident:s 0:punct:= 0:literal:"x 1:ident:b`},
	} {
		lines, _ := Strip(strings.Split(tc.src, "\n"))
		if got := showTokens(Tokens(lines)); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}
