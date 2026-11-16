package lint

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
)

// TestDeclaredDeps: the module names a build script declares, by the rule verify-module's phantom-dep check
// reads them with: api, nre and implementation calls on a ":X" literal or project(":X"), in script order;
// a comment, a string, another configuration, a method call and a templated name declare nothing.
func TestDeclaredDeps(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, script, want string }{
		{"the three configurations", "dependencies {\n  nre(\":nre\")\n  api(\":baja\")\n  implementation(\":control-rt\")\n}\n", "nre baja control-rt"},
		{"a project dependency", "dependencies { api(project(\":ModA-rt\")) }\n", "ModA-rt"},
		{"a comment and a string are no code", "// api(\":x-rt\")\nval s = \"api(\\\":y-rt\\\")\"\n/* nre(\":z\") */\n", ""},
		{"another configuration", "dependencies { moduleTestImplementation(\":test-wb\")\n  uberjar(\":lib\")\n  xapi(\":x\") }\n", ""},
		{"a method call", "dependencies { handler.api(\":x-rt\") }\n", ""},
		{"a library coordinate", "dependencies { api(\"junit:junit:4.13.2\") }\n", ""},
		{"a templated name", "dependencies { api(\":$name-rt\") }\n", ""},
		{"a repeated name is kept twice", "api(\":baja\")\napi(\":baja\")\n", "baja baja"},
	}
	for _, c := range cases {
		toks, err := ktssrc.Tokens(c.script)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := strings.Join(DeclaredDeps(toks), " "); got != c.want {
			t.Errorf("%s: DeclaredDeps = %q, want %q", c.name, got, c.want)
		}
	}
}
