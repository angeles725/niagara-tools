package retro

// Go twins of the test_retro.TestDerive tests on the refusal classes and the state dir default
// (go_server.go_twin("retro", "unit")).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	mcptoken "github.com/angeles725/niagara-tools/go/internal/mcpn4/token"
)

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// reasonConstants are the Reason* constants of package safety, read from its source so a new
// one cannot be missed.
func reasonConstants(t *testing.T) map[string]string {
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "safety", "reasons.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		if v, ok := n.(*ast.ValueSpec); ok && strings.HasPrefix(v.Names[0].Name, "Reason") {
			out[v.Names[0].Name], _ = strconv.Unquote(v.Values[0].(*ast.BasicLit).Value)
		}
		return true
	})
	return out
}

func TestEverySharedRefusalReasonIsClassifiedByTheRetro(t *testing.T) {
	consts := reasonConstants(t)
	if len(consts) != 16 {
		t.Errorf("%d Reason constants", len(consts))
	}
	for name, text := range consts {
		if RefusalClass(text) == "" {
			t.Errorf("%s (%q) is not classified", name, text)
		}
	}
}

func TestTheRaiseSitesUseTheSharedReasonConstants(t *testing.T) {
	scope, _ := ord.NewScope([]string{"station:|slot:/A"})
	for v, reason := range map[string]string{"station:|slot:/B": safety.ReasonScopeOutside, "/etc": safety.ReasonScopePlain} {
		if err := scope.Check(v); err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("%s: %v", v, err)
		}
	}
	none, _ := ord.NewScope(nil)
	if err := none.Check("station:|slot:/A"); err == nil || !strings.Contains(err.Error(), safety.ReasonScopeNone) {
		t.Errorf("no scope: %v", err)
	}
	tokens := mcptoken.New(60, func() float64 { return 1000 })
	if err := tokens.Consume("t", obj(), "h", nil); err == nil || !strings.Contains(err.Error(), safety.ReasonTokenMissing) {
		t.Errorf("token: %v", err)
	}
}

func TestTheStateDirDefaultIsOwnedBySafetyAlone(t *testing.T) {
	if safety.DefaultStateDir != "~/.local/state/mcp-n4" {
		t.Errorf("%q", safety.DefaultStateDir)
	}
	for _, dir := range []string{".", filepath.Join("..", "readtools")} {
		pkgs, err := parser.ParseDir(token.NewFileSet(), dir, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range pkgs {
			for name, f := range p.Files {
				if obj := f.Scope.Lookup("DefaultStateDir"); obj != nil {
					t.Errorf("%s declares DefaultStateDir", name)
				}
			}
		}
	}
}
