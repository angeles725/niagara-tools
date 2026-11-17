package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// n4_replicate_points (field retro 2026-11-16, row U10): the writable Modbus point Gw under net
// replicated onto other devices' point folders (net/T1, ...), each target with its own copies[].

// replicaClient is copyClient plus one empty point folder per target name under net.
func replicaClient(t *testing.T, tree *pyjson.Object, targets ...string) *typed {
	t.Helper()
	c := copyClient(t, tree)
	for i, n := range targets {
		c.loads.nodes.Set(net+"/"+n, obj("", obj("h", "5"+string(rune('0'+i)), "t", "modbusCore:ModbusClientPointDeviceExt")))
	}
	return c
}

// replicaArgs replicates Gw onto each target, with the given copies[] by target.
func replicaArgs(targets ...any) *pyjson.Object {
	list := []any{}
	for _, tg := range targets {
		switch v := tg.(type) {
		case string:
			list = append(list, obj("parent_ord", net+"/"+v))
		case *pyjson.Object:
			list = append(list, v)
		}
	}
	return obj("source_parent_ord", net, "points", []any{"Gw"}, "targets", list)
}

func budget(max int) *plan.State { return &plan.State{MaxWrites: max} }

func TestEachTargetGetsItsOwnAddressOverride(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1", "T2")
	args := replicaArgs(obj("parent_ord", net+"/T1", "copies", []any{obj("point", "Gw", "data_address", "40010")}), "T2")
	p, err := writeops.ReplicatePoints.PlanWith(c, args, budget(200), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 || p.Writes != 2 {
		t.Fatalf("ops %d, writes %d; want 2 and 2", len(p.Ops), p.Writes)
	}
	address := `{"nm": "p", "t": "baja:String", "n": "address", "v": "%s"}`
	for i, want := range []string{"40010", "40002"} {
		op := js(t, p.Ops[i])
		if !strings.Contains(op, strings.Replace(address, "%s", want, 1)) {
			t.Errorf("op %d lacks address %s: %s", i, want, op)
		}
		if !strings.Contains(op, `"h": "5`+string(rune('0'+i))+`"`) || !strings.Contains(op, `"n": "Gw"`) {
			t.Errorf("op %d does not add Gw under target %d: %s", i, i+1, op)
		}
	}
	if notes := js(t, p.Notes); strings.Contains(notes, "Duplicate Writable") {
		t.Errorf("copies on different devices are noted as duplicates: %s", notes)
	}
}

func TestAPerTargetNewNameRenamesOnlyThatTargetsCopy(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1", "T2")
	args := replicaArgs(obj("parent_ord", net+"/T1", "copies", []any{obj("point", "Gw", "new_name", "Setpoint")}), "T2")
	p, err := writeops.ReplicatePoints.PlanWith(c, args, budget(200), nil)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := js(t, p.Ops[0]), js(t, p.Ops[1]); !strings.Contains(a, `"n": "Setpoint"`) || !strings.Contains(b, `"n": "Gw"`) {
		t.Errorf("ops %s\n %s", a, b)
	}
}

func TestReplicatingOntoTheSourceDeviceNotesTheSharedRegister(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"))
	args := obj("source_parent_ord", net, "points", []any{"Gw"}, "targets", []any{
		obj("parent_ord", net, "copies", []any{obj("point", "Gw", "new_name", "G2")})})
	p, err := writeops.ReplicatePoints.PlanWith(c, args, budget(200), nil)
	if err != nil {
		t.Fatal(err)
	}
	if notes := js(t, p.Notes); !strings.Contains(notes, "'G2' would use the register address Modbus:40002") {
		t.Errorf("notes %s lack the shared register", notes)
	}
}

func TestAReplicationOverTheRemainingBudgetIsRefusedInThePlan(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1", "T2")
	w := budget(5)
	w.WritesExecuted = 4
	_, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1", "T2"), w, nil)
	if err == nil || !strings.Contains(err.Error(), "this replication needs 2 write(s) but only 1 remain") {
		t.Errorf("err = %v", err)
	}
}

func TestReplicaArgumentsAreValidatedBeforeTheStation(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1")
	for args, want := range map[*pyjson.Object]string{
		replicaArgs():           "points and targets must each list at least one entry",
		replicaArgs("T1", "T1"): "targets[].parent_ord must be distinct",
		replicaArgs(obj("parent_ord", net+"/T1", "copies", []any{obj("point", "Nope", "data_address", "1")})): "point 'Nope' is not one of points",
		replicaArgs(obj("parent_ord", net+"/T1", "copies", []any{obj("point", "Gw", "data_address", " ")})):   "must be a non-empty address",
	} {
		if _, err := writeops.ReplicatePoints.PlanWith(c, args, budget(200), nil); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", js(t, args), err, want)
		}
	}
}

// existingPoint is a target folder holding the point name at address (a writable one with a
// Modbus client proxy when typ ends in Writable), as a flat load of depth 4 returns it.
func existingPoint(h, name, typ, address string) *pyjson.Object {
	return obj("", obj("h", h, "t", "modbusCore:ModbusClientPointDeviceExt"),
		name, obj("n", name, "t", typ),
		name+"/proxyExt", obj("n", "proxyExt", "t", "modbusCore:ModbusClientNumericProxyExt"),
		name+"/proxyExt/dataAddress", obj("n", "dataAddress", "t", "modbusCore:FlexAddress"),
		name+"/proxyExt/dataAddress/addressFormat", obj("n", "addressFormat", "t", "modbusCore:AddressFormatEnum", "v", "modbus"),
		name+"/proxyExt/dataAddress/address", obj("n", "address", "t", "baja:String", "v", address))
}

func TestACopyAtTheRegisterOfAPointAlreadyOnTheTargetIsNoted(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1", "T2")
	c.loads.nodes.Set(net+"/T1", existingPoint("50", "Old", "control:NumericWritable", "40002"))
	p, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1", "T2"), budget(200), nil)
	if err != nil {
		t.Fatal(err)
	}
	notes := js(t, p.Notes)
	want := net + "/T1: 'Gw' would use the register address Modbus:40002, which the writable point 'Old' already on that device uses"
	if !strings.Contains(notes, want) {
		t.Errorf("notes %s lack %q", notes, want)
	}
	if strings.Contains(notes, net+"/T2: 'Gw'") {
		t.Errorf("T2 holds no point at 40002 but is noted: %s", notes)
	}
}

func TestAReadOnlyPointAlreadyOnTheTargetDoesNotHoldTheRegister(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1")
	c.loads.nodes.Set(net+"/T1", existingPoint("50", "Old", "control:NumericPoint", "40002"))
	p, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1"), budget(200), nil)
	if err != nil {
		t.Fatal(err)
	}
	if notes := js(t, p.Notes); strings.Contains(notes, "Duplicate Writable") {
		t.Errorf("a read-only point is noted as holding the register: %s", notes)
	}
}

// A replication of N adds needs N writes: its n4_rollback is outside the budget, so none is kept
// for it (retro 2026-11-17 Δ6, superseding #777's one write kept).
func TestAReplicationOfNAddsNeedsNWrites(t *testing.T) {
	c := replicaClient(t, pointTree("control:NumericWritable", "40002"), "T1", "T2")
	if _, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1", "T2"), budget(2), nil); err != nil {
		t.Errorf("2 adds fit a budget of 2: %v", err)
	}
	if _, err := writeops.ReplicatePoints.PlanWith(c, replicaArgs("T1", "T2"), budget(1), nil); err == nil ||
		!strings.Contains(err.Error(), "this replication needs 2 write(s) but only 1 remain") {
		t.Errorf("err = %v", err)
	}
}
