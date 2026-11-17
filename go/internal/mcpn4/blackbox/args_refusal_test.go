package blackbox

// Every tool refuses an unknown argument or an enum violation with JSON-RPC -32602 naming the
// argument, before anything reaches the station (client-A overamps retro 2026-11-16, row M11:
// capacity_records: 50000 on n4_add_history was accepted and dropped, the 500-record default
// kept).

import "testing"

// rpcRefusal is the JSON-RPC error of tools/call tool with args: its code and message.
func rpcRefusal(t *testing.T, s *Server, tool string, args Args) (int, string) {
	t.Helper()
	reply := s.Request("tools/call", Args{"name": tool, "arguments": args})
	msg, _ := Dig(reply, "error", "message").(string)
	return ErrorCode(reply), msg
}

func TestEveryToolRefusesAnUnknownArgumentWith32602BeforeTheStation(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	before := h.Station.Requests()
	cases := []struct {
		tool string
		args Args
		want string
	}{
		{"n4_add_history", historyArgs(Args{"capacity_records": 50000}, "Temp"),
			"unknown argument 'capacity_records' for n4_add_history; did you mean 'capacity'?"},
		{"n4_navigate", Args{"depht": 1}, "unknown argument 'depht' for n4_navigate; did you mean 'depth'?"},
		{"n4_read_slots", Args{"ord": Folder, "zzz": true}, "unknown argument 'zzz' for n4_read_slots"},
	}
	for _, c := range cases {
		code, msg := rpcRefusal(t, h.Srv, c.tool, c.args)
		if code != -32602 || msg != c.want {
			t.Errorf("%s %v: code %d message %q, want -32602 %q", c.tool, c.args, code, msg, c.want)
		}
	}
	if got := h.Station.Requests(); got != before {
		t.Errorf("a refused call reached the station: %d requests, before %d", got, before)
	}
}

func TestEveryToolRefusesAnEnumViolationWith32602NamingTheAllowedValues(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	before := h.Station.Requests()
	cases := []struct {
		tool string
		args Args
		want string
	}{
		{"n4_add_history", covArgs(Args{"full_policy": "drop"}, "Temp"),
			"argument full_policy must be one of roll, stop; got 'drop'"},
		{"n4_add_history", historyArgs(Args{"mode": "sampled"}, "Temp"),
			"argument mode must be one of interval, cov; got 'sampled'"},
		{"n4_read_tags", Args{"ord": Folder, "relations": "some"},
			"argument relations must be one of direct, all, none; got 'some'"},
		{"n4_commissioning_report", Args{"sections": []any{"points", "pumps"}},
			"argument sections[1] must be one of points, histories, alarms, links, naming, station, spyder; got 'pumps'"},
	}
	for _, c := range cases {
		code, msg := rpcRefusal(t, h.Srv, c.tool, c.args)
		if code != -32602 || msg != c.want {
			t.Errorf("%s %v: code %d message %q, want -32602 %q", c.tool, c.args, code, msg, c.want)
		}
	}
	if got := h.Station.Requests(); got != before {
		t.Errorf("a refused call reached the station: %d requests, before %d", got, before)
	}
}

func TestKnownArgumentsAndOpenMapsStillPass(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	dry := h.Srv.Dry("n4_add_history", covArgs(Args{"capacity": 50000, "full_policy": "stop"}, "Temp"))
	if tok, _ := dry["confirmation_token"].(string); tok == "" {
		t.Fatalf("a valid call was refused: %v", dry)
	}
}

// alarm_values is an open map (additionalProperties): its tags are the caller's, never refused
// as unknown arguments; the call is planned against the station (retro row M11b: asserted
// positively, not only as "not this refusal").
func TestAnOpenMapsKeysPassToTheTool(t *testing.T) {
	h := newAlarmRig(t)
	h.Station.SetSupertype("control:EnumWritable", "control:EnumPoint")
	h.Station.SetSupertype("control:EnumPoint", "baja:Component")
	h.Add("3", "Mode", "control:EnumWritable")
	before := h.Station.Requests()
	dry := h.Srv.Dry("n4_add_alarm", Args{"points": []any{Folder + "/Mode"},
		"kind": "enum_change_of_state", "alarm_values": Args{"anyTag": 1}})
	if tok, _ := dry["confirmation_token"].(string); tok == "" {
		t.Fatalf("a valid call with an open map was refused: %v", dry)
	}
	if h.Station.Requests() == before {
		t.Error("the call with an open map never reached the station")
	}
}
