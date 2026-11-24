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
	s.Handlers["n4_set_flags"] = e.Handler("n4_set_flags", SetFlags)
	s.Handlers["n4_invoke_action"] = e.Handler("n4_invoke_action", InvokeAction)
	s.Handlers["n4_create_component"] = e.Handler("n4_create_component", CreateComponent)
	s.Handlers["n4_add_history"] = e.Handler("n4_add_history", AddHistory)
	s.Handlers["n4_add_alarm"] = e.Handler("n4_add_alarm", AddAlarm)
	s.Handlers["n4_ack_alarm"] = e.Handler("n4_ack_alarm", AckAlarm)
	s.Handlers["n4_copy_component"] = e.Handler("n4_copy_component", CopyComponent)
	s.Handlers["n4_replicate_points"] = e.Handler("n4_replicate_points", ReplicatePoints)
	s.Handlers["n4_deploy_template"] = e.Handler("n4_deploy_template", DeployTemplate)
	s.Handlers["n4_create_link"] = e.Handler("n4_create_link", CreateLink)
	s.Handlers["n4_rename_component"] = e.Handler("n4_rename_component", RenameComponent)
	s.Handlers["n4_discover_devices"] = e.Handler("n4_discover_devices", DiscoverDevices)
	s.Handlers["n4_discover_points"] = e.Handler("n4_discover_points", DiscoverPoints)
	s.Handlers["n4_register_bacnet_spyder"] = e.Handler("n4_register_bacnet_spyder", RegisterSpyder)
	s.Handlers["n4_remove_component"] = e.Handler("n4_remove_component", RemoveComponent)
	s.Handlers["n4_rollback"] = e.Handler("n4_rollback", Rollback)
	s.Handlers["n4_save_station"] = e.Handler("n4_save_station", SaveStation)
	s.Handlers["n4_set_schedule"] = e.Handler("n4_set_schedule", SetSchedule)
	s.Handlers["n4_set_tags"] = e.Handler("n4_set_tags", SetTags)
}
