package jssrc

import (
	"fmt"
	"strings"
	"testing"
)

// show renders tokens as "line:kind:text" words, one per token.
func show(toks []Token) string {
	var b []string
	for _, t := range toks {
		b = append(b, fmt.Sprintf("%d:%s:%s", t.Line, t.Kind, t.Text))
	}
	return strings.Join(b, " ")
}

func TestTokens(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"declaration", `const D = "/home/x";`, `0:ident:const 0:ident:D 0:punct:= 0:string:/home/x 0:punct:;`},
		{"single quote and escape", `var a = 'it\'s';`, `0:ident:var 0:ident:a 0:punct:= 0:string:it\'s 0:punct:;`},
		{"line comment", "a = 1; // b = \"x\"\nc", `0:ident:a 0:punct:= 0:number:1 0:punct:; 1:ident:c`},
		{"block comment over lines", "a /* \"x\"\n */ b", `0:ident:a 1:ident:b`},
		{"url in a string is not a comment", `u = "http://h/x";`, `0:ident:u 0:punct:= 0:string:http://h/x 0:punct:;`},
		{"template", "t = `/home/${u}/x`;", "0:ident:t 0:punct:= 0:template:/home/${u}/x 0:punct:;"},
		{"template nesting", "t = `a${ {k: `b`}.k }c`; d", "0:ident:t 0:punct:= 0:template:a${ {k: `b`}.k }c 0:punct:; 0:ident:d"},
		{"template over lines", "t = `a\nb`; c", "0:ident:t 0:punct:= 0:template:a\nb 1:punct:; 1:ident:c"},
		{"regex after =", `r = /"[a-z]\/"/g; s = "x"`, `0:ident:r 0:punct:= 0:regex:/"[a-z]\/"/g 0:punct:; 0:ident:s 0:punct:= 0:string:x`},
		{"regex class with slash", `r = /[/"]/; s`, `0:ident:r 0:punct:= 0:regex:/[/"]/ 0:punct:; 0:ident:s`},
		{"division", `x = a / b / "c"`, `0:ident:x 0:punct:= 0:ident:a 0:punct:/ 0:ident:b 0:punct:/ 0:string:c`},
		{"regex after return", "return /a\"/.test(s)", `0:ident:return 0:regex:/a"/ 0:punct:. 0:ident:test 0:punct:( 0:ident:s 0:punct:)`},
		{"operators", `a === b => c ?. d`, `0:ident:a 0:punct:=== 0:ident:b 0:punct:=> 0:ident:c 0:punct:?. 0:ident:d`},
		{"object property", `o = {dir: "/tmp/x", 'k': 2}`, `0:ident:o 0:punct:= 0:punct:{ 0:ident:dir 0:punct:: 0:string:/tmp/x 0:punct:, 0:string:k 0:punct:: 0:number:2 0:punct:}`},
		{"crlf", "a = \"x\";\r\nb", `0:ident:a 0:punct:= 0:string:x 0:punct:; 1:ident:b`},
		{"unterminated string ends at the line", "a = \"x\nb", `0:ident:a 0:punct:= 0:string:x 1:ident:b`},
	} {
		if got := show(Tokens(strings.Split(tc.src, "\n"))); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}
