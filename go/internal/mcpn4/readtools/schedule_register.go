package readtools

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"

// registerSchedules serves the schedule read tools (n4_list_schedule_kinds, n4_read_schedule).
func registerSchedules(s *jsonrpc.Server, t *Tools) {
	s.Handlers["n4_list_schedule_kinds"] = t.ListScheduleKinds
	s.Handlers["n4_read_schedule"] = t.ReadSchedule
}
