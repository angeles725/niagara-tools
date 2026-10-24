package ktssrc

import (
	"strings"
	"testing"
)

// show renders tokens as "<kind>:<text>" (a String as S:<value>, T marking a template) joined by spaces.
func show(toks []Token) string {
	var b []string
	for _, t := range toks {
		switch t.Kind {
		case String:
			v := "S:" + t.Value
			if t.Template {
				v = "T:" + t.Value
			}
			b = append(b, v)
		case Char:
			b = append(b, "C:"+t.Text)
		case Number:
			b = append(b, "N:"+t.Text)
		case Punct:
			b = append(b, t.Text)
		default:
			b = append(b, t.Text)
		}
	}
	return strings.Join(b, " ")
}

func TestTokens(t *testing.T) {
	for in, want := range map[string]string{
		`api("g:a:1")`:                           `api ( S:g:a:1 )`,
		"api(\n  \"g:a:1\"\n)":                   `api ( S:g:a:1 )`,
		`// api("g:a:1")` + "\nx":                `x`,
		`/* a /* nested */ api("x") */ y`:        `y`,
		`val s = "a\"b"`:                         `val s = S:a\"b`,
		`"""raw "api" \n"""`:                     `S:raw "api" \n`,
		`"""a""""`:                               `S:a"`,
		`"g:a:$v"`:                               `T:g:a:$v`,
		`"g:a:${libs.v("x")}" z`:                 `T:g:a:${libs.v("x")} z`,
		`"${ "}" }" q`:                           `T:${ "}" } q`,
		`'a' '\'' '"'`:                           `C:'a' C:'\'' C:'"'`,
		"`my name`(1.5e3)":                       "my name ( N:1.5e3 )",
		`"// not a comment" /* "not a string */`: `S:// not a comment`,
	} {
		toks, err := Tokens(in)
		if got := show(toks); err != nil || got != want {
			t.Errorf("Tokens(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestTokensLines(t *testing.T) {
	toks, err := Tokens("a\n/* x\ny */ b \"\"\"1\n2\"\"\" c")
	if err != nil {
		t.Fatal(err)
	}
	var got []int
	for _, tk := range toks {
		got = append(got, tk.Line)
	}
	if want := []int{1, 3, 3, 4}; len(got) != len(want) || got[0] != 1 || got[1] != 3 || got[2] != 3 || got[3] != 4 {
		t.Errorf("lines %v, want %v", got, want)
	}
}

func TestTokensErrors(t *testing.T) {
	for in, want := range map[string]string{
		`api("g:a`:   "line 1: unterminated string",
		"a\n/* open": "line 2: unterminated /* comment",
		`"""open`:    "line 1: unterminated string",
		`"${ open`:   "line 1: unterminated ${ template",
		`'x`:         "line 1: unterminated char literal",
		"`name":      "line 1: unterminated `name`",
		"\"a\nb\"":   "line 1: unterminated string",
	} {
		if _, err := Tokens(in); err == nil || err.Error() != want {
			t.Errorf("Tokens(%q) error = %v, want %q", in, err, want)
		}
	}
}

// TestCode: the code view keeps every token on its line, drops comments, reduces literals to their quotes and
// keeps one blank per gap.
func TestCode(t *testing.T) {
	for in, want := range map[string]string{
		"a.set(false)":                "a.set(false)",
		"  x   =  \"v\" // c\ny":      "x = \"\"\ny",
		"a /* x\ny */ b":              "a\nb",
		"a/* x */b":                   "a b",
		"/* a /* b */ c */ d":         "d",
		"v = \"\"\"r\nr2\nr3\"\"\" w": "v = \"\"\"\n\n\"\"\" w",
		`v = "${"in"}" 'c'`:           `v = "" ''`,
		"`my name` x":                 "my name x",
		"a\n\n\nb":                    "a\n\n\nb",
		"":                            "",
	} {
		toks, err := Tokens(in)
		if got := strings.Join(Code(toks), "\n"); err != nil || got != want {
			t.Errorf("Code(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

// TestOffsets: Off and End bound each token's source bytes, a `name`'s backquotes included.
func TestOffsets(t *testing.T) {
	src := "val `a b` = \"\"\"x\n\"\"\" // c"
	toks, err := Tokens(src)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tok := range toks {
		got = append(got, src[tok.Off:tok.End])
	}
	if want := "val|`a b`|=|\"\"\"x\n\"\"\""; strings.Join(got, "|") != want {
		t.Errorf("spans %q, want %q", strings.Join(got, "|"), want)
	}
}
