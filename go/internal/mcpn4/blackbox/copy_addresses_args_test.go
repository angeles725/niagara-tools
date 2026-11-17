package blackbox

// n4_copy_component copies[] (per-copy proxyExt/dataAddress) is checked by the shared argument
// validator before anything reaches the station: an entry without data_address or with an
// unknown address_format is refused with -32602; a well-formed entry passes to the plan.

import "testing"

func TestCopyAddressOverridesAreCheckedBeforeTheStation(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	base := Args{"parent_ord": Folder, "name": "Temp", "new_names": []any{"Temp2"}, "dry_run": true}
	before := h.Station.Requests()
	for _, bad := range []any{
		map[string]any{"name": "Temp2"},
		map[string]any{"name": "Temp2", "data_address": "40010", "address_format": "octal"},
	} {
		reply := h.Srv.Request("tools/call", Args{"name": "n4_copy_component",
			"arguments": withArgs(base, Args{"copies": []any{bad}})})
		if ErrorCode(reply) != -32602 {
			t.Errorf("%v: want -32602, got %v", bad, reply)
		}
	}
	if got := h.Station.Requests(); got != before {
		t.Fatalf("a refused call reached the station: %d requests, before %d", got, before)
	}
	good := []any{map[string]any{"name": "Temp2", "data_address": "40010", "address_format": "modbus"}}
	reply := h.Srv.Request("tools/call", Args{"name": "n4_copy_component", "arguments": withArgs(base, Args{"copies": good})})
	if ErrorCode(reply) == -32602 {
		t.Errorf("a well-formed copies[] was refused by the validator: %v", reply)
	}
}
