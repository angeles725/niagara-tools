package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// A scope root that already exists directly under the station root is refused, and the refusal
// names it with one slash (found live on Prueba_4_15: "station:|slot://ImplT4").
func TestExistingScopeRootUnderTheStationRootIsNamedWithOneSlash(t *testing.T) {
	scope, err := ord.NewScope([]string{"station:|slot:/ImplT4"})
	if err != nil {
		t.Fatal(err)
	}
	c := &loads{nodes: obj("station:|slot:/", obj("", obj("h", "1"), "ImplT4", obj("h", "2", "t", "baja:Folder")))}
	args := obj("parent_ord", "station:|slot:/", "name", "ImplT4", "type", "baja:Folder")
	_, err = writeops.CreateComponent.PlanWith(c, args, &plan.State{Scope: scope}, nil)
	if err == nil || !strings.Contains(err.Error(), "'station:|slot:/ImplT4' already exists") {
		t.Fatalf("err = %v", err)
	}
}
