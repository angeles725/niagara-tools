package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
)

// Register serves the write tools this version ports through e; every other write tool keeps
// answering jsonrpc.NotImplemented.
func Register(s *jsonrpc.Server, e *plan.Engine) {
	s.EnableWrites()
	s.Handlers["n4_set_slot"] = e.Handler("n4_set_slot", SetSlot)
	s.Handlers["n4_invoke_action"] = e.Handler("n4_invoke_action", InvokeAction)
	s.Handlers["n4_create_component"] = e.Handler("n4_create_component", CreateComponent)
	s.Handlers["n4_create_link"] = e.Handler("n4_create_link", CreateLink)
	s.Handlers["n4_remove_component"] = e.Handler("n4_remove_component", RemoveComponent)
}
